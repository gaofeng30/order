package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gaofeng30/order/services/api/internal/billing"
)

const (
	billingWorkerInterval = time.Hour
	billingWorkerLimit    = 100
)

var billingLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

type billingRunner interface {
	RunReconcile(context.Context, time.Time, uint16) (billing.ReconcileResult, error)
}

func runBillingWorker(ctx context.Context, runner billingRunner, logger *slog.Logger) {
	ticker := time.NewTicker(billingWorkerInterval)
	defer ticker.Stop()
	consumeBillingTicks(ctx, runner, ticker.C, logger)
}

func consumeBillingTicks(ctx context.Context, runner billingRunner, ticks <-chan time.Time, logger *slog.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case now, open := <-ticks:
			if !open {
				return
			}
			localYesterday := now.In(billingLocation).AddDate(0, 0, -1)
			billDate := time.Date(localYesterday.Year(), localYesterday.Month(), localYesterday.Day(), 0, 0, 0, 0, time.UTC)
			result, err := runner.RunReconcile(ctx, billDate, billingWorkerLimit)
			if errors.Is(err, billing.ErrBillMismatch) {
				logger.WarnContext(ctx, "billing reconciliation mismatch",
					"provider_only", len(result.ProviderOnly), "system_only", len(result.SystemOnly))
				continue
			}
			if err != nil {
				logger.WarnContext(ctx, "billing reconciliation unavailable")
				continue
			}
			logger.InfoContext(ctx, "billing reconciliation completed", "matched", result.Matched)
		}
	}
}
