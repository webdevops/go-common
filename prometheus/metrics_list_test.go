package prometheus

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func Test_MetricsList(t *testing.T) {
	m := NewMetricsList()
	metricsListGenerateMetrics(t, m)
	metricsListTestList(t, m)
}

func Test_MetricsListCache(t *testing.T) {
	ttl := time.Duration(2 * time.Second)

	metricsCache := cache.New(1*time.Minute, 1*time.Second)
	m := NewMetricsList()
	m.SetCache(metricsCache)
	metricsListGenerateMetrics(t, m)
	expectListCount(t, m, 5)

	if err := m.StoreToCache("test", ttl); err != nil {
		t.Errorf(err.Error())
	}
	expectListCount(t, m, 5)

	// load cache into existing list
	metricsListTestList(t, m)
	m.LoadFromCache("test")
	expectListCount(t, m, 5)
	metricsListTestList(t, m)

	// load cache into new list
	m2 := NewMetricsList()
	m2.SetCache(metricsCache)
	expectListCount(t, m2, 0)
	m2.LoadFromCache("test")
	expectListCount(t, m2, 5)
	metricsListTestList(t, m2)

	time.Sleep(time.Duration(1 * time.Second))
	m2.LoadFromCache("test")
	expectListCount(t, m2, 5)
	metricsListTestList(t, m2)

	time.Sleep(time.Duration(1 * time.Second))
	time.Sleep(time.Duration(1 * time.Second))

	// load expired cache into existing list
	m2.LoadFromCache("test")
	expectListCount(t, m2, 0)

	// load expired cache into new list
	m3 := NewMetricsList()
	m3.SetCache(metricsCache)
	expectListCount(t, m3, 0)
	m3.LoadFromCache("test")
	expectListCount(t, m3, 0)
}

func metricsListGenerateMetrics(t *testing.T, m *MetricList) {
	expectListCount(t, m, 0)
	m.AddInfo(prometheus.Labels{"key": "info"})
	expectListCount(t, m, 1)
	m.Add(prometheus.Labels{"key": "custom"}, 123)
	expectListCount(t, m, 2)
	m.AddDuration(prometheus.Labels{"key": "duration"}, time.Duration(42*time.Hour))
	expectListCount(t, m, 3)
	m.AddIfGreaterZero(prometheus.Labels{"key": "not existing"}, 0)
	expectListCount(t, m, 3)
	m.AddIfGreaterZero(prometheus.Labels{"key": "not existing"}, -12)
	expectListCount(t, m, 3)
	m.AddIfGreaterZero(prometheus.Labels{"key": "existing"}, 12)
	expectListCount(t, m, 4)

	loc, err := time.LoadLocation("UTC")
	if err != nil {
		t.Errorf("Error occurred during loading location UTC: %v", err)
	}
	timestamp := time.Date(2020, 01, 01, 0, 0, 0, 0, loc)
	m.AddTime(prometheus.Labels{"key": "timestamp"}, timestamp)
	expectListCount(t, m, 5)
}

func metricsListTestList(t *testing.T, m *MetricList) {
	expectMetricRowLabel(t, m.GetList()[0], "key", "info")
	expectMetricRowValue(t, m.GetList()[0], 1)

	expectMetricRowLabel(t, m.GetList()[1], "key", "custom")
	expectMetricRowValue(t, m.GetList()[1], 123)

	expectMetricRowLabel(t, m.GetList()[2], "key", "duration")
	expectMetricRowValue(t, m.GetList()[2], 151200)

	expectMetricRowLabel(t, m.GetList()[3], "key", "existing")
	expectMetricRowValue(t, m.GetList()[3], 12)

	expectMetricRowLabel(t, m.GetList()[4], "key", "timestamp")
	expectMetricRowValue(t, m.GetList()[4], 1577836800)
}

func expectListCount(t *testing.T, m *MetricList, expectedCount int) {
	t.Helper()

	list := m.GetList()

	itemCount := len(list)
	if itemCount != expectedCount {
		t.Errorf("Expected item count: %v  Actual item count: %v", expectedCount, itemCount)
	}
}

func expectMetricRowLabel(t *testing.T, m MetricRow, expectedLabel, expectedValue string) {
	t.Helper()

	for label, value := range m.labels {
		if label == expectedLabel && value == expectedValue {
			return
		}
	}

	t.Errorf("Expected label %v with value %v not found", expectedLabel, expectedValue)
}

func expectMetricRowValue(t *testing.T, m MetricRow, expectedValue float64) {
	t.Helper()

	if m.value != expectedValue {
		t.Errorf("Expected metric value: %v  Actual metric value: %v", expectedValue, m.value)
	}
}
