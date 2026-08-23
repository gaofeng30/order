package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/gaofeng30/order/services/api/internal/subscription"
)

const (
	subscriptionWorkerInterval = time.Second
	subscriptionWorkerLimit    = 100
)

type subscriptionRunner interface {
	RunDue(context.Context, time.Time, uint16) (subscription.RunResult, error)
}

func runSubscriptionWorker(ctx context.Context, runner subscriptionRunner, logger *slog.Logger) {
	ticker := time.NewTicker(subscriptionWorkerInterval)
	defer ticker.Stop()
	consumeSubscriptionTicks(ctx, runner, ticker.C, logger)
}

func consumeSubscriptionTicks(ctx context.Context, runner subscriptionRunner, ticks <-chan time.Time, logger *slog.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case now, open := <-ticks:
			if !open {
				return
			}
			result, err := runner.RunDue(ctx, now.UTC(), subscriptionWorkerLimit)
			if err != nil {
				logger.WarnContext(ctx, "subscription worker unavailable")
				continue
			}
			if result.Claimed > 0 {
				logger.InfoContext(ctx, "subscription worker batch completed",
					"claimed", result.Claimed,
					"sent", result.Sent,
					"temporary_failed", result.TemporaryFailed,
					"permanent_failed", result.PermanentFailed,
				)
			}
		}
	}
}
