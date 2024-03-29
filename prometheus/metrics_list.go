package prometheus

import (
	"sync"
	"time"

	cache "github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/webdevops/go-common/utils/to"
)

type MetricRow struct {
	Labels prometheus.Labels `json:"labels"`
	Value  float64           `json:"value"`
}

type MetricList struct {
	List []MetricRow `json:"list"`
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

	if m.List == nil {
		m.List = []MetricRow{}
	}
}

func (m *MetricList) SetCache(instance *cache.Cache) {
	m.metricsCache = instance
}

func (m *MetricList) append(row MetricRow) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if m.List == nil {
		m.List = []MetricRow{}
	}

	m.List = append(m.List, row)
}

func (m *MetricList) LoadFromCache(key string) bool {
	m.Reset()

	if m.metricsCache != nil {
		m.mux.Lock()
		defer m.mux.Unlock()

		if val, fetched := m.metricsCache.Get(key); fetched {
			// loaded from cache
			m.List = val.([]MetricRow)
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
	m.append(MetricRow{Labels: labels, Value: value})
}

func (m *MetricList) AddInfo(labels prometheus.Labels) {
	m.append(MetricRow{Labels: labels, Value: 1})
}

func (m *MetricList) AddIfNotNil(labels prometheus.Labels, value *float64) {
	if value != nil {
		m.append(MetricRow{Labels: labels, Value: *value})
	}
}

func (m *MetricList) AddIfNotZero(labels prometheus.Labels, value float64) {
	if value != 0 {
		m.append(MetricRow{Labels: labels, Value: value})
	}
}

func (m *MetricList) AddIfGreaterZero(labels prometheus.Labels, value float64) {
	if value > 0 {
		m.append(MetricRow{Labels: labels, Value: value})
	}
}

func (m *MetricList) AddTime(labels prometheus.Labels, value time.Time) {
	timeValue := to.UnixTime(value)

	if timeValue > 0 {
		m.append(MetricRow{Labels: labels, Value: timeValue})
	}
}

func (m *MetricList) AddDuration(labels prometheus.Labels, value time.Duration) {
	m.append(MetricRow{Labels: labels, Value: value.Seconds()})
}

func (m *MetricList) AddBool(labels prometheus.Labels, state bool) {
	value := float64(0)
	if state {
		value = 1
	}

	m.append(MetricRow{Labels: labels, Value: value})
}

func (m *MetricList) Reset() {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.List = []MetricRow{}
}

func (m *MetricList) GetList() []MetricRow {
	m.mux.Lock()
	defer m.mux.Unlock()

	if m.List == nil {
		m.List = []MetricRow{}
	}

	return m.List
}

func (m *MetricList) GaugeSet(gauge *prometheus.GaugeVec) {
	for _, metric := range m.GetList() {
		gauge.With(metric.Labels).Set(metric.Value)
	}
}

func (m *MetricList) GaugeSetInc(gauge *prometheus.GaugeVec) {
	for _, metric := range m.GetList() {
		if metricGauge, err := gauge.GetMetricWith(metric.Labels); err == nil {
			metricGauge.Add(metric.Value)
		} else {
			panic(err)
		}
	}
}

func (m *MetricList) SummarySet(summary *prometheus.SummaryVec) {
	for _, metric := range m.GetList() {
		summary.With(metric.Labels).Observe(metric.Value)
	}
}

func (m *MetricList) HistogramSet(histogram *prometheus.HistogramVec) {
	for _, metric := range m.GetList() {
		histogram.With(metric.Labels).Observe(metric.Value)
	}
}

func (m *MetricList) CounterAdd(counter *prometheus.CounterVec) {
	for _, metric := range m.GetList() {
		counter.With(metric.Labels).Add(metric.Value)
	}
}
