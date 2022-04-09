package collector

type Processor interface {
	Setup(collector *Collector)
	Reset()
	Collect(callback chan<- func())
}
