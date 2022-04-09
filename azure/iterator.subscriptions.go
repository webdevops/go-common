package azure

import (
	"context"
	"sync"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
)

type (
	IteratorSubscriptions struct {
		client        *Client
		subscriptions *[]subscriptions.Subscription
	}
)

func NewIteratorSubscriptions(client *Client) *IteratorSubscriptions {
	iterator := IteratorSubscriptions{}
	iterator.client = client
	return &iterator
}

func (i *IteratorSubscriptions) SetFixedSubscriptions(subscriptionList []subscriptions.Subscription) {
	i.subscriptions = &subscriptionList
}

func (i *IteratorSubscriptions) ForEach(callback func(subscription subscriptions.Subscription)) error {
	subscriptionList, err := i.listSubscriptions()
	if err != nil {
		return err
	}

	for _, subscription := range subscriptionList {
		callback(subscription)
	}

	return nil
}

func (i *IteratorSubscriptions) ForEachAsync(callback func(subscription subscriptions.Subscription)) error {
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

func (i *IteratorSubscriptions) listSubscriptions() ([]subscriptions.Subscription, error) {
	var list []subscriptions.Subscription
	
	if i.subscriptions != nil {
		list = *i.subscriptions
	} else {
		if result, err := i.client.ListCachedSubscriptions(context.Background()); err != nil {
			list = *result
		} else {
			return list, err
		}
	}

	return list, nil
}
