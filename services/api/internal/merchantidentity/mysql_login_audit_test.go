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

func assertAlreadyBoundCannotConfirmRejectedLogin(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, time.August, 23, 5, 55, 0, 373737000, time.UTC)
	userID := insertMerchantTestUser(t, db, "opaque-provider-subject-already-proof", now)
	accountID := insertMerchantTestAccount(t, db, "+49", RoleOwner, true, nil, now)
	repository := NewRepository(db)
	rejectedHash := hashLoginCode("rejected-code-with-later-already-bound")

	start, err := repository.StartLogin(ctx, userID, rejectedHash, "internal-already-proof-start", now)
	if err != nil || start.AlreadyBound || start.OpenID == "" {
		t.Fatalf("start rejected-code attempt = %v", err)
	}
	boundAt := now.Add(time.Microsecond)
	if _, err := repository.CompleteLogin(ctx, userID, "+49", hashLoginCode("different-success-code"), "internal-already-proof-binding", boundAt); err != nil {
		t.Fatalf("complete independent first binding = %v", err)
	}
	already, err := repository.StartLogin(ctx, userID, rejectedHash, "internal-already-proof-candidate", now.Add(2*time.Microsecond))
	if err != nil || !already.AlreadyBound || already.Existing.Merchant == nil || already.Existing.Merchant.Role != RoleOwner || already.Existing.Merchant.AuthVersion != 2 {
		t.Fatalf("create ALREADY_BOUND proof candidate = %v", err)
	}
	var candidateReason string
	var candidateHash []byte
	if err := db.QueryRowContext(ctx, "SELECT reason,idempotency_key_hash FROM merchant_action_audits WHERE request_id=?", []byte("internal-already-proof-candidate")).Scan(&candidateReason, &candidateHash); err != nil || candidateReason != "ALREADY_BOUND" || !bytes.Equal(candidateHash, rejectedHash[:]) {
		t.Fatal("ALREADY_BOUND proof candidate was not exact")
	}

	if _, err := repository.RecoverRejectedLogin(ctx, userID, rejectedHash, "internal-already-proof-rejected", now, now.Add(3*time.Microsecond)); !errors.Is(err, ErrPhoneCodeRejected) {
		t.Fatalf("ALREADY_BOUND recovery result = %v", err)
	}
	var merchantAccountID, snapshotID, snapshotAuth sql.NullInt64
	var snapshotRole sql.NullString
	var result, reason string
	var rejectionHash []byte
	if err := db.QueryRowContext(ctx, `
		SELECT merchant_account_id,account_id_snapshot,role_snapshot,auth_version_snapshot,result,reason,idempotency_key_hash
		FROM merchant_action_audits WHERE request_id=?
	`, []byte("internal-already-proof-rejected")).Scan(&merchantAccountID, &snapshotID, &snapshotRole, &snapshotAuth, &result, &reason, &rejectionHash); err != nil {
		t.Fatal("read ALREADY_BOUND rejection audit failed")
	}
	if merchantAccountID.Valid || snapshotID.Valid || snapshotRole.Valid || snapshotAuth.Valid || result != "REJECTED" || reason != "PHONE_CODE_REJECTED" || !bytes.Equal(rejectionHash, rejectedHash[:]) {
		t.Fatal("ALREADY_BOUND rejection audit was not unresolved and exact")
	}
	var primaryPhone string
	var primaryBoundAt, accountBoundAt time.Time
	var boundUserID, recordVersion, authVersion uint64
	if err := db.QueryRowContext(ctx, "SELECT primary_phone,primary_phone_bound_at FROM miniprogram_users WHERE id=?", userID).Scan(&primaryPhone, &primaryBoundAt); err != nil || primaryPhone != "+49" || !primaryBoundAt.Equal(boundAt) {
		t.Fatal("rejected recovery changed the primary phone binding")
	}
	if err := db.QueryRowContext(ctx, "SELECT bound_user_id,bound_at,record_version,auth_version FROM merchant_accounts WHERE id=?", accountID).Scan(&boundUserID, &accountBoundAt, &recordVersion, &authVersion); err != nil || boundUserID != userID || !accountBoundAt.Equal(boundAt) || recordVersion != 2 || authVersion != 2 {
		t.Fatal("rejected recovery changed the merchant binding")
	}
}

func assertExistingPrimaryPhoneCompletesFirstBinding(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, time.August, 23, 6, 0, 0, 383838000, time.UTC)
	primaryBoundAt := now.Add(-time.Minute)
	userID := insertMerchantTestUser(t, db, "opaque-provider-subject-existing-primary", now)
	bindPrimaryPhoneFixture(t, db, userID, "+50", primaryBoundAt)
	accountID := insertMerchantTestAccount(t, db, "+50", RoleSubaccount, true, nil, now)
	provider := &countingPhoneProvider{phone: "+50"}
	service := newService(NewRepository(db), provider, func() time.Time { return now })
	code := "first-binding-existing-primary-code"
	requestID := "internal-FIRST_BINDING_EXISTING_PRIMARY"

	projection, err := service.Login(ctx, userID, code, requestID)
	if err != nil || projection.Merchant == nil || projection.Merchant.Role != RoleSubaccount || projection.Merchant.AuthVersion != 2 || !projection.PrimaryPhoneBound || provider.calls != 1 {
		t.Fatalf("FIRST_BINDING_EXISTING_PRIMARY result = %v", err)
	}

	var merchantAccountID, snapshotID, snapshotAuth uint64
	var snapshotRole Role
	var result, reason string
	var idempotencyHash []byte
	if err := db.QueryRowContext(ctx, `
		SELECT merchant_account_id,account_id_snapshot,role_snapshot,auth_version_snapshot,result,reason,idempotency_key_hash
		FROM merchant_action_audits WHERE request_id=?
	`, []byte(requestID)).Scan(&merchantAccountID, &snapshotID, &snapshotRole, &snapshotAuth, &result, &reason, &idempotencyHash); err != nil {
		t.Fatal("read FIRST_BINDING_EXISTING_PRIMARY audit failed")
	}
	wantHash := hashLoginCode(code)
	if merchantAccountID != accountID || snapshotID != accountID || snapshotRole != RoleSubaccount || snapshotAuth != 2 || result != "SUCCEEDED" || reason != "FIRST_BINDING" || !bytes.Equal(idempotencyHash, wantHash[:]) {
		t.Fatal("FIRST_BINDING_EXISTING_PRIMARY audit facts were not exact")
	}

	var primaryPhone string
	var storedPrimaryBoundAt, accountBoundAt time.Time
	if err := db.QueryRowContext(ctx, "SELECT primary_phone,primary_phone_bound_at FROM miniprogram_users WHERE id=?", userID).Scan(&primaryPhone, &storedPrimaryBoundAt); err != nil || primaryPhone != "+50" || !storedPrimaryBoundAt.Equal(primaryBoundAt) {
		t.Fatal("FIRST_BINDING_EXISTING_PRIMARY changed the existing primary phone binding")
	}
	var boundUserID, recordVersion, authVersion uint64
	if err := db.QueryRowContext(ctx, "SELECT bound_user_id,bound_at,record_version,auth_version FROM merchant_accounts WHERE id=?", accountID).Scan(&boundUserID, &accountBoundAt, &recordVersion, &authVersion); err != nil || boundUserID != userID || !accountBoundAt.Equal(now) || recordVersion != 2 || authVersion != 2 {
		t.Fatal("FIRST_BINDING_EXISTING_PRIMARY merchant binding was not exact")
	}
}
