package collector

import (
	prometheusCommon "github.com/webdevops/go-common/prometheus"
)

type (
	MetricList struct {
		*prometheusCommon.MetricList

		vec   interface{}
		reset bool
	}
)
