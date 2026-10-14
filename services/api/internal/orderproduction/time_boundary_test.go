package orderproduction

import (
	"testing"
	"time"
)

func TestThirtyMinuteBoundary(t *testing.T) {
	pickup := time.Date(2026, 10, 14, 12, 0, 0, 0, time.FixedZone("CST", 8*3600))
	for _, tc := range []struct {
		name   string
		now    time.Time
		want   State
		cancel bool
	}{
		{"before", pickup.Add(-30*time.Minute - time.Microsecond), StateReserved, true},
		{"exact", pickup.Add(-30 * time.Minute), StatePreparing, false},
		{"after", pickup.Add(-30*time.Minute + time.Microsecond), StatePreparing, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := InitialState(tc.now, pickup)
			if err != nil || got != tc.want {
				t.Errorf("initial state=%s err=%v want=%s", got, err, tc.want)
			}
			d, err := Advance(StateReserved, tc.now, pickup)
			if err != nil || d.State != tc.want {
				t.Errorf("advance=%+v err=%v", d, err)
			}
			_, err = Transition(TransitionInput{Current: StateReserved, Trigger: TriggerUserCancel, ObservedAt: tc.now, PickupAt: pickup})
			if (err == nil) != tc.cancel {
				t.Errorf("cancellation allowed=%t want=%t", err == nil, tc.cancel)
			}
		})
	}
	for _, state := range []State{StateReadyForPickup, StateCompleted, StateRefunding, StateRefunded} {
		d, err := Advance(state, pickup.Add(time.Hour), pickup)
		if err != nil || d.State != state || d.Changed {
			t.Errorf("terminal/advanced state overwritten: %s %+v %v", state, d, err)
		}
	}
}
