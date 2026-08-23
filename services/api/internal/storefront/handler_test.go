package storefront

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPublicSettingsFailsClosedForInvalidBaseFacts(t *testing.T) {
	for _, mutate := range []func(*Settings){
		func(settings *Settings) { settings.StoreName = "" },
		func(settings *Settings) { settings.BusinessStatus = "unexpected" },
		func(settings *Settings) { settings.Flavors = []string{"少盐", "少盐"} },
		func(settings *Settings) { settings.Flavors = []string{" "} },
		func(settings *Settings) {
			settings.LaunchLayer = &LaunchLayer{ImageObjectKey: "", CenterX: .5, CenterY: .5, WidthRatio: .5, AspectRatio: 1}
		},
	} {
		settings := validSettings()
		mutate(&settings)
		response := performStorefrontRequest(storefrontTestRouter(NewHandler(&settingsReaderStub{settings: settings})))
		assertExactStorefrontResponse(t, response, http.StatusServiceUnavailable, `{"error":{"code":"STOREFRONT_UNAVAILABLE","message":"storefront temporarily unavailable"}}`)
	}
}

func TestPublicSettingsRedactsReaderFailure(t *testing.T) {
	response := performStorefrontRequest(storefrontTestRouter(NewHandler(&settingsReaderStub{err: errors.New("database canary")})))
	assertExactStorefrontResponse(t, response, http.StatusServiceUnavailable, `{"error":{"code":"STOREFRONT_UNAVAILABLE","message":"storefront temporarily unavailable"}}`)
	if strings.Contains(response.Body.String(), "canary") {
		t.Fatal("reader failure leaked")
	}
}

func validSettings() Settings {
	return Settings{
		StoreName: "绥安食品", StoreAddress: "党政办公中心后院老食堂", PickupPoint: "党政办公中心后院老食堂北门",
		Announcement: "今日公告", BusinessStatus: BusinessOpen, Flavors: []string{},
	}
}

type settingsReaderStub struct {
	settings Settings
	err      error
	calls    int
}

func (reader *settingsReaderStub) Get(context.Context) (Settings, error) {
	reader.calls++
	return reader.settings, reader.err
}

func storefrontTestRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	handler.RegisterRoutes(engine)
	return engine
}

func performStorefrontRequest(router http.Handler) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/storefront/settings", nil))
	return response
}

func assertExactStorefrontResponse(t *testing.T, recorder *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	got := strings.TrimSpace(recorder.Body.String())
	if recorder.Code != status || got != body {
		t.Fatalf("response = %d/%q, want %d/%q", recorder.Code, got, status, body)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}
}
