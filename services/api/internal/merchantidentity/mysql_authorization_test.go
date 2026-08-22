package merchantidentity

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

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
