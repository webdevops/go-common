package armclient

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	cache "github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

type (
	ArmClient struct {
		cloud cloud.Configuration

		logger *log.Logger

		cache    *cache.Cache
		cacheTtl time.Duration

		subscriptionFilter []string

		cacheAuthorizerTtl time.Duration

		userAgent string
	}
)

// NewArmClient creates new Azure SDK ARM client
func NewArmClient(cloudConfig cloud.Configuration, logger *log.Logger) *ArmClient {
	client := &ArmClient{}
	client.cloud = cloudConfig

	client.cacheTtl = 30 * time.Minute
	client.cache = cache.New(60*time.Minute, 60*time.Second)

	client.cacheAuthorizerTtl = 15 * time.Minute

	client.logger = logger
	client.userAgent = "go-common/unknown"

	return client
}

// NewArmClientWithCloudName creates new Azure SDK ARM client with environment name as string
func NewArmClientWithCloudName(cloudName string, logger *log.Logger) (*ArmClient, error) {
	var cloudConfig cloud.Configuration

	switch strings.ToLower(cloudName) {
	case "azurepublic", "azurepubliccloud":
		cloudConfig = cloud.AzurePublic
	case "azurechina", "azurechinacloud":
		cloudConfig = cloud.AzurePublic
	case "azuregovernment", "azuregovernmentcloud", "azureusgovernmentcloud":
		cloudConfig = cloud.AzureGovernment
	default:
		return nil, fmt.Errorf(`unable to set Azure Cloud "%v", not valid`, cloudName)
	}

	return NewArmClient(cloudConfig, logger), nil
}

// GetCred returns Azure ARM credential
func (azureClient *ArmClient) GetCred() azcore.TokenCredential {
	cacheKey := "authorizer"
	if v, ok := azureClient.cache.Get(cacheKey); ok {
		if cred, ok := v.(azcore.TokenCredential); ok {
			return cred
		}
	}

	cred, err := azureClient.createAuthorizer()
	if err != nil {
		panic(err)
	}

	azureClient.cache.Set(cacheKey, cred, azureClient.cacheAuthorizerTtl)

	return cred
}

// createAuthorizer creates new azure credential authorizer based on azure environment
func (azureClient *ArmClient) createAuthorizer() (azcore.TokenCredential, error) {
	// azure authorizer
	switch strings.ToLower(os.Getenv("AZURE_AUTH")) {
	case "az", "cli", "azcli":
		// azurecli authentication
		opts := azidentity.AzureCLICredentialOptions{}
		return azidentity.NewAzureCLICredential(&opts)
	default:
		// general azure authentication (env vars, service principal, msi, ...)
		opts := azidentity.DefaultAzureCredentialOptions{
			ClientOptions: azcore.ClientOptions{
				Cloud:            azureClient.cloud,
				PerCallPolicies:  nil,
				PerRetryPolicies: nil,
			},
		}
		return azidentity.NewDefaultAzureCredential(&opts)
	}
}

// GetCloud returns selected Azure cloud/environment configuration
func (azureClient *ArmClient) GetCloud() cloud.Configuration {
	return azureClient.cloud
}

// SetUserAgent set user agent for all API calls
func (azureClient *ArmClient) SetUserAgent(useragent string) {
	azureClient.userAgent = useragent
}

// SetCacheTtl set TTL for service discovery cache
func (azureClient *ArmClient) SetCacheTtl(ttl time.Duration) {
	azureClient.cacheTtl = ttl
}

// SetSubscriptionFilter set subscription filter, other subscriptions will be ignored
func (azureClient *ArmClient) SetSubscriptionFilter(subscriptionId ...string) {
	azureClient.subscriptionFilter = subscriptionId
}
