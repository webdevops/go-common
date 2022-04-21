package azure

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/remeh/sizedwaitgroup"
	log "github.com/sirupsen/logrus"
)

type (
	SubscriptionsIterator struct {
		client        *Client
		subscriptions *[]subscriptions.Subscription

		concurrency int
	}
)

func NewSubscriptionIterator(client *Client, subscriptionID ...string) *SubscriptionsIterator {
	i := SubscriptionsIterator{}
	i.client = client
	i.concurrency = IteratorDefaultConcurrency
	if len(subscriptionID) >= 1 {
		i.SetSubscriptions(subscriptionID...)
	}
	return &i
}

func (i *SubscriptionsIterator) SetSubscriptions(subscriptionID ...string) *SubscriptionsIterator {
	subscriptionsClient := subscriptions.NewClientWithBaseURI(i.client.Environment.ResourceManagerEndpoint)
	i.client.DecorateAzureAutorest(&subscriptionsClient.Client)

	subscriptionList := []subscriptions.Subscription{}
	for _, subscriptionID := range subscriptionID {
		subscription, err := subscriptionsClient.Get(context.Background(), subscriptionID)
		if err != nil {
			panic(err)
		}
		subscriptionList = append(subscriptionList, subscription)
	}

	i.subscriptions = &subscriptionList
	return i
}

func (i *SubscriptionsIterator) SetConcurrency(concurrency int) *SubscriptionsIterator {
	i.concurrency = concurrency
	return i
}

func (i *SubscriptionsIterator) ForEach(logger *log.Entry, callback func(subscription subscriptions.Subscription, logger *log.Entry)) error {
	subscriptionList, err := i.ListSubscriptions()
	if err != nil {
		return err
	}

	for _, subscription := range subscriptionList {
		contextLogger := logger.WithFields(log.Fields{
			"subscriptionID":   *subscription.SubscriptionID,
			"subscriptionName": *subscription.DisplayName,
		})
		callback(subscription, contextLogger)
	}

	return nil
}

func (i *SubscriptionsIterator) ForEachAsync(logger *log.Entry, callback func(subscription subscriptions.Subscription, logger *log.Entry)) error {
	var panicList = []string{}
	panicLock := sync.Mutex{}
	wg := sizedwaitgroup.New(i.concurrency)

	subscriptionList, err := i.ListSubscriptions()
	if err != nil {
		return err
	}

	for _, subscription := range subscriptionList {
		wg.Add()

		go func(subscription subscriptions.Subscription) {
			defer wg.Done()
			contextLogger := logger.WithFields(log.Fields{
				"subscriptionID":   *subscription.SubscriptionID,
				"subscriptionName": *subscription.DisplayName,
			})

			finished := false
			defer func() {
				if !finished {
					if err := recover(); err != nil {
						panicLock.Lock()
						defer panicLock.Unlock()

						msg := ""
						switch v := err.(type) {
						case *log.Entry:
							msg = fmt.Sprintf("panic: %s\n%s", v.Message, debug.Stack())
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

func (i *SubscriptionsIterator) ListSubscriptions() ([]subscriptions.Subscription, error) {
	var list []subscriptions.Subscription

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
