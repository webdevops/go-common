package azure_tracing

import (
	"context"
	"github.com/Azure/go-autorest/tracing"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"strconv"
	"time"
)

const (
	contextTracingName = "webdevops:prometheus:tracing"
)

type (
	azureTracer struct{}
)

var (
	azureApiMetric *prometheus.HistogramVec
)

func Enable() {
	azureApiMetric = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "azurerm_api_requests",
			Help:    "AzureRM API requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"statusCode"},
	)
	prometheus.MustRegister(azureApiMetric)

	tracer := azureTracer{}
	tracing.Register(tracer)
}

func (t azureTracer) NewTransport(base *http.Transport) http.RoundTripper {
	return base
}

func (t azureTracer) StartSpan(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, contextTracingName, time.Now().UTC())
}

func (t azureTracer) EndSpan(ctx context.Context, httpStatusCode int, err error) {
	if httpStatusCode <= 0 {
		return
	}

	if startTime, ok := ctx.Value(contextTracingName).(time.Time); ok {
		azureApiMetric.WithLabelValues(strconv.FormatInt(int64(httpStatusCode), 10)).Observe(time.Since(startTime).Seconds())
	}
	return
}
