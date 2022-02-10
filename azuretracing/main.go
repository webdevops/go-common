package azuretracing

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	contextTracingName = "webdevops:prom:tracing"
)

var (
	azureApiRequest   *prometheus.HistogramVec
	azureApiRatelimit *prometheus.GaugeVec

	subscriptionRegexp = regexp.MustCompile(`^/subscriptions/([^/]+)/?.*$`)
)

func init() {
	azureApiRequest = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "azurerm_api_request",
			Help:    "AzureRM API requests",
			Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60},
		},
		[]string{
			"endpoint",
			"method",
			"statusCode",
		},
	)
	prometheus.MustRegister(azureApiRequest)

	azureApiRatelimit = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_api_ratelimit",
			Help: "AzureRM API ratelimit",
		},
		[]string{
			"endpoint",
			"subscriptionID",
			"scope",
			"type",
		},
	)
	prometheus.MustRegister(azureApiRatelimit)
}

func RegisterAzureMetricAutoClean(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
		azureApiRatelimit.Reset()
	})
}

func DecoreAzureAutoRest(client *autorest.Client) {
	collectAzureApiRateLimitMetric := func(r *http.Response, headerName string, labels prometheus.Labels) {
		ratelimit := r.Header.Get(headerName)
		if v, err := strconv.ParseInt(ratelimit, 10, 64); err == nil {
			azureApiRatelimit.With(labels).Set(float64(v))
		}
	}

	client.RequestInspector = func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err == nil {
				ctx := r.Context()
				ctx = context.WithValue(ctx, contextTracingName, time.Now().UTC())
				r = r.WithContext(ctx)
			}
			return r, err
		})
	}

	client.ResponseInspector = func(p autorest.Responder) autorest.Responder {
		return autorest.ResponderFunc(func(r *http.Response) error {
			hostname := strings.ToLower(r.Request.URL.Hostname())
			path := r.Request.URL.Path

			// try to detect subscriptionId from url
			subscriptionId := ""
			if matches := subscriptionRegexp.FindStringSubmatch(path); len(matches) >= 2 {
				subscriptionId = strings.ToLower(matches[1])
			}

			// collect request and latency
			if startTime, ok := r.Request.Context().Value(contextTracingName).(time.Time); ok {
				azureApiRequest.With(prometheus.Labels{
					"endpoint":   hostname,
					"method":     strings.ToLower(r.Request.Method),
					"statusCode": strconv.FormatInt(int64(r.StatusCode), 10),
				}).Observe(time.Since(startTime).Seconds())
			}

			// subscription rate limits
			collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-subscription-reads", prometheus.Labels{"endpoint": hostname, "subscriptionID": subscriptionId, "scope": "subscription", "type": "read"})
			collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-subscription-resource-requests", prometheus.Labels{"endpoint": hostname, "subscriptionID": subscriptionId, "scope": "subscription", "type": "resource-requests"})
			collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-subscription-resource-entities-read", prometheus.Labels{"endpoint": hostname, "subscriptionID": subscriptionId, "scope": "subscription", "type": "resource-entities-read"})

			// tenant rate limits
			collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-tenant-reads", prometheus.Labels{"endpoint": hostname, "subscriptionID": subscriptionId, "scope": "tenant", "type": "read"})
			collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-tenant-resource-requests", prometheus.Labels{"endpoint": hostname, "subscriptionID": subscriptionId, "scope": "tenant", "type": "resource-requests"})
			collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-tenant-resource-entities-read", prometheus.Labels{"endpoint": hostname, "subscriptionID": subscriptionId, "scope": "tenant", "type": "resource-entities-read"})
			return nil
		})
	}
}
