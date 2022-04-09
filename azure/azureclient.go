package azure

import (
	"context"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	cache "github.com/patrickmn/go-cache"

	"github.com/webdevops/go-common/prometheus/azuretracing"
)

type (
	Client struct {
		Environment azure.Environment
		Authorizer  autorest.Authorizer

		cache    *cache.Cache
		cacheTtl time.Duration

		userAgent string
	}
)

func NewClient(environment azure.Environment, authorizer autorest.Authorizer) *Client {
	azureClient := &Client{}
	azureClient.Environment = environment
	azureClient.Authorizer = authorizer

	azureClient.cacheTtl = 30 * time.Minute
	azureClient.cache = cache.New(60*time.Minute, 60*time.Second)

	return azureClient
}

func NewClientFromEnvironment(environmentName string) (*Client, error) {
	// azure authorizer
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	environment, err := azure.EnvironmentFromName(environmentName)
	if err != nil {
		return nil, err
	}

	return NewClient(environment, authorizer), nil
}

func (azureClient *Client) SetUserAgent(useragent string) {
	azureClient.userAgent = useragent
}

func (azureClient *Client) SetCacheTtl(ttl time.Duration) {
	azureClient.cacheTtl = ttl
}

func (azureClient *Client) DecorateAzureAutorest(client *autorest.Client) {
	client.Authorizer = azureClient.Authorizer
	if azureClient.userAgent != "" {
		if err := client.AddToUserAgent(azureClient.userAgent); err != nil {
			panic(err)
		}
	}

	azuretracing.DecorateAzureAutoRestClient(client)
}

func (azureClient *Client) ListCachedSubscriptions(ctx context.Context) (*map[string]subscriptions.Subscription, error) {
	cacheKey := "subscriptions"
	if v, ok := azureClient.cache.Get(cacheKey); ok {
		if cacheData, ok := v.(*map[string]subscriptions.Subscription); ok {
			return cacheData, nil
		}
	}

	list, err := azureClient.ListSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	azureClient.cache.Set(cacheKey, list, azureClient.cacheTtl)

	return list, nil
}

func (azureClient *Client) ListSubscriptions(ctx context.Context) (*map[string]subscriptions.Subscription, error) {
	client := subscriptions.NewClientWithBaseURI(azureClient.Environment.ResourceManagerEndpoint)
	azureClient.DecorateAzureAutorest(&client.Client)

	result, err := client.ListComplete(ctx)
	if err != nil {
		return nil, err
	}

	list := map[string]subscriptions.Subscription{}
	for result.NotDone() {
		row := result.Value()

		subscriptionId := strings.ToLower(to.String(row.SubscriptionID))
		list[subscriptionId] = row

		if result.NextWithContext(ctx) != nil {
			break
		}
	}

	return &list, nil
}

func (azureClient *Client) ListCachedResourceGroups(ctx context.Context, subscription subscriptions.Subscription) (*map[string]resources.Group, error) {
	cacheKey := "resourcegroups:" + to.String(subscription.SubscriptionID)
	if v, ok := azureClient.cache.Get(cacheKey); ok {
		if cacheData, ok := v.(*map[string]resources.Group); ok {
			return cacheData, nil
		}
	}

	list, err := azureClient.ListResourceGroups(ctx, subscription)
	if err != nil {
		return nil, err
	}

	azureClient.cache.Set(cacheKey, list, azureClient.cacheTtl)

	return list, nil
}

func (azureClient *Client) ListResourceGroups(ctx context.Context, subscription subscriptions.Subscription) (*map[string]resources.Group, error) {
	client := resources.NewGroupsClientWithBaseURI(azureClient.Environment.ResourceManagerEndpoint, *subscription.SubscriptionID)
	azureClient.DecorateAzureAutorest(&client.Client)

	result, err := client.ListComplete(ctx, "", nil)
	if err != nil {
		return nil, err
	}

	list := map[string]resources.Group{}
	for result.NotDone() {
		row := result.Value()

		resourceGroupName := strings.ToLower(to.String(row.Name))
		list[resourceGroupName] = row

		if result.NextWithContext(ctx) != nil {
			break
		}
	}

	return &list, nil
}
