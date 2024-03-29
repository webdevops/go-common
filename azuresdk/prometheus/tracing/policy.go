package tracing

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/prometheus/client_golang/prometheus"
)

func NewTracingPolicy() tracingPolicy {
	return tracingPolicy{}
}

type tracingPolicy struct{}

func (p tracingPolicy) Do(req *policy.Request) (*http.Response, error) {
	// Mutate/process request.
	start := time.Now()
	// Forward the request to the next policy in the pipeline.
	res, err := req.Next()

	if res == nil {
		return res, err
	}

	requestDuration := time.Since(start)

	// get hostname (shorten it to 3 parts)
	hostname := strings.ToLower(req.Raw().Host)
	if hostnameParts := strings.Split(hostname, "."); len(hostnameParts) > HostnameMaxParts {
		hostname = strings.Join(hostnameParts[len(hostnameParts)-HostnameMaxParts:], ".")
	}

	// path with trimmed / at start (could be multiple)
	path := strings.ToLower("/" + strings.TrimLeft(res.Request.URL.Path, "/"))

	// try to detect subscriptionId from url
	subscriptionId := ""
	if matches := subscriptionRegexp.FindStringSubmatch(path); len(matches) >= 2 {
		subscriptionId = strings.ToLower(matches[1])
	}

	// try to detect subscriptionId from url
	resourceProvider := ""
	if matches := providerRegexp.FindStringSubmatch(path); len(matches) >= 3 {
		resourceProvider = strings.ToLower(matches[2])
	}

	tenantId := extractTenantIdFromRequest(res)

	routingRegion := ""
	if headerValue := res.Header.Get("x-ms-routing-request-id"); headerValue != "" {
		if headerValueParts := strings.Split(headerValue, ":"); len(headerValueParts) >= 1 {
			routingRegion = headerValueParts[0]
		}
	} else if headerValue := res.Header.Get("x-ms-keyvault-region"); headerValue != "" {
		routingRegion = headerValue
	}

	// collect request and latency
	if prometheusAzureApiRequest != nil {
		requestLabels := prometheus.Labels{}

		if tracingLabelsApiEndpoint {
			requestLabels["apiEndpoint"] = hostname
		}

		if tracingLabelsRoutingRegion {
			requestLabels["routingRegion"] = strings.ToLower(routingRegion)
		}

		if tracingLabelsSubscriptionID {
			requestLabels["subscriptionID"] = subscriptionId
		}

		if tracingLabelsTenantID {
			requestLabels["tenantID"] = tenantId
		}

		if tracingLabelsResourceProvider {
			requestLabels["resourceProvider"] = resourceProvider
		}

		if tracingLabelsMethod {
			requestLabels["method"] = strings.ToLower(res.Request.Method)
		}

		if tracingLabelsStatusCode {
			requestLabels["statusCode"] = strconv.FormatInt(int64(res.StatusCode), 10)
		}

		prometheusAzureApiRequest.With(requestLabels).Observe(requestDuration.Seconds())
	}

	if prometheusAzureApiRatelimit != nil {
		collectAzureApiRateLimitMetric := func(r *http.Response, headerName string, scopeLabel, typeLabel string) {
			headerValue := r.Header.Get(headerName)

			if v, err := strconv.ParseInt(headerValue, 10, 64); err == nil {
				// single value
				prometheusAzureApiRatelimit.With(prometheus.Labels{
					"apiEndpoint":    hostname,
					"subscriptionID": subscriptionId,
					"tenantID":       tenantId,
					"scope":          scopeLabel,
					"type":           typeLabel,
				}).Set(float64(v))
			} else if strings.Contains(headerValue, ":") {

				// multi value (comma sparated eg "QueriesPerHour:496,QueriesPerMin:37,QueriesPer10Sec:11")
				for _, val := range strings.Split(headerValue, ",") {
					if parts := strings.SplitN(val, ":", 2); len(parts) == 2 {
						quotaName := parts[0]
						quotaValue := parts[1]
						if v, err := strconv.ParseInt(quotaValue, 10, 64); err == nil {
							prometheusAzureApiRatelimit.With(prometheus.Labels{
								"apiEndpoint":    hostname,
								"subscriptionID": subscriptionId,
								"tenantID":       tenantId,
								"scope":          scopeLabel,
								"type":           fmt.Sprintf("%s.%s", typeLabel, quotaName),
							}).Set(float64(v))
						}
					}
				}

			}
		}

		// special resourcegraph limits
		if strings.HasPrefix(path, "/providers/microsoft.resourcegraph/") {
			collectAzureApiRateLimitMetric(res, "x-ms-user-quota-remaining", "resourcegraph", "quota")
		}

		// costmanagement limits
		collectAzureApiRateLimitMetric(res, "x-ms-ratelimit-microsoft.costmanagement-qpu-consumed", "costmanagement", "qpu-consumed")
		collectAzureApiRateLimitMetric(res, "x-ms-ratelimit-microsoft.costmanagement-qpu-remaining", "costmanagement", "qpu-remaining")
		collectAzureApiRateLimitMetric(res, "x-ms-ratelimit-remaining-microsoft.costmanagement-entity-requests", "costmanagement", "entity-requests")
		collectAzureApiRateLimitMetric(res, "x-ms-ratelimit-remaining-microsoft.costmanagement-tenant-requests", "costmanagement", "tenant-requests")

		// consumption limits
		collectAzureApiRateLimitMetric(res, "x-ms-ratelimit-remaining-microsoft.consumption-tenant-requests", "consumption", "tenant-requests")

		// subscription rate limits
		collectAzureApiRateLimitMetric(res, "x-ms-ratelimit-remaining-subscription-reads", "subscription", "reads")
		collectAzureApiRateLimitMetric(res, "x-ms-ratelimit-remaining-subscription-writes", "subscription", "writes")
		collectAzureApiRateLimitMetric(res, "x-ms-ratelimit-remaining-subscription-resource-requests", "subscription", "resourceRequests")
		collectAzureApiRateLimitMetric(res, "x-ms-ratelimit-remaining-subscription-resource-entities-read", "subscription", "resource-entities-read")

		// tenant rate limits
		collectAzureApiRateLimitMetric(res, "x-ms-ratelimit-remaining-tenant-reads", "tenant", "reads")
		collectAzureApiRateLimitMetric(res, "x-ms-ratelimit-remaining-tenant-writes", "tenant", "writes")
		collectAzureApiRateLimitMetric(res, "x-ms-ratelimit-remaining-tenant-resource-requests", "tenant", "resource-requests")
		collectAzureApiRateLimitMetric(res, "x-ms-ratelimit-remaining-tenant-resource-entities-read", "tenant", "resource-entities-read")
	}

	return res, err
}
