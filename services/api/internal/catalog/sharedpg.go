package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/gaofeng30/order/services/api/internal/menu"
)

type sharedPGExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}

type SharedPGRepository struct{ client sharedPGExecutor }

func NewSharedPGRepository(client sharedPGExecutor) *SharedPGRepository {
	return &SharedPGRepository{client: client}
}

var errSharedCatalog = errors.New("shared catalog unavailable")

type sharedProductFacts struct {
	ID            string          `json:"id"`
	CategoryID    string          `json:"category_id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Specification string          `json:"specification"`
	MealPeriod    string          `json:"meal_period"`
	Images        json.RawMessage `json:"images"`
	Listed        bool            `json:"listed"`
	SoldOut       bool            `json:"sold_out"`
	Price         uint32          `json:"price_cents"`
}

func sharedCatalogID(raw string) (uint64, bool) {
	id, err := strconv.ParseUint(raw, 10, 64)
	return id, err == nil && id > 0 && strconv.FormatUint(id, 10) == raw
}

func (f sharedProductFacts) product() (Product, bool) {
	id, idOK := sharedCatalogID(f.ID)
	category, categoryOK := sharedCatalogID(f.CategoryID)
	p := Product{ID: id, CategoryID: category, Name: f.Name, Description: f.Description, Specification: f.Specification, MealPeriod: f.MealPeriod, Listed: f.Listed, SoldOut: f.SoldOut, OriginalUnitPriceCents: f.Price}
	return p, idOK && categoryOK && decodeImageKeys(f.Images, &p.ImageObjectKeys) && p.valid() && p.Listed
}

func (r *SharedPGRepository) Browse(ctx context.Context) ([]Category, error) {
	if r == nil || r.client == nil || ctx == nil {
		return nil, errSharedCatalog
	}
	var response struct {
		Outcome    string `json:"outcome"`
		Categories []struct {
			ID       string               `json:"id"`
			Name     string               `json:"name"`
			Products []sharedProductFacts `json:"products"`
		} `json:"categories"`
	}
	if r.client.CallRPC(ctx, "order_catalog_browse_v1", map[string]any{}, &response) != nil || response.Outcome != "SUCCEEDED" || response.Categories == nil {
		return nil, errSharedCatalog
	}
	result := make([]Category, 0, len(response.Categories))
	seen := make(map[uint64]bool)
	for _, facts := range response.Categories {
		id, ok := sharedCatalogID(facts.ID)
		if !ok || seen[id] || !validCatalogText(facts.Name, true) || facts.Products == nil {
			return nil, errSharedCatalog
		}
		seen[id] = true
		category := Category{ID: id, Name: facts.Name, Products: make([]Product, 0, len(facts.Products))}
		for _, pf := range facts.Products {
			p, ok := pf.product()
			if !ok || p.CategoryID != id {
				return nil, errSharedCatalog
			}
			category.Products = append(category.Products, p)
		}
		result = append(result, category)
	}
	return result, nil
}

func (r *SharedPGRepository) Detail(ctx context.Context, id uint64, serviceDate string) (Product, CurrentFacts, error) {
	if r == nil || r.client == nil || ctx == nil {
		return Product{}, CurrentFacts{}, errSharedCatalog
	}
	var response struct {
		Outcome        string                  `json:"outcome"`
		Product        sharedProductFacts      `json:"product"`
		BusinessStatus string                  `json:"business_status"`
		Date           *string                 `json:"service_date"`
		Open           *bool                   `json:"is_open"`
		MealPeriods    []menu.MealPeriodRecord `json:"meal_periods"`
	}
	if r.client.CallRPC(ctx, "order_catalog_detail_v1", map[string]any{"p_product": strconv.FormatUint(id, 10), "p_service_date": serviceDate}, &response) != nil {
		return Product{}, CurrentFacts{}, errSharedCatalog
	}
	if response.Outcome == "NOT_FOUND" {
		return Product{}, CurrentFacts{}, ErrProductNotFound
	}
	p, ok := response.Product.product()
	if response.Outcome != "SUCCEEDED" || !ok || (response.Date == nil) != (response.Open == nil) || (response.Date != nil && *response.Date != serviceDate) || response.MealPeriods == nil {
		return Product{}, CurrentFacts{}, errSharedCatalog
	}
	facts := CurrentFacts{BusinessStatus: response.BusinessStatus, ServiceDatePresent: response.Date != nil, ServiceDateOpen: response.Open != nil && *response.Open, MealPeriods: response.MealPeriods}
	return p, facts, nil
}
