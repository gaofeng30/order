package orderadvance

import (
	"errors"
	"testing"
	"time"
)

func TestProductionDueBoundaryIsExactThirtyMinutes(t *testing.T) {
	pickupAt := time.Date(2026, 8, 25, 4, 0, 0, 0, time.UTC)
	before, err := productionDecision(pickupAt.Add(-30*time.Minute-time.Microsecond), pickupAt)
	if err != nil || before {
		t.Fatalf("before boundary = %v/%v", before, err)
	}
	at, err := productionDecision(pickupAt.Add(-30*time.Minute), pickupAt)
	if err != nil || !at {
		t.Fatalf("at boundary = %v/%v", at, err)
	}
	late, err := productionDecision(pickupAt.Add(time.Hour), pickupAt)
	if err != nil || !late {
		t.Fatalf("late compensation = %v/%v", late, err)
	}
}

func TestRunProductionDueFailsClosedForInvalidRuntime(t *testing.T) {
	service := New(nil)
	if _, err := service.RunProductionDue(nil, time.Now(), 10); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("nil context error = %v", err)
	}
	if _, err := service.RunProductionDue(t.Context(), time.Now(), 10); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil database error = %v", err)
	}
}
