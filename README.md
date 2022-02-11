# Library for common prometheus tasks


## azuretracing metrics

Azuretracing metrics collects latency and latency from azure-sdk-for-go and creates metrics and is controllable using
environment variables (eg. setting buckets, disabling metrics or disable autoreset).

| Metric                                   | Description                                                                            |
|------------------------------------------|----------------------------------------------------------------------------------------|
| `azurerm_api_ratelimit`                  | Azure ratelimit metrics (only on /metrics, resets after query due to limited validity) |
| `azurerm_api_request_*`                  | Azure request count and latency as histogram                                           |

| Environment variable                     | Example                          | Description                                              |
|------------------------------------------|----------------------------------|----------------------------------------------------------|
| `METRIC_AZURERM_API_REQUEST_BUCKETS`     | `1, 2.5, 5, 10, 30, 60, 90, 120` | Sets buckets for `azurerm_api_request` histogram metric  |
| `METRIC_AZURERM_API_REQUEST_DISABLE`     | `false`                          | Disables `azurerm_api_request_*` metric                  |
| `METRIC_AZURERM_API_RATELIMIT_DISABLE`   | `false`                          | Disables `azurerm_api_ratelimit` metric                  |
| `METRIC_AZURERM_API_RATELIMIT_AUTORESET` | `false`                          | Disables `azurerm_api_ratelimit` autoreset after fetch   |
