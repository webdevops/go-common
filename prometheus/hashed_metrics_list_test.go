package prometheus

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func Test_HashedMetricsList(t *testing.T) {
	m := NewHashedMetricsList()
	hashedMetricsListGenerateMetrics(t, m)
	hashedMetricsListTestList(t, m)
}

func hashedMetricsListGenerateMetrics(t *testing.T, m *HashedMetricList) {
	expectHashedListCount(t, m, 0)
	m.Inc(prometheus.Labels{"key": "info", "foo": "bar"})
	expectHashedListCount(t, m, 1)

	m.Inc(prometheus.Labels{"key": "info", "foo": "bar"})
	expectHashedListCount(t, m, 1)

	m.Inc(prometheus.Labels{"key": "test", "foo": "bar"})
	expectHashedListCount(t, m, 2)

	m.Inc(prometheus.Labels{"key": "info", "foo": "bar"})
	expectHashedListCount(t, m, 2)

	m.Inc(prometheus.Labels{"key": "test", "foo": "bar"})
	expectHashedListCount(t, m, 2)

	m.Inc(prometheus.Labels{"key": "test2", "foo": "bar"})
	expectHashedListCount(t, m, 3)

	m.Inc(prometheus.Labels{"key": "info", "foo": "bar"})
	expectHashedListCount(t, m, 3)
}

func hashedMetricsListTestList(t *testing.T, m *HashedMetricList) {
	var infoMetric, testMetric, test2Metric *MetricRow

	for _, v := range m.GetList() {
		row := v
		switch row.Labels["key"] {
		case "info":
			infoMetric = &row
		case "test":
			testMetric = &row
		case "test2":
			test2Metric = &row
		}
	}

	if infoMetric != nil {
		expectMetricRowLabel(t, *infoMetric, "key", "info")
		expectMetricRowValue(t, *infoMetric, 4)
	} else {
		t.Errorf("info metric no found")
	}

	if testMetric != nil {
		expectMetricRowLabel(t, *testMetric, "key", "test")
		expectMetricRowValue(t, *testMetric, 2)
	} else {
		t.Errorf("test metric no found")
	}
	if test2Metric != nil {
		expectMetricRowLabel(t, *test2Metric, "key", "test2")
		expectMetricRowValue(t, *test2Metric, 1)
	} else {
		t.Errorf("test metric no found")
	}
}

func expectHashedListCount(t *testing.T, m *HashedMetricList, expectedCount int) {
	list := m.GetList()

	itemCount := len(list)
	if itemCount != expectedCount {
		t.Errorf("Expected item count: %v  Actual item count: %v", expectedCount, itemCount)
	}
}
