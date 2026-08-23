package merchantidentity

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/wechat"
)

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

func assertConcurrentDifferentCodeMerchantLogin(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, time.August, 23, 5, 30, 0, 343434000, time.UTC)
	userID := insertMerchantTestUser(t, db, "opaque-provider-subject-different-codes", now)
	accountID := insertMerchantTestAccount(t, db, "+42", RoleOwner, true, nil, now)
	repository := NewRepository(db)
	committed := make(chan struct{})
	store := &signalingCompleteStore{Store: repository, committed: committed}
	provider := &concurrentSameCodeProvider{
		phone: "+42", secondAtProvider: make(chan struct{}), committed: committed,
	}
	service := newService(store, provider, func() time.Time { return now })

	type loginResult struct {
		identity Identity
		err      error
	}
	results := make(chan loginResult, 2)
	for index, code := range []string{"first-one-use-code", "different-one-use-code"} {
		requestID := "internal-different-code-" + strconv.Itoa(index)
		go func() {
			identity, err := service.Login(ctx, userID, code, requestID)
			results <- loginResult{identity: identity, err: err}
		}()
	}
	successes := 0
	rejections := 0
	for index := 0; index < 2; index++ {
		result := <-results
		switch {
		case result.err == nil && result.identity.Merchant != nil && result.identity.Merchant.AuthVersion == 2:
			successes++
		case errors.Is(result.err, ErrPhoneCodeRejected):
			rejections++
		default:
			t.Fatalf("different-code concurrent result invalid: %v", result.err)
		}
	}
	if successes != 1 || rejections != 1 || provider.Calls() != 2 {
		t.Fatalf("different-code result successes=%d rejections=%d provider_calls=%d", successes, rejections, provider.Calls())
	}
	var boundUserID, authVersion uint64
	if err := db.QueryRowContext(ctx, "SELECT bound_user_id,auth_version FROM merchant_accounts WHERE id=?", accountID).Scan(&boundUserID, &authVersion); err != nil || boundUserID != userID || authVersion != 2 {
		t.Fatal("different-code concurrency changed binding more than once")
	}
	var auditSuccesses, auditRejections int
	if err := db.QueryRowContext(ctx, `
		SELECT SUM(result='SUCCEEDED'),SUM(result='REJECTED')
		FROM merchant_action_audits WHERE request_id IN (?,?)
	`, []byte("internal-different-code-0"), []byte("internal-different-code-1")).Scan(&auditSuccesses, &auditRejections); err != nil {
		t.Fatal("read different-code login audits failed")
	}
	if auditSuccesses != 1 || auditRejections != 1 {
		t.Fatal("different-code audits did not retain one success and one rejection")
	}
	var accountSnapshot, roleSnapshot, authSnapshot sql.NullString
	if err := db.QueryRowContext(ctx, `
		SELECT account_id_snapshot,role_snapshot,auth_version_snapshot
		FROM merchant_action_audits
		WHERE request_id IN (?,?) AND result='REJECTED'
	`, []byte("internal-different-code-0"), []byte("internal-different-code-1")).Scan(&accountSnapshot, &roleSnapshot, &authSnapshot); err != nil {
		t.Fatal("read different-code rejection snapshot failed")
	}
	if accountSnapshot.Valid || roleSnapshot.Valid || authSnapshot.Valid {
		t.Fatal("different-code rejection retained an account snapshot")
	}
}

func assertConcurrentSuccessfulPhoneMismatch(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, time.August, 23, 5, 45, 0, 353535000, time.UTC)
	userID := insertMerchantTestUser(t, db, "opaque-provider-subject-concurrent-phones", now)
	firstAccountID := insertMerchantTestAccount(t, db, "+43", RoleOwner, true, nil, now)
	secondAccountID := insertMerchantTestAccount(t, db, "+44", RoleSubaccount, true, nil, now)

	var providerMu sync.Mutex
	providerCalls := 0
	bothAtProvider := make(chan struct{})
	provider := phoneProviderFunc(func(_ context.Context, code, _ string) (string, error) {
		providerMu.Lock()
		providerCalls++
		if providerCalls == 2 {
			close(bothAtProvider)
		}
		providerMu.Unlock()
		<-bothAtProvider
		if code == "first-valid-phone-code" {
			return "+43", nil
		}
		return "+44", nil
	})
	repository := NewRepository(db)
	service := newService(repository, provider, func() time.Time { return now })

	type loginResult struct {
		identity Identity
		err      error
	}
	results := make(chan loginResult, 2)
	for index, code := range []string{"first-valid-phone-code", "second-valid-phone-code"} {
		requestID := "internal-concurrent-phone-" + strconv.Itoa(index)
		go func() {
			identity, err := service.Login(ctx, userID, code, requestID)
			results <- loginResult{identity: identity, err: err}
		}()
	}
	successes := 0
	mismatches := 0
	for index := 0; index < 2; index++ {
		result := <-results
		switch {
		case result.err == nil && result.identity.Merchant != nil:
			successes++
		case errors.Is(result.err, ErrPrimaryPhoneMismatch):
			mismatches++
		default:
			t.Fatalf("concurrent different-phone result invalid: %v", result.err)
		}
	}
	if successes != 1 || mismatches != 1 || providerCalls != 2 {
		t.Fatalf("concurrent different-phone successes=%d mismatches=%d provider_calls=%d", successes, mismatches, providerCalls)
	}

	var primaryPhone string
	var primaryPhoneBoundAt time.Time
	if err := db.QueryRowContext(ctx, "SELECT primary_phone,primary_phone_bound_at FROM miniprogram_users WHERE id=?", userID).Scan(&primaryPhone, &primaryPhoneBoundAt); err != nil {
		t.Fatal("read concurrent primary phone failed")
	}
	var boundAccountID uint64
	if err := db.QueryRowContext(ctx, "SELECT id FROM merchant_accounts WHERE bound_user_id=?", userID).Scan(&boundAccountID); err != nil {
		t.Fatal("read concurrent bound account failed")
	}
	if (primaryPhone == "+43" && boundAccountID != firstAccountID) || (primaryPhone == "+44" && boundAccountID != secondAccountID) {
		t.Fatal("concurrent primary phone and merchant binding diverged")
	}
	var auditSuccesses, auditMismatches int
	if err := db.QueryRowContext(ctx, `
		SELECT SUM(result='SUCCEEDED'),SUM(result='REJECTED' AND reason='PRIMARY_PHONE_MISMATCH')
		FROM merchant_action_audits WHERE request_id IN (?,?)
	`, []byte("internal-concurrent-phone-0"), []byte("internal-concurrent-phone-1")).Scan(&auditSuccesses, &auditMismatches); err != nil {
		t.Fatal("read concurrent different-phone audits failed")
	}
	if auditSuccesses != 1 || auditMismatches != 1 {
		t.Fatal("concurrent different-phone audits did not retain one success and one mismatch")
	}

	if _, err := db.ExecContext(ctx, `
		UPDATE merchant_accounts
		SET enabled=FALSE,record_version=record_version+1,auth_version=auth_version+1,updated_at=?
		WHERE id=?
	`, now.Add(time.Microsecond), boundAccountID); err != nil {
		t.Fatal("disable concurrent binding failed")
	}
	var beforeBoundUserID, beforeRecordVersion, beforeAuthVersion uint64
	var beforeEnabled bool
	var beforeBoundAt time.Time
	if err := db.QueryRowContext(ctx, `
		SELECT bound_user_id,bound_at,enabled,record_version,auth_version FROM merchant_accounts WHERE id=?
	`, boundAccountID).Scan(&beforeBoundUserID, &beforeBoundAt, &beforeEnabled, &beforeRecordVersion, &beforeAuthVersion); err != nil {
		t.Fatal("snapshot disabled concurrent binding failed")
	}
	differentPhone := "+43"
	unboundAccountID := firstAccountID
	if primaryPhone == differentPhone {
		differentPhone = "+44"
		unboundAccountID = secondAccountID
	}
	disabledRequestID := "internal-concurrent-phone-disabled"
	if _, err := repository.CompleteLogin(ctx, userID, differentPhone, hashLoginCode("disabled-binding-code"), disabledRequestID, now.Add(2*time.Microsecond)); !errors.Is(err, ErrPrimaryPhoneMismatch) {
		t.Fatalf("disabled concurrent binding mismatch error = %v", err)
	}
	var boundUserID, recordVersion, authVersion uint64
	var enabled bool
	var boundAt time.Time
	if err := db.QueryRowContext(ctx, `
		SELECT bound_user_id,bound_at,enabled,record_version,auth_version FROM merchant_accounts WHERE id=?
	`, boundAccountID).Scan(&boundUserID, &boundAt, &enabled, &recordVersion, &authVersion); err != nil {
		t.Fatal("read disabled concurrent binding failed")
	}
	if boundUserID != beforeBoundUserID || !boundAt.Equal(beforeBoundAt) || enabled != beforeEnabled || recordVersion != beforeRecordVersion || authVersion != beforeAuthVersion {
		t.Fatal("mismatch changed the disabled concurrent binding")
	}
	var afterPrimaryPhone string
	var afterPrimaryPhoneBoundAt time.Time
	if err := db.QueryRowContext(ctx, "SELECT primary_phone,primary_phone_bound_at FROM miniprogram_users WHERE id=?", userID).Scan(&afterPrimaryPhone, &afterPrimaryPhoneBoundAt); err != nil || afterPrimaryPhone != primaryPhone || !afterPrimaryPhoneBoundAt.Equal(primaryPhoneBoundAt) {
		t.Fatal("mismatch changed the primary phone binding")
	}
	var otherBoundUser sql.NullInt64
	var otherBoundAt sql.NullTime
	var otherRecordVersion, otherAuthVersion uint64
	if err := db.QueryRowContext(ctx, "SELECT bound_user_id,bound_at,record_version,auth_version FROM merchant_accounts WHERE id=?", unboundAccountID).Scan(&otherBoundUser, &otherBoundAt, &otherRecordVersion, &otherAuthVersion); err != nil || otherBoundUser.Valid || otherBoundAt.Valid || otherRecordVersion != 1 || otherAuthVersion != 1 {
		t.Fatal("mismatch partially bound the provider-phone account")
	}
	var snapshotID, snapshotAuth uint64
	var snapshotRole Role
	var result, reason string
	if err := db.QueryRowContext(ctx, `
		SELECT account_id_snapshot,role_snapshot,auth_version_snapshot,result,reason
		FROM merchant_action_audits WHERE request_id=?
	`, []byte(disabledRequestID)).Scan(&snapshotID, &snapshotRole, &snapshotAuth, &result, &reason); err != nil {
		t.Fatal("read disabled concurrent mismatch audit failed")
	}
	if snapshotID != boundAccountID || !validRole(snapshotRole) || snapshotAuth != 3 || result != "REJECTED" || reason != "PRIMARY_PHONE_MISMATCH" {
		t.Fatal("disabled concurrent mismatch audit snapshot was incomplete")
	}
}

type signalingCompleteStore struct {
	Store
	committed chan struct{}
	once      sync.Once
}

func (store *signalingCompleteStore) CompleteLogin(ctx context.Context, userID uint64, phone string, codeHash LoginCodeHash, requestID string, at time.Time) (Identity, error) {
	identity, err := store.Store.CompleteLogin(ctx, userID, phone, codeHash, requestID, at)
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
		if _, err := repository.CompleteLogin(ctx, userID, "+51", hashLoginCode("rollback-code"), "internal-rollback", now); !errors.Is(err, ErrUnavailable) {
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
		if _, err := NewRepository(db).CompleteLogin(ctx, userID, "+52", hashLoginCode("audit-unavailable-code"), "internal-audit-unavailable", now); !errors.Is(err, ErrUnavailable) {
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
		codeHash := hashLoginCode("commit-unknown-code")
		if _, err := repository.CompleteLogin(ctx, userID, "+53", codeHash, "internal-commit-unknown", now); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("commit-unknown result = %v", err)
		}
		var boundUserID, authVersion uint64
		if err := db.QueryRowContext(ctx, "SELECT bound_user_id,auth_version FROM merchant_accounts WHERE id=?", accountID).Scan(&boundUserID, &authVersion); err != nil || boundUserID != userID || authVersion != 2 {
			t.Fatal("committed fact was not durable after unknown outcome")
		}
		recovered, err := NewRepository(db).RecoverRejectedLogin(
			ctx, userID, codeHash, "internal-commit-recovery", now, now.Add(time.Microsecond),
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
		ctx, bindingTx, userID, hashLoginCode("concurrently-consumed-code"), "internal-success-"+suffix, boundSnapshot,
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
