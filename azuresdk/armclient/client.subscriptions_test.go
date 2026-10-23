package armclient

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	cache "github.com/patrickmn/go-cache"

	"github.com/webdevops/go-common/azuresdk/cloudconfig"
	"github.com/webdevops/go-common/utils/to"
)

func TestSubscriptionCacheExpiration(t *testing.T) {
	tests := []struct {
		name       string
		ttl        time.Duration
		empty      bool
		expiration time.Duration
	}{
		{name: "empty default", ttl: time.Hour, empty: true, expiration: time.Minute},
		{name: "empty shorter", ttl: 10 * time.Second, empty: true, expiration: 10 * time.Second},
		{name: "empty equal", ttl: time.Minute, empty: true, expiration: time.Minute},
		{name: "empty zero", ttl: 0, empty: true, expiration: time.Minute},
		{name: "empty no expiration", ttl: cache.NoExpiration, empty: true, expiration: time.Minute},
		{name: "nonempty default", ttl: time.Hour, expiration: time.Hour},
		{name: "nonempty shorter", ttl: 10 * time.Second, expiration: 10 * time.Second},
		{name: "nonempty zero", ttl: 0},
		{name: "nonempty no expiration", ttl: cache.NoExpiration},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				client := newSubscriptionCacheTestClient(tt.ttl, nil)
				list := map[string]*armsubscriptions.Subscription{}
				if !tt.empty {
					list["subscription-1"] = &armsubscriptions.Subscription{
						SubscriptionID: to.Ptr("subscription-1"),
					}
				}

				client.cacheSubscriptions(list)
				_, expiration, ok := client.cache.GetWithExpiration(CacheIdentifierSubscriptions)
				if !ok {
					t.Fatal("successful discovery was not cached")
				}
				if tt.expiration == 0 {
					if !expiration.IsZero() {
						t.Fatalf("expected no expiration, got %v", expiration)
					}
					return
				}
				if got := time.Until(expiration); got != tt.expiration {
					t.Fatalf("expected cache lifetime %v, got %v", tt.expiration, got)
				}

				time.Sleep(tt.expiration - time.Nanosecond)
				if _, ok := client.cache.Get(CacheIdentifierSubscriptions); !ok {
					t.Fatal("subscription cache expired too soon")
				}
				time.Sleep(2 * time.Nanosecond)
				if _, ok := client.cache.Get(CacheIdentifierSubscriptions); ok {
					t.Fatal("subscription cache did not expire")
				}
			})
		})
	}
}

func TestListCachedSubscriptionsRetriesAfterEmptyExpiration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		authErr := errors.New("test credential unavailable")
		cred := &subscriptionTestCredential{err: authErr}
		client := newSubscriptionCacheTestClient(time.Hour, cred)
		// Seed the same cache entry as a successful empty ListSubscriptions during Connect.
		client.cacheSubscriptions(map[string]*armsubscriptions.Subscription{})

		for range 3 {
			list, err := client.ListCachedSubscriptions(context.Background())
			if err != nil || len(list) != 0 {
				t.Fatalf("expected cached empty success, got %v, %v", list, err)
			}
			time.Sleep(15 * time.Second)
		}
		if cred.calls != 0 {
			t.Fatalf("cached empty results triggered %d credential calls", cred.calls)
		}

		time.Sleep(15*time.Second + time.Nanosecond)
		for range 2 {
			list, err := client.ListCachedSubscriptions(context.Background())
			if list != nil || !errors.Is(err, authErr) {
				t.Fatalf("expected discovery retry to surface credential error, got %v, %v", list, err)
			}
		}
		if cred.calls != 2 {
			t.Fatalf("expected two uncached discovery retries, got %d", cred.calls)
		}
		if _, ok := client.cache.Get(CacheIdentifierSubscriptions); ok {
			t.Fatal("failed discovery was cached")
		}

		// Once discovery succeeds, the normal positive-result cache lifetime applies.
		client.cacheSubscriptions(map[string]*armsubscriptions.Subscription{
			"subscription-1": {SubscriptionID: to.Ptr("subscription-1")},
		})
		time.Sleep(2 * time.Minute)
		subscription, err := client.GetCachedSubscription(context.Background(), "subscription-1")
		if err != nil || subscription == nil || *subscription.SubscriptionID != "subscription-1" {
			t.Fatalf("expected recovered cached subscription, got %v, %v", subscription, err)
		}
		if cred.calls != 2 {
			t.Fatal("non-empty results were not retained for the configured TTL")
		}
	})
}

func TestListSubscriptionsErrorPreservesCache(t *testing.T) {
	authErr := errors.New("test credential unavailable")
	client := newSubscriptionCacheTestClient(time.Hour, &subscriptionTestCredential{err: authErr})
	client.cacheSubscriptions(map[string]*armsubscriptions.Subscription{
		"subscription-1": {SubscriptionID: to.Ptr("subscription-1")},
	})

	list, err := client.ListSubscriptions(context.Background())
	if list != nil || !errors.Is(err, authErr) {
		t.Fatalf("expected discovery error, got %v, %v", list, err)
	}
	subscription, err := client.GetCachedSubscription(context.Background(), "subscription-1")
	if err != nil || subscription == nil {
		t.Fatalf("failed refresh replaced successful cache entry: %v", err)
	}
}

func newSubscriptionCacheTestClient(ttl time.Duration, credential *subscriptionTestCredential) *ArmClient {
	var cred azcore.TokenCredential = credential
	return &ArmClient{
		cloud: cloudconfig.CloudEnvironment{
			Configuration: cloud.AzurePublic,
			Name:          cloudconfig.AzurePublicCloud,
		},
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		cache:    cache.New(ttl, 0),
		cacheTtl: ttl,
		cred:     &cred,
	}
}

type subscriptionTestCredential struct {
	err   error
	calls int
}

func (cred *subscriptionTestCredential) GetToken(context.Context, policy.TokenRequestOptions) (azcore.AccessToken, error) {
	cred.calls++
	return azcore.AccessToken{}, cred.err
}
