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
		Aud   *string `json:"aud"`
		Tid   *string `json:"tid"`
		AppId *string `json:"appid"`
		Oid   *string `json:"oid"`
		Upn   *string `json:"upn"`
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

	if t.Aud != nil {
		info["aud"] = *t.Aud
	}

	if t.Tid != nil {
		info["tid"] = *t.Tid
	}

	if t.AppId != nil {
		info["appid"] = *t.AppId
	}

	if t.Oid != nil {
		info["oid"] = *t.Oid
	}

	if t.Upn != nil {
		info["upd"] = *t.Upn
	}

	return info
}

func (t *AccessTokenInfo) ToJsonString() (info string) {
	if content, err := json.Marshal(t); err == nil {
		info = string(content)
	}

	return
}

func (t *AccessTokenInfo) ToString() string {
	var parts []string

	if t.AppId != nil {
		parts = append(parts, fmt.Sprintf("appid=%s", *t.AppId))
	}

	if t.Oid != nil {
		parts = append(parts, fmt.Sprintf("oid=%s", *t.Oid))
	}

	if t.Upn != nil {
		parts = append(parts, fmt.Sprintf("upn=%s", *t.Upn))
	}

	return strings.Join(parts, ", ")
}
