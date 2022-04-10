package azure

import (
	"context"

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
