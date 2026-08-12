package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ErrProductNotFound is returned when a product is absent or hidden.
var ErrProductNotFound = errors.New("catalog product not found")

const listQuery = `SELECT
  c.id, c.name,
  p.id, p.category_id, p.name, p.description, p.specification, p.price_cents
FROM categories AS c
LEFT JOIN products AS p
  ON p.category_id = c.id AND p.is_listed = TRUE
WHERE c.is_active = TRUE
ORDER BY c.sort_order ASC, c.id ASC, p.sort_order ASC, p.id ASC`

const detailQuery = `SELECT
  p.id, p.category_id, p.name, p.description, p.specification, p.price_cents
FROM products AS p
INNER JOIN categories AS c ON c.id = p.category_id
WHERE p.id = ? AND p.is_listed = TRUE AND c.is_active = TRUE
LIMIT 1`

// Repository reads the catalog from the application's existing MySQL pool.
type Repository struct {
	db *sql.DB
}

// NewRepository constructs a catalog repository over the shared pool.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// List returns visible categories and products using one consistent statement.
func (repository *Repository) List(ctx context.Context) ([]Category, error) {
	rows, err := repository.db.QueryContext(ctx, listQuery)
	if err != nil {
		return nil, fmt.Errorf("query catalog: %w", err)
	}

	result := make([]Category, 0)
	seen := make(map[uint64]struct{})
	for rows.Next() {
		var categoryID uint64
		var categoryName string
		var productID sql.Null[uint64]
		var productCategoryID sql.Null[uint64]
		var productName sql.Null[string]
		var description sql.Null[string]
		var specification sql.Null[string]
		var priceCents sql.Null[uint32]
		if err := rows.Scan(&categoryID, &categoryName, &productID, &productCategoryID, &productName, &description, &specification, &priceCents); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan catalog: %w", err)
		}
		if categoryID == 0 {
			rows.Close()
			return nil, errors.New("catalog category invariant failed")
		}

		if len(result) == 0 || result[len(result)-1].ID != categoryID {
			if _, exists := seen[categoryID]; exists {
				rows.Close()
				return nil, errors.New("catalog category order invariant failed")
			}
			seen[categoryID] = struct{}{}
			result = append(result, Category{ID: categoryID, Name: categoryName, Products: make([]Product, 0)})
		} else if result[len(result)-1].Name != categoryName {
			rows.Close()
			return nil, errors.New("catalog category data invariant failed")
		}

		validProductFields := 0
		for _, valid := range []bool{productID.Valid, productCategoryID.Valid, productName.Valid, description.Valid, specification.Valid, priceCents.Valid} {
			if valid {
				validProductFields++
			}
		}
		if validProductFields == 0 {
			continue
		}
		if validProductFields != 6 || productID.V == 0 || productCategoryID.V != categoryID {
			rows.Close()
			return nil, errors.New("catalog product invariant failed")
		}
		current := &result[len(result)-1]
		current.Products = append(current.Products, Product{
			ID: productID.V, CategoryID: productCategoryID.V, Name: productName.V,
			Description: description.V, Specification: specification.V, PriceCents: priceCents.V,
		})
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

// GetProduct returns a visible product or ErrProductNotFound.
func (repository *Repository) GetProduct(ctx context.Context, id uint64) (Product, error) {
	var product Product
	err := repository.db.QueryRowContext(ctx, detailQuery, id).Scan(
		&product.ID, &product.CategoryID, &product.Name, &product.Description, &product.Specification, &product.PriceCents,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, ErrProductNotFound
	}
	if err != nil {
		return Product{}, fmt.Errorf("query catalog product: %w", err)
	}
	if product.ID == 0 || product.CategoryID == 0 {
		return Product{}, errors.New("catalog product data invariant failed")
	}
	return product, nil
}
