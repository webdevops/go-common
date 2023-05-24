package armclient

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/webdevops/go-common/utils/to"
)

var (
	azureTagNameToPrometheusNameRegExp = regexp.MustCompile("[^_a-zA-Z0-9]")
)

const (
	AzurePrometheusLabelPrefix = "tag_"

	AzureTagOptionCharacter = "?"

	AzureTagSourceResource      = "resource"
	AzureTagSourceResourceGroup = "resourcegroup"
	AzureTagSourceSubscription  = "subscription"
)

type (
	ArmClientTagManager struct {
		client *ArmClient
		logger *zap.SugaredLogger
	}
)

type (
	ResourceTagResult struct {
		Source     string
		TagName    string
		TagValue   string
		TargetName string
	}

	ResourceTagManager struct {
		Tags   []ResourceTagConfigTag
		client *ArmClient
	}

	ResourceTagConfigTag struct {
		Name       string
		Source     string
		TargetName string
		Inherit    bool
		Transform  ResourceTagConfigTransform
	}

	ResourceTagConfigTransform struct {
		ToLower bool
		ToUpper bool
	}
)

// GetResourceTag return list of resourceTags by resourceId
func (tagmgr *ArmClientTagManager) GetResourceTag(ctx context.Context, resourceID string, config *ResourceTagManager) ([]ResourceTagResult, error) {
	var (
		azureResource      *armresources.GenericResourceExpanded
		azureResourceGroup *armresources.ResourceGroup
		azureSubscription  *armsubscriptions.Subscription
	)

	ret := make([]ResourceTagResult, len(config.Tags))

	// prefill tag config, this should not be empty in case of error
	i := -1
	for _, tagConfig := range config.Tags {
		i++

		// default
		ret[i] = ResourceTagResult{
			TagName:    tagConfig.Name,
			TagValue:   "",
			TargetName: tagConfig.TargetName,
		}
	}

	// parse resource id
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

	i = -1
	for _, tagConfig := range config.Tags {
		i++

		// default
		result := ResourceTagResult{
			TagName:    tagConfig.Name,
			TagValue:   "",
			TargetName: tagConfig.TargetName,
		}

		// automatic set tag source based on resource info
		if tagConfig.Source == "" {
			tagConfig.Source = AzureTagSourceResource
			if resourceInfo.ResourceName == "" {
				tagConfig.Source = AzureTagSourceResourceGroup
			}
			if resourceInfo.ResourceGroup == "" {
				tagConfig.Source = AzureTagSourceSubscription
			}
		}

		// fetch tag value from source resource
		// we might want to ignore if tag source is resource but resource is resourceGroup
		switch tagConfig.Source {
		case AzureTagSourceResource:
			if resourceInfo.ResourceName != "" {
				if val, err := fetchTagValue(tagConfig.Name, AzureTagSourceResource); err == nil {
					result.TagValue = val
					result.Source = AzureTagSourceResource
				} else {
					tagmgr.logger.Debugf(`unable to fetch tagValue for resourceID "%s": %v`, resourceID, err.Error())
					result.TagValue = ""
				}
			}
		case AzureTagSourceResourceGroup:
			if resourceInfo.ResourceGroup != "" {
				if val, err := fetchTagValue(tagConfig.Name, AzureTagSourceResourceGroup); err == nil {
					result.TagValue = val
					result.Source = AzureTagSourceResourceGroup
				} else {
					tagmgr.logger.Debugf(`unable to fetch tagValue for resourceID "%s": %v`, resourceID, err.Error())
					result.TagValue = ""
				}
			}
		case AzureTagSourceSubscription:
			if resourceInfo.Subscription != "" {
				if val, err := fetchTagValue(tagConfig.Name, AzureTagSourceSubscription); err == nil {
					result.TagValue = val
					result.Source = AzureTagSourceSubscription
				} else {
					tagmgr.logger.Debugf(`unable to fetch tagValue for resourceID "%s": %v`, resourceID, err.Error())
					result.TagValue = ""
				}
			}
		}

		if tagConfig.Inherit {
			// only inherit if empty
			// try resource -> resourcegroup
			if result.TagValue == "" {
				// fetch tag value
				if val, err := fetchTagValue(tagConfig.Name, AzureTagSourceResourceGroup); err == nil {
					result.TagValue = val
					result.Source = AzureTagSourceResourceGroup
				} else {
					tagmgr.logger.Debugf(`unable to fetch tagValue for resourceID "%s" (inherit from ResourceGroup): %v`, resourceID, err.Error())
					result.TagValue = ""
				}
			}

			// only inherit if empty
			// try resourcegroup -> subscription
			if result.TagValue == "" {
				// fetch tag value
				if val, err := fetchTagValue(tagConfig.Name, AzureTagSourceSubscription); err == nil {
					result.TagValue = val
					result.Source = AzureTagSourceSubscription
				} else {
					tagmgr.logger.Debugf(`unable to fetch tagValue for resourceID "%s" (inherit from Subscription): %v`, resourceID, err.Error())
					result.TagValue = ""
				}
			}
		}

		// apply transformations
		if tagConfig.Transform.ToLower {
			result.TagValue = strings.ToLower(result.TagValue)
		}

		if tagConfig.Transform.ToUpper {
			result.TagValue = strings.ToUpper(result.TagValue)
		}

		ret[i] = result
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

func (tagmgr *ArmClientTagManager) ParseTagConfig(tags []string) (*ResourceTagManager, error) {
	return tagmgr.ParseTagConfigWithCustomPrefix(tags, AzurePrometheusLabelPrefix)
}

func (tagmgr *ArmClientTagManager) ParseTagConfigWithCustomPrefix(tags []string, labelPrefix string) (*ResourceTagManager, error) {
	config := &ResourceTagManager{
		Tags:   make([]ResourceTagConfigTag, len(tags)),
		client: tagmgr.client,
	}

	i := 0
	for _, tag := range tags {
		tagConfig, err := tagmgr.parseTagConfig(tag, labelPrefix)
		if err != nil {
			tagmgr.logger.Panicf(`unable to parse tag config "%s": %v`, tag, err.Error())
		}
		config.Tags[i] = tagConfig
		i++
	}

	return config, nil
}

// AddResourceTagsToPrometheusLabelsDefinitionWithCustomPrefix adds tags to label list with custom prefix
func (tagmgr *ArmClientTagManager) parseTagConfig(tag, labelPrefix string) (ResourceTagConfigTag, error) {
	var err error
	config := ResourceTagConfigTag{
		Name:       tag,
		TargetName: tag,
		Inherit:    false,
		Transform:  ResourceTagConfigTransform{},
	}

	options := url.Values{}

	// fetch options
	if strings.Contains(config.Name, AzureTagOptionCharacter) {
		if parts := strings.SplitN(config.Name, AzureTagOptionCharacter, 2); len(parts) == 2 {
			config.Name = parts[0]
			config.TargetName = parts[0]
			options, err = url.ParseQuery(parts[1])
			if err != nil {
				return config, err
			}
		}
	}

	if options != nil {
		if options.Has("name") {
			config.TargetName = options.Get("name")
		}

		if options.Has("inherit") {
			config.Inherit = true
		}

		if options.Has("source") {
			switch strings.ToLower(options.Get("source")) {
			case AzureTagSourceResource:
				config.Source = AzureTagSourceResource
			case AzureTagSourceResourceGroup:
				config.Source = AzureTagSourceResourceGroup
			case AzureTagSourceSubscription:
				config.Source = AzureTagSourceSubscription
			default:
				return config, fmt.Errorf(`invalid source "%s"`, options.Get("source"))
			}
		}

		if options.Has("toLower") || options.Has("tolower") {
			config.Transform.ToLower = true
		}

		if options.Has("toUpper") || options.Has("toupper") {
			config.Transform.ToUpper = true
		}
	}

	config.TargetName = labelPrefix + azureTagNameToPrometheusNameRegExp.ReplaceAllLiteralString(strings.ToLower(config.TargetName), "_")

	return config, nil
}

// AddToPrometheusLabels add prometheus tag labels to existing labels
func (c *ResourceTagManager) AddToPrometheusLabels(labels []string) []string {
	for _, tagConfig := range c.Tags {
		labels = append(labels, tagConfig.TargetName)
	}
	return labels
}

// AddResourceTagsToPrometheusLabels adds resource tags to prometheus labels
func (c *ResourceTagManager) AddResourceTagsToPrometheusLabels(ctx context.Context, labels prometheus.Labels, resourceID string) prometheus.Labels {
	resourceTags, err := c.client.TagManager.GetResourceTag(ctx, resourceID, c)
	if err != nil {
		c.client.TagManager.logger.Warnf(`unable to fetch resource tags for resource "%s": %v`, resourceID, err.Error())
	}

	for _, tag := range resourceTags {
		labels[tag.TargetName] = tag.TagValue
	}

	return labels
}
