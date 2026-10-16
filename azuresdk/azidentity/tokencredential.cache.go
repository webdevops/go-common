package azidentity

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/patrickmn/go-cache"
)

const (
	AzTokenCredentialCachedExpiryTimeNegativeOffset = 5 * time.Minute
	AzTokenCredentialCachedExpiryTimeDefault = 10 * time.Minute
	AzTokenCredentialCachedCleanupInterval = 1 * time.Minute
)

type (
	AzCachedCredential struct {
		azcore.TokenCredential
		cache *cache.Cache
	}
)

// NewAzCliCachedCredential returns a new TokenCredential for az cli and token cache
func NewAzCliCachedCredential() (azcore.TokenCredential, error) {
	cred, err := NewAzCliCredential()
	if err != nil {
		return cred, err
	}

	return AzCachedCredential{TokenCredential: cred, cache: cache.New(AzTokenCredentialCachedExpiryTimeDefault, AzTokenCredentialCachedCleanupInterval)}, nil
}

// NewAzPowershellCachedCredential returns a new TokenCredential for powershell and token cache
func NewAzPowershellCachedCredential() (azcore.TokenCredential, error) {
	cred, err := NewAzPowershellCredential()
	if err != nil {
		return cred, err
	}

	return AzCachedCredential{TokenCredential: cred, cache: cache.New(AzTokenCredentialCachedExpiryTimeDefault, AzTokenCredentialCachedCleanupInterval)}, nil
}

// NewAzCachedCredential returns a new TokenCredential for az cli and token cache
func NewAzCachedCredential(cred azcore.TokenCredential) (azcore.TokenCredential, error) {
	return AzCachedCredential{TokenCredential: cred, cache: cache.New(AzTokenCredentialCachedExpiryTimeDefault, AzTokenCredentialCachedCleanupInterval)}, nil
}

// GetToken generates a token by calling az cli and caches it if possible for at least 10 minutes
func (c AzCachedCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	// fix double slash //.default to improve caching
	for i, v := range opts.Scopes {
		if strings.HasSuffix(v, "//.default") {
			opts.Scopes[i] = strings.TrimSuffix(v, "//.default") + "/.default"
		}
	}

	cacheKey := c.generateOptsCacheHash(opts)

	// invalid cache, pass through
	if cacheKey == "" {
		return c.TokenCredential.GetToken(ctx, opts)
	}

	// get from cache
	if val, exists := c.cache.Get(cacheKey); exists {
		if v, ok := val.(azcore.AccessToken); ok {
			return v, nil
		}
	}

	token, err := c.TokenCredential.GetToken(ctx, opts)
	if err != nil {
		return token, err
	}

	// set cache
	expiryDuration := time.Until(token.ExpiresOn) - AzTokenCredentialCachedExpiryTimeNegativeOffset
	c.cache.Set(cacheKey, token, expiryDuration)

	return token, nil
}

// generateOptsCacheHash generates a hash from the token options
func (c AzCachedCredential) generateOptsCacheHash(opts policy.TokenRequestOptions) string {
	hashData, err := json.Marshal(opts)
	if err != nil {
		return ""
	}

	hash := sha512.Sum512(hashData)

	return hex.EncodeToString(hash[:])
}
