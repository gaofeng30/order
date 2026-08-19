package menu

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const mealPeriodsQuery = `SELECT
  code, cutoff_time, pickup_start_time, pickup_end_time, interval_minutes
FROM meal_periods
ORDER BY code`

const menuQuery = `SELECT
  c.id, c.name,
  p.id, p.category_id, p.name, p.description, p.specification, p.price_cents,
  (s.product_id IS NOT NULL) AS sold_out
FROM categories AS c
INNER JOIN products AS p
  ON p.category_id = c.id
  AND p.is_listed = TRUE
  AND p.meal_period IN ('all', ?)
LEFT JOIN product_sold_out_dates AS s
  ON s.product_id = p.id AND s.service_date = ?
WHERE c.is_active = TRUE
ORDER BY c.sort_order ASC, c.id ASC, p.sort_order ASC, p.id ASC`

// Repository reads current menu configuration and one selected-date menu from the shared MySQL pool.
type Repository struct {
	db *sql.DB
}

// NewRepository constructs a menu repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// MealPeriods reads the complete current meal configuration with one bounded query.
func (repository *Repository) MealPeriods(ctx context.Context) ([]MealPeriodRecord, error) {
	rows, err := repository.db.QueryContext(ctx, mealPeriodsQuery)
	if err != nil {
		return nil, fmt.Errorf("query meal periods: %w", err)
	}
	records := make([]MealPeriodRecord, 0, 2)
	for rows.Next() {
		var record MealPeriodRecord
		if err := rows.Scan(&record.Code, &record.CutoffTime, &record.PickupStartTime, &record.PickupEndTime, &record.IntervalMinutes); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan meal period: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate meal periods: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close meal periods: %w", err)
	}
	return records, nil
}

// List reads visible products for one meal and joins only the requested service date's sold-out facts.
func (repository *Repository) List(ctx context.Context, serviceDate string, meal MealCode) ([]Category, error) {
	if meal != MealLunch && meal != MealDinner {
		return nil, errors.New("menu meal invariant failed")
	}
	rows, err := repository.db.QueryContext(ctx, menuQuery, string(meal), serviceDate)
	if err != nil {
		return nil, fmt.Errorf("query menu: %w", err)
	}
	categories := make([]Category, 0)
	seen := make(map[uint64]struct{})
	for rows.Next() {
		var category Category
		var product Product
		if err := rows.Scan(
			&category.ID, &category.Name,
			&product.ID, &product.CategoryID, &product.Name, &product.Description, &product.Specification, &product.PriceCents,
			&product.SoldOut,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan menu: %w", err)
		}
		if category.ID == 0 || product.ID == 0 || product.CategoryID != category.ID {
			rows.Close()
			return nil, errors.New("menu row invariant failed")
		}
		if len(categories) == 0 || categories[len(categories)-1].ID != category.ID {
			if _, exists := seen[category.ID]; exists {
				rows.Close()
				return nil, errors.New("menu category order invariant failed")
			}
			seen[category.ID] = struct{}{}
			category.Products = make([]Product, 0)
			categories = append(categories, category)
		} else if categories[len(categories)-1].Name != category.Name {
			rows.Close()
			return nil, errors.New("menu category data invariant failed")
		}
		current := &categories[len(categories)-1]
		current.Products = append(current.Products, product)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate menu: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close menu: %w", err)
	}
	return categories, nil
}
