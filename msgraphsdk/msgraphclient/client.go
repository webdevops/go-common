package msgraphclient

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	a "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/patrickmn/go-cache"

	log "github.com/sirupsen/logrus"
)

type (
	MsGraphClient struct {
		cloud cloud.Configuration

		logger *log.Logger

		cache              *cache.Cache
		cacheTtl           time.Duration
		cacheAuthorizerTtl time.Duration

		userAgent string
	}
)

// NewMsGraphClient creates new MS Graph client
func NewMsGraphClient(cloudConfig cloud.Configuration, logger *log.Logger) *MsGraphClient {
	client := &MsGraphClient{}
	client.cloud = cloudConfig

	client.cacheTtl = 1 * time.Hour
	client.cache = cache.New(60*time.Minute, 60*time.Second)

	client.cacheAuthorizerTtl = 15 * time.Minute

	client.logger = logger
	client.userAgent = "go-common/unknown"

	return client
}

// NewMsGraphClientWithCloudName creates new MS Graph client with environment name as string
func NewMsGraphClientWithCloudName(cloudName string, logger *log.Logger) (*MsGraphClient, error) {
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

	return NewMsGraphClient(cloudConfig, logger), nil
}

// ServiceClient returns msgraph service client
func (c *MsGraphClient) ServiceClient() *msgraphsdk.GraphServiceClient {
	cacheKey := "authorizer"
	if v, ok := c.cache.Get(cacheKey); ok {
		if cred, ok := v.(*msgraphsdk.GraphServiceClient); ok {
			return cred
		}
	}

	serviceClient := c.createServiceClient()

	c.cache.Set(cacheKey, serviceClient, c.cacheAuthorizerTtl)

	return serviceClient
}

// createServiceClient returns new msgraph service client
func (c *MsGraphClient) createServiceClient() *msgraphsdk.GraphServiceClient {
	// azure authorizer
	switch strings.ToLower(os.Getenv("AZURE_AUTH")) {
	case "az", "cli", "azcli":
		cred, err := azidentity.NewAzureCLICredential(nil)
		if err != nil {
			c.logger.Panic(err)
		}

		auth, err := a.NewAzureIdentityAuthenticationProvider(cred)
		if err != nil {
			c.logger.Panic(err)
		}

		adapter, err := msgraphsdk.NewGraphRequestAdapter(auth)
		if err != nil {
			c.logger.Panic(err)
		}

		return msgraphsdk.NewGraphServiceClient(adapter)
	default:
		// general azure authentication (env vars, service principal, msi, ...)
		opts := azidentity.EnvironmentCredentialOptions{
			ClientOptions: azcore.ClientOptions{
				Cloud:            c.cloud,
				PerCallPolicies:  nil,
				PerRetryPolicies: nil,
			},
		}
		cred, err := azidentity.NewEnvironmentCredential(&opts)
		if err != nil {
			c.logger.Panic(err)
		}

		auth, err := a.NewAzureIdentityAuthenticationProvider(cred)
		if err != nil {
			c.logger.Panic(err)
		}

		adapter, err := msgraphsdk.NewGraphRequestAdapter(auth)
		if err != nil {
			c.logger.Panic(err)
		}

		return msgraphsdk.NewGraphServiceClient(adapter)
	}
}
