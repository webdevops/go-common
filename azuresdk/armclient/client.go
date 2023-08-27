package armclient

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	cache "github.com/patrickmn/go-cache"
	zap "go.uber.org/zap"

	commonAzidentity "github.com/webdevops/go-common/azuresdk/azidentity"
	"github.com/webdevops/go-common/azuresdk/cloudconfig"
	"github.com/webdevops/go-common/azuresdk/prometheus/tracing"
	"github.com/webdevops/go-common/utils/to"
)

const (
	EnvVarServiceDiscoveryTtl = "AZURE_SERVICEDISCOVERY_CACHE_TTL"
)

type (
	ArmClient struct {
		TagManager *ArmClientTagManager

		cloud cloudconfig.CloudEnvironment

		logger *zap.SugaredLogger

		cache    *cache.Cache
		cacheTtl time.Duration

		subscriptionList []string

		cred *azcore.TokenCredential

		userAgent string
	}
)

// NewArmClientFromEnvironment creates new Azure SDK ARM client from environment settings
func NewArmClientFromEnvironment(logger *zap.SugaredLogger) (*ArmClient, error) {
	var azureEnvironment string

	if azureEnvironment = os.Getenv("AZURE_ENVIRONMENT"); azureEnvironment == "" {
		logger.Panic(`env var AZURE_ENVIRONMENT is not set`)
	}

	return NewArmClientWithCloudName(azureEnvironment, logger)
}

// NewArmClient creates new Azure SDK ARM client
func NewArmClient(cloudConfig cloudconfig.CloudEnvironment, logger *zap.SugaredLogger) *ArmClient {
	client := &ArmClient{}
	client.cloud = cloudConfig

	client.logger = logger
	client.userAgent = "go-common/unknown"

	client.TagManager = &ArmClientTagManager{
		client: client,
		logger: logger.With(zap.String("component", "armClientTagManager")),
	}

	cacheTtl := 60 * time.Minute
	if val := os.Getenv(EnvVarServiceDiscoveryTtl); val != "" {
		if ttl, err := time.ParseDuration(val); err == nil {
			cacheTtl = ttl
		} else {
			logger.Fatalf(`%s is not a valid value, got "%v", expected duration`, EnvVarServiceDiscoveryTtl, val)
		}
	}
	client.SetCacheTtl(cacheTtl)

	return client
}

// NewArmClientWithCloudName creates new Azure SDK ARM client with environment name as string
func NewArmClientWithCloudName(cloudName string, logger *zap.SugaredLogger) (*ArmClient, error) {
	cloudConfig, err := cloudconfig.NewCloudConfig(cloudName)
	if err != nil {
		logger.Panic(err.Error())
	}

	return NewArmClient(cloudConfig, logger), nil
}

// Connect triggers and logs connect message
func (azureClient *ArmClient) Connect() error {
	ctx := context.Background()

	azureClient.logger.Infof(
		`connecting to Azure Environment "%v" (AzureAD:%s ResourceManager:%s)`,
		azureClient.cloud.Name,
		azureClient.cloud.ActiveDirectoryAuthorityHost,
		azureClient.cloud.Services[cloud.ResourceManager].Endpoint,
	)

	// try to get token
	scope := strings.TrimSuffix(azureClient.cloud.Services[cloud.ResourceManager].Endpoint, "/.default") + "/.default"
	accessToken, err := azureClient.GetCred().GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{scope}})
	if err != nil {
		return err
	}

	if tokenInfo := commonAzidentity.ParseAccessToken(accessToken); tokenInfo != nil {
		azureClient.logger.With(zap.Any("client", tokenInfo.ToMap())).Infof(`using Azure client: %v`, tokenInfo.ToString())
	} else {
		azureClient.logger.Warn(`unable to get Azure client information, cannot parse accesstoken`)
	}

	subscriptionList, err := azureClient.ListSubscriptions(ctx)
	if err != nil {
		return err
	}

	azureClient.logger.Infof(`found %v Azure Subscriptions`, len(subscriptionList))
	for subscriptionId, subscription := range subscriptionList {
		azureClient.logger.Debugf(`found Azure Subscription "%v" (%v)`, subscriptionId, to.String(subscription.DisplayName))
	}

	return nil
}

// GetCred returns Azure ARM credential
func (azureClient *ArmClient) GetCred() azcore.TokenCredential {
	if azureClient.cred == nil {
		cred, err := commonAzidentity.NewAzDefaultCredential(azureClient.NewAzCoreClientOptions())
		if err != nil {
			panic(err)
		}
		azureClient.cred = &cred
	}

	return *azureClient.cred
}

// GetCloudName returns selected Azure Environment name (eg AzurePublicCloud)
func (azureClient *ArmClient) GetCloudName() cloudconfig.CloudName {
	return azureClient.cloud.Name
}

// GetCloudConfig returns selected Azure cloud/environment configuration
func (azureClient *ArmClient) GetCloudConfig() cloud.Configuration {
	return azureClient.cloud.Configuration
}

// NewAzCoreClientOptions returns new client options for all arm clients
func (azureClient *ArmClient) NewAzCoreClientOptions() *azcore.ClientOptions {
	clientOptions := azcore.ClientOptions{
		Cloud:            azureClient.cloud.Configuration,
		PerCallPolicies:  []policy.Policy{},
		PerRetryPolicies: nil,
	}

	// azure prometheus tracing
	if tracing.TracingIsEnabled() {
		clientOptions.PerRetryPolicies = append(
			clientOptions.PerRetryPolicies,
			tracing.NewTracingPolicy(),
		)
	}

	return &clientOptions
}

// NewArmClientOptions returns new client options for all arm clients
func (azureClient *ArmClient) NewArmClientOptions() *arm.ClientOptions {
	clientOptions := arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Cloud: azureClient.cloud.Configuration,
		},
	}

	// azure prometheus tracing
	if tracing.TracingIsEnabled() {
		clientOptions.PerRetryPolicies = append(
			clientOptions.PerRetryPolicies,
			tracing.NewTracingPolicy(),
		)
	}

	return &clientOptions
}

// UseAzCliAuth use (force) az cli authentication
func (azureClient *ArmClient) UseAzCliAuth() {
	cred, err := commonAzidentity.NewAzCliCredential()
	if err != nil {
		panic(err)
	}
	azureClient.cred = &cred
}

// SetUserAgent set user agent for all API calls
func (azureClient *ArmClient) SetUserAgent(useragent string) {
	azureClient.userAgent = useragent
}

// SetCacheTtl set TTL for service discovery cache
func (azureClient *ArmClient) SetCacheTtl(ttl time.Duration) {
	azureClient.cacheTtl = ttl
	azureClient.cache = cache.New(ttl, 60*time.Second)
}

// SetSubscriptionFilter set subscription filter, other subscriptions will be ignored
//
// Deprecated: use SetSubscriptionID instead
func (azureClient *ArmClient) SetSubscriptionFilter(subscriptionId ...string) {
	azureClient.SetSubscriptionID(subscriptionId...)
}

// SetSubscriptionID set subscription filter, other subscriptions will be ignored
func (azureClient *ArmClient) SetSubscriptionID(subscriptionId ...string) {
	azureClient.subscriptionList = subscriptionId
}

func (azureClient *ArmClient) cacheData(identifier string, callback func() (interface{}, error)) (interface{}, error) {
	if v, ok := azureClient.cache.Get(identifier); ok {
		return v, nil
	}

	result, err := callback()
	if err == nil {
		azureClient.cache.SetDefault(identifier, result)
	}

	return result, err
}
