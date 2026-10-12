package catalog

import (
	"strings"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/menu"
)

type Category struct {
	ID       uint64
	Name     string
	Products []Product
}

// Product is one current server-owned catalog projection. ImageObjectKeys are
// ordered durable facts; public URLs are resolved only for the response.
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

type CurrentFacts struct {
	BusinessStatus     string
	ServiceDatePresent bool
	ServiceDateOpen    bool
	MealPeriods        []menu.MealPeriodRecord
}

func (product Product) valid() bool {
	if product.ID == 0 || product.CategoryID == 0 || !validCatalogText(product.Name, true) ||
		!validCatalogText(product.Description, false) || !validCatalogText(product.Specification, false) ||
		(product.MealPeriod != "all" && product.MealPeriod != "lunch" && product.MealPeriod != "dinner") ||
		product.ImageObjectKeys == nil || len(product.ImageObjectKeys) > 3 || product.OriginalUnitPriceCents == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(product.ImageObjectKeys))
	for _, key := range product.ImageObjectKeys {
		if !validImageObjectKey(key) {
			return false
		}
		if _, duplicate := seen[key]; duplicate {
			return false
		}
		seen[key] = struct{}{}
	}
	return true
}

func validCatalogText(value string, required bool) bool {
	if !utf8.ValidString(value) {
		return false
	}
	if required {
		return value != "" && strings.TrimSpace(value) == value
	}
	return true
}

func validImageObjectKey(value string) bool {
	return utf8.ValidString(value) && len(value) >= 1 && len(value) <= 1024
}
