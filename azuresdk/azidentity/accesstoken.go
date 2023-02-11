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

func (t *AccessTokenInfo) ToMap() map[string]string {
	info := map[string]string{}

	if t.Aud != "" {
		info["aud"] = t.Aud
	}

	if t.Tid != "" {
		info["tid"] = t.Tid
	}

	if t.AppId != "" {
		info["appid"] = t.AppId
	}

	if t.Oid != "" {
		info["oid"] = t.Oid
	}

	if t.Upn != "" {
		info["upd"] = t.Upn
	}

	return info
}

func (t *AccessTokenInfo) ToString() string {
	var parts []string
	for key, val := range t.ToMap() {
		parts = append(parts, fmt.Sprintf("%s=%s", key, val))
	}

	return strings.Join(parts, ", ")
}
