package armclient

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/webdevops/go-common/utils/to"
)

const (
	CacheIdentifierResourceGroupList = "resourcegroups:%s"
	CacheIdentifierResourceGroup     = "resourcegroups:%s:%s"
)

// ListCachedResourceGroups return cached list of Azure ResourceGroups as map (key is name of ResourceGroup)
func (azureClient *ArmClient) ListCachedResourceGroups(ctx context.Context, subscriptionID string) (map[string]*armresources.ResourceGroup, error) {
	result, err := azureClient.cacheData(fmt.Sprintf(CacheIdentifierResourceGroupList, subscriptionID), func() (interface{}, error) {
		azureClient.logger.With(slog.String("subscriptionID", subscriptionID)).Debug("updating cached Azure ResourceGroup list")
		list, err := azureClient.ListResourceGroups(ctx, subscriptionID)
		if err != nil {
			return list, err
		}
		azureClient.logger.With(slog.String("subscriptionID", subscriptionID)).Debug(fmt.Sprintf("found %v Azure ResourceGroups", len(list)))
		return list, nil
	})
	if err != nil {
		return nil, err
	}

	return result.(map[string]*armresources.ResourceGroup), nil
}

// ListResourceGroups return list of Azure ResourceGroups as map (key is name of ResourceGroup)
func (azureClient *ArmClient) ListResourceGroups(ctx context.Context, subscriptionID string) (map[string]*armresources.ResourceGroup, error) {
	list := map[string]*armresources.ResourceGroup{}

	client, err := armresources.NewResourceGroupsClient(subscriptionID, azureClient.GetCred(), azureClient.NewArmClientOptions())
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

		for _, resourceGroup := range result.Value {
			list[to.StringLower(resourceGroup.Name)] = resourceGroup
		}
	}

	// update cache
	azureClient.cache.SetDefault(fmt.Sprintf(CacheIdentifierResourceGroupList, subscriptionID), list)

	return list, nil
}
