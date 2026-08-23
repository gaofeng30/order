package quote

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMissingProviderDependenciesFailAsUnavailable(t *testing.T) {
	input := CreateInput{
		ContactName: "张三", PickupDate: "2026-08-24", PickupTime: "12:00",
		Items: []ItemInput{{ProductID: 8, Quantity: 1}},
	}
	providers := []*Provider{nil, newTestProvider(nil, time.Now), newTestProvider(openQuoteDriverDB(t, &quoteDriverState{}), nil)}
	for _, provider := range providers {
		if _, err := provider.Create(context.Background(), testWriteMeta(42, "attempt"), input); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("Create() dependency error = %v", err)
		}
	}
	readProviders := []*Provider{nil, newTestProvider(nil, time.Now)}
	for _, provider := range readProviders {
		if _, err := provider.Read(context.Background(), 42, 91); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("Read() dependency error = %v", err)
		}
	}
}
