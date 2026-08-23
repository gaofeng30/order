package merchantidentity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// Repository stores merchant identity state in MySQL.
type Repository struct {
	db     *sql.DB
	commit func(*sql.Tx) error
}

// NewRepository constructs the production merchant identity repository.
func NewRepository(db *sql.DB) *Repository {
	return newRepository(db, func(transaction *sql.Tx) error { return transaction.Commit() })
}

func newRepository(db *sql.DB, commit func(*sql.Tx) error) *Repository {
	return &Repository{db: db, commit: commit}
}

type accountState struct {
	ID            uint64
	Role          Role
	Enabled       bool
	RecordVersion uint64
	AuthVersion   uint64
}

// ReadIdentity returns the current user phone state and enabled merchant binding.
func (repository *Repository) ReadIdentity(ctx context.Context, userID uint64) (Identity, error) {
	if repository.db == nil || userID == 0 {
		return Identity{}, ErrUnavailable
	}
	var phone, extraPhone, extraName sql.NullString
	var phoneBoundAt sql.NullTime
	var extraPhoneSetAt sql.NullTime
	var extraNameKey, whitelistExtraNameKey []byte
	var recordVersion uint64
	var accountID, authVersion sql.NullInt64
	var role sql.NullString
	var ratePercent uint8
	var whitelistVersion uint64
	var primaryEnabled, extraEnabled sql.NullBool
	err := repository.db.QueryRowContext(ctx, `
		SELECT CONVERT(u.primary_phone USING ascii),u.primary_phone_bound_at,
		       CONVERT(u.extra_phone USING ascii),u.extra_name,u.extra_name_key,u.extra_phone_set_at,u.record_version,
		       a.id,a.role,a.auth_version,d.rate_percent,d.whitelist_version,
		       primary_staff.enabled,extra_staff.enabled,extra_staff.name_key
		FROM miniprogram_users AS u
		JOIN discount_settings AS d ON d.id=1
		LEFT JOIN merchant_accounts AS a ON a.bound_user_id=u.id AND a.enabled=TRUE AND a.deleted_at IS NULL
		LEFT JOIN staff_whitelist AS primary_staff ON primary_staff.phone=u.primary_phone
		LEFT JOIN staff_whitelist AS extra_staff ON extra_staff.phone=u.extra_phone
		WHERE u.id=?
	`, userID).Scan(
		&phone, &phoneBoundAt, &extraPhone, &extraName, &extraNameKey, &extraPhoneSetAt, &recordVersion,
		&accountID, &role, &authVersion, &ratePercent, &whitelistVersion,
		&primaryEnabled, &extraEnabled, &whitelistExtraNameKey,
	)
	if err != nil || !validPhoneState(phone, phoneBoundAt) || recordVersion == 0 || ratePercent < 1 || ratePercent > 100 || whitelistVersion == 0 {
		return Identity{}, ErrUnavailable
	}
	extraCount := 0
	for _, present := range []bool{extraPhone.Valid, extraName.Valid, extraNameKey != nil, extraPhoneSetAt.Valid} {
		if present {
			extraCount++
		}
	}
	if extraCount != 0 && extraCount != 4 || (!phone.Valid && extraCount != 0) {
		return Identity{}, ErrUnavailable
	}
	projection := Identity{
		PrimaryPhoneBound: phone.Valid,
		Pricing:           PricingProjection{Kind: PricingVisitor, RatePercent: 100},
	}
	if phone.Valid {
		projection.PrimaryPhoneMasked = maskIdentityPhone(phone.String)
	}
	if extraCount == 4 {
		_, expectedKey, ok := canonicalExtraIdentity(extraPhone.String, extraName.String)
		if !ok || !bytes.Equal(extraNameKey, expectedKey) {
			return Identity{}, ErrUnavailable
		}
		projection.ExtraPhone = &ExtraPhoneProjection{MaskedPhone: maskIdentityPhone(extraPhone.String), Name: extraName.String}
	}
	staff := primaryEnabled.Valid && primaryEnabled.Bool
	if !staff && projection.ExtraPhone != nil && extraEnabled.Valid && extraEnabled.Bool && bytes.Equal(extraNameKey, whitelistExtraNameKey) {
		staff = true
	}
	if staff {
		projection.Pricing = PricingProjection{Kind: PricingStaff, RatePercent: ratePercent}
	}
	if !accountID.Valid && !role.Valid && !authVersion.Valid {
		return projection, nil
	}
	if !accountID.Valid || accountID.Int64 <= 0 || !role.Valid || !authVersion.Valid || authVersion.Int64 <= 0 || !phone.Valid {
		return Identity{}, ErrUnavailable
	}
	parsedRole := Role(role.String)
	if !validRole(parsedRole) {
		return Identity{}, ErrUnavailable
	}
	projection.Merchant = &MerchantProjection{Role: parsedRole, AuthVersion: uint64(authVersion.Int64)}
	return projection, nil
}

// StartLogin audits and returns an existing binding, or returns only the provider subject.
func (repository *Repository) StartLogin(ctx context.Context, userID uint64, codeHash LoginCodeHash, requestID string, at time.Time) (result LoginStart, resultErr error) {
	if repository.db == nil || userID == 0 || !validLoginCodeHash(codeHash) || !validRequestID(requestID) || at.IsZero() {
		return LoginStart{}, ErrUnavailable
	}
	transaction, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return LoginStart{}, ErrUnavailable
	}
	defer func() {
		if resultErr != nil {
			_ = transaction.Rollback()
		}
	}()

	var openID string
	var phone sql.NullString
	var phoneBoundAt sql.NullTime
	if err := transaction.QueryRowContext(ctx, `
		SELECT openid,primary_phone,primary_phone_bound_at
		FROM miniprogram_users
		WHERE id=?
		FOR SHARE
	`, userID).Scan(&openID, &phone, &phoneBoundAt); err != nil || openID == "" || !validPhoneState(phone, phoneBoundAt) {
		return LoginStart{}, ErrUnavailable
	}
	account, found, err := readBoundAccount(ctx, transaction, userID, "FOR SHARE")
	if err != nil {
		return LoginStart{}, ErrUnavailable
	}
	if !found {
		if err := repository.commit(transaction); err != nil {
			return LoginStart{}, ErrUnavailable
		}
		return LoginStart{OpenID: openID}, nil
	}
	if !validAccount(account) {
		return LoginStart{}, ErrUnavailable
	}
	if !account.Enabled {
		if err := insertLoginAudit(ctx, transaction, userID, codeHash, requestID, &account, "REJECTED", "ACCOUNT_NOT_AVAILABLE", "BOUND_DISABLED", "BOUND_DISABLED", at); err != nil {
			return LoginStart{}, ErrUnavailable
		}
		if err := repository.commit(transaction); err != nil {
			return LoginStart{}, ErrUnavailable
		}
		return LoginStart{}, ErrMerchantAccountNotAvailable
	}
	if !phone.Valid {
		return LoginStart{}, ErrUnavailable
	}
	if err := insertLoginAudit(ctx, transaction, userID, codeHash, requestID, &account, "SUCCEEDED", "ALREADY_BOUND", "BOUND_ENABLED", "BOUND_ENABLED", at); err != nil {
		return LoginStart{}, ErrUnavailable
	}
	if err := repository.commit(transaction); err != nil {
		return LoginStart{}, ErrUnavailable
	}
	return LoginStart{
		AlreadyBound: true,
		Existing: Identity{
			PrimaryPhoneBound: true,
			Merchant:          &MerchantProjection{Role: account.Role, AuthVersion: account.AuthVersion},
		},
	}, nil
}

// CompleteLogin atomically binds the primary phone, account versions, and success audit.
func (repository *Repository) CompleteLogin(ctx context.Context, userID uint64, phone string, codeHash LoginCodeHash, requestID string, at time.Time) (Identity, error) {
	if repository.db == nil || userID == 0 || !canonicalPhone(phone) || !validLoginCodeHash(codeHash) || !validRequestID(requestID) || at.IsZero() {
		return Identity{}, ErrUnavailable
	}
	for attempt := 0; attempt < 2; attempt++ {
		projection, err := repository.completeLoginOnce(ctx, userID, phone, codeHash, requestID, at)
		if retryableTransaction(err) && attempt == 0 {
			continue
		}
		if err == nil || isBusinessError(err) {
			return projection, err
		}
		return Identity{}, ErrUnavailable
	}
	return Identity{}, ErrUnavailable
}

func (repository *Repository) completeLoginOnce(ctx context.Context, userID uint64, phone string, codeHash LoginCodeHash, requestID string, at time.Time) (result Identity, resultErr error) {
	transaction, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return Identity{}, err
	}
	defer func() {
		if resultErr != nil {
			_ = transaction.Rollback()
		}
	}()

	var currentPhone sql.NullString
	var currentPhoneBoundAt sql.NullTime
	if err := transaction.QueryRowContext(ctx, `
		SELECT primary_phone,primary_phone_bound_at
		FROM miniprogram_users
		WHERE id=?
		FOR UPDATE
	`, userID).Scan(&currentPhone, &currentPhoneBoundAt); err != nil || !validPhoneState(currentPhone, currentPhoneBoundAt) {
		if err == nil {
			err = ErrUnavailable
		}
		return Identity{}, err
	}

	boundAccount, found, err := readBoundAccount(ctx, transaction, userID, "FOR UPDATE")
	if err != nil {
		return Identity{}, err
	}
	if found {
		if !validAccount(boundAccount) || !currentPhone.Valid {
			return Identity{}, ErrUnavailable
		}
		if currentPhone.String != phone {
			state := "BOUND_ENABLED"
			if !boundAccount.Enabled {
				state = "BOUND_DISABLED"
			}
			return repository.finishLogin(ctx, transaction, userID, codeHash, requestID, &boundAccount, "REJECTED", "PRIMARY_PHONE_MISMATCH", state, state, at, Identity{}, ErrPrimaryPhoneMismatch)
		}
		if !boundAccount.Enabled {
			return repository.finishLogin(ctx, transaction, userID, codeHash, requestID, &boundAccount, "REJECTED", "ACCOUNT_NOT_AVAILABLE", "BOUND_DISABLED", "BOUND_DISABLED", at, Identity{}, ErrMerchantAccountNotAvailable)
		}
		projection := accountIdentity(boundAccount)
		return repository.finishLogin(ctx, transaction, userID, codeHash, requestID, &boundAccount, "SUCCEEDED", "CONCURRENT_BINDING_CONFIRMED", "BOUND_ENABLED", "BOUND_ENABLED", at, projection, nil)
	}

	account, found, err := readPhoneAccount(ctx, transaction, phone)
	if err != nil {
		return Identity{}, err
	}
	if !found {
		if currentPhone.Valid && currentPhone.String != phone {
			return repository.finishLogin(ctx, transaction, userID, codeHash, requestID, nil, "REJECTED", "PRIMARY_PHONE_MISMATCH", "UNRESOLVED", "UNRESOLVED", at, Identity{}, ErrPrimaryPhoneMismatch)
		}
		return repository.finishLogin(ctx, transaction, userID, codeHash, requestID, nil, "REJECTED", "ACCOUNT_NOT_AVAILABLE", "UNRESOLVED", "UNRESOLVED", at, Identity{}, ErrMerchantAccountNotAvailable)
	}
	if !validAccount(account) {
		return Identity{}, ErrUnavailable
	}

	var accountBoundUser sql.NullInt64
	if err := transaction.QueryRowContext(ctx, "SELECT bound_user_id FROM merchant_accounts WHERE id=?", account.ID).Scan(&accountBoundUser); err != nil {
		return Identity{}, err
	}
	if currentPhone.Valid && currentPhone.String != phone {
		state := "UNBOUND"
		if accountBoundUser.Valid {
			state = "BOUND_OTHER"
		} else if !account.Enabled {
			state = "UNBOUND_DISABLED"
		}
		return repository.finishLogin(ctx, transaction, userID, codeHash, requestID, &account, "REJECTED", "PRIMARY_PHONE_MISMATCH", state, state, at, Identity{}, ErrPrimaryPhoneMismatch)
	}
	if !account.Enabled {
		return repository.finishLogin(ctx, transaction, userID, codeHash, requestID, &account, "REJECTED", "ACCOUNT_NOT_AVAILABLE", "UNBOUND_DISABLED", "UNBOUND_DISABLED", at, Identity{}, ErrMerchantAccountNotAvailable)
	}
	if accountBoundUser.Valid {
		return repository.finishLogin(ctx, transaction, userID, codeHash, requestID, &account, "REJECTED", "ACCOUNT_NOT_AVAILABLE", "BOUND_OTHER", "BOUND_OTHER", at, Identity{}, ErrMerchantAccountNotAvailable)
	}
	if !currentPhone.Valid {
		if _, err := transaction.ExecContext(ctx, `
			UPDATE miniprogram_users
			SET primary_phone=?,primary_phone_bound_at=?
			WHERE id=? AND primary_phone IS NULL AND primary_phone_bound_at IS NULL
		`, phone, at, userID); err != nil {
			if duplicateKey(err) {
				return repository.finishLogin(ctx, transaction, userID, codeHash, requestID, &account, "REJECTED", "PHONE_IN_USE", "UNBOUND", "UNBOUND", at, Identity{}, ErrPhoneInUse)
			}
			return Identity{}, err
		}
	}

	update, err := transaction.ExecContext(ctx, `
		UPDATE merchant_accounts
		SET bound_user_id=?,bound_at=?,record_version=record_version+1,auth_version=auth_version+1,updated_at=?
		WHERE id=? AND bound_user_id IS NULL AND record_version=? AND auth_version=?
	`, userID, at, at, account.ID, account.RecordVersion, account.AuthVersion)
	if err != nil {
		return Identity{}, err
	}
	rows, err := update.RowsAffected()
	if err != nil || rows != 1 {
		if err == nil {
			err = ErrUnavailable
		}
		return Identity{}, err
	}
	account.RecordVersion++
	account.AuthVersion++
	projection := accountIdentity(account)
	return repository.finishLogin(ctx, transaction, userID, codeHash, requestID, &account, "SUCCEEDED", "FIRST_BINDING", "UNBOUND", "BOUND_ENABLED", at, projection, nil)
}

// RecoverRejectedLogin returns success only when a same-user, same-version success audit is visible.
func (repository *Repository) RecoverRejectedLogin(ctx context.Context, userID uint64, codeHash LoginCodeHash, requestID string, startedAt, at time.Time) (result Identity, resultErr error) {
	if repository.db == nil || userID == 0 || !validLoginCodeHash(codeHash) || !validRequestID(requestID) || startedAt.IsZero() || at.Before(startedAt) {
		return Identity{}, ErrUnavailable
	}
	transaction, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return Identity{}, ErrUnavailable
	}
	defer func() {
		if resultErr != nil {
			_ = transaction.Rollback()
		}
	}()
	var phone sql.NullString
	var phoneBoundAt sql.NullTime
	if err := transaction.QueryRowContext(ctx, `
		SELECT primary_phone,primary_phone_bound_at FROM miniprogram_users WHERE id=? FOR SHARE
	`, userID).Scan(&phone, &phoneBoundAt); err != nil || !validPhoneState(phone, phoneBoundAt) {
		return Identity{}, ErrUnavailable
	}
	account, found, err := readBoundAccount(ctx, transaction, userID, "FOR SHARE")
	if err != nil {
		return Identity{}, ErrUnavailable
	}
	if found && !validAccount(account) {
		return Identity{}, ErrUnavailable
	}
	confirmed := false
	if found && account.Enabled && phone.Valid {
		var marker uint8
		err := transaction.QueryRowContext(ctx, `
			SELECT 1
			FROM action_audits
			WHERE actor_user_id=?
			  AND actor_account_id_snapshot=?
			  AND actor_auth_version_snapshot=?
			  AND target_type='merchant_login_code'
			  AND target_key_hash=?
			  AND action='merchant.login'
			  AND result='SUCCEEDED'
			  AND reason_code IN ('FIRST_BINDING','CONCURRENT_BINDING_CONFIRMED')
			  AND occurred_at>=?
			ORDER BY id DESC
			LIMIT 1
		`, userID, account.ID, account.AuthVersion, codeHash[:], startedAt).Scan(&marker)
		confirmed = err == nil && marker == 1
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return Identity{}, ErrUnavailable
		}
	}
	if confirmed {
		projection := accountIdentity(account)
		return repository.finishLogin(ctx, transaction, userID, codeHash, requestID, &account, "SUCCEEDED", "CONCURRENT_BINDING_CONFIRMED", "BOUND_ENABLED", "BOUND_ENABLED", at, projection, nil)
	}
	state := "UNRESOLVED"
	if found {
		if account.Enabled {
			state = "BOUND_UNCONFIRMED"
		} else {
			state = "BOUND_DISABLED"
		}
	}
	return repository.finishLogin(ctx, transaction, userID, codeHash, requestID, nil, "REJECTED", "PHONE_CODE_REJECTED", state, state, at, Identity{}, ErrPhoneCodeRejected)
}

func (repository *Repository) finishLogin(ctx context.Context, transaction *sql.Tx, userID uint64, codeHash LoginCodeHash, requestID string, account *accountState, result, reason, before, after string, at time.Time, projection Identity, businessErr error) (Identity, error) {
	if err := insertLoginAudit(ctx, transaction, userID, codeHash, requestID, account, result, reason, before, after, at); err != nil {
		return Identity{}, err
	}
	if err := repository.commit(transaction); err != nil {
		return Identity{}, err
	}
	return projection, businessErr
}

func readBoundAccount(ctx context.Context, transaction *sql.Tx, userID uint64, lock string) (accountState, bool, error) {
	return scanAccount(transaction.QueryRowContext(ctx, `
		SELECT id,role,enabled,record_version,auth_version
		FROM merchant_accounts
		WHERE bound_user_id=?
		`+lock, userID))
}

func readPhoneAccount(ctx context.Context, transaction *sql.Tx, phone string) (accountState, bool, error) {
	return scanAccount(transaction.QueryRowContext(ctx, `
		SELECT id,role,enabled,record_version,auth_version
		FROM merchant_accounts
		WHERE phone=?
		FOR UPDATE
	`, phone))
}

type scanner interface {
	Scan(...any) error
}

func scanAccount(row scanner) (accountState, bool, error) {
	var account accountState
	var role string
	err := row.Scan(&account.ID, &role, &account.Enabled, &account.RecordVersion, &account.AuthVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return accountState{}, false, nil
	}
	if err != nil {
		return accountState{}, false, err
	}
	account.Role = Role(role)
	return account, true, nil
}

func insertLoginAudit(ctx context.Context, transaction *sql.Tx, userID uint64, codeHash LoginCodeHash, requestID string, account *accountState, result, reason, before, after string, at time.Time) error {
	var accountID, snapshotID, role, authVersion any
	if account != nil {
		accountID = account.ID
		snapshotID = account.ID
		role = account.Role
		authVersion = account.AuthVersion
	}
	scopeHash := merchantLoginScopeHash(userID)
	requestIDHash := sha256.Sum256([]byte(requestID))
	beforeJSON, err := json.Marshal(map[string]string{"state": before})
	if err != nil {
		return err
	}
	afterJSON, err := json.Marshal(map[string]string{"state": after})
	if err != nil {
		return err
	}
	_, err = transaction.ExecContext(ctx, `
		INSERT INTO action_audits(
			entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,
			actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,
			action,result,reason_code,target_type,target_key_hash,request_id_hash,
			before_state_json,after_state_json,occurred_at
		) VALUES ('LEGACY_EVIDENCE','MERCHANT',?,?,?,?,?,?,'merchant.login',?,?,'merchant_login_code',?,?,?, ?,?)
	`, scopeHash[:], userID, accountID, snapshotID, role, authVersion, result, reason, codeHash[:], requestIDHash[:], beforeJSON, afterJSON, at)
	return err
}

func merchantLoginScopeHash(userID uint64) [sha256.Size]byte {
	var material [23]byte
	copy(material[:15], "MERCHANT_LOGIN\x00")
	binary.BigEndian.PutUint64(material[15:], userID)
	return sha256.Sum256(material[:])
}

func accountIdentity(account accountState) Identity {
	return Identity{
		PrimaryPhoneBound: true,
		Merchant:          &MerchantProjection{Role: account.Role, AuthVersion: account.AuthVersion},
	}
}

func validPhoneState(phone sql.NullString, boundAt sql.NullTime) bool {
	return phone.Valid == boundAt.Valid && (!phone.Valid || canonicalPhone(phone.String))
}

func validRole(role Role) bool {
	return role == RoleOwner || role == RoleSubaccount
}

func validAccount(account accountState) bool {
	return account.ID > 0 && validRole(account.Role) && account.RecordVersion > 0 && account.AuthVersion > 0
}

func validRequestID(requestID string) bool {
	return requestID != "" && len(requestID) <= 64 && strings.TrimSpace(requestID) == requestID
}

func validLoginCodeHash(codeHash LoginCodeHash) bool {
	return codeHash != (LoginCodeHash{})
}

func duplicateKey(err error) bool {
	var mysqlError *mysql.MySQLError
	return errors.As(err, &mysqlError) && mysqlError.Number == 1062
}

func retryableTransaction(err error) bool {
	var mysqlError *mysql.MySQLError
	return errors.As(err, &mysqlError) && (mysqlError.Number == 1213 || mysqlError.Number == 1205)
}

func isBusinessError(err error) bool {
	return errors.Is(err, ErrMerchantAccountNotAvailable) ||
		errors.Is(err, ErrPhoneInUse) ||
		errors.Is(err, ErrPrimaryPhoneMismatch) ||
		errors.Is(err, ErrPhoneCodeRejected) ||
		errors.Is(err, ErrForbidden)
}
