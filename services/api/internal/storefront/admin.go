package storefront

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	ErrAdminInvalidInput        = errors.New("storefront invalid input")
	ErrAdminConflict            = errors.New("storefront conflict")
	ErrAdminIdempotencyConflict = errors.New("storefront idempotency conflict")
	ErrAdminUnavailable         = errors.New("storefront unavailable")
)

type WriteMeta struct {
	ActorUserID    uint64
	IdempotencyKey string
	RequestID      string
}

type MealPeriodConfig struct {
	Code, Name, CutoffTime, PickupFrom, PickupTo string
}

type ServiceDateConfig struct {
	Date, Status string
}

type AdminSettings struct {
	StoreName, PickupPoint, Notice, StoreStatus, ServiceDate string
	PickupStepMin                                            uint16
	MealPeriods                                              []MealPeriodConfig
	ServiceDates                                             []ServiceDateConfig
}

type LaunchLayerConfig struct {
	ImageObjectKey              string
	Enabled                     bool
	SizeRatio, CenterX, CenterY float64
	AspectRatio                 float64
}

type SettingsCommand struct {
	StoreStatus, Notice, PickupPoint string
	PickupStepMin                    uint16
	MealPeriods                      []MealPeriodConfig
	ServiceDates                     []ServiceDateConfig
}

// AdminApplication owns storefront/date/meal validation and atomic audit receipt writes.
type AdminApplication interface {
	Admin(context.Context) (AdminSettings, error)
	Configure(context.Context, WriteMeta, SettingsCommand) (AdminSettings, error)
	LaunchLayer(context.Context) (LaunchLayerConfig, error)
	ConfigureLaunchLayer(context.Context, WriteMeta, *LaunchLayerConfig) (*LaunchLayerConfig, error)
}

type AdminHandler struct{ app AdminApplication }

func NewAdminHandler(app AdminApplication) *AdminHandler { return &AdminHandler{app: app} }

func (h *AdminHandler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/settings", h.getSettings)
	group.PUT("/settings", h.putSettings)
	group.GET("/meal-periods", h.getSettings)
	group.PUT("/meal-periods", h.putSettings)
	group.GET("/launch-layer", h.getLayer)
	group.PUT("/launch-layer", h.putLayer)
	group.DELETE("/launch-layer", h.deleteLayer)
}

type mealDTO struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	CutoffTime string `json:"cutoff_time"`
	PickupFrom string `json:"pickup_from"`
	PickupTo   string `json:"pickup_to"`
}
type serviceDateDTO struct {
	Date   string `json:"date"`
	Status string `json:"status"`
}
type settingsDTO struct {
	StoreName     string           `json:"store_name"`
	PickupPoint   string           `json:"pickup_point"`
	Notice        string           `json:"notice"`
	StoreStatus   string           `json:"store_status"`
	ServiceDate   string           `json:"service_date"`
	PickupStepMin uint16           `json:"pickup_step_min"`
	MealPeriods   []mealDTO        `json:"meal_periods"`
	ServiceDates  []serviceDateDTO `json:"service_dates"`
}
type settingsWrite struct {
	StoreStatus   string           `json:"store_status"`
	PickupPoint   string           `json:"pickup_point"`
	Notice        string           `json:"notice"`
	PickupStepMin uint16           `json:"pickup_step_min"`
	MealPeriods   []mealDTO        `json:"meal_periods"`
	ServiceDates  []serviceDateDTO `json:"service_dates"`
}
type layerDTO struct {
	ImageObjectKey string  `json:"image_object_key"`
	Enabled        bool    `json:"enabled"`
	SizeRatio      float64 `json:"size_ratio"`
	CenterX        float64 `json:"center_x"`
	CenterY        float64 `json:"center_y"`
	AspectRatio    float64 `json:"aspect_ratio"`
}

func settingsView(s AdminSettings) settingsDTO {
	meals := make([]mealDTO, 0, len(s.MealPeriods))
	for _, m := range s.MealPeriods {
		meals = append(meals, mealDTO{m.Code, m.Name, m.CutoffTime, m.PickupFrom, m.PickupTo})
	}
	dates := make([]serviceDateDTO, 0, len(s.ServiceDates))
	for _, d := range s.ServiceDates {
		dates = append(dates, serviceDateDTO{d.Date, d.Status})
	}
	return settingsDTO{s.StoreName, s.PickupPoint, s.Notice, s.StoreStatus, s.ServiceDate, s.PickupStepMin, meals, dates}
}
func layerView(l LaunchLayerConfig) layerDTO {
	return layerDTO{l.ImageObjectKey, l.Enabled, l.SizeRatio, l.CenterX, l.CenterY, l.AspectRatio}
}
func (h *AdminHandler) getSettings(c *gin.Context) {
	s, err := h.app.Admin(c.Request.Context())
	if err != nil {
		writeAdminError(c, err)
		return
	}
	c.JSON(http.StatusOK, settingsView(s))
}
func (h *AdminHandler) putSettings(c *gin.Context) {
	var in settingsWrite
	if !strictAdminJSON(c, &in) || !validAdminSettings(in) {
		writeAdminError(c, ErrAdminInvalidInput)
		return
	}
	meta, ok := storefrontWriteMeta(c)
	if !ok {
		writeAdminError(c, ErrAdminInvalidInput)
		return
	}
	meals := make([]MealPeriodConfig, 0, len(in.MealPeriods))
	for _, m := range in.MealPeriods {
		meals = append(meals, MealPeriodConfig{m.Code, m.Name, m.CutoffTime, m.PickupFrom, m.PickupTo})
	}
	dates := make([]ServiceDateConfig, 0, len(in.ServiceDates))
	for _, d := range in.ServiceDates {
		dates = append(dates, ServiceDateConfig{d.Date, d.Status})
	}
	out, err := h.app.Configure(c.Request.Context(), meta, SettingsCommand{in.StoreStatus, in.Notice, in.PickupPoint, in.PickupStepMin, meals, dates})
	if err != nil {
		writeAdminError(c, err)
		return
	}
	c.JSON(http.StatusOK, settingsView(out))
}
func (h *AdminHandler) getLayer(c *gin.Context) {
	l, err := h.app.LaunchLayer(c.Request.Context())
	if err != nil {
		writeAdminError(c, err)
		return
	}
	c.JSON(http.StatusOK, layerView(l))
}
func (h *AdminHandler) putLayer(c *gin.Context) {
	var in layerDTO
	if !strictAdminJSON(c, &in) || in.ImageObjectKey == "" || in.SizeRatio <= 0 || in.SizeRatio > 1 || in.CenterX < 0 || in.CenterX > 1 || in.CenterY < 0 || in.CenterY > 1 || in.AspectRatio <= 0 {
		writeAdminError(c, ErrAdminInvalidInput)
		return
	}
	meta, ok := storefrontWriteMeta(c)
	if !ok {
		writeAdminError(c, ErrAdminInvalidInput)
		return
	}
	out, err := h.app.ConfigureLaunchLayer(c.Request.Context(), meta, &LaunchLayerConfig{in.ImageObjectKey, in.Enabled, in.SizeRatio, in.CenterX, in.CenterY, in.AspectRatio})
	if err != nil {
		writeAdminError(c, err)
		return
	}
	c.JSON(http.StatusOK, layerView(*out))
}
func (h *AdminHandler) deleteLayer(c *gin.Context) {
	meta, ok := storefrontWriteMeta(c)
	if !ok {
		writeAdminError(c, ErrAdminInvalidInput)
		return
	}
	_, err := h.app.ConfigureLaunchLayer(c.Request.Context(), meta, nil)
	if err != nil {
		writeAdminError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func validAdminSettings(in settingsWrite) bool {
	if in.StoreStatus != "open" && in.StoreStatus != "closed" && in.StoreStatus != "cutoff" && in.StoreStatus != "营业中" && in.StoreStatus != "休息中" && in.StoreStatus != "已截单" {
		return false
	}
	if in.PickupStepMin == 0 || strings.TrimSpace(in.PickupPoint) == "" || len(in.MealPeriods) != 2 {
		return false
	}
	seen := map[string]bool{}
	for _, m := range in.MealPeriods {
		code := strings.ToLower(m.Code)
		if (code != "lunch" && code != "dinner") || seen[code] || m.CutoffTime == "" || m.PickupFrom == "" || m.PickupTo == "" || m.PickupFrom > m.PickupTo {
			return false
		}
		seen[code] = true
	}
	return true
}
func strictAdminJSON(c *gin.Context, out any) bool {
	if !strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
		return false
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 65537))
	if err != nil || len(body) == 0 || len(body) > 65536 {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(out) != nil {
		return false
	}
	var extra any
	return errors.Is(decoder.Decode(&extra), io.EOF)
}
func storefrontWriteMeta(c *gin.Context) (WriteMeta, bool) {
	actor := c.GetUint64("actor_user_id")
	keys := c.Request.Header.Values("Idempotency-Key")
	if actor == 0 || len(keys) != 1 || strings.TrimSpace(keys[0]) == "" || strings.ContainsAny(keys[0], " \t\r\n") {
		return WriteMeta{}, false
	}
	return WriteMeta{actor, keys[0], c.GetString("request_id")}, true
}
func writeAdminError(c *gin.Context, err error) {
	status, code, message := http.StatusServiceUnavailable, "STOREFRONT_UNAVAILABLE", "storefront temporarily unavailable"
	if errors.Is(err, ErrAdminInvalidInput) {
		status, code, message = http.StatusBadRequest, "INVALID_REQUEST", "invalid request"
	} else if errors.Is(err, ErrAdminConflict) {
		status, code, message = http.StatusConflict, "STOREFRONT_CONFLICT", "storefront conflict"
	} else if errors.Is(err, ErrAdminIdempotencyConflict) {
		status, code, message = http.StatusConflict, "IDEMPOTENCY_CONFLICT", "idempotency conflict"
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
