package main

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/refund"
)

type refundRunnerProbe struct {
	calls int
	now   time.Time
	limit uint16
}

func (runner *refundRunnerProbe) RunDue(_ context.Context, now time.Time, limit uint16) (refund.RunResult, error) {
	runner.calls++
	runner.now = now
	runner.limit = limit
	return refund.RunResult{Claimed: 1, Observed: 1, Pending: 1}, nil
}

func TestRefundWorkerConsumesOneTickAtFixedLimit(t *testing.T) {
	runner := &refundRunnerProbe{}
	ticks := make(chan time.Time, 1)
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.FixedZone("local", 8*60*60))
	ticks <- now
	close(ticks)
	consumeRefundTicks(context.Background(), runner, ticks, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	if runner.calls != 1 || !runner.now.Equal(now.UTC()) || runner.limit != 100 {
		t.Fatalf("worker call = %d, %v, %d", runner.calls, runner.now, runner.limit)
	}
	if refundWorkerInterval != time.Second || refundWorkerLimit != 100 {
		t.Fatalf("worker schedule = %v/%d", refundWorkerInterval, refundWorkerLimit)
	}
}
