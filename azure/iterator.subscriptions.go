package azure

import (
	"context"
	"sync"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	log "github.com/sirupsen/logrus"
)

type (
	SubscriptionsIterator struct {
		client        *Client
		subscriptions *[]subscriptions.Subscription
	}
)

func NewSubscriptionIterator(client *Client, subscriptionID ...string) *SubscriptionsIterator {
	iterator := SubscriptionsIterator{}
	iterator.client = client
	if len(subscriptionID) >= 1 {
		iterator.SetSubscriptions(subscriptionID...)
	}
	return &iterator
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
	wg := sync.WaitGroup{}

	subscriptionList, err := i.ListSubscriptions()
	if err != nil {
		return err
	}

	for _, subscription := range subscriptionList {
		wg.Add(1)

		go func(subscription subscriptions.Subscription) {
			defer wg.Done()
			contextLogger := logger.WithFields(log.Fields{
				"subscriptionID":   *subscription.SubscriptionID,
				"subscriptionName": *subscription.DisplayName,
			})
			callback(subscription, contextLogger)
		}(subscription)
	}

	wg.Wait()
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
