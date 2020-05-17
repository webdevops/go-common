package prometheus_common

import (
	"github.com/muesli/cache2go"
	"github.com/prometheus/client_golang/prometheus"
	"sync"
	"time"
)

type MetricRow struct {
	labels prometheus.Labels
	value  float64
}

type MetricList struct {
	list []MetricRow
	mux sync.Mutex
}

func (m *MetricList) append(row MetricRow) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.append(row)
}

func (m *MetricList) LoadFromCache(cachetable *cache2go.CacheTable, key string) (bool) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if val, err := cachetable.Value(key); err == nil {
		// loaded from cache
		m.list = val.Data().([]MetricRow)
		return true
	}
	
	return false
}

func (m *MetricList) StoreToCache(cachetable *cache2go.CacheTable, key string, duration time.Duration) {
	cachetable.Add(key, duration, m.GetList())
}

func (m *MetricList) Add(labels prometheus.Labels, value float64) {
	m.append(MetricRow{labels: labels, value: value})
}

func (m *MetricList) AddInfo(labels prometheus.Labels) {
	m.append(MetricRow{labels: labels, value: 1})
}

func (m *MetricList) AddIfNotZero(labels prometheus.Labels, value float64) {
	if value != 0 {
		m.append(MetricRow{labels: labels, value: value})
	}
}

func (m *MetricList) AddIfGreaterZero(labels prometheus.Labels, value float64) {
	if value > 0 {
		m.append(MetricRow{labels: labels, value: value})
	}
}

func (m *MetricList) AddTime(labels prometheus.Labels, value time.Time) {
	timeValue := timeToFloat64(value)

	if timeValue > 0 {
		m.append(MetricRow{labels: labels, value: timeValue})
	}
}

func (m *MetricList) AddDuration(labels prometheus.Labels, value time.Duration) {
	m.append(MetricRow{labels: labels, value: value.Seconds()})
}

func (m *MetricList) Reset() {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.list = []MetricRow{}
}

func (m *MetricList) GetList() ([]MetricRow) {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.list
}

func (m *MetricList) GaugeSet(gauge *prometheus.GaugeVec) {
	for _, metric := range m.GetList() {
		gauge.With(metric.labels).Set(metric.value)
	}
}

func (m *MetricList) CounterAdd(counter *prometheus.CounterVec) {
	for _, metric := range m.GetList() {
		counter.With(metric.labels).Add(metric.value)
	}
}
