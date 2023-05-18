package armclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"

	"github.com/webdevops/go-common/utils/to"
)

const (
	ResourceGraphQueryOptionsTop = 1000
)

// ExecuteResourceGraphQuery executes a ResourceGraph query and returns the full result
func (azureClient *ArmClient) ExecuteResourceGraphQuery(ctx context.Context, subscriptions []string, query string) ([]map[string]interface{}, error) {
	list := []map[string]interface{}{}

	resourceGraphClient, err := armresourcegraph.NewClient(azureClient.GetCred(), azureClient.NewArmClientOptions())
	if err != nil {
		return list, err
	}

	requestQueryTop := int32(ResourceGraphQueryOptionsTop)
	requestQuerySkip := int32(0)

	// Set options
	resultFormat := armresourcegraph.ResultFormatObjectArray
	requestOptions := armresourcegraph.QueryRequestOptions{
		ResultFormat: &resultFormat,
		Top:          &requestQueryTop,
		Skip:         &requestQuerySkip,
	}

	for {
		// Create the query request
		request := armresourcegraph.QueryRequest{
			Subscriptions: to.SlicePtr(subscriptions),
			Query:         &query,
			Options:       &requestOptions,
		}

		var result, queryErr = resourceGraphClient.Resources(ctx, request, nil)
		if queryErr != nil {
			return list, queryErr
		}

		if resultList, ok := result.Data.([]interface{}); ok {
			for _, row := range resultList {
				if rowData, ok := row.(map[string]interface{}); ok {
					list = append(list, rowData)
				}
			}
		} else {
			// got invalid or empty data, skipping
			break
		}

		*requestOptions.Skip += requestQueryTop
		if result.TotalRecords != nil {
			if int64(*requestOptions.Skip) >= *result.TotalRecords {
				break
			}
		}
	}

	return list, err
}

// ListResourceIdsWithKustoFilter return list of Azure ResourceIDs using ResourceGraph query
func (azureClient *ArmClient) ListResourceIdsWithKustoFilter(ctx context.Context, subscriptions []string, filter []string) (map[string]string, error) {
	list := map[string]string{}

	query := "resources \n"
	for _, val := range filter {
		val = strings.TrimSpace(val)
		val = strings.TrimLeft(val, "|")
		if len(val) >= 1 {
			query += fmt.Sprintf("| %s \n", filter)
		}
	}
	query += "| project id"

	result, err := azureClient.ExecuteResourceGraphQuery(ctx, subscriptions, query)
	if err != nil {
		return list, err
	}

	for _, row := range result {
		if val, exists := row["id"].(string); exists {
			resourceId := strings.ToLower(val)
			list[resourceId] = resourceId
		}
	}

	return list, err
}
