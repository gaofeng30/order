package menu

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gaofeng30/order/services/api/internal/httpdto"
	"github.com/gaofeng30/order/services/api/internal/identity"
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

type Reader interface {
	ReadMenu(context.Context, string) (MenuSnapshot, error)
	ReadPickupFacts(context.Context, []string) (PickupFacts, error)
}
type Authenticator interface {
	Authenticate(context.Context, string) (uint64, error)
}
type PricingResolver interface {
	ResolvePrices(context.Context, uint64, []uint32) ([]*uint32, error)
}
type PublicURLer interface {
	PublicURL(context.Context, string) (string, error)
}

type Handler struct {
	reader Reader
	now    func() time.Time
	auth   Authenticator
	prices PricingResolver
	urls   PublicURLer
}

type HandlerOption func(*Handler)

func WithAuthenticator(value Authenticator) HandlerOption {
	return func(handler *Handler) { handler.auth = value }
}
func WithPricing(value PricingResolver) HandlerOption {
	return func(handler *Handler) { handler.prices = value }
}
func WithPublicURLs(value PublicURLer) HandlerOption {
	return func(handler *Handler) { handler.urls = value }
}

func NewHandler(reader Reader, now func() time.Time, options ...HandlerOption) *Handler {
	handler := &Handler{reader: reader, now: now}
	for _, option := range options {
		if option != nil {
			option(handler)
		}
	}
	return handler
}

func (handler *Handler) RegisterRoutes(engine *gin.Engine) {
	engine.GET("/api/v1/menu", handler.get)
	engine.GET("/api/v1/menu/pickup-options", handler.getPickupOptions)
}

type menuResponse struct {
	Selection   selectionResponse   `json:"selection"`
	StoreStatus storeStatusResponse `json:"store_status"`
	Categories  []categoryResponse  `json:"categories"`
}
type selectionResponse struct {
	Date       string   `json:"date"`
	Time       string   `json:"time"`
	MealPeriod MealCode `json:"meal_period"`
}
type storeStatusResponse struct {
	BusinessStatus       string `json:"business_status"`
	ServiceDateAvailable bool   `json:"service_date_available"`
	MealAvailable        bool   `json:"meal_available"`
	CutoffPassed         bool   `json:"cutoff_passed"`
}
type categoryResponse struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Products []productResponse `json:"products"`
}
type imageResponse struct {
	ObjectKey string `json:"object_key"`
	URL       string `json:"url"`
}
type productResponse struct {
	ID                     string          `json:"id"`
	CategoryID             string          `json:"category_id"`
	Name                   string          `json:"name"`
	Description            string          `json:"description"`
	Specification          string          `json:"specification"`
	MealPeriod             string          `json:"meal_period"`
	Images                 []imageResponse `json:"images"`
	Listed                 bool            `json:"listed"`
	SoldOut                bool            `json:"sold_out"`
	OriginalUnitPriceCents uint32          `json:"original_unit_price_cents"`
	StaffUnitPriceCents    *uint32         `json:"staff_unit_price_cents,omitempty"`
}
type menuErrorEnvelope struct {
	Error menuErrorResponse `json:"error"`
}
type menuErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (handler *Handler) get(ctx *gin.Context) {
	userID, authOK := handler.optionalUser(ctx)
	if !authOK {
		return
	}
	now := handler.now().In(shanghaiLocation)
	dateValue, pickupValue, serviceDate, ok := parseRequestSelection(ctx, now)
	if !ok {
		writeInvalidSelection(ctx)
		return
	}
	snapshot, err := handler.reader.ReadMenu(ctx.Request.Context(), dateValue)
	if err != nil || !validBusinessStatus(snapshot.BusinessStatus) {
		writeMenuUnavailable(ctx)
		return
	}
	meal, err := ResolveMeal(snapshot.MealPeriods, serviceDate, pickupValue, now)
	if errors.Is(err, ErrInvalidSelection) {
		writeInvalidSelection(ctx)
		return
	}
	if err != nil {
		writeMenuUnavailable(ctx)
		return
	}
	categories := filterMealCategories(snapshot.Categories, meal.Code)
	projected, ok := handler.projectCategories(ctx.Request.Context(), categories, userID)
	if !ok {
		writeMenuUnavailable(ctx)
		return
	}
	serviceAvailable := snapshot.ServiceDatePresent && snapshot.ServiceDateOpen
	mealAvailable := snapshot.BusinessStatus == "open" && serviceAvailable && meal.Orderable
	ctx.JSON(http.StatusOK, menuResponse{
		Selection: selectionResponse{Date: dateValue, Time: pickupValue, MealPeriod: meal.Code},
		StoreStatus: storeStatusResponse{
			BusinessStatus: snapshot.BusinessStatus, ServiceDateAvailable: serviceAvailable,
			MealAvailable: mealAvailable, CutoffPassed: !meal.Orderable || snapshot.BusinessStatus == "cutoff",
		},
		Categories: projected,
	})
}

func (handler *Handler) optionalUser(ctx *gin.Context) (uint64, bool) {
	if len(ctx.Request.Header.Values("Authorization")) == 0 {
		return 0, true
	}
	token, err := httpdto.BearerToken(ctx.Request)
	if err != nil {
		writeMenuUnauthenticated(ctx)
		return 0, false
	}
	if handler.auth == nil {
		writeMenuUnavailable(ctx)
		return 0, false
	}
	userID, err := handler.auth.Authenticate(ctx.Request.Context(), token)
	if errors.Is(err, identity.ErrUnauthenticated) {
		writeMenuUnauthenticated(ctx)
		return 0, false
	}
	if err != nil || userID == 0 {
		writeMenuUnavailable(ctx)
		return 0, false
	}
	return userID, true
}

func (handler *Handler) projectCategories(ctx context.Context, categories []Category, userID uint64) ([]categoryResponse, bool) {
	products := make([]Product, 0)
	for _, category := range categories {
		products = append(products, category.Products...)
	}
	originals := make([]uint32, len(products))
	for index, product := range products {
		if !product.valid() {
			return nil, false
		}
		originals[index] = product.OriginalUnitPriceCents
	}
	staffPrices := make([]*uint32, len(products))
	if userID != 0 {
		if handler.prices == nil {
			return nil, false
		}
		resolved, err := handler.prices.ResolvePrices(ctx, userID, originals)
		if err != nil || len(resolved) != len(products) {
			return nil, false
		}
		staffPrices = resolved
	}
	result := make([]categoryResponse, 0, len(categories))
	productIndex := 0
	for _, category := range categories {
		item := categoryResponse{ID: strconv.FormatUint(category.ID, 10), Name: category.Name, Products: make([]productResponse, 0, len(category.Products))}
		for _, product := range category.Products {
			staffPrice := staffPrices[productIndex]
			if staffPrice != nil && *staffPrice > product.OriginalUnitPriceCents {
				return nil, false
			}
			images := make([]imageResponse, 0, len(product.ImageObjectKeys))
			for _, key := range product.ImageObjectKeys {
				if handler.urls == nil {
					return nil, false
				}
				url, err := handler.urls.PublicURL(ctx, key)
				if err != nil || url == "" {
					return nil, false
				}
				images = append(images, imageResponse{ObjectKey: key, URL: url})
			}
			item.Products = append(item.Products, productResponse{
				ID: strconv.FormatUint(product.ID, 10), CategoryID: strconv.FormatUint(product.CategoryID, 10),
				Name: product.Name, Description: product.Description, Specification: product.Specification,
				MealPeriod: product.MealPeriod, Images: images, Listed: product.Listed, SoldOut: product.SoldOut,
				OriginalUnitPriceCents: product.OriginalUnitPriceCents, StaffUnitPriceCents: staffPrice,
			})
			productIndex++
		}
		result = append(result, item)
	}
	return result, true
}

func filterMealCategories(categories []Category, meal MealCode) []Category {
	result := make([]Category, 0, len(categories))
	for _, category := range categories {
		item := Category{ID: category.ID, Name: category.Name, Products: make([]Product, 0, len(category.Products))}
		for _, product := range category.Products {
			if product.MealPeriod == "all" || product.MealPeriod == string(meal) {
				item.Products = append(item.Products, product)
			}
		}
		if len(item.Products) > 0 {
			result = append(result, item)
		}
	}
	return result
}

func validBusinessStatus(value string) bool {
	return value == "open" || value == "closed" || value == "cutoff"
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
	if len(value) != 10 || value[4] != '-' || value[7] != '-' {
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
func writeMenuUnauthenticated(ctx *gin.Context) {
	ctx.JSON(http.StatusUnauthorized, menuErrorEnvelope{Error: menuErrorResponse{Code: "UNAUTHENTICATED", Message: "authentication required"}})
}
