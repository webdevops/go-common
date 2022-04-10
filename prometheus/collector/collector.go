package collector

import (
	"context"
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

	Concurrency int
	waitGroup   *sizedwaitgroup.SizedWaitGroup

	logger *log.Entry

	processor ProcessorInterface
}

func New(name string, processor ProcessorInterface, logger *log.Logger) *Collector {
	c := &Collector{}
	c.context = context.Background()
	c.Name = name
	c.processor = processor
	c.Concurrency = -1
	if logger != nil {
		c.logger = logger.WithFields(log.Fields{
			"collector": name,
		})
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
	c.context = ctx
}

func (c *Collector) SetConcurrency(concurrency int) {
	c.Concurrency = concurrency
}

func (c *Collector) GetLastScrapeDuration() *time.Duration {
	return c.lastScrapeDuration
}
func (c *Collector) GetLastScapeTime() *time.Time {
	return c.lastScrapeTime
}

func (c *Collector) Start() error {
	if c.waitGroup == nil {
		wg := sizedwaitgroup.New(c.Concurrency)
		c.waitGroup = &wg
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
		c.waitGroup.Wait()
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
