package collector

import (
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
