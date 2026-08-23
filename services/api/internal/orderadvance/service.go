package orderadvance

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/gaofeng30/order/services/api/internal/orderproduction"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

const productionLeadTime = 30 * time.Minute

var (
	ErrInvalidInput = errors.New("order production invalid input")
	ErrUnavailable  = errors.New("order production unavailable")
)

type RunResult struct {
	Scanned  uint16
	Advanced uint16
}

type Service struct{ db *sql.DB }

func New(db *sql.DB) *Service { return &Service{db: db} }

// RunProductionDue advances due RESERVED orders without inventing a seventh
// state. A missed tick is compensated by the same <= threshold query later.
func (service *Service) RunProductionDue(ctx context.Context, now time.Time, limit uint16) (RunResult, error) {
	if ctx == nil || now.IsZero() || limit == 0 || limit > 100 {
		return RunResult{}, ErrInvalidInput
	}
	if service == nil || service.db == nil {
		return RunResult{}, ErrUnavailable
	}
	now = now.UTC().Truncate(time.Microsecond)
	rows, err := service.db.QueryContext(ctx, `SELECT id FROM orders WHERE state='RESERVED' AND pickup_at<=? ORDER BY pickup_at,id LIMIT ?`, now.Add(productionLeadTime), limit)
	if err != nil {
		return RunResult{}, ErrUnavailable
	}
	ids := make([]uint64, 0, limit)
	for rows.Next() {
		var id uint64
		if rows.Scan(&id) != nil || id == 0 {
			rows.Close()
			return RunResult{}, ErrUnavailable
		}
		ids = append(ids, id)
	}
	if rows.Err() != nil {
		rows.Close()
		return RunResult{}, ErrUnavailable
	}
	rows.Close()
	result := RunResult{Scanned: uint16(len(ids))}
	for _, id := range ids {
		advanced, err := service.advanceWithRetry(ctx, id, now)
		if err != nil {
			return result, ErrUnavailable
		}
		if advanced {
			result.Advanced++
		}
	}
	return result, nil
}

func (service *Service) advanceWithRetry(ctx context.Context, id uint64, now time.Time) (bool, error) {
	for attempt := 0; attempt < 2; attempt++ {
		advanced, err := service.advanceOnce(ctx, id, now)
		if err == nil {
			return advanced, nil
		}
		if !retryable(err) || attempt == 1 {
			return false, err
		}
	}
	return false, ErrUnavailable
}

func (service *Service) advanceOnce(ctx context.Context, id uint64, now time.Time) (bool, error) {
	transaction, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer transaction.Rollback()
	var state string
	var pickupAt, materializedAt time.Time
	var recordVersion uint64
	err = transaction.QueryRowContext(ctx, `SELECT state,pickup_at,materialized_at,record_version FROM orders WHERE id=? FOR UPDATE`, id).Scan(&state, &pickupAt, &materializedAt, &recordVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if state != string(orderproduction.StateReserved) {
		return false, nil
	}
	due, err := productionDecision(now, pickupAt.UTC())
	if err != nil || now.Before(materializedAt.UTC()) || recordVersion == 0 {
		return false, ErrUnavailable
	}
	if !due {
		return false, nil
	}
	update, err := transaction.ExecContext(ctx, `UPDATE orders SET state='PREPARING',preparing_at=?,record_version=record_version+1,updated_at=? WHERE id=? AND state='RESERVED' AND record_version=?`, now, now, id, recordVersion)
	if err != nil {
		return false, err
	}
	changed, err := update.RowsAffected()
	if err != nil || changed != 1 {
		return false, ErrUnavailable
	}
	before, _ := json.Marshal(struct {
		State string `json:"state"`
	}{State: string(orderproduction.StateReserved)})
	after, _ := json.Marshal(struct {
		State       string    `json:"state"`
		PreparingAt time.Time `json:"preparing_at"`
	}{State: string(orderproduction.StatePreparing), PreparingAt: now})
	scope := sha256.Sum256([]byte("SYSTEM:ORDER_PRODUCTION"))
	if _, err := transaction.ExecContext(ctx, `INSERT INTO action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,action,target_type,target_id,operation_key_hash,result,reason_code,before_state_json,after_state_json,occurred_at) VALUES('SYSTEM_EVIDENCE','SYSTEM',?,NULL,NULL,'order.production_due','ORDER',?,NULL,'SUCCEEDED','PRODUCTION_STARTED',?,?,?)`, scope[:], id, before, after, now); err != nil {
		return false, err
	}
	if err := transaction.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func productionDecision(now, pickupAt time.Time) (bool, error) {
	decision, err := orderproduction.Advance(orderproduction.StateReserved, now, pickupAt)
	if err != nil {
		return false, err
	}
	return decision.Changed && decision.State == orderproduction.StatePreparing, nil
}

func retryable(err error) bool {
	var mysqlError *mysqlDriver.MySQLError
	return errors.As(err, &mysqlError) && (mysqlError.Number == 1205 || mysqlError.Number == 1213)
}
