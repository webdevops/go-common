package msgraphclient

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/patrickmn/go-cache"

	"github.com/webdevops/go-common/azuresdk/azidentity"
	"github.com/webdevops/go-common/azuresdk/cloudconfig"
	"github.com/webdevops/go-common/azuresdk/prometheus/tracing"
)

type (
	MsGraphClient struct {
		cloud    cloudconfig.CloudEnvironment
		tenantID string

		logger *slog.Logger

		cache    *cache.Cache
		cacheTtl time.Duration

		cred          *azcore.TokenCredential
		serviceClient *msgraphsdk.GraphServiceClient
		adapter       *msgraphsdk.GraphRequestAdapter

		userAgent string
	}
)

// NewMsGraphClientFromEnvironment creates new MS Graph client from environment settings
func NewMsGraphClientFromEnvironment(logger *slog.Logger) (*MsGraphClient, error) {
	var azureEnvironment, azureTenant string

	if azureEnvironment = os.Getenv("AZURE_ENVIRONMENT"); azureEnvironment == "" {
		logger.Info(`env var AZURE_ENVIRONMENT is not set, assuming "AzurePublicCloud"`)
		azureEnvironment = string(cloudconfig.AzurePublicCloud)

		if err := os.Setenv("AZURE_ENVIRONMENT", azureEnvironment); err != nil {
			return nil, fmt.Errorf(`unable to set AZURE_ENVIRONMENT`)
		}
	}

	if azureTenant = os.Getenv("AZURE_TENANT_ID"); azureTenant == "" {
		return nil, fmt.Errorf(`env var AZURE_TENANT_ID is not set`)
	}

	return NewMsGraphClientWithCloudName(azureEnvironment, azureTenant, logger)
}

// NewMsGraphClient creates new MS Graph client
func NewMsGraphClient(cloudConfig cloudconfig.CloudEnvironment, tenantID string, logger *slog.Logger) *MsGraphClient {
	client := &MsGraphClient{}
	client.cloud = cloudConfig
	client.tenantID = tenantID

	client.cacheTtl = 1 * time.Hour
	client.cache = cache.New(60*time.Minute, 60*time.Second)

	client.logger = logger
	client.userAgent = "go-common/unknown"

	return client
}

// NewMsGraphClientWithCloudName creates new MS Graph client with environment name as string
func NewMsGraphClientWithCloudName(cloudName string, tenantID string, logger *slog.Logger) (*MsGraphClient, error) {
	cloudConfig, err := cloudconfig.NewCloudConfig(cloudName)
	if err != nil {
		return nil, err
	}
	return NewMsGraphClient(cloudConfig, tenantID, logger), nil
}

// ServiceClient returns msgraph service client
func (c *MsGraphClient) ServiceClient() *msgraphsdk.GraphServiceClient {
	if c.serviceClient == nil {
		scopes := []string{}
		// set endpoint from cloudconfig
		if c.cloud.Services != nil {
			if serviceConfig, exists := c.cloud.Services[cloudconfig.ServiceNameMicrosoftGraph]; exists {
				scopes = []string{serviceConfig.Audience + "/.default"}
			}
		}

		client, err := msgraphsdk.NewGraphServiceClientWithCredentialsAndHosts(c.getCred(), scopes, nil)
		if err != nil {
			panic(err)
		}

		// set endpoint from cloudconfig
		if c.cloud.Services != nil {
			if serviceConfig, exists := c.cloud.Services[cloudconfig.ServiceNameMicrosoftGraph]; exists {
				client.GetAdapter().SetBaseUrl(serviceConfig.Endpoint + "/v1.0")
			}
		}

		c.serviceClient = client
	}
	return c.serviceClient

}

// RequestAdapter returns msgraph request adapter
func (c *MsGraphClient) RequestAdapter() *msgraphsdk.GraphRequestAdapter {
	if c.adapter == nil {
		auth, err := a.NewAzureIdentityAuthenticationProvider(c.getCred())
		if err != nil {
			panic(err)
		}

		adapter, err := msgraphsdk.NewGraphRequestAdapter(auth)
		if err != nil {
			panic(err)
		}

		// set endpoint from cloudconfig
		if c.cloud.Services != nil {
			if serviceConfig, exists := c.cloud.Services[cloudconfig.ServiceNameMicrosoftGraph]; exists {
				adapter.SetBaseUrl(serviceConfig.Endpoint + "/v1.0")
			}
		}

		c.adapter = adapter
	}

	return c.adapter
}

// RequestAdapter returns msgraph request adapter
func (c *MsGraphClient) getCred() azcore.TokenCredential {
	if c.cred == nil {
		cred, err := azidentity.NewAzDefaultCredential(c.NewAzCoreClientOptions())
		if err != nil {
			panic(err)
		}
		c.cred = &cred
	}

	return *c.cred
}

// UseAzCliAuth use (force) az cli authentication
func (c *MsGraphClient) UseAzCliAuth() {
	cred, err := azidentity.NewAzCliCredential()
	if err != nil {
		panic(err)
	}
	c.cred = &cred
	c.adapter = nil
	c.serviceClient = nil
}

// NewAzCoreClientOptions returns new client options for all arm clients
func (c *MsGraphClient) NewAzCoreClientOptions() *azcore.ClientOptions {
	clientOptions := azcore.ClientOptions{
		Cloud:            c.cloud.Configuration,
		PerCallPolicies:  []policy.Policy{},
		PerRetryPolicies: c.perRetryPolicies(),
		Telemetry:        c.telemetryOptions(),
	}
	return &clientOptions
}

// perRetryPolicies generates all default retry policies
func (c *MsGraphClient) perRetryPolicies() (policies []policy.Policy) {
	// azure prometheus tracing
	if tracing.TracingIsEnabled() {
		policies = append(
			policies,
			tracing.NewTracingPolicy(),
		)
	}

	return
}

// telemetryOptions generates telemetry options
func (c *MsGraphClient) telemetryOptions() policy.TelemetryOptions {
	return policy.TelemetryOptions{
		ApplicationID: strings.TrimSpace(c.userAgent),
		Disabled:      false,
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

// GetTenantID returns the current set TenantID
func (c *MsGraphClient) GetTenantID() string {
	return c.tenantID
}
