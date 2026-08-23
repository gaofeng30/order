package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gaofeng30/order/services/api/internal/audit"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"golang.org/x/text/unicode/norm"
)

type ImageURLer interface {
	PublicURL(context.Context, string) (string, error)
}

type MySQLAdminApplication struct {
	db       *sql.DB
	receipts *audit.ReceiptStore
	images   ImageURLer
}

func NewMySQLAdminApplication(db *sql.DB, images ImageURLer) *MySQLAdminApplication {
	return &MySQLAdminApplication{db: db, receipts: audit.NewReceiptStore(db), images: images}
}

func (a *MySQLAdminApplication) ListCategories(ctx context.Context) ([]AdminCategory, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT c.id,c.name,c.sort_order,c.is_active,COUNT(p.id) FROM categories c LEFT JOIN products p ON p.category_id=c.id GROUP BY c.id,c.name,c.sort_order,c.is_active ORDER BY c.sort_order,c.id`)
	if err != nil {
		return nil, ErrAdminUnavailable
	}
	defer rows.Close()
	out := []AdminCategory{}
	for rows.Next() {
		var c AdminCategory
		if rows.Scan(&c.ID, &c.Name, &c.SortOrder, &c.Enabled, &c.ProductCount) != nil {
			return nil, ErrAdminUnavailable
		}
		out = append(out, c)
	}
	if rows.Err() != nil {
		return nil, ErrAdminUnavailable
	}
	return out, nil
}

func (a *MySQLAdminApplication) ListProducts(ctx context.Context, q AdminQuery) ([]AdminProduct, error) {
	if q.ServiceDate == "" {
		return nil, ErrAdminInvalidInput
	}
	rows, err := a.db.QueryContext(ctx, `SELECT p.id,p.category_id,c.name,p.name,p.description,p.meal_period,p.images_json,p.price_cents,p.is_listed,(s.product_id IS NOT NULL) FROM products p JOIN categories c ON c.id=p.category_id LEFT JOIN product_sold_out_dates s ON s.product_id=p.id AND s.service_date=? ORDER BY p.category_id,p.sort_order,p.id`, q.ServiceDate)
	if err != nil {
		return nil, ErrAdminUnavailable
	}
	defer rows.Close()
	out := []AdminProduct{}
	for rows.Next() {
		p, err := a.scanProduct(ctx, rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if rows.Err() != nil {
		return nil, ErrAdminUnavailable
	}
	return out, nil
}

func (a *MySQLAdminApplication) GetProduct(ctx context.Context, id uint64, q AdminQuery) (AdminProduct, error) {
	if id == 0 || q.ServiceDate == "" {
		return AdminProduct{}, ErrAdminInvalidInput
	}
	return a.scanProduct(ctx, a.db.QueryRowContext(ctx, `SELECT p.id,p.category_id,c.name,p.name,p.description,p.meal_period,p.images_json,p.price_cents,p.is_listed,(s.product_id IS NOT NULL) FROM products p JOIN categories c ON c.id=p.category_id LEFT JOIN product_sold_out_dates s ON s.product_id=p.id AND s.service_date=? WHERE p.id=?`, q.ServiceDate, id).Scan)
}

type scanner func(...any) error

func (a *MySQLAdminApplication) scanProduct(ctx context.Context, scan scanner) (AdminProduct, error) {
	var p AdminProduct
	var raw []byte
	err := scan(&p.ID, &p.CategoryID, &p.CategoryName, &p.Name, &p.Description, &p.MealPeriod, &raw, &p.PriceCents, &p.Listed, &p.SoldOut)
	if errors.Is(err, sql.ErrNoRows) {
		return AdminProduct{}, ErrAdminNotFound
	}
	if err != nil {
		return AdminProduct{}, ErrAdminUnavailable
	}
	var stored []struct {
		ObjectKey string `json:"object_key"`
	}
	if !json.Valid(raw) || json.Unmarshal(raw, &stored) != nil || len(stored) > 3 {
		return AdminProduct{}, ErrAdminUnavailable
	}
	seen := map[string]bool{}
	for i, item := range stored {
		if item.ObjectKey == "" || seen[item.ObjectKey] {
			return AdminProduct{}, ErrAdminUnavailable
		}
		seen[item.ObjectKey] = true
		if a.images == nil {
			return AdminProduct{}, ErrAdminUnavailable
		}
		url, err := a.images.PublicURL(ctx, item.ObjectKey)
		if err != nil {
			return AdminProduct{}, ErrAdminUnavailable
		}
		p.Images = append(p.Images, AdminImage{ObjectKey: item.ObjectKey, URL: url, SortOrder: uint8(i)})
	}
	return p, nil
}

func (a *MySQLAdminApplication) Execute(ctx context.Context, meta WriteMeta, cmd CatalogCommand) (CatalogResult, error) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return CatalogResult{}, ErrAdminUnavailable
	}
	accountID, role, authVersion, err := audit.LockOwner(ctx, tx, meta.ActorUserID)
	if err != nil {
		tx.Rollback()
		return CatalogResult{}, ErrAdminUnavailable
	}
	result, err := a.executeTx(ctx, tx, cmd)
	if err != nil {
		tx.Rollback()
		return CatalogResult{}, err
	}
	action := string(cmd.Kind)
	targetType, targetID := "", uint64(0)
	if cmd.ProductID > 0 {
		targetType, targetID = "PRODUCT", cmd.ProductID
	} else if cmd.CategoryID > 0 {
		targetType, targetID = "CATEGORY", cmd.CategoryID
	}
	rm := audit.CommandMeta{ActorUserID: meta.ActorUserID, ActorAccountID: accountID, ActorRole: role, ActorAuthVersion: authVersion, IdempotencyKey: meta.IdempotencyKey, RequestID: meta.RequestID}
	err = a.receipts.AppendInTx(ctx, tx, rm, action, targetType, targetID, cmd, result)
	if errors.Is(err, audit.ErrDuplicateReceipt) {
		tx.Rollback()
		var replay CatalogResult
		ok, replayErr := a.receipts.Replay(ctx, meta.ActorUserID, accountID, action, meta.IdempotencyKey, cmd, &replay)
		if errors.Is(replayErr, audit.ErrIdempotencyConflict) {
			return CatalogResult{}, ErrAdminIdempotencyConflict
		}
		if replayErr != nil || !ok {
			return CatalogResult{}, ErrAdminUnavailable
		}
		return replay, nil
	}
	if err != nil {
		tx.Rollback()
		return CatalogResult{}, ErrAdminUnavailable
	}
	if tx.Commit() != nil {
		return CatalogResult{}, ErrAdminUnavailable
	}
	return result, nil
}

func (a *MySQLAdminApplication) executeTx(ctx context.Context, tx *sql.Tx, cmd CatalogCommand) (CatalogResult, error) {
	switch cmd.Kind {
	case CommandCreateCategory:
		name, key, ok := catalogName(cmd.Name)
		if !ok {
			return CatalogResult{}, ErrAdminInvalidInput
		}
		sort := uint32(1)
		var last uint32
		err := tx.QueryRowContext(ctx, `SELECT sort_order FROM categories ORDER BY sort_order DESC,id DESC LIMIT 1 FOR UPDATE`).Scan(&last)
		if err == nil {
			sort = last + 1
		} else if !errors.Is(err, sql.ErrNoRows) {
			return CatalogResult{}, ErrAdminUnavailable
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO categories(name,name_key,sort_order,is_active,record_version) VALUES(?,?,?,?,1)`, name, key, sort, true)
		if err != nil {
			return CatalogResult{}, adminSQLError(err)
		}
		id, _ := res.LastInsertId()
		return CatalogResult{Category: &AdminCategory{ID: uint64(id), Name: name, SortOrder: sort, Enabled: true}}, nil
	case CommandUpdateCategory:
		var current AdminCategory
		if tx.QueryRowContext(ctx, `SELECT id,name,sort_order,is_active FROM categories WHERE id=? FOR UPDATE`, cmd.CategoryID).Scan(&current.ID, &current.Name, &current.SortOrder, &current.Enabled) != nil {
			return CatalogResult{}, ErrAdminNotFound
		}
		name := current.Name
		var key []byte
		if cmd.Name != "" {
			var ok bool
			name, key, ok = catalogName(cmd.Name)
			if !ok {
				return CatalogResult{}, ErrAdminInvalidInput
			}
		}
		enabled := current.Enabled
		if cmd.Enabled != nil {
			enabled = *cmd.Enabled
		}
		var err error
		if key == nil {
			_, err = tx.ExecContext(ctx, `UPDATE categories SET is_active=?,record_version=record_version+1 WHERE id=?`, enabled, cmd.CategoryID)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE categories SET name=?,name_key=?,is_active=?,record_version=record_version+1 WHERE id=?`, name, key, enabled, cmd.CategoryID)
		}
		if err != nil {
			return CatalogResult{}, adminSQLError(err)
		}
		current.Name, current.Enabled = name, enabled
		return CatalogResult{Category: &current}, nil
	case CommandDeleteCategory:
		var categoryID uint64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM categories WHERE id=? FOR UPDATE`, cmd.CategoryID).Scan(&categoryID); errors.Is(err, sql.ErrNoRows) {
			return CatalogResult{}, ErrAdminNotFound
		} else if err != nil {
			return CatalogResult{}, ErrAdminUnavailable
		}
		var productID uint64
		err := tx.QueryRowContext(ctx, `SELECT id FROM products WHERE category_id=? ORDER BY id LIMIT 1 FOR UPDATE`, cmd.CategoryID).Scan(&productID)
		if err == nil {
			return CatalogResult{}, ErrAdminConflict
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return CatalogResult{}, ErrAdminUnavailable
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM categories WHERE id=?`, categoryID)
		if err != nil {
			return CatalogResult{}, adminSQLError(err)
		}
		n, _ := res.RowsAffected()
		if n != 1 {
			return CatalogResult{}, ErrAdminNotFound
		}
		return CatalogResult{}, nil
	case CommandReorderCategory:
		return CatalogResult{}, a.reorder(ctx, tx, "categories", "", cmd.OrderedIDs)
	case CommandCreateProduct, CommandUpdateProduct:
		return a.writeProduct(ctx, tx, cmd)
	case CommandDeleteProduct:
		res, err := tx.ExecContext(ctx, `DELETE FROM products WHERE id=?`, cmd.ProductID)
		if err != nil {
			return CatalogResult{}, adminSQLError(err)
		}
		n, _ := res.RowsAffected()
		if n != 1 {
			return CatalogResult{}, ErrAdminNotFound
		}
		return CatalogResult{}, nil
	case CommandProductStatus:
		if cmd.Listed == nil {
			return CatalogResult{}, ErrAdminInvalidInput
		}
		res, err := tx.ExecContext(ctx, `UPDATE products SET is_listed=?,record_version=record_version+1 WHERE id=?`, *cmd.Listed, cmd.ProductID)
		if err != nil {
			return CatalogResult{}, adminSQLError(err)
		}
		n, _ := res.RowsAffected()
		if n != 1 {
			return CatalogResult{}, ErrAdminNotFound
		}
		return CatalogResult{}, nil
	case CommandProductSoldOut:
		if cmd.SoldOut == nil || cmd.ServiceDate == "" {
			return CatalogResult{}, ErrAdminInvalidInput
		}
		if *cmd.SoldOut {
			_, err := tx.ExecContext(ctx, `INSERT INTO product_sold_out_dates(service_date,product_id) VALUES(?,?) ON DUPLICATE KEY UPDATE product_id=VALUES(product_id)`, cmd.ServiceDate, cmd.ProductID)
			if err != nil {
				return CatalogResult{}, adminSQLError(err)
			}
		} else {
			_, err := tx.ExecContext(ctx, `DELETE FROM product_sold_out_dates WHERE service_date=? AND product_id=?`, cmd.ServiceDate, cmd.ProductID)
			if err != nil {
				return CatalogResult{}, adminSQLError(err)
			}
		}
		return CatalogResult{}, nil
	case CommandReorderProducts:
		return CatalogResult{}, a.reorder(ctx, tx, "products", fmt.Sprintf("category_id=%d", cmd.CategoryID), cmd.OrderedIDs)
	default:
		return CatalogResult{}, ErrAdminInvalidInput
	}
}

func (a *MySQLAdminApplication) writeProduct(ctx context.Context, tx *sql.Tx, cmd CatalogCommand) (CatalogResult, error) {
	name, key, ok := catalogName(cmd.Name)
	if !ok || cmd.PriceCents == 0 || cmd.CategoryID == 0 || len(cmd.Images) > 3 {
		return CatalogResult{}, ErrAdminInvalidInput
	}
	var categoryName string
	if tx.QueryRowContext(ctx, `SELECT name FROM categories WHERE id=? FOR UPDATE`, cmd.CategoryID).Scan(&categoryName) != nil {
		return CatalogResult{}, ErrAdminConflict
	}
	stored := make([]map[string]string, 0, len(cmd.Images))
	seen := map[string]bool{}
	for _, image := range cmd.Images {
		if image.ObjectKey == "" || len(image.ObjectKey) > 1024 || seen[image.ObjectKey] {
			return CatalogResult{}, ErrAdminInvalidInput
		}
		seen[image.ObjectKey] = true
		stored = append(stored, map[string]string{"object_key": image.ObjectKey})
	}
	raw, _ := json.Marshal(stored)
	var id uint64
	if cmd.Kind == CommandCreateProduct {
		sort := uint32(1)
		var last uint32
		err := tx.QueryRowContext(ctx, `SELECT sort_order FROM products WHERE category_id=? ORDER BY sort_order DESC,id DESC LIMIT 1 FOR UPDATE`, cmd.CategoryID).Scan(&last)
		if err == nil {
			sort = last + 1
		} else if !errors.Is(err, sql.ErrNoRows) {
			return CatalogResult{}, ErrAdminUnavailable
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO products(category_id,name,name_key,description,specification,images_json,price_cents,sort_order,is_listed,meal_period,record_version) VALUES(?,?,?,?, '', ?,?,?,TRUE,?,1)`, cmd.CategoryID, name, key, cmd.Description, raw, cmd.PriceCents, sort, cmd.MealPeriod)
		if err != nil {
			return CatalogResult{}, adminSQLError(err)
		}
		v, _ := res.LastInsertId()
		id = uint64(v)
	} else {
		id = cmd.ProductID
		res, err := tx.ExecContext(ctx, `UPDATE products SET category_id=?,name=?,name_key=?,description=?,images_json=?,price_cents=?,meal_period=?,record_version=record_version+1 WHERE id=?`, cmd.CategoryID, name, key, cmd.Description, raw, cmd.PriceCents, cmd.MealPeriod, id)
		if err != nil {
			return CatalogResult{}, adminSQLError(err)
		}
		n, _ := res.RowsAffected()
		if n != 1 {
			return CatalogResult{}, ErrAdminNotFound
		}
	}
	return CatalogResult{Product: &AdminProduct{ID: id, CategoryID: cmd.CategoryID, CategoryName: categoryName, Name: name, Description: cmd.Description, MealPeriod: cmd.MealPeriod, Images: cmd.Images, PriceCents: cmd.PriceCents, Listed: true}}, nil
}

func (a *MySQLAdminApplication) reorder(ctx context.Context, tx *sql.Tx, table, where string, ids []uint64) error {
	if len(ids) == 0 {
		return ErrAdminInvalidInput
	}
	query := `SELECT id FROM ` + table
	if where != "" {
		query += ` WHERE ` + where
	}
	query += ` ORDER BY id FOR UPDATE`
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return ErrAdminUnavailable
	}
	existing := []uint64{}
	for rows.Next() {
		var id uint64
		if rows.Scan(&id) != nil {
			rows.Close()
			return ErrAdminUnavailable
		}
		existing = append(existing, id)
	}
	rows.Close()
	if len(existing) != len(ids) {
		return ErrAdminConflict
	}
	seen := map[uint64]bool{}
	for _, id := range existing {
		seen[id] = true
	}
	for _, id := range ids {
		if !seen[id] {
			return ErrAdminConflict
		}
	}
	for i, id := range ids {
		if _, err := tx.ExecContext(ctx, `UPDATE `+table+` SET sort_order=?,record_version=record_version+1 WHERE id=?`, i+1, id); err != nil {
			return adminSQLError(err)
		}
	}
	return nil
}
func catalogName(value string) (string, []byte, bool) {
	name := strings.TrimSpace(norm.NFKC.String(value))
	key := []byte(name)
	return name, key, name != "" && len(key) <= 400
}
func adminSQLError(err error) error {
	var me *mysqlDriver.MySQLError
	if errors.As(err, &me) && (me.Number == 1062 || me.Number == 1451 || me.Number == 1452 || me.Number == 3819) {
		return ErrAdminConflict
	}
	return ErrAdminUnavailable
}
