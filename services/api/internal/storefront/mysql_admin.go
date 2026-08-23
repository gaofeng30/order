package storefront

import (
	"context"
	"database/sql"
	"errors"
	"sort"

	"github.com/gaofeng30/order/services/api/internal/audit"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

type MySQLAdminApplication struct {
	db       *sql.DB
	receipts *audit.ReceiptStore
}

func NewMySQLAdminApplication(db *sql.DB) *MySQLAdminApplication {
	return &MySQLAdminApplication{db: db, receipts: audit.NewReceiptStore(db)}
}

type storefrontQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (a *MySQLAdminApplication) Admin(ctx context.Context) (AdminSettings, error) {
	return readAdminSettings(ctx, a.db)
}
func readAdminSettings(ctx context.Context, q storefrontQueryer) (AdminSettings, error) {
	var s AdminSettings
	var status string
	if q.QueryRowContext(ctx, `SELECT store_name,pickup_point,announcement,business_status,DATE_FORMAT(DATE(CONVERT_TZ(UTC_TIMESTAMP(6),'+00:00','+08:00')),'%Y-%m-%d') FROM storefront_settings WHERE id=1`).Scan(&s.StoreName, &s.PickupPoint, &s.Notice, &status, &s.ServiceDate) != nil {
		return AdminSettings{}, ErrAdminUnavailable
	}
	s.StoreStatus = status
	rows, err := q.QueryContext(ctx, `SELECT code,TIME_FORMAT(cutoff_time,'%H:%i'),TIME_FORMAT(pickup_start_time,'%H:%i'),TIME_FORMAT(pickup_end_time,'%H:%i'),interval_minutes FROM meal_periods ORDER BY FIELD(code,'lunch','dinner')`)
	if err != nil {
		return AdminSettings{}, ErrAdminUnavailable
	}
	for rows.Next() {
		var m MealPeriodConfig
		var interval uint16
		if rows.Scan(&m.Code, &m.CutoffTime, &m.PickupFrom, &m.PickupTo, &interval) != nil {
			rows.Close()
			return AdminSettings{}, ErrAdminUnavailable
		}
		m.Name = map[string]string{"lunch": "午餐", "dinner": "晚餐"}[m.Code]
		if s.PickupStepMin == 0 {
			s.PickupStepMin = interval
		} else if s.PickupStepMin != interval {
			rows.Close()
			return AdminSettings{}, ErrAdminUnavailable
		}
		s.MealPeriods = append(s.MealPeriods, m)
	}
	rows.Close()
	if len(s.MealPeriods) != 2 {
		return AdminSettings{}, ErrAdminUnavailable
	}
	rows, err = q.QueryContext(ctx, `SELECT DATE_FORMAT(service_date,'%Y-%m-%d'),IF(is_open,'open','closed') FROM service_dates WHERE service_date>=DATE(CONVERT_TZ(UTC_TIMESTAMP(6),'+00:00','+08:00')) ORDER BY service_date LIMIT 31`)
	if err != nil {
		return AdminSettings{}, ErrAdminUnavailable
	}
	for rows.Next() {
		var d ServiceDateConfig
		if rows.Scan(&d.Date, &d.Status) != nil {
			rows.Close()
			return AdminSettings{}, ErrAdminUnavailable
		}
		s.ServiceDates = append(s.ServiceDates, d)
	}
	rows.Close()
	return s, nil
}
func (a *MySQLAdminApplication) Configure(ctx context.Context, meta WriteMeta, cmd SettingsCommand) (AdminSettings, error) {
	if len(cmd.MealPeriods) != 2 || cmd.PickupStepMin == 0 {
		return AdminSettings{}, ErrAdminInvalidInput
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return AdminSettings{}, ErrAdminUnavailable
	}
	accountID, role, authVersion, err := audit.LockOwner(ctx, tx, meta.ActorUserID)
	if err != nil {
		tx.Rollback()
		return AdminSettings{}, ErrAdminUnavailable
	}
	var singleton uint8
	if err = tx.QueryRowContext(ctx, `SELECT id FROM storefront_settings WHERE id=1 FOR UPDATE`).Scan(&singleton); err != nil {
		tx.Rollback()
		return AdminSettings{}, ErrAdminUnavailable
	}
	dates := append([]ServiceDateConfig(nil), cmd.ServiceDates...)
	sort.Slice(dates, func(i, j int) bool { return dates[i].Date < dates[j].Date })
	for _, d := range dates {
		if d.Date == "" || (d.Status != "open" && d.Status != "closed") {
			tx.Rollback()
			return AdminSettings{}, ErrAdminInvalidInput
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO service_dates(service_date,is_open,record_version,updated_by_account_id,updated_at) VALUES(?,?,1,?,UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE is_open=VALUES(is_open),record_version=record_version+1,updated_by_account_id=VALUES(updated_by_account_id),updated_at=VALUES(updated_at)`, d.Date, d.Status == "open", accountID); err != nil {
			tx.Rollback()
			return AdminSettings{}, storefrontSQLError(err)
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT code FROM meal_periods WHERE code IN ('lunch','dinner') ORDER BY FIELD(code,'lunch','dinner') FOR UPDATE`)
	if err != nil {
		tx.Rollback()
		return AdminSettings{}, ErrAdminUnavailable
	}
	count := 0
	for rows.Next() {
		count++
	}
	rows.Close()
	if count != 2 {
		tx.Rollback()
		return AdminSettings{}, ErrAdminUnavailable
	}
	if _, err = tx.ExecContext(ctx, `UPDATE storefront_settings SET pickup_point=?,announcement=?,business_status=?,record_version=record_version+1 WHERE id=1`, cmd.PickupPoint, cmd.Notice, normalizeStatus(cmd.StoreStatus)); err != nil {
		tx.Rollback()
		return AdminSettings{}, storefrontSQLError(err)
	}
	for _, m := range cmd.MealPeriods {
		if m.Code != "lunch" && m.Code != "dinner" {
			tx.Rollback()
			return AdminSettings{}, ErrAdminInvalidInput
		}
		if _, err = tx.ExecContext(ctx, `UPDATE meal_periods SET cutoff_time=?,pickup_start_time=?,pickup_end_time=?,interval_minutes=? WHERE code=?`, m.CutoffTime, m.PickupFrom, m.PickupTo, cmd.PickupStepMin, m.Code); err != nil {
			tx.Rollback()
			return AdminSettings{}, storefrontSQLError(err)
		}
	}
	result, err := readAdminSettings(ctx, tx)
	if err != nil {
		tx.Rollback()
		return AdminSettings{}, err
	}
	rm := audit.CommandMeta{ActorUserID: meta.ActorUserID, ActorAccountID: accountID, ActorRole: role, ActorAuthVersion: authVersion, IdempotencyKey: meta.IdempotencyKey, RequestID: meta.RequestID}
	err = a.receipts.AppendInTx(ctx, tx, rm, "CONFIGURE_STOREFRONT", "STOREFRONT", 1, cmd, result)
	if errors.Is(err, audit.ErrDuplicateReceipt) {
		tx.Rollback()
		var replay AdminSettings
		ok, re := a.receipts.Replay(ctx, meta.ActorUserID, accountID, "CONFIGURE_STOREFRONT", meta.IdempotencyKey, cmd, &replay)
		if errors.Is(re, audit.ErrIdempotencyConflict) {
			return AdminSettings{}, ErrAdminIdempotencyConflict
		}
		if re != nil || !ok {
			return AdminSettings{}, ErrAdminUnavailable
		}
		return replay, nil
	}
	if err != nil || tx.Commit() != nil {
		return AdminSettings{}, ErrAdminUnavailable
	}
	return result, nil
}
func (a *MySQLAdminApplication) LaunchLayer(ctx context.Context) (LaunchLayerConfig, error) {
	var l LaunchLayerConfig
	var key sql.NullString
	var size, cx, cy, ar sql.NullFloat64
	err := a.db.QueryRowContext(ctx, `SELECT CONVERT(launch_image_object_key USING utf8mb4),width_ratio,center_x,center_y,aspect_ratio FROM storefront_settings WHERE id=1`).Scan(&key, &size, &cx, &cy, &ar)
	if err != nil {
		return LaunchLayerConfig{}, ErrAdminUnavailable
	}
	present := 0
	for _, v := range []bool{key.Valid, size.Valid, cx.Valid, cy.Valid, ar.Valid} {
		if v {
			present++
		}
	}
	if present == 0 {
		return l, nil
	}
	if present != 5 {
		return LaunchLayerConfig{}, ErrAdminUnavailable
	}
	return LaunchLayerConfig{ImageObjectKey: key.String, Enabled: true, SizeRatio: size.Float64, CenterX: cx.Float64, CenterY: cy.Float64, AspectRatio: ar.Float64}, nil
}
func (a *MySQLAdminApplication) ConfigureLaunchLayer(ctx context.Context, meta WriteMeta, layer *LaunchLayerConfig) (*LaunchLayerConfig, error) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, ErrAdminUnavailable
	}
	accountID, role, authVersion, err := audit.LockOwner(ctx, tx, meta.ActorUserID)
	if err != nil {
		tx.Rollback()
		return nil, ErrAdminUnavailable
	}
	var singleton uint8
	if err = tx.QueryRowContext(ctx, `SELECT id FROM storefront_settings WHERE id=1 FOR UPDATE`).Scan(&singleton); err != nil {
		tx.Rollback()
		return nil, ErrAdminUnavailable
	}
	var result LaunchLayerConfig
	if layer == nil || !layer.Enabled {
		_, err = tx.ExecContext(ctx, `UPDATE storefront_settings SET launch_image_object_key=NULL,center_x=NULL,center_y=NULL,width_ratio=NULL,aspect_ratio=NULL,record_version=record_version+1 WHERE id=1`)
	} else {
		result = *layer
		_, err = tx.ExecContext(ctx, `UPDATE storefront_settings SET launch_image_object_key=?,center_x=?,center_y=?,width_ratio=?,aspect_ratio=?,record_version=record_version+1 WHERE id=1`, layer.ImageObjectKey, layer.CenterX, layer.CenterY, layer.SizeRatio, layer.AspectRatio)
	}
	if err != nil {
		tx.Rollback()
		return nil, storefrontSQLError(err)
	}
	rm := audit.CommandMeta{ActorUserID: meta.ActorUserID, ActorAccountID: accountID, ActorRole: role, ActorAuthVersion: authVersion, IdempotencyKey: meta.IdempotencyKey, RequestID: meta.RequestID}
	action := "CONFIGURE_LAUNCH_LAYER"
	err = a.receipts.AppendInTx(ctx, tx, rm, action, "STOREFRONT", 1, layer, result)
	if errors.Is(err, audit.ErrDuplicateReceipt) {
		tx.Rollback()
		var replay LaunchLayerConfig
		ok, re := a.receipts.Replay(ctx, meta.ActorUserID, accountID, action, meta.IdempotencyKey, layer, &replay)
		if errors.Is(re, audit.ErrIdempotencyConflict) {
			return nil, ErrAdminIdempotencyConflict
		}
		if re != nil || !ok {
			return nil, ErrAdminUnavailable
		}
		return &replay, nil
	}
	if err != nil || tx.Commit() != nil {
		return nil, ErrAdminUnavailable
	}
	return &result, nil
}
func normalizeStatus(v string) string {
	if mapped := map[string]string{"营业中": "open", "休息中": "closed", "已截单": "cutoff"}[v]; mapped != "" {
		return mapped
	}
	return v
}
func storefrontSQLError(err error) error {
	var me *mysqlDriver.MySQLError
	if errors.As(err, &me) && (me.Number == 1062 || me.Number == 1451 || me.Number == 1452 || me.Number == 3819) {
		return ErrAdminConflict
	}
	return ErrAdminUnavailable
}
