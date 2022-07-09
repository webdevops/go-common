package prometheus

import (
	"sync"
	"time"

	cache "github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/webdevops/go-common/utils/to"
)

type MetricRow struct {
	labels prometheus.Labels
	value  float64
}

type MetricList struct {
	list []MetricRow
	mux  *sync.Mutex

	metricsCache *cache.Cache
}

func NewMetricsList() *MetricList {
	m := MetricList{}
	m.Init()
	return &m
}

func (m *MetricList) Init() {
	m.mux = &sync.Mutex{}
}

func (m *MetricList) SetCache(instance *cache.Cache) {
	m.metricsCache = instance
}

func (m *MetricList) append(row MetricRow) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.list = append(m.list, row)
}

func (m *MetricList) LoadFromCache(key string) bool {
	m.Reset()

	if m.metricsCache != nil {
		m.mux.Lock()
		defer m.mux.Unlock()

		if val, fetched := m.metricsCache.Get(key); fetched {
			// loaded from cache
			m.list = val.([]MetricRow)
			return true
		}
	}

	return false
}

func (m *MetricList) StoreToCache(key string, duration time.Duration) error {
	if m.metricsCache != nil {
		return m.metricsCache.Add(key, m.GetList(), duration)
	}
	return nil
}

func (m *MetricList) Add(labels prometheus.Labels, value float64) {
	m.append(MetricRow{labels: labels, value: value})
}

func (m *MetricList) AddInfo(labels prometheus.Labels) {
	m.append(MetricRow{labels: labels, value: 1})
}

func (m *MetricList) AddIfNotNil(labels prometheus.Labels, value *float64) {
	if value != nil {
		m.append(MetricRow{labels: labels, value: *value})
	}
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
	timeValue := to.UnixTime(value)

	if timeValue > 0 {
		m.append(MetricRow{labels: labels, value: timeValue})
	}
}

func (m *MetricList) AddDuration(labels prometheus.Labels, value time.Duration) {
	m.append(MetricRow{labels: labels, value: value.Seconds()})
}

func (m *MetricList) AddBool(labels prometheus.Labels, state bool) {
	value := float64(0)
	if state {
		value = 1
	}

	m.append(MetricRow{labels: labels, value: value})
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

func (m *MetricList) GaugeSetInc(gauge *prometheus.GaugeVec) {
	for _, metric := range m.GetList() {
		if metricGauge, err := gauge.GetMetricWith(metric.labels); err == nil {
			metricGauge.Add(metric.value)
		} else {
			panic(err)
		}
	}
}

func (m *MetricList) SummarySet(summary *prometheus.SummaryVec) {
	for _, metric := range m.GetList() {
		summary.With(metric.labels).Observe(metric.value)
	}
}

func (m *MetricList) HistogramSet(histogram *prometheus.HistogramVec) {
	for _, metric := range m.GetList() {
		histogram.With(metric.labels).Observe(metric.value)
	}
}

func (m *MetricList) CounterAdd(counter *prometheus.CounterVec) {
	for _, metric := range m.GetList() {
		counter.With(metric.labels).Add(metric.value)
	}
}
