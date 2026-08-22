package merchantidentity

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/migrations"
)

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
