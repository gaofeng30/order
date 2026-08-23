package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gaofeng30/order/services/api/internal/menu"
)

var ErrProductNotFound = errors.New("catalog product not found")

const browseQuery = `SELECT
  c.id, c.name,
  p.id, p.category_id, p.name, p.description, p.specification, p.meal_period,
  p.images_json, p.is_listed, p.price_cents
FROM categories AS c
LEFT JOIN products AS p
  ON p.category_id = c.id AND p.is_listed = TRUE
WHERE c.is_active = TRUE
ORDER BY c.sort_order ASC, c.id ASC, p.sort_order ASC, p.id ASC`

const detailProductQuery = `SELECT
  p.id, p.category_id, p.name, p.description, p.specification, p.meal_period,
  p.images_json, p.is_listed, p.price_cents,
  (sold.product_id IS NOT NULL) AS sold_out
FROM products AS p
INNER JOIN categories AS c ON c.id = p.category_id AND c.is_active = TRUE
LEFT JOIN product_sold_out_dates AS sold ON sold.product_id = p.id AND sold.service_date = ?
WHERE p.id = ? AND p.is_listed = TRUE
LIMIT 1`

const currentFactsQuery = `SELECT
  settings.business_status,
  DATE_FORMAT(dates.service_date,'%Y-%m-%d'), dates.is_open
FROM storefront_settings AS settings
LEFT JOIN service_dates AS dates ON dates.service_date = ?
WHERE settings.id = 1
LIMIT 1`

const catalogMealPeriodsQuery = `SELECT
  code, cutoff_time, pickup_start_time, pickup_end_time, interval_minutes
FROM meal_periods
ORDER BY code`

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (repository *Repository) Browse(ctx context.Context) ([]Category, error) {
	rows, err := repository.db.QueryContext(ctx, browseQuery)
	if err != nil {
		return nil, fmt.Errorf("query catalog: %w", err)
	}
	return foldBrowseRows(rows)
}

func (repository *Repository) Detail(ctx context.Context, id uint64, serviceDate string) (result Product, facts CurrentFacts, resultErr error) {
	transaction, err := repository.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Product{}, CurrentFacts{}, fmt.Errorf("begin catalog detail: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = transaction.Rollback()
		}
	}()

	var date sql.NullString
	var open sql.NullBool
	if err := transaction.QueryRowContext(ctx, currentFactsQuery, serviceDate).Scan(&facts.BusinessStatus, &date, &open); err != nil {
		return Product{}, CurrentFacts{}, fmt.Errorf("query catalog current facts: %w", err)
	}
	if date.Valid != open.Valid || (date.Valid && date.String != serviceDate) {
		return Product{}, CurrentFacts{}, errors.New("catalog service date invariant failed")
	}
	facts.ServiceDatePresent = date.Valid
	facts.ServiceDateOpen = open.Valid && open.Bool

	periodRows, err := transaction.QueryContext(ctx, catalogMealPeriodsQuery)
	if err != nil {
		return Product{}, CurrentFacts{}, fmt.Errorf("query catalog meal periods: %w", err)
	}
	facts.MealPeriods, err = scanMealPeriods(periodRows)
	if err != nil {
		return Product{}, CurrentFacts{}, err
	}

	var rawImages []byte
	err = transaction.QueryRowContext(ctx, detailProductQuery, serviceDate, id).Scan(
		&result.ID, &result.CategoryID, &result.Name, &result.Description, &result.Specification, &result.MealPeriod,
		&rawImages, &result.Listed, &result.OriginalUnitPriceCents, &result.SoldOut,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, CurrentFacts{}, ErrProductNotFound
	}
	if err != nil || !decodeImageKeys(rawImages, &result.ImageObjectKeys) || !result.valid() || !result.Listed {
		if err != nil {
			return Product{}, CurrentFacts{}, fmt.Errorf("query catalog product: %w", err)
		}
		return Product{}, CurrentFacts{}, errors.New("catalog product invariant failed")
	}
	if err := transaction.Commit(); err != nil {
		return Product{}, CurrentFacts{}, fmt.Errorf("commit catalog detail: %w", err)
	}
	committed = true
	return result, facts, nil
}

func foldBrowseRows(rows *sql.Rows) ([]Category, error) {
	defer rows.Close()
	result := make([]Category, 0)
	seen := make(map[uint64]struct{})
	for rows.Next() {
		var categoryID uint64
		var categoryName string
		var productID, productCategoryID sql.Null[uint64]
		var productName, description, specification, mealPeriod sql.Null[string]
		var rawImages []byte
		var listed sql.NullBool
		var price sql.Null[uint32]
		if err := rows.Scan(&categoryID, &categoryName, &productID, &productCategoryID, &productName, &description, &specification, &mealPeriod, &rawImages, &listed, &price); err != nil {
			return nil, fmt.Errorf("scan catalog: %w", err)
		}
		if categoryID == 0 || !validCatalogText(categoryName, true) {
			return nil, errors.New("catalog category invariant failed")
		}
		if len(result) == 0 || result[len(result)-1].ID != categoryID {
			if _, duplicate := seen[categoryID]; duplicate {
				return nil, errors.New("catalog category order invariant failed")
			}
			seen[categoryID] = struct{}{}
			result = append(result, Category{ID: categoryID, Name: categoryName, Products: make([]Product, 0)})
		} else if result[len(result)-1].Name != categoryName {
			return nil, errors.New("catalog category data invariant failed")
		}
		validFields := 0
		for _, valid := range []bool{productID.Valid, productCategoryID.Valid, productName.Valid, description.Valid, specification.Valid, mealPeriod.Valid, rawImages != nil, listed.Valid, price.Valid} {
			if valid {
				validFields++
			}
		}
		if validFields == 0 {
			continue
		}
		product := Product{
			ID: productID.V, CategoryID: productCategoryID.V, Name: productName.V, Description: description.V,
			Specification: specification.V, MealPeriod: mealPeriod.V, Listed: listed.Bool, OriginalUnitPriceCents: price.V,
		}
		if validFields != 9 || product.CategoryID != categoryID || !decodeImageKeys(rawImages, &product.ImageObjectKeys) || !product.Listed || !product.valid() {
			return nil, errors.New("catalog product invariant failed")
		}
		result[len(result)-1].Products = append(result[len(result)-1].Products, product)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate catalog: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close catalog rows: %w", err)
	}
	return result, nil
}

func scanMealPeriods(rows *sql.Rows) ([]menu.MealPeriodRecord, error) {
	defer rows.Close()
	result := make([]menu.MealPeriodRecord, 0, 2)
	for rows.Next() {
		var record menu.MealPeriodRecord
		if err := rows.Scan(&record.Code, &record.CutoffTime, &record.PickupStartTime, &record.PickupEndTime, &record.IntervalMinutes); err != nil {
			return nil, fmt.Errorf("scan catalog meal period: %w", err)
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate catalog meal periods: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close catalog meal periods: %w", err)
	}
	return result, nil
}

func decodeImageKeys(raw []byte, destination *[]string) bool {
	var stored []map[string]any
	if !json.Valid(raw) || json.Unmarshal(raw, &stored) != nil || stored == nil || len(stored) > 3 {
		return false
	}
	keys := make([]string, 0, len(stored))
	seen := make(map[string]struct{}, len(stored))
	for _, item := range stored {
		key, keyOK := item["object_key"].(string)
		if len(item) != 1 || !keyOK || !validImageObjectKey(key) {
			return false
		}
		if _, duplicate := seen[key]; duplicate {
			return false
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	*destination = keys
	return true
}
