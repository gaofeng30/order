package menu

import (
	"strings"
	"unicode/utf8"
)

type Category struct {
	ID       uint64
	Name     string
	Products []Product
}

type Product struct {
	ID                     uint64
	CategoryID             uint64
	Name                   string
	Description            string
	Specification          string
	MealPeriod             string
	ImageObjectKeys        []string
	Listed                 bool
	SoldOut                bool
	OriginalUnitPriceCents uint32
}

type MenuSnapshot struct {
	BusinessStatus     string
	ServiceDatePresent bool
	ServiceDateOpen    bool
	MealPeriods        []MealPeriodRecord
	Categories         []Category
}

type PickupFacts struct {
	BusinessStatus string
	MealPeriods    []MealPeriodRecord
	// ServiceDates contains only rows that exist; missing rows are closed.
	ServiceDates map[string]bool
}

func (product Product) valid() bool {
	if product.ID == 0 || product.CategoryID == 0 || !validMenuText(product.Name, true) ||
		!validMenuText(product.Description, false) || !validMenuText(product.Specification, false) ||
		(product.MealPeriod != "all" && product.MealPeriod != "lunch" && product.MealPeriod != "dinner") ||
		product.ImageObjectKeys == nil || len(product.ImageObjectKeys) > 3 || !product.Listed || product.OriginalUnitPriceCents == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(product.ImageObjectKeys))
	for _, key := range product.ImageObjectKeys {
		if !utf8.ValidString(key) || len(key) < 1 || len(key) > 1024 {
			return false
		}
		if _, duplicate := seen[key]; duplicate {
			return false
		}
		seen[key] = struct{}{}
	}
	return true
}

func validMenuText(value string, required bool) bool {
	if !utf8.ValidString(value) {
		return false
	}
	return !required || (value != "" && strings.TrimSpace(value) == value)
}
