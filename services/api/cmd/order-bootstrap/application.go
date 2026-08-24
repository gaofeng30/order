package main

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"
)

var (
	errBootstrapConflict    = errors.New("bootstrap state conflicts with requested initial state")
	errBootstrapUnavailable = errors.New("bootstrap storage unavailable")
)

var bootstrapFlavorOptions = [...]string{"少饭", "加饭", "少盐", "加辣", "酱汁分装", "免葱蒜", "打包分装", "多双餐具"}

type bootstrapError struct {
	kind  error
	cause error
}

func (err bootstrapError) Error() string { return err.kind.Error() }
func (err bootstrapError) Unwrap() error { return err.kind }

type bootstrapState struct {
	storefront *storefrontBootstrapState
	discount   *discountBootstrapState
	owners     []ownerBootstrapState
}

type storefrontBootstrapState struct {
	storeName, storeAddress, pickupPoint, announcement, status string
	launchNulls                                                bool
	flavorType                                                 string
	flavorCount                                                int
	flavors                                                    [len(bootstrapFlavorOptions)]string
	recordVersion                                              uint64
}

type discountBootstrapState struct {
	ratePercent                       int
	discountVersion, whitelistVersion uint64
}

type ownerBootstrapState struct {
	phone, name                string
	enabled                    bool
	recordVersion, authVersion uint64
	nullDefaults, timesEqual   bool
}

func bootstrap(ctx context.Context, db *sql.DB, input bootstrapInput) (bootstrapOutcome, error) {
	for attempt := 0; attempt < 2; attempt++ {
		outcome, err := bootstrapOnce(ctx, db, input)
		if err == nil {
			return outcome, nil
		}
		if attempt == 0 && retryableBootstrapTransactionError(err) {
			continue
		}
		return "", err
	}
	return "", errBootstrapUnavailable
}

func bootstrapOnce(ctx context.Context, db *sql.DB, input bootstrapInput) (bootstrapOutcome, error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return "", unavailableBootstrapError(err)
	}
	outcome, err := bootstrapInTransaction(ctx, tx, input)
	if err != nil {
		_ = tx.Rollback()
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", unavailableBootstrapError(err)
	}
	return outcome, nil
}

func bootstrapInTransaction(ctx context.Context, tx *sql.Tx, input bootstrapInput) (bootstrapOutcome, error) {
	state, err := readBootstrapStateForUpdate(ctx, tx)
	if err != nil {
		return "", unavailableBootstrapError(err)
	}
	allEmpty := state.storefront == nil && state.discount == nil && len(state.owners) == 0
	if !allEmpty {
		if state.exactlyMatches(input) {
			return outcomeUnchanged, nil
		}
		return "", errBootstrapConflict
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO storefront_settings(
			id,store_name,store_address,pickup_point,announcement,business_status,
			launch_image_object_key,center_x,center_y,width_ratio,aspect_ratio,flavor_options_json,record_version
		) VALUES(1,?,?,?,'','closed',NULL,NULL,NULL,NULL,NULL,JSON_ARRAY(?,?,?,?,?,?,?,?),1)
	`, input.StoreName, input.StoreAddress, input.PickupPoint,
		bootstrapFlavorOptions[0], bootstrapFlavorOptions[1], bootstrapFlavorOptions[2], bootstrapFlavorOptions[3],
		bootstrapFlavorOptions[4], bootstrapFlavorOptions[5], bootstrapFlavorOptions[6], bootstrapFlavorOptions[7]); err != nil {
		return "", unavailableBootstrapError(err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO discount_settings(id,rate_percent,discount_version,whitelist_version,updated_at)
		VALUES(1,100,1,1,UTC_TIMESTAMP(6))
	`); err != nil {
		return "", unavailableBootstrapError(err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO merchant_accounts(
			phone,name,role,enabled,record_version,auth_version,bound_user_id,bound_at,
			deleted_at,deleted_by_account_id,created_at,updated_at,created_by,updated_by
		) VALUES(?,?,'OWNER',TRUE,1,1,NULL,NULL,NULL,NULL,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),NULL,NULL)
	`, input.OwnerPhone, input.OwnerName); err != nil {
		return "", unavailableBootstrapError(err)
	}
	return outcomeCreated, nil
}

func unavailableBootstrapError(cause error) error {
	return bootstrapError{kind: errBootstrapUnavailable, cause: cause}
}

func retryableBootstrapTransactionError(err error) bool {
	var wrapped bootstrapError
	if !errors.As(err, &wrapped) {
		return false
	}
	var mysqlError *mysql.MySQLError
	if !errors.As(wrapped.cause, &mysqlError) {
		return false
	}
	return mysqlError.Number == 1062 || mysqlError.Number == 1205 || mysqlError.Number == 1213
}

func readBootstrapStateForUpdate(ctx context.Context, tx *sql.Tx) (bootstrapState, error) {
	state := bootstrapState{}
	storefront := storefrontBootstrapState{}
	err := tx.QueryRowContext(ctx, `
		SELECT store_name,store_address,pickup_point,announcement,business_status,
		       (launch_image_object_key IS NULL AND center_x IS NULL AND center_y IS NULL AND width_ratio IS NULL AND aspect_ratio IS NULL),
		       JSON_TYPE(flavor_options_json),JSON_LENGTH(flavor_options_json),
		       COALESCE(JSON_UNQUOTE(JSON_EXTRACT(flavor_options_json,'$[0]')),''),
		       COALESCE(JSON_UNQUOTE(JSON_EXTRACT(flavor_options_json,'$[1]')),''),
		       COALESCE(JSON_UNQUOTE(JSON_EXTRACT(flavor_options_json,'$[2]')),''),
		       COALESCE(JSON_UNQUOTE(JSON_EXTRACT(flavor_options_json,'$[3]')),''),
		       COALESCE(JSON_UNQUOTE(JSON_EXTRACT(flavor_options_json,'$[4]')),''),
		       COALESCE(JSON_UNQUOTE(JSON_EXTRACT(flavor_options_json,'$[5]')),''),
		       COALESCE(JSON_UNQUOTE(JSON_EXTRACT(flavor_options_json,'$[6]')),''),
		       COALESCE(JSON_UNQUOTE(JSON_EXTRACT(flavor_options_json,'$[7]')),''),record_version
		FROM storefront_settings WHERE id=1 FOR UPDATE
	`).Scan(
		&storefront.storeName, &storefront.storeAddress, &storefront.pickupPoint, &storefront.announcement, &storefront.status,
		&storefront.launchNulls, &storefront.flavorType, &storefront.flavorCount,
		&storefront.flavors[0], &storefront.flavors[1], &storefront.flavors[2], &storefront.flavors[3],
		&storefront.flavors[4], &storefront.flavors[5], &storefront.flavors[6], &storefront.flavors[7], &storefront.recordVersion,
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return bootstrapState{}, err
	}
	if err == nil {
		state.storefront = &storefront
	}

	discount := discountBootstrapState{}
	err = tx.QueryRowContext(ctx, `
		SELECT rate_percent,discount_version,whitelist_version
		FROM discount_settings WHERE id=1 FOR UPDATE
	`).Scan(&discount.ratePercent, &discount.discountVersion, &discount.whitelistVersion)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return bootstrapState{}, err
	}
	if err == nil {
		state.discount = &discount
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT CONVERT(phone USING ascii),name,enabled,record_version,auth_version,
		       (bound_user_id IS NULL AND bound_at IS NULL AND created_by IS NULL AND updated_by IS NULL AND deleted_at IS NULL AND deleted_by_account_id IS NULL),
		       (created_at=updated_at)
		FROM merchant_accounts WHERE role='OWNER' ORDER BY id FOR UPDATE
	`)
	if err != nil {
		return bootstrapState{}, err
	}
	defer rows.Close()
	for rows.Next() {
		owner := ownerBootstrapState{}
		if err := rows.Scan(&owner.phone, &owner.name, &owner.enabled, &owner.recordVersion, &owner.authVersion, &owner.nullDefaults, &owner.timesEqual); err != nil {
			return bootstrapState{}, err
		}
		state.owners = append(state.owners, owner)
	}
	if err := rows.Err(); err != nil {
		return bootstrapState{}, err
	}
	return state, nil
}

func (state bootstrapState) exactlyMatches(input bootstrapInput) bool {
	if state.storefront == nil || state.discount == nil || len(state.owners) != 1 {
		return false
	}
	storefront := state.storefront
	if storefront.storeName != input.StoreName || storefront.storeAddress != input.StoreAddress || storefront.pickupPoint != input.PickupPoint || storefront.announcement != "" || storefront.status != "closed" || !storefront.launchNulls || storefront.flavorType != "ARRAY" || storefront.flavorCount != len(bootstrapFlavorOptions) || storefront.flavors != bootstrapFlavorOptions || storefront.recordVersion != 1 {
		return false
	}
	discount := state.discount
	if discount.ratePercent != 100 || discount.discountVersion != 1 || discount.whitelistVersion != 1 {
		return false
	}
	owner := state.owners[0]
	return owner.phone == input.OwnerPhone && owner.name == input.OwnerName && owner.enabled && owner.recordVersion == 1 && owner.authVersion == 1 && owner.nullDefaults && owner.timesEqual
}
