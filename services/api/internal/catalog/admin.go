package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	ErrAdminInvalidInput        = errors.New("catalog invalid input")
	ErrAdminNotFound            = errors.New("catalog not found")
	ErrAdminConflict            = errors.New("catalog conflict")
	ErrAdminIdempotencyConflict = errors.New("catalog idempotency conflict")
	ErrAdminUnavailable         = errors.New("catalog unavailable")
)

// WriteMeta is the authenticated command receipt identity supplied by root composition.
type WriteMeta struct {
	ActorUserID    uint64
	IdempotencyKey string
	RequestID      string
}

type AdminCategory struct {
	ID           uint64
	Name         string
	SortOrder    uint32
	Enabled      bool
	ProductCount uint32
}

type AdminImage struct {
	ObjectKey string
	URL       string
	SortOrder uint8
}

type AdminProduct struct {
	ID, CategoryID    uint64
	Name, Description string
	CategoryName      string
	MealPeriod        string
	Images            []AdminImage
	PriceCents        uint32
	Listed, SoldOut   bool
}

type AdminQuery struct {
	ServiceDate string
}

type CatalogCommandKind string

const (
	CommandCreateCategory  CatalogCommandKind = "CREATE_CATEGORY"
	CommandUpdateCategory  CatalogCommandKind = "UPDATE_CATEGORY"
	CommandDeleteCategory  CatalogCommandKind = "DELETE_CATEGORY"
	CommandReorderCategory CatalogCommandKind = "REORDER_CATEGORIES"
	CommandCreateProduct   CatalogCommandKind = "CREATE_PRODUCT"
	CommandUpdateProduct   CatalogCommandKind = "UPDATE_PRODUCT"
	CommandDeleteProduct   CatalogCommandKind = "DELETE_PRODUCT"
	CommandProductStatus   CatalogCommandKind = "SET_PRODUCT_STATUS"
	CommandProductSoldOut  CatalogCommandKind = "SET_PRODUCT_SOLD_OUT"
	CommandReorderProducts CatalogCommandKind = "REORDER_PRODUCTS"
)

type CatalogCommand struct {
	Kind                  CatalogCommandKind
	CategoryID, ProductID uint64
	Name, Description     string
	PriceCents            uint32
	MealPeriod            string
	Images                []AdminImage
	Enabled, Listed       *bool
	ServiceDate           string
	SoldOut               *bool
	OrderedIDs            []uint64
}

type CatalogResult struct {
	Category *AdminCategory
	Product  *AdminProduct
}

// AdminApplication is the deep PC configuration seam. MySQL locking, receipts and
// last-write validation remain inside the implementation supplied at composition.
type AdminApplication interface {
	ListCategories(context.Context) ([]AdminCategory, error)
	ListProducts(context.Context, AdminQuery) ([]AdminProduct, error)
	GetProduct(context.Context, uint64, AdminQuery) (AdminProduct, error)
	Execute(context.Context, WriteMeta, CatalogCommand) (CatalogResult, error)
}

type AdminHandler struct{ app AdminApplication }

func NewAdminHandler(app AdminApplication) *AdminHandler { return &AdminHandler{app: app} }

// RegisterRoutes mounts feature-owned PC catalog routes on an authenticated owner group.
func (h *AdminHandler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/categories", h.categories)
	group.POST("/categories", h.createCategory)
	group.PUT("/categories/:id", h.updateCategory)
	group.DELETE("/categories/:id", h.deleteCategory)
	group.PUT("/categories/order", h.reorderCategories)
	group.GET("/products", h.products)
	group.GET("/products/:id", h.product)
	group.POST("/products", h.createProduct)
	group.PUT("/products/:id", h.updateProduct)
	group.DELETE("/products/:id", h.deleteProduct)
	group.PUT("/products/:id/status", h.productStatus)
	group.PUT("/products/:id/soldout", h.productSoldOut)
	group.PUT("/products/order", h.reorderProducts)
}

type categoryDTO struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SortOrder    uint32 `json:"sort_order"`
	Enabled      bool   `json:"enabled"`
	ProductCount uint32 `json:"product_count"`
}

type imageDTO struct {
	ObjectKey string `json:"object_key"`
	URL       string `json:"url"`
	SortOrder uint8  `json:"sort_order"`
}
type productDTO struct {
	ID           string     `json:"id"`
	CategoryID   string     `json:"category_id"`
	CategoryName string     `json:"category_name"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	MealPeriod   string     `json:"meal_period"`
	Images       []imageDTO `json:"images"`
	PriceCents   uint32     `json:"price_cents"`
	Listed       bool       `json:"listed"`
	SoldOut      bool       `json:"sold_out"`
}

func categoryView(c AdminCategory) categoryDTO {
	return categoryDTO{strconv.FormatUint(c.ID, 10), c.Name, c.SortOrder, c.Enabled, c.ProductCount}
}
func productView(p AdminProduct) productDTO {
	images := make([]imageDTO, 0, len(p.Images))
	for _, image := range p.Images {
		images = append(images, imageDTO{image.ObjectKey, image.URL, image.SortOrder})
	}
	return productDTO{strconv.FormatUint(p.ID, 10), strconv.FormatUint(p.CategoryID, 10), p.CategoryName, p.Name, p.Description, p.MealPeriod, images, p.PriceCents, p.Listed, p.SoldOut}
}

func (h *AdminHandler) categories(c *gin.Context) {
	items, err := h.app.ListCategories(c.Request.Context())
	if err != nil {
		writeAdminError(c, err)
		return
	}
	out := make([]categoryDTO, 0, len(items))
	for _, item := range items {
		out = append(out, categoryView(item))
	}
	c.JSON(http.StatusOK, gin.H{"categories": out})
}
func (h *AdminHandler) products(c *gin.Context) {
	items, err := h.app.ListProducts(c.Request.Context(), AdminQuery{ServiceDate: c.Query("service_date")})
	if err != nil {
		writeAdminError(c, err)
		return
	}
	out := make([]productDTO, 0, len(items))
	for _, item := range items {
		out = append(out, productView(item))
	}
	c.JSON(http.StatusOK, gin.H{"products": out})
}
func (h *AdminHandler) product(c *gin.Context) {
	id, ok := adminID(c.Param("id"))
	if !ok {
		writeAdminError(c, ErrAdminNotFound)
		return
	}
	p, err := h.app.GetProduct(c.Request.Context(), id, AdminQuery{ServiceDate: c.Query("service_date")})
	if err != nil {
		writeAdminError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product": productView(p)})
}

type categoryWrite struct {
	Name    string `json:"name"`
	Enabled *bool  `json:"enabled"`
}
type orderWrite struct {
	IDs        []string `json:"ids"`
	CategoryID string   `json:"category_id"`
}
type productWrite struct {
	Name        string     `json:"name"`
	PriceCents  uint32     `json:"price_cents"`
	CategoryID  string     `json:"category_id"`
	MealPeriod  string     `json:"meal_period"`
	Description string     `json:"description"`
	Images      []imageDTO `json:"images"`
}
type statusWrite struct {
	Status string `json:"status"`
}
type soldOutWrite struct {
	ServiceDate string `json:"service_date"`
	SoldOut     *bool  `json:"sold_out"`
}

func (h *AdminHandler) createCategory(c *gin.Context) {
	var in categoryWrite
	if !strictJSON(c, &in) || strings.TrimSpace(in.Name) == "" {
		writeAdminError(c, ErrAdminInvalidInput)
		return
	}
	h.command(c, CatalogCommand{Kind: CommandCreateCategory, Name: strings.TrimSpace(in.Name)}, http.StatusCreated)
}
func (h *AdminHandler) updateCategory(c *gin.Context) {
	id, ok := adminID(c.Param("id"))
	var in categoryWrite
	if !ok || !strictJSON(c, &in) {
		writeAdminError(c, ErrAdminInvalidInput)
		return
	}
	h.command(c, CatalogCommand{Kind: CommandUpdateCategory, CategoryID: id, Name: strings.TrimSpace(in.Name), Enabled: in.Enabled}, http.StatusOK)
}
func (h *AdminHandler) deleteCategory(c *gin.Context) {
	id, ok := adminID(c.Param("id"))
	if !ok {
		writeAdminError(c, ErrAdminNotFound)
		return
	}
	h.command(c, CatalogCommand{Kind: CommandDeleteCategory, CategoryID: id}, http.StatusOK)
}
func (h *AdminHandler) reorderCategories(c *gin.Context) {
	var in orderWrite
	if !strictJSON(c, &in) {
		writeAdminError(c, ErrAdminInvalidInput)
		return
	}
	ids, ok := adminIDs(in.IDs)
	if !ok {
		writeAdminError(c, ErrAdminInvalidInput)
		return
	}
	h.command(c, CatalogCommand{Kind: CommandReorderCategory, OrderedIDs: ids}, http.StatusOK)
}
func (h *AdminHandler) createProduct(c *gin.Context) {
	h.productWrite(c, 0, CommandCreateProduct, http.StatusCreated)
}
func (h *AdminHandler) updateProduct(c *gin.Context) {
	id, ok := adminID(c.Param("id"))
	if !ok {
		writeAdminError(c, ErrAdminNotFound)
		return
	}
	h.productWrite(c, id, CommandUpdateProduct, http.StatusOK)
}
func (h *AdminHandler) productWrite(c *gin.Context, id uint64, kind CatalogCommandKind, status int) {
	var in productWrite
	if !strictJSON(c, &in) {
		writeAdminError(c, ErrAdminInvalidInput)
		return
	}
	categoryID, ok := adminID(in.CategoryID)
	if !ok || strings.TrimSpace(in.Name) == "" || in.PriceCents == 0 || (in.MealPeriod != "all" && in.MealPeriod != "lunch" && in.MealPeriod != "dinner") || len(in.Images) > 3 {
		writeAdminError(c, ErrAdminInvalidInput)
		return
	}
	images := make([]AdminImage, 0, len(in.Images))
	for i, image := range in.Images {
		if image.ObjectKey == "" {
			writeAdminError(c, ErrAdminInvalidInput)
			return
		}
		images = append(images, AdminImage{ObjectKey: image.ObjectKey, SortOrder: uint8(i)})
	}
	h.command(c, CatalogCommand{Kind: kind, ProductID: id, CategoryID: categoryID, Name: strings.TrimSpace(in.Name), Description: in.Description, PriceCents: in.PriceCents, MealPeriod: in.MealPeriod, Images: images}, status)
}
func (h *AdminHandler) deleteProduct(c *gin.Context) {
	id, ok := adminID(c.Param("id"))
	if !ok {
		writeAdminError(c, ErrAdminNotFound)
		return
	}
	h.command(c, CatalogCommand{Kind: CommandDeleteProduct, ProductID: id}, http.StatusOK)
}
func (h *AdminHandler) productStatus(c *gin.Context) {
	id, ok := adminID(c.Param("id"))
	var in statusWrite
	if !ok || !strictJSON(c, &in) || (in.Status != "ON" && in.Status != "OFF") {
		writeAdminError(c, ErrAdminInvalidInput)
		return
	}
	listed := in.Status == "ON"
	h.command(c, CatalogCommand{Kind: CommandProductStatus, ProductID: id, Listed: &listed}, http.StatusOK)
}
func (h *AdminHandler) productSoldOut(c *gin.Context) {
	id, ok := adminID(c.Param("id"))
	var in soldOutWrite
	if !ok || !strictJSON(c, &in) || in.ServiceDate == "" || in.SoldOut == nil {
		writeAdminError(c, ErrAdminInvalidInput)
		return
	}
	h.command(c, CatalogCommand{Kind: CommandProductSoldOut, ProductID: id, ServiceDate: in.ServiceDate, SoldOut: in.SoldOut}, http.StatusOK)
}
func (h *AdminHandler) reorderProducts(c *gin.Context) {
	var in orderWrite
	if !strictJSON(c, &in) {
		writeAdminError(c, ErrAdminInvalidInput)
		return
	}
	categoryID, ok := adminID(in.CategoryID)
	ids, idsOK := adminIDs(in.IDs)
	if !ok || !idsOK {
		writeAdminError(c, ErrAdminInvalidInput)
		return
	}
	h.command(c, CatalogCommand{Kind: CommandReorderProducts, CategoryID: categoryID, OrderedIDs: ids}, http.StatusOK)
}
func (h *AdminHandler) command(c *gin.Context, command CatalogCommand, status int) {
	meta, ok := adminWriteMeta(c)
	if !ok {
		writeAdminError(c, ErrAdminInvalidInput)
		return
	}
	result, err := h.app.Execute(c.Request.Context(), meta, command)
	if err != nil {
		writeAdminError(c, err)
		return
	}
	if result.Product != nil {
		c.JSON(status, gin.H{"product": productView(*result.Product)})
		return
	}
	if result.Category != nil {
		c.JSON(status, gin.H{"category": categoryView(*result.Category)})
		return
	}
	c.JSON(status, gin.H{"ok": true})
}

func strictJSON(c *gin.Context, out any) bool {
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
func adminID(value string) (uint64, bool) {
	id, err := strconv.ParseUint(value, 10, 64)
	return id, err == nil && id > 0
}
func adminIDs(values []string) ([]uint64, bool) {
	if len(values) == 0 {
		return nil, false
	}
	out := make([]uint64, 0, len(values))
	seen := map[uint64]bool{}
	for _, v := range values {
		id, ok := adminID(v)
		if !ok || seen[id] {
			return nil, false
		}
		seen[id] = true
		out = append(out, id)
	}
	return out, true
}
func adminWriteMeta(c *gin.Context) (WriteMeta, bool) {
	actor := c.GetUint64("actor_user_id")
	keys := c.Request.Header.Values("Idempotency-Key")
	if actor == 0 || len(keys) != 1 || strings.TrimSpace(keys[0]) == "" || strings.ContainsAny(keys[0], " \t\r\n") {
		return WriteMeta{}, false
	}
	return WriteMeta{ActorUserID: actor, IdempotencyKey: keys[0], RequestID: c.GetString("request_id")}, true
}
func writeAdminError(c *gin.Context, err error) {
	status, code, message := http.StatusServiceUnavailable, "CATALOG_UNAVAILABLE", "catalog temporarily unavailable"
	switch {
	case errors.Is(err, ErrAdminInvalidInput):
		status, code, message = http.StatusBadRequest, "INVALID_REQUEST", "invalid request"
	case errors.Is(err, ErrAdminNotFound):
		status, code, message = http.StatusNotFound, "NOT_FOUND", "resource not found"
	case errors.Is(err, ErrAdminConflict):
		status, code, message = http.StatusConflict, "CATALOG_CONFLICT", "catalog conflict"
	case errors.Is(err, ErrAdminIdempotencyConflict):
		status, code, message = http.StatusConflict, "IDEMPOTENCY_CONFLICT", "idempotency conflict"
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
