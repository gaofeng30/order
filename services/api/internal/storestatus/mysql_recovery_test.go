package storestatus

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gaofeng30/order/services/api/internal/storefront"
)

func TestApplyAuditReadFailureRollsBackAndSameKeyRecovers(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 11, 5, 0, 456789000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-audit-recovery", now)
		insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		core := New(db, merchantidentity.NewRepository(db), func() time.Time { return now })
		command := Command{
			UserID: userID, DesiredStatus: storefront.BusinessClosed,
			IdempotencyKey: "audit-recovery-command", RequestID: "audit-recovery-request",
		}
		if _, err := db.ExecContext(context.Background(), "RENAME TABLE merchant_action_audits TO merchant_action_audits_unavailable"); err != nil {
			t.Fatal("hide audit table failed")
		}
		restored := false
		defer func() {
			if !restored {
				_, _ = db.ExecContext(context.Background(), "RENAME TABLE merchant_action_audits_unavailable TO merchant_action_audits")
			}
		}()

		result, err := core.Apply(context.Background(), command)
		if !errors.Is(err, ErrUnavailable) || result != (Result{}) {
			t.Fatalf("audit-failure Apply() = %#v, %v", result, err)
		}
		if got := readBusinessStatus(t, db); got != string(storefront.BusinessOpen) {
			t.Fatalf("audit failure committed business_status %q", got)
		}
		if _, err := db.ExecContext(context.Background(), "RENAME TABLE merchant_action_audits_unavailable TO merchant_action_audits"); err != nil {
			t.Fatal("restore audit table failed")
		}
		restored = true

		recovered, err := core.Apply(context.Background(), command)
		if err != nil || recovered != (Result{Before: storefront.BusinessOpen, After: storefront.BusinessClosed, Changed: true}) {
			t.Fatalf("recovered Apply() = %#v, %v", recovered, err)
		}
		if readBusinessStatus(t, db) != "closed" || countStoreStatusAudits(t, db) != 1 {
			t.Fatal("audit recovery did not commit exactly once")
		}
	})
}

func TestApplyAuditInsertFailureRollsBackAndSameKeyRecovers(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 11, 7, 0, 456789000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-audit-insert-recovery", now)
		insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		core := New(db, merchantidentity.NewRepository(db), func() time.Time { return now })
		command := Command{
			UserID: userID, DesiredStatus: storefront.BusinessClosed,
			IdempotencyKey: "audit-insert-recovery-command", RequestID: "audit-insert-recovery-request",
		}
		if _, err := db.ExecContext(context.Background(), `
			CREATE TRIGGER reject_store_status_audit
			BEFORE INSERT ON merchant_action_audits FOR EACH ROW
			SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='controlled store status audit insert failure'
		`); err != nil {
			t.Fatal("install controlled audit insert failure failed")
		}
		triggerDropped := false
		defer func() {
			if !triggerDropped {
				_, _ = db.ExecContext(context.Background(), "DROP TRIGGER reject_store_status_audit")
			}
		}()

		result, err := core.Apply(context.Background(), command)
		if !errors.Is(err, ErrUnavailable) || result != (Result{}) {
			t.Fatalf("audit-insert-failure Apply() = %#v, %v", result, err)
		}
		if readBusinessStatus(t, db) != "open" || countStoreStatusAudits(t, db) != 0 {
			t.Fatal("audit insert failure committed status or audit")
		}
		if _, err := db.ExecContext(context.Background(), "DROP TRIGGER reject_store_status_audit"); err != nil {
			t.Fatal("remove controlled audit insert failure failed")
		}
		triggerDropped = true

		recovered, err := core.Apply(context.Background(), command)
		want := Result{Before: storefront.BusinessOpen, After: storefront.BusinessClosed, Changed: true}
		if err != nil || recovered != want || readBusinessStatus(t, db) != "closed" || countStoreStatusAudits(t, db) != 1 {
			t.Fatalf("audit-insert-recovered Apply() = %#v, %v; want %#v", recovered, err, want)
		}
	})
}

func TestApplyMissingAndBadSingletonRecoverOnNextAttempt(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		withStoreStatusSchema(t, func(db *sql.DB) {
			now := time.Date(2026, time.August, 23, 11, 10, 0, 567890000, time.UTC)
			userID := insertStoreStatusUser(t, db, "opaque-store-status-missing-row", now)
			insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
			core := New(db, merchantidentity.NewRepository(db), func() time.Time { return now })
			command := Command{
				UserID: userID, DesiredStatus: storefront.BusinessClosed,
				IdempotencyKey: "missing-row-command", RequestID: "missing-row-request",
			}
			if result, err := core.Apply(context.Background(), command); !errors.Is(err, ErrUnavailable) || result != (Result{}) {
				t.Fatalf("missing-row Apply() = %#v, %v", result, err)
			}
			if countStoreStatusAudits(t, db) != 0 {
				t.Fatal("missing row produced an audit")
			}
			insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
			if result, err := core.Apply(context.Background(), command); err != nil || result.After != storefront.BusinessClosed || !result.Changed {
				t.Fatalf("missing-row recovery Apply() = %#v, %v", result, err)
			}
		})
	})

	t.Run("bad status", func(t *testing.T) {
		withStoreStatusSchema(t, func(db *sql.DB) {
			now := time.Date(2026, time.August, 23, 11, 15, 0, 678901000, time.UTC)
			userID := insertStoreStatusUser(t, db, "opaque-store-status-bad-row", now)
			insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
			insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
			if _, err := db.ExecContext(context.Background(), "ALTER TABLE storefront_settings MODIFY business_status VARCHAR(32) NOT NULL"); err != nil {
				t.Fatal("prepare bad status drift failed")
			}
			if _, err := db.ExecContext(context.Background(), "UPDATE storefront_settings SET business_status='broken' WHERE id=1"); err != nil {
				t.Fatal("persist bad status drift failed")
			}
			core := New(db, merchantidentity.NewRepository(db), func() time.Time { return now })
			command := Command{
				UserID: userID, DesiredStatus: storefront.BusinessClosed,
				IdempotencyKey: "bad-row-command", RequestID: "bad-row-request",
			}
			if result, err := core.Apply(context.Background(), command); !errors.Is(err, ErrUnavailable) || result != (Result{}) {
				t.Fatalf("bad-row Apply() = %#v, %v", result, err)
			}
			if countStoreStatusAudits(t, db) != 0 {
				t.Fatal("bad row produced an audit")
			}
			if _, err := db.ExecContext(context.Background(), "UPDATE storefront_settings SET business_status='open' WHERE id=1"); err != nil {
				t.Fatal("restore valid status failed")
			}
			if result, err := core.Apply(context.Background(), command); err != nil || result.After != storefront.BusinessClosed || !result.Changed {
				t.Fatalf("bad-row recovery Apply() = %#v, %v", result, err)
			}
		})
	})
}

func TestApplyRejectsZeroClockBeforeTransaction(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 11, 20, 0, 789012000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-zero-clock", now)
		insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		result, err := New(db, merchantidentity.NewRepository(db), func() time.Time { return time.Time{} }).Apply(context.Background(), Command{
			UserID: userID, DesiredStatus: storefront.BusinessClosed,
			IdempotencyKey: "zero-clock-command", RequestID: "zero-clock-request",
		})
		if !errors.Is(err, ErrUnavailable) || result != (Result{}) || readBusinessStatus(t, db) != "open" || countStoreStatusAudits(t, db) != 0 {
			t.Fatalf("zero-clock Apply() = %#v, %v", result, err)
		}
	})
}

func TestApplyCommitFailureRollsBackAndSameKeyRecovers(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 11, 25, 0, 890123000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-commit-recovery", now)
		insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		command := Command{
			UserID: userID, DesiredStatus: storefront.BusinessClosed,
			IdempotencyKey: "commit-recovery-command", RequestID: "commit-recovery-request",
		}
		core := newCore(
			db,
			merchantidentity.NewRepository(db),
			func() time.Time { return now },
			func(transaction *sql.Tx) error {
				if err := transaction.Rollback(); err != nil {
					return err
				}
				return errors.New("controlled commit failure")
			},
		)

		result, err := core.Apply(context.Background(), command)
		if !errors.Is(err, ErrUnavailable) || result != (Result{}) {
			t.Fatalf("commit-failure Apply() = %#v, %v", result, err)
		}
		if readBusinessStatus(t, db) != "open" || countStoreStatusAudits(t, db) != 0 {
			t.Fatal("commit failure retained partial state or audit")
		}
		recovered, err := New(db, merchantidentity.NewRepository(db), func() time.Time { return now }).Apply(context.Background(), command)
		if err != nil || recovered != (Result{Before: storefront.BusinessOpen, After: storefront.BusinessClosed, Changed: true}) {
			t.Fatalf("commit recovery Apply() = %#v, %v", recovered, err)
		}
		if readBusinessStatus(t, db) != "closed" || countStoreStatusAudits(t, db) != 1 {
			t.Fatal("commit recovery did not commit exactly once")
		}
	})
}

func TestApplyRetriesOneRealDeadlockWithFreshAuthorization(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 11, 30, 0, 901234000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-deadlock", now)
		accountID := insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))

		managementTx, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			t.Fatal("begin deadlock management transaction failed")
		}
		defer managementTx.Rollback()
		var singletonID uint8
		if err := managementTx.QueryRowContext(context.Background(), "SELECT id FROM storefront_settings WHERE id=1 FOR UPDATE").Scan(&singletonID); err != nil || singletonID != 1 {
			t.Fatal("lock deadlock storefront fixture failed")
		}
		for index := range 20 {
			if _, err := managementTx.ExecContext(context.Background(), `
				INSERT INTO merchant_action_audits(actor_user_id,action,result,reason,request_id,occurred_at)
				VALUES (?,'deadlock.fixture','SUCCEEDED','FIXTURE',?,?)
			`, userID, []byte("deadlock-fixture-"+string(rune('a'+index))), now); err != nil {
				t.Fatal("weight deadlock management transaction failed")
			}
		}

		type applyResult struct {
			result Result
			err    error
		}
		applyDone := make(chan applyResult, 1)
		go func() {
			result, err := New(db, merchantidentity.NewRepository(db), func() time.Time { return now }).Apply(context.Background(), Command{
				UserID: userID, DesiredStatus: storefront.BusinessClosed,
				IdempotencyKey: "deadlock-command", RequestID: "deadlock-request",
			})
			applyDone <- applyResult{result: result, err: err}
		}()
		waitForStoreStatusLock(t, db, "storefront_settings")
		if _, err := managementTx.ExecContext(context.Background(), `
			UPDATE merchant_accounts
			SET role='SUBACCOUNT',record_version=record_version+1,auth_version=auth_version+1,updated_at=?
			WHERE id=?
		`, now.Add(time.Minute), accountID); err != nil {
			t.Fatalf("management transaction was selected as deadlock victim: %v", err)
		}
		if err := managementTx.Commit(); err != nil {
			t.Fatal("commit deadlock management transaction failed")
		}

		select {
		case got := <-applyDone:
			want := Result{Before: storefront.BusinessOpen, After: storefront.BusinessClosed, Changed: true}
			if got.err != nil || got.result != want {
				t.Fatalf("deadlock-retried Apply() = %#v, %v; want %#v", got.result, got.err, want)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("deadlock-retried Apply did not complete")
		}
		var role merchantidentity.Role
		var authVersion uint64
		if err := db.QueryRowContext(context.Background(), `
			SELECT role_snapshot,auth_version_snapshot FROM merchant_action_audits
			WHERE actor_user_id=? AND action=?
		`, userID, merchantidentity.ActionStoreStatusWrite).Scan(&role, &authVersion); err != nil {
			t.Fatal("read deadlock retry audit failed")
		}
		if role != merchantidentity.RoleSubaccount || authVersion != 2 || readBusinessStatus(t, db) != "closed" || countStoreStatusAudits(t, db) != 1 {
			t.Fatalf("deadlock retry facts = role %s auth %d", role, authVersion)
		}
	})
}

func TestApplyRetriesAuthorizationLockTimeoutWithFreshRole(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 11, 33, 0, 123456000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-authorization-timeout", now)
		accountID := insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		var schemaName string
		if err := db.QueryRowContext(context.Background(), "SELECT DATABASE()").Scan(&schemaName); err != nil {
			t.Fatal("read authorization-timeout schema failed")
		}
		config, ok := storeStatusIntegrationConfig(t, schemaName)
		if !ok {
			t.Fatal("authorization-timeout environment disappeared")
		}
		applyDB, err := database.Open(config)
		if err != nil {
			t.Fatal("open authorization-timeout connection failed")
		}
		defer applyDB.Close()
		applyDB.SetMaxOpenConns(1)
		applyDB.SetMaxIdleConns(1)
		applyDB.SetConnMaxLifetime(0)
		applyDB.SetConnMaxIdleTime(0)
		if _, err := applyDB.ExecContext(context.Background(), "SET SESSION innodb_lock_wait_timeout=1"); err != nil {
			t.Fatal("shorten authorization lock timeout failed")
		}
		var lockWaitTimeout int
		if err := applyDB.QueryRowContext(context.Background(), "SELECT @@SESSION.innodb_lock_wait_timeout").Scan(&lockWaitTimeout); err != nil || lockWaitTimeout != 1 {
			t.Fatalf("authorization lock timeout = %d, %v", lockWaitTimeout, err)
		}

		managementTx, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			t.Fatal("begin authorization-timeout management transaction failed")
		}
		defer managementTx.Rollback()
		if _, err := managementTx.ExecContext(context.Background(), `
			UPDATE merchant_accounts
			SET role='SUBACCOUNT',record_version=record_version+1,auth_version=auth_version+1,updated_at=?
			WHERE id=?
		`, now.Add(time.Minute), accountID); err != nil {
			t.Fatal("lock and update authorization-timeout account failed")
		}
		authorizer := &outcomeAuthorizer{
			delegate:     merchantidentity.NewRepository(applyDB),
			outcomes:     make(chan error, 2),
			releaseFirst: make(chan struct{}),
		}
		firstReleased := false
		defer func() {
			if !firstReleased {
				close(authorizer.releaseFirst)
			}
		}()
		type applyResult struct {
			result Result
			err    error
		}
		applyDone := make(chan applyResult, 1)
		go func() {
			result, err := New(applyDB, authorizer, func() time.Time { return now }).Apply(context.Background(), Command{
				UserID: userID, DesiredStatus: storefront.BusinessClosed,
				IdempotencyKey: "authorization-timeout-command", RequestID: "authorization-timeout-request",
			})
			applyDone <- applyResult{result: result, err: err}
		}()
		waitForStoreStatusLock(t, db, "merchant_accounts")
		select {
		case firstErr := <-authorizer.outcomes:
			if !errors.Is(firstErr, merchantidentity.ErrUnavailable) {
				t.Fatalf("first authorization outcome = %v, want folded unavailable", firstErr)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("authorization lock wait did not time out")
		}
		if readBusinessStatus(t, db) != "open" || countStoreStatusAudits(t, db) != 0 {
			t.Fatal("timed-out authorization attempt wrote state or audit")
		}
		if err := managementTx.Commit(); err != nil {
			t.Fatal("release authorization-timeout account lock failed")
		}
		close(authorizer.releaseFirst)
		firstReleased = true

		select {
		case got := <-applyDone:
			want := Result{Before: storefront.BusinessOpen, After: storefront.BusinessClosed, Changed: true}
			if got.err != nil || got.result != want {
				t.Fatalf("authorization-timeout Apply() = %#v, %v; want %#v", got.result, got.err, want)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("authorization-timeout retry did not complete")
		}
		if authorizer.calls != 2 {
			t.Fatalf("authorization calls = %d, want exactly 2", authorizer.calls)
		}
		select {
		case secondErr := <-authorizer.outcomes:
			if secondErr != nil {
				t.Fatalf("second authorization outcome = %v", secondErr)
			}
		default:
			t.Fatal("second authorization attempt was not observed")
		}
		var role merchantidentity.Role
		var authVersion uint64
		if err := db.QueryRowContext(context.Background(), `
			SELECT role_snapshot,auth_version_snapshot FROM merchant_action_audits
			WHERE actor_user_id=? AND action=?
		`, userID, merchantidentity.ActionStoreStatusWrite).Scan(&role, &authVersion); err != nil {
			t.Fatal("read authorization-timeout audit failed")
		}
		if role != merchantidentity.RoleSubaccount || authVersion != 2 || readBusinessStatus(t, db) != "closed" || countStoreStatusAudits(t, db) != 1 {
			t.Fatalf("authorization-timeout facts = role %s auth %d", role, authVersion)
		}
	})
}

type outcomeAuthorizer struct {
	delegate     merchantidentity.Authorizer
	outcomes     chan error
	releaseFirst chan struct{}
	calls        int
}

func (authorizer *outcomeAuthorizer) AuthorizeInTx(
	ctx context.Context,
	transaction *sql.Tx,
	userID uint64,
	action merchantidentity.Action,
	target merchantidentity.Target,
) (merchantidentity.Authorization, error) {
	authorization, err := authorizer.delegate.AuthorizeInTx(ctx, transaction, userID, action, target)
	authorizer.calls++
	authorizer.outcomes <- err
	if authorizer.calls == 1 {
		select {
		case <-authorizer.releaseFirst:
		case <-ctx.Done():
			return merchantidentity.Authorization{}, ctx.Err()
		}
	}
	return authorization, err
}

func TestApplyPermanentAuthorizationUnavailableRetriesOnceThenFailsClosed(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 11, 34, 0, 123456000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-permanent-auth-unavailable", now)
		insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		authorizer := &fixedErrorAuthorizer{err: merchantidentity.ErrUnavailable}

		result, err := New(db, authorizer, func() time.Time { return now }).Apply(context.Background(), Command{
			UserID: userID, DesiredStatus: storefront.BusinessClosed,
			IdempotencyKey: "permanent-auth-unavailable-command", RequestID: "permanent-auth-unavailable-request",
		})

		if !errors.Is(err, merchantidentity.ErrUnavailable) || result != (Result{}) {
			t.Fatalf("permanent authorization unavailable Apply() = %#v, %v", result, err)
		}
		if authorizer.calls != 2 {
			t.Fatalf("permanent authorization unavailable calls = %d, want 2", authorizer.calls)
		}
		if readBusinessStatus(t, db) != "open" || countStoreStatusAudits(t, db) != 0 {
			t.Fatal("permanent authorization unavailable wrote state or audit")
		}
	})
}

func TestApplyForbiddenAuthorizationDoesNotRetry(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 11, 34, 30, 123456000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-forbidden-auth", now)
		insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		authorizer := &fixedErrorAuthorizer{err: merchantidentity.ErrForbidden}

		result, err := New(db, authorizer, func() time.Time { return now }).Apply(context.Background(), Command{
			UserID: userID, DesiredStatus: storefront.BusinessClosed,
			IdempotencyKey: "forbidden-auth-command", RequestID: "forbidden-auth-request",
		})

		if !errors.Is(err, merchantidentity.ErrForbidden) || result != (Result{}) {
			t.Fatalf("forbidden authorization Apply() = %#v, %v", result, err)
		}
		if authorizer.calls != 1 {
			t.Fatalf("forbidden authorization calls = %d, want 1", authorizer.calls)
		}
		if readBusinessStatus(t, db) != "open" || countStoreStatusAudits(t, db) != 0 {
			t.Fatal("forbidden authorization wrote state or audit")
		}
	})
}

type fixedErrorAuthorizer struct {
	err   error
	calls int
}

func (authorizer *fixedErrorAuthorizer) AuthorizeInTx(
	context.Context,
	*sql.Tx,
	uint64,
	merchantidentity.Action,
	merchantidentity.Target,
) (merchantidentity.Authorization, error) {
	authorizer.calls++
	return merchantidentity.Authorization{}, authorizer.err
}

func TestApplyDatabaseFailureReturnsUnavailableAndNextConnectionRecovers(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 11, 35, 0, 123456000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-database-recovery", now)
		insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		var schemaName string
		if err := db.QueryRowContext(context.Background(), "SELECT DATABASE()").Scan(&schemaName); err != nil {
			t.Fatal("read database recovery schema failed")
		}
		config, ok := storeStatusIntegrationConfig(t, schemaName)
		if !ok {
			t.Fatal("database recovery environment disappeared")
		}
		core := New(db, merchantidentity.NewRepository(db), func() time.Time { return now })
		command := Command{
			UserID: userID, DesiredStatus: storefront.BusinessClosed,
			IdempotencyKey: "database-recovery-command", RequestID: "database-recovery-request",
		}
		if err := db.Close(); err != nil {
			t.Fatal("close database recovery connection failed")
		}

		result, err := core.Apply(context.Background(), command)
		if !errors.Is(err, ErrUnavailable) || result != (Result{}) {
			t.Fatalf("closed-database Apply() = %#v, %v", result, err)
		}

		recoveredDB, err := database.Open(config)
		if err != nil {
			t.Fatal("restore database recovery connection failed")
		}
		defer recoveredDB.Close()
		recovered, err := New(recoveredDB, merchantidentity.NewRepository(recoveredDB), func() time.Time { return now }).Apply(context.Background(), command)
		want := Result{Before: storefront.BusinessOpen, After: storefront.BusinessClosed, Changed: true}
		if err != nil || recovered != want || readBusinessStatus(t, recoveredDB) != "closed" || countStoreStatusAudits(t, recoveredDB) != 1 {
			t.Fatalf("database-recovered Apply() = %#v, %v; want %#v", recovered, err, want)
		}
	})
}
