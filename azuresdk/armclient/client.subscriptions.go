package armclient

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/webdevops/go-common/utils/to"
)

const (
	CacheIdentifierSubscriptions = "subscriptions"
)

// ListCachedSubscriptionsWithFilter return list of subscription with filter by subscription ids
func (azureClient *ArmClient) ListCachedSubscriptionsWithFilter(ctx context.Context, subscriptionFilter ...string) (map[string]*armsubscriptions.Subscription, error) {
	availableSubscriptions, err := azureClient.ListCachedSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	// filter subscriptions
	if len(subscriptionFilter) > 0 {
		tmp := map[string]*armsubscriptions.Subscription{}
		for _, subscription := range availableSubscriptions {
			for _, subscriptionID := range subscriptionFilter {
				if strings.EqualFold(subscriptionID, *subscription.SubscriptionID) {
					tmp[*subscription.SubscriptionID] = subscription
				}
			}
		}

		availableSubscriptions = tmp
	}

	return availableSubscriptions, nil
}

// ListCachedSubscriptions return cached list of Azure Subscriptions as map (key is subscription id)
func (azureClient *ArmClient) ListCachedSubscriptions(ctx context.Context) (map[string]*armsubscriptions.Subscription, error) {
	result, err := azureClient.cacheData(CacheIdentifierSubscriptions, func() (interface{}, error) {
		azureClient.logger.Debug("updating cached Azure Subscription list")
		list, err := azureClient.ListSubscriptions(ctx)
		if err != nil {
			return nil, err
		}
		azureClient.logger.Debugf("found %v Azure Subscriptions", len(list))
		return list, nil
	})
	if err != nil {
		return nil, err
	}

	return result.(map[string]*armsubscriptions.Subscription), nil
}

// ListSubscriptions return list of Azure Subscriptions as map (key is subscription id)
func (azureClient *ArmClient) ListSubscriptions(ctx context.Context) (map[string]*armsubscriptions.Subscription, error) {
	var (
		subscriptionTagSelector *labels.Selector
	)
	list := map[string]*armsubscriptions.Subscription{}

	// parse tag selector (using kubernetes label selector)
	if val := os.Getenv(EnvVarServiceDiscoverySubscriptionTagSelector); val != "" {
		selector, err := labels.Parse(val)
		if err != nil {
			panic(err)
		}
		subscriptionTagSelector = &selector
	}

	client, err := armsubscriptions.NewClient(azureClient.GetCred(), azureClient.NewArmClientOptions())
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		result, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		if result.Value == nil {
			continue
		}

		for _, subscription := range result.Value {
			useSubscription := true

			if len(azureClient.subscriptionList) > 0 {
				// use subscription filter
				useSubscription = false
				for _, subscriptionId := range azureClient.subscriptionList {
					if strings.EqualFold(*subscription.SubscriptionID, subscriptionId) {
						useSubscription = true
						break
					}
				}
			}

			// filter by tag selector (using kubernetes label selector)
			if subscriptionTagSelector != nil {
				tags := labels.Set(to.StringMap(subscription.Tags))
				if (*subscriptionTagSelector).Matches(tags) {
					useSubscription = true
				} else {
					useSubscription = false
				}
			}

			if useSubscription {
				list[*subscription.SubscriptionID] = subscription
			}
		}
	}

	// update cache
	azureClient.cache.SetDefault(CacheIdentifierSubscriptions, list)

	return list, nil
}

// GetCachedSubscription returns a cached subscription
func (azureClient *ArmClient) GetCachedSubscription(ctx context.Context, subscriptionID string) (*armsubscriptions.Subscription, error) {
	list, err := azureClient.ListCachedSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	if subscription, exists := list[subscriptionID]; exists {
		return subscription, nil
	}

	return nil, fmt.Errorf(`no subscription with id "%s" found`, subscriptionID)
}

// GetSubscription returns a subscription
func (azureClient *ArmClient) GetSubscription(ctx context.Context, subscriptionID string) (*armsubscriptions.Subscription, error) {
	client, err := armsubscriptions.NewClient(azureClient.GetCred(), azureClient.NewArmClientOptions())
	if err != nil {
		return nil, err
	}

	result, err := client.Get(ctx, subscriptionID, nil)
	if err != nil {
		return nil, err
	}

	return &result.Subscription, nil
}
