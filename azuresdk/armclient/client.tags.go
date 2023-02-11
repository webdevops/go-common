package armclient

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

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
	ArmClientTagManager struct {
		client *ArmClient
		logger *log.Logger
	}
)

type (
	ResourceTagResult struct {
		Source   string
		TagName  string
		TagValue string
	}
)

// GetResourceTag return list of resourceTags by resourceId
func (tagmgr *ArmClientTagManager) GetResourceTag(ctx context.Context, resourceID string, tagList []string) ([]ResourceTagResult, error) {
	var (
		azureResource      *armresources.GenericResourceExpanded
		azureResourceGroup *armresources.ResourceGroup
		azureSubscription  *armsubscriptions.Subscription
	)
	var ret []ResourceTagResult

	resourceID = strings.ToLower(resourceID)

	resourceInfo, err := ParseResourceId(resourceID)
	if err != nil {
		return ret, err
	}

	fetchTagValue := func(tagName, tagSource string) (string, error) {
		tagValue := ""

		// fetch tag value
		switch tagSource {
		case AzureTagSourceResource:
			if azureResource == nil {
				if resource, err := tagmgr.client.GetCachedResource(ctx, resourceID); err == nil && resource != nil {
					azureResource = resource
				} else {
					return tagValue, err
				}
			}

			if azureResource != nil {
				if val, exists := azureResource.Tags[tagName]; exists {
					tagValue = to.String(val)
				}
			}

		case AzureTagSourceResourceGroup:
			// get resourceGroup
			if azureResourceGroup == nil {
				resourceGroupName := strings.ToLower(resourceInfo.ResourceGroup)
				if list, err := tagmgr.client.ListCachedResourceGroups(ctx, resourceInfo.Subscription); err == nil {
					if val, exists := list[resourceGroupName]; exists {
						azureResourceGroup = val
					} else {
						return tagValue, fmt.Errorf(`resourceGroup "%v" not found`, resourceGroupName)
					}
				} else {
					return tagValue, err
				}
			}

			if azureResourceGroup != nil {
				if val, exists := azureResourceGroup.Tags[tagName]; exists {
					tagValue = to.String(val)
				}
			}

		case AzureTagSourceSubscription:
			// get subscription
			if azureSubscription == nil {
				if list, err := tagmgr.client.ListCachedSubscriptions(ctx); err == nil {
					if val, exists := list[resourceInfo.Subscription]; exists {
						azureSubscription = val
					} else {
						return tagValue, fmt.Errorf(`subscription "%v" not found`, resourceInfo.Subscription)
					}
				} else {
					return tagValue, err
				}
			}

			if azureSubscription != nil {
				if val, exists := azureSubscription.Tags[tagName]; exists {
					tagValue = to.String(val)
				}
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
		tagOptions := ""
		tagValue := ""

		tagSource := AzureTagSourceResource
		if resourceInfo.ResourceName == "" {
			tagSource = AzureTagSourceResourceGroup
		}

		if resourceInfo.ResourceGroup == "" {
			tagSource = AzureTagSourceSubscription
		}

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
			tagmgr.logger.Debugf(`unable to fetch tagValue for resourceID "%s": %v`, resourceID, err.Error())
			tagValue = ""
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
				if tagValue == "" {
					// fetch tag value
					if val, err := fetchTagValue(tagName, AzureTagSourceResourceGroup); err == nil {
						tagValue = val
					} else {
						tagmgr.logger.Debugf(`unable to fetch tagValue for resourceID "%s" (inherit from ResourceGroup): %v`, resourceID, err.Error())
						tagValue = ""
					}
				}

				// only inherit if empty
				// try resourcegroup -> subscription
				if tagValue == "" {
					// fetch tag value
					if val, err := fetchTagValue(tagName, AzureTagSourceSubscription); err == nil {
						tagValue = val
					} else {
						tagmgr.logger.Debugf(`unable to fetch tagValue for resourceID "%s" (inherit from Subscription): %v`, resourceID, err.Error())
						tagValue = ""
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

// GetCachedTagsForResource returns list of cached tags per resource
func (tagmgr *ArmClientTagManager) GetCachedTagsForResource(ctx context.Context, resourceID string) (*armresources.Tags, error) {
	identifier := "tags:" + resourceID
	result, err := tagmgr.client.cacheData(identifier, func() (interface{}, error) {
		list, err := tagmgr.GetTagsForResource(ctx, resourceID)
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

// GetTagsForResource returns list of tags per resource
func (tagmgr *ArmClientTagManager) GetTagsForResource(ctx context.Context, resourceID string) (*armresources.Tags, error) {
	resourceInfo, err := ParseResourceId(resourceID)
	if err != nil {
		return nil, err
	}

	client, err := armresources.NewTagsClient(resourceInfo.Subscription, tagmgr.client.GetCred(), tagmgr.client.NewArmClientOptions())
	if err != nil {
		return nil, err
	}

	tags, err := client.GetAtScope(ctx, resourceID, nil)
	if err != nil {
		return nil, err
	}

	return tags.TagsResource.Properties, nil
}

// AddResourceTagsToPrometheusLabels adds resource tags to prometheus labels
func (tagmgr *ArmClientTagManager) AddResourceTagsToPrometheusLabels(ctx context.Context, labels prometheus.Labels, resourceID string, tagList []string) prometheus.Labels {
	return tagmgr.AddResourceTagsToPrometheusLabelsWithCustomPrefix(ctx, labels, resourceID, tagList, AzurePrometheusLabelPrefix)
}

// AddResourceTagsToPrometheusLabelsWithCustomPrefix adds resource tags to prometheus labels with custom prefix
func (tagmgr *ArmClientTagManager) AddResourceTagsToPrometheusLabelsWithCustomPrefix(ctx context.Context, labels prometheus.Labels, resourceID string, tagList []string, labelPrefix string) prometheus.Labels {
	resourceTags, err := tagmgr.GetResourceTag(ctx, resourceID, tagList)
	if err != nil {
		tagmgr.logger.Warnf(`unable to fetch resource tags for resource "%s": %v`, resourceID, err.Error())
		return labels
	}

	for _, tag := range resourceTags {
		tagLabel := labelPrefix + azureTagNameToPrometheusNameRegExp.ReplaceAllLiteralString(tag.TagName, "_")
		labels[tagLabel] = tag.TagValue
	}

	return labels
}

// AddResourceTagsToPrometheusLabelsDefinition adds tags to label list
func (tagmgr *ArmClientTagManager) AddResourceTagsToPrometheusLabelsDefinition(labels, tags []string) []string {
	return tagmgr.AddResourceTagsToPrometheusLabelsDefinitionWithCustomPrefix(labels, tags, AzurePrometheusLabelPrefix)
}

// AddResourceTagsToPrometheusLabelsDefinitionWithCustomPrefix adds tags to label list with custom prefix
func (tagmgr *ArmClientTagManager) AddResourceTagsToPrometheusLabelsDefinitionWithCustomPrefix(labels, tags []string, labelPrefix string) []string {
	for _, rawTagName := range tags {
		tagName := rawTagName
		tagOptions := ""

		// detect if tag has different source
		// eg subscription/foobar
		if strings.Contains(tagName, AzureTagSourceSeparator) {
			if parts := strings.SplitN(tagName, AzureTagSourceSeparator, 2); len(parts) == 2 {
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

		// apply options
		if tagOptions != "" {
			options, err := url.ParseQuery(tagOptions)
			if err == nil {
				if val := options.Get("name"); len(val) >= 1 {
					tagName = val
				}
			}
		}

		labels = append(labels, labelPrefix+tagName)
	}

	return labels
}
