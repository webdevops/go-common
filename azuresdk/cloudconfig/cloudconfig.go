package cloudconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
)

// NewCloudConfig creates a new cloud configuration object based on cloud name (eg. AzurePublicCloud)
func NewCloudConfig(cloudName string) (cloud.Configuration, error) {
	var cloudConfig cloud.Configuration

	switch strings.ToLower(cloudName) {
	// ----------------------------------------------------
	// Azure Public cloud (default)
	case "azurepublic", "azurepubliccloud":
		cloudConfig = cloud.AzurePublic

	// ----------------------------------------------------
	// Azure China cloud
	case "azurechina", "azurechinacloud":
		cloudConfig = cloud.AzurePublic

	// ----------------------------------------------------
	// Azure Government cloud
	case "azuregovernment", "azuregovernmentcloud", "azureusgovernmentcloud":
		cloudConfig = cloud.AzureGovernment

	// ----------------------------------------------------
	// Azure Private Cloud (onpremise, custom configuration via json)
	case "azureprivate", "azurepprivatecloud":
		cloudConfig = cloud.Configuration{}
		cloudConfigJson := os.Getenv("AZURE_CLOUD_CONFIG")
		if len(cloudConfigJson) == 0 {
			return cloudConfig, fmt.Errorf(`AzurePrivateCloud needs cloudconfig json passed via env var AZURE_CLOUD_CONFIG, see https://github.com/webdevops/go-common/tree/main/azuresdk`)
		}

		if err := json.Unmarshal([]byte(cloudConfigJson), &cloudConfig); err != nil {
			return cloudConfig, fmt.Errorf(`unable to parse json for AzurePrivateCloud from env var AZURE_CLOUD_CONFIG, see https://github.com/webdevops/go-common/tree/main/azuresdk: %v`, err.Error())
		}
	default:
		return cloudConfig, fmt.Errorf(`unable to set Azure Cloud "%v", not valid`, cloudName)
	}

	return cloudConfig, nil
}
