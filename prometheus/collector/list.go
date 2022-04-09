package collector

import (
	"sync"
)

var (
	collectorListLock sync.Mutex
	collectorList     map[string]*Collector
)

func GetList() map[string]*Collector {
	return collectorList
}

func addCollectorToList(collector *Collector) {
	collectorListLock.Lock()
	defer collectorListLock.Unlock()
	collectorList[collector.Name] = collector
}

func init() {
	collectorList = map[string]*Collector{}
}
