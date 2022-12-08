package collector

import (
	"time"

	prometheusCommon "github.com/webdevops/go-common/prometheus"
)

type (
	Metrics struct {
		Expiry *time.Time             `json:"expiry"`
		List   map[string]*MetricList `json:"metrics"`
	}

	MetricList struct {
		*prometheusCommon.MetricList

		vec   interface{}
		reset bool
	}
)

func NewMetrics() *Metrics {
	return &Metrics{
		Expiry: nil,
		List:   map[string]*MetricList{},
	}
}
