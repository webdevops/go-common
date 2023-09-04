# Azure common library webdevops projects

## ArmClient

### Env vars

| Variable name                                      | Default               | Description                                                                                                                                                                                                                           |
|----------------------------------------------------|-----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `AZURE_SERVICEDISCOVERY_CACHE_TTL`                 | `60m` (time.Duration) | ServiceDiscovery cache (eg. subscription, resourceGroup list,...)                                                                                                                                                                     |
| `AZURE_SERVICEDISCOVERY_SUBSCRIPTION_TAG_SELECTOR` |                       | Tag selector `tagName=tagValue,tagName2=tagValue2` to filter subscriptions for ServiceDiscovery (uses [kubernetes label selector library](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors)) |

### Authentication

Hint: please also check [microsoft azure-sdk documentation](https://docs.microsoft.com/en-us/azure/developer/go/azure-sdk-authentication) for advanced usage.

#### Service principal with a secret

| Variable name          | Value                                        |
|------------------------|----------------------------------------------|
| `AZURE_CLIENT_ID`      | Application ID of an Azure service principal |
| `AZURE_TENANT_ID`      | ID of the application's Azure AD tenant      |
| `AZURE_CLIENT_SECRET`  | Password of the Azure service principal      |

#### Service principal with certificate

| Variable name                   | Value                                                                           |
|---------------------------------|---------------------------------------------------------------------------------|
| `AZURE_CLIENT_ID`               | ID of an Azure AD application                                                   |
| `AZURE_TENANT_ID`               | ID of the application's Azure AD tenant                                         |
| `AZURE_CLIENT_CERTIFICATE_PATH` | Path to a certificate file including private key (without password protection)  |

#### AzureCLI authentication

To force authentication via AzureCLI set `AZURE_AUTH=az` and the token is fetched from Azure CLI.
For this method the `az` binary must be executable in `$PATH` (inside the container/environment).

#### WorkloadIdentity/Federation authentication

To force authentication via WorkloadIdentity/Federation set `AZURE_AUTH=federation`.
Following environment variables needs to be set (automatically set via workloadidentity in AKS clusters):

| Variable name                  | Value                                                                              |
|--------------------------------|------------------------------------------------------------------------------------|
| `AZURE_AUTHORITY_HOST`         | The Azure Active Directory (AAD) endpoint.                                         |
| `AZURE_CLIENT_ID`              | The client ID of the AAD application or user-assigned managed identity.            |
| `AZURE_TENANT_ID`              | The tenant ID of the registered AAD application or user-assigned managed identity. |
| `AZURE_FEDERATED_TOKEN_FILE`   | The path of the projected service account token file.                              |

### Azure Cloud/Environment support

| `AZURE_ENVIRONMENT`    | Description                                                                                  |
|------------------------|----------------------------------------------------------------------------------------------|
| `AzurePublicCloud`     | Default Azure cloud, using https://portal.azure.com                                          |
| `AzureChinaCloud`      | Azure cloud in China, using https://porta.azure.cn                                           |
| `AzureGovernmentCloud` | US Government Azure cloud                                                                    |
| `AzurePrivateCloud`    | Private on-premise installation of Azure Cloud, needs additional configuration for endpoints |

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
        },
        "microsoftGraph": {
            "audience": "https://graph.microsoft.com",
            "endpoint": "https://graph.microsoft.com"
        }
    }
}
```

## Tag manager

Tag can be dynamically added to metrics and processed though filters

format is: `tagname?option1` or `tagname?option1&option2`

| Tag option | Description                                                                                                     |
|------------|-----------------------------------------------------------------------------------------------------------------|
| `name`     | Name of target label                                                                                            |
| `toLower`  | Lowercasing Azure tag value                                                                                     |
| `toUpper`  | Uppercasing Azure tag value                                                                                     |
| `inherit`  | If tag is not set, try to fetch from parent structure (Resource -> ResourceGroup -> Subscription)               |
| `source`   | Defines where the tag value should be fetched from, allowed values: `resource`, `resourceGroup`, `subscription` |

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
