package collector

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "collector_info",
			Help: "Collector info",
		},
		[]string{
			"collector",
		},
	)

	metricPanicCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "collector_panics",
			Help: "Collector run duration",
		},
		[]string{
			"collector",
		},
	)

	metricDuration = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "collector_duration_seconds",
			Help: "Collector run duration",
		},
		[]string{
			"collector",
		},
	)

	metricSuccess = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "collector_success",
			Help: "Collector success status",
		},
		[]string{
			"collector",
		},
	)

	metricLastCollect = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "collector_collect_timestamp_seconds",
			Help: "Collector last collected timestamp",
		},
		[]string{
			"collector",
		},
	)
)

func init() {
	prometheus.MustRegister(
		metricInfo,
		metricPanicCount,
		metricDuration,
		metricSuccess,
		metricLastCollect,
	)
}
