package subscription

import (
	"context"
	"testing"
	"time"
)

type retryContractStore struct {
	sharedSubscriptionStore
	claimed         bool
	next            time.Time
	sent, permanent int
}

func (s *retryContractStore) claimDue(context.Context, time.Time, time.Time, uint16, [16]byte, time.Duration) ([]claimedDelivery, error) {
	if s.claimed {
		return nil, nil
	}
	s.claimed = true
	return []claimedDelivery{{Delivery: Delivery{OutboxID: 1}}}, nil
}
func (s *retryContractStore) markTemporaryFailure(_ context.Context, _ claimedDelivery, _ string, _ time.Time, next time.Time) error {
	s.next = next
	return nil
}
func (s *retryContractStore) markSent(context.Context, claimedDelivery, SendResult, time.Time) error {
	s.sent++
	return nil
}
func (s *retryContractStore) markPermanentFailure(context.Context, claimedDelivery, string, time.Time) error {
	s.permanent++
	return nil
}

type contractProvider struct{ err error }

func (p contractProvider) SendSubscription(context.Context, Delivery) (SendResult, error) {
	return SendResult{ProviderMessageID: "receipt"}, p.err
}
func TestSharedNotificationRetryContract(t *testing.T) {
	now := time.Date(2026, 10, 15, 4, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name            string
		err             error
		cutoff          time.Time
		delay           time.Duration
		sent, permanent int
	}{
		{"temporary failure", &SendError{Code: "NETWORK_UNAVAILABLE"}, now, 30 * time.Minute, 0, 0},
		{"future cutoff cannot retry in same run", &SendError{Code: "NETWORK_UNAVAILABLE"}, now.Add(time.Hour), 90 * time.Minute, 0, 0},
		{"success stops", nil, now, 0, 1, 0},
		{"permanent failure stops", &SendError{Code: "INVALID_TEMPLATE", Permanent: true}, now, 0, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &retryContractStore{}
			service := &SharedService{store: store, provider: contractProvider{tc.err}, now: func() time.Time { return now }, owner: func() ([16]byte, error) { return [16]byte{1}, nil }}
			result, err := service.RunDue(context.Background(), tc.cutoff, 10)
			if err != nil || result.Claimed != 1 {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if tc.delay > 0 && !store.next.Equal(now.Add(tc.delay)) {
				t.Fatalf("retry at %v, want %v", store.next, now.Add(tc.delay))
			}
			if tc.delay == 0 && !store.next.IsZero() {
				t.Fatal("terminal outcome scheduled a retry")
			}
			if store.sent != tc.sent || store.permanent != tc.permanent {
				t.Fatalf("sent=%d permanent=%d", store.sent, store.permanent)
			}
		})
	}
}
