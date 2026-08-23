package paymentorder

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/paymentobservation"
)

func TestEffectiveDeadlineUsesEarlierQuoteWindowOrPickup(t *testing.T) {
	created := time.Date(2026, 8, 25, 2, 0, 0, 0, time.UTC)
	if got := EffectiveDeadline(created, created.Add(30*time.Minute)); !got.Equal(created.Add(10 * time.Minute)) {
		t.Fatalf("EffectiveDeadline(quote window) = %s", got)
	}
	if got := EffectiveDeadline(created, created.Add(4*time.Minute)); !got.Equal(created.Add(4 * time.Minute)) {
		t.Fatalf("EffectiveDeadline(pickup) = %s", got)
	}
}

func TestQueryResultCannotReleaseAnotherOrNewerLease(t *testing.T) {
	owner := [16]byte{1}
	claim := &queryClaim{owner: owner, version: 7}
	record := prepaymentRecord{leaseKind: sql.NullString{String: "QUERY", Valid: true}, leaseOwner: owner[:], recordVersion: 7}
	if !queryClaimOwns(record, claim) {
		t.Fatal("exact owner/version did not own query lease")
	}
	record.recordVersion++
	if queryClaimOwns(record, claim) {
		t.Fatal("stale version owned newer query lease")
	}
	record.recordVersion = claim.version
	record.leaseOwner = append([]byte(nil), owner[:]...)
	record.leaseOwner[0]++
	if queryClaimOwns(record, claim) {
		t.Fatal("different owner owned query lease")
	}
}

func TestCreateEligibilityHasExactOneMinuteBoundary(t *testing.T) {
	now := time.Date(2026, 8, 25, 2, 0, 0, 0, time.UTC)
	if err := RequireCreateWindow(now, now.Add(time.Minute)); err != nil {
		t.Fatalf("RequireCreateWindow(exact minute) = %v", err)
	}
	if err := RequireCreateWindow(now, now.Add(time.Minute-time.Microsecond)); !errors.Is(err, ErrQuoteUnavailable) {
		t.Fatalf("RequireCreateWindow(below minute) = %v", err)
	}
}

func TestMaterializationModeUsesTrustedSuccessTimeOnly(t *testing.T) {
	deadline := time.Date(2026, 8, 25, 2, 10, 0, 0, time.UTC)
	accepted := paymentobservation.Observation{Validation: paymentobservation.ValidationAccepted, State: paymentobservation.StatePaid, SuccessTime: deadline.Add(-time.Microsecond)}
	if got := DecideMaterializationMode(accepted, deadline); got != MaterializationAuto {
		t.Fatalf("before deadline mode = %s", got)
	}
	accepted.SuccessTime = deadline
	if got := DecideMaterializationMode(accepted, deadline); got != MaterializationDelayedManual {
		t.Fatalf("exact deadline mode = %s", got)
	}
	accepted.SuccessTime = deadline.Add(time.Microsecond)
	if got := DecideMaterializationMode(accepted, deadline); got != MaterializationDelayedManual {
		t.Fatalf("after deadline mode = %s", got)
	}
	accepted.SuccessTime = deadline.Add(-time.Minute)
	accepted.Validation = paymentobservation.ValidationRejectedMismatch
	if got := DecideMaterializationMode(accepted, deadline); got != MaterializationDelayedManual {
		t.Fatalf("mismatch mode = %s", got)
	}
}

func TestProviderStateNeverRegressesAcrossOutOfOrderFacts(t *testing.T) {
	for _, test := range []struct {
		current  ProviderState
		observed paymentobservation.State
		want     ProviderState
	}{
		{current: ProviderPaid, observed: paymentobservation.StateNotPaid, want: ProviderPaid},
		{current: ProviderPaid, observed: paymentobservation.StateClosed, want: ProviderPaid},
		{current: ProviderClosed, observed: paymentobservation.StateNotPaid, want: ProviderClosed},
		{current: ProviderClosed, observed: paymentobservation.StatePaid, want: ProviderPaid},
		{current: ProviderPaymentRequested, observed: paymentobservation.StatePaid, want: ProviderPaid},
	} {
		if got := advanceProviderState(test.current, test.observed); got != test.want {
			t.Fatalf("advanceProviderState(%s,%s) = %s, want %s", test.current, test.observed, got, test.want)
		}
	}
}
