package azure

import (
	"sync"
)

type (
	InterfaceIterator struct {
		client *Client
		list   []interface{}
	}
)

func NewInterfaceIterator() *InterfaceIterator {
	iterator := InterfaceIterator{}
	return &iterator
}

func (i *InterfaceIterator) SetList(list ...interface{}) {
	i.list = list
}

func (i *InterfaceIterator) GetList() []interface{} {
	return i.list
}

func (i *InterfaceIterator) ForEach(callback func(object interface{})) error {
	for _, subscription := range i.list {
		callback(subscription)
	}
	return nil
}

func (i *InterfaceIterator) ForEachAsync(callback func(object interface{})) error {
	wg := sync.WaitGroup{}
	for _, object := range i.list {
		wg.Add(1)

		go func(object interface{}) {
			defer wg.Done()
			callback(object)
		}(object)
	}

	wg.Wait()
	return nil
}
