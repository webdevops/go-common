package azidentity

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/patrickmn/go-cache"
)

const (
	AzCliCachedExpiryTimeNegativeOffset = 5 * time.Minute
)

type (
	AzCliCachedCredential struct {
		azcore.TokenCredential
		cache *cache.Cache
	}
)

// NewAzCliCachedCredential returns a new TokenCredential for az cli and token cache
func NewAzCliCachedCredential() (azcore.TokenCredential, error) {
	creds, err := NewAzCliCredential()
	if err != nil {
		return creds, err
	}

	return AzCliCachedCredential{TokenCredential: creds, cache: cache.New(1*time.Minute, 1*time.Minute)}, nil
}

// GetToken generates a token by calling az cli and caches it if possible for at least 10 minutes
func (c AzCliCachedCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
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
	expiryDuration := time.Until(token.ExpiresOn) - AzCliCachedExpiryTimeNegativeOffset
	c.cache.Set(cacheKey, token, expiryDuration)

	return token, nil
}

// generateOptsCacheHash generates a hash from the token options
func (c AzCliCachedCredential) generateOptsCacheHash(opts policy.TokenRequestOptions) string {
	hashData, err := json.Marshal(opts)
	if err != nil {
		return ""
	}

	hash := sha512.Sum512(hashData)

	return hex.EncodeToString(hash[:])
}
