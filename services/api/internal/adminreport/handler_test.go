package adminreport

import (
	"context"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeReport struct{ called bool }

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
func (f *fakeReport) ProcessPending(context.Context, WriteMeta, uint64, PendingAction, string) (any, error) {
	f.called = true
	return nil, nil
}
func (*fakeReport) RequestRefund(context.Context, WriteMeta, uint64, string) (Order, Refund, error) {
	return Order{}, Refund{}, nil
}
func (*fakeReport) Advance(context.Context, WriteMeta, uint64) (Order, error) { return Order{}, nil }
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
