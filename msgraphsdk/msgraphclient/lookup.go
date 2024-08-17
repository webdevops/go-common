package msgraphclient

import (
	"context"
	"strings"

	"github.com/microsoftgraph/msgraph-sdk-go/directoryobjects"
	"github.com/microsoftgraph/msgraph-sdk-go/models"

	"github.com/webdevops/go-common/utils/to"
)

const (
	DirectoryObjectTypeUnknown          = "unknown"
	DirectoryObjectTypeUser             = "user"
	DirectoryObjectTypeGroup            = "group"
	DirectoryObjectTypeApplication      = "application"
	DirectoryObjectTypeServicePrincipal = "serviceprincipal"
)

type (
	DirectoryObject struct {
		// general
		OdataType   string
		Type        string
		DisplayName string
		ObjectID    string

		DirectoryObject models.DirectoryObjectable

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

// IsUser returns true if object is a user
func (obj *DirectoryObject) IsUser() bool {
	return obj.Type == DirectoryObjectTypeUser
}

// IsGroup returns true if object is a group
func (obj *DirectoryObject) IsGroup() bool {
	return obj.Type == DirectoryObjectTypeGroup
}

// IsApplication returns true if object is an application
func (obj *DirectoryObject) IsApplication() bool {
	return obj.Type == DirectoryObjectTypeApplication
}

// IsServicePrincipal returns true if object is a servicePrincipal
func (obj *DirectoryObject) IsServicePrincipal() bool {
	return obj.Type == DirectoryObjectTypeServicePrincipal
}

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

		result, err := c.ServiceClient().DirectoryObjects().GetByIds().PostAsGetByIdsPostResponse(ctx, requestBody, nil)
		if err != nil {
			return ret, err
		}

		for _, row := range result.GetValue() {
			objectInfo := &DirectoryObject{
				ObjectID:        to.String(row.GetId()),
				OdataType:       to.String(row.GetOdataType()),
				Type:            DirectoryObjectTypeUnknown,
				DirectoryObject: row,
			}

			if user, ok := row.(models.Userable); ok {
				objectInfo.Type = DirectoryObjectTypeUser
				objectInfo.DisplayName = to.String(user.GetDisplayName())
				objectInfo.UserPrincipalName = user.GetUserPrincipalName()
				objectInfo.Email = user.GetMail()
			} else if group, ok := row.(models.Groupable); ok {
				objectInfo.Type = DirectoryObjectTypeGroup
				objectInfo.DisplayName = to.String(group.GetDisplayName())
			} else if app, ok := row.(models.Applicationable); ok {
				objectInfo.Type = DirectoryObjectTypeApplication
				objectInfo.DisplayName = to.String(app.GetDisplayName())
				objectInfo.ApplicationID = app.GetAppId()
			} else if sp, ok := row.(models.ServicePrincipalable); ok {
				objectInfo.Type = DirectoryObjectTypeServicePrincipal
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
