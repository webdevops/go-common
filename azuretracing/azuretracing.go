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
	hostnameMaxParts   = 3
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
			"routingRegion",
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
			if hostnameParts := strings.Split(hostname, "."); len(hostnameParts) > hostnameMaxParts {
				hostname = strings.Join(hostnameParts[len(hostnameParts)-hostnameMaxParts:], ".")
			}

			// try to detect subscriptionId from url
			subscriptionId := ""
			if matches := subscriptionRegexp.FindStringSubmatch(path); len(matches) >= 2 {
				subscriptionId = strings.ToLower(matches[1])
			}

			routingRegion := ""
			if headerValue := r.Header.Get("x-ms-routing-request-id"); headerValue != "" {
				if headerValueParts := strings.Split(headerValue, ":"); len(headerValueParts) >= 1 {
					routingRegion = headerValueParts[0]
				}
			} else if headerValue := r.Header.Get("x-ms-keyvault-region"); headerValue != "" {
				routingRegion = headerValue
			}

			// collect request and latency
			if startTime, ok := r.Request.Context().Value(contextTracingName).(time.Time); ok {
				azureApiRequest.With(prometheus.Labels{
					"endpoint":      hostname,
					"routingRegion": strings.ToLower(routingRegion),
					"method":        strings.ToLower(r.Request.Method),
					"statusCode":    strconv.FormatInt(int64(r.StatusCode), 10),
				}).Observe(time.Since(startTime).Seconds())
			}

			collectAzureApiRateLimitMetric := func(r *http.Response, headerName string, scopeLabel, typeLabel string) {
				ratelimit := r.Header.Get(headerName)
				if v, err := strconv.ParseInt(ratelimit, 10, 64); err == nil {
					azureApiRatelimit.With(prometheus.Labels{
						"endpoint":       hostname,
						"subscriptionID": subscriptionId,
						"scope":          scopeLabel,
						"type":           typeLabel,
					}).Set(float64(v))
				}
			}

			// subscription rate limits
			collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-subscription-reads", "subscription", "reads")
			collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-subscription-resource-requests", "subscription", "resource-requests")
			collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-subscription-resource-entities-read", "subscription", "resource-entities-read")

			// tenant rate limits
			collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-tenant-reads", "tenant", "reads")
			collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-tenant-resource-requests", "tenant", "resource-requests")
			collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-tenant-resource-entities-read", "tenant", "resource-entities-read")

			return nil
		})
	}
}
