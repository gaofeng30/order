package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/paymentorder"
	"github.com/gaofeng30/order/services/api/internal/refund"
	"github.com/gaofeng30/order/services/api/internal/subscription"
)

type wakeRPC struct {
	kind     string
	finished bool
}

func (r *wakeRPC) CallRPC(_ context.Context, name string, args map[string]any, out any) error {
	reply := `{"outcome":"CLAIMED","kind":"` + r.kind + `","id":"42"}`
	if name == "order_compensation_finish_v1" {
		r.finished = args["p_success"] == true
		reply = `{"outcome":"FINISHED"}`
	}
	return json.Unmarshal([]byte(reply), out)
}

type wakePayment struct{ id uint64 }

func (r *wakePayment) RunOne(_ context.Context, now time.Time, id uint64) (paymentorder.RunResult, error) {
	if now.IsZero() {
		panic("no server time")
	}
	r.id = id
	return paymentorder.RunResult{}, nil
}

type wakeRefund struct{ id uint64 }

func (r *wakeRefund) RunOne(_ context.Context, now time.Time, id uint64) (refund.RunResult, error) {
	if now.IsZero() {
		panic("no server time")
	}
	r.id = id
	return refund.RunResult{}, nil
}

type wakeSubscription struct{ id uint64 }

func (r *wakeSubscription) RunOne(_ context.Context, now time.Time, id uint64) (subscription.RunResult, error) {
	if now.IsZero() {
		panic("no server time")
	}
	r.id = id
	return subscription.RunResult{}, nil
}
func TestProductionWakeRoutesOnlyTheClaimedRecord(t *testing.T) {
	for _, kind := range []string{"payment", "refund", "subscription"} {
		t.Run(kind, func(t *testing.T) {
			rpc := &wakeRPC{kind: kind}
			p, r, s := &wakePayment{}, &wakeRefund{}, &wakeSubscription{}
			handler := newCompensationRoutes(rpc, p, r, s)
			req := httptest.NewRequest("POST", "/internal/compensation/wake", nil)
			req.Header.Set("Authorization", "Bearer "+strings.Repeat("ab", 32))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != 204 || !rpc.finished {
				t.Fatalf("status=%d finished=%v", w.Code, rpc.finished)
			}
			for name, id := range map[string]uint64{"payment": p.id, "refund": r.id, "subscription": s.id} {
				want := uint64(0)
				if name == kind {
					want = 42
				}
				if id != want {
					t.Fatalf("%s ran id=%d want=%d", name, id, want)
				}
			}
		})
	}
}
