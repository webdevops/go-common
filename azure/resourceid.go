package azure

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	resourceIdRegExp = regexp.MustCompile(`(?i)/subscriptions/(?P<subscription>[^/]+)(/resourceGroups/(?P<resourceGroup>[^/]+))?(/providers/(?P<resourceProvider>[^/]*)/(?P<resourceProviderNamespace>[^/]*)/(?P<resourceName>[^/]+)(/(?P<resourceSubPath>.+))?)?`)
)

type (
	AzureResourceDetails struct {
		OriginalResourceId        string
		Subscription              string
		ResourceGroup             string
		ResourceProvider          string
		ResourceProviderNamespace string
		ResourceType              string
		ResourceName              string
		ResourceSubPath           string
	}
)

func (info *AzureResourceDetails) ResourceId() (resourceId string) {
	if info.Subscription != "" {
		resourceId += fmt.Sprintf("/subscriptions/%s", info.Subscription)
	} else {
		return
	}

	if info.ResourceGroup != "" {
		resourceId += fmt.Sprintf("/resourceGroups/%s", info.ResourceGroup)
	}

	if info.ResourceProvider != "" && info.ResourceProviderNamespace != "" && info.ResourceName != "" {
		resourceId += fmt.Sprintf("/providers/%s/%s/%s", info.ResourceProvider, info.ResourceProviderNamespace, info.ResourceName)

		if info.ResourceSubPath != "" {
			resourceId += fmt.Sprintf("/%s", info.ResourceSubPath)
		}
	}

	return
}

func ParseResourceId(resourceId string) (info *AzureResourceDetails, err error) {
	info = &AzureResourceDetails{}

	if matches := resourceIdRegExp.FindStringSubmatch(resourceId); len(matches) >= 1 {
		info.OriginalResourceId = resourceId
		for i, name := range resourceIdRegExp.SubexpNames() {
			if i != 0 && name != "" {
				switch name {
				case "subscription":
					info.Subscription = strings.ToLower(matches[i])
				case "resourceGroup":
					info.ResourceGroup = strings.ToLower(matches[i])
				case "resourceProvider":
					info.ResourceProvider = strings.ToLower(matches[i])
				case "resourceProviderNamespace":
					info.ResourceProviderNamespace = strings.ToLower(matches[i])
				case "resourceName":
					info.ResourceName = strings.ToLower(matches[i])
				case "resourceSubPath":
					info.ResourceSubPath = strings.Trim(matches[i], "/")
				}
			}
		}
	} else {
		err = fmt.Errorf("unable to parse Azure resourceID \"%v\"", resourceId)
	}

	return
}
