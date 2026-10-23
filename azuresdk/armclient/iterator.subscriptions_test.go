package armclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	cache "github.com/patrickmn/go-cache"

	"github.com/webdevops/go-common/azuresdk/cloudconfig"
)

const (
	iteratorTestSubscriptionID      = "00000000-0000-0000-0000-000000000001"
	iteratorTestOtherSubscriptionID = "00000000-0000-0000-0000-000000000002"
)

func TestSubscriptionIteratorRecoversAfterEmptyDiscovery(t *testing.T) {
	for _, preload := range []bool{false, true} {
		name := "constructor"
		if preload {
			name = "direct discovery before constructor"
		}
		t.Run(name, func(t *testing.T) {
			client, mock := newIteratorHTTPTestClient(t)
			if preload {
				if _, err := client.ListSubscriptions(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			iterator := NewSubscriptionIterator(client, iteratorTestSubscriptionID)
			for range 3 {
				assertIteratorSubscriptions(t, iterator)
			}
			assertIteratorRequests(t, mock, 1)

			mock.setSubscriptions(iteratorTestSubscriptionID, iteratorTestOtherSubscriptionID)
			waitForIteratorCacheExpiration(t, client)
			assertIteratorSubscriptions(t, iterator, iteratorTestSubscriptionID)
			assertIteratorRequests(t, mock, 2)
			assertIteratorSubscriptions(t, iterator, iteratorTestSubscriptionID)
			assertIteratorRequests(t, mock, 2)
		})
	}
}

func TestSubscriptionIteratorFiltersAndCopiesIDs(t *testing.T) {
	client, mock := newIteratorHTTPTestClient(t)
	mock.setSubscriptions(iteratorTestSubscriptionID, iteratorTestOtherSubscriptionID)
	ids := []string{iteratorTestSubscriptionID}
	iterator := NewSubscriptionIterator(client, ids...)
	ids[0] = iteratorTestOtherSubscriptionID
	assertIteratorSubscriptions(t, iterator, iteratorTestSubscriptionID)

	replacement := []string{iteratorTestOtherSubscriptionID}
	if iterator.SetSubscriptions(replacement...) != iterator {
		t.Fatal("SetSubscriptions did not return the iterator")
	}
	replacement[0] = iteratorTestSubscriptionID
	assertIteratorSubscriptions(t, iterator, iteratorTestOtherSubscriptionID)

	iterator.SetSubscriptions()
	assertIteratorSubscriptions(t, iterator, iteratorTestSubscriptionID, iteratorTestOtherSubscriptionID)
	assertIteratorRequests(t, mock, 1)
}

func TestSubscriptionIteratorNormalStartup(t *testing.T) {
	client, mock := newIteratorHTTPTestClient(t)
	mock.setSubscriptions(iteratorTestSubscriptionID, iteratorTestOtherSubscriptionID)
	iterator := NewSubscriptionIterator(client, iteratorTestSubscriptionID)
	assertIteratorRequests(t, mock, 1)
	assertIteratorSubscriptions(t, iterator, iteratorTestSubscriptionID)
	assertIteratorRequests(t, mock, 1)

	unfiltered := NewSubscriptionIterator(client)
	assertIteratorSubscriptions(t, unfiltered, iteratorTestSubscriptionID, iteratorTestOtherSubscriptionID)
	assertIteratorRequests(t, mock, 1)
}

func TestSubscriptionIteratorSurfacesRefreshErrors(t *testing.T) {
	client, mock := newIteratorHTTPTestClient(t)
	mock.setSubscriptions(iteratorTestSubscriptionID)
	iterator := NewSubscriptionIterator(client, iteratorTestSubscriptionID)
	assertIteratorSubscriptions(t, iterator, iteratorTestSubscriptionID)
	waitForIteratorCacheExpiration(t, client)
	mock.setForbidden()

	for range 2 {
		list, err := iterator.ListSubscriptions()
		var responseErr *azcore.ResponseError
		if list != nil || !errors.As(err, &responseErr) ||
			responseErr.StatusCode != http.StatusForbidden || responseErr.ErrorCode != "AuthorizationFailed" {
			t.Fatalf("expected surfaced ARM authorization error, got list=%v err=%v", list, err)
		}
	}
	assertIteratorRequests(t, mock, 3)

	mock.setSubscriptions(iteratorTestSubscriptionID, iteratorTestOtherSubscriptionID)
	assertIteratorSubscriptions(t, iterator, iteratorTestSubscriptionID)
	assertIteratorRequests(t, mock, 4)
}

func TestSubscriptionIteratorPreservesEagerDiscoveryErrors(t *testing.T) {
	for _, constructor := range []bool{false, true} {
		name := "setter"
		if constructor {
			name = "constructor"
		}
		t.Run(name, func(t *testing.T) {
			client, mock := newIteratorHTTPTestClient(t)
			mock.setForbidden()
			defer func() {
				if recovered := recover(); recovered == nil {
					t.Error("expected existing eager-discovery panic")
				}
				assertIteratorRequests(t, mock, 1)
			}()
			if constructor {
				NewSubscriptionIterator(client, iteratorTestSubscriptionID)
			} else {
				NewSubscriptionIterator(client).SetSubscriptions(iteratorTestSubscriptionID)
			}
		})
	}
}

type iteratorHTTPMock struct {
	mu       sync.Mutex
	ids      []string
	status   int
	requests int
}

func (mock *iteratorHTTPMock) setSubscriptions(ids ...string) {
	mock.mu.Lock()
	defer mock.mu.Unlock()
	mock.ids = append([]string(nil), ids...)
	mock.status = http.StatusOK
}

func (mock *iteratorHTTPMock) setForbidden() {
	mock.mu.Lock()
	defer mock.mu.Unlock()
	mock.status = http.StatusForbidden
}

func (mock *iteratorHTTPMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mock.mu.Lock()
	defer mock.mu.Unlock()
	mock.requests++
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet || r.URL.Path != "/subscriptions" ||
		r.URL.Query().Get("api-version") != "2022-12-01" {
		http.Error(w, "unexpected ARM request", http.StatusBadRequest)
		return
	}
	if mock.status == http.StatusForbidden {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"error":{"code":"AuthorizationFailed","message":"mock authorization failure"}}`)
		return
	}
	value := make([]map[string]string, 0, len(mock.ids))
	for _, id := range mock.ids {
		value = append(value, map[string]string{
			"subscriptionId": id,
			"displayName":    "mock subscription",
			"state":          "Enabled",
		})
	}
	if err := json.NewEncoder(w).Encode(struct {
		Value []map[string]string `json:"value"`
	}{Value: value}); err != nil {
		panic(err)
	}
}

func newIteratorHTTPTestClient(t *testing.T) (*ArmClient, *iteratorHTTPMock) {
	t.Helper()
	mock := &iteratorHTTPMock{}
	server := httptest.NewServer(mock)
	t.Cleanup(server.Close)
	// A nil credential permits local HTTP without changing TLS settings or contacting an identity service.
	var credential azcore.TokenCredential
	ttl := 100 * time.Millisecond
	client := &ArmClient{
		cloud: cloudconfig.CloudEnvironment{
			Configuration: cloud.Configuration{
				Services: map[cloud.ServiceName]cloud.ServiceConfiguration{
					cloud.ResourceManager: {Endpoint: server.URL, Audience: server.URL},
				},
			},
		},
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		cache:    cache.New(ttl, 0),
		cacheTtl: ttl,
		cred:     &credential,
	}
	return client, mock
}

func waitForIteratorCacheExpiration(t *testing.T, client *ArmClient) {
	t.Helper()
	_, expiration, ok := client.cache.GetWithExpiration(CacheIdentifierSubscriptions)
	if !ok {
		return
	}
	if expiration.IsZero() {
		t.Fatal("expected a finite subscription cache expiration")
	}
	time.Sleep(time.Until(expiration) + time.Millisecond)
}

func assertIteratorSubscriptions(t *testing.T, iterator *SubscriptionsIterator, ids ...string) {
	t.Helper()
	list, err := iterator.ListSubscriptions()
	if err != nil {
		t.Fatalf("list iterator subscriptions: %v", err)
	}
	if len(list) != len(ids) {
		t.Fatalf("expected subscriptions %v, got %v", ids, list)
	}
	for _, id := range ids {
		subscription, ok := list[id]
		if !ok || subscription == nil || subscription.SubscriptionID == nil || *subscription.SubscriptionID != id {
			t.Fatalf("expected subscription %s, got %v", id, subscription)
		}
	}
}

func assertIteratorRequests(t *testing.T, mock *iteratorHTTPMock, expected int) {
	t.Helper()
	mock.mu.Lock()
	defer mock.mu.Unlock()
	if mock.requests != expected {
		t.Fatalf("expected %d ARM requests, got %d", expected, mock.requests)
	}
}
