package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/gaofeng30/order/services/api/internal/paymentorder"
)

const (
	paymentWorkerInterval = time.Second
	paymentWorkerLimit    = 100
)

type paymentRunner interface {
	RunDue(context.Context, time.Time, uint16) (paymentorder.RunResult, error)
}

func runPaymentWorker(ctx context.Context, runner paymentRunner, logger *slog.Logger) {
	ticker := time.NewTicker(paymentWorkerInterval)
	defer ticker.Stop()
	consumePaymentTicks(ctx, runner, ticker.C, logger)
}

func consumePaymentTicks(ctx context.Context, runner paymentRunner, ticks <-chan time.Time, logger *slog.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case now, open := <-ticks:
			if !open {
				return
			}
			result, err := runner.RunDue(ctx, now.UTC(), paymentWorkerLimit)
			if err != nil {
				logger.WarnContext(ctx, "payment worker unavailable")
				continue
			}
			if result.Queried > 0 || result.Observed > 0 || result.Materialized > 0 || result.Pending > 0 {
				logger.InfoContext(ctx, "payment worker batch completed",
					"queried", result.Queried,
					"observed", result.Observed,
					"materialized", result.Materialized,
					"pending", result.Pending,
				)
			}
		}
	}
}
