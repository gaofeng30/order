package merchantidentity

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/internal/wechat"
	"github.com/gaofeng30/order/services/api/migrations"
	"github.com/gin-gonic/gin"
)

var merchantSchemaPattern = regexp.MustCompile(`^order_merchant_identity_test_[0-9a-f]{32}$`)

func TestMerchantIdentityMySQL8Integration(t *testing.T) {
	withMerchantSchema(t, func(db *sql.DB) {
		migrationSet, err := migrate.Load(migrations.FS)
		if err != nil || len(migrationSet) != 13 {
			t.Fatal("load exact v1-v13 migration set failed")
		}
		foundation, err := migrate.Run(context.Background(), db, migrationSet[:1])
		if err != nil || foundation.ToVersion != 1 || foundation.AppliedCount != 1 {
			t.Fatal("establish v1 foundation failed")
		}
		advanced, err := migrate.Run(context.Background(), db, migrationSet)
		if err != nil || advanced.FromVersion != 1 || advanced.ToVersion != 13 || advanced.AppliedCount != 12 {
			t.Fatalf("advance v1 to v13 failed at v%d: %s", migrate.Version(err), migrate.Reason(err))
		}
		repeat, err := migrate.Run(context.Background(), db, migrationSet)
		if err != nil || repeat.FromVersion != 13 || repeat.ToVersion != 13 || repeat.AppliedCount != 0 {
			t.Fatal("repeat v13 migration was not a zero-write success")
		}

		t.Run("first binding and durable audit are atomic", func(t *testing.T) {
			assertFirstMerchantBinding(t, db)
		})
		t.Run("unresolved account rejection is durable without phone writes", func(t *testing.T) {
			assertUnresolvedMerchantRejection(t, db)
		})
		t.Run("schema constraints and hard-delete audit retention", func(t *testing.T) {
			assertMerchantSchemaConstraints(t, db)
		})
		t.Run("resolved business rejections are audited without partial writes", func(t *testing.T) {
			assertResolvedMerchantRejections(t, db)
		})
		t.Run("binding survives edits while enabled role and deletion apply live", func(t *testing.T) {
			assertLiveMerchantAuthorization(t, db)
		})
		t.Run("same code concurrency converges only after committed binding proof", func(t *testing.T) {
			assertConcurrentSameCodeMerchantLogin(t, db)
		})
		t.Run("rollback audit failure and commit unknown preserve recoverable facts", func(t *testing.T) {
			assertMerchantTransactionRecovery(t, db)
		})
		t.Run("real deadlock retries the database transaction without provider retry", func(t *testing.T) {
			assertMerchantDeadlockRecovery(t, db)
		})
		t.Run("authorization and account changes have one commit order", func(t *testing.T) {
			assertAuthorizationCommitOrdering(t, db)
		})
		t.Run("provider rejection is audited once without concurrent proof", func(t *testing.T) {
			assertRejectedPhoneCodeAudit(t, db)
		})
		t.Run("provider rejection after concurrent version drift has unresolved audit", func(t *testing.T) {
			assertRejectedPhoneCodeAfterConcurrentAccountChange(t, db, false)
		})
		t.Run("provider rejection after concurrent disable has unresolved audit", func(t *testing.T) {
			assertRejectedPhoneCodeAfterConcurrentAccountChange(t, db, true)
		})
		t.Run("real session HTTP and durable audit exclude PII canaries", func(t *testing.T) {
			assertMerchantPIIBoundaries(t, db)
		})
	})
}

func assertFirstMerchantBinding(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, time.August, 23, 1, 2, 3, 456789000, time.UTC)
	result, err := db.ExecContext(ctx, `
		INSERT INTO miniprogram_users(openid,created_at,last_login_at)
		VALUES (?,?,?)
	`, "opaque-provider-subject-a", now.Add(-time.Hour), now.Add(-time.Minute))
	if err != nil {
		t.Fatal("insert first-binding user failed")
	}
	userID, err := result.LastInsertId()
	if err != nil || userID <= 0 {
		t.Fatal("resolve first-binding user failed")
	}
	accountResult, err := db.ExecContext(ctx, `
		INSERT INTO merchant_accounts(phone,name,role,created_at,updated_at)
		VALUES (?,?,?,?,?)
	`, "+7", "Synthetic Owner", RoleOwner, now.Add(-time.Hour), now.Add(-time.Hour))
	if err != nil {
		t.Fatal("insert first-binding merchant account failed")
	}
	accountID, err := accountResult.LastInsertId()
	if err != nil || accountID <= 0 {
		t.Fatal("resolve first-binding merchant account failed")
	}

	repository := NewRepository(db)
	service := newService(repository, staticPhoneProvider{phone: "+7"}, func() time.Time { return now })
	projection, err := service.Login(ctx, uint64(userID), "one-use-provider-code", "internal-request-a")
	if err != nil || projection.Merchant == nil || projection.Merchant.Role != RoleOwner || projection.Merchant.AuthVersion != 2 || !projection.PrimaryPhoneBound {
		t.Fatalf("first merchant login result invalid: %T", err)
	}

	var phone string
	var phoneBoundAt time.Time
	if err := db.QueryRowContext(ctx, "SELECT primary_phone,primary_phone_bound_at FROM miniprogram_users WHERE id=?", userID).Scan(&phone, &phoneBoundAt); err != nil {
		t.Fatal("read first-binding user result failed")
	}
	if phone != "+7" || !phoneBoundAt.Equal(now) {
		t.Fatal("first merchant login did not bind the canonical primary phone")
	}
	var boundUserID uint64
	var boundAt time.Time
	var recordVersion, authVersion uint64
	if err := db.QueryRowContext(ctx, "SELECT bound_user_id,bound_at,record_version,auth_version FROM merchant_accounts WHERE id=?", accountID).Scan(&boundUserID, &boundAt, &recordVersion, &authVersion); err != nil {
		t.Fatal("read first-binding account result failed")
	}
	if boundUserID != uint64(userID) || !boundAt.Equal(now) || recordVersion != 2 || authVersion != 2 {
		t.Fatal("first merchant login did not atomically bind and version the account")
	}
	var auditAccountID, snapshotID, auditActorID, snapshotAuth uint64
	var snapshotRole Role
	var action, auditResult, reason string
	if err := db.QueryRowContext(ctx, `
		SELECT merchant_account_id,account_id_snapshot,role_snapshot,auth_version_snapshot,
		       actor_user_id,action,result,reason
		FROM merchant_action_audits
		WHERE actor_user_id=?
	`, userID).Scan(&auditAccountID, &snapshotID, &snapshotRole, &snapshotAuth, &auditActorID, &action, &auditResult, &reason); err != nil {
		t.Fatal("read first-binding audit failed")
	}
	if auditAccountID != uint64(accountID) || snapshotID != uint64(accountID) || snapshotRole != RoleOwner || snapshotAuth != 2 || auditActorID != uint64(userID) || action != "merchant.login" || auditResult != "SUCCEEDED" || reason != "FIRST_BINDING" {
		t.Fatal("first-binding audit snapshot was incomplete")
	}
}

func assertUnresolvedMerchantRejection(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, time.August, 23, 2, 0, 0, 123456000, time.UTC)
	userID := insertMerchantTestUser(t, db, "opaque-provider-subject-b", now)
	service := newService(NewRepository(db), staticPhoneProvider{phone: "+8"}, func() time.Time { return now })

	if _, err := service.Login(ctx, userID, "one-use-provider-code", "internal-request-b"); !errors.Is(err, ErrMerchantAccountNotAvailable) {
		t.Fatalf("unresolved merchant login error = %v", err)
	}
	var phone sql.NullString
	var boundAt sql.NullTime
	if err := db.QueryRowContext(ctx, "SELECT primary_phone,primary_phone_bound_at FROM miniprogram_users WHERE id=?", userID).Scan(&phone, &boundAt); err != nil {
		t.Fatal("read unresolved rejection user failed")
	}
	if phone.Valid || boundAt.Valid {
		t.Fatal("unresolved merchant rejection wrote primary phone state")
	}
	var merchantAccountID, snapshotID, snapshotRole, snapshotAuth sql.NullString
	var result, reason string
	if err := db.QueryRowContext(ctx, `
		SELECT merchant_account_id,account_id_snapshot,role_snapshot,auth_version_snapshot,result,reason
		FROM merchant_action_audits
		WHERE actor_user_id=? AND request_id=?
	`, userID, []byte("internal-request-b")).Scan(&merchantAccountID, &snapshotID, &snapshotRole, &snapshotAuth, &result, &reason); err != nil {
		t.Fatal("read unresolved rejection audit failed")
	}
	if merchantAccountID.Valid || snapshotID.Valid || snapshotRole.Valid || snapshotAuth.Valid || result != "REJECTED" || reason != "ACCOUNT_NOT_AVAILABLE" {
		t.Fatal("unresolved rejection audit did not preserve the all-empty snapshot contract")
	}
}

func insertMerchantTestUser(t *testing.T, db *sql.DB, providerSubject string, at time.Time) uint64 {
	t.Helper()
	result, err := db.ExecContext(context.Background(), `
		INSERT INTO miniprogram_users(openid,created_at,last_login_at)
		VALUES (?,?,?)
	`, providerSubject, at.Add(-time.Hour), at.Add(-time.Minute))
	if err != nil {
		t.Fatal("insert merchant test user failed")
	}
	userID, err := result.LastInsertId()
	if err != nil || userID <= 0 {
		t.Fatal("resolve merchant test user failed")
	}
	return uint64(userID)
}

func assertMerchantSchemaConstraints(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, time.August, 23, 2, 30, 0, 654321000, time.UTC)
	userA := insertMerchantTestUser(t, db, "opaque-provider-subject-c", now)
	userB := insertMerchantTestUser(t, db, "opaque-provider-subject-d", now)
	accountID := insertMerchantTestAccount(t, db, "+9", RoleSubaccount, true, nil, now)

	invalidStatements := []struct {
		name string
		sql  string
		args []any
	}{
		{name: "noncanonical phone", sql: "INSERT INTO merchant_accounts(phone,name,role,created_at,updated_at) VALUES (?,?,?,?,?)", args: []any{"+0", "Synthetic", RoleOwner, now, now}},
		{name: "empty name", sql: "INSERT INTO merchant_accounts(phone,name,role,created_at,updated_at) VALUES (?,?,?,?,?)", args: []any{"+11", "", RoleOwner, now, now}},
		{name: "invalid role", sql: "INSERT INTO merchant_accounts(phone,name,role,created_at,updated_at) VALUES (?,?,?,?,?)", args: []any{"+12", "Synthetic", "ADMIN", now, now}},
		{name: "invalid enabled", sql: "INSERT INTO merchant_accounts(phone,name,role,enabled,created_at,updated_at) VALUES (?,?,?,?,?,?)", args: []any{"+18", "Synthetic", RoleOwner, 2, now, now}},
		{name: "zero versions", sql: "INSERT INTO merchant_accounts(phone,name,role,record_version,auth_version,created_at,updated_at) VALUES (?,?,?,?,?,?,?)", args: []any{"+13", "Synthetic", RoleOwner, 0, 1, now, now}},
		{name: "partial binding", sql: "INSERT INTO merchant_accounts(phone,name,role,bound_user_id,created_at,updated_at) VALUES (?,?,?,?,?,?)", args: []any{"+14", "Synthetic", RoleOwner, userA, now, now}},
		{name: "missing user fk", sql: "INSERT INTO merchant_accounts(phone,name,role,bound_user_id,bound_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?)", args: []any{"+15", "Synthetic", RoleOwner, uint64(1 << 62), now, now, now}},
		{name: "duplicate phone", sql: "INSERT INTO merchant_accounts(phone,name,role,created_at,updated_at) VALUES (?,?,?,?,?)", args: []any{"+9", "Synthetic", RoleOwner, now, now}},
	}
	for _, test := range invalidStatements {
		if _, err := db.ExecContext(ctx, test.sql, test.args...); err == nil {
			t.Fatalf("schema accepted %s", test.name)
		}
	}

	boundAt := now.Add(time.Minute)
	firstBoundID := insertMerchantTestAccount(t, db, "+16", RoleOwner, true, &merchantBindingFixture{UserID: userA, BoundAt: boundAt}, now)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO merchant_accounts(phone,name,role,bound_user_id,bound_at,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?)
	`, "+17", "Synthetic", RoleSubaccount, userA, boundAt, now, now); err == nil {
		t.Fatal("schema accepted duplicate bound_user_id")
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM miniprogram_users WHERE id=?", userA); err == nil {
		t.Fatal("bound-user foreign key allowed user deletion")
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO merchant_action_audits(
			merchant_account_id,account_id_snapshot,role_snapshot,auth_version_snapshot,
			actor_user_id,action,result,reason,target_type,target_id,request_id,occurred_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
	`, accountID, accountID, RoleSubaccount, 1, userB, ActionOrderRead, "SUCCEEDED", "AUTHORIZED", "order", uint64(91), []byte("internal-request-c"), now); err != nil {
		t.Fatal("insert resolved retention audit failed")
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO merchant_action_audits(account_id_snapshot,actor_user_id,action,result,reason,request_id,occurred_at)
		VALUES (?,?,?,?,?,?,?)
	`, accountID, userB, ActionOrderRead, "REJECTED", "INVALID", []byte("internal-request-d"), now); err == nil {
		t.Fatal("audit schema accepted a partial account snapshot")
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM merchant_accounts WHERE id=?", accountID); err != nil {
		t.Fatal("hard delete of audited account failed")
	}
	var liveAccount sql.NullInt64
	var snapshotID, snapshotAuth uint64
	var snapshotRole Role
	var action, result, reason, targetType string
	var targetID, actorID uint64
	var occurredAt time.Time
	if err := db.QueryRowContext(ctx, `
		SELECT merchant_account_id,account_id_snapshot,role_snapshot,auth_version_snapshot,
		       actor_user_id,action,result,reason,target_type,target_id,occurred_at
		FROM merchant_action_audits WHERE request_id=?
	`, []byte("internal-request-c")).Scan(&liveAccount, &snapshotID, &snapshotRole, &snapshotAuth, &actorID, &action, &result, &reason, &targetType, &targetID, &occurredAt); err != nil {
		t.Fatal("read retained audit after hard delete failed")
	}
	if liveAccount.Valid || snapshotID != accountID || snapshotRole != RoleSubaccount || snapshotAuth != 1 || actorID != userB || action != string(ActionOrderRead) || result != "SUCCEEDED" || reason != "AUTHORIZED" || targetType != "order" || targetID != 91 || !occurredAt.Equal(now) {
		t.Fatal("hard delete did not retain the complete non-PII audit snapshot")
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM merchant_accounts WHERE id=?", firstBoundID); err != nil {
		t.Fatal("cleanup bound schema fixture failed")
	}
}

type merchantBindingFixture struct {
	UserID  uint64
	BoundAt time.Time
}

func insertMerchantTestAccount(t *testing.T, db *sql.DB, phone string, role Role, enabled bool, binding *merchantBindingFixture, at time.Time) uint64 {
	t.Helper()
	var boundUserID, boundAt any
	if binding != nil {
		boundUserID = binding.UserID
		boundAt = binding.BoundAt
	}
	result, err := db.ExecContext(context.Background(), `
		INSERT INTO merchant_accounts(phone,name,role,enabled,bound_user_id,bound_at,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?)
	`, phone, "Synthetic Merchant", role, enabled, boundUserID, boundAt, at.Add(-time.Hour), at.Add(-time.Hour))
	if err != nil {
		t.Fatal("insert merchant test account failed")
	}
	accountID, err := result.LastInsertId()
	if err != nil || accountID <= 0 {
		t.Fatal("resolve merchant test account failed")
	}
	return uint64(accountID)
}

func assertResolvedMerchantRejections(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, time.August, 23, 3, 0, 0, 100000000, time.UTC)

	type rejectionCase struct {
		name       string
		phone      string
		wantErr    error
		wantReason string
		setup      func(*testing.T, *sql.DB, uint64, time.Time) uint64
	}
	tests := []rejectionCase{
		{
			name: "disabled account", phone: "+21", wantErr: ErrMerchantAccountNotAvailable, wantReason: "ACCOUNT_NOT_AVAILABLE",
			setup: func(t *testing.T, db *sql.DB, _ uint64, at time.Time) uint64 {
				return insertMerchantTestAccount(t, db, "+21", RoleOwner, false, nil, at)
			},
		},
		{
			name: "account bound to another user", phone: "+22", wantErr: ErrMerchantAccountNotAvailable, wantReason: "ACCOUNT_NOT_AVAILABLE",
			setup: func(t *testing.T, db *sql.DB, _ uint64, at time.Time) uint64 {
				otherUser := insertMerchantTestUser(t, db, "opaque-provider-subject-rejection-other", at)
				bindPrimaryPhoneFixture(t, db, otherUser, "+22", at)
				return insertMerchantTestAccount(t, db, "+22", RoleSubaccount, true, &merchantBindingFixture{UserID: otherUser, BoundAt: at}, at)
			},
		},
		{
			name: "primary phone mismatch", phone: "+24", wantErr: ErrPrimaryPhoneMismatch, wantReason: "PRIMARY_PHONE_MISMATCH",
			setup: func(t *testing.T, db *sql.DB, caller uint64, at time.Time) uint64 {
				bindPrimaryPhoneFixture(t, db, caller, "+23", at)
				return insertMerchantTestAccount(t, db, "+24", RoleOwner, true, nil, at)
			},
		},
		{
			name: "primary phone owned by another user", phone: "+25", wantErr: ErrPhoneInUse, wantReason: "PHONE_IN_USE",
			setup: func(t *testing.T, db *sql.DB, _ uint64, at time.Time) uint64 {
				phoneOwner := insertMerchantTestUser(t, db, "opaque-provider-subject-phone-owner", at)
				bindPrimaryPhoneFixture(t, db, phoneOwner, "+25", at)
				return insertMerchantTestAccount(t, db, "+25", RoleOwner, true, nil, at)
			},
		},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			caller := insertMerchantTestUser(t, db, "opaque-provider-subject-rejection-"+strconv.Itoa(index), now)
			accountID := test.setup(t, db, caller, now)
			provider := &countingPhoneProvider{phone: test.phone}
			requestID := "internal-rejection-" + strconv.Itoa(index)
			service := newService(NewRepository(db), provider, func() time.Time { return now })
			if _, err := service.Login(ctx, caller, "one-use-provider-code", requestID); !errors.Is(err, test.wantErr) {
				t.Fatalf("merchant rejection error = %v", err)
			}
			if provider.calls != 1 {
				t.Fatalf("provider calls = %d", provider.calls)
			}
			var boundUserID sql.NullInt64
			var boundAt sql.NullTime
			if err := db.QueryRowContext(ctx, "SELECT bound_user_id,bound_at FROM merchant_accounts WHERE id=?", accountID).Scan(&boundUserID, &boundAt); err != nil {
				t.Fatal("read rejected merchant account failed")
			}
			if test.name == "account bound to another user" {
				if !boundUserID.Valid || !boundAt.Valid || uint64(boundUserID.Int64) == caller {
					t.Fatal("other-user binding changed during rejection")
				}
			} else if boundUserID.Valid || boundAt.Valid {
				t.Fatal("business rejection wrote merchant binding")
			}
			var phone sql.NullString
			var phoneBoundAt sql.NullTime
			if err := db.QueryRowContext(ctx, "SELECT primary_phone,primary_phone_bound_at FROM miniprogram_users WHERE id=?", caller).Scan(&phone, &phoneBoundAt); err != nil {
				t.Fatal("read rejected caller phone failed")
			}
			if test.name == "primary phone mismatch" {
				if !phone.Valid || phone.String != "+23" || !phoneBoundAt.Valid {
					t.Fatal("primary mismatch changed the existing phone")
				}
			} else if phone.Valid || phoneBoundAt.Valid {
				t.Fatal("business rejection wrote caller primary phone")
			}
			var snapshotID, snapshotAuth uint64
			var snapshotRole Role
			var result, reason string
			if err := db.QueryRowContext(ctx, `
				SELECT account_id_snapshot,role_snapshot,auth_version_snapshot,result,reason
				FROM merchant_action_audits WHERE request_id=?
			`, []byte(requestID)).Scan(&snapshotID, &snapshotRole, &snapshotAuth, &result, &reason); err != nil {
				t.Fatal("read resolved rejection audit failed")
			}
			if snapshotID != accountID || !validRole(snapshotRole) || snapshotAuth == 0 || result != "REJECTED" || reason != test.wantReason {
				t.Fatal("resolved rejection audit snapshot was incomplete")
			}
		})
	}
}

type countingPhoneProvider struct {
	phone string
	err   error
	calls int
}

func (provider *countingPhoneProvider) Exchange(context.Context, string, string) (string, error) {
	provider.calls++
	return provider.phone, provider.err
}

func bindPrimaryPhoneFixture(t *testing.T, db *sql.DB, userID uint64, phone string, at time.Time) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), `
		UPDATE miniprogram_users SET primary_phone=?,primary_phone_bound_at=? WHERE id=?
	`, phone, at, userID); err != nil {
		t.Fatal("bind primary phone fixture failed")
	}
}

func assertLiveMerchantAuthorization(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, time.August, 23, 4, 0, 0, 222222000, time.UTC)
	userID := insertMerchantTestUser(t, db, "opaque-provider-subject-live", now)
	bindPrimaryPhoneFixture(t, db, userID, "+31", now)
	accountID := insertMerchantTestAccount(t, db, "+31", RoleOwner, true, &merchantBindingFixture{UserID: userID, BoundAt: now}, now)
	repository := NewRepository(db)
	provider := &countingPhoneProvider{err: errors.New("provider must be bypassed")}
	service := newService(repository, provider, func() time.Time { return now })

	projection, err := service.Login(ctx, userID, "unused-code", "internal-live-owner")
	if err != nil || projection.Merchant == nil || projection.Merchant.Role != RoleOwner || projection.Merchant.AuthVersion != 1 || provider.calls != 0 {
		t.Fatal("existing OWNER binding did not bypass provider")
	}
	for _, action := range []Action{ActionOrderRead, ActionOrderMarkReady, ActionOrderRedeem, ActionProductSoldOutWrite, ActionStoreStatusWrite, ActionMerchantAccountManage} {
		authorization, err := authorizeTestAction(t, repository, db, userID, action)
		if err != nil || authorization.MerchantAccountID != accountID || authorization.Actor != ActorMerchantOwner || authorization.RecordVersion != 1 || authorization.AuthVersion != 1 {
			t.Fatalf("OWNER authorization failed for %s", action)
		}
	}

	if _, err := db.ExecContext(ctx, `
		UPDATE merchant_accounts SET enabled=FALSE,record_version=record_version+1,auth_version=auth_version+1,updated_at=? WHERE id=?
	`, now.Add(time.Minute), accountID); err != nil {
		t.Fatal("disable live merchant fixture failed")
	}
	disabled, err := service.Identity(ctx, userID)
	if err != nil || disabled.Merchant != nil || !disabled.PrimaryPhoneBound {
		t.Fatal("disabled account did not disappear from identity immediately")
	}
	if _, err := authorizeTestAction(t, repository, db, userID, ActionOrderRead); !errors.Is(err, ErrMerchantAccountNotAvailable) {
		t.Fatalf("disabled authorization error = %v", err)
	}
	if _, err := service.Login(ctx, userID, "unused-code", "internal-live-disabled"); !errors.Is(err, ErrMerchantAccountNotAvailable) {
		t.Fatalf("disabled existing-binding login error = %v", err)
	}
	if provider.calls != 0 {
		t.Fatal("disabled existing binding reached the phone provider")
	}
	var disabledAuditAccountID uint64
	var disabledAuditResult, disabledAuditReason string
	if err := db.QueryRowContext(ctx, `
		SELECT account_id_snapshot,result,reason FROM merchant_action_audits WHERE request_id=?
	`, []byte("internal-live-disabled")).Scan(&disabledAuditAccountID, &disabledAuditResult, &disabledAuditReason); err != nil {
		t.Fatal("read disabled existing-binding audit failed")
	}
	if disabledAuditAccountID != accountID || disabledAuditResult != "REJECTED" || disabledAuditReason != "ACCOUNT_NOT_AVAILABLE" {
		t.Fatal("disabled existing binding was not durably audited")
	}

	if _, err := db.ExecContext(ctx, `
		UPDATE merchant_accounts SET enabled=TRUE,record_version=record_version+1,auth_version=auth_version+1,updated_at=? WHERE id=?
	`, now.Add(2*time.Minute), accountID); err != nil {
		t.Fatal("re-enable live merchant fixture failed")
	}
	restartedService := newService(repository, provider, func() time.Time { return now.Add(3 * time.Minute) })
	restored, err := restartedService.Identity(ctx, userID)
	if err != nil || restored.Merchant == nil || restored.Merchant.Role != RoleOwner || restored.Merchant.AuthVersion != 3 {
		t.Fatal("re-enabled account did not restore the persisted binding")
	}

	if _, err := db.ExecContext(ctx, `
		UPDATE merchant_accounts SET role='SUBACCOUNT',phone='+32',record_version=record_version+1,auth_version=auth_version+1,updated_at=? WHERE id=?
	`, now.Add(4*time.Minute), accountID); err != nil {
		t.Fatal("edit live merchant fixture failed")
	}
	edited, err := restartedService.Login(ctx, userID, "unused-code", "internal-live-subaccount")
	if err != nil || edited.Merchant == nil || edited.Merchant.Role != RoleSubaccount || edited.Merchant.AuthVersion != 4 || provider.calls != 0 {
		t.Fatal("role/phone edit did not preserve provider-free binding")
	}
	for _, action := range []Action{ActionOrderRead, ActionOrderMarkReady, ActionOrderRedeem, ActionProductSoldOutWrite, ActionStoreStatusWrite} {
		authorization, err := authorizeTestAction(t, repository, db, userID, action)
		if err != nil || authorization.Actor != ActorMerchantSubaccount || authorization.RecordVersion != 4 || authorization.AuthVersion != 4 {
			t.Fatalf("SUBACCOUNT authorization failed for %s", action)
		}
	}
	if _, err := authorizeTestAction(t, repository, db, userID, ActionMerchantAccountManage); !errors.Is(err, ErrForbidden) {
		t.Fatalf("SUBACCOUNT account.manage error = %v", err)
	}

	var primaryPhone string
	if err := db.QueryRowContext(ctx, "SELECT primary_phone FROM miniprogram_users WHERE id=?", userID).Scan(&primaryPhone); err != nil || primaryPhone != "+31" {
		t.Fatal("account phone edit changed or re-compared the existing primary phone")
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM merchant_accounts WHERE id=?", accountID); err != nil {
		t.Fatal("delete live merchant fixture failed")
	}
	deleted, err := restartedService.Identity(ctx, userID)
	if err != nil || deleted.Merchant != nil || !deleted.PrimaryPhoneBound {
		t.Fatal("deleted account did not invalidate merchant identity")
	}
	if _, err := authorizeTestAction(t, repository, db, userID, ActionOrderRead); !errors.Is(err, ErrMerchantAccountNotAvailable) {
		t.Fatalf("deleted authorization error = %v", err)
	}
}

func authorizeTestAction(t *testing.T, repository *Repository, db *sql.DB, userID uint64, action Action) (Authorization, error) {
	t.Helper()
	transaction, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal("begin authorization test transaction failed")
	}
	authorization, authorizeErr := repository.AuthorizeInTx(
		context.Background(), transaction, userID, action, Target{Type: "internal_resource", ID: 1},
	)
	if err := transaction.Rollback(); err != nil {
		t.Fatal("rollback authorization test transaction failed")
	}
	return authorization, authorizeErr
}

func assertConcurrentSameCodeMerchantLogin(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, time.August, 23, 5, 0, 0, 333333000, time.UTC)
	userID := insertMerchantTestUser(t, db, "opaque-provider-subject-concurrent", now)
	accountID := insertMerchantTestAccount(t, db, "+41", RoleOwner, true, nil, now)
	repository := NewRepository(db)
	committed := make(chan struct{})
	store := &signalingCompleteStore{Store: repository, committed: committed}
	provider := &concurrentSameCodeProvider{
		phone: "+41", secondAtProvider: make(chan struct{}), committed: committed,
	}
	service := newService(store, provider, func() time.Time { return now })

	type loginResult struct {
		identity Identity
		err      error
	}
	results := make(chan loginResult, 2)
	for index := 0; index < 2; index++ {
		requestID := "internal-concurrent-" + strconv.Itoa(index)
		go func() {
			identity, err := service.Login(ctx, userID, "shared-one-use-code", requestID)
			results <- loginResult{identity: identity, err: err}
		}()
	}
	for index := 0; index < 2; index++ {
		result := <-results
		if result.err != nil || result.identity.Merchant == nil || result.identity.Merchant.Role != RoleOwner || result.identity.Merchant.AuthVersion != 2 {
			t.Fatalf("concurrent merchant login result %d failed: %v", index, result.err)
		}
	}
	if provider.Calls() != 2 {
		t.Fatalf("same-code provider calls = %d", provider.Calls())
	}
	var boundUserID, recordVersion, authVersion uint64
	if err := db.QueryRowContext(ctx, "SELECT bound_user_id,record_version,auth_version FROM merchant_accounts WHERE id=?", accountID).Scan(&boundUserID, &recordVersion, &authVersion); err != nil {
		t.Fatal("read concurrent binding result failed")
	}
	if boundUserID != userID || recordVersion != 2 || authVersion != 2 {
		t.Fatal("same-code concurrency changed binding or versions more than once")
	}
	var successes, rejections int
	if err := db.QueryRowContext(ctx, `
		SELECT SUM(result='SUCCEEDED'),SUM(result='REJECTED')
		FROM merchant_action_audits WHERE actor_user_id=? AND request_id IN (?,?)
	`, userID, []byte("internal-concurrent-0"), []byte("internal-concurrent-1")).Scan(&successes, &rejections); err != nil {
		t.Fatal("read concurrent login audits failed")
	}
	if successes != 2 || rejections != 0 {
		t.Fatal("same-code recovery did not produce two durable success results")
	}
}

type signalingCompleteStore struct {
	Store
	committed chan struct{}
	once      sync.Once
}

func (store *signalingCompleteStore) CompleteLogin(ctx context.Context, userID uint64, phone, requestID string, at time.Time) (Identity, error) {
	identity, err := store.Store.CompleteLogin(ctx, userID, phone, requestID, at)
	if err == nil {
		store.once.Do(func() { close(store.committed) })
	}
	return identity, err
}

type concurrentSameCodeProvider struct {
	phone            string
	secondAtProvider chan struct{}
	committed        chan struct{}
	mu               sync.Mutex
	calls            int
}

func (provider *concurrentSameCodeProvider) Exchange(context.Context, string, string) (string, error) {
	provider.mu.Lock()
	provider.calls++
	call := provider.calls
	provider.mu.Unlock()
	if call == 1 {
		<-provider.secondAtProvider
		return provider.phone, nil
	}
	if call == 2 {
		close(provider.secondAtProvider)
		<-provider.committed
		return "", wechat.ErrPhoneCodeRejected
	}
	return "", errors.New("unexpected provider retry")
}

func (provider *concurrentSameCodeProvider) Calls() int {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	return provider.calls
}

func assertMerchantTransactionRecovery(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, time.August, 23, 6, 0, 0, 444444000, time.UTC)

	t.Run("explicit rollback", func(t *testing.T) {
		userID := insertMerchantTestUser(t, db, "opaque-provider-subject-rollback", now)
		accountID := insertMerchantTestAccount(t, db, "+51", RoleOwner, true, nil, now)
		repository := newRepository(db, func(transaction *sql.Tx) error {
			if err := transaction.Rollback(); err != nil {
				return err
			}
			return errors.New("controlled rollback")
		})
		if _, err := repository.CompleteLogin(ctx, userID, "+51", "internal-rollback", now); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("rollback result = %v", err)
		}
		assertNoMerchantBindingWrites(t, db, userID, accountID, "internal-rollback")
	})

	t.Run("audit unavailable", func(t *testing.T) {
		userID := insertMerchantTestUser(t, db, "opaque-provider-subject-audit-unavailable", now)
		accountID := insertMerchantTestAccount(t, db, "+52", RoleSubaccount, true, nil, now)
		if _, err := db.ExecContext(ctx, "RENAME TABLE merchant_action_audits TO merchant_action_audits_unavailable"); err != nil {
			t.Fatal("hide audit table failed")
		}
		restored := false
		defer func() {
			if !restored {
				_, _ = db.ExecContext(context.Background(), "RENAME TABLE merchant_action_audits_unavailable TO merchant_action_audits")
			}
		}()
		if _, err := NewRepository(db).CompleteLogin(ctx, userID, "+52", "internal-audit-unavailable", now); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("audit unavailable result = %v", err)
		}
		if _, err := db.ExecContext(ctx, "RENAME TABLE merchant_action_audits_unavailable TO merchant_action_audits"); err != nil {
			t.Fatal("restore audit table failed")
		}
		restored = true
		assertNoMerchantBindingWrites(t, db, userID, accountID, "internal-audit-unavailable")
	})

	t.Run("commit outcome unknown", func(t *testing.T) {
		userID := insertMerchantTestUser(t, db, "opaque-provider-subject-commit-unknown", now)
		accountID := insertMerchantTestAccount(t, db, "+53", RoleOwner, true, nil, now)
		repository := newRepository(db, func(transaction *sql.Tx) error {
			if err := transaction.Commit(); err != nil {
				return err
			}
			return errors.New("controlled unknown commit outcome")
		})
		if _, err := repository.CompleteLogin(ctx, userID, "+53", "internal-commit-unknown", now); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("commit-unknown result = %v", err)
		}
		var boundUserID, authVersion uint64
		if err := db.QueryRowContext(ctx, "SELECT bound_user_id,auth_version FROM merchant_accounts WHERE id=?", accountID).Scan(&boundUserID, &authVersion); err != nil || boundUserID != userID || authVersion != 2 {
			t.Fatal("committed fact was not durable after unknown outcome")
		}
		recovered, err := NewRepository(db).RecoverRejectedLogin(
			ctx, userID, "internal-commit-recovery", now, now.Add(time.Microsecond),
		)
		if err != nil || recovered.Merchant == nil || recovered.Merchant.Role != RoleOwner || recovered.Merchant.AuthVersion != 2 {
			t.Fatalf("commit-unknown recovery failed: %v", err)
		}
	})
}

func assertNoMerchantBindingWrites(t *testing.T, db *sql.DB, userID, accountID uint64, requestID string) {
	t.Helper()
	var phone sql.NullString
	var phoneBoundAt sql.NullTime
	if err := db.QueryRowContext(context.Background(), "SELECT primary_phone,primary_phone_bound_at FROM miniprogram_users WHERE id=?", userID).Scan(&phone, &phoneBoundAt); err != nil {
		t.Fatal("read rollback user state failed")
	}
	var boundUserID sql.NullInt64
	var boundAt sql.NullTime
	var recordVersion, authVersion uint64
	if err := db.QueryRowContext(context.Background(), "SELECT bound_user_id,bound_at,record_version,auth_version FROM merchant_accounts WHERE id=?", accountID).Scan(&boundUserID, &boundAt, &recordVersion, &authVersion); err != nil {
		t.Fatal("read rollback account state failed")
	}
	var auditCount int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM merchant_action_audits WHERE request_id=?", []byte(requestID)).Scan(&auditCount); err != nil {
		t.Fatal("read rollback audit state failed")
	}
	if phone.Valid || phoneBoundAt.Valid || boundUserID.Valid || boundAt.Valid || recordVersion != 1 || authVersion != 1 || auditCount != 0 {
		t.Fatal("failed transaction left a partial phone, binding, version or audit write")
	}
}

func assertMerchantDeadlockRecovery(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, time.August, 23, 7, 0, 0, 555555000, time.UTC)
	userID := insertMerchantTestUser(t, db, "opaque-provider-subject-deadlock", now)
	accountID := insertMerchantTestAccount(t, db, "+61", RoleOwner, true, nil, now)

	managementTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal("begin deadlock management transaction failed")
	}
	defer managementTx.Rollback()
	var lockedAccount uint64
	if err := managementTx.QueryRowContext(ctx, "SELECT id FROM merchant_accounts WHERE id=? FOR UPDATE", accountID).Scan(&lockedAccount); err != nil || lockedAccount != accountID {
		t.Fatal("lock deadlock account fixture failed")
	}
	for index := 0; index < 20; index++ {
		if _, err := managementTx.ExecContext(ctx, `
			INSERT INTO merchant_action_audits(actor_user_id,action,result,reason,request_id,occurred_at)
			VALUES (?,'deadlock.fixture','SUCCEEDED','FIXTURE',?,?)
		`, userID, []byte("internal-deadlock-fixture-"+strconv.Itoa(index)), now); err != nil {
			t.Fatal("weight deadlock management transaction failed")
		}
	}

	provider := &countingPhoneProvider{phone: "+61"}
	service := newService(NewRepository(db), provider, func() time.Time { return now })
	type asyncResult struct {
		identity Identity
		err      error
	}
	loginResult := make(chan asyncResult, 1)
	go func() {
		identity, err := service.Login(ctx, userID, "one-use-provider-code", "internal-deadlock-login")
		loginResult <- asyncResult{identity: identity, err: err}
	}()

	var schemaName string
	if err := db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&schemaName); err != nil {
		t.Fatal("read deadlock schema failed")
	}
	waiting := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var waits int
		err := db.QueryRowContext(ctx, `
			SELECT COUNT(*)
			FROM performance_schema.data_lock_waits AS w
			JOIN performance_schema.data_locks AS requested
			  ON requested.ENGINE_LOCK_ID=w.REQUESTING_ENGINE_LOCK_ID
			WHERE requested.OBJECT_SCHEMA=? AND requested.OBJECT_NAME='merchant_accounts'
		`, schemaName).Scan(&waits)
		if err != nil {
			t.Fatal("inspect real deadlock wait failed")
		}
		if waits > 0 {
			waiting = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !waiting {
		t.Fatal("merchant login did not reach the real account lock wait")
	}
	var lockedUser uint64
	if err := managementTx.QueryRowContext(ctx, "SELECT id FROM miniprogram_users WHERE id=? FOR UPDATE", userID).Scan(&lockedUser); err != nil || lockedUser != userID {
		t.Fatalf("management transaction was selected as deadlock victim: %v", err)
	}
	if err := managementTx.Commit(); err != nil {
		t.Fatal("commit deadlock management transaction failed")
	}

	select {
	case result := <-loginResult:
		if result.err != nil || result.identity.Merchant == nil || result.identity.Merchant.AuthVersion != 2 {
			t.Fatalf("deadlock-retried login failed: %v", result.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("deadlock-retried login did not complete")
	}
	if provider.calls != 1 {
		t.Fatalf("deadlock path provider calls = %d", provider.calls)
	}
	var boundUserID, recordVersion, authVersion uint64
	if err := db.QueryRowContext(ctx, "SELECT bound_user_id,record_version,auth_version FROM merchant_accounts WHERE id=?", accountID).Scan(&boundUserID, &recordVersion, &authVersion); err != nil {
		t.Fatal("read deadlock binding result failed")
	}
	var loginAuditCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM merchant_action_audits WHERE request_id=?", []byte("internal-deadlock-login")).Scan(&loginAuditCount); err != nil {
		t.Fatal("read deadlock login audit failed")
	}
	if boundUserID != userID || recordVersion != 2 || authVersion != 2 || loginAuditCount != 1 {
		t.Fatal("deadlock recovery left duplicate or partial merchant writes")
	}
}

func assertAuthorizationCommitOrdering(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, time.August, 23, 8, 0, 0, 666666000, time.UTC)
	userID := insertMerchantTestUser(t, db, "opaque-provider-subject-auth-order", now)
	bindPrimaryPhoneFixture(t, db, userID, "+71", now)
	accountID := insertMerchantTestAccount(t, db, "+71", RoleOwner, true, &merchantBindingFixture{UserID: userID, BoundAt: now}, now)
	repository := NewRepository(db)

	authorizationTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal("begin ordered authorization transaction failed")
	}
	authorization, err := repository.AuthorizeInTx(ctx, authorizationTx, userID, ActionOrderRead, Target{Type: "order", ID: 701})
	if err != nil || authorization.MerchantAccountID != accountID || authorization.Actor != ActorMerchantOwner || authorization.AuthVersion != 1 {
		t.Fatalf("ordered authorization failed: %v", err)
	}
	if _, err := authorizationTx.ExecContext(ctx, `
		INSERT INTO merchant_action_audits(
			merchant_account_id,account_id_snapshot,role_snapshot,auth_version_snapshot,
			actor_user_id,action,result,reason,target_type,target_id,request_id,occurred_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
	`, accountID, accountID, RoleOwner, 1, userID, ActionOrderRead, "SUCCEEDED", "AUTHORIZED", "order", uint64(701), []byte("internal-authorized-write"), now); err != nil {
		t.Fatal("insert ordered authorized write audit failed")
	}

	managementResult := make(chan error, 1)
	go func() {
		transaction, err := db.BeginTx(ctx, nil)
		if err != nil {
			managementResult <- err
			return
		}
		defer transaction.Rollback()
		if _, err := transaction.ExecContext(ctx, `
			UPDATE merchant_accounts
			SET role='SUBACCOUNT',record_version=record_version+1,auth_version=auth_version+1,updated_at=?
			WHERE id=? AND record_version=1 AND auth_version=1
		`, now.Add(time.Minute), accountID); err != nil {
			managementResult <- err
			return
		}
		managementResult <- transaction.Commit()
	}()

	var schemaName string
	if err := db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&schemaName); err != nil {
		t.Fatal("read authorization-order schema failed")
	}
	waiting := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var waits int
		if err := db.QueryRowContext(ctx, `
			SELECT COUNT(*)
			FROM performance_schema.data_lock_waits AS w
			JOIN performance_schema.data_locks AS requested
			  ON requested.ENGINE_LOCK_ID=w.REQUESTING_ENGINE_LOCK_ID
			WHERE requested.OBJECT_SCHEMA=? AND requested.OBJECT_NAME='merchant_accounts'
		`, schemaName).Scan(&waits); err != nil {
			t.Fatal("inspect authorization account wait failed")
		}
		if waits > 0 {
			waiting = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !waiting {
		t.Fatal("account role update did not wait on authorization shared lock")
	}
	select {
	case err := <-managementResult:
		t.Fatalf("account change completed before caller transaction ended: %v", err)
	default:
	}
	if err := authorizationTx.Commit(); err != nil {
		t.Fatal("commit authorized business transaction failed")
	}
	select {
	case err := <-managementResult:
		if err != nil {
			t.Fatalf("commit ordered account change failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("account change did not resume after caller commit")
	}

	nextAuthorization, err := authorizeTestAction(t, repository, db, userID, ActionOrderRead)
	if err != nil || nextAuthorization.Actor != ActorMerchantSubaccount || nextAuthorization.RecordVersion != 2 || nextAuthorization.AuthVersion != 2 {
		t.Fatalf("next request did not observe committed role change: %v", err)
	}
	var committedAuditCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM merchant_action_audits WHERE request_id=?", []byte("internal-authorized-write")).Scan(&committedAuditCount); err != nil || committedAuditCount != 1 {
		t.Fatal("authorized caller write did not commit exactly once before role change")
	}

	rollbackTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal("begin rollback authorization transaction failed")
	}
	rollbackAuthorization, err := repository.AuthorizeInTx(ctx, rollbackTx, userID, ActionOrderRead, Target{Type: "order", ID: 702})
	if err != nil || rollbackAuthorization.Actor != ActorMerchantSubaccount {
		t.Fatalf("rollback authorization failed: %v", err)
	}
	if _, err := rollbackTx.ExecContext(ctx, `
		INSERT INTO merchant_action_audits(
			merchant_account_id,account_id_snapshot,role_snapshot,auth_version_snapshot,
			actor_user_id,action,result,reason,target_type,target_id,request_id,occurred_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
	`, accountID, accountID, RoleSubaccount, 2, userID, ActionOrderRead, "SUCCEEDED", "AUTHORIZED", "order", uint64(702), []byte("internal-authorized-rollback"), now); err != nil {
		t.Fatal("insert rollback authorized write failed")
	}
	if err := rollbackTx.Rollback(); err != nil {
		t.Fatal("rollback authorized caller transaction failed")
	}
	var rolledBackAuditCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM merchant_action_audits WHERE request_id=?", []byte("internal-authorized-rollback")).Scan(&rolledBackAuditCount); err != nil || rolledBackAuditCount != 0 {
		t.Fatal("caller rollback retained a business audit")
	}
}

func assertRejectedPhoneCodeAudit(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, time.August, 23, 9, 0, 0, 777777000, time.UTC)
	userID := insertMerchantTestUser(t, db, "opaque-provider-subject-rejected", now)
	provider := &countingPhoneProvider{err: wechat.ErrPhoneCodeRejected}
	service := newService(NewRepository(db), provider, func() time.Time { return now })
	if _, err := service.Login(ctx, userID, "rejected-one-use-code", "internal-rejected-code"); !errors.Is(err, ErrPhoneCodeRejected) {
		t.Fatalf("rejected code result = %v", err)
	}
	if provider.calls != 1 {
		t.Fatalf("rejected code provider calls = %d", provider.calls)
	}
	var snapshotID, snapshotRole, snapshotAuth sql.NullString
	var result, reason string
	if err := db.QueryRowContext(ctx, `
		SELECT account_id_snapshot,role_snapshot,auth_version_snapshot,result,reason
		FROM merchant_action_audits WHERE request_id=?
	`, []byte("internal-rejected-code")).Scan(&snapshotID, &snapshotRole, &snapshotAuth, &result, &reason); err != nil {
		t.Fatal("read rejected-code audit failed")
	}
	if snapshotID.Valid || snapshotRole.Valid || snapshotAuth.Valid || result != "REJECTED" || reason != "PHONE_CODE_REJECTED" {
		t.Fatal("rejected code audit did not retain the unresolved business result")
	}
}

func assertRejectedPhoneCodeAfterConcurrentAccountChange(t *testing.T, db *sql.DB, disable bool) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, time.August, 23, 9, 30, 0, 888888000, time.UTC)
	suffix := "version-drift"
	phone := "+91"
	if disable {
		suffix = "disabled"
		phone = "+92"
	}
	userID := insertMerchantTestUser(t, db, "opaque-provider-subject-"+suffix, now)
	accountID := insertMerchantTestAccount(t, db, phone, RoleOwner, true, nil, now)
	provider := &blockingRejectedPhoneProvider{started: make(chan struct{}), release: make(chan struct{})}
	service := newService(NewRepository(db), provider, func() time.Time { return now })
	requestID := "internal-rejected-" + suffix

	result := make(chan error, 1)
	go func() {
		_, err := service.Login(ctx, userID, "concurrently-consumed-code", requestID)
		result <- err
	}()
	select {
	case <-provider.started:
	case <-time.After(5 * time.Second):
		t.Fatal("provider rejection scenario did not reach exchange")
	}

	bindingTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal("begin concurrent successful binding transaction failed")
	}
	defer bindingTx.Rollback()
	if _, err := bindingTx.ExecContext(ctx, `
		UPDATE miniprogram_users
		SET primary_phone=?,primary_phone_bound_at=?
		WHERE id=? AND primary_phone IS NULL AND primary_phone_bound_at IS NULL
	`, phone, now, userID); err != nil {
		t.Fatal("bind concurrent primary phone failed")
	}
	if _, err := bindingTx.ExecContext(ctx, `
		UPDATE merchant_accounts
		SET bound_user_id=?,bound_at=?,record_version=2,auth_version=2,updated_at=?
		WHERE id=? AND bound_user_id IS NULL AND record_version=1 AND auth_version=1
	`, userID, now, now, accountID); err != nil {
		t.Fatal("bind concurrent merchant account failed")
	}
	boundSnapshot := &accountState{ID: accountID, Role: RoleOwner, Enabled: true, RecordVersion: 2, AuthVersion: 2}
	if err := insertLoginAudit(
		ctx, bindingTx, userID, "internal-success-"+suffix, boundSnapshot,
		"SUCCEEDED", "FIRST_BINDING", "UNBOUND", "BOUND_ENABLED", now,
	); err != nil {
		t.Fatal("write concurrent successful binding audit failed")
	}
	if err := bindingTx.Commit(); err != nil {
		t.Fatal("commit concurrent successful binding failed")
	}

	if disable {
		if _, err := db.ExecContext(ctx, `
			UPDATE merchant_accounts
			SET enabled=FALSE,record_version=3,auth_version=3,updated_at=?
			WHERE id=? AND record_version=2 AND auth_version=2
		`, now.Add(time.Microsecond), accountID); err != nil {
			t.Fatal("disable concurrently bound merchant account failed")
		}
	} else {
		if _, err := db.ExecContext(ctx, `
			UPDATE merchant_accounts
			SET role='SUBACCOUNT',record_version=3,auth_version=3,updated_at=?
			WHERE id=? AND record_version=2 AND auth_version=2
		`, now.Add(time.Microsecond), accountID); err != nil {
			t.Fatal("change concurrently bound merchant account version failed")
		}
	}
	close(provider.release)
	select {
	case err := <-result:
		if !errors.Is(err, ErrPhoneCodeRejected) {
			t.Fatalf("unconfirmed provider rejection result = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("unconfirmed provider rejection did not complete")
	}
	if provider.calls != 1 {
		t.Fatalf("unconfirmed provider rejection calls = %d", provider.calls)
	}

	var merchantAccountID, snapshotID, snapshotRole, snapshotAuth sql.NullString
	var auditResult, reason string
	if err := db.QueryRowContext(ctx, `
		SELECT merchant_account_id,account_id_snapshot,role_snapshot,auth_version_snapshot,result,reason
		FROM merchant_action_audits WHERE request_id=?
	`, []byte(requestID)).Scan(&merchantAccountID, &snapshotID, &snapshotRole, &snapshotAuth, &auditResult, &reason); err != nil {
		t.Fatal("read unconfirmed provider rejection audit failed")
	}
	if merchantAccountID.Valid || snapshotID.Valid || snapshotRole.Valid || snapshotAuth.Valid || auditResult != "REJECTED" || reason != "PHONE_CODE_REJECTED" {
		t.Fatal("unconfirmed provider rejection audit retained an account snapshot")
	}
}

type blockingRejectedPhoneProvider struct {
	started chan struct{}
	release chan struct{}
	calls   int
}

func (provider *blockingRejectedPhoneProvider) Exchange(context.Context, string, string) (string, error) {
	provider.calls++
	close(provider.started)
	<-provider.release
	return "", wechat.ErrPhoneCodeRejected
}

func assertMerchantPIIBoundaries(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	const providerSubjectCanary = "provider-subject-pii-canary"
	const accountNameCanary = "account-name-pii-canary"
	const providerCodeCanary = "provider-code-pii-canary"
	userID := insertMerchantTestUser(t, db, providerSubjectCanary, now)
	bindPrimaryPhoneFixture(t, db, userID, "+81", now)
	result, err := db.ExecContext(ctx, `
		INSERT INTO merchant_accounts(phone,name,role,bound_user_id,bound_at,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?)
	`, "+81", accountNameCanary, RoleOwner, userID, now, now, now)
	if err != nil {
		t.Fatal("insert PII boundary account failed")
	}
	accountID, err := result.LastInsertId()
	if err != nil || accountID <= 0 {
		t.Fatal("resolve PII boundary account failed")
	}

	tokenBytes := bytes.Repeat([]byte{0x8a}, 32)
	sessionToken := base64.RawURLEncoding.EncodeToString(tokenBytes)
	tokenHash := sha256.Sum256([]byte(sessionToken))
	if _, err := db.ExecContext(ctx, `
		INSERT INTO miniprogram_sessions(token_hash,user_id,issued_at,expires_at)
		VALUES (?,?,?,?)
	`, tokenHash[:], userID, now.Add(-time.Minute), now.Add(time.Hour)); err != nil {
		t.Fatal("insert PII boundary session failed")
	}
	sessionService := identity.NewService(unusedSessionExchanger{}, identity.NewRepository(db))
	provider := &countingPhoneProvider{err: errors.New("existing binding must bypass provider")}
	merchantService := newService(NewRepository(db), provider, func() time.Time { return now })
	handler := NewHandler(sessionService, merchantService)
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(func(ctx *gin.Context) {
		ctx.Set("request_id", "internal-pii-request")
		ctx.Next()
	})
	handler.RegisterRoutes(engine)

	identityRequest := httptest.NewRequest(http.MethodGet, "/api/v1/me/identity", nil)
	identityRequest.Header.Set("Authorization", "Bearer "+sessionToken)
	identityResponse := httptest.NewRecorder()
	engine.ServeHTTP(identityResponse, identityRequest)
	if identityResponse.Code != http.StatusOK || identityResponse.Body.String() != `{"user":{"primary_phone_bound":true},"merchant":{"role":"OWNER","auth_version":1}}` {
		t.Fatal("real-session identity response mismatch")
	}
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/me/merchant-login", bytes.NewBufferString(`{"code":"`+providerCodeCanary+`"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginRequest.Header.Set("Authorization", "Bearer "+sessionToken)
	loginResponse := httptest.NewRecorder()
	engine.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK || provider.calls != 0 {
		t.Fatal("real-session idempotent merchant login failed")
	}

	combinedResponse := identityResponse.Body.String() + loginResponse.Body.String()
	for _, canary := range []string{providerSubjectCanary, accountNameCanary, "+81", providerCodeCanary, sessionToken} {
		if strings.Contains(combinedResponse, canary) {
			t.Fatal("merchant identity HTTP response contained a PII canary")
		}
	}
	var auditText string
	if err := db.QueryRowContext(ctx, `
		SELECT CONCAT_WS('|',merchant_account_id,account_id_snapshot,role_snapshot,auth_version_snapshot,
		       actor_user_id,action,result,reason,target_type,target_id,request_id,state_before,state_after,occurred_at)
		FROM merchant_action_audits
		WHERE merchant_account_id=? AND request_id=?
	`, accountID, []byte("internal-pii-request")).Scan(&auditText); err != nil {
		t.Fatal("read PII boundary audit failed")
	}
	for _, canary := range []string{providerSubjectCanary, accountNameCanary, "+81", providerCodeCanary, sessionToken} {
		if strings.Contains(auditText, canary) {
			t.Fatal("durable merchant audit contained a PII canary")
		}
	}
}

type unusedSessionExchanger struct{}

func (unusedSessionExchanger) Exchange(context.Context, string) (string, error) {
	return "", errors.New("session issuance is outside this test")
}

type staticPhoneProvider struct {
	phone string
	err   error
}

func (provider staticPhoneProvider) Exchange(context.Context, string, string) (string, error) {
	return provider.phone, provider.err
}

func withMerchantSchema(t *testing.T, run func(*sql.DB)) {
	t.Helper()
	serverConfig, ok := merchantIntegrationConfig(t, "mysql")
	if !ok {
		t.Skip("merchant identity MySQL integration environment not provided")
	}
	serverDB, err := database.Open(serverConfig)
	if err != nil {
		t.Fatal("open isolated MySQL server failed")
	}
	defer serverDB.Close()
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		t.Fatal("generate isolated schema suffix failed")
	}
	schemaName := "order_merchant_identity_test_" + hex.EncodeToString(random)
	if !merchantSchemaPattern.MatchString(schemaName) {
		t.Fatal("generated schema name was not isolated")
	}
	if _, err := serverDB.ExecContext(context.Background(), "CREATE DATABASE `"+schemaName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal("create isolated merchant identity schema failed")
	}
	defer func() {
		if _, err := serverDB.ExecContext(context.Background(), "DROP DATABASE `"+schemaName+"`"); err != nil {
			t.Error("drop isolated merchant identity schema failed")
		}
	}()

	config, ok := merchantIntegrationConfig(t, schemaName)
	if !ok {
		t.Fatal("merchant identity MySQL environment disappeared")
	}
	db, err := database.Open(config)
	if err != nil {
		t.Fatal("open isolated merchant identity schema failed")
	}
	defer db.Close()
	run(db)
}

func merchantIntegrationConfig(t *testing.T, databaseName string) (database.ConnectionConfig, bool) {
	t.Helper()
	keys := []string{"ORDER_TEST_MYSQL_HOST", "ORDER_TEST_MYSQL_PORT", "ORDER_TEST_MYSQL_USER", "ORDER_TEST_MYSQL_PASSWORD", "ORDER_TEST_MYSQL_TLS_MODE", "ORDER_TEST_MYSQL_INSTANCE", "ORDER_TEST_MYSQL_ISOLATED"}
	present := 0
	for _, key := range keys {
		if os.Getenv(key) != "" {
			present++
		}
	}
	if present == 0 {
		return database.ConnectionConfig{}, false
	}
	if present != len(keys) || os.Getenv("ORDER_TEST_MYSQL_INSTANCE") != "order-mysql-w3" || os.Getenv("ORDER_TEST_MYSQL_ISOLATED") != "YES" {
		t.Fatal("merchant identity MySQL requires the complete isolated test environment")
	}
	port, err := strconv.ParseUint(os.Getenv("ORDER_TEST_MYSQL_PORT"), 10, 16)
	if err != nil {
		t.Fatal("ORDER_TEST_MYSQL_PORT must be a valid port")
	}
	return database.ConnectionConfig{
		Host: os.Getenv("ORDER_TEST_MYSQL_HOST"), Port: uint16(port), Database: databaseName,
		User: os.Getenv("ORDER_TEST_MYSQL_USER"), Password: os.Getenv("ORDER_TEST_MYSQL_PASSWORD"), TLSMode: os.Getenv("ORDER_TEST_MYSQL_TLS_MODE"),
	}, true
}
