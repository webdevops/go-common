package collector

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/remeh/sizedwaitgroup"
	"github.com/robfig/cron"
	"go.uber.org/zap"

	prometheusCommon "github.com/webdevops/go-common/prometheus"
)

type Collector struct {
	Name string

	context context.Context

	scrapeTime *time.Duration
	sleepTime  *time.Duration
	cronSpec   *string

	cron *cron.Cron

	lastScrapeDuration  *time.Duration
	lastScrapeTime      *time.Time
	nextScrapeTime      *time.Time
	collectionStartTime time.Time

	cache               *cacheSpecDef
	cacheRestoreEnabled bool

	panic struct {
		threshold int64
		counter   int64
	}

	data *CollectorData

	registry *prometheus.Registry

	concurrency int
	waitGroup   *sizedwaitgroup.SizedWaitGroup

	logger *zap.SugaredLogger

	processor ProcessorInterface
}

type CollectorData struct {
	Metrics map[string]*MetricList `json:"metrics"`
	Data    map[string]interface{} `json:"data"`
	Expiry  *time.Time             `json:"expiry"`
}

func NewCollectorData() *CollectorData {
	return &CollectorData{
		Metrics: map[string]*MetricList{},
		Data:    map[string]interface{}{},
		Expiry:  nil,
	}
}

func New(name string, processor ProcessorInterface, logger *zap.SugaredLogger) *Collector {
	c := &Collector{}
	c.context = context.Background()
	c.Name = name
	c.data = NewCollectorData()
	c.processor = processor
	c.concurrency = -1
	c.panic.threshold = 5
	c.panic.counter = 0
	c.cacheRestoreEnabled = true
	if logger != nil {
		c.logger = logger.With(zap.String(`collector`, name))
	}
	processor.Setup(c)

	addCollectorToList(c)

	return c
}

func (c *Collector) SetPanicThreshold(threshold int64) {
	c.panic.threshold = threshold
}

func (c *Collector) SetCronSpec(cron *cron.Cron, cronSpec string) {
	c.cron = cron
	c.cronSpec = &cronSpec
}

func (c *Collector) SetScapeTime(scrapeTime time.Duration) {
	c.scrapeTime = &scrapeTime
}

func (c *Collector) SetNextSleepDuration(sleepDuration time.Duration) {
	c.sleepTime = &sleepDuration
}

func (c *Collector) SetContext(ctx context.Context) {
	c.context = ctx
}

func (c *Collector) SetConcurrency(concurrency int) {
	c.concurrency = concurrency
}

func (c *Collector) SetPrometheusRegistry(registry *prometheus.Registry) {
	c.registry = registry
}

func (c *Collector) GetPrometheusRegistry() *prometheus.Registry {
	return c.registry
}

func (c *Collector) GetLastScrapeDuration() *time.Duration {
	return c.lastScrapeDuration
}

func (c *Collector) GetLastScapeTime() *time.Time {
	return c.lastScrapeTime
}

func (c *Collector) GetNextScrapeTime() *time.Time {
	return c.nextScrapeTime
}

func (c *Collector) Start() error {
	if c.waitGroup == nil {
		wg := sizedwaitgroup.New(c.concurrency)
		c.waitGroup = &wg
	}

	if c.scrapeTime != nil {
		// scrape time execution
		go func() {
			// randomize collector start times
			startTimeOffset := float64(5)
			startTimeRandom := float64(5)
			startupWaitTime := time.Duration((rand.Float64()*startTimeRandom)+startTimeOffset) * time.Second // #nosec:G404 random value only used for startup time
			time.Sleep(startupWaitTime)

			for {
				c.run()
				time.Sleep(*c.sleepTime)
			}
		}()
	} else if c.cronSpec != nil {
		// cron execution
		return c.cron.AddFunc(*c.cronSpec, c.run)
	}
	return nil
}

func (c *Collector) run() {
	// set next sleep duration (automatic calculation, can be overwritten by collect)
	c.SetNextSleepDuration(*c.scrapeTime)

	// cleanup internal metric lists (to ensure clean metric lists)
	c.cleanupMetricLists()

	// start collection
	c.collectionStart()

	// try restore from cache (first run only)
	normalRun := true
	if c.collectionRestoreCache() {
		normalRun = false
		// metrics restored from cache, do not collect them but try to restore them
		func() {
			defer func() {
				// restore failed, reset metrics
				if err := recover(); err != nil {
					c.logger.Warnf(`caught panic while restore cached metrics: %v`, err)

					c.logger.Info(`enabling normal collection run, ignoring and resetting cached metrics`)
					c.resetMetrics()
					c.cleanupMetricLists()

					// enable normal run, we have to get metrics
					// from the collector as restore failed
					normalRun = true
				}
			}()

			// try to restore metrics from cache
			c.collectRun(false)
		}()
	}

	if normalRun {
		// metrics could not be restored from cache, start collect run
		c.collectRun(true)
		c.collectionSaveCache()
	}

	// cleanup internal metric lists (reduce memory load)
	c.cleanupMetricLists()

	// finish run and calculate next run
	c.collectionFinish()
}

func (c *Collector) collectRun(doCollect bool) {
	var panicDetected bool
	var callbackList []func()

	if doCollect {
		callbackChannel := make(chan func())

		go func() {
			finished := false

			// catch panics and increase panic counter
			// pass through panics after panic counter exceeds threshold
			defer func() {
				close(callbackChannel)

				if !finished {
					panicDetected = true
					atomic.AddInt64(&c.panic.counter, 1)
					panicCounter := atomic.LoadInt64(&c.panic.counter)
					if c.panic.threshold == -1 || panicCounter <= c.panic.threshold {
						if err := recover(); err != nil {
							switch v := err.(type) {
							case error:
								c.logger.Error(fmt.Sprintf("panic occurred (panic threshold %v of %v): ", panicCounter, c.panic.threshold), v.Error())
							default:
								c.logger.Error(fmt.Sprintf("panic occurred (panic threshold %v of %v): ", panicCounter, c.panic.threshold), v)
							}

						}
					}
				}

				if !panicDetected {
					// reset panic counter after successful run without panics
					atomic.StoreInt64(&c.panic.counter, 0)
				}
			}()

			c.processor.Collect(callbackChannel)
			c.waitGroup.Wait()
			finished = true
		}()

		for callback := range callbackChannel {
			callbackList = append(callbackList, callback)
		}
	}

	// ensure that metrics are written completely
	// promhttp handler should wait for rlock
	lock.Lock()
	defer lock.Unlock()

	c.resetMetrics()

	// process callbacks (set metrics)
	for _, callback := range callbackList {
		callback()
	}

	// set metrics from metrics
	for _, metric := range c.data.Metrics {
		switch vec := metric.vec.(type) {
		case *prometheus.GaugeVec:
			metric.GaugeSet(vec)
		case *prometheus.HistogramVec:
			metric.HistogramSet(vec)
		case *prometheus.SummaryVec:
			metric.SummarySet(vec)
		}
	}

}

func (c *Collector) resetMetrics() {
	// reset metric values
	c.processor.Reset()

	// reset first
	for _, metric := range c.data.Metrics {
		if metric.reset {
			switch vec := metric.vec.(type) {
			case *prometheus.GaugeVec:
				vec.Reset()
			case *prometheus.HistogramVec:
				vec.Reset()
			case *prometheus.SummaryVec:
				vec.Reset()
			}
		}
	}

}

func (c *Collector) SetData(name string, val interface{}) {
	c.data.Data[name] = val
}

func (c *Collector) GetData(name string) interface{} {
	if val, exists := c.data.Data[name]; exists {
		return val
	}
	return nil
}

func (c *Collector) RegisterMetricList(name string, vec interface{}, reset bool) *MetricList {
	c.data.Metrics[name] = &MetricList{
		MetricList: prometheusCommon.NewMetricsList(),
		vec:        vec,
		reset:      reset,
	}

	if c.registry != nil {
		switch vec := vec.(type) {
		case *prometheus.GaugeVec:
			c.registry.MustRegister(vec)
		case *prometheus.HistogramVec:
			c.registry.MustRegister(vec)
		case *prometheus.SummaryVec:
			c.registry.MustRegister(vec)
		default:
			panic(`not allowed prometheus metric vec found`)
		}
	} else {
		switch vec := vec.(type) {
		case *prometheus.GaugeVec:
			prometheus.MustRegister(vec)
		case *prometheus.HistogramVec:
			prometheus.MustRegister(vec)
		case *prometheus.SummaryVec:
			prometheus.MustRegister(vec)
		default:
			panic(`not allowed prometheus metric vec found`)
		}
	}

	return c.data.Metrics[name]
}

func (c *Collector) GetMetricList(name string) *MetricList {
	return c.data.Metrics[name]
}

func (c *Collector) cleanupMetricLists() {
	for _, metric := range c.data.Metrics {
		metric.MetricList.Reset()
	}
}

func (c *Collector) collectionStart() {
	c.collectionStartTime = time.Now()

	if c.logger != nil {
		c.logger.Info("starting metrics collection")
	}
}

func (c *Collector) collectionFinish() {
	c.lastScrapeTime = &c.collectionStartTime

	duration := time.Since(c.collectionStartTime)
	c.lastScrapeDuration = &duration

	nextScrapeTime := time.Now().Add(*c.sleepTime)
	c.nextScrapeTime = &nextScrapeTime

	if c.logger != nil {
		c.logger.With(
			zap.Float64("duration", c.lastScrapeDuration.Seconds()),
			zap.Time("nextRun", c.nextScrapeTime.UTC()),
		).Infof("finished metrics collection, next run in %s", c.sleepTime.String())
	}
}
