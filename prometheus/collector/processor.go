package collector

import (
	"context"
	"time"

	"github.com/remeh/sizedwaitgroup"
	"go.uber.org/zap"
)

type (
	ProcessorInterface interface {
		Setup(collector *Collector)
		Reset()
		Collect(callback chan<- func())
	}

	Processor struct {
		Collector *Collector
	}
)

func (p *Processor) Setup(collector *Collector) {
	p.Collector = collector
}

func (p *Processor) Logger() *zap.SugaredLogger {
	return p.Collector.logger
}

func (p *Processor) Context() context.Context {
	return p.Collector.context
}

func (p *Processor) WaitGroup() *sizedwaitgroup.SizedWaitGroup {
	return p.Collector.waitGroup
}

func (p *Processor) GetLastScapeTime() *time.Time {
	return p.Collector.GetLastScapeTime()
}
