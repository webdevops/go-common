package prometheus

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	cache "github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
)

type HashedMetricList struct {
	List map[string]*MetricRow
	mux  *sync.Mutex

	metricsCache *cache.Cache
}

func NewHashedMetricsList() *HashedMetricList {
	m := HashedMetricList{}
	m.Init()
	return &m
}

func (m *HashedMetricList) Init() {
	m.mux = &sync.Mutex{}
	m.Reset()
}

func (m *HashedMetricList) SetCache(instance *cache.Cache) {
	m.metricsCache = instance
}

func (m *HashedMetricList) LoadFromCache(key string) bool {
	m.Reset()

	if m.metricsCache != nil {
		m.mux.Lock()
		defer m.mux.Unlock()

		if val, fetched := m.metricsCache.Get(key); fetched {
			// loaded from cache
			m.List = val.(map[string]*MetricRow)
			return true
		}
	}

	return false
}

func (m *HashedMetricList) StoreToCache(key string, duration time.Duration) error {
	if m.metricsCache != nil {
		return m.metricsCache.Add(key, m.GetList(), duration)
	}

	return nil
}

func (m *HashedMetricList) Reset() {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.List = map[string]*MetricRow{}
}

func (m *HashedMetricList) GetList() []MetricRow {
	m.mux.Lock()
	defer m.mux.Unlock()

	list := []MetricRow{}
	for _, row := range m.List {
		list = append(list, *row)
	}

	return list
}

func (m *HashedMetricList) Inc(labels prometheus.Labels) {
	m.mux.Lock()
	defer m.mux.Unlock()

	metricKey := ""
	for key, value := range labels {
		metricKey = metricKey + key + "=" + value + ";"
	}
	hashKey := fmt.Sprintf("%x", sha256.Sum256([]byte(metricKey)))
	if _, exists := m.List[hashKey]; exists {
		m.List[hashKey].Value++
	} else {
		m.List[hashKey] = &MetricRow{
			Labels: labels,
			Value:  1,
		}
	}
}

func (m *HashedMetricList) GaugeSet(gauge *prometheus.GaugeVec) {
	for _, metric := range m.GetList() {
		gauge.With(metric.Labels).Set(metric.Value)
	}
}

func (m *HashedMetricList) CounterAdd(counter *prometheus.CounterVec) {
	for _, metric := range m.GetList() {
		counter.With(metric.Labels).Add(metric.Value)
	}
}
