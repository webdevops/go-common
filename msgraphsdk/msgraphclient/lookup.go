package msgraphclient

import (
	"context"
	"strings"

	jsonserialization "github.com/microsoft/kiota-serialization-json-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/directoryobjects/getbyids"

	"github.com/webdevops/go-common/utils/to"
)

type (
	DirectoryObject struct {
		Type                 string
		ServicePrincipalType string
		ManagedIdentity      string
		DisplayName          string
		ObjectId             string
		ApplicationId        string
	}
)

// LookupPrincipalIdMap returns information about AzureAD directory object by objectid
func (c *MsGraphClient) LookupPrincipalIdMap(ctx context.Context, client *msgraphsdk.GraphServiceClient, princpalIds []string) (map[string]*DirectoryObject, error) {
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
	for PrincipalObjectID, directoryObjectInfo := range ret {
		if directoryObjectInfo == nil {
			lookupPrincipalObjectIDList = append(lookupPrincipalObjectIDList, PrincipalObjectID)
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

		opts := getbyids.GetByIdsPostRequestBody{}
		opts.SetIds(principalObjectIDChunkList)

		result, err := client.DirectoryObjects().GetByIds().Post(&opts)
		if err != nil {
			return ret, err
		}

		for _, row := range result.GetValue() {
			objectId := to.String(row.GetId())
			objectData := row.GetAdditionalData()

			objectType := ""
			if val, exists := objectData["@odata.type"]; exists {
				objectType = to.String(val.(*string))
				objectType = strings.ToLower(strings.TrimPrefix(objectType, "#microsoft.graph."))
			}

			servicePrincipalType := ""
			if val, exists := objectData["servicePrincipalType"]; exists {
				servicePrincipalType = to.String(val.(*string))
			}

			displayName := ""
			if val, exists := objectData["displayName"]; exists {
				displayName = to.String(val.(*string))
			}

			applicationId := ""
			if val, exists := objectData["appId"]; exists {
				applicationId = to.String(val.(*string))
			}

			managedIdentity := ""
			if strings.EqualFold(servicePrincipalType, "ManagedIdentity") {
				if alternativeNames, ok := objectData["alternativeNames"].([]*jsonserialization.JsonParseNode); ok {
					if len(alternativeNames) >= 2 {
						if val, err := alternativeNames[1].GetStringValue(); err == nil {
							managedIdentity = to.String(val)
						}
					}
				}
			}

			ret[objectId] = &DirectoryObject{
				ObjectId:             objectId,
				ApplicationId:        applicationId,
				Type:                 objectType,
				ServicePrincipalType: servicePrincipalType,
				ManagedIdentity:      managedIdentity,
				DisplayName:          displayName,
			}

			// store in cache
			cacheKey := "object:" + objectId
			c.cache.Set(cacheKey, ret[objectId], c.cacheTtl)
		}
	}

	return ret, nil
}
