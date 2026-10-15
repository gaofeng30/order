package storefront

import (
	"context"
	"database/sql"
	"errors"
	"sort"

	"github.com/gaofeng30/order/services/api/internal/audit"
	"github.com/jackc/pgx/v5/pgconn"
)

type PostgreSQLAdminApplication struct {
	db        *sql.DB
	receipts  *audit.ReceiptStore
	confirmed ObjectConfirmationReader
}

type ObjectConfirmationReader interface {
	Confirmed(context.Context, []string) (map[string]bool, error)
}

func NewPostgreSQLAdminApplication(db *sql.DB, readers ...ObjectConfirmationReader) *PostgreSQLAdminApplication {
	application := &PostgreSQLAdminApplication{db: db, receipts: audit.NewReceiptStore(db)}
	if len(readers) == 1 {
		application.confirmed = readers[0]
	}
	return application
}

type storefrontQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (a *PostgreSQLAdminApplication) Admin(ctx context.Context) (AdminSettings, error) {
	return readAdminSettings(ctx, a.db)
}
func readAdminSettings(ctx context.Context, q storefrontQueryer) (AdminSettings, error) {
	var s AdminSettings
	var status string
	if q.QueryRowContext(ctx, `SELECT store_name,pickup_point,announcement,business_status,to_char(clock_timestamp() AT TIME ZONE 'Asia/Shanghai','YYYY-MM-DD') FROM storefront_settings WHERE id=1`).Scan(&s.StoreName, &s.PickupPoint, &s.Notice, &status, &s.ServiceDate) != nil {
		return AdminSettings{}, ErrAdminUnavailable
	}
	s.StoreStatus = status
	rows, err := q.QueryContext(ctx, `SELECT code,to_char(DATE '2000-01-01'+cutoff_time,'HH24:MI'),to_char(DATE '2000-01-01'+pickup_start_time,'HH24:MI'),to_char(DATE '2000-01-01'+pickup_end_time,'HH24:MI'),interval_minutes FROM meal_periods ORDER BY CASE code WHEN 'lunch' THEN 1 WHEN 'dinner' THEN 2 ELSE 3 END`)
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
	rows, err = q.QueryContext(ctx, `SELECT to_char(service_date,'YYYY-MM-DD'),CASE WHEN is_open THEN 'open' ELSE 'closed' END FROM service_dates WHERE service_date>=(clock_timestamp() AT TIME ZONE 'Asia/Shanghai')::date ORDER BY service_date LIMIT 31`)
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
func (a *PostgreSQLAdminApplication) Configure(ctx context.Context, meta WriteMeta, cmd SettingsCommand) (AdminSettings, error) {
	if len(cmd.MealPeriods) != 2 || cmd.PickupStepMin == 0 {
		return AdminSettings{}, ErrAdminInvalidInput
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return AdminSettings{}, ErrAdminUnavailable
	}
	defer tx.Rollback()
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
		if _, err = tx.ExecContext(ctx, `INSERT INTO service_dates(service_date,is_open,record_version,updated_by_account_id,updated_at) VALUES($1,$2,1,$3,clock_timestamp()) ON CONFLICT (service_date) DO UPDATE SET is_open=EXCLUDED.is_open,record_version=service_dates.record_version+1,updated_by_account_id=EXCLUDED.updated_by_account_id,updated_at=EXCLUDED.updated_at`, d.Date, d.Status == "open", accountID); err != nil {
			tx.Rollback()
			return AdminSettings{}, storefrontSQLError(err)
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT code FROM meal_periods WHERE code IN ('lunch','dinner') ORDER BY CASE code WHEN 'lunch' THEN 1 WHEN 'dinner' THEN 2 ELSE 3 END FOR UPDATE`)
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
	if _, err = tx.ExecContext(ctx, `UPDATE storefront_settings SET pickup_point=$1,announcement=$2,business_status=$3,record_version=record_version+1 WHERE id=1`, cmd.PickupPoint, cmd.Notice, normalizeStatus(cmd.StoreStatus)); err != nil {
		tx.Rollback()
		return AdminSettings{}, storefrontSQLError(err)
	}
	for _, m := range cmd.MealPeriods {
		if m.Code != "lunch" && m.Code != "dinner" {
			tx.Rollback()
			return AdminSettings{}, ErrAdminInvalidInput
		}
		if _, err = tx.ExecContext(ctx, `UPDATE meal_periods SET cutoff_time=$1,pickup_start_time=$2,pickup_end_time=$3,interval_minutes=$4 WHERE code=$5`, m.CutoffTime, m.PickupFrom, m.PickupTo, cmd.PickupStepMin, m.Code); err != nil {
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
func (a *PostgreSQLAdminApplication) LaunchLayer(ctx context.Context) (LaunchLayerConfig, error) {
	var l LaunchLayerConfig
	var key sql.NullString
	var size, cx, cy, ar sql.NullFloat64
	err := a.db.QueryRowContext(ctx, `SELECT convert_from(launch_image_object_key,'UTF8'),width_ratio,center_x,center_y,aspect_ratio FROM storefront_settings WHERE id=1`).Scan(&key, &size, &cx, &cy, &ar)
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
func (a *PostgreSQLAdminApplication) ConfigureLaunchLayer(ctx context.Context, meta WriteMeta, layer *LaunchLayerConfig) (*LaunchLayerConfig, error) {
	// An immutable image receipt can be read before the owner-lock transaction,
	// so publishing the layer does not require a second pool connection.
	confirmed := true
	var confirmErr error
	if layer != nil && layer.Enabled && a.confirmed != nil {
		confirmed, confirmErr = a.confirmedLaunchImage(ctx, layer.ImageObjectKey)
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, ErrAdminUnavailable
	}
	accountID, role, authVersion, err := audit.LockOwner(ctx, tx, meta.ActorUserID)
	if err != nil {
		tx.Rollback()
		return nil, ErrAdminUnavailable
	}
	defer tx.Rollback()
	if layer != nil && layer.Enabled && a.confirmed != nil {
		if confirmErr != nil {
			return nil, ErrAdminUnavailable
		}
		if !confirmed {
			return nil, ErrAdminInvalidInput
		}
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
		_, err = tx.ExecContext(ctx, `UPDATE storefront_settings SET launch_image_object_key=$1,center_x=$2,center_y=$3,width_ratio=$4,aspect_ratio=$5,record_version=record_version+1 WHERE id=1`, []byte(layer.ImageObjectKey), layer.CenterX, layer.CenterY, layer.SizeRatio, layer.AspectRatio)
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
func (a *PostgreSQLAdminApplication) confirmedLaunchImage(ctx context.Context, key string) (bool, error) {
	if a.confirmed == nil {
		return true, nil
	}
	confirmed, err := a.confirmed.Confirmed(ctx, []string{key})
	if err != nil {
		return false, err
	}
	return confirmed[key], nil
}
func normalizeStatus(v string) string {
	if mapped := map[string]string{"营业中": "open", "休息中": "closed", "已截单": "cutoff"}[v]; mapped != "" {
		return mapped
	}
	return v
}
func storefrontSQLError(err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && (pgError.Code == "23505" || pgError.Code == "23503" || pgError.Code == "23514") {
		return ErrAdminConflict
	}
	return ErrAdminUnavailable
}
