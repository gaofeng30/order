package subscription

import (
	"context"
	"testing"
	"time"
)

type singleNotificationStore struct {
	retryContractStore
	t                 *testing.T
	claimedID, sentID uint64
}

func (s *singleNotificationStore) claimOne(_ context.Context, id uint64, _ time.Time, _ time.Time, _ [16]byte) ([]claimedDelivery, error) {
	s.claimedID = id
	return []claimedDelivery{{Delivery: Delivery{OutboxID: id}}}, nil
}
func (s *singleNotificationStore) claimDue(context.Context, time.Time, time.Time, uint16, [16]byte, time.Duration) ([]claimedDelivery, error) {
	s.t.Fatal("targeted notification scanned other intents")
	return nil, nil
}
func (s *singleNotificationStore) markSent(_ context.Context, d claimedDelivery, _ SendResult, _ time.Time) error {
	s.sentID = d.OutboxID
	return nil
}
func TestRunOneSendsOnlyTheClaimedNotification(t *testing.T) {
	store := &singleNotificationStore{t: t}
	service := &SharedService{store: store, provider: contractProvider{}, now: time.Now, owner: func() ([16]byte, error) { return [16]byte{1}, nil }}
	result, err := service.RunOne(context.Background(), time.Now(), 42)
	if err != nil || result.Claimed != 1 || result.Sent != 1 || store.claimedID != 42 || store.sentID != 42 {
		t.Fatalf("result=%+v err=%v claim=%d sent=%d", result, err, store.claimedID, store.sentID)
	}
}
