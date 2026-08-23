package storefront

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeStorefrontAdmin struct{}

func (fakeStorefrontAdmin) Admin(context.Context) (AdminSettings, error) { return AdminSettings{}, nil }
func (fakeStorefrontAdmin) Configure(context.Context, WriteMeta, SettingsCommand) (AdminSettings, error) {
	return AdminSettings{}, nil
}
func (fakeStorefrontAdmin) LaunchLayer(context.Context) (LaunchLayerConfig, error) {
	return LaunchLayerConfig{}, nil
}
func (fakeStorefrontAdmin) ConfigureLaunchLayer(context.Context, WriteMeta, *LaunchLayerConfig) (*LaunchLayerConfig, error) {
	return &LaunchLayerConfig{}, nil
}
func TestAdminSettingsRejectsThirdMealPeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	g := e.Group("/api/v1/admin")
	g.Use(func(c *gin.Context) { c.Set("actor_user_id", uint64(1)); c.Next() })
	NewAdminHandler(fakeStorefrontAdmin{}).RegisterRoutes(g)
	body := `{"store_status":"open","pickup_point":"门店","notice":"","pickup_step_min":10,"meal_periods":[{"code":"lunch","name":"午餐","cutoff_time":"11:00","pickup_from":"11:30","pickup_to":"13:00"},{"code":"dinner","name":"晚餐","cutoff_time":"17:00","pickup_from":"17:30","pickup_to":"19:00"},{"code":"night","name":"夜宵","cutoff_time":"21:00","pickup_from":"21:30","pickup_to":"22:00"}],"service_dates":[]}`
	r := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "op")
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
