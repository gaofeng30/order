package storefront

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type publicURLStub struct {
	url string
	err error
}

func (stub publicURLStub) PublicURL(context.Context, string) (string, error) {
	return stub.url, stub.err
}

func TestPublicV27ReturnsFrozenStorefrontDTO(t *testing.T) {
	settings := validSettings()
	settings.Flavors = []string{"少盐", "不辣"}
	settings.LaunchLayer = &LaunchLayer{
		ImageObjectKey: "launch/sha256.png", CenterX: 0.5, CenterY: 0.4, WidthRatio: 0.8, AspectRatio: 1.5,
	}
	router := storefrontTestRouter(NewHandler(&settingsReaderStub{settings: settings}, publicURLStub{url: "https://static.example/launch.png"}))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/storefront/settings", nil))

	assertExactStorefrontResponse(t, response, http.StatusOK, `{"storefront":{"name":"绥安食品","address":"党政办公中心后院老食堂","pickup_point":"党政办公中心后院老食堂北门","announcement":"今日公告","business_status":"open","launch_layer":{"image":{"object_key":"launch/sha256.png","url":"https://static.example/launch.png"},"center_x":0.5,"center_y":0.4,"width_ratio":0.8,"aspect_ratio":1.5},"flavors":["少盐","不辣"]}}`)
}

func TestPublicV27OmitsUnreadableLaunchObjectOnly(t *testing.T) {
	settings := validSettings()
	settings.Flavors = []string{}
	settings.LaunchLayer = &LaunchLayer{
		ImageObjectKey: "launch/missing.png", CenterX: 0.5, CenterY: 0.4, WidthRatio: 0.8, AspectRatio: 1.5,
	}
	router := storefrontTestRouter(NewHandler(&settingsReaderStub{settings: settings}, publicURLStub{err: errors.New("object missing")}))

	response := performStorefrontRequest(router)
	assertExactStorefrontResponse(t, response, http.StatusOK, `{"storefront":{"name":"绥安食品","address":"党政办公中心后院老食堂","pickup_point":"党政办公中心后院老食堂北门","announcement":"今日公告","business_status":"open","flavors":[]}}`)
}
