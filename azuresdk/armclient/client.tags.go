package armclient

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"

	"github.com/webdevops/go-common/utils/to"
)

const (
	AzureTagSourceSeparator = "/"
	AzureTagOptionCharacter = "?"

	AzureTagSourceResource      = "resource"
	AzureTagSourceResourceGroup = "resourcegroup"
	AzureTagSourceSubscription  = "subscription"
)

type (
	ResourceTagResult struct {
		Source   string
		TagName  string
		TagValue string
	}
)

// ListCachedSubscriptionsWithFilter return list of subscription with filter by subscription ids
func (azureClient *ArmClient) GetResourceTag(ctx context.Context, resourceID string, tagList []string) ([]ResourceTagResult, error) {
	var (
		azureResourceTags  *armresources.Tags
		azureResourceGroup *armresources.ResourceGroup
		azureSubscription  *armsubscriptions.Subscription
	)
	var ret []ResourceTagResult

	resourceInfo, err := ParseResourceId(resourceID)
	if err != nil {
		return ret, err
	}

	fetchTagValue := func(tagName, tagSource string) (string, error) {
		tagValue := ""

		// fetch tag value
		switch tagSource {
		case AzureTagSourceResource:
			if val, err := azureClient.GetCachedTagsForResource(ctx, resourceID); err == nil {
				azureResourceTags = val
			} else {
				return tagValue, err
			}

			if val, exists := azureResourceTags.Tags[tagName]; exists {
				tagValue = to.String(val)
			}

		case AzureTagSourceResourceGroup:
			// get resourceGroup
			if azureResourceGroup == nil {
				resourceGroupName := strings.ToLower(resourceInfo.ResourceName)
				if list, err := azureClient.ListCachedResourceGroups(ctx, resourceInfo.Subscription); err == nil {
					if val, exists := list[resourceGroupName]; exists {
						azureResourceGroup = val
					} else {
						return tagValue, fmt.Errorf(`resourceGroup "%v" not found`, resourceGroupName)
					}
				} else {
					return tagValue, err
				}
			}

			if val, exists := azureResourceGroup.Tags[tagName]; exists {
				tagValue = to.String(val)
			}

		case AzureTagSourceSubscription:
			// get subscription
			if azureSubscription == nil {
				if list, err := azureClient.ListCachedSubscriptions(ctx); err == nil {
					if val, exists := list[resourceInfo.Subscription]; exists {
						azureSubscription = val
					} else {
						return tagValue, fmt.Errorf(`subscription "%v" not found`, resourceInfo.Subscription)
					}
				} else {
					return tagValue, err
				}
			}

			if val, exists := azureResourceGroup.Tags[tagName]; exists {
				tagValue = to.String(val)
			}

		default:
			return tagValue, fmt.Errorf(`resourceTag source "%v" is not supported`, tagSource)
		}

		tagValue = strings.TrimSpace(tagValue)

		return tagValue, nil
	}

	for _, rawTagName := range tagList {
		// default
		tagName := rawTagName
		tagSource := AzureTagSourceResource
		tagOptions := ""
		tagValue := ""

		// detect if tag has different source
		// eg subscription/foobar
		if strings.Contains(tagName, AzureTagSourceSeparator) {
			if parts := strings.SplitN(tagName, AzureTagSourceSeparator, 2); len(parts) == 2 {
				tagSource = strings.ToLower(parts[0])
				tagName = parts[1]
			}
		}

		// fetch options
		if strings.Contains(tagName, AzureTagOptionCharacter) {
			if parts := strings.SplitN(tagName, AzureTagOptionCharacter, 2); len(parts) == 2 {
				tagName = parts[0]
				tagOptions = parts[1]
			}
		}

		// fetch tag value
		if val, err := fetchTagValue(tagName, tagSource); err == nil {
			tagValue = val
		} else {
			return ret, err
		}

		// apply options
		if tagOptions != "" {
			options, err := url.ParseQuery(tagOptions)
			if err != nil {
				return ret, err
			}

			if options.Has("inherit") {
				// only inherit if empty
				// try resource -> resourcegroup
				if tagValue == "" && tagSource == AzureTagSourceResource {
					// fetch tag value
					if val, err := fetchTagValue(tagName, AzureTagSourceResourceGroup); err == nil {
						tagValue = val
					} else {
						return ret, err
					}
				}

				// only inherit if empty
				// try resourcegroup -> subscription
				if tagValue == "" && tagSource == AzureTagSourceResourceGroup {
					// fetch tag value
					if val, err := fetchTagValue(tagName, AzureTagSourceSubscription); err == nil {
						tagValue = val
					} else {
						return ret, err
					}
				}
			}

			if val := options.Get("name"); len(val) >= 1 {
				tagName = val
			}

			if options.Has("toLower") || options.Has("tolower") {
				tagValue = strings.ToLower(tagValue)
			}

			if options.Has("toUpper") || options.Has("toupper") {
				tagValue = strings.ToUpper(tagValue)
			}
		}

		ret = append(
			ret,
			ResourceTagResult{
				Source:   tagSource,
				TagName:  tagName,
				TagValue: tagValue,
			},
		)
	}

	return ret, nil
}

func (azureClient *ArmClient) GetCachedTagsForResource(ctx context.Context, resourceID string) (*armresources.Tags, error) {
	identifier := "tags:" + resourceID
	result, err := azureClient.cacheData(identifier, func() (interface{}, error) {
		list, err := azureClient.GetTagsForResource(ctx, resourceID)
		if err != nil {
			return list, err
		}
		return list, nil
	})
	if err != nil {
		return nil, err
	}

	return result.(*armresources.Tags), nil
}

func (azureClient *ArmClient) GetTagsForResource(ctx context.Context, resourceID string) (*armresources.Tags, error) {
	resourceInfo, err := ParseResourceId(resourceID)
	if err != nil {
		return nil, err
	}

	client, err := armresources.NewTagsClient(resourceInfo.Subscription, azureClient.GetCred(), azureClient.NewArmClientOptions())
	if err != nil {
		return nil, err
	}

	tags, err := client.GetAtScope(ctx, resourceID, nil)
	if err != nil {
		return nil, err
	}

	return tags.TagsResource.Properties, nil
}
