package storefront

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	storefrontUnavailableCode    = "STOREFRONT_UNAVAILABLE"
	storefrontUnavailableMessage = "storefront temporarily unavailable"
)

// Reader is the public storefront settings query boundary.
type Reader interface {
	Get(context.Context) (Settings, error)
}

// Handler serves the anonymous storefront settings API.
type Handler struct {
	reader Reader
}

// NewHandler constructs a storefront settings handler.
func NewHandler(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// RegisterRoutes adds the anonymous storefront settings route.
func (handler *Handler) RegisterRoutes(engine *gin.Engine) {
	engine.GET("/api/v1/storefront/settings", handler.get)
}

type settingsEnvelope struct {
	Settings settingsResponse `json:"settings"`
}

type settingsResponse struct {
	StoreName      string               `json:"store_name"`
	StoreAddress   string               `json:"store_address"`
	PickupPoint    string               `json:"pickup_point"`
	Announcement   string               `json:"announcement"`
	BusinessStatus BusinessStatus       `json:"business_status"`
	LaunchLayer    *launchLayerResponse `json:"launch_layer"`
}

type launchLayerResponse struct {
	PNGURL      string  `json:"png_url"`
	CenterX     float64 `json:"center_x"`
	CenterY     float64 `json:"center_y"`
	WidthRatio  float64 `json:"width_ratio"`
	AspectRatio float64 `json:"aspect_ratio"`
}

type errorEnvelope struct {
	Error errorResponse `json:"error"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (handler *Handler) get(ctx *gin.Context) {
	settings, err := handler.reader.Get(ctx.Request.Context())
	if err != nil || !settings.valid() {
		writeStorefrontUnavailable(ctx)
		return
	}
	response := settingsResponse{
		StoreName: settings.StoreName, StoreAddress: settings.StoreAddress, PickupPoint: settings.PickupPoint,
		Announcement: settings.Announcement, BusinessStatus: settings.BusinessStatus,
	}
	if settings.LaunchLayer != nil {
		response.LaunchLayer = &launchLayerResponse{
			PNGURL: settings.LaunchLayer.PNGURL, CenterX: settings.LaunchLayer.CenterX, CenterY: settings.LaunchLayer.CenterY,
			WidthRatio: settings.LaunchLayer.WidthRatio, AspectRatio: settings.LaunchLayer.AspectRatio,
		}
	}
	ctx.JSON(http.StatusOK, settingsEnvelope{Settings: response})
}

func writeStorefrontUnavailable(ctx *gin.Context) {
	ctx.JSON(http.StatusServiceUnavailable, errorEnvelope{Error: errorResponse{
		Code: storefrontUnavailableCode, Message: storefrontUnavailableMessage,
	}})
}
