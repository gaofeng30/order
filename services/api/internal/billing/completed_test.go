package billing

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type completedProbe struct {
	done  bool
	calls int
}

func (p *completedProbe) CallRPC(_ context.Context, name string, _ map[string]any, out any) error {
	p.calls++
	if name != "order_billing_completed_v1" {
		return ErrUnavailable
	}
	raw, _ := json.Marshal(map[string]any{"outcome": "SUCCEEDED", "completed": p.done, "matched": "3", "digest": "0101010101010101010101010101010101010101010101010101010101010101"})
	return json.Unmarshal(raw, out)
}

type noDownload struct{ calls int }

func (p *noDownload) DownloadTransactionBill(context.Context, time.Time) (TransactionBill, error) {
	p.calls++
	return TransactionBill{}, ErrBillUnavailable
}
func TestCompletedBillSurvivesNewServiceWithoutDownload(t *testing.T) {
	rpc := &completedProbe{done: true}
	provider := &noDownload{}
	cfg := Config{Environment: "uat", AppID: "wx15e136b256d4dc5a", MerchantID: "lab", PaymentNotifyURL: "https://example.invalid/pay", RefundNotifyURL: "https://example.invalid/refund"}
	date := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 2; i++ {
		s := NewSharedPG(rpc, provider, cfg)
		r, e := s.RunReconcile(context.Background(), date, 100)
		if e != nil || r.Matched != 3 || !r.BillDate.Equal(date) {
			t.Fatalf("result=%+v err=%v", r, e)
		}
	}
	if provider.calls != 0 {
		t.Fatal("completed bill redownloaded")
	}
	rpc.done = false
	_, e := NewSharedPG(rpc, provider, cfg).RunReconcile(context.Background(), date, 100)
	if e != ErrBillUnavailable || provider.calls != 1 {
		t.Fatal("unfinished bill suppressed")
	}
}
