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

type Reader interface {
	Get(context.Context) (Settings, error)
}
type PublicURLer interface {
	PublicURL(context.Context, string) (string, error)
}

type Handler struct {
	reader Reader
	urls   PublicURLer
}

// NewHandler keeps the resolver optional only so root composition can be
// updated independently. A layer without a usable resolver is safely omitted.
func NewHandler(reader Reader, urls ...PublicURLer) *Handler {
	handler := &Handler{reader: reader}
	if len(urls) == 1 {
		handler.urls = urls[0]
	}
	return handler
}

func (handler *Handler) RegisterRoutes(engine *gin.Engine) {
	engine.GET("/api/v1/storefront/settings", handler.get)
}

type settingsEnvelope struct {
	Storefront settingsResponse `json:"storefront"`
}
type settingsResponse struct {
	Name           string               `json:"name"`
	Address        string               `json:"address"`
	PickupPoint    string               `json:"pickup_point"`
	Announcement   string               `json:"announcement"`
	BusinessStatus BusinessStatus       `json:"business_status"`
	LaunchLayer    *launchLayerResponse `json:"launch_layer,omitempty"`
	Flavors        []string             `json:"flavors"`
}
type imageResponse struct {
	ObjectKey string `json:"object_key"`
	URL       string `json:"url"`
}
type launchLayerResponse struct {
	Image       imageResponse `json:"image"`
	CenterX     float64       `json:"center_x"`
	CenterY     float64       `json:"center_y"`
	WidthRatio  float64       `json:"width_ratio"`
	AspectRatio float64       `json:"aspect_ratio"`
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
		Name: settings.StoreName, Address: settings.StoreAddress, PickupPoint: settings.PickupPoint,
		Announcement: settings.Announcement, BusinessStatus: settings.BusinessStatus,
		Flavors: append(make([]string, 0, len(settings.Flavors)), settings.Flavors...),
	}
	if settings.LaunchLayer != nil && handler.urls != nil {
		if publicURL, resolveErr := handler.urls.PublicURL(ctx.Request.Context(), settings.LaunchLayer.ImageObjectKey); resolveErr == nil && publicURL != "" {
			response.LaunchLayer = &launchLayerResponse{
				Image:   imageResponse{ObjectKey: settings.LaunchLayer.ImageObjectKey, URL: publicURL},
				CenterX: settings.LaunchLayer.CenterX, CenterY: settings.LaunchLayer.CenterY,
				WidthRatio: settings.LaunchLayer.WidthRatio, AspectRatio: settings.LaunchLayer.AspectRatio,
			}
		}
	}
	ctx.JSON(http.StatusOK, settingsEnvelope{Storefront: response})
}

func writeStorefrontUnavailable(ctx *gin.Context) {
	ctx.JSON(http.StatusServiceUnavailable, errorEnvelope{Error: errorResponse{
		Code: storefrontUnavailableCode, Message: storefrontUnavailableMessage,
	}})
}
