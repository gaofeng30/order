package storefront

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPublicSettingsReturnsExactDTOWithoutLaunchLayer(t *testing.T) {
	reader := &settingsReaderStub{settings: validSettings()}
	router := storefrontTestRouter(NewHandler(reader))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/storefront/settings", nil))

	assertExactStorefrontResponse(t, response, http.StatusOK, `{"settings":{"store_name":"绥安食品","store_address":"党政办公中心后院老食堂","pickup_point":"党政办公中心后院老食堂北门","announcement":"今日公告","business_status":"open","launch_layer":null}}`)
	if reader.calls != 1 {
		t.Fatalf("reader calls = %d, want 1", reader.calls)
	}
}

func TestPublicSettingsReturnsOnlyFrozenLaunchLayerFields(t *testing.T) {
	settings := validSettings()
	settings.LaunchLayer = &LaunchLayer{
		PNGURL: "https://static.example.com/launch.PNG", CenterX: 0, CenterY: 1, WidthRatio: 1, AspectRatio: 2,
	}
	reader := &settingsReaderStub{settings: settings}
	router := storefrontTestRouter(NewHandler(reader))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/storefront/settings", nil))

	assertExactStorefrontResponse(t, response, http.StatusOK, `{"settings":{"store_name":"绥安食品","store_address":"党政办公中心后院老食堂","pickup_point":"党政办公中心后院老食堂北门","announcement":"今日公告","business_status":"open","launch_layer":{"png_url":"https://static.example.com/launch.PNG","center_x":0,"center_y":1,"width_ratio":1,"aspect_ratio":2}}}`)
}

func TestPublicSettingsMapsReaderFailureToExactUnavailableDTO(t *testing.T) {
	reader := &settingsReaderStub{err: errors.New("database canary")}
	router := storefrontTestRouter(NewHandler(reader))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/storefront/settings", nil))

	assertExactStorefrontResponse(t, response, http.StatusServiceUnavailable, `{"error":{"code":"STOREFRONT_UNAVAILABLE","message":"storefront temporarily unavailable"}}`)
	for _, forbidden := range []string{"database canary", "SELECT", "mysql"} {
		if strings.Contains(response.Body.String(), forbidden) {
			t.Fatalf("response leaked internal value %q", forbidden)
		}
	}
}

func TestPublicSettingsFailsClosedForInvalidBusinessStatus(t *testing.T) {
	settings := validSettings()
	settings.BusinessStatus = BusinessStatus("unexpected")
	reader := &settingsReaderStub{settings: settings}
	router := storefrontTestRouter(NewHandler(reader))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/storefront/settings", nil))

	assertExactStorefrontResponse(t, response, http.StatusServiceUnavailable, `{"error":{"code":"STOREFRONT_UNAVAILABLE","message":"storefront temporarily unavailable"}}`)
}

func TestPublicSettingsValidatesFrozenTextContract(t *testing.T) {
	invalid := []struct {
		name   string
		mutate func(*Settings)
	}{
		{name: "empty store name", mutate: func(settings *Settings) { settings.StoreName = "" }},
		{name: "invalid store name utf8", mutate: func(settings *Settings) { settings.StoreName = string([]byte{0xff}) }},
		{name: "untrimmed store address", mutate: func(settings *Settings) { settings.StoreAddress = " 地址" }},
		{name: "whitespace pickup point", mutate: func(settings *Settings) { settings.PickupPoint = "\u3000" }},
		{name: "invalid announcement utf8", mutate: func(settings *Settings) { settings.Announcement = string([]byte{0xff}) }},
		{name: "announcement over 1000 runes", mutate: func(settings *Settings) { settings.Announcement = strings.Repeat("公", 1001) }},
	}
	for _, testCase := range invalid {
		t.Run(testCase.name, func(t *testing.T) {
			settings := validSettings()
			testCase.mutate(&settings)
			response := performStorefrontRequest(storefrontTestRouter(NewHandler(&settingsReaderStub{settings: settings})))
			assertExactStorefrontResponse(t, response, http.StatusServiceUnavailable, `{"error":{"code":"STOREFRONT_UNAVAILABLE","message":"storefront temporarily unavailable"}}`)
		})
	}

	boundary := validSettings()
	boundary.StoreName = strings.Repeat("店", 2000)
	boundary.Announcement = strings.Repeat("公", 1000)
	if response := performStorefrontRequest(storefrontTestRouter(NewHandler(&settingsReaderStub{settings: boundary}))); response.Code != http.StatusOK {
		t.Fatalf("valid text boundary status = %d, want 200", response.Code)
	}
	emptyAnnouncement := validSettings()
	emptyAnnouncement.Announcement = ""
	if response := performStorefrontRequest(storefrontTestRouter(NewHandler(&settingsReaderStub{settings: emptyAnnouncement}))); response.Code != http.StatusOK {
		t.Fatalf("empty announcement status = %d, want 200", response.Code)
	}
}

func TestPublicSettingsValidatesFrozenLaunchLayerContract(t *testing.T) {
	validLayer := LaunchLayer{PNGURL: "https://static.example.com:65535/path/launch.PNG?revision=1", CenterX: 0, CenterY: 1, WidthRatio: 1, AspectRatio: 0.01}
	invalid := []struct {
		name   string
		mutate func(*LaunchLayer)
	}{
		{name: "empty URL", mutate: func(layer *LaunchLayer) { layer.PNGURL = "" }},
		{name: "invalid URL utf8", mutate: func(layer *LaunchLayer) { layer.PNGURL = string([]byte{0xff}) }},
		{name: "http URL", mutate: func(layer *LaunchLayer) { layer.PNGURL = "http://static.example.com/launch.png" }},
		{name: "missing host", mutate: func(layer *LaunchLayer) { layer.PNGURL = "https:///launch.png" }},
		{name: "userinfo", mutate: func(layer *LaunchLayer) { layer.PNGURL = "https://user@static.example.com/launch.png" }},
		{name: "fragment", mutate: func(layer *LaunchLayer) { layer.PNGURL = "https://static.example.com/launch.png#top" }},
		{name: "empty fragment", mutate: func(layer *LaunchLayer) { layer.PNGURL = "https://static.example.com/launch.png#" }},
		{name: "not PNG", mutate: func(layer *LaunchLayer) { layer.PNGURL = "https://static.example.com/launch.jpg" }},
		{name: "port zero", mutate: func(layer *LaunchLayer) { layer.PNGURL = "https://static.example.com:0/launch.png" }},
		{name: "port overflow", mutate: func(layer *LaunchLayer) { layer.PNGURL = "https://static.example.com:65536/launch.png" }},
		{name: "negative center x", mutate: func(layer *LaunchLayer) { layer.CenterX = -0.0001 }},
		{name: "center x over one", mutate: func(layer *LaunchLayer) { layer.CenterX = 1.0001 }},
		{name: "negative center y", mutate: func(layer *LaunchLayer) { layer.CenterY = -0.0001 }},
		{name: "center y over one", mutate: func(layer *LaunchLayer) { layer.CenterY = 1.0001 }},
		{name: "zero width", mutate: func(layer *LaunchLayer) { layer.WidthRatio = 0 }},
		{name: "width over one", mutate: func(layer *LaunchLayer) { layer.WidthRatio = 1.0001 }},
		{name: "zero aspect ratio", mutate: func(layer *LaunchLayer) { layer.AspectRatio = 0 }},
		{name: "NaN", mutate: func(layer *LaunchLayer) { layer.CenterX = math.NaN() }},
		{name: "infinity", mutate: func(layer *LaunchLayer) { layer.AspectRatio = math.Inf(1) }},
	}
	for _, testCase := range invalid {
		t.Run(testCase.name, func(t *testing.T) {
			layer := validLayer
			testCase.mutate(&layer)
			settings := validSettings()
			settings.LaunchLayer = &layer
			response := performStorefrontRequest(storefrontTestRouter(NewHandler(&settingsReaderStub{settings: settings})))
			assertExactStorefrontResponse(t, response, http.StatusServiceUnavailable, `{"error":{"code":"STOREFRONT_UNAVAILABLE","message":"storefront temporarily unavailable"}}`)
		})
	}

	settings := validSettings()
	settings.LaunchLayer = &validLayer
	if response := performStorefrontRequest(storefrontTestRouter(NewHandler(&settingsReaderStub{settings: settings}))); response.Code != http.StatusOK {
		t.Fatalf("valid launch boundary status = %d, want 200", response.Code)
	}
}

func validSettings() Settings {
	return Settings{
		StoreName:      "绥安食品",
		StoreAddress:   "党政办公中心后院老食堂",
		PickupPoint:    "党政办公中心后院老食堂北门",
		Announcement:   "今日公告",
		BusinessStatus: BusinessOpen,
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
