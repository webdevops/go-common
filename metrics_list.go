package prometheus_common

import (
	"github.com/patrickmn/go-cache"
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
	mux  *sync.Mutex
}

func NewMetricsList() *MetricList {
	m := MetricList{}
	m.Init()
	return &m
}

func (m *MetricList) Init() {
	m.mux = &sync.Mutex{}
}

func (m *MetricList) append(row MetricRow) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.list = append(m.list, row)
}

func (m *MetricList) LoadFromCache(cache *cache.Cache, key string) bool {
	m.Reset()

	m.mux.Lock()
	defer m.mux.Unlock()

	if val, fetched := cache.Get(key); fetched {
		// loaded from cache
		m.list = val.([]MetricRow)
		return true
	}

	return false
}

func (m *MetricList) StoreToCache(cache *cache.Cache, key string, duration time.Duration) {
	cache.Add(key, m.GetList(), duration)
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

func (m *MetricList) GetList() []MetricRow {
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
