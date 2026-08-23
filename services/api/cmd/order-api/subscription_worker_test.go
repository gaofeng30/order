package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/subscription"
)

type subscriptionRunnerStub struct {
	times  []time.Time
	limits []uint16
	err    error
}

func (stub *subscriptionRunnerStub) RunDue(_ context.Context, now time.Time, limit uint16) (subscription.RunResult, error) {
	stub.times = append(stub.times, now)
	stub.limits = append(stub.limits, limit)
	return subscription.RunResult{Claimed: 1, Sent: 1}, stub.err
}

func TestConsumeSubscriptionTicksRunsOneBoundedBatchPerTick(t *testing.T) {
	runner := &subscriptionRunnerStub{}
	ticks := make(chan time.Time, 2)
	first := time.Date(2026, 8, 24, 3, 0, 0, 123456000, time.FixedZone("test", 8*60*60))
	second := first.Add(time.Second)
	ticks <- first
	ticks <- second
	close(ticks)
	consumeSubscriptionTicks(context.Background(), runner, ticks, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	if len(runner.times) != 2 || len(runner.limits) != 2 {
		t.Fatalf("calls = %d", len(runner.times))
	}
	if !runner.times[0].Equal(first.UTC()) || runner.times[0].Location() != time.UTC || runner.limits[0] != subscriptionWorkerLimit {
		t.Fatalf("first call = %s limit=%d", runner.times[0], runner.limits[0])
	}
}

func TestConsumeSubscriptionTicksContinuesAfterUnavailableBatch(t *testing.T) {
	runner := &subscriptionRunnerStub{err: errors.New("db unavailable")}
	ticks := make(chan time.Time, 2)
	ticks <- time.Date(2026, 8, 24, 3, 0, 0, 0, time.UTC)
	ticks <- time.Date(2026, 8, 24, 3, 0, 1, 0, time.UTC)
	close(ticks)
	consumeSubscriptionTicks(context.Background(), runner, ticks, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	if len(runner.times) != 2 {
		t.Fatalf("calls = %d, want 2", len(runner.times))
	}
}
