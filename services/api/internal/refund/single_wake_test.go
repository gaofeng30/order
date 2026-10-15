package refund

import (
	"context"
	"testing"
	"time"
)

type targetedRefundStore struct {
	refundStore
	t   *testing.T
	ids []uint64
}

func (s *targetedRefundStore) claimCreate(_ context.Context, id uint64, _ [16]byte, _ time.Time, _ time.Duration) (createClaim, bool, error) {
	s.ids = append(s.ids, id)
	return createClaim{}, false, nil
}
func (s *targetedRefundStore) claimOneQuery(_ context.Context, id uint64, _ [16]byte, _ time.Time, _ time.Time, _ time.Duration) (queryClaim, bool, error) {
	s.ids = append(s.ids, id)
	return queryClaim{}, false, nil
}
func (s *targetedRefundStore) listDueIDs(context.Context, time.Time, time.Time, uint16) ([]uint64, error) {
	s.t.Fatal("targeted wake scanned unrelated refunds")
	return nil, nil
}

type unusedRefundProvider struct{ Provider }

func TestRunOneKeepsBothClaimsOnTheNamedRefund(t *testing.T) {
	store := &targetedRefundStore{t: t}
	service := New(nil, &unusedRefundProvider{}, Config{Environment: "test", RefundNotifyURL: "https://example.invalid/notify"})
	service.store = store
	if _, err := service.RunOne(context.Background(), time.Now(), 42); err != nil {
		t.Fatal(err)
	}
	if len(store.ids) != 2 || store.ids[0] != 42 || store.ids[1] != 42 {
		t.Fatalf("claimed IDs=%v", store.ids)
	}
}
