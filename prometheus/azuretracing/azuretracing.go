package azuretracing

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	ContextTracingName = "webdevops:prom:tracing"
	hostnameMaxParts   = 3

	envVarApiRequestBuckets     = "METRIC_AZURERM_API_REQUEST_BUCKETS"
	envVarApiRequestEnabled     = "METRIC_AZURERM_API_REQUEST_ENABLE"
	envVarApiRequestLables      = "METRIC_AZURERM_API_REQUEST_LABELS"
	envVarApiRatelimitEnabled   = "METRIC_AZURERM_API_RATELIMIT_ENABLE"
	envVarApiRatelimitAutoreset = "METRIC_AZURERM_API_RATELIMIT_AUTORESET"
)

type (
	azureTracing struct {
		settings struct {
			azureApiRequest struct {
				enabled bool
				buckets []float64

				labels struct {
					apiEndpoint      bool
					routingRegion    bool
					subscriptionID   bool
					tenantID         bool
					resourceProvider bool
					method           bool
					statusCode       bool
				}
			}

			azureApiRatelimit struct {
				enabled   bool
				autoreset bool
			}
		}

		prometheus struct {
			azureApiRequest   *prometheus.HistogramVec
			azureApiRatelimit *prometheus.GaugeVec
		}
	}

	// azure jwt auth token from header
	azureJwtAuthToken struct {
		Aud      string   `json:"aud"`
		Iss      string   `json:"iss"`
		Iat      int      `json:"iat"`
		Nbf      int      `json:"nbf"`
		Exp      int      `json:"exp"`
		Aio      string   `json:"aio"`
		Appid    string   `json:"appid"`
		Appidacr string   `json:"appidacr"`
		Groups   []string `json:"groups"`
		Idp      string   `json:"idp"`
		Idtyp    string   `json:"idtyp"`
		Oid      string   `json:"oid"`
		Rh       string   `json:"rh"`
		Sub      string   `json:"sub"`
		Tid      string   `json:"tid"`
		Uti      string   `json:"uti"`
		Ver      string   `json:"ver"`
		XmsTcdt  int      `json:"xms_tcdt"`
	}
)

var (
	AzureTracing *azureTracing

	DefBuckets = []float64{1, 2.5, 5, 10, 30, 60, 90, 120}

	envVarSplit        = regexp.MustCompile(`([\s,]+)`)
	subscriptionRegexp = regexp.MustCompile(`^/subscriptions/([^/]+)/?.*$`)
	providerRegexp     = regexp.MustCompile(`^/subscriptions/[^/]+/resourcegroups/[^/]+/providers/([^/]+/[^/]+)/.*$`)
)

func init() {
	AzureTracing = &azureTracing{}

	// azureApiRequest settings
	AzureTracing.settings.azureApiRequest.enabled = checkIfEnvVarIsEnabled(envVarApiRequestEnabled, true)
	AzureTracing.settings.azureApiRequest.labels.apiEndpoint = checkIfEnvVarContains(envVarApiRequestLables, "apiEndpoint", true)
	AzureTracing.settings.azureApiRequest.labels.routingRegion = checkIfEnvVarContains(envVarApiRequestLables, "routingRegion", true)
	AzureTracing.settings.azureApiRequest.labels.subscriptionID = checkIfEnvVarContains(envVarApiRequestLables, "subscriptionID", true)
	AzureTracing.settings.azureApiRequest.labels.tenantID = checkIfEnvVarContains(envVarApiRequestLables, "tenantID", true)
	AzureTracing.settings.azureApiRequest.labels.resourceProvider = checkIfEnvVarContains(envVarApiRequestLables, "resourceProvider", true)
	AzureTracing.settings.azureApiRequest.labels.method = checkIfEnvVarContains(envVarApiRequestLables, "method", true)
	AzureTracing.settings.azureApiRequest.labels.statusCode = checkIfEnvVarContains(envVarApiRequestLables, "statusCode", true)

	AzureTracing.settings.azureApiRequest.buckets = DefBuckets
	if envVal := os.Getenv(envVarApiRequestBuckets); envVal != "" {
		AzureTracing.settings.azureApiRequest.buckets = []float64{}
		for _, bucketString := range envVarSplit.Split(envVal, -1) {
			bucketString = strings.TrimSpace(bucketString)
			if val, err := strconv.ParseFloat(bucketString, 64); err == nil {
				AzureTracing.settings.azureApiRequest.buckets = append(
					AzureTracing.settings.azureApiRequest.buckets,
					val,
				)
			} else {
				panic(fmt.Sprintf("unable to parse env var %v=\"%v\": %v", envVarApiRequestBuckets, os.Getenv(envVarApiRequestBuckets), err))
			}
		}
	}

	// azureApiRatelimit
	AzureTracing.settings.azureApiRatelimit.enabled = checkIfEnvVarIsEnabled(envVarApiRatelimitEnabled, true)
	AzureTracing.settings.azureApiRatelimit.autoreset = checkIfEnvVarIsEnabled(envVarApiRatelimitAutoreset, true)

	if AzureTracing.settings.azureApiRequest.enabled {
		labels := []string{}

		if AzureTracing.settings.azureApiRequest.labels.apiEndpoint {
			labels = append(labels, "apiEndpoint")
		}

		if AzureTracing.settings.azureApiRequest.labels.routingRegion {
			labels = append(labels, "routingRegion")
		}

		if AzureTracing.settings.azureApiRequest.labels.subscriptionID {
			labels = append(labels, "subscriptionID")
		}

		if AzureTracing.settings.azureApiRequest.labels.tenantID {
			labels = append(labels, "tenantID")
		}

		if AzureTracing.settings.azureApiRequest.labels.resourceProvider {
			labels = append(labels, "resourceProvider")
		}

		if AzureTracing.settings.azureApiRequest.labels.method {
			labels = append(labels, "method")
		}

		if AzureTracing.settings.azureApiRequest.labels.statusCode {
			labels = append(labels, "statusCode")
		}

		AzureTracing.prometheus.azureApiRequest = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "azurerm_api_request",
				Help:    "AzureRM API requests",
				Buckets: AzureTracing.settings.azureApiRequest.buckets,
			},
			labels,
		)
		prometheus.MustRegister(AzureTracing.prometheus.azureApiRequest)
	}

	if AzureTracing.settings.azureApiRatelimit.enabled {
		AzureTracing.prometheus.azureApiRatelimit = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "azurerm_api_ratelimit",
				Help: "AzureRM API ratelimit",
			},
			[]string{
				"apiEndpoint",
				"subscriptionID",
				"tenantID",
				"scope",
				"type",
			},
		)
		prometheus.MustRegister(AzureTracing.prometheus.azureApiRatelimit)
	}
}

func RegisterAzureMetricAutoClean(handler http.Handler) http.Handler {
	if AzureTracing.prometheus.azureApiRatelimit == nil || !AzureTracing.settings.azureApiRatelimit.autoreset {
		// metric or autoreset disabled, nothing to do here
		return handler
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
		AzureTracing.prometheus.azureApiRatelimit.Reset()
	})
}

func DecorateAzureAutoRestClient(client *autorest.Client) {
	DecorateAzureAutoRestClientWithCallbacks(client, nil, nil)
}

func DecorateAzureAutoRestClientWithCallbacks(client *autorest.Client, requestInspectorCallback *func(r *http.Request) (*http.Request, error), responseInspector *func(r *http.Response) error) {
	if AzureTracing.prometheus.azureApiRequest == nil && AzureTracing.prometheus.azureApiRatelimit == nil {
		// all metrics disabled, nothing to do here
		return
	}

	client.RequestInspector = func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err == nil {
				ctx := r.Context()
				ctx = context.WithValue(ctx, ContextTracingName, time.Now().UTC()) // nolint:staticcheck
				r = r.WithContext(ctx)

				if requestInspectorCallback != nil {
					r, err = (*requestInspectorCallback)(r)
				}
			}

			return r, err
		})
	}

	client.ResponseInspector = func(p autorest.Responder) autorest.Responder {
		return autorest.ResponderFunc(func(r *http.Response) error {
			// get hostname (shorten it to 3 parts)
			hostname := strings.ToLower(r.Request.URL.Hostname())
			if hostnameParts := strings.Split(hostname, "."); len(hostnameParts) > hostnameMaxParts {
				hostname = strings.Join(hostnameParts[len(hostnameParts)-hostnameMaxParts:], ".")
			}

			// path with trimmed / at start (could be multiple)
			path := strings.ToLower("/" + strings.TrimLeft(r.Request.URL.Path, "/"))

			// try to detect subscriptionId from url
			subscriptionId := ""
			if matches := subscriptionRegexp.FindStringSubmatch(path); len(matches) >= 2 {
				subscriptionId = strings.ToLower(matches[1])
			}

			// try to detect subscriptionId from url
			resourceProvider := ""
			if matches := providerRegexp.FindStringSubmatch(path); len(matches) >= 2 {
				resourceProvider = strings.ToLower(matches[1])
			}

			tenantId := extractTenantIdFromRequest(r)

			routingRegion := ""
			if headerValue := r.Header.Get("x-ms-routing-request-id"); headerValue != "" {
				if headerValueParts := strings.Split(headerValue, ":"); len(headerValueParts) >= 1 {
					routingRegion = headerValueParts[0]
				}
			} else if headerValue := r.Header.Get("x-ms-keyvault-region"); headerValue != "" {
				routingRegion = headerValue
			}

			// collect request and latency
			if AzureTracing.prometheus.azureApiRequest != nil {
				if startTime, ok := r.Request.Context().Value(ContextTracingName).(time.Time); ok {
					requestLabels := prometheus.Labels{}

					if AzureTracing.settings.azureApiRequest.labels.apiEndpoint {
						requestLabels["apiEndpoint"] = hostname
					}

					if AzureTracing.settings.azureApiRequest.labels.routingRegion {
						requestLabels["routingRegion"] = strings.ToLower(routingRegion)
					}

					if AzureTracing.settings.azureApiRequest.labels.subscriptionID {
						requestLabels["subscriptionID"] = subscriptionId
					}

					if AzureTracing.settings.azureApiRequest.labels.tenantID {
						requestLabels["tenantID"] = tenantId
					}

					if AzureTracing.settings.azureApiRequest.labels.resourceProvider {
						requestLabels["resourceProvider"] = resourceProvider
					}

					if AzureTracing.settings.azureApiRequest.labels.method {
						requestLabels["method"] = strings.ToLower(r.Request.Method)
					}

					if AzureTracing.settings.azureApiRequest.labels.statusCode {
						requestLabels["statusCode"] = strconv.FormatInt(int64(r.StatusCode), 10)
					}

					AzureTracing.prometheus.azureApiRequest.With(requestLabels).Observe(time.Since(startTime).Seconds())
				}
			}

			if AzureTracing.prometheus.azureApiRatelimit != nil {
				collectAzureApiRateLimitMetric := func(r *http.Response, headerName string, scopeLabel, typeLabel string) {
					ratelimit := r.Header.Get(headerName)
					if v, err := strconv.ParseInt(ratelimit, 10, 64); err == nil {
						AzureTracing.prometheus.azureApiRatelimit.With(prometheus.Labels{
							"apiEndpoint":    hostname,
							"subscriptionID": subscriptionId,
							"tenantID":       tenantId,
							"scope":          scopeLabel,
							"type":           typeLabel,
						}).Set(float64(v))
					}
				}

				// special resourcegraph limits
				if strings.HasPrefix(path, "/providers/microsoft.resourcegraph/") {
					collectAzureApiRateLimitMetric(r, "x-ms-user-quota-remaining", "resourcegraph", "quota")
				}

				// subscription rate limits
				collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-subscription-reads", "subscription", "reads")
				collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-subscription-writes", "subscription", "writes")
				collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-subscription-resource-requests", "subscription", "resource-requests")
				collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-subscription-resource-entities-read", "subscription", "resource-entities-read")

				// tenant rate limits
				collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-tenant-reads", "tenant", "reads")
				collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-tenant-writes", "tenant", "writes")
				collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-tenant-resource-requests", "tenant", "resource-requests")
				collectAzureApiRateLimitMetric(r, "x-ms-ratelimit-remaining-tenant-resource-entities-read", "tenant", "resource-entities-read")
			}

			if responseInspector != nil {
				return (*responseInspector)(r)
			}

			return nil
		})
	}
}

func extractTenantIdFromRequest(r *http.Response) string {
	authToken := r.Request.Header.Get("authorization")
	if strings.HasPrefix(authToken, "Bearer") {
		authToken = strings.TrimSpace(strings.TrimPrefix(authToken, "Bearer"))
		authTokenParts := strings.Split(authToken, ".")
		if len(authTokenParts) == 3 {
			if val, err := base64.RawURLEncoding.DecodeString(authTokenParts[1]); err == nil {
				jwt := azureJwtAuthToken{}
				if err := json.Unmarshal(val, &jwt); err == nil {
					return jwt.Tid
				}
			}
		}
	}

	return ""
}

func checkIfEnvVarContains(name string, value string, defaultVal bool) bool {
	envVal := strings.TrimSpace(os.Getenv(name))

	if envVal != "" {
		for _, part := range envVarSplit.Split(envVal, -1) {
			if strings.EqualFold(part, value) {
				return true
			}
		}

		return false
	}

	return defaultVal
}

func checkIfEnvVarIsEnabled(name string, defaultVal bool) bool {
	status := defaultVal

	val := os.Getenv(name)
	val = strings.ToLower(strings.TrimSpace(val))

	switch val {
	case "1", "true", "y", "yes", "enabled":
		status = true

	case "0", "false", "n", "no", "disabled":
		status = false
	}

	return status
}
