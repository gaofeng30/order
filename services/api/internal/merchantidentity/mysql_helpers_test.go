package merchantidentity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/wechat"
)

var merchantSchemaPattern = regexp.MustCompile(`^order_merchant_identity_test_[0-9a-f]{32}$`)

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

func assertMerchantLoginAuditTarget(t *testing.T, db *sql.DB, requestID string, accountID uint64) {
	t.Helper()
	var targetType sql.NullString
	var targetID sql.NullInt64
	var targetKeyHash []byte
	var actorAccountID, snapshotAccountID sql.NullInt64
	if err := db.QueryRowContext(context.Background(), `
		SELECT target_type,target_id,target_key_hash,actor_account_id,actor_account_id_snapshot
		FROM action_audits WHERE request_id_hash=?
	`, merchantRequestIDHash(requestID)).Scan(&targetType, &targetID, &targetKeyHash, &actorAccountID, &snapshotAccountID); err != nil {
		t.Fatal("read merchant login audit target failed")
	}
	if !targetType.Valid || targetType.String != "merchant_login_code" || targetID.Valid || len(targetKeyHash) != sha256.Size {
		t.Fatal("merchant login audit did not retain the hashed provider-code target")
	}
	if accountID == 0 {
		if actorAccountID.Valid || snapshotAccountID.Valid {
			t.Fatal("unresolved merchant login audit retained an account")
		}
		return
	}
	if !actorAccountID.Valid || uint64(actorAccountID.Int64) != accountID || !snapshotAccountID.Valid || uint64(snapshotAccountID.Int64) != accountID {
		t.Fatal("resolved merchant login audit account was not exact")
	}
}

func merchantRequestIDHash(requestID string) []byte {
	hash := sha256.Sum256([]byte(requestID))
	return hash[:]
}

type merchantAuditExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func insertMerchantAuditFixture(ctx context.Context, execer merchantAuditExecer, accountID uint64, role Role, authVersion, userID uint64, action Action, result, reason, targetType string, targetID uint64, requestID string, at time.Time) error {
	scopeHash := sha256.Sum256([]byte("TEST_MERCHANT_SCOPE\x00" + strconv.FormatUint(userID, 10) + "\x00" + strconv.FormatUint(accountID, 10)))
	var liveAccount, snapshotID, snapshotRole, snapshotAuth any
	if accountID != 0 {
		liveAccount, snapshotID, snapshotRole, snapshotAuth = accountID, accountID, role, authVersion
	}
	_, err := execer.ExecContext(ctx, `
		INSERT INTO action_audits(
			entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,
			actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,
			action,result,reason_code,target_type,target_id,request_id_hash,occurred_at
		) VALUES ('LEGACY_EVIDENCE','MERCHANT',?,?,?,?,?,?,?,?,?,?,?,?,?)
	`, scopeHash[:], userID, liveAccount, snapshotID, snapshotRole, snapshotAuth, action, result, reason, targetType, targetID, merchantRequestIDHash(requestID), at)
	return err
}

func insertSystemAuditFixture(ctx context.Context, execer merchantAuditExecer, requestID string, at time.Time) error {
	scopeHash := sha256.Sum256([]byte("TEST_SYSTEM_SCOPE\x00" + requestID))
	_, err := execer.ExecContext(ctx, `
		INSERT INTO action_audits(
			entry_kind,actor_kind,actor_scope_hash,action,result,reason_code,request_id_hash,occurred_at
		) VALUES ('SYSTEM_EVIDENCE','SYSTEM',?,'deadlock.fixture','SUCCEEDED','FIXTURE',?,?)
	`, scopeHash[:], merchantRequestIDHash(requestID), at)
	return err
}

type staticPhoneProvider struct {
	phone string
	err   error
}

func (provider staticPhoneProvider) Exchange(context.Context, string, string) (string, error) {
	return provider.phone, provider.err
}

type phoneProviderFunc func(context.Context, string, string) (string, error)

func (provider phoneProviderFunc) Exchange(ctx context.Context, code, openID string) (string, error) {
	return provider(ctx, code, openID)
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
