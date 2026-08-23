package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/gaofeng30/order/services/api/internal/orderadvance"
)

const (
	orderProductionWorkerInterval = time.Second
	orderProductionWorkerLimit    = 100
)

type orderProductionRunner interface {
	RunProductionDue(context.Context, time.Time, uint16) (orderadvance.RunResult, error)
}

func runOrderProductionWorker(ctx context.Context, runner orderProductionRunner, logger *slog.Logger) {
	ticker := time.NewTicker(orderProductionWorkerInterval)
	defer ticker.Stop()
	consumeOrderProductionTicks(ctx, runner, ticker.C, logger)
}

func consumeOrderProductionTicks(ctx context.Context, runner orderProductionRunner, ticks <-chan time.Time, logger *slog.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case now, open := <-ticks:
			if !open {
				return
			}
			result, err := runner.RunProductionDue(ctx, now.UTC(), orderProductionWorkerLimit)
			if err != nil {
				logger.WarnContext(ctx, "order production worker unavailable")
				continue
			}
			if result.Advanced > 0 {
				logger.InfoContext(ctx, "order production worker batch completed", "scanned", result.Scanned, "advanced", result.Advanced)
			}
		}
	}
}
