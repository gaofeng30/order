package menu

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"
)

const mealPeriodsQuery = `SELECT
  code, cutoff_time, pickup_start_time, pickup_end_time, interval_minutes
FROM meal_periods
ORDER BY code`

const menuFactsQuery = `SELECT
  settings.business_status,
  DATE_FORMAT(dates.service_date,'%Y-%m-%d'), dates.is_open
FROM storefront_settings AS settings
LEFT JOIN service_dates AS dates ON dates.service_date = ?
WHERE settings.id = 1
LIMIT 1`

const menuQuery = `SELECT
  c.id, c.name,
  p.id, p.category_id, p.name, p.description, p.specification, p.meal_period,
  p.images_json, p.is_listed, p.price_cents,
  (sold.product_id IS NOT NULL) AS sold_out
FROM categories AS c
INNER JOIN products AS p ON p.category_id = c.id AND p.is_listed = TRUE
LEFT JOIN product_sold_out_dates AS sold ON sold.product_id = p.id AND sold.service_date = ?
WHERE c.is_active = TRUE
ORDER BY c.sort_order ASC, c.id ASC, p.sort_order ASC, p.id ASC`

const pickupStoreQuery = `SELECT business_status FROM storefront_settings WHERE id = 1 LIMIT 1`
const pickupDatesQuery = `SELECT DATE_FORMAT(service_date,'%Y-%m-%d'), is_open
FROM service_dates
WHERE service_date IN (?, ?)
ORDER BY service_date`

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

// MealPeriods exposes the same stored schedule reader used by both public
// snapshots; it exists for migration and fresh-MySQL verification.
func (repository *Repository) MealPeriods(ctx context.Context) ([]MealPeriodRecord, error) {
	return queryMealPeriods(ctx, repository.db)
}

func (repository *Repository) ReadMenu(ctx context.Context, serviceDate string) (result MenuSnapshot, resultErr error) {
	transaction, err := repository.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return MenuSnapshot{}, fmt.Errorf("begin menu snapshot: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = transaction.Rollback()
		}
	}()
	var storedDate sql.NullString
	var open sql.NullBool
	if err := transaction.QueryRowContext(ctx, menuFactsQuery, serviceDate).Scan(&result.BusinessStatus, &storedDate, &open); err != nil {
		return MenuSnapshot{}, fmt.Errorf("query menu current facts: %w", err)
	}
	if storedDate.Valid != open.Valid || (storedDate.Valid && storedDate.String != serviceDate) {
		return MenuSnapshot{}, errors.New("menu service date invariant failed")
	}
	result.ServiceDatePresent = storedDate.Valid
	result.ServiceDateOpen = open.Valid && open.Bool
	result.MealPeriods, err = queryMealPeriods(ctx, transaction)
	if err != nil {
		return MenuSnapshot{}, err
	}
	rows, err := transaction.QueryContext(ctx, menuQuery, serviceDate)
	if err != nil {
		return MenuSnapshot{}, fmt.Errorf("query menu products: %w", err)
	}
	result.Categories, err = foldMenuRows(rows)
	if err != nil {
		return MenuSnapshot{}, err
	}
	if err := transaction.Commit(); err != nil {
		return MenuSnapshot{}, fmt.Errorf("commit menu snapshot: %w", err)
	}
	committed = true
	return result, nil
}

func (repository *Repository) ReadPickupFacts(ctx context.Context, serviceDates []string) (result PickupFacts, resultErr error) {
	if len(serviceDates) != 2 || serviceDates[0] == serviceDates[1] {
		return PickupFacts{}, errors.New("pickup date request invariant failed")
	}
	transaction, err := repository.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return PickupFacts{}, fmt.Errorf("begin pickup facts: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = transaction.Rollback()
		}
	}()
	if err := transaction.QueryRowContext(ctx, pickupStoreQuery).Scan(&result.BusinessStatus); err != nil {
		return PickupFacts{}, fmt.Errorf("query pickup store: %w", err)
	}
	result.MealPeriods, err = queryMealPeriods(ctx, transaction)
	if err != nil {
		return PickupFacts{}, err
	}
	rows, err := transaction.QueryContext(ctx, pickupDatesQuery, serviceDates[0], serviceDates[1])
	if err != nil {
		return PickupFacts{}, fmt.Errorf("query pickup dates: %w", err)
	}
	result.ServiceDates = make(map[string]bool, 2)
	expected := map[string]struct{}{serviceDates[0]: {}, serviceDates[1]: {}}
	for rows.Next() {
		var date string
		var open bool
		if err := rows.Scan(&date, &open); err != nil {
			rows.Close()
			return PickupFacts{}, fmt.Errorf("scan pickup date: %w", err)
		}
		if _, valid := expected[date]; !valid {
			rows.Close()
			return PickupFacts{}, errors.New("pickup service date invariant failed")
		}
		if _, duplicate := result.ServiceDates[date]; duplicate {
			rows.Close()
			return PickupFacts{}, errors.New("pickup service date duplicate")
		}
		result.ServiceDates[date] = open
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return PickupFacts{}, fmt.Errorf("iterate pickup dates: %w", err)
	}
	if err := rows.Close(); err != nil {
		return PickupFacts{}, fmt.Errorf("close pickup dates: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return PickupFacts{}, fmt.Errorf("commit pickup facts: %w", err)
	}
	committed = true
	return result, nil
}

type menuQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func queryMealPeriods(ctx context.Context, queryer menuQueryer) ([]MealPeriodRecord, error) {
	rows, err := queryer.QueryContext(ctx, mealPeriodsQuery)
	if err != nil {
		return nil, fmt.Errorf("query meal periods: %w", err)
	}
	defer rows.Close()
	result := make([]MealPeriodRecord, 0, 2)
	for rows.Next() {
		var record MealPeriodRecord
		if err := rows.Scan(&record.Code, &record.CutoffTime, &record.PickupStartTime, &record.PickupEndTime, &record.IntervalMinutes); err != nil {
			return nil, fmt.Errorf("scan meal period: %w", err)
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate meal periods: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close meal periods: %w", err)
	}
	return result, nil
}

func foldMenuRows(rows *sql.Rows) ([]Category, error) {
	defer rows.Close()
	categories := make([]Category, 0)
	seen := make(map[uint64]struct{})
	for rows.Next() {
		var category Category
		var product Product
		var rawImages []byte
		if err := rows.Scan(
			&category.ID, &category.Name,
			&product.ID, &product.CategoryID, &product.Name, &product.Description, &product.Specification, &product.MealPeriod,
			&rawImages, &product.Listed, &product.OriginalUnitPriceCents, &product.SoldOut,
		); err != nil {
			return nil, fmt.Errorf("scan menu: %w", err)
		}
		if category.ID == 0 || product.CategoryID != category.ID || !decodeMenuImages(rawImages, &product.ImageObjectKeys) || !product.valid() {
			return nil, errors.New("menu row invariant failed")
		}
		if len(categories) == 0 || categories[len(categories)-1].ID != category.ID {
			if _, duplicate := seen[category.ID]; duplicate {
				return nil, errors.New("menu category order invariant failed")
			}
			seen[category.ID] = struct{}{}
			category.Products = make([]Product, 0)
			categories = append(categories, category)
		} else if categories[len(categories)-1].Name != category.Name {
			return nil, errors.New("menu category data invariant failed")
		}
		categories[len(categories)-1].Products = append(categories[len(categories)-1].Products, product)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate menu: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close menu rows: %w", err)
	}
	return categories, nil
}

func decodeMenuImages(raw []byte, destination *[]string) bool {
	var stored []map[string]any
	if !json.Valid(raw) || json.Unmarshal(raw, &stored) != nil || stored == nil || len(stored) > 3 {
		return false
	}
	keys := make([]string, 0, len(stored))
	seen := make(map[string]struct{}, len(stored))
	for _, image := range stored {
		key, keyOK := image["object_key"].(string)
		if len(image) != 1 || !keyOK || !utf8.ValidString(key) || len(key) < 1 || len(key) > 1024 {
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
