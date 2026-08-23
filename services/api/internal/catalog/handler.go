package catalog

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gaofeng30/order/services/api/internal/httpdto"
	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/menu"
	"github.com/gin-gonic/gin"
)

const (
	productNotFoundCode       = "PRODUCT_NOT_FOUND"
	productNotFoundMessage    = "product not found"
	catalogUnavailableCode    = "CATALOG_UNAVAILABLE"
	catalogUnavailableMessage = "catalog temporarily unavailable"
	unauthenticatedCode       = "UNAUTHENTICATED"
	unauthenticatedMessage    = "authentication required"
)

var catalogLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

type Reader interface {
	Browse(context.Context) ([]Category, error)
	Detail(context.Context, uint64, string) (Product, CurrentFacts, error)
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
	auth   Authenticator
	prices PricingResolver
	urls   PublicURLer
	now    func() time.Time
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
func WithClock(value func() time.Time) HandlerOption {
	return func(handler *Handler) { handler.now = value }
}

func NewHandler(reader Reader, options ...HandlerOption) *Handler {
	handler := &Handler{reader: reader, now: time.Now}
	for _, option := range options {
		if option != nil {
			option(handler)
		}
	}
	return handler
}

func (handler *Handler) RegisterRoutes(engine *gin.Engine) {
	engine.GET("/api/v1/catalog", handler.list)
	engine.GET("/api/v1/catalog/products/:id", handler.detail)
}

type catalogResponse struct {
	Categories []categoryResponse `json:"categories"`
}
type categoryResponse struct {
	ID       string                   `json:"id"`
	Name     string                   `json:"name"`
	Products []catalogProductResponse `json:"products"`
}
type productEnvelope struct {
	Product detailProductResponse `json:"product"`
}
type imageResponse struct {
	ObjectKey string `json:"object_key"`
	URL       string `json:"url"`
}
type catalogProductResponse struct {
	ID                     string          `json:"id"`
	CategoryID             string          `json:"category_id"`
	Name                   string          `json:"name"`
	Description            string          `json:"description"`
	Specification          string          `json:"specification"`
	MealPeriod             string          `json:"meal_period"`
	Images                 []imageResponse `json:"images"`
	Listed                 bool            `json:"listed"`
	OriginalUnitPriceCents uint32          `json:"original_unit_price_cents"`
	StaffUnitPriceCents    *uint32         `json:"staff_unit_price_cents,omitempty"`
}
type detailProductResponse struct {
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
type errorEnvelope struct {
	Error errorResponse `json:"error"`
}
type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (handler *Handler) list(ctx *gin.Context) {
	userID, authOK := handler.optionalUser(ctx)
	if !authOK {
		return
	}
	categories, err := handler.reader.Browse(ctx.Request.Context())
	if err != nil {
		writeCatalogUnavailable(ctx)
		return
	}
	products := flattenProducts(categories)
	views, ok := handler.projectProducts(ctx.Request.Context(), products, userID)
	if !ok {
		writeCatalogUnavailable(ctx)
		return
	}
	response := catalogResponse{Categories: make([]categoryResponse, 0, len(categories))}
	viewIndex := 0
	for _, category := range categories {
		item := categoryResponse{ID: strconv.FormatUint(category.ID, 10), Name: category.Name, Products: make([]catalogProductResponse, 0, len(category.Products))}
		for range category.Products {
			view := views[viewIndex]
			item.Products = append(item.Products, catalogProductResponse{
				ID: view.ID, CategoryID: view.CategoryID, Name: view.Name, Description: view.Description,
				Specification: view.Specification, MealPeriod: view.MealPeriod, Images: view.Images, Listed: view.Listed,
				OriginalUnitPriceCents: view.OriginalUnitPriceCents, StaffUnitPriceCents: view.StaffUnitPriceCents,
			})
			viewIndex++
		}
		response.Categories = append(response.Categories, item)
	}
	ctx.JSON(http.StatusOK, response)
}

func (handler *Handler) detail(ctx *gin.Context) {
	userID, authOK := handler.optionalUser(ctx)
	if !authOK {
		return
	}
	id, ok := parseProductID(ctx.Param("id"))
	if !ok {
		writeProductNotFound(ctx)
		return
	}
	now := handler.now().In(catalogLocation)
	dateValue, pickupValue, serviceDate, ok := parseDetailSelection(ctx, now)
	if !ok {
		writeInvalidSelection(ctx)
		return
	}
	product, facts, err := handler.reader.Detail(ctx.Request.Context(), id, dateValue)
	if errors.Is(err, ErrProductNotFound) {
		writeProductNotFound(ctx)
		return
	}
	if err != nil || facts.BusinessStatus != "open" || !facts.ServiceDatePresent || !facts.ServiceDateOpen {
		writeCatalogUnavailable(ctx)
		return
	}
	selection, err := menu.ResolveMeal(facts.MealPeriods, serviceDate, pickupValue, now)
	if errors.Is(err, menu.ErrInvalidSelection) {
		writeInvalidSelection(ctx)
		return
	}
	if err != nil || !selection.Orderable {
		writeCatalogUnavailable(ctx)
		return
	}
	if product.MealPeriod != "all" && product.MealPeriod != string(selection.Code) {
		writeProductNotFound(ctx)
		return
	}
	views, projected := handler.projectProducts(ctx.Request.Context(), []Product{product}, userID)
	if !projected {
		writeCatalogUnavailable(ctx)
		return
	}
	view := views[0]
	ctx.JSON(http.StatusOK, productEnvelope{Product: detailProductResponse{
		ID: view.ID, CategoryID: view.CategoryID, Name: view.Name, Description: view.Description,
		Specification: view.Specification, MealPeriod: view.MealPeriod, Images: view.Images, Listed: view.Listed,
		SoldOut: product.SoldOut, OriginalUnitPriceCents: view.OriginalUnitPriceCents, StaffUnitPriceCents: view.StaffUnitPriceCents,
	}})
}

func (handler *Handler) optionalUser(ctx *gin.Context) (uint64, bool) {
	if len(ctx.Request.Header.Values("Authorization")) == 0 {
		return 0, true
	}
	token, err := httpdto.BearerToken(ctx.Request)
	if err != nil {
		writeUnauthenticated(ctx)
		return 0, false
	}
	if handler.auth == nil {
		writeCatalogUnavailable(ctx)
		return 0, false
	}
	userID, err := handler.auth.Authenticate(ctx.Request.Context(), token)
	if errors.Is(err, identity.ErrUnauthenticated) {
		writeUnauthenticated(ctx)
		return 0, false
	}
	if err != nil || userID == 0 {
		writeCatalogUnavailable(ctx)
		return 0, false
	}
	return userID, true
}

func (handler *Handler) projectProducts(ctx context.Context, products []Product, userID uint64) ([]productProjection, bool) {
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
	result := make([]productProjection, 0, len(products))
	for index, product := range products {
		if staffPrices[index] != nil && *staffPrices[index] > product.OriginalUnitPriceCents {
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
		result = append(result, productProjection{
			ID: strconv.FormatUint(product.ID, 10), CategoryID: strconv.FormatUint(product.CategoryID, 10), Name: product.Name,
			Description: product.Description, Specification: product.Specification, MealPeriod: product.MealPeriod,
			Images: images, Listed: product.Listed, OriginalUnitPriceCents: product.OriginalUnitPriceCents,
			StaffUnitPriceCents: staffPrices[index],
		})
	}
	return result, true
}

type productProjection struct {
	ID, CategoryID, Name, Description, Specification, MealPeriod string
	Images                                                       []imageResponse
	Listed                                                       bool
	OriginalUnitPriceCents                                       uint32
	StaffUnitPriceCents                                          *uint32
}

func flattenProducts(categories []Category) []Product {
	result := make([]Product, 0)
	for _, category := range categories {
		result = append(result, category.Products...)
	}
	return result
}

func parseDetailSelection(ctx *gin.Context, now time.Time) (string, string, time.Time, bool) {
	query := ctx.Request.URL.Query()
	dates, dateExists := query["date"]
	times, timeExists := query["time"]
	if !dateExists || !timeExists || len(dates) != 1 || len(times) != 1 || !strictDate(dates[0]) || !strictMinute(times[0]) {
		return "", "", time.Time{}, false
	}
	date, err := time.ParseInLocation("2006-01-02", dates[0], catalogLocation)
	if err != nil {
		return "", "", time.Time{}, false
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, catalogLocation)
	if !date.Equal(today) && !date.Equal(today.AddDate(0, 0, 1)) {
		return "", "", time.Time{}, false
	}
	return dates[0], times[0], date, true
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

func strictMinute(value string) bool {
	if len(value) != 5 || value[2] != ':' {
		return false
	}
	hour, hourErr := strconv.Atoi(value[:2])
	minute, minuteErr := strconv.Atoi(value[3:])
	return hourErr == nil && minuteErr == nil && hour >= 0 && hour < 24 && minute >= 0 && minute < 60
}

func parseProductID(value string) (uint64, bool) {
	if value == "" {
		return 0, false
	}
	for index := range value {
		if value[index] < '0' || value[index] > '9' {
			return 0, false
		}
	}
	id, err := strconv.ParseUint(value, 10, 64)
	return id, err == nil && id > 0
}

func writeProductNotFound(ctx *gin.Context) {
	ctx.JSON(http.StatusNotFound, errorEnvelope{Error: errorResponse{Code: productNotFoundCode, Message: productNotFoundMessage}})
}
func writeCatalogUnavailable(ctx *gin.Context) {
	ctx.JSON(http.StatusServiceUnavailable, errorEnvelope{Error: errorResponse{Code: catalogUnavailableCode, Message: catalogUnavailableMessage}})
}
func writeInvalidSelection(ctx *gin.Context) {
	ctx.JSON(http.StatusBadRequest, errorEnvelope{Error: errorResponse{Code: "INVALID_MENU_SELECTION", Message: "invalid menu selection"}})
}
func writeUnauthenticated(ctx *gin.Context) {
	ctx.JSON(http.StatusUnauthorized, errorEnvelope{Error: errorResponse{Code: unauthenticatedCode, Message: unauthenticatedMessage}})
}
