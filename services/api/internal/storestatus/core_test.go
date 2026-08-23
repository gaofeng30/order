package storestatus

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gaofeng30/order/services/api/internal/storefront"
)

func TestApplyRejectsInvalidStatusBeforeDependencies(t *testing.T) {
	core := New(nil, nil, nil)

	result, err := core.Apply(context.Background(), Command{
		UserID: 41, DesiredStatus: storefront.BusinessStatus("unexpected"),
		IdempotencyKey: "status-command-invalid", RequestID: "request-invalid",
	})

	if !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("Apply() error = %v, want invalid command", err)
	}
	if result != (Result{}) {
		t.Fatalf("Apply() result = %#v, want zero", result)
	}
}

func TestApplyRejectsMalformedCommandBeforeDependencies(t *testing.T) {
	tests := []struct {
		name    string
		command Command
	}{
		{name: "zero user", command: Command{DesiredStatus: storefront.BusinessOpen, IdempotencyKey: "key", RequestID: "request"}},
		{name: "empty key", command: Command{UserID: 1, DesiredStatus: storefront.BusinessOpen, RequestID: "request"}},
		{name: "untrimmed key", command: Command{UserID: 1, DesiredStatus: storefront.BusinessOpen, IdempotencyKey: " key", RequestID: "request"}},
		{name: "invalid key utf8", command: Command{UserID: 1, DesiredStatus: storefront.BusinessOpen, IdempotencyKey: string([]byte{0xff}), RequestID: "request"}},
		{name: "empty request", command: Command{UserID: 1, DesiredStatus: storefront.BusinessOpen, IdempotencyKey: "key"}},
		{name: "untrimmed request", command: Command{UserID: 1, DesiredStatus: storefront.BusinessOpen, IdempotencyKey: "key", RequestID: "request "}},
		{name: "oversized request", command: Command{UserID: 1, DesiredStatus: storefront.BusinessOpen, IdempotencyKey: "key", RequestID: strings.Repeat("r", 65)}},
		{name: "invalid request utf8", command: Command{UserID: 1, DesiredStatus: storefront.BusinessOpen, IdempotencyKey: "key", RequestID: string([]byte{0xff})}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := New(nil, nil, nil).Apply(context.Background(), test.command)
			if !errors.Is(err, ErrInvalidCommand) || result != (Result{}) {
				t.Fatalf("Apply() = %#v, %v; want zero invalid command", result, err)
			}
		})
	}
}
