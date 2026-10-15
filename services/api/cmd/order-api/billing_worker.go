package main

import (
	"context"
	"errors"
	"github.com/gaofeng30/order/services/api/internal/billing"
	"log/slog"
	"time"
)

const billingWorkerLimit = 100

var billingLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

type billingRunner interface {
	RunReconcile(context.Context, time.Time, uint16) (billing.ReconcileResult, error)
}

func runBillingWorker(ctx context.Context, runner billingRunner, logger *slog.Logger) {
	runBillingSchedule(ctx, runner, logger, time.Now, time.After)
}

// Bills become available after 10:00 Beijing time. A successful day has no more
// timer work until the next bill; only an actual failure retries after 30 minutes.
func runBillingSchedule(ctx context.Context, runner billingRunner, logger *slog.Logger, now func() time.Time, after func(time.Duration) <-chan time.Time) {
	localYesterday := now().In(billingLocation).AddDate(0, 0, -1)
	date := time.Date(localYesterday.Year(), localYesterday.Month(), localYesterday.Day(), 0, 0, 0, 0, time.UTC)
	available := func(d time.Time) time.Time {
		return time.Date(d.Year(), d.Month(), d.Day()+1, 10, 5, 0, 0, billingLocation)
	}
	pending := map[time.Time]time.Time{}
	for {
		selected, next := date, available(date)
		for failedDate, retryAt := range pending {
			if retryAt.Before(next) {
				selected, next = failedDate, retryAt
			}
		}
		delay := next.Sub(now())
		if delay < 0 {
			delay = 0
		}
		select {
		case <-ctx.Done():
			return
		case <-after(delay):
		}
		if ctx.Err() != nil {
			return
		}
		if selected.Equal(date) {
			date = date.AddDate(0, 0, 1)
		}
		result, err := runner.RunReconcile(ctx, selected, billingWorkerLimit)
		if err != nil {
			if errors.Is(err, billing.ErrBillMismatch) {
				logger.WarnContext(ctx, "billing reconciliation mismatch", "provider_only", len(result.ProviderOnly), "system_only", len(result.SystemOnly))
			} else {
				logger.WarnContext(ctx, "billing reconciliation unavailable")
			}
			pending[selected] = now().Add(30 * time.Minute)
			continue
		}
		logger.InfoContext(ctx, "billing reconciliation completed", "matched", result.Matched)
		delete(pending, selected)
	}
}
