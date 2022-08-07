package azure

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	cache "github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"

	"github.com/webdevops/go-common/utils/to"

	"github.com/webdevops/go-common/prometheus/azuretracing"
)

type (
	Client struct {
		Environment azure.Environment

		logger *log.Logger

		cache    *cache.Cache
		cacheTtl time.Duration

		cacheAuthorizerTtl time.Duration

		subscriptionFilter []string

		userAgent string

		tagInheritance struct {
			subscriptionTags  []string
			resourceGroupTags []string

			lookup struct {
				lock sync.RWMutex

				lastUpdate map[string]time.Time

				subscriptionTags  map[string]map[string]string
				resourceGroupTags map[string]map[string]map[string]string
			}
		}
	}
)

func NewClient(environment azure.Environment, logger *log.Logger) *Client {
	azureClient := &Client{}
	azureClient.Environment = environment

	azureClient.cacheTtl = 30 * time.Minute
	azureClient.cache = cache.New(60*time.Minute, 60*time.Second)

	azureClient.cacheAuthorizerTtl = 15 * time.Minute

	azureClient.logger = logger

	azureClient.tagInheritance.lookup.lastUpdate = map[string]time.Time{}
	azureClient.tagInheritance.lookup.subscriptionTags = map[string]map[string]string{}

	return azureClient
}

func NewClientFromEnvironment(environmentName string, logger *log.Logger) (*Client, error) {
	environment, err := azure.EnvironmentFromName(environmentName)
	if err != nil {
		return nil, err
	}

	return NewClient(environment, logger), nil
}

func (azureClient *Client) SetSubscriptionFilter(subscriptionId ...string) {
	azureClient.subscriptionFilter = subscriptionId
}

func (azureClient *Client) GetAuthorizer() autorest.Authorizer {
	return azureClient.GetAuthorizerWithResource(azureClient.Environment.ResourceManagerEndpoint)
}

func (azureClient *Client) GetAuthorizerWithResource(resource string) autorest.Authorizer {
	cacheKey := "authorizer:" + resource
	if v, ok := azureClient.cache.Get(cacheKey); ok {
		if authorizer, ok := v.(autorest.Authorizer); ok {
			return authorizer
		}
	}

	authorizer, err := azureClient.createAuthorizer(resource)
	if err != nil {
		panic(err)
	}

	azureClient.cache.Set(cacheKey, authorizer, azureClient.cacheAuthorizerTtl)

	return authorizer
}

func (azureClient *Client) createAuthorizer(resource string) (autorest.Authorizer, error) {
	// azure authorizer
	switch strings.ToLower(os.Getenv("AZURE_AUTH")) {
	case "az", "cli", "azcli":
		return auth.NewAuthorizerFromCLIWithResource(resource)
	default:
		return auth.NewAuthorizerFromEnvironmentWithResource(resource)
	}
}

func (azureClient *Client) GetEnvironment() azure.Environment {
	return azureClient.Environment
}

func (azureClient *Client) SetUserAgent(useragent string) {
	azureClient.userAgent = useragent
}

func (azureClient *Client) SetCacheTtl(ttl time.Duration) {
	azureClient.cacheTtl = ttl
}

func (azureClient *Client) DecorateAzureAutorest(client *autorest.Client) {
	azureClient.DecorateAzureAutorestWithAuthorizer(client, azureClient.GetAuthorizer())
}

func (azureClient *Client) DecorateAzureAutorestWithAuthorizer(client *autorest.Client, authorizer autorest.Authorizer) {
	client.Authorizer = authorizer
	if azureClient.userAgent != "" {
		if err := client.AddToUserAgent(azureClient.userAgent); err != nil {
			panic(err)
		}
	}

	azuretracing.DecorateAzureAutoRestClient(client)
}

func (azureClient *Client) ListCachedSubscriptionsWithFilter(ctx context.Context, subscriptionID ...string) (map[string]subscriptions.Subscription, error) {
	availableSubscriptions, err := azureClient.ListCachedSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	// filter subscriptions
	if len(subscriptionID) > 0 {
		tmp := map[string]subscriptions.Subscription{}
		for _, subscription := range availableSubscriptions {
			for _, filterSubscriptionID := range subscriptionID {
				if strings.EqualFold(filterSubscriptionID, *subscription.SubscriptionID) {
					tmp[*subscription.SubscriptionID] = subscription
				}
			}
		}

		availableSubscriptions = tmp
	}

	return availableSubscriptions, nil
}

func (azureClient *Client) ListCachedSubscriptions(ctx context.Context) (map[string]subscriptions.Subscription, error) {
	cacheKey := "subscriptions"
	if v, ok := azureClient.cache.Get(cacheKey); ok {
		if cacheData, ok := v.(map[string]subscriptions.Subscription); ok {
			return cacheData, nil
		}
	}

	azureClient.logger.Debug("updating cached Azure Subscription list")
	list, err := azureClient.ListSubscriptions(ctx)
	if err != nil {
		return nil, err
	}
	azureClient.logger.Debugf("found %v Azure Subscriptions", len(list))

	azureClient.cache.Set(cacheKey, list, azureClient.cacheTtl)

	return list, nil
}

func (azureClient *Client) ListSubscriptions(ctx context.Context) (map[string]subscriptions.Subscription, error) {
	list := map[string]subscriptions.Subscription{}
	client := subscriptions.NewClientWithBaseURI(azureClient.Environment.ResourceManagerEndpoint)
	azureClient.DecorateAzureAutorest(&client.Client)

	result, err := client.ListComplete(ctx)
	if err != nil {
		return list, err
	}

	for result.NotDone() {
		subscription := result.Value()

		if len(azureClient.subscriptionFilter) > 0 {
			// use subscription filter
			for _, subscriptionId := range azureClient.subscriptionFilter {
				if strings.EqualFold(*subscription.SubscriptionID, subscriptionId) {
					list[*subscription.SubscriptionID] = subscription
					break
				}
			}
		} else {
			list[*subscription.SubscriptionID] = subscription
		}

		if result.NextWithContext(ctx) != nil {
			break
		}
	}

	return list, nil
}

func (azureClient *Client) ListCachedResourceGroups(ctx context.Context, subscription subscriptions.Subscription) (map[string]resources.Group, error) {
	cacheKey := "resourcegroups:" + to.String(subscription.SubscriptionID)
	if v, ok := azureClient.cache.Get(cacheKey); ok {
		if cacheData, ok := v.(map[string]resources.Group); ok {
			return cacheData, nil
		}
	}

	azureClient.logger.WithField("subscriptionID", *subscription.SubscriptionID).Debug("updating cached Azure ResourceGroup list")
	list, err := azureClient.ListResourceGroups(ctx, subscription)
	if err != nil {
		return list, err
	}
	azureClient.logger.WithField("subscriptionID", *subscription.SubscriptionID).Debugf("found %v Azure ResourceGroups", len(list))

	azureClient.cache.Set(cacheKey, list, azureClient.cacheTtl)

	return list, nil
}

func (azureClient *Client) ListResourceGroups(ctx context.Context, subscription subscriptions.Subscription) (map[string]resources.Group, error) {
	list := map[string]resources.Group{}

	client := resources.NewGroupsClientWithBaseURI(azureClient.Environment.ResourceManagerEndpoint, *subscription.SubscriptionID)
	azureClient.DecorateAzureAutorest(&client.Client)

	result, err := client.ListComplete(ctx, "", nil)
	if err != nil {
		return list, err
	}

	for result.NotDone() {
		row := result.Value()

		resourceGroupName := strings.ToLower(to.String(row.Name))
		list[resourceGroupName] = row

		if result.NextWithContext(ctx) != nil {
			break
		}
	}

	return list, nil
}

func (azureClient *Client) InheritTags(resourceId string, resourceTags interface{}) map[string]string {
	azureClient.tagInheritance.lookup.lock.RLock()
	defer azureClient.tagInheritance.lookup.lock.RUnlock()

	tags := translateTagsToStringMap(resourceTags)

	resourceInfo, err := ParseResourceId(resourceId)
	if err != nil {
		return tags
	}

	cacheKey := "taginherit:lookup"
	if _, ok := azureClient.cache.Get(cacheKey); !ok {
		azureClient.updateTagInhertLookupMaps()
		azureClient.cache.Set(cacheKey, nil, azureClient.cacheTtl)
	}

	if resourceInfo.Subscription != "" {
		// try to inherit subscription tags
		subscriptionId := strings.ToLower(resourceInfo.Subscription)
		if subscriptionTags, exists := azureClient.tagInheritance.lookup.subscriptionTags[subscriptionId]; exists {
			for _, tagName := range azureClient.tagInheritance.subscriptionTags {
				if _, exists := tags[tagName]; !exists {
					if tagValue, exists := subscriptionTags[tagName]; exists {
						tags[tagName] = tagValue
					}
				}
			}
		}

		// try to inherit resourcegroup tags
		if resourceInfo.ResourceGroup != "" {
			resourceGroupName := strings.ToLower(resourceInfo.ResourceGroup)
			if resourceGroupTags, exists := azureClient.tagInheritance.lookup.resourceGroupTags[subscriptionId][resourceGroupName]; exists {
				for _, tagName := range azureClient.tagInheritance.subscriptionTags {
					if _, exists := tags[tagName]; !exists {
						if tagValue, exists := resourceGroupTags[tagName]; exists {
							tags[tagName] = tagValue
						}
					}
				}
			}
		}
	}
	return tags
}

func (azureClient *Client) updateTagInhertLookupMaps() {
	if !azureClient.tagInheritance.lookup.lock.TryLock() {
		// update already running
		return
	}
	defer azureClient.tagInheritance.lookup.lock.Unlock()

	azureClient.logger.Debug("updating Subscription and ResourceGroup tag cache")

	ctx := context.Background()

	subscriptionList, err := azureClient.ListCachedSubscriptions(ctx)
	if err != nil {
		azureClient.logger.Panic(err)
	}

	azureClient.tagInheritance.lookup.subscriptionTags = map[string]map[string]string{}
	azureClient.tagInheritance.lookup.resourceGroupTags = map[string]map[string]map[string]string{}
	for _, subscription := range subscriptionList {
		subscriptionId := to.String(subscription.SubscriptionID)
		azureClient.tagInheritance.lookup.subscriptionTags[subscriptionId] = translateTagsToStringMap(subscription.Tags)

		resourceGroupList, err := azureClient.ListCachedResourceGroups(ctx, subscription)
		if err != nil {
			azureClient.logger.Panic(err)
		}

		for _, resourceGroup := range resourceGroupList {
			resourceGroupName := strings.ToLower(to.String(resourceGroup.Name))
			azureClient.tagInheritance.lookup.resourceGroupTags[subscriptionId][resourceGroupName] = translateTagsToStringMap(resourceGroup.Tags)
		}
	}
}
