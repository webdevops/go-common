package armclient

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/remeh/sizedwaitgroup"
	"go.uber.org/zap"
)

type (
	SubscriptionsIterator struct {
		client        *ArmClient
		subscriptions *map[string]*armsubscriptions.Subscription

		concurrency int
	}
)

// NewSubscriptionIterator Creates new Azure Subscription iterator from Azure ARM client
func NewSubscriptionIterator(client *ArmClient, subscriptionID ...string) *SubscriptionsIterator {
	i := SubscriptionsIterator{}
	i.client = client
	i.concurrency = IteratorDefaultConcurrency
	if len(subscriptionID) >= 1 {
		i.SetSubscriptions(subscriptionID...)
	}
	return &i
}

// SetSubscriptions Set subscription id filter
func (i *SubscriptionsIterator) SetSubscriptions(subscriptionID ...string) *SubscriptionsIterator {
	ctx := context.Background()
	list, err := i.client.ListCachedSubscriptionsWithFilter(ctx, subscriptionID...)
	if err != nil {
		panic(err.Error())
	}

	i.subscriptions = &list
	return i
}

// SetConcurrency Set concurreny (for async loops)
func (i *SubscriptionsIterator) SetConcurrency(concurrency int) *SubscriptionsIterator {
	i.concurrency = concurrency
	return i
}

// ForEach Loop for each Azure Subscription without concurrency
func (i *SubscriptionsIterator) ForEach(logger *zap.SugaredLogger, callback func(subscription *armsubscriptions.Subscription, logger *zap.SugaredLogger)) error {
	subscriptionList, err := i.ListSubscriptions()
	if err != nil {
		return err
	}

	for _, subscription := range subscriptionList {
		contextLogger := logger.With(
			zap.String(`subscriptionID`, *subscription.SubscriptionID),
			zap.String(`subscriptionName`, *subscription.DisplayName),
		)
		callback(subscription, contextLogger)
	}

	return nil
}

// ForEachAsync Loop for each Azure Subscription with concurrency as background gofunc
func (i *SubscriptionsIterator) ForEachAsync(logger *zap.SugaredLogger, callback func(subscription *armsubscriptions.Subscription, logger *zap.SugaredLogger)) error {
	var panicList = []string{}
	panicLock := sync.Mutex{}
	wg := sizedwaitgroup.New(i.concurrency)

	subscriptionList, err := i.ListSubscriptions()
	if err != nil {
		return err
	}

	for _, subscription := range subscriptionList {
		wg.Add()

		go func(subscription *armsubscriptions.Subscription) {
			defer wg.Done()
			contextLogger := logger.With(
				zap.String(`subscriptionID`, *subscription.SubscriptionID),
				zap.String(`subscriptionName`, *subscription.DisplayName),
			)

			finished := false
			defer func() {
				if !finished {
					if err := recover(); err != nil {
						panicLock.Lock()
						defer panicLock.Unlock()

						msg := ""
						switch v := err.(type) {
						case error:
							msg = fmt.Sprintf("panic: %s\n%s", v.Error(), debug.Stack())
						default:
							msg = fmt.Sprintf("panic: %s\n%s", v, debug.Stack())
						}

						contextLogger.Errorf(msg)
						panicList = append(panicList, msg)
					}
				}
			}()

			callback(subscription, contextLogger)
			finished = true
		}(subscription)
	}

	wg.Wait()

	if len(panicList) >= 1 {
		panic("caught panics while processing SubscriptionsIterator.ForEachAsync: \n" + strings.Join(panicList, "\n-------------------------------------------------------------------------------\n"))
	}

	return nil
}

// ListSubscriptions Returns list of subscriptions for looping
func (i *SubscriptionsIterator) ListSubscriptions() (map[string]*armsubscriptions.Subscription, error) {
	var list map[string]*armsubscriptions.Subscription

	if i.subscriptions != nil {
		list = *i.subscriptions
	} else {
		if result, err := i.client.ListCachedSubscriptions(context.Background()); err == nil {
			list = result
		} else {
			return list, err
		}
	}

	return list, nil
}
