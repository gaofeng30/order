package subscription

import (
	"context"
	"strconv"
	"sync"
)

// FakeProvider is deterministic and intended only for local tests and local runtime injection.
type FakeProvider struct {
	mu         sync.Mutex
	results    []fakeSend
	deliveries []Delivery
}

type fakeSend struct {
	result SendResult
	err    error
}

func NewFakeProvider() *FakeProvider { return &FakeProvider{} }

func (provider *FakeProvider) Queue(result SendResult, err error) {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	provider.results = append(provider.results, fakeSend{result: result, err: err})
}

func (provider *FakeProvider) SendSubscription(_ context.Context, delivery Delivery) (SendResult, error) {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	provider.deliveries = append(provider.deliveries, delivery)
	if len(provider.results) > 0 {
		next := provider.results[0]
		provider.results = provider.results[1:]
		return next.result, next.err
	}
	return SendResult{ProviderMessageID: "fake-message-" + strconv.FormatUint(delivery.OutboxID, 10)}, nil
}

func (provider *FakeProvider) Deliveries() []Delivery {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	return append([]Delivery(nil), provider.deliveries...)
}
