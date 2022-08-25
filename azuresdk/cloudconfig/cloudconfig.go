package cloudconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
)

const MicrosoftGraph cloud.ServiceName = "microsoftGraph"

// NewCloudConfig creates a new cloud configuration object based on cloud name (eg. AzurePublicCloud)
func NewCloudConfig(cloudName string) (config cloud.Configuration, err error) {
	switch strings.ToLower(cloudName) {
	// ----------------------------------------------------
	// Azure Public cloud (default)
	case "azurepublic", "azurepubliccloud", "azurecloud":
		config, err = cloud.AzurePublic, nil
		injectServiceConfig(&config, MicrosoftGraph, cloud.ServiceConfiguration{
			Audience: "https://graph.microsoft.com/",
			Endpoint: "https://graph.microsoft.com",
		})

	// ----------------------------------------------------
	// Azure China cloud
	case "azurechina", "azurechinacloud":
		config, err = cloud.AzureChina, nil
		injectServiceConfig(&config, MicrosoftGraph, cloud.ServiceConfiguration{
			Audience: "https://microsoftgraph.chinaclouapi.cn/",
			Endpoint: "https://microsoftgraph.chinaclouapi.cn",
		})

	// ----------------------------------------------------
	// Azure Government cloud
	case "azuregovernment", "azuregovernmentcloud", "azureusgovernmentcloud":
		config, err = cloud.AzureGovernment, nil
		injectServiceConfig(&config, MicrosoftGraph, cloud.ServiceConfiguration{
			Audience: "https://login.microsoftonline.us/",
			Endpoint: "https://login.microsoftonline.us",
		})

	// ----------------------------------------------------
	// Azure Private Cloud (onpremise, custom configuration via json)
	case "azureprivate", "azurepprivatecloud":
		config, err = createAzurePrivateCloudConfig()

	default:
		err = fmt.Errorf(`unable to set Azure Cloud "%v", not valid`, cloudName)
	}

	return
}

// injectServiceConfig injects a serviceconfiguration into cloud config
func injectServiceConfig(config *cloud.Configuration, serviceName cloud.ServiceName, serviceConfig cloud.ServiceConfiguration) {
	if config.Services == nil {
		config.Services = map[cloud.ServiceName]cloud.ServiceConfiguration{}
	}

	config.Services[serviceName] = serviceConfig
}

// createAzurePrivateCloudConfig creates azureprivate (onpremise) cloudconfig from either AZURE_CLOUD_CONFIG (string) or AZURE_CLOUD_CONFIG_FILE (file)
func createAzurePrivateCloudConfig() (cloud.Configuration, error) {
	var cloudConfigJson []byte
	cloudConfig := cloud.Configuration{}

	if val := os.Getenv("AZURE_CLOUD_CONFIG"); len(val) > 0 {
		// cloud config via JSON string
		cloudConfigJson = []byte(val)
	} else if val := os.Getenv("AZURE_CLOUD_CONFIG_FILE"); len(val) > 0 {
		// cloud config via JSON file
		data, err := os.ReadFile(val) // #nosec G304
		if err != nil {
			return cloudConfig, fmt.Errorf(`unable to parse json for AzurePrivateCloud from env var AZURE_CLOUD_CONFIG_FILE, see https://github.com/webdevops/go-common/tree/main/azuresdk: %v`, err.Error())
		}
		cloudConfigJson = data
	}

	if len(cloudConfigJson) == 0 {
		return cloudConfig, fmt.Errorf(`AzurePrivateCloud needs cloudconfig json passed via env var AZURE_CLOUD_CONFIG or AZURE_CLOUD_CONFIG_FILE, see https://github.com/webdevops/go-common/tree/main/azuresdk`)
	}

	if err := json.Unmarshal([]byte(cloudConfigJson), &cloudConfig); err != nil {
		return cloudConfig, fmt.Errorf(`unable to parse json for AzurePrivateCloud from env var AZURE_CLOUD_CONFIG or AZURE_CLOUD_CONFIG_FILE, see https://github.com/webdevops/go-common/tree/main/azuresdk: %v`, err.Error())
	}

	return cloudConfig, nil
}
