package msgraphclient

import (
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

	"github.com/webdevops/go-common/azuresdk/cloudconfig"
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
	cloudConfig, err := cloudconfig.NewCloudConfig(cloudName)
	if err != nil {
		logger.Panic(err.Error())
	}
	return NewMsGraphClient(cloudConfig, logger), nil
}

// ServiceClient returns msgraph service client
func (c *MsGraphClient) ServiceClient() *msgraphsdk.GraphServiceClient {
	return msgraphsdk.NewGraphServiceClient(c.RequestAdapter())
}

// RequestAdapter returns msgraph request adapter
func (c *MsGraphClient) RequestAdapter() *msgraphsdk.GraphRequestAdapter {
	cacheKey := "adapter"
	if v, ok := c.cache.Get(cacheKey); ok {
		if adapter, ok := v.(*msgraphsdk.GraphRequestAdapter); ok {
			return adapter
		}
	}

	adapter := c.createRequestAdapter()

	c.cache.Set(cacheKey, adapter, c.cacheAuthorizerTtl)

	return adapter
}

// createRequestAdapter returns new msgraph request adapter
func (c *MsGraphClient) createRequestAdapter() *msgraphsdk.GraphRequestAdapter {
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

		return adapter
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

		return adapter
	}
}

// SetUserAgent set user agent for all API calls
func (c *MsGraphClient) SetUserAgent(useragent string) {
	c.userAgent = useragent
}

// SetCacheTtl set TTL for service discovery cache
func (c *MsGraphClient) SetCacheTtl(ttl time.Duration) {
	c.cacheTtl = ttl
}
