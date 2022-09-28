package collector

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/remeh/sizedwaitgroup"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
)

type Collector struct {
	Name string

	context context.Context

	scrapeTime *time.Duration
	cronSpec   *string

	cron *cron.Cron

	lastScrapeDuration  *time.Duration
	lastScrapeTime      *time.Time
	collectionStartTime time.Time

	panic struct {
		threshold int64
		counter   int64
	}

	concurrency int
	waitGroup   *sizedwaitgroup.SizedWaitGroup

	logger *log.Entry

	processor ProcessorInterface
}

func New(name string, processor ProcessorInterface, logger *log.Logger) *Collector {
	c := &Collector{}
	c.context = context.Background()
	c.Name = name
	c.processor = processor
	c.concurrency = -1
	c.panic.threshold = 5
	c.panic.counter = 0
	if logger != nil {
		c.logger = logger.WithFields(log.Fields{
			"collector": name,
		})
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

func (c *Collector) SetContext(ctx context.Context) {
	c.context = ctx
}

func (c *Collector) SetConcurrency(concurrency int) {
	c.concurrency = concurrency
}

func (c *Collector) GetLastScrapeDuration() *time.Duration {
	return c.lastScrapeDuration
}

func (c *Collector) GetLastScapeTime() *time.Time {
	return c.lastScrapeTime
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
				c.collect()
				time.Sleep(*c.scrapeTime)
			}
		}()
	} else if c.cronSpec != nil {
		// cron execution
		return c.cron.AddFunc(*c.cronSpec, c.collect)
	}
	return nil
}

func (c *Collector) collect() {
	c.collectionStart()
	panicDetected := false

	callbackChannel := make(chan func())

	go func() {
		finished := false
		defer func() {
			close(callbackChannel)

			if !finished {
				panicDetected = true
				atomic.AddInt64(&c.panic.counter, 1)
				panicCounter := atomic.LoadInt64(&c.panic.counter)
				if c.panic.threshold == -1 || panicCounter <= c.panic.threshold {
					if err := recover(); err != nil {
						switch v := err.(type) {
						case *log.Entry:
							c.logger.Error(fmt.Sprintf("panic occurred (panic threshold %v of %v): ", panicCounter, c.panic.threshold), v.Message)
						case error:
							c.logger.Error(fmt.Sprintf("panic occurred (panic threshold %v of %v): ", panicCounter, c.panic.threshold), v.Error())
						default:
							c.logger.Error(fmt.Sprintf("panic occurred (panic threshold %v of %v): ", panicCounter, c.panic.threshold), v)
						}

					}
				}
			}
		}()

		c.processor.Collect(callbackChannel)
		c.waitGroup.Wait()
		finished = true
	}()

	var callbackList []func()
	for callback := range callbackChannel {
		callbackList = append(callbackList, callback)
	}

	// reset metric values
	c.processor.Reset()

	// process callbacks (set metrics)
	for _, callback := range callbackList {
		callback()
	}

	if !panicDetected {
		// reset panic counter after successful run without panics
		atomic.StoreInt64(&c.panic.counter, 0)
	}

	c.collectionFinish()
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

	if c.logger != nil {
		c.logger.WithField("duration", c.lastScrapeDuration.Seconds()).Info("finished metrics collection")
	}
}
