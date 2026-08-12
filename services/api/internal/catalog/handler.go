package catalog

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	productNotFoundCode       = "PRODUCT_NOT_FOUND"
	productNotFoundMessage    = "product not found"
	catalogUnavailableCode    = "CATALOG_UNAVAILABLE"
	catalogUnavailableMessage = "catalog temporarily unavailable"
)

// Reader is the narrow query boundary needed by the public catalog routes.
type Reader interface {
	List(context.Context) ([]Category, error)
	GetProduct(context.Context, uint64) (Product, error)
}

// Handler serves the anonymous catalog read API.
type Handler struct {
	reader Reader
}

// NewHandler constructs a catalog HTTP handler.
func NewHandler(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// RegisterRoutes adds only the two anonymous GET catalog routes.
func (handler *Handler) RegisterRoutes(engine *gin.Engine) {
	engine.GET("/api/v1/catalog", handler.list)
	engine.GET("/api/v1/catalog/products/:id", handler.detail)
}

type catalogResponse struct {
	Categories []categoryResponse `json:"categories"`
}

type categoryResponse struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Products []productResponse `json:"products"`
}

type productEnvelope struct {
	Product productResponse `json:"product"`
}

type productResponse struct {
	ID            string `json:"id"`
	CategoryID    string `json:"category_id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Specification string `json:"specification"`
	PriceCents    uint32 `json:"price_cents"`
}

type errorEnvelope struct {
	Error errorResponse `json:"error"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (handler *Handler) list(ctx *gin.Context) {
	categories, err := handler.reader.List(ctx.Request.Context())
	if err != nil {
		writeCatalogUnavailable(ctx)
		return
	}
	response := catalogResponse{Categories: make([]categoryResponse, 0, len(categories))}
	for _, category := range categories {
		item := categoryResponse{
			ID: strconv.FormatUint(category.ID, 10), Name: category.Name,
			Products: make([]productResponse, 0, len(category.Products)),
		}
		for _, product := range category.Products {
			item.Products = append(item.Products, newProductResponse(product))
		}
		response.Categories = append(response.Categories, item)
	}
	ctx.JSON(http.StatusOK, response)
}

func (handler *Handler) detail(ctx *gin.Context) {
	id, ok := parseProductID(ctx.Param("id"))
	if !ok {
		writeProductNotFound(ctx)
		return
	}
	product, err := handler.reader.GetProduct(ctx.Request.Context(), id)
	if errors.Is(err, ErrProductNotFound) {
		writeProductNotFound(ctx)
		return
	}
	if err != nil {
		writeCatalogUnavailable(ctx)
		return
	}
	ctx.JSON(http.StatusOK, productEnvelope{Product: newProductResponse(product)})
}

func parseProductID(value string) (uint64, bool) {
	if value == "" {
		return 0, false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return 0, false
		}
	}
	id, err := strconv.ParseUint(value, 10, 64)
	return id, err == nil && id > 0
}

func newProductResponse(product Product) productResponse {
	return productResponse{
		ID: strconv.FormatUint(product.ID, 10), CategoryID: strconv.FormatUint(product.CategoryID, 10),
		Name: product.Name, Description: product.Description, Specification: product.Specification, PriceCents: product.PriceCents,
	}
}

func writeProductNotFound(ctx *gin.Context) {
	ctx.JSON(http.StatusNotFound, errorEnvelope{Error: errorResponse{Code: productNotFoundCode, Message: productNotFoundMessage}})
}

func writeCatalogUnavailable(ctx *gin.Context) {
	ctx.JSON(http.StatusServiceUnavailable, errorEnvelope{Error: errorResponse{Code: catalogUnavailableCode, Message: catalogUnavailableMessage}})
}
