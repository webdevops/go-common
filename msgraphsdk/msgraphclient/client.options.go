package msgraphclient

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

type ClientOptionFunc func(*MsGraphClient)

// WithCred sets the az credential
func WithCred(cred *azcore.TokenCredential) ClientOptionFunc {
	return func(client *MsGraphClient) {
		client.cred = cred
	}
}

// WithUserAgent sets the HTTP user agent
func WithUserAgent(userAgent string) ClientOptionFunc {
	return func(client *MsGraphClient) {
		client.userAgent = userAgent
	}
}
