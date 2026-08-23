package staffdiscount

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeStaffApp struct {
	cmd  Command
	meta WriteMeta
}

func (*fakeStaffApp) List(context.Context, string) ([]Staff, error) { return nil, nil }
func (*fakeStaffApp) DiscountRate(context.Context) (uint8, error)   { return 90, nil }
func (f *fakeStaffApp) Execute(_ context.Context, m WriteMeta, c Command) (Result, error) {
	f.meta = m
	f.cmd = c
	return Result{RatePercent: c.RatePercent}, nil
}
func TestDiscountRateUsesAuthenticatedReceiptMeta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	f := &fakeStaffApp{}
	e := gin.New()
	g := e.Group("/api/v1/admin")
	g.Use(func(c *gin.Context) { c.Set("actor_user_id", uint64(7)); c.Next() })
	NewHandler(f).RegisterRoutes(g)
	r := httptest.NewRequest(http.MethodPut, "/api/v1/admin/discount-rate", strings.NewReader(`{"rate_percent":88}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "rate-1")
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	if w.Code != http.StatusOK || f.meta.ActorUserID != 7 || f.cmd.RatePercent != 88 {
		t.Fatalf("status=%d meta=%#v cmd=%#v", w.Code, f.meta, f.cmd)
	}
}

func TestStaffIdentityKeyUsesNFKCAndRemovesAllUnicodeWhitespace(t *testing.T) {
	name, key, phone, ok := staffInput(" Ｌ林\u3000 建\u00a0国 ", "13800006620")
	if !ok || name != "L林  建 国" || string(key) != "L林建国" || phone != "+8613800006620" {
		t.Fatalf("name=%q key=%q phone=%q ok=%v", name, key, phone, ok)
	}
}

func TestStaffPhoneIsMaskedAtTheApplicationBoundary(t *testing.T) {
	if got := maskPhone("+8613800006620"); got != "138****6620" {
		t.Fatalf("mask=%q", got)
	}
}
