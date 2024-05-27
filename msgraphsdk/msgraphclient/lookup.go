package msgraphclient

import (
	"context"
	"strings"

	"github.com/microsoftgraph/msgraph-sdk-go/directoryobjects"
	"github.com/microsoftgraph/msgraph-sdk-go/models"

	"github.com/webdevops/go-common/utils/to"
)

type (
	DirectoryObject struct {
		// general
		OdataType   string
		Type        string
		DisplayName string
		ObjectID    string

		// user
		UserPrincipalName *string
		Email             *string

		// servicePrincipal/application
		ServicePrincipalType *string
		ManagedIdentity      *string
		ApplicationID        *string

		// group
	}
)

// LookupPrincipalID returns information about AzureAD directory object by objectid
func (c *MsGraphClient) LookupPrincipalID(ctx context.Context, princpalIds ...string) (map[string]*DirectoryObject, error) {
	ret := map[string]*DirectoryObject{}

	// inject cached entries
	for _, objectId := range princpalIds {
		cacheKey := "object:" + objectId
		if val, ok := c.cache.Get(cacheKey); ok {
			if directoryObjectInfo, ok := val.(*DirectoryObject); ok {
				ret[objectId] = directoryObjectInfo
			}
		}
	}

	// build list of not cached entries
	lookupPrincipalObjectIDList := []string{}
	for _, princpalId := range princpalIds {
		if _, exists := ret[princpalId]; !exists {
			lookupPrincipalObjectIDList = append(lookupPrincipalObjectIDList, princpalId)
		}
	}

	// azure limits objects ids
	chunkSize := 999
	for i := 0; i < len(lookupPrincipalObjectIDList); i += chunkSize {
		end := i + chunkSize
		if end > len(lookupPrincipalObjectIDList) {
			end = len(lookupPrincipalObjectIDList)
		}

		principalObjectIDChunkList := lookupPrincipalObjectIDList[i:end]

		requestBody := directoryobjects.NewGetByIdsPostRequestBody()
		requestBody.SetIds(principalObjectIDChunkList)

		result, err := c.ServiceClient().DirectoryObjects().GetByIds().Post(ctx, requestBody, nil)
		if err != nil {
			return ret, err
		}

		for _, row := range result.GetValue() {
			objectInfo := &DirectoryObject{
				ObjectID:  to.String(row.GetId()),
				OdataType: to.String(row.GetOdataType()),
				Type:      "unknown",
			}

			if user, ok := row.(models.Userable); ok {
				objectInfo.Type = "user"
				objectInfo.DisplayName = to.String(user.GetDisplayName())
				objectInfo.UserPrincipalName = user.GetUserPrincipalName()
				objectInfo.Email = user.GetMail()
			} else if group, ok := row.(models.Groupable); ok {
				objectInfo.Type = "group"
				objectInfo.DisplayName = to.String(group.GetDisplayName())
			} else if app, ok := row.(models.Applicationable); ok {
				objectInfo.Type = "application"
				objectInfo.DisplayName = to.String(app.GetDisplayName())
				objectInfo.ApplicationID = app.GetAppId()
			} else if sp, ok := row.(models.ServicePrincipalable); ok {
				objectInfo.Type = "serviceprincipal"
				objectInfo.DisplayName = to.String(sp.GetDisplayName())
				objectInfo.ApplicationID = sp.GetAppId()
				objectInfo.ServicePrincipalType = sp.GetServicePrincipalType()

				if strings.EqualFold(to.String(objectInfo.ServicePrincipalType), "ManagedIdentity") {
					spAlternativeNames := sp.GetAlternativeNames()
					if len(spAlternativeNames) >= 2 {
						objectInfo.ManagedIdentity = to.Ptr(spAlternativeNames[1])
					}
				}
			}

			ret[objectInfo.ObjectID] = objectInfo

			// store in cache
			cacheKey := "object:" + objectInfo.ObjectID
			c.cache.Set(cacheKey, objectInfo, c.cacheTtl)
		}
	}

	return ret, nil
}
