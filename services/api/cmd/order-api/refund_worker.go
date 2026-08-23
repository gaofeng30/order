package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/gaofeng30/order/services/api/internal/refund"
)

const (
	refundWorkerInterval = time.Second
	refundWorkerLimit    = 100
)

type refundRunner interface {
	RunDue(context.Context, time.Time, uint16) (refund.RunResult, error)
}

func runRefundWorker(ctx context.Context, runner refundRunner, logger *slog.Logger) {
	ticker := time.NewTicker(refundWorkerInterval)
	defer ticker.Stop()
	consumeRefundTicks(ctx, runner, ticker.C, logger)
}

func consumeRefundTicks(ctx context.Context, runner refundRunner, ticks <-chan time.Time, logger *slog.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case now, open := <-ticks:
			if !open {
				return
			}
			result, err := runner.RunDue(ctx, now.UTC(), refundWorkerLimit)
			if err != nil {
				logger.WarnContext(ctx, "refund worker unavailable")
				continue
			}
			if result.Claimed > 0 || result.Observed > 0 || result.Applied > 0 || result.Pending > 0 {
				logger.InfoContext(ctx, "refund worker batch completed",
					"claimed", result.Claimed,
					"observed", result.Observed,
					"applied", result.Applied,
					"pending", result.Pending,
				)
			}
		}
	}
}
