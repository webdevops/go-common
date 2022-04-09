package collector

import (
	"context"
	"time"

	"github.com/remeh/sizedwaitgroup"
	"github.com/robfig/cron"
	"go.uber.org/zap"
)

type Collector struct {
	Name string

	Context context.Context

	scrapeTime *time.Duration
	cronSpec   *string

	cron *cron.Cron

	LastScrapeDuration  *time.Duration
	collectionStartTime time.Time

	Concurrency int
	WaitGroup   *sizedwaitgroup.SizedWaitGroup

	Logger *zap.SugaredLogger

	processor Processor
}

func New(name string, processor Processor, logger *zap.Logger) *Collector {
	c := &Collector{}
	c.Context = context.Background()
	c.Name = name
	c.processor = processor
	c.Concurrency = -1
	if logger != nil {
		c.Logger = logger.Sugar().With(zap.String("collector", name))
	}
	processor.Setup(c)

	addCollectorToList(c)

	return c
}

func (c *Collector) SetCronSpec(cron *cron.Cron, cronSpec string) {
	c.cron = cron
	c.cronSpec = &cronSpec
}

func (c *Collector) SetScapeTime(scrapeTime time.Duration) {
	c.scrapeTime = &scrapeTime
}

func (c *Collector) SetContext(ctx context.Context) {
	c.Context = ctx
}

func (c *Collector) SetConcurrency(concurrency int) {
	c.Concurrency = concurrency
}

func (c *Collector) Start() error {
	if c.WaitGroup == nil {
		wg := sizedwaitgroup.New(c.Concurrency)
		c.WaitGroup = &wg
	}

	if c.scrapeTime != nil {
		// scrape time execution
		go func() {
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

	callbackChannel := make(chan func())

	go func() {
		c.processor.Collect(callbackChannel)
		close(callbackChannel)
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

	c.collectionFinish()
}

func (c *Collector) collectionStart() {
	c.collectionStartTime = time.Now()

	if c.Logger != nil {
		c.Logger.Info("starting metrics collection")
	}
}

func (c *Collector) collectionFinish() {
	duration := time.Since(c.collectionStartTime)
	c.LastScrapeDuration = &duration

	if c.Logger != nil {
		c.Logger.Infow(
			"finished metrics collection",
			zap.Float64("duration", c.LastScrapeDuration.Seconds()),
		)
	}
}
