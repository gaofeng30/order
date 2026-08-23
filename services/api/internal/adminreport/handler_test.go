package adminreport

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeReport struct {
	called        bool
	pendingResult PendingResult
}

func (*fakeReport) SearchOrders(context.Context, uint64, OrderFilter) ([]Order, uint64, error) {
	return nil, 0, nil
}
func (*fakeReport) Stats(context.Context, uint64, time.Time) (Stats, error) { return Stats{}, nil }
func (*fakeReport) Payments(context.Context, uint64, BillingRange, PageQuery) ([]Payment, uint64, error) {
	return nil, 0, nil
}
func (*fakeReport) Refunds(context.Context, uint64, BillingRange, PageQuery) ([]Refund, uint64, error) {
	return nil, 0, nil
}
func (*fakeReport) Summary(context.Context, uint64, BillingRange) (Summary, error) {
	return Summary{}, nil
}
func (*fakeReport) ExportCSV(context.Context, uint64, BillingRange) (io.ReadCloser, error) {
	return nil, ErrUnavailable
}
func (*fakeReport) ListPending(context.Context, uint64, PageQuery) ([]Pending, uint64, error) {
	return nil, 0, nil
}
func (*fakeReport) GetOrder(context.Context, uint64, uint64) (Order, error) { return Order{}, nil }
func (f *fakeReport) ProcessPending(context.Context, WriteMeta, uint64, PendingAction, string) (PendingResult, error) {
	f.called = true
	return f.pendingResult, nil
}
func (*fakeReport) RequestRefund(context.Context, WriteMeta, uint64, string) (Order, Refund, error) {
	return Order{}, Refund{}, nil
}

func TestPendingMutationProjectsTypedNestedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name, body, key, wantID string
		result                  PendingResult
	}{
		{
			name: "materialized order", body: `{"action":"MATERIALIZE","reason":""}`, key: "order", wantID: "41",
			result: PendingResult{Order: &Order{ID: 41, State: "制作中", Items: []Item{}}},
		},
		{
			name: "paid prepayment refund", body: `{"action":"REFUND","reason":"取餐已过期"}`, key: "refund", wantID: "91",
			result: PendingResult{Refund: &Refund{ID: 91, State: "退款中"}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := &fakeReport{pendingResult: test.result}
			engine := gin.New()
			group := engine.Group("/api/v1/admin")
			group.Use(func(c *gin.Context) { c.Set("actor_user_id", uint64(7)); c.Set("request_id", "request-1"); c.Next() })
			NewHandler(app).RegisterRoutes(group)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/pending-payments/17", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Idempotency-Key", "operation-1")
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			var body map[string]map[string]any
			if json.Unmarshal(response.Body.Bytes(), &body) != nil || response.Code != http.StatusOK || body[test.key]["id"] != test.wantID {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}
func TestPendingMutationWithoutIdempotencyKeyFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	f := &fakeReport{}
	e := gin.New()
	g := e.Group("/api/v1/admin")
	g.Use(func(c *gin.Context) { c.Set("actor_user_id", uint64(1)); c.Next() })
	NewHandler(f).RegisterRoutes(g)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/pending-payments/7", strings.NewReader(`{"action":"MATERIALIZE","reason":""}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || f.called {
		t.Fatalf("status=%d called=%v", w.Code, f.called)
	}
}
