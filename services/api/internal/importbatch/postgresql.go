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
	"time"

	"github.com/gaofeng30/order/services/api/internal/audit"
	"github.com/jackc/pgx/v5/pgconn"
)

type PostgreSQLApplication struct {
	db       *sql.DB
	receipts *audit.ReceiptStore
	now      func() time.Time
}

func NewPostgreSQLApplication(db *sql.DB) *PostgreSQLApplication {
	return &PostgreSQLApplication{db: db, receipts: audit.NewReceiptStore(db), now: time.Now}
}

func (a *PostgreSQLApplication) Preview(ctx context.Context, meta WriteMeta, kind Kind, file XLSX) (Preview, error) {
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
	defer tx.Rollback()
	accountID, role, authVersion, err := audit.LockOwner(ctx, tx, meta.ActorUserID)
	if err != nil {
		tx.Rollback()
		return Preview{}, ErrUnavailable
	}
	var id uint64
	err = tx.QueryRowContext(ctx, `INSERT INTO import_batches(kind,actor_account_id,file_object_key,file_digest,file_size_bytes,row_count,preview_token_hash,preview_json,new_count,update_count,error_count,state,expires_at,record_version,created_at,updated_at) VALUES($1,$2,NULL,$3,$4,$5,$6,$7,$8,$9,$10,'PREVIEWED',$11,1,$12,$13) RETURNING id`, kind, accountID, digest[:], len(file.Bytes), len(rows)-1, tokenHash[:], raw, plan.NewCount, plan.UpdateCount, plan.ErrorCount, expires, now, now).Scan(&id)
	if err != nil {
		tx.Rollback()
		return Preview{}, importSQLError(err)
	}
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
func (a *PostgreSQLApplication) buildPlan(ctx context.Context, kind Kind, rows [][]string) (previewPlan, error) {
	header, plan, err := planHeader(kind, rows)
	if err != nil {
		return previewPlan{}, err
	}
	existingProducts := map[string]bool{}
	existingCategories := map[string]bool{}
	existingStaff := map[string]bool{}
	if kind == Product {
		dbRows, err := a.db.QueryContext(ctx, `SELECT name_key FROM products`)
		if err != nil {
			return previewPlan{}, ErrUnavailable
		}
		for dbRows.Next() {
			var key string
			if dbRows.Scan(&key) != nil {
				if dbRows.Err() != nil {
					dbRows.Close()
					return previewPlan{}, ErrUnavailable
				}
				dbRows.Close()
				return previewPlan{}, ErrUnavailable
			}
			existingProducts[key] = true
		}
		if dbRows.Err() != nil {
			dbRows.Close()
			return previewPlan{}, ErrUnavailable
		}
		dbRows.Close()
		dbRows, err = a.db.QueryContext(ctx, `SELECT name_key FROM categories`)
		if err != nil {
			return previewPlan{}, ErrUnavailable
		}
		for dbRows.Next() {
			var key string
			if dbRows.Scan(&key) != nil {
				if dbRows.Err() != nil {
					dbRows.Close()
					return previewPlan{}, ErrUnavailable
				}
				dbRows.Close()
				return previewPlan{}, ErrUnavailable
			}
			existingCategories[key] = true
		}
		if dbRows.Err() != nil {
			dbRows.Close()
			return previewPlan{}, ErrUnavailable
		}
		dbRows.Close()
	} else {
		dbRows, err := a.db.QueryContext(ctx, `SELECT convert_from(phone,'UTF8') FROM staff_whitelist`)
		if err != nil {
			return previewPlan{}, ErrUnavailable
		}
		for dbRows.Next() {
			var phone string
			if dbRows.Scan(&phone) != nil {
				if dbRows.Err() != nil {
					dbRows.Close()
					return previewPlan{}, ErrUnavailable
				}
				dbRows.Close()
				return previewPlan{}, ErrUnavailable
			}
			existingStaff[phone] = true
		}
		if dbRows.Err() != nil {
			dbRows.Close()
			return previewPlan{}, ErrUnavailable
		}
		dbRows.Close()
	}
	return planRows(kind, rows, header, plan, existingProducts, existingCategories, existingStaff)
}
func (a *PostgreSQLApplication) Commit(ctx context.Context, meta WriteMeta, token string, skip bool) (CommitResult, error) {
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
	defer tx.Rollback()
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
	err = tx.QueryRowContext(ctx, `SELECT id,kind,preview_json,state,expires_at,new_count,update_count,error_count,commit_idempotency_key_hash FROM import_batches WHERE preview_token_hash=$1 AND actor_account_id=$2 FOR UPDATE`, tokenHash[:], accountID).Scan(&id, &kind, &raw, &state, &expires, &newCount, &updateCount, &errorCount, &committedKey)
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
	res, err := tx.ExecContext(ctx, `UPDATE import_batches SET preview_json=$1,state='COMMITTED',commit_idempotency_key_hash=$2,skip_invalid=$3,result_json=$4,committed_at=clock_timestamp(),record_version=record_version+1,updated_at=clock_timestamp() WHERE id=$5 AND state='PREVIEWED'`, preview, operation[:], skip, resultRaw, id)
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
func (a *PostgreSQLApplication) commitStaff(ctx context.Context, tx *sql.Tx, plan previewPlan) error {
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
	if rows.Err() != nil {
		rows.Close()
		return ErrUnavailable
	}
	rows.Close()
	for _, r := range plan.Rows {
		if r.Outcome == "ERROR" {
			continue
		}
		name, key := staffName(r.Name)
		if r.Outcome == "ADD" {
			_, err = tx.ExecContext(ctx, `INSERT INTO staff_whitelist(phone,name,name_key,enabled,record_version,created_at,updated_at) VALUES($1,$2,$3,TRUE,1,clock_timestamp(),clock_timestamp())`, []byte(r.Phone), name, key)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE staff_whitelist SET name=$1,name_key=$2,record_version=record_version+1,updated_at=clock_timestamp() WHERE phone=$3`, name, key, []byte(r.Phone))
		}
		if err != nil {
			return importSQLError(err)
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE discount_settings SET whitelist_version=whitelist_version+1,updated_at=clock_timestamp() WHERE id=1`)
	if err != nil {
		return ErrUnavailable
	}
	return nil
}
func (a *PostgreSQLApplication) commitProducts(ctx context.Context, tx *sql.Tx, plan previewPlan) error {
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
			if rows.Err() != nil {
				rows.Close()
				return ErrUnavailable
			}
			rows.Close()
			return ErrUnavailable
		}
		categories[string(key)] = id
	}
	if rows.Err() != nil {
		rows.Close()
		return ErrUnavailable
	}
	rows.Close()
	for _, name := range plan.NewCategories {
		key := catalogKey(name)
		var id uint64
		err := tx.QueryRowContext(ctx, `INSERT INTO categories(name,name_key,sort_order,is_active,record_version) SELECT $1,$2,COALESCE(MAX(sort_order),0)+1,TRUE,1 FROM categories RETURNING id`, name, string(key)).Scan(&id)
		if err != nil {
			return importSQLError(err)
		}
		categories[string(key)] = uint64(id)
	}
	rows, err = tx.QueryContext(ctx, `SELECT id FROM products ORDER BY id FOR UPDATE`)
	if err != nil {
		return ErrUnavailable
	}
	for rows.Next() {
	}
	if rows.Err() != nil {
		rows.Close()
		return ErrUnavailable
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
		_, err = tx.ExecContext(ctx, `INSERT INTO products(category_id,name,name_key,description,specification,images_json,price_cents,sort_order,is_listed,meal_period,record_version) SELECT $1,$2,$3,$4,'','[]'::jsonb,$5,COALESCE(MAX(sort_order),0)+1,TRUE,$6,1 FROM products WHERE category_id=$7`, categoryID, r.Name, string(catalogKey(r.Name)), r.Description, r.PriceCents, r.Meal, categoryID)
		if err != nil {
			return importSQLError(err)
		}
	}
	return nil
}
func randomToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
func importSQLError(err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && (pgError.Code == "23505" || pgError.Code == "23503" || pgError.Code == "23514") {
		return ErrInvalidInput
	}
	return ErrUnavailable
}
