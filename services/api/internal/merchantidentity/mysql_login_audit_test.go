package merchantidentity

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

func assertResolvedLoginAuditBranches(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, time.August, 23, 5, 50, 0, 363636000, time.UTC)
	userID := insertMerchantTestUser(t, db, "opaque-provider-subject-login-audit-branches", now)
	bindPrimaryPhoneFixture(t, db, userID, "+47", now)
	accountID := insertMerchantTestAccount(t, db, "+47", RoleSubaccount, true, &merchantBindingFixture{UserID: userID, BoundAt: now}, now)
	peerAccountID := insertMerchantTestAccount(t, db, "+48", RoleOwner, true, nil, now)
	repository := NewRepository(db)
	provider := &countingPhoneProvider{err: errors.New("provider must be bypassed")}
	service := newService(repository, provider, func() time.Time { return now })

	assertState := func(label string) {
		t.Helper()
		var primaryPhone string
		var primaryPhoneBoundAt time.Time
		if err := db.QueryRowContext(ctx, "SELECT primary_phone,primary_phone_bound_at FROM miniprogram_users WHERE id=?", userID).Scan(&primaryPhone, &primaryPhoneBoundAt); err != nil || primaryPhone != "+47" || !primaryPhoneBoundAt.Equal(now) {
			t.Fatalf("%s changed the primary phone binding", label)
		}
		var boundUserID, recordVersion, authVersion uint64
		var boundAt time.Time
		var enabled bool
		var role Role
		if err := db.QueryRowContext(ctx, "SELECT bound_user_id,bound_at,enabled,record_version,auth_version,role FROM merchant_accounts WHERE id=?", accountID).Scan(&boundUserID, &boundAt, &enabled, &recordVersion, &authVersion, &role); err != nil || boundUserID != userID || !boundAt.Equal(now) || !enabled || recordVersion != 1 || authVersion != 1 || role != RoleSubaccount {
			t.Fatalf("%s changed the existing merchant binding", label)
		}
		var peerBoundUser sql.NullInt64
		var peerBoundAt sql.NullTime
		var peerRecordVersion, peerAuthVersion uint64
		if err := db.QueryRowContext(ctx, "SELECT bound_user_id,bound_at,record_version,auth_version FROM merchant_accounts WHERE id=?", peerAccountID).Scan(&peerBoundUser, &peerBoundAt, &peerRecordVersion, &peerAuthVersion); err != nil || peerBoundUser.Valid || peerBoundAt.Valid || peerRecordVersion != 1 || peerAuthVersion != 1 {
			t.Fatalf("%s changed the peer merchant account", label)
		}
	}
	assertAudit := func(requestID, wantResult, wantReason string, wantHash LoginCodeHash) {
		t.Helper()
		var merchantAccountID, snapshotID, snapshotAuth uint64
		var snapshotRole Role
		var result, reason string
		var idempotencyHash []byte
		if err := db.QueryRowContext(ctx, `
			SELECT merchant_account_id,account_id_snapshot,role_snapshot,auth_version_snapshot,result,reason,idempotency_key_hash
			FROM merchant_action_audits WHERE request_id=?
		`, []byte(requestID)).Scan(&merchantAccountID, &snapshotID, &snapshotRole, &snapshotAuth, &result, &reason, &idempotencyHash); err != nil {
			t.Fatalf("read %s audit failed", requestID)
		}
		if merchantAccountID != accountID || snapshotID != accountID || snapshotRole != RoleSubaccount || snapshotAuth != 1 || result != wantResult || reason != wantReason || !bytes.Equal(idempotencyHash, wantHash[:]) {
			t.Fatalf("%s audit facts were not exact", requestID)
		}
	}

	alreadyHash := hashLoginCode("already-bound-branch-code")
	already, err := service.Login(ctx, userID, "already-bound-branch-code", "internal-branch-ALREADY_BOUND")
	if err != nil || already.Merchant == nil || already.Merchant.Role != RoleSubaccount || already.Merchant.AuthVersion != 1 || provider.calls != 0 {
		t.Fatalf("ALREADY_BOUND result = %v", err)
	}
	assertAudit("internal-branch-ALREADY_BOUND", "SUCCEEDED", "ALREADY_BOUND", alreadyHash)
	assertState("ALREADY_BOUND")

	sameHash := hashLoginCode("complete-found-same-code")
	same, err := repository.CompleteLogin(ctx, userID, "+47", sameHash, "internal-branch-COMPLETE_FOUND_SAME_ENABLED", now.Add(time.Microsecond))
	if err != nil || same.Merchant == nil || same.Merchant.Role != RoleSubaccount || same.Merchant.AuthVersion != 1 {
		t.Fatalf("COMPLETE_FOUND_SAME_ENABLED result = %v", err)
	}
	assertAudit("internal-branch-COMPLETE_FOUND_SAME_ENABLED", "SUCCEEDED", "CONCURRENT_BINDING_CONFIRMED", sameHash)
	assertState("COMPLETE_FOUND_SAME_ENABLED")

	differentHash := hashLoginCode("complete-found-different-code")
	if _, err := repository.CompleteLogin(ctx, userID, "+48", differentHash, "internal-branch-COMPLETE_FOUND_DIFFERENT_ENABLED", now.Add(2*time.Microsecond)); !errors.Is(err, ErrPrimaryPhoneMismatch) {
		t.Fatalf("COMPLETE_FOUND_DIFFERENT_ENABLED result = %v", err)
	}
	assertAudit("internal-branch-COMPLETE_FOUND_DIFFERENT_ENABLED", "REJECTED", "PRIMARY_PHONE_MISMATCH", differentHash)
	assertState("COMPLETE_FOUND_DIFFERENT_ENABLED")

	recovered, err := repository.RecoverRejectedLogin(ctx, userID, sameHash, "internal-branch-RECOVER_CONFIRMED_SUCCESS", now, now.Add(3*time.Microsecond))
	if err != nil || recovered.Merchant == nil || recovered.Merchant.Role != RoleSubaccount || recovered.Merchant.AuthVersion != 1 {
		t.Fatalf("RECOVER_CONFIRMED_SUCCESS result = %v", err)
	}
	assertAudit("internal-branch-RECOVER_CONFIRMED_SUCCESS", "SUCCEEDED", "CONCURRENT_BINDING_CONFIRMED", sameHash)
	assertState("RECOVER_CONFIRMED_SUCCESS")
}
