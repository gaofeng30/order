package importbatch

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gaofeng30/order/services/api/internal/audit"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"golang.org/x/text/unicode/norm"
)

type MySQLApplication struct {
	db       *sql.DB
	receipts *audit.ReceiptStore
	now      func() time.Time
}

func NewMySQLApplication(db *sql.DB) *MySQLApplication {
	return &MySQLApplication{db: db, receipts: audit.NewReceiptStore(db), now: time.Now}
}

type previewPlan struct {
	Kind           Kind      `json:"kind"`
	Rows           []planRow `json:"rows"`
	IgnoredColumns []string  `json:"ignored_columns,omitempty"`
	NewCategories  []string  `json:"new_categories,omitempty"`
	NewCount       uint32    `json:"new_count"`
	UpdateCount    uint32    `json:"update_count"`
	ErrorCount     uint32    `json:"error_count"`
}
type planRow struct {
	Row         uint32 `json:"row"`
	Outcome     string `json:"outcome"`
	Reason      string `json:"reason,omitempty"`
	Name        string `json:"name,omitempty"`
	Phone       string `json:"phone,omitempty"`
	Category    string `json:"category,omitempty"`
	Meal        string `json:"meal,omitempty"`
	Description string `json:"description,omitempty"`
	PriceCents  uint32 `json:"price_cents,omitempty"`
}

func (a *MySQLApplication) Preview(ctx context.Context, meta WriteMeta, kind Kind, file XLSX) (Preview, error) {
	rows, err := ParseRows(file.Bytes, MaxRows(kind))
	if err != nil {
		return Preview{}, err
	}
	if len(rows) < 2 {
		return Preview{}, ErrInvalidTemplate
	}
	plan, err := a.buildPlan(ctx, kind, rows)
	if err != nil {
		return Preview{}, err
	}
	token, err := randomToken()
	if err != nil {
		return Preview{}, ErrUnavailable
	}
	tokenHash := sha256.Sum256([]byte(token))
	digest := sha256.Sum256(file.Bytes)
	request := struct {
		Kind   Kind
		Digest [32]byte
	}{kind, digest}
	raw, _ := json.Marshal(plan)
	now := a.now().UTC().Truncate(time.Microsecond)
	expires := now.Add(15 * time.Minute)
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return Preview{}, ErrUnavailable
	}
	accountID, role, authVersion, err := audit.LockOwner(ctx, tx, meta.ActorUserID)
	if err != nil {
		tx.Rollback()
		return Preview{}, ErrUnavailable
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO import_batches(kind,actor_account_id,file_object_key,file_digest,file_size_bytes,row_count,preview_token_hash,preview_json,new_count,update_count,error_count,state,expires_at,record_version,created_at,updated_at) VALUES(?,?,NULL,?,?,?,?,?,?,?,?,'PREVIEWED',?,1,?,?)`, kind, accountID, digest[:], len(file.Bytes), len(rows)-1, tokenHash[:], raw, plan.NewCount, plan.UpdateCount, plan.ErrorCount, expires, now, now)
	if err != nil {
		tx.Rollback()
		return Preview{}, importSQLError(err)
	}
	id, _ := res.LastInsertId()
	out := planPreview(token, plan, expires)
	rm := audit.CommandMeta{ActorUserID: meta.ActorUserID, ActorAccountID: accountID, ActorRole: role, ActorAuthVersion: authVersion, IdempotencyKey: meta.IdempotencyKey, RequestID: meta.RequestID}
	err = a.receipts.AppendInTx(ctx, tx, rm, "PREVIEW_IMPORT", "IMPORT_BATCH", uint64(id), request, out)
	if errors.Is(err, audit.ErrDuplicateReceipt) {
		tx.Rollback()
		var replay Preview
		ok, re := a.receipts.Replay(ctx, meta.ActorUserID, accountID, "PREVIEW_IMPORT", meta.IdempotencyKey, request, &replay)
		if errors.Is(re, audit.ErrIdempotencyConflict) {
			return Preview{}, ErrIdempotencyConflict
		}
		if re != nil || !ok {
			return Preview{}, ErrUnavailable
		}
		return replay, nil
	}
	if err != nil || tx.Commit() != nil {
		return Preview{}, ErrUnavailable
	}
	return out, nil
}
func (a *MySQLApplication) buildPlan(ctx context.Context, kind Kind, rows [][]string) (previewPlan, error) {
	header := map[string]int{}
	known := map[string]bool{}
	required := []string{}
	if kind == Product {
		required = []string{"菜品名称", "售价", "分类", "餐段可售"}
		for _, v := range append(required, "描述") {
			known[v] = true
		}
	} else if kind == Staff {
		required = []string{"姓名", "手机号"}
		for _, v := range required {
			known[v] = true
		}
	} else {
		return previewPlan{}, ErrInvalidInput
	}
	plan := previewPlan{Kind: kind}
	for i, v := range rows[0] {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if known[v] {
			if _, dup := header[v]; dup {
				return previewPlan{}, ErrInvalidTemplate
			}
			header[v] = i
		} else {
			plan.IgnoredColumns = append(plan.IgnoredColumns, v)
		}
	}
	for _, v := range required {
		if _, ok := header[v]; !ok {
			return previewPlan{}, ErrInvalidTemplate
		}
	}
	existingProducts := map[string]bool{}
	existingCategories := map[string]bool{}
	existingStaff := map[string]bool{}
	if kind == Product {
		dbRows, err := a.db.QueryContext(ctx, `SELECT CONVERT(name_key USING utf8mb4) FROM products`)
		if err != nil {
			return previewPlan{}, ErrUnavailable
		}
		for dbRows.Next() {
			var key string
			if dbRows.Scan(&key) != nil {
				dbRows.Close()
				return previewPlan{}, ErrUnavailable
			}
			existingProducts[key] = true
		}
		dbRows.Close()
		dbRows, err = a.db.QueryContext(ctx, `SELECT CONVERT(name_key USING utf8mb4) FROM categories`)
		if err != nil {
			return previewPlan{}, ErrUnavailable
		}
		for dbRows.Next() {
			var key string
			if dbRows.Scan(&key) != nil {
				dbRows.Close()
				return previewPlan{}, ErrUnavailable
			}
			existingCategories[key] = true
		}
		dbRows.Close()
	} else {
		dbRows, err := a.db.QueryContext(ctx, `SELECT CONVERT(phone USING ascii) FROM staff_whitelist`)
		if err != nil {
			return previewPlan{}, ErrUnavailable
		}
		for dbRows.Next() {
			var phone string
			if dbRows.Scan(&phone) != nil {
				dbRows.Close()
				return previewPlan{}, ErrUnavailable
			}
			existingStaff[phone] = true
		}
		dbRows.Close()
	}
	seen := map[string]uint32{}
	newCats := map[string]bool{}
	for i, row := range rows[1:] {
		line := uint32(i + 2)
		get := func(name string) string {
			idx, ok := header[name]
			if !ok || idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}
		item := planRow{Row: line, Outcome: "ADD"}
		if kind == Product {
			item.Name = strings.TrimSpace(norm.NFKC.String(get("菜品名称")))
			item.Category = strings.TrimSpace(norm.NFKC.String(get("分类")))
			item.Meal = map[string]string{"全天": "all", "午餐": "lunch", "晚餐": "dinner"}[get("餐段可售")]
			item.Description = get("描述")
			var priceErr error
			item.PriceCents, priceErr = parseCents(get("售价"))
			key := item.Name
			if item.Name == "" || item.Category == "" || item.Meal == "" || priceErr != nil {
				item.Outcome, item.Reason = "ERROR", "必填字段或售价格式不正确"
			} else if seen[key] > 0 {
				item.Outcome, item.Reason = "ERROR", fmt.Sprintf("菜品名称在第 %d 行重复", seen[key])
			} else if existingProducts[key] {
				item.Outcome, item.Reason = "ERROR", "菜品已存在，导入只新增"
			} else {
				seen[key] = line
				catKey := item.Category
				if !existingCategories[catKey] && !newCats[catKey] {
					newCats[catKey] = true
					plan.NewCategories = append(plan.NewCategories, item.Category)
				}
			}
		} else {
			item.Name = strings.TrimSpace(norm.NFKC.String(get("姓名")))
			item.Phone = canonicalImportPhone(get("手机号"))
			key := item.Phone
			if item.Name == "" || item.Phone == "" {
				item.Outcome, item.Reason = "ERROR", "姓名或手机号格式不正确"
			} else if seen[key] > 0 {
				item.Outcome, item.Reason = "ERROR", fmt.Sprintf("手机号在第 %d 行重复", seen[key])
			} else {
				seen[key] = line
				if existingStaff[key] {
					item.Outcome = "UPDATE"
				}
			}
		}
		switch item.Outcome {
		case "ADD":
			plan.NewCount++
		case "UPDATE":
			plan.UpdateCount++
		default:
			plan.ErrorCount++
		}
		plan.Rows = append(plan.Rows, item)
	}
	return plan, nil
}
func (a *MySQLApplication) Commit(ctx context.Context, meta WriteMeta, token string, skip bool) (CommitResult, error) {
	tokenHash := sha256.Sum256([]byte(token))
	operation := sha256.Sum256([]byte("operation:" + meta.IdempotencyKey))
	request := struct {
		TokenHash [32]byte
		Skip      bool
	}{tokenHash, skip}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return CommitResult{}, ErrUnavailable
	}
	accountID, role, authVersion, err := audit.LockOwner(ctx, tx, meta.ActorUserID)
	if err != nil {
		tx.Rollback()
		return CommitResult{}, ErrUnavailable
	}
	var id uint64
	var kind Kind
	var raw []byte
	var committedKey []byte
	var state string
	var expires time.Time
	var newCount, updateCount, errorCount uint32
	err = tx.QueryRowContext(ctx, `SELECT id,kind,preview_json,state,expires_at,new_count,update_count,error_count,commit_idempotency_key_hash FROM import_batches WHERE preview_token_hash=? AND actor_account_id=? FOR UPDATE`, tokenHash[:], accountID).Scan(&id, &kind, &raw, &state, &expires, &newCount, &updateCount, &errorCount, &committedKey)
	if errors.Is(err, sql.ErrNoRows) {
		tx.Rollback()
		return CommitResult{}, ErrPreviewExpired
	}
	if err != nil {
		tx.Rollback()
		return CommitResult{}, ErrUnavailable
	}
	if state == "COMMITTED" {
		tx.Rollback()
		if len(committedKey) != len(operation) || subtle.ConstantTimeCompare(committedKey, operation[:]) != 1 {
			return CommitResult{}, ErrIdempotencyConflict
		}
		var replay CommitResult
		ok, replayErr := a.receipts.Replay(ctx, meta.ActorUserID, accountID, "COMMIT_IMPORT", meta.IdempotencyKey, request, &replay)
		if errors.Is(replayErr, audit.ErrIdempotencyConflict) {
			return CommitResult{}, ErrIdempotencyConflict
		}
		if replayErr != nil || !ok {
			return CommitResult{}, ErrUnavailable
		}
		replay.Duplicate = true
		return replay, nil
	}
	if state != "PREVIEWED" || !a.now().Before(expires) {
		tx.Rollback()
		return CommitResult{}, ErrPreviewExpired
	}
	if errorCount > 0 && !skip {
		tx.Rollback()
		return CommitResult{}, ErrInvalidInput
	}
	var plan previewPlan
	if !json.Valid(raw) || json.Unmarshal(raw, &plan) != nil || plan.Kind != kind {
		tx.Rollback()
		return CommitResult{}, ErrUnavailable
	}
	if kind == Staff {
		err = a.commitStaff(ctx, tx, plan)
	} else {
		err = a.commitProducts(ctx, tx, plan)
	}
	if err != nil {
		tx.Rollback()
		return CommitResult{}, err
	}
	out := CommitResult{BatchID: id, NewCount: newCount, UpdateCount: updateCount, SkippedCount: errorCount}
	resultRaw, _ := json.Marshal(out)
	var preview any = raw
	if kind == Staff {
		preview = nil
	}
	res, err := tx.ExecContext(ctx, `UPDATE import_batches SET preview_json=?,state='COMMITTED',commit_idempotency_key_hash=?,skip_invalid=?,result_json=?,committed_at=UTC_TIMESTAMP(6),record_version=record_version+1,updated_at=UTC_TIMESTAMP(6) WHERE id=? AND state='PREVIEWED'`, preview, operation[:], skip, resultRaw, id)
	if err != nil {
		tx.Rollback()
		return CommitResult{}, importSQLError(err)
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		tx.Rollback()
		return CommitResult{}, ErrUnavailable
	}
	rm := audit.CommandMeta{ActorUserID: meta.ActorUserID, ActorAccountID: accountID, ActorRole: role, ActorAuthVersion: authVersion, IdempotencyKey: meta.IdempotencyKey, RequestID: meta.RequestID}
	err = a.receipts.AppendInTx(ctx, tx, rm, "COMMIT_IMPORT", "IMPORT_BATCH", id, request, out)
	if errors.Is(err, audit.ErrDuplicateReceipt) {
		tx.Rollback()
		var replay CommitResult
		ok, re := a.receipts.Replay(ctx, meta.ActorUserID, accountID, "COMMIT_IMPORT", meta.IdempotencyKey, request, &replay)
		if errors.Is(re, audit.ErrIdempotencyConflict) {
			return CommitResult{}, ErrIdempotencyConflict
		}
		if re != nil || !ok {
			return CommitResult{}, ErrUnavailable
		}
		replay.Duplicate = true
		return replay, nil
	}
	if err != nil || tx.Commit() != nil {
		return CommitResult{}, ErrUnavailable
	}
	return out, nil
}
func (a *MySQLApplication) commitStaff(ctx context.Context, tx *sql.Tx, plan previewPlan) error {
	var singleton uint8
	if tx.QueryRowContext(ctx, `SELECT id FROM discount_settings WHERE id=1 FOR UPDATE`).Scan(&singleton) != nil {
		return ErrUnavailable
	}
	rows, err := tx.QueryContext(ctx, `SELECT id FROM staff_whitelist ORDER BY id FOR UPDATE`)
	if err != nil {
		return ErrUnavailable
	}
	for rows.Next() {
	}
	rows.Close()
	for _, r := range plan.Rows {
		if r.Outcome == "ERROR" {
			continue
		}
		name, key := staffName(r.Name)
		if r.Outcome == "ADD" {
			_, err = tx.ExecContext(ctx, `INSERT INTO staff_whitelist(phone,name,name_key,enabled,record_version,created_at,updated_at) VALUES(?,?,?,TRUE,1,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, r.Phone, name, key)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE staff_whitelist SET name=?,name_key=?,record_version=record_version+1,updated_at=UTC_TIMESTAMP(6) WHERE phone=?`, name, key, r.Phone)
		}
		if err != nil {
			return importSQLError(err)
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE discount_settings SET whitelist_version=whitelist_version+1,updated_at=UTC_TIMESTAMP(6) WHERE id=1`)
	if err != nil {
		return ErrUnavailable
	}
	return nil
}
func (a *MySQLApplication) commitProducts(ctx context.Context, tx *sql.Tx, plan previewPlan) error {
	rows, err := tx.QueryContext(ctx, `SELECT id,name,name_key FROM categories ORDER BY id FOR UPDATE`)
	if err != nil {
		return ErrUnavailable
	}
	categories := map[string]uint64{}
	for rows.Next() {
		var id uint64
		var name string
		var key []byte
		if rows.Scan(&id, &name, &key) != nil {
			rows.Close()
			return ErrUnavailable
		}
		categories[string(key)] = id
	}
	rows.Close()
	for _, name := range plan.NewCategories {
		key := catalogKey(name)
		res, err := tx.ExecContext(ctx, `INSERT INTO categories(name,name_key,sort_order,is_active,record_version) SELECT ?,?,COALESCE(MAX(sort_order),0)+1,TRUE,1 FROM categories`, name, key)
		if err != nil {
			return importSQLError(err)
		}
		id, _ := res.LastInsertId()
		categories[string(key)] = uint64(id)
	}
	rows, err = tx.QueryContext(ctx, `SELECT id FROM products ORDER BY id FOR UPDATE`)
	if err != nil {
		return ErrUnavailable
	}
	for rows.Next() {
	}
	rows.Close()
	for _, r := range plan.Rows {
		if r.Outcome == "ERROR" {
			continue
		}
		categoryID := categories[string(catalogKey(r.Category))]
		if categoryID == 0 {
			return ErrUnavailable
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO products(category_id,name,name_key,description,specification,images_json,price_cents,sort_order,is_listed,meal_period,record_version) SELECT ?,?,?,?,'',JSON_ARRAY(),?,COALESCE(MAX(sort_order),0)+1,TRUE,?,1 FROM products WHERE category_id=?`, categoryID, r.Name, catalogKey(r.Name), r.Description, r.PriceCents, r.Meal, categoryID)
		if err != nil {
			return importSQLError(err)
		}
	}
	return nil
}
func planPreview(token string, p previewPlan, expires time.Time) Preview {
	rows := make([]RowOutcome, 0, len(p.Rows))
	for _, r := range p.Rows {
		rows = append(rows, RowOutcome{r.Row, r.Outcome, r.Reason})
	}
	return Preview{Token: token, NewCount: p.NewCount, UpdateCount: p.UpdateCount, ErrorCount: p.ErrorCount, NewCategories: p.NewCategories, IgnoredColumns: p.IgnoredColumns, Rows: rows, ExpiresAt: expires}
}
func randomToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
func parseCents(v string) (uint32, error) {
	parts := strings.Split(v, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, ErrInvalidInput
	}
	whole, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return 0, err
	}
	frac := uint64(0)
	if len(parts) == 2 {
		if len(parts[1]) == 1 {
			parts[1] += "0"
		}
		if len(parts[1]) != 2 {
			return 0, ErrInvalidInput
		}
		frac, err = strconv.ParseUint(parts[1], 10, 8)
		if err != nil {
			return 0, err
		}
	}
	total := whole*100 + frac
	if total == 0 || total > uint64(^uint32(0)) {
		return 0, ErrInvalidInput
	}
	return uint32(total), nil
}
func canonicalImportPhone(v string) string {
	v = strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(v), " ", ""), "-", "")
	if len(v) == 11 && v[0] == '1' {
		return "+86" + v
	}
	if len(v) >= 2 && len(v) <= 16 && v[0] == '+' {
		for _, r := range v[1:] {
			if r < '0' || r > '9' {
				return ""
			}
		}
		return v
	}
	return ""
}
func catalogKey(v string) []byte { return []byte(strings.TrimSpace(norm.NFKC.String(v))) }
func staffName(v string) (string, []byte) {
	name := strings.TrimSpace(norm.NFKC.String(v))
	key := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, name)
	return name, []byte(key)
}
func importSQLError(err error) error {
	var me *mysqlDriver.MySQLError
	if errors.As(err, &me) && (me.Number == 1062 || me.Number == 1451 || me.Number == 1452 || me.Number == 3819) {
		return ErrInvalidInput
	}
	return ErrUnavailable
}
