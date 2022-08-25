package hamiltonclient

import (
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/patrickmn/go-cache"

	authWrapper "github.com/manicminer/hamilton-autorest/auth"
	log "github.com/sirupsen/logrus"

	"github.com/webdevops/go-common/azuresdk/cloudconfig"
)

type (
	MsGraphClient struct {
		cloud cloud.Configuration

		tenantID string

		logger *log.Logger

		cache              *cache.Cache
		cacheTtl           time.Duration
		cacheAuthorizerTtl time.Duration

		userAgent string
	}
)

// NewMsGraphClient creates new MS Graph client
func NewMsGraphClient(cloudConfig cloud.Configuration, tenantID string, logger *log.Logger) *MsGraphClient {
	client := &MsGraphClient{}
	client.cloud = cloudConfig

	client.tenantID = tenantID

	client.cacheTtl = 1 * time.Hour
	client.cache = cache.New(60*time.Minute, 60*time.Second)

	client.cacheAuthorizerTtl = 15 * time.Minute

	client.logger = logger
	client.userAgent = "go-common/unknown"

	return client
}

// NewMsGraphClientWithCloudName creates new MS Graph client with environment name as string
func NewMsGraphClientWithCloudName(cloudName string, tenantID string, logger *log.Logger) (*MsGraphClient, error) {
	cloudConfig, err := cloudconfig.NewCloudConfig(cloudName)
	if err != nil {
		logger.Panic(err.Error())
	}
	return NewMsGraphClient(cloudConfig, tenantID, logger), nil
}

// Authorizer returns hamilton authorizer
func (c *MsGraphClient) Authorizer() *authWrapper.AuthorizerWrapper {
	cacheKey := "authorizer"
	if v, ok := c.cache.Get(cacheKey); ok {
		if authorizer, ok := v.(*authWrapper.AuthorizerWrapper); ok {
			return authorizer
		}
	}

	authorizer := c.createAuthorizer()

	c.cache.Set(cacheKey, authorizer, c.cacheAuthorizerTtl)

	return authorizer
}

// createAuthorizer returns new msgraph request adapter
func (c *MsGraphClient) createAuthorizer() *authWrapper.AuthorizerWrapper {
	var (
		authorizer autorest.Authorizer
		err        error
	)
	// azure authorizer

	oauth, err := adal.NewOAuthConfig(c.cloud.Services[cloudconfig.MicrosoftGraph].Endpoint, c.tenantID)
	if err != nil {
		c.logger.Panic(err)
	}
	if oauth == nil {
		c.logger.Panicf(`OAuthConfig was nil for tenant %s`, c.tenantID)
	}

	// azure authorizer
	switch strings.ToLower(os.Getenv("AZURE_AUTH")) {
	case "az", "cli", "azcli":
		authorizer, err = auth.NewAuthorizerFromCLIWithResource(c.cloud.Services[cloudconfig.MicrosoftGraph].Endpoint)
		if err != nil {
			c.logger.Panic(err)
		}

	default:
		authorizer, err = auth.NewAuthorizerFromEnvironmentWithResource(c.cloud.Services[cloudconfig.MicrosoftGraph].Endpoint)
		if err != nil {
			c.logger.Panic(err)
		}
	}

	wrapper, err := authWrapper.NewAuthorizerWrapper(authorizer)
	if err != nil {
		c.logger.Panic(err)
	}

	return wrapper.(*authWrapper.AuthorizerWrapper)
}

// SetUserAgent set user agent for all API calls
func (c *MsGraphClient) SetUserAgent(useragent string) {
	c.userAgent = useragent
}

// SetCacheTtl set TTL for service discovery cache
func (c *MsGraphClient) SetCacheTtl(ttl time.Duration) {
	c.cacheTtl = ttl
}

// GetTenantID returns the current set TenantID
func (c *MsGraphClient) GetTenantID() string {
	return c.tenantID
}
