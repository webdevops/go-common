package collector

import (
	"context"

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
	return p.Collector.Logger
}

func (p *Processor) Context() context.Context {
	return p.Collector.Context
}
