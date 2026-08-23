package main

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/orderadvance"
)

type orderProductionRunnerProbe struct {
	calls int
	now   time.Time
	limit uint16
}

func (runner *orderProductionRunnerProbe) RunProductionDue(_ context.Context, now time.Time, limit uint16) (orderadvance.RunResult, error) {
	runner.calls++
	runner.now = now
	runner.limit = limit
	return orderadvance.RunResult{}, nil
}

func TestOrderProductionWorkerConsumesOneTick(t *testing.T) {
	runner := &orderProductionRunnerProbe{}
	ticks := make(chan time.Time, 1)
	now := time.Date(2026, 8, 24, 4, 50, 0, 0, time.FixedZone("local", 8*60*60))
	ticks <- now
	close(ticks)
	consumeOrderProductionTicks(context.Background(), runner, ticks, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	if runner.calls != 1 || !runner.now.Equal(now.UTC()) || runner.limit != orderProductionWorkerLimit {
		t.Fatalf("worker call = %d, %v, %d", runner.calls, runner.now, runner.limit)
	}
}
