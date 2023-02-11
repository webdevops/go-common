package azidentity

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

type (
	AccessTokenInfo struct {
		Aud   string `json:"aud"`
		Tid   string `json:"tid"`
		AppId string `json:"appid"`
		Oid   string `json:"oid"`
		Upn   string `json:"upn"`
	}
)

func ParseAccessToken(token azcore.AccessToken) *AccessTokenInfo {
	// parse token
	tokenParts := strings.SplitN(token.Token, ".", 3)
	if len(tokenParts) == 3 {
		if tokenInfoDecode, err := base64.RawStdEncoding.DecodeString(tokenParts[1]); err == nil {
			tokenInfo := AccessTokenInfo{}

			if err := json.Unmarshal(tokenInfoDecode, &tokenInfo); err == nil {
				return &tokenInfo
			}
		}
	}

	return nil
}

func (t *AccessTokenInfo) ToString() string {
	parts := []string{}

	if t.Aud != "" {
		parts = append(parts, fmt.Sprintf("aud=%s", t.Aud))
	}

	if t.Tid != "" {
		parts = append(parts, fmt.Sprintf("tid=%s", t.Tid))
	}

	if t.AppId != "" {
		parts = append(parts, fmt.Sprintf("appid=%s", t.AppId))
	}

	if t.Oid != "" {
		parts = append(parts, fmt.Sprintf("oid=%s", t.Oid))
	}

	if t.Upn != "" {
		parts = append(parts, fmt.Sprintf("upn=%s", t.Upn))
	}

	return strings.Join(parts, ", ")
}
