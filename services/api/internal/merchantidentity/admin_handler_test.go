package merchantidentity

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeMerchantAdmin struct{ executeErr error }

func (*fakeMerchantAdmin) CurrentAccount(context.Context, uint64) (Account, error) {
	return Account{}, nil
}
func (*fakeMerchantAdmin) ListAccounts(context.Context, uint64, string) ([]Account, error) {
	return nil, nil
}
func (f *fakeMerchantAdmin) ExecuteAccount(context.Context, AdminWriteMeta, AccountCommand) (*Account, error) {
	return nil, f.executeErr
}
func (*fakeMerchantAdmin) BeginPCLogin(context.Context) (PCLogin, error) { return PCLogin{}, nil }
func (*fakeMerchantAdmin) ApprovePCLogin(context.Context, uint64, string, string, string) error {
	return nil
}
func (*fakeMerchantAdmin) PollPCLogin(context.Context, string, string) (PCSession, error) {
	return PCSession{State: "WAITING"}, nil
}
func (*fakeMerchantAdmin) AuthenticatePC(context.Context, string) (uint64, error) { return 1, nil }
func TestLastOwnerIsStableConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	g := e.Group("/api/v1/admin")
	g.Use(func(c *gin.Context) { c.Set("actor_user_id", uint64(1)); c.Next() })
	NewAdminHandler(&fakeMerchantAdmin{executeErr: ErrLastOwner}).RegisterAdminRoutes(g)
	r := httptest.NewRequest(http.MethodPut, "/api/v1/admin/merchant-accounts/9", strings.NewReader(`{"enabled":false}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "disable-owner")
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "LAST_OWNER") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
func TestPCPollNeverReturnsSessionWhileWaiting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	api := e.Group("/api/v1")
	NewAdminHandler(&fakeMerchantAdmin{}).RegisterPCAuthRoutes(api)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/poll", strings.NewReader(`{"login_id":"l","poll_secret":"p"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	if w.Code != http.StatusAccepted || strings.Contains(w.Body.String(), "token") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
