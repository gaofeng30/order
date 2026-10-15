package main

import (
	"context"
	"time"

	"github.com/gaofeng30/order/services/api/internal/compensation"
	"github.com/gaofeng30/order/services/api/internal/paymentorder"
	"github.com/gaofeng30/order/services/api/internal/refund"
	"github.com/gaofeng30/order/services/api/internal/subscription"
	"github.com/gaofeng30/order/services/api/sharedpg"
)

func newCompensationRoutes(client sharedpg.Executor,
	payments interface {
		RunOne(context.Context, time.Time, uint64) (paymentorder.RunResult, error)
	},
	refunds interface {
		RunOne(context.Context, time.Time, uint64) (refund.RunResult, error)
	},
	notifications interface {
		RunOne(context.Context, time.Time, uint64) (subscription.RunResult, error)
	},
) *compensation.Handler {
	return compensation.NewHandler(compensation.NewExecutor(compensation.NewSharedPGStore(client), map[string]func(context.Context, uint64) error{
		"payment": func(ctx context.Context, id uint64) error {
			_, err := payments.RunOne(ctx, time.Now().UTC(), id)
			return err
		},
		"refund": func(ctx context.Context, id uint64) error {
			_, err := refunds.RunOne(ctx, time.Now().UTC(), id)
			return err
		},
		"subscription": func(ctx context.Context, id uint64) error {
			_, err := notifications.RunOne(ctx, time.Now().UTC(), id)
			return err
		},
	}))
}
