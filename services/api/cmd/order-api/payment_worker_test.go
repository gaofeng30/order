package main

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/paymentorder"
)

type paymentRunnerProbe struct {
	calls int
	now   time.Time
	limit uint16
}

func (runner *paymentRunnerProbe) RunDue(_ context.Context, now time.Time, limit uint16) (paymentorder.RunResult, error) {
	runner.calls++
	runner.now = now
	runner.limit = limit
	return paymentorder.RunResult{}, nil
}

func TestPaymentWorkerConsumesOneTickAndStopsWithContext(t *testing.T) {
	runner := &paymentRunnerProbe{}
	ticks := make(chan time.Time, 1)
	now := time.Date(2026, 8, 24, 4, 40, 0, 0, time.FixedZone("local", 8*60*60))
	ticks <- now
	close(ticks)
	consumePaymentTicks(context.Background(), runner, ticks, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	if runner.calls != 1 || !runner.now.Equal(now.UTC()) || runner.limit != paymentWorkerLimit {
		t.Fatalf("worker call = %d, %v, %d", runner.calls, runner.now, runner.limit)
	}
}
