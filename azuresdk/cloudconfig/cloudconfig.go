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
	switch strings.ToLower(cloudName) {
	// ----------------------------------------------------
	// Azure Public cloud (default)
	case "azurepublic", "azurepubliccloud":
		return cloud.AzurePublic, nil

	// ----------------------------------------------------
	// Azure China cloud
	case "azurechina", "azurechinacloud":
		return cloud.AzureChina, nil

	// ----------------------------------------------------
	// Azure Government cloud
	case "azuregovernment", "azuregovernmentcloud", "azureusgovernmentcloud":
		return cloud.AzureGovernment, nil

	// ----------------------------------------------------
	// Azure Private Cloud (onpremise, custom configuration via json)
	case "azureprivate", "azurepprivatecloud":
		return createAzurePrivateCloudConfig()
	}

	return cloud.Configuration{}, fmt.Errorf(`unable to set Azure Cloud "%v", not valid`, cloudName)
}

func createAzurePrivateCloudConfig() (cloud.Configuration, error) {
	var cloudConfigJson []byte
	cloudConfig := cloud.Configuration{}

	if val := os.Getenv("AZURE_CLOUD_CONFIG"); len(val) > 0 {
		// cloud config via JSON string
		cloudConfigJson = []byte(val)
	} else if val := os.Getenv("AZURE_CLOUD_CONFIG_FILE"); len(val) > 0 {
		// cloud config via JSON file
		data, err := os.ReadFile(val)
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
