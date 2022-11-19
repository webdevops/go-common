package loganalytics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"

	"github.com/webdevops/go-common/azuresdk/armclient"
	"github.com/webdevops/go-common/azuresdk/cloudconfig"
)

const (
	OperationInsightsWorkspaceUrlSuffix = "/v1"
)

type (
	LogAnaltyicsQueryResult struct {
		Tables *[]struct {
			Name    string `json:"name"`
			Columns *[]struct {
				Name *string `json:"name"`
				Type *string `json:"type"`
			} `json:"columns"`
			Rows *[][]interface{} `json:"rows"`
		} `json:"tables"`
	}
)

func ExecuteQuery(ctx context.Context, armClient *armclient.ArmClient, mainWorkspaceId string, query string, timespan *string, additionalWorkspaceIds *[]string) (*LogAnaltyicsQueryResult, error) {
	var baseUrl string
	if val, exists := armClient.GetCloudConfig().Services[cloudconfig.ServiceNameLogAnalyticsWorkspace]; exists {
		baseUrl = val.Endpoint
	} else {
		return nil, fmt.Errorf(`"logAnalytics" config not set in cloudconfig`)
	}

	scopeUrl := fmt.Sprintf("%s/.default", baseUrl)
	queryUrl := fmt.Sprintf("%s/v1/workspaces/%s/query", baseUrl, mainWorkspaceId)

	credToken, err := armClient.GetCred().GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{scopeUrl},
	})
	if err != nil {
		return nil, err
	}

	requestBody := struct {
		Query      *string   `json:"query"`
		Workspaces *[]string `json:"workspaces"`
		Timespan   *string   `json:"timespan"`
	}{
		Query:      &query,
		Workspaces: additionalWorkspaceIds,
		Timespan:   timespan,
	}

	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	bytes.NewBuffer(requestBodyBytes)

	req, err := http.NewRequest(http.MethodPost, queryUrl, bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		return nil, err
	}
	req.Method = http.MethodPost
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("authorization", fmt.Sprintf("Bearer %s", credToken.Token))
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var result LogAnaltyicsQueryResult
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
