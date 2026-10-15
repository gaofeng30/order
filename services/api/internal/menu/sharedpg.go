package menu

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
)

type sharedPGExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}

type SharedPGRepository struct{ client sharedPGExecutor }

func NewSharedPGRepository(client sharedPGExecutor) *SharedPGRepository {
	return &SharedPGRepository{client: client}
}

var errSharedMenu = errors.New("shared menu unavailable")

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

func sharedMenuID(raw string) (uint64, bool) {
	id, err := strconv.ParseUint(raw, 10, 64)
	return id, err == nil && id > 0 && strconv.FormatUint(id, 10) == raw
}

func (f sharedProductFacts) product() (Product, bool) {
	id, idOK := sharedMenuID(f.ID)
	category, categoryOK := sharedMenuID(f.CategoryID)
	p := Product{ID: id, CategoryID: category, Name: f.Name, Description: f.Description, Specification: f.Specification, MealPeriod: f.MealPeriod, Listed: f.Listed, SoldOut: f.SoldOut, OriginalUnitPriceCents: f.Price}
	return p, idOK && categoryOK && decodeMenuImages(f.Images, &p.ImageObjectKeys) && p.valid() && p.Listed
}

type sharedMenuResponse struct {
	Outcome        string             `json:"outcome"`
	BusinessStatus string             `json:"business_status"`
	Date           *string            `json:"service_date"`
	Open           *bool              `json:"is_open"`
	MealPeriods    []MealPeriodRecord `json:"meal_periods"`
	Categories     []struct {
		ID       string               `json:"id"`
		Name     string               `json:"name"`
		Products []sharedProductFacts `json:"products"`
	} `json:"categories"`
}

func (r *SharedPGRepository) ReadMenu(ctx context.Context, serviceDate string) (MenuSnapshot, error) {
	if r == nil || r.client == nil || ctx == nil {
		return MenuSnapshot{}, errSharedMenu
	}
	var f sharedMenuResponse
	if r.client.CallRPC(ctx, "order_menu_snapshot_v1", map[string]any{"p_service_date": serviceDate}, &f) != nil || f.Outcome != "SUCCEEDED" || f.MealPeriods == nil || f.Categories == nil || (f.Date == nil) != (f.Open == nil) || (f.Date != nil && *f.Date != serviceDate) {
		return MenuSnapshot{}, errSharedMenu
	}
	result := MenuSnapshot{BusinessStatus: f.BusinessStatus, ServiceDatePresent: f.Date != nil, ServiceDateOpen: f.Open != nil && *f.Open, MealPeriods: f.MealPeriods, Categories: make([]Category, 0, len(f.Categories))}
	seen := map[uint64]bool{}
	for _, cf := range f.Categories {
		id, ok := sharedMenuID(cf.ID)
		if !ok || seen[id] || len(cf.Products) == 0 {
			return MenuSnapshot{}, errSharedMenu
		}
		seen[id] = true
		category := Category{ID: id, Name: cf.Name, Products: make([]Product, 0, len(cf.Products))}
		for _, pf := range cf.Products {
			p, ok := pf.product()
			if !ok || p.CategoryID != id {
				return MenuSnapshot{}, errSharedMenu
			}
			category.Products = append(category.Products, p)
		}
		result.Categories = append(result.Categories, category)
	}
	return result, nil
}
func (r *SharedPGRepository) ReadPickupFacts(ctx context.Context, dates []string) (PickupFacts, error) {
	if r == nil || r.client == nil || ctx == nil || len(dates) != 2 || dates[0] == dates[1] {
		return PickupFacts{}, errSharedMenu
	}
	var f struct {
		Outcome        string             `json:"outcome"`
		BusinessStatus string             `json:"business_status"`
		MealPeriods    []MealPeriodRecord `json:"meal_periods"`
		Dates          []struct {
			Date           string     `json:"date"`
			Open           bool       `json:"is_open"`
			AvailableMeals []MealCode `json:"available_meals"`
		} `json:"service_dates"`
	}
	if r.client.CallRPC(ctx, "order_menu_pickup_facts_v1", map[string]any{"p_date_a": dates[0], "p_date_b": dates[1]}, &f) != nil || f.Outcome != "SUCCEEDED" || f.MealPeriods == nil || f.Dates == nil {
		return PickupFacts{}, errSharedMenu
	}
	result := PickupFacts{BusinessStatus: f.BusinessStatus, MealPeriods: f.MealPeriods, ServiceDates: make(map[string]bool, 2), AvailableMeals: make(map[string]map[MealCode]bool, 2)}
	for _, d := range f.Dates {
		_, duplicate := result.ServiceDates[d.Date]
		if d.AvailableMeals == nil || duplicate || (d.Date != dates[0] && d.Date != dates[1]) {
			return PickupFacts{}, errSharedMenu
		}
		result.ServiceDates[d.Date] = d.Open
		result.AvailableMeals[d.Date] = map[MealCode]bool{}
		for _, meal := range d.AvailableMeals {
			if (meal != MealLunch && meal != MealDinner) || result.AvailableMeals[d.Date][meal] {
				return PickupFacts{}, errSharedMenu
			}
			result.AvailableMeals[d.Date][meal] = true
		}
	}
	return result, nil
}

var _ Reader = (*SharedPGRepository)(nil)
