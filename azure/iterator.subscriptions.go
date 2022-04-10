package azure

import (
	"context"
	"sync"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
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

func (i *SubscriptionsIterator) ForEach(callback func(subscription subscriptions.Subscription)) error {
	subscriptionList, err := i.listSubscriptions()
	if err != nil {
		return err
	}

	for _, subscription := range subscriptionList {
		callback(subscription)
	}

	return nil
}

func (i *SubscriptionsIterator) ForEachAsync(callback func(subscription subscriptions.Subscription)) error {
	wg := sync.WaitGroup{}

	subscriptionList, err := i.listSubscriptions()
	if err != nil {
		return err
	}

	for _, subscription := range subscriptionList {
		wg.Add(1)

		go func(subscription subscriptions.Subscription) {
			defer wg.Done()
			callback(subscription)
		}(subscription)
	}

	wg.Wait()
	return nil
}

func (i *SubscriptionsIterator) listSubscriptions() ([]subscriptions.Subscription, error) {
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
