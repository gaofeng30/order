package adminreport

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

type exportRPC struct{ calls int }

func (f *exportRPC) CallRPC(_ context.Context, name string, p map[string]any, out any) error {
	if name != "order_admin_report_export_v1" {
		return fmt.Errorf("export fetched unused order/item details via %s", name)
	}
	f.calls++
	rows := [][]any{}
	if p["p_after"] == "0" {
		for i := 1; i <= 100; i++ {
			rows = append(rows, []any{fmt.Sprint(i), fmt.Sprintf("order-%d", i), "0001", map[string]any{"seconds": "1791878400", "nanosecond": 0}, "transaction", "1001"})
		}
	}
	raw, _ := json.Marshal(map[string]any{"outcome": "FOUND", "rows": rows})
	return json.Unmarshal(raw, out)
}
func TestExportUsesNarrowProjection(t *testing.T) {
	rpc := &exportRPC{}
	reader, err := NewSharedPGApplication(rpc, nil).ExportCSV(context.Background(), 1, BillingRange{"2026-10-13", "2026-10-13"})
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	b, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if rpc.calls != 2 || !strings.HasPrefix(string(b), "\ufeff订单号,取餐号,支付时间,微信交易号,实付金额\n") || !strings.Contains(string(b), "order-100,0001,") || !strings.HasSuffix(string(b), ",transaction,10.01\n") || strings.Count(string(b), "\n") != 101 {
		t.Fatalf("changed export contract: calls=%d csv=%s", rpc.calls, b)
	}
}

type countsRPC struct {
	outcome string
	rows    [][]any
	calls   int
}

func (r *countsRPC) CallRPC(_ context.Context, name string, p map[string]any, out any) error {
	r.calls++
	if name != "order_admin_report_counts_v1" || p["p_owner"] != "7" {
		return ErrUnavailable
	}
	b, _ := json.Marshal(map[string]any{"outcome": r.outcome, "rows": r.rows})
	return json.Unmarshal(b, out)
}
func TestOrderCountsContract(t *testing.T) {
	rpc := &countsRPC{outcome: "FOUND", rows: [][]any{{"COMPLETED", "1000"}, {"PREPARING", "2"}}}
	app := NewSharedPGApplication(rpc, nil)
	counts, err := app.OrderCounts(context.Background(), 7)
	if err != nil || len(counts) != 6 || counts["COMPLETED"] != 1000 || counts["RESERVED"] != 0 || rpc.calls != 1 {
		t.Fatalf("%v %v", counts, err)
	}
	rpc.rows = append(rpc.rows, []any{"COMPLETED", "1"})
	if _, err = app.OrderCounts(context.Background(), 7); err == nil {
		t.Fatal("duplicate status accepted")
	}
	rpc.outcome = "FORBIDDEN"
	if _, err = app.OrderCounts(context.Background(), 7); err != ErrForbidden {
		t.Fatal("owner authorization lost")
	}
}
func TestCountsHTTPUsesOwnerRead(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, user := range []uint64{0, 7} {
		rpc := &countsRPC{outcome: "FOUND", rows: [][]any{{"COMPLETED", "1000"}}}
		engine := gin.New()
		group := engine.Group("/api/v1/admin")
		group.Use(func(c *gin.Context) { c.Set("actor_user_id", user); c.Next() })
		NewHandler(NewSharedPGApplication(rpc, nil)).RegisterRoutes(group)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/admin/orders/counts", nil))
		want := 200
		if user == 0 {
			want = 403
		}
		if rec.Code != want || rpc.calls != int(user/7) {
			t.Fatalf("user=%d code=%d calls=%d body=%s", user, rec.Code, rpc.calls, rec.Body.String())
		}
	}
}
