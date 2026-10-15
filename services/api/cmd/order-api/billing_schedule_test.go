package main

import (
	"context"
	"errors"
	"github.com/gaofeng30/order/services/api/internal/billing"
	"io"
	"log/slog"
	"testing"
	"time"
)

type billScheduleRunner struct {
	dates []time.Time
	fail  bool
}

func (r *billScheduleRunner) RunReconcile(_ context.Context, d time.Time, _ uint16) (billing.ReconcileResult, error) {
	r.dates = append(r.dates, d)
	if r.fail {
		r.fail = false
		return billing.ReconcileResult{}, errors.New("transient")
	}
	return billing.ReconcileResult{}, nil
}
func TestBillingDailyAndRetrySchedule(t *testing.T) {
	for _, fail := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		n := time.Date(2026, 10, 14, 9, 0, 0, 0, billingLocation)
		r := &billScheduleRunner{fail: fail}
		var delays []time.Duration
		after := func(d time.Duration) <-chan time.Time {
			delays = append(delays, d)
			ch := make(chan time.Time, 1)
			if len(delays) == 4 {
				cancel()
				return ch
			}
			n = n.Add(d)
			ch <- n
			return ch
		}
		runBillingSchedule(ctx, r, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return n }, after)
		if delays[0] != 65*time.Minute {
			t.Fatalf("premature bill fetch: %v", delays)
		}
		if fail {
			if delays[1] != 30*time.Minute || !r.dates[0].Equal(r.dates[1]) {
				t.Fatalf("failure date lost: %v %v", delays, r.dates)
			}
		} else if delays[1] != 24*time.Hour {
			t.Fatalf("success polled again: %v", delays)
		}
		cancel()
	}
}

type stuckBillRunner struct{ dates []time.Time }

func (r *stuckBillRunner) RunReconcile(_ context.Context, d time.Time, _ uint16) (billing.ReconcileResult, error) {
	r.dates = append(r.dates, d)
	if d.Day() == 13 {
		return billing.ReconcileResult{}, billing.ErrBillMismatch
	}
	return billing.ReconcileResult{}, nil
}
func TestBillingUnresolvedDayDoesNotBlockNextDay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	now := time.Date(2026, 10, 14, 10, 5, 0, 0, billingLocation)
	r := &stuckBillRunner{}
	steps := 0
	after := func(d time.Duration) <-chan time.Time {
		c := make(chan time.Time, 1)
		steps++
		if steps > 52 {
			cancel()
			return c
		}
		now = now.Add(d)
		c <- now
		return c
	}
	runBillingSchedule(ctx, r, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return now }, after)
	for _, d := range r.dates {
		if d.Day() == 14 {
			return
		}
	}
	t.Fatal("unresolved yesterday blocked the next bill")
}
