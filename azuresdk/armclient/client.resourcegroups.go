package armclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/webdevops/go-common/utils/to"
)

// ListCachedResourceGroups return cached list of Azure ResourceGroups as map (key is name of ResourceGroup)
func (azureClient *ArmClient) ListCachedResourceGroups(ctx context.Context, subscriptionID string) (map[string]*armresources.ResourceGroup, error) {
	identifier := "resourcegroups:" + subscriptionID
	result, err := azureClient.cacheData(identifier, func() (interface{}, error) {
		azureClient.logger.WithField("subscriptionID", subscriptionID).Debug("updating cached Azure ResourceGroup list")
		list, err := azureClient.ListResourceGroups(ctx, subscriptionID)
		if err != nil {
			return list, err
		}
		azureClient.logger.WithField("subscriptionID", subscriptionID).Debugf("found %v Azure ResourceGroups", len(list))
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

	return list, nil
}
