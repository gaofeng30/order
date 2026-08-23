package menu

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	menuTimezone            = "Asia/Shanghai"
	invalidSelectionCode    = "INVALID_MENU_SELECTION"
	invalidSelectionMessage = "invalid menu selection"
	menuUnavailableCode     = "MENU_UNAVAILABLE"
	menuUnavailableMessage  = "menu temporarily unavailable"
)

var shanghaiLocation = time.FixedZone(menuTimezone, 8*60*60)

// Reader is the fixed two-query boundary used by the public menu handler.
type Reader interface {
	MealPeriods(context.Context) ([]MealPeriodRecord, error)
	List(context.Context, string, MealCode) ([]Category, error)
}

// Handler serves the anonymous reservation menu read contract.
type Handler struct {
	reader Reader
	now    func() time.Time
}

// NewHandler constructs a menu handler with one injectable request clock.
func NewHandler(reader Reader, now func() time.Time) *Handler {
	return &Handler{reader: reader, now: now}
}

// RegisterRoutes adds the versioned anonymous menu read routes.
func (handler *Handler) RegisterRoutes(engine *gin.Engine) {
	engine.GET("/api/v1/menu", handler.get)
	engine.GET("/api/v1/menu/pickup-options", handler.getPickupOptions)
}

type menuResponse struct {
	Selection  selectionResponse  `json:"selection"`
	Meal       mealResponse       `json:"meal"`
	Categories []categoryResponse `json:"categories"`
}

type selectionResponse struct {
	Date     string `json:"date"`
	Time     string `json:"time"`
	Timezone string `json:"timezone"`
}

type mealResponse struct {
	Code      MealCode `json:"code"`
	CutoffAt  string   `json:"cutoff_at"`
	Orderable bool     `json:"orderable"`
}

type categoryResponse struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Products []productResponse `json:"products"`
}

type productResponse struct {
	ID            string `json:"id"`
	CategoryID    string `json:"category_id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Specification string `json:"specification"`
	PriceCents    uint32 `json:"price_cents"`
	SoldOut       bool   `json:"sold_out"`
	Orderable     bool   `json:"orderable"`
}

type menuErrorEnvelope struct {
	Error menuErrorResponse `json:"error"`
}

type menuErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (handler *Handler) get(ctx *gin.Context) {
	now := handler.now().In(shanghaiLocation)
	dateValue, pickupValue, serviceDate, ok := parseRequestSelection(ctx, now)
	if !ok {
		writeInvalidSelection(ctx)
		return
	}
	periods, err := handler.reader.MealPeriods(ctx.Request.Context())
	if err != nil {
		writeMenuUnavailable(ctx)
		return
	}
	meal, err := ResolveMeal(periods, serviceDate, pickupValue, now)
	if errors.Is(err, ErrInvalidSelection) {
		writeInvalidSelection(ctx)
		return
	}
	if err != nil {
		writeMenuUnavailable(ctx)
		return
	}
	categories, err := handler.reader.List(ctx.Request.Context(), dateValue, meal.Code)
	if err != nil {
		writeMenuUnavailable(ctx)
		return
	}

	response := menuResponse{
		Selection:  selectionResponse{Date: dateValue, Time: pickupValue, Timezone: menuTimezone},
		Meal:       mealResponse{Code: meal.Code, CutoffAt: meal.CutoffAt.Format(time.RFC3339), Orderable: meal.Orderable},
		Categories: newCategoryResponses(categories, meal.Orderable),
	}
	ctx.JSON(http.StatusOK, response)
}

func newCategoryResponses(categories []Category, mealOrderable bool) []categoryResponse {
	result := make([]categoryResponse, 0, len(categories))
	for _, category := range categories {
		item := categoryResponse{
			ID: strconv.FormatUint(category.ID, 10), Name: category.Name,
			Products: make([]productResponse, 0, len(category.Products)),
		}
		for _, product := range category.Products {
			item.Products = append(item.Products, productResponse{
				ID: strconv.FormatUint(product.ID, 10), CategoryID: strconv.FormatUint(product.CategoryID, 10),
				Name: product.Name, Description: product.Description, Specification: product.Specification,
				PriceCents: product.PriceCents, SoldOut: product.SoldOut, Orderable: mealOrderable && !product.SoldOut,
			})
		}
		result = append(result, item)
	}
	return result
}

func parseRequestSelection(ctx *gin.Context, now time.Time) (string, string, time.Time, bool) {
	query := ctx.Request.URL.Query()
	dates, dateExists := query["date"]
	times, timeExists := query["time"]
	if !dateExists || !timeExists || len(dates) != 1 || len(times) != 1 || !strictDate(dates[0]) {
		return "", "", time.Time{}, false
	}
	if _, ok := parsePickupMinute(times[0]); !ok {
		return "", "", time.Time{}, false
	}
	serviceDate, err := time.ParseInLocation("2006-01-02", dates[0], shanghaiLocation)
	if err != nil {
		return "", "", time.Time{}, false
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, shanghaiLocation)
	if !serviceDate.Equal(today) && !serviceDate.Equal(today.AddDate(0, 0, 1)) {
		return "", "", time.Time{}, false
	}
	return dates[0], times[0], serviceDate, true
}

func strictDate(value string) bool {
	if len(value) != len("2006-01-02") || value[4] != '-' || value[7] != '-' {
		return false
	}
	for index := range value {
		if index == 4 || index == 7 {
			continue
		}
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}

func writeInvalidSelection(ctx *gin.Context) {
	ctx.JSON(http.StatusBadRequest, menuErrorEnvelope{Error: menuErrorResponse{Code: invalidSelectionCode, Message: invalidSelectionMessage}})
}

func writeMenuUnavailable(ctx *gin.Context) {
	ctx.JSON(http.StatusServiceUnavailable, menuErrorEnvelope{Error: menuErrorResponse{Code: menuUnavailableCode, Message: menuUnavailableMessage}})
}
