package armclient

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	resourceIdRegExp = regexp.MustCompile(`(?i)^/subscriptions/(?P<subscription>[^/]+)(/resourceGroups/(?P<resourceGroup>[^/]+))?(/providers/(?P<resourceProviderNamespace>[^/]+)/(?P<resourceProvider>[^/]+)/(?P<resourceName>[^/]+)(/(?P<resourceSubPath>.+))?)?/?$`)
)

type (
	AzureResourceDetails struct {
		OriginalResourceId        string
		Subscription              string
		ResourceGroup             string
		ResourceProviderNamespace string
		ResourceProviderName      string
		ResourceType              string
		ResourceName              string
		ResourceSubPath           string
	}
)

// Rebuild ResourceID
func (resource *AzureResourceDetails) ResourceId() (resourceId string) {
	if resource.Subscription != "" {
		resourceId += fmt.Sprintf(
			"/subscriptions/%s",
			resource.Subscription,
		)
	} else {
		return
	}

	if resource.ResourceGroup != "" {
		resourceId += fmt.Sprintf(
			"/resourceGroups/%s",
			resource.ResourceGroup,
		)
	}

	if resource.ResourceProviderNamespace != "" && resource.ResourceProviderName != "" && resource.ResourceName != "" {
		resourceId += fmt.Sprintf(
			"/providers/%s/%s/%s",
			resource.ResourceProviderNamespace,
			resource.ResourceProviderName,
			resource.ResourceName,
		)

		if resource.ResourceSubPath != "" {
			resourceId += fmt.Sprintf(
				"/%s",
				resource.ResourceSubPath,
			)
		}
	}

	return
}

func (resource *AzureResourceDetails) ResourceProvider() (provider string) {
	if resource.ResourceProviderName != "" && resource.ResourceProviderNamespace != "" && resource.ResourceName != "" {

	}
	return provider
}

// Parse Azure ResourceID and returns AzureResourceDetails object with splitted and lowercased information fields
func ParseResourceId(resourceId string) (resource *AzureResourceDetails, err error) {
	resource = &AzureResourceDetails{}

	if matches := resourceIdRegExp.FindStringSubmatch(resourceId); len(matches) >= 1 {
		resource.OriginalResourceId = resourceId
		for i, name := range resourceIdRegExp.SubexpNames() {
			v := strings.TrimSpace(matches[i])
			if i != 0 && name != "" {
				switch name {
				case "subscription":
					resource.Subscription = strings.ToLower(v)
				case "resourceGroup":
					resource.ResourceGroup = strings.ToLower(v)
				case "resourceProvider":
					resource.ResourceProviderName = strings.ToLower(v)
				case "resourceProviderNamespace":
					resource.ResourceProviderNamespace = strings.ToLower(v)
				case "resourceName":
					resource.ResourceName = strings.ToLower(v)
				case "resourceSubPath":
					resource.ResourceSubPath = strings.Trim(v, "/")
				}
			}
		}

		// build resourcetype
		if resource.ResourceProviderNamespace != "" && resource.ResourceProviderName != "" {
			resource.ResourceType = fmt.Sprintf(
				"%s/%s",
				resource.ResourceProviderNamespace,
				resource.ResourceProviderName,
			)
		}

	} else {
		err = fmt.Errorf("unable to parse Azure resourceID \"%v\"", resourceId)
	}

	return
}
