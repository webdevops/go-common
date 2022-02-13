# Library for common prometheus tasks

## AzureTracing metrics

Azuretracing metrics collects latency and latency from azure-sdk-for-go and creates metrics and is controllable using
environment variables (eg. setting buckets, disabling metrics or disable autoreset).

| Metric                                   | Description                                                                            |
|------------------------------------------|----------------------------------------------------------------------------------------|
| `azurerm_api_ratelimit`                  | Azure ratelimit metrics (only on /metrics, resets after query due to limited validity) |
| `azurerm_api_request_*`                  | Azure request count and latency as histogram                                           |

### Settings

| Environment variable                     | Example                          | Description                                                    |
|------------------------------------------|----------------------------------|----------------------------------------------------------------|
| `METRIC_AZURERM_API_REQUEST_BUCKETS`     | `1, 2.5, 5, 10, 30, 60, 90, 120` | Sets buckets for `azurerm_api_request` histogram metric        |
| `METRIC_AZURERM_API_REQUEST_ENABLE`      | `false`                          | Enables/disables `azurerm_api_request_*` metric                |
| `METRIC_AZURERM_API_REQUEST_LABELS`      | `endpoint, method, statusCode`   | Controls labels of `azurerm_api_request_*` metric              |
| `METRIC_AZURERM_API_RATELIMIT_ENABLE`    | `false`                          | Enables/disables `azurerm_api_ratelimit` metric                |
| `METRIC_AZURERM_API_RATELIMIT_ENABLE`    | `false`                          | Enables/disables `azurerm_api_ratelimit` autoreset after fetch |


| `azurerm_api_request` label | Status             | Description                                                                                              |
|-----------------------------|--------------------|----------------------------------------------------------------------------------------------------------|
| `endpoint`                  | enabled by default | hostname of endpoint (max 3 parts)                                                                       |
| `routingRegion`             | enabled by default | detected region for API call, either routing region from Azure Management API or Azure resource location |
| `subscriptionID`            | enabled by default | detected subscriptionID                                                                                  | 
| `tenantID`                  | enabled by default | detected tenantID (extracted from jwt auth token)                                                        | 
| `resourceProvider`          | enabled by default | detected Azure Management API provider                                                                   |
| `method`                    | enabled by default | HTTP method                                                                                              |
| `statusCode`                | enabled by default | HTTP status code                                                                                         |
