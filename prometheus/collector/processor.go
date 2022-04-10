package collector

import (
	"context"

	"github.com/remeh/sizedwaitgroup"
	log "github.com/sirupsen/logrus"
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

func (p *Processor) Logger() *log.Entry {
	return p.Collector.logger
}

func (p *Processor) Context() context.Context {
	return p.Collector.context
}

func (p *Processor) WaitGroup() *sizedwaitgroup.SizedWaitGroup {
	return p.Collector.waitGroup
}
