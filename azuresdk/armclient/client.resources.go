package armclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/webdevops/go-common/utils/to"
)

const (
	CacheIdentifierResourcesList = "resources:%s"
	CacheIdentifierResourcesID   = "resourceID:%s"
)

// GetCachedResource return cached Azure Resource by resourceID
func (azureClient *ArmClient) GetCachedResource(ctx context.Context, resourceID string) (*armresources.GenericResourceExpanded, error) {
	cacheKey := fmt.Sprintf(CacheIdentifierResourcesID, strings.ToLower(resourceID))
	result, err := azureClient.cacheData(cacheKey, func() (interface{}, error) {
		var resource *armresources.GenericResourceExpanded

		resourceInfo, err := ParseResourceId(resourceID)
		if err != nil {
			return nil, err
		}

		list, err := azureClient.ListCachedResources(ctx, resourceInfo.Subscription)
		if err != nil {
			return list, err
		}

		if val, exists := list[resourceID]; exists {
			return val, nil
		}

		// not found
		return resource, nil
	})
	if err != nil {
		return nil, err
	}

	return result.(*armresources.GenericResourceExpanded), nil
}

// ListCachedResources return cached list of Azure Resources as map (key is ResourceID)
func (azureClient *ArmClient) ListCachedResources(ctx context.Context, subscriptionID string) (map[string]*armresources.GenericResourceExpanded, error) {
	result, err := azureClient.cacheData(fmt.Sprintf(CacheIdentifierResourcesList, subscriptionID), func() (interface{}, error) {
		azureClient.logger.WithField("subscriptionID", subscriptionID).Debug("updating cached Azure Resource list")
		list, err := azureClient.ListResources(ctx, subscriptionID)
		if err != nil {
			return list, err
		}
		azureClient.logger.WithField("subscriptionID", subscriptionID).Debugf("found %v Azure Resources", len(list))
		return list, nil
	})
	if err != nil {
		return nil, err
	}

	return result.(map[string]*armresources.GenericResourceExpanded), nil
}

// ListResources return list of Azure Resources as map (key is ResourceID)
func (azureClient *ArmClient) ListResources(ctx context.Context, subscriptionID string) (map[string]*armresources.GenericResourceExpanded, error) {
	list := map[string]*armresources.GenericResourceExpanded{}

	client, err := armresources.NewClient(subscriptionID, azureClient.GetCred(), azureClient.NewArmClientOptions())
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

		for _, resource := range result.Value {
			list[to.StringLower(resource.ID)] = resource
		}
	}

	// update cache
	azureClient.cache.SetDefault(fmt.Sprintf(CacheIdentifierResourcesList, subscriptionID), list)

	for resourceID, resource := range list {
		azureClient.cache.SetDefault(fmt.Sprintf(CacheIdentifierResourcesID, resourceID), resource)
	}

	return list, nil
}
