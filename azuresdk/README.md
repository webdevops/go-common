# Go Common library for webdevops projects

## ArmClient

### Environment support

- Azure Public cloud (default)
- Azure China cloud
- Azure Government cloud
- Azure Private cloud (onpremise, needs additional cloud configuration)

#### Azure Private cloud

Azure private cloud needs additional custom cloud configuration which can be passed environment variables:

| Env var                   | Description                          |
|---------------------------|--------------------------------------|
| `AZURE_CLOUD_CONFIG`      | JSON config as string (single line)  |
| `AZURE_CLOUD_CONFIG_FILE` | Path to JSON config as string        |

Example configuration:
```json
{
    "activeDirectoryAuthorityHost": "https://login.microsoftonline.com/",
    "services": {
        "resourceManager": {
            "audience": "https://management.core.windows.net/",
            "endpoint": "https://management.azure.com"
        }
    }
}
```

### Tag handling

Tag can be dynamically added to metrics and processed though filters

format is: `tagname?filter1` or `tagname?filter1&filter2`

| Tag filter | Description                 |
|------------|-----------------------------|
| `toLower`  | Lowercasing Azure tag value |
| `toUpper`  | Uppercasing Azure tag value |

## AzureTracing metrics

Azuretracing metrics collects latency and latency from azure-sdk-for-go and creates metrics and is controllable using
environment variables (eg. setting buckets, disabling metrics or disable autoreset).

| Metric                                   | Description                                                                            |
|------------------------------------------|----------------------------------------------------------------------------------------|
| `azurerm_api_ratelimit`                  | Azure ratelimit metrics (only on /metrics, resets after query due to limited validity) |
| `azurerm_api_request_*`                  | Azure request count and latency as histogram                                           |

### Settings

| Environment variable                     | Example                           | Description                                                    |
|------------------------------------------|-----------------------------------|----------------------------------------------------------------|
| `METRIC_AZURERM_API_REQUEST_BUCKETS`     | `1, 5, 15, 30, 90`                | Sets buckets for `azurerm_api_request` histogram metric        |
| `METRIC_AZURERM_API_REQUEST_ENABLE`      | `false`                           | Enables/disables `azurerm_api_request_*` metric                |
| `METRIC_AZURERM_API_REQUEST_LABELS`      | `apiEndpoint, method, statusCode` | Controls labels of `azurerm_api_request_*` metric              |
| `METRIC_AZURERM_API_RATELIMIT_ENABLE`    | `false`                           | Enables/disables `azurerm_api_ratelimit` metric                |
| `METRIC_AZURERM_API_RATELIMIT_AUTORESET` | `false`                           | Enables/disables `azurerm_api_ratelimit` autoreset after fetch |


| `azurerm_api_request` label | Status              | Description                                                                                              |
|-----------------------------|---------------------|----------------------------------------------------------------------------------------------------------|
| `apiEndpoint`               | enabled by default  | hostname of endpoint (max 3 parts)                                                                       |
| `routingRegion`             | disabled by default | detected region for API call, either routing region from Azure Management API or Azure resource location |
| `subscriptionID`            | enabled by default  | detected subscriptionID                                                                                  |
| `tenantID`                  | enabled by default  | detected tenantID (extracted from jwt auth token)                                                        |
| `resourceProvider`          | enabled by default  | detected Azure Management API provider                                                                   |
| `method`                    | enabled by default  | HTTP method                                                                                              |
| `statusCode`                | enabled by default  | HTTP status code                                                                                         |
