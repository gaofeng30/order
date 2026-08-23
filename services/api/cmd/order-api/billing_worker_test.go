package main

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/billing"
)

type billingRunnerProbe struct {
	dates  []time.Time
	limits []uint16
}

func (probe *billingRunnerProbe) RunReconcile(_ context.Context, date time.Time, limit uint16) (billing.ReconcileResult, error) {
	probe.dates = append(probe.dates, date)
	probe.limits = append(probe.limits, limit)
	return billing.ReconcileResult{BillDate: date}, nil
}

func TestBillingWorkerReconcilesPreviousShanghaiBusinessDate(t *testing.T) {
	probe := &billingRunnerProbe{}
	ticks := make(chan time.Time, 1)
	ticks <- time.Date(2026, 8, 25, 2, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	close(ticks)
	consumeBillingTicks(context.Background(), probe, ticks, slog.Default())

	want := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	if len(probe.dates) != 1 || !probe.dates[0].Equal(want) || probe.dates[0].Location() != time.UTC ||
		len(probe.limits) != 1 || probe.limits[0] != billingWorkerLimit {
		t.Fatalf("billing worker calls = %#v/%#v", probe.dates, probe.limits)
	}
}
