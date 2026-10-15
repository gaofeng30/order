package compensation

import (
	"context"
	"errors"
	"testing"
)

type memoryWakeStore struct {
	kind      string
	accepted  bool
	claimErr  error
	finished  []bool
	finishErr error
}

func (s *memoryWakeStore) Claim(context.Context, string) (Target, bool, error) {
	return Target{Kind: s.kind, ID: 42}, s.accepted, s.claimErr
}
func (s *memoryWakeStore) Finish(_ context.Context, _ string, success bool) error {
	s.finished = append(s.finished, success)
	return s.finishErr
}

func TestExecutorRequiresDurableClaimBeforeRunner(t *testing.T) {
	for _, s := range []*memoryWakeStore{{kind: "payment"}, {kind: "payment", claimErr: errors.New("offline")}, {kind: "unreviewed", accepted: true}} {
		calls := 0
		execute := NewExecutor(s, map[string]func(context.Context, uint64) error{"payment": func(context.Context, uint64) error { calls++; return nil }})
		if err := execute(context.Background(), "opaque"); err == nil {
			t.Fatal("accepted invalid claim")
		}
		if calls != 0 || len(s.finished) != 0 {
			t.Fatal("work occurred before authorized claim")
		}
	}
}

func TestExecutorPersistsOutcomeAndDoesNotReportSuccessOnFinalizeFailure(t *testing.T) {
	for _, tc := range []struct {
		name              string
		runErr, finishErr error
		success           bool
	}{{"success", nil, nil, true}, {"provider failure", errors.New("provider"), nil, false}, {"database finish failure", nil, errors.New("database"), true}} {
		t.Run(tc.name, func(t *testing.T) {
			s := &memoryWakeStore{kind: "refund", accepted: true, finishErr: tc.finishErr}
			calls := 0
			execute := NewExecutor(s, map[string]func(context.Context, uint64) error{"refund": func(context.Context, uint64) error { calls++; return tc.runErr }})
			err := execute(context.Background(), "opaque")
			if (err == nil) != (tc.runErr == nil && tc.finishErr == nil) {
				t.Fatalf("unexpected result %v", err)
			}
			if calls != 1 || len(s.finished) != 1 || s.finished[0] != tc.success {
				t.Fatalf("calls=%d outcomes=%v", calls, s.finished)
			}
		})
	}
}

func TestExecutorRunsOnlyTheClaimedRecord(t *testing.T) {
	store := &memoryWakeStore{kind: "subscription", accepted: true}
	var got uint64
	execute := NewExecutor(store, map[string]func(context.Context, uint64) error{"subscription": func(_ context.Context, id uint64) error { got = id; return nil }})
	if err := execute(context.Background(), "opaque"); err != nil {
		t.Fatal(err)
	}
	if got != 42 {
		t.Fatalf("executed record %d instead of 42", got)
	}
}
