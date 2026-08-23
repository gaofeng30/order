package identity

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/internal/wechat"
	"github.com/gaofeng30/order/services/api/migrations"
)

var identitySchemaPattern = regexp.MustCompile(`^order_identity_test_[0-9a-f]{32}$`)

func TestMiniprogramSessionMySQLIntegration(t *testing.T) {
	withIdentitySchema(t, func(db *sql.DB) {
		migrationSet := loadHistoricalMigrations(t, 10)

		foundation, err := migrate.Run(context.Background(), db, migrationSet[:1])
		if err != nil || foundation.AppliedCount != 1 || foundation.ToVersion != 1 {
			t.Fatal("foundation migration did not establish version 1")
		}
		identityResult, err := migrate.Run(context.Background(), db, migrationSet)
		if err != nil || identityResult.FromVersion != 1 || identityResult.ToVersion != 10 || identityResult.AppliedCount != 9 {
			t.Fatal("identity migrations did not advance version 1 to version 10")
		}
		repeat, err := migrate.Run(context.Background(), db, migrationSet)
		if err != nil || repeat.FromVersion != 10 || repeat.ToVersion != 10 || repeat.AppliedCount != 0 {
			t.Fatal("repeated identity migration was not a zero-write success")
		}

		assertIdentitySchema(t, db)
		t.Run("first login hash lookup and expiry", func(t *testing.T) {
			assertFirstLoginAndExpiry(t, db)
		})
		t.Run("same openid concurrent login", func(t *testing.T) {
			assertConcurrentLogin(t, db)
		})
		t.Run("transaction rollback", func(t *testing.T) {
			assertTransactionRollback(t, db)
		})
	})
}

func TestMiniprogramPhoneMySQLIntegration(t *testing.T) {
	withIdentitySchema(t, func(db *sql.DB) {
		migrationSet := loadHistoricalMigrations(t, 10)
		if result, err := migrate.Run(context.Background(), db, migrationSet[:9]); err != nil || result.ToVersion != 9 || result.AppliedCount != 9 {
			t.Fatal("establish v9 identity baseline failed")
		}
		createdAt := time.Date(2026, time.August, 20, 7, 0, 0, 123456000, time.UTC)
		if _, err := db.ExecContext(context.Background(), "INSERT INTO miniprogram_users(openid,created_at,last_login_at) VALUES (?,?,?)", "pre-v10-openid", createdAt, createdAt); err != nil {
			t.Fatal("insert pre-v10 identity row failed")
		}
		if result, err := migrate.Run(context.Background(), db, migrationSet); err != nil || result.FromVersion != 9 || result.ToVersion != 10 || result.AppliedCount != 1 {
			t.Fatal("advance identity v9 to v10 failed")
		}
		assertIdentitySchema(t, db)
		var phone, boundAt sql.NullString
		if err := db.QueryRowContext(context.Background(), "SELECT primary_phone,primary_phone_bound_at FROM miniprogram_users WHERE openid=?", "pre-v10-openid").Scan(&phone, &boundAt); err != nil {
			t.Fatal("read pre-v10 identity row failed")
		}
		if phone.Valid || boundAt.Valid {
			t.Fatal("v10 changed existing identity phone state")
		}
		if _, err := db.ExecContext(context.Background(), "UPDATE miniprogram_users SET primary_phone=? WHERE openid=?", "+8613800001234", "pre-v10-openid"); err == nil {
			t.Fatal("phone pair check accepted a partial binding")
		}

		t.Run("first bind and same phone idempotency", func(t *testing.T) {
			assertFirstPhoneBind(t, db)
		})
		t.Run("different phone same user concurrency", func(t *testing.T) {
			assertConcurrentDifferentPhoneSameUser(t, db)
		})
		t.Run("same phone cross user concurrency", func(t *testing.T) {
			assertConcurrentSamePhoneCrossUser(t, db)
		})
		t.Run("statement and commit rollback", func(t *testing.T) {
			assertPhoneTransactionRollback(t, db)
		})
		t.Run("same code rejection recovery", func(t *testing.T) {
			assertSameCodePhoneRecovery(t, db)
		})
	})
}

func TestMiniprogramPhoneStatusMySQLIntegration(t *testing.T) {
	withIdentitySchema(t, func(db *sql.DB) {
		migrationSet, err := migrate.Load(migrations.FS)
		if err != nil {
			t.Fatal("load migrations failed")
		}
		if result, err := migrate.Run(context.Background(), db, migrationSet); err != nil || result.FromVersion != 0 || result.ToVersion != migrationSet[len(migrationSet)-1].Version || result.AppliedCount != len(migrationSet) {
			t.Fatal("establish phone-status schema failed")
		}

		repository := NewRepository(db)
		now := time.Date(2026, time.August, 20, 13, 0, 0, 123456000, time.UTC)
		boundSessionService := newService(
			mysqlStaticExchanger{openid: "status-bound-openid"}, repository,
			func() time.Time { return now }, bytes.NewReader(bytes.Repeat([]byte{0x55}, tokenEntropyBytes)),
		)
		boundSession, err := boundSessionService.Issue(context.Background(), "bound-login-code")
		if err != nil {
			t.Fatal("create bound status session failed")
		}
		unboundSessionService := newService(
			mysqlStaticExchanger{openid: "status-unbound-openid"}, repository,
			func() time.Time { return now }, bytes.NewReader(bytes.Repeat([]byte{0x66}, tokenEntropyBytes)),
		)
		unboundSession, err := unboundSessionService.Issue(context.Background(), "unbound-login-code")
		if err != nil {
			t.Fatal("create unbound status session failed")
		}
		var boundUserID uint64
		if err := db.QueryRowContext(context.Background(), "SELECT id FROM miniprogram_users WHERE openid=?", "status-bound-openid").Scan(&boundUserID); err != nil {
			t.Fatal("read bound status user failed")
		}
		boundAt := now.Add(time.Minute)
		if _, err := repository.BindPrimaryPhone(context.Background(), boundUserID, "+8613712345678", boundAt); err != nil {
			t.Fatal("seed bound status phone failed")
		}
		for _, invalidPhone := range []string{"", "+0"} {
			if _, err := db.ExecContext(context.Background(), "UPDATE miniprogram_users SET primary_phone=?,primary_phone_bound_at=? WHERE openid=?", invalidPhone, boundAt, "status-unbound-openid"); err == nil {
				t.Fatalf("v44 accepted invalid primary phone %q", invalidPhone)
			}
		}

		before := readPhoneStatusDatabaseSnapshot(t, db)
		provider := &phoneStatusProviderSpy{}
		store := &phoneStatusStoreSpy{repository: repository}
		phoneService := newPhoneService(provider, store, func() time.Time { return now })
		router, _ := phoneHandlerTestRouter(boundSessionService, phoneService)

		assertMySQLPhoneStatusHTTP(t, router, boundSession.AccessToken, http.StatusOK, `{"primary_phone_bound":true,"masked_phone":"+*********5678"}`)
		assertMySQLPhoneStatusHTTP(t, router, unboundSession.AccessToken, http.StatusOK, `{"primary_phone_bound":false,"masked_phone":null}`)

		unknownToken := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x77}, tokenEntropyBytes))
		assertMySQLPhoneStatusHTTP(t, router, unknownToken, http.StatusUnauthorized, `{"error":{"code":"UNAUTHENTICATED","message":"authentication required"}}`)
		now = boundSession.ExpiresAt
		assertMySQLPhoneStatusHTTP(t, router, boundSession.AccessToken, http.StatusUnauthorized, `{"error":{"code":"UNAUTHENTICATED","message":"authentication required"}}`)
		now = boundSession.ExpiresAt.Add(-time.Hour)

		var schemaName string
		if err := db.QueryRowContext(context.Background(), "SELECT DATABASE()").Scan(&schemaName); err != nil {
			t.Fatal("read phone-status schema name failed")
		}
		closedConfig, ok := identityIntegrationConfig(t, schemaName)
		if !ok {
			t.Fatal("phone-status database environment disappeared")
		}
		closedDB, err := database.Open(closedConfig)
		if err != nil {
			t.Fatal("open phone-status failure database failed")
		}
		if err := closedDB.Close(); err != nil {
			t.Fatal("close phone-status failure database failed")
		}
		closedRepository := NewRepository(closedDB)
		closedProvider := &phoneStatusProviderSpy{}
		closedStore := &phoneStatusStoreSpy{repository: closedRepository}
		closedPhoneService := newPhoneService(closedProvider, closedStore, func() time.Time { return now })
		phoneUnavailableRouter, _ := phoneHandlerTestRouter(boundSessionService, closedPhoneService)
		assertMySQLPhoneStatusHTTP(t, phoneUnavailableRouter, boundSession.AccessToken, http.StatusServiceUnavailable, `{"error":{"code":"PRIMARY_PHONE_STATUS_UNAVAILABLE","message":"primary phone status temporarily unavailable"}}`)

		closedSessionService := newService(mysqlStaticExchanger{}, closedRepository, func() time.Time { return now }, bytes.NewReader(bytes.Repeat([]byte{0x88}, tokenEntropyBytes)))
		authUnavailableRouter, _ := phoneHandlerTestRouter(closedSessionService, phoneService)
		assertMySQLPhoneStatusHTTP(t, authUnavailableRouter, boundSession.AccessToken, http.StatusServiceUnavailable, `{"error":{"code":"PRIMARY_PHONE_STATUS_UNAVAILABLE","message":"primary phone status temporarily unavailable"}}`)

		after := readPhoneStatusDatabaseSnapshot(t, db)
		if !reflect.DeepEqual(after, before) {
			t.Fatal("primary-phone status reads changed user or session data")
		}
		if provider.calls != 0 || store.bindCalls != 0 || closedProvider.calls != 0 || closedStore.bindCalls != 0 {
			t.Fatal("primary-phone status reached provider or binding write")
		}
	})
}

func TestHistoricalMigrationPrefixRequiresExactVersion(t *testing.T) {
	missingRequiredVersion := make([]migrate.Migration, 9)
	if _, err := historicalMigrationPrefix(missingRequiredVersion, 10); err == nil {
		t.Fatal("historical migration prefix accepted a missing required version")
	}

	wrongRequiredVersion := make([]migrate.Migration, 10)
	wrongRequiredVersion[9].Version = 11
	if _, err := historicalMigrationPrefix(wrongRequiredVersion, 10); err == nil {
		t.Fatal("historical migration prefix accepted the wrong required version")
	}
}

func loadHistoricalMigrations(t *testing.T, requiredVersion uint64) []migrate.Migration {
	t.Helper()
	migrationSet, err := migrate.Load(migrations.FS)
	if err != nil {
		t.Fatal("load historical migrations failed")
	}
	prefix, err := historicalMigrationPrefix(migrationSet, requiredVersion)
	if err != nil {
		t.Fatal(err)
	}
	return prefix
}

func historicalMigrationPrefix(migrationSet []migrate.Migration, requiredVersion uint64) ([]migrate.Migration, error) {
	if uint64(len(migrationSet)) < requiredVersion {
		return nil, fmt.Errorf("required migration v%d is missing", requiredVersion)
	}
	if migrationSet[requiredVersion-1].Version != requiredVersion {
		return nil, fmt.Errorf("required migration index has version %d, want %d", migrationSet[requiredVersion-1].Version, requiredVersion)
	}
	return migrationSet[:requiredVersion], nil
}

type phoneStatusProviderSpy struct{ calls int }

func (provider *phoneStatusProviderSpy) Exchange(context.Context, string, string) (string, error) {
	provider.calls++
	return "", errors.New("phone-status provider must not be called")
}

type phoneStatusStoreSpy struct {
	repository *Repository
	bindCalls  int
}

func (store *phoneStatusStoreSpy) FindPhoneUser(ctx context.Context, userID uint64) (PhoneUser, error) {
	return store.repository.FindPhoneUser(ctx, userID)
}

func (store *phoneStatusStoreSpy) BindPrimaryPhone(context.Context, uint64, string, time.Time) (string, error) {
	store.bindCalls++
	return "", errors.New("phone-status bind must not be called")
}

type phoneStatusUserSnapshot struct {
	ID           uint64
	OpenID       string
	CreatedAt    time.Time
	LastLoginAt  time.Time
	PrimaryPhone sql.NullString
	BoundAt      sql.NullTime
}

type phoneStatusSessionSnapshot struct {
	TokenHash []byte
	UserID    uint64
	IssuedAt  time.Time
	ExpiresAt time.Time
}

type phoneStatusDatabaseSnapshot struct {
	Users    []phoneStatusUserSnapshot
	Sessions []phoneStatusSessionSnapshot
}

func readPhoneStatusDatabaseSnapshot(t *testing.T, db *sql.DB) phoneStatusDatabaseSnapshot {
	t.Helper()
	var snapshot phoneStatusDatabaseSnapshot
	userRows, err := db.QueryContext(context.Background(), "SELECT id,openid,created_at,last_login_at,primary_phone,primary_phone_bound_at FROM miniprogram_users ORDER BY id")
	if err != nil {
		t.Fatal("read phone-status user snapshot failed")
	}
	for userRows.Next() {
		var row phoneStatusUserSnapshot
		if err := userRows.Scan(&row.ID, &row.OpenID, &row.CreatedAt, &row.LastLoginAt, &row.PrimaryPhone, &row.BoundAt); err != nil {
			userRows.Close()
			t.Fatal("scan phone-status user snapshot failed")
		}
		snapshot.Users = append(snapshot.Users, row)
	}
	if err := userRows.Err(); err != nil {
		userRows.Close()
		t.Fatal("iterate phone-status user snapshot failed")
	}
	if err := userRows.Close(); err != nil {
		t.Fatal("close phone-status user snapshot failed")
	}
	sessionRows, err := db.QueryContext(context.Background(), "SELECT token_hash,user_id,issued_at,expires_at FROM miniprogram_sessions ORDER BY token_hash")
	if err != nil {
		t.Fatal("read phone-status session snapshot failed")
	}
	for sessionRows.Next() {
		var row phoneStatusSessionSnapshot
		if err := sessionRows.Scan(&row.TokenHash, &row.UserID, &row.IssuedAt, &row.ExpiresAt); err != nil {
			sessionRows.Close()
			t.Fatal("scan phone-status session snapshot failed")
		}
		snapshot.Sessions = append(snapshot.Sessions, row)
	}
	if err := sessionRows.Err(); err != nil {
		sessionRows.Close()
		t.Fatal("iterate phone-status session snapshot failed")
	}
	if err := sessionRows.Close(); err != nil {
		t.Fatal("close phone-status session snapshot failed")
	}
	return snapshot
}

func assertMySQLPhoneStatusHTTP(t *testing.T, handler http.Handler, token string, wantStatus int, wantBody string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/primary-phone", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != wantStatus {
		t.Fatalf("real MySQL primary-phone status code = %d, want %d", response.Code, wantStatus)
	}
	if strings.TrimSpace(response.Body.String()) != wantBody {
		t.Fatal("real MySQL primary-phone status body mismatch")
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("real MySQL primary-phone status response is cacheable")
	}
}

func assertFirstPhoneBind(t *testing.T, db *sql.DB) {
	t.Helper()
	userID := insertPhoneUser(t, db, "phone-first-openid")
	repository := NewRepository(db)
	boundAt := time.Date(2026, time.August, 20, 8, 0, 0, 123456000, time.UTC)
	const phone = "+8613712345678"

	user, err := repository.FindPhoneUser(context.Background(), userID)
	if err != nil || user.OpenID != "phone-first-openid" || user.PrimaryPhone != "" {
		t.Fatal("unbound phone user read mismatch")
	}
	if got, err := repository.BindPrimaryPhone(context.Background(), userID, phone, boundAt); err != nil || got != phone {
		t.Fatal("first primary phone bind failed")
	}
	later := boundAt.Add(time.Hour)
	if got, err := repository.BindPrimaryPhone(context.Background(), userID, phone, later); err != nil || got != phone {
		t.Fatal("same primary phone bind was not idempotent")
	}
	var storedPhone string
	var storedAt time.Time
	if err := db.QueryRowContext(context.Background(), "SELECT primary_phone,primary_phone_bound_at FROM miniprogram_users WHERE id=?", userID).Scan(&storedPhone, &storedAt); err != nil {
		t.Fatal("read first primary phone bind failed")
	}
	if storedPhone != phone || !storedAt.Equal(boundAt) {
		t.Fatal("same-phone retry changed primary phone or bound-at")
	}
}

func assertConcurrentDifferentPhoneSameUser(t *testing.T, db *sql.DB) {
	t.Helper()
	userID := insertPhoneUser(t, db, "phone-same-user-openid")
	repository := NewRepository(db)
	phones := []string{"+8613800001001", "+8613800001002"}
	times := []time.Time{
		time.Date(2026, time.August, 20, 9, 0, 0, 100000000, time.UTC),
		time.Date(2026, time.August, 20, 9, 0, 0, 200000000, time.UTC),
	}
	start := make(chan struct{})
	errorsFound := make(chan error, 2)
	for index := range phones {
		index := index
		go func() {
			<-start
			_, err := repository.BindPrimaryPhone(context.Background(), userID, phones[index], times[index])
			errorsFound <- err
		}()
	}
	close(start)
	results := []error{<-errorsFound, <-errorsFound}
	successes, conflicts := 0, 0
	for _, err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrPrimaryPhoneAlreadyBound):
			conflicts++
		default:
			t.Fatal("same-user concurrent bind returned unstable error")
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("same-user concurrent results = success %d, conflict %d", successes, conflicts)
	}
	var storedPhone string
	var storedAt time.Time
	if err := db.QueryRowContext(context.Background(), "SELECT primary_phone,primary_phone_bound_at FROM miniprogram_users WHERE id=?", userID).Scan(&storedPhone, &storedAt); err != nil {
		t.Fatal("read same-user concurrent bind failed")
	}
	winner := -1
	for index := range phones {
		if storedPhone == phones[index] {
			winner = index
		}
	}
	if winner < 0 || !storedAt.Equal(times[winner]) {
		t.Fatal("same-user concurrent bind did not preserve winner state")
	}
}

func assertConcurrentSamePhoneCrossUser(t *testing.T, db *sql.DB) {
	t.Helper()
	userIDs := []uint64{
		insertPhoneUser(t, db, "phone-cross-user-one-openid"),
		insertPhoneUser(t, db, "phone-cross-user-two-openid"),
	}
	repository := NewRepository(db)
	const phone = "+8613800002001"
	boundAt := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)
	start := make(chan struct{})
	errorsFound := make(chan error, 2)
	for _, userID := range userIDs {
		userID := userID
		go func() {
			<-start
			_, err := repository.BindPrimaryPhone(context.Background(), userID, phone, boundAt)
			errorsFound <- err
		}()
	}
	close(start)
	results := []error{<-errorsFound, <-errorsFound}
	successes, conflicts := 0, 0
	for _, err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrPhoneInUse):
			conflicts++
		default:
			t.Fatal("cross-user concurrent bind returned unstable error")
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("cross-user concurrent results = success %d, conflict %d", successes, conflicts)
	}
	var owners int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM miniprogram_users WHERE primary_phone=?", phone).Scan(&owners); err != nil || owners != 1 {
		t.Fatal("cross-user primary phone uniqueness mismatch")
	}
}

func assertPhoneTransactionRollback(t *testing.T, db *sql.DB) {
	t.Helper()
	boundAt := time.Date(2026, time.August, 20, 11, 0, 0, 0, time.UTC)
	statementUserID := insertPhoneUser(t, db, "phone-statement-rollback-openid")
	repository := NewRepository(db)
	if _, err := repository.BindPrimaryPhone(context.Background(), statementUserID, "+1234567890123456", boundAt); err == nil {
		t.Fatal("oversized phone statement unexpectedly succeeded")
	}
	assertPhoneUnbound(t, db, statementUserID)

	commitUserID := insertPhoneUser(t, db, "phone-commit-rollback-openid")
	commitRepository := newRepository(db, func(transaction *sql.Tx) { _ = transaction.Rollback() })
	if _, err := commitRepository.BindPrimaryPhone(context.Background(), commitUserID, "+8613800003001", boundAt); err == nil {
		t.Fatal("forced phone commit failure unexpectedly succeeded")
	}
	assertPhoneUnbound(t, db, commitUserID)
}

func assertSameCodePhoneRecovery(t *testing.T, db *sql.DB) {
	t.Helper()
	userID := insertPhoneUser(t, db, "phone-recovery-openid")
	repository := NewRepository(db)
	boundAt := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	provider := mysqlRejectAfterBindProvider{repository: repository, userID: userID, phone: "+8613800004001", boundAt: boundAt}
	service := newPhoneService(provider, repository, func() time.Time { return boundAt.Add(time.Hour) })

	result, err := service.Bind(context.Background(), userID, "single-use-code")
	if err != nil || result.MaskedPhone != "+*********4001" {
		t.Fatal("real repository same-code recovery failed")
	}
}

type mysqlRejectAfterBindProvider struct {
	repository *Repository
	userID     uint64
	phone      string
	boundAt    time.Time
}

func (provider mysqlRejectAfterBindProvider) Exchange(ctx context.Context, _ string, openID string) (string, error) {
	if openID == "" {
		return "", errors.New("missing provider identity")
	}
	if _, err := provider.repository.BindPrimaryPhone(ctx, provider.userID, provider.phone, provider.boundAt); err != nil {
		return "", err
	}
	return "", wechat.ErrPhoneCodeRejected
}

func insertPhoneUser(t *testing.T, db *sql.DB, openID string) uint64 {
	t.Helper()
	at := time.Date(2026, time.August, 20, 7, 30, 0, 0, time.UTC)
	result, err := db.ExecContext(context.Background(), "INSERT INTO miniprogram_users(openid,created_at,last_login_at) VALUES (?,?,?)", openID, at, at)
	if err != nil {
		t.Fatal("insert phone user failed")
	}
	userID, err := result.LastInsertId()
	if err != nil || userID <= 0 {
		t.Fatal("read inserted phone user id failed")
	}
	return uint64(userID)
}

func assertPhoneUnbound(t *testing.T, db *sql.DB, userID uint64) {
	t.Helper()
	var phone, boundAt sql.NullString
	if err := db.QueryRowContext(context.Background(), "SELECT primary_phone,primary_phone_bound_at FROM miniprogram_users WHERE id=?", userID).Scan(&phone, &boundAt); err != nil {
		t.Fatal("read rolled-back phone user failed")
	}
	if phone.Valid || boundAt.Valid {
		t.Fatal("failed phone transaction changed persistent state")
	}
}

func assertFirstLoginAndExpiry(t *testing.T, db *sql.DB) {
	t.Helper()
	clock := time.Date(2026, time.August, 20, 4, 5, 6, 123456000, time.UTC)
	repository := NewRepository(db)
	service := newService(mysqlStaticExchanger{openid: "first-login-openid"}, repository, func() time.Time { return clock }, bytes.NewReader(bytes.Repeat([]byte{0x11}, tokenEntropyBytes)))

	issued, err := service.Issue(context.Background(), "one-time-code")
	if err != nil {
		t.Fatal("first login failed")
	}
	wantHash := sha256.Sum256([]byte(issued.AccessToken))
	var userID uint64
	var openid string
	var createdAt, lastLoginAt time.Time
	if err := db.QueryRowContext(context.Background(), "SELECT id,openid,created_at,last_login_at FROM miniprogram_users WHERE openid=?", "first-login-openid").Scan(&userID, &openid, &createdAt, &lastLoginAt); err != nil {
		t.Fatal("read first login user failed")
	}
	if userID == 0 || openid != "first-login-openid" || !createdAt.Equal(clock) || !lastLoginAt.Equal(clock) {
		t.Fatal("first login user row is not exact")
	}
	var storedHash []byte
	var storedUserID uint64
	var issuedAt, expiresAt time.Time
	if err := db.QueryRowContext(context.Background(), "SELECT token_hash,user_id,issued_at,expires_at FROM miniprogram_sessions WHERE user_id=?", userID).Scan(&storedHash, &storedUserID, &issuedAt, &expiresAt); err != nil {
		t.Fatal("read first login session failed")
	}
	if !bytes.Equal(storedHash, wantHash[:]) || storedUserID != userID || !issuedAt.Equal(clock) || !expiresAt.Equal(clock.Add(sessionTTL)) || !issued.ExpiresAt.Equal(expiresAt) {
		t.Fatal("first login session row is not exact")
	}
	clock = issuedAt.Add(-time.Microsecond)
	if _, err := service.Authenticate(context.Background(), issued.AccessToken); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("session before issue error = %v", err)
	}
	clock = expiresAt.Add(-time.Microsecond)
	if got, err := service.Authenticate(context.Background(), issued.AccessToken); err != nil || got != userID {
		t.Fatalf("session before expiry = %d/%v", got, err)
	}
	clock = expiresAt
	if _, err := service.Authenticate(context.Background(), issued.AccessToken); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("session at expiry error = %v", err)
	}
}

func assertConcurrentLogin(t *testing.T, db *sql.DB) {
	t.Helper()
	repository := NewRepository(db)
	issuedAt := time.Date(2026, time.August, 20, 5, 0, 0, 0, time.UTC)
	start := make(chan struct{})
	errorsFound := make(chan error, 2)
	var wait sync.WaitGroup
	for index, entropyByte := range []byte{0x22, 0x33} {
		index, entropyByte := index, entropyByte
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			service := newService(mysqlStaticExchanger{openid: "concurrent-openid"}, repository, func() time.Time { return issuedAt.Add(time.Duration(index) * time.Microsecond) }, bytes.NewReader(bytes.Repeat([]byte{entropyByte}, tokenEntropyBytes)))
			_, err := service.Issue(context.Background(), "one-time-code")
			errorsFound <- err
		}()
	}
	close(start)
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		if err != nil {
			t.Fatal("concurrent login failed")
		}
	}

	var users, sessions int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM miniprogram_users WHERE openid=?", "concurrent-openid").Scan(&users); err != nil {
		t.Fatal("count concurrent users failed")
	}
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM miniprogram_sessions s JOIN miniprogram_users u ON u.id=s.user_id WHERE u.openid=?", "concurrent-openid").Scan(&sessions); err != nil {
		t.Fatal("count concurrent sessions failed")
	}
	if users != 1 || sessions != 2 {
		t.Fatalf("concurrent rows = users %d, sessions %d", users, sessions)
	}
}

func assertTransactionRollback(t *testing.T, db *sql.DB) {
	t.Helper()
	repository := NewRepository(db)
	firstLogin := time.Date(2026, time.August, 20, 6, 0, 0, 0, time.UTC)
	entropy := bytes.Repeat([]byte{0x44}, tokenEntropyBytes)
	seed := newService(mysqlStaticExchanger{openid: "rollback-existing-openid"}, repository, func() time.Time { return firstLogin }, bytes.NewReader(entropy))
	if _, err := seed.Issue(context.Background(), "one-time-code"); err != nil {
		t.Fatal("seed rollback session failed")
	}

	collision := newService(mysqlStaticExchanger{openid: "rollback-existing-openid"}, repository, func() time.Time { return firstLogin.Add(time.Hour) }, bytes.NewReader(entropy))
	if _, err := collision.Issue(context.Background(), "one-time-code"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("collision error = %v", err)
	}
	var lastLogin time.Time
	var existingSessions int
	if err := db.QueryRowContext(context.Background(), "SELECT last_login_at FROM miniprogram_users WHERE openid=?", "rollback-existing-openid").Scan(&lastLogin); err != nil {
		t.Fatal("read collision user failed")
	}
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM miniprogram_sessions s JOIN miniprogram_users u ON u.id=s.user_id WHERE u.openid=?", "rollback-existing-openid").Scan(&existingSessions); err != nil {
		t.Fatal("count collision sessions failed")
	}
	if !lastLogin.Equal(firstLogin) || existingSessions != 1 {
		t.Fatal("collision changed user or session state")
	}

	statementHash := sha256.Sum256([]byte("statement-rollback-token"))
	invalidExpiry := firstLogin.Add(2 * time.Hour)
	if err := repository.CreateSession(context.Background(), CreateSessionParams{OpenID: "statement-rollback-openid", TokenHash: statementHash, IssuedAt: invalidExpiry, ExpiresAt: invalidExpiry}); err == nil {
		t.Fatal("invalid expiry statement unexpectedly succeeded")
	}
	assertNoIdentityRows(t, db, "statement-rollback-openid")

	commitHash := sha256.Sum256([]byte("commit-rollback-token"))
	commitRepository := newRepository(db, func(transaction *sql.Tx) { _ = transaction.Rollback() })
	if err := commitRepository.CreateSession(context.Background(), CreateSessionParams{OpenID: "commit-rollback-openid", TokenHash: commitHash, IssuedAt: firstLogin, ExpiresAt: firstLogin.Add(sessionTTL)}); err == nil {
		t.Fatal("forced commit failure unexpectedly succeeded")
	}
	assertNoIdentityRows(t, db, "commit-rollback-openid")
}

func assertNoIdentityRows(t *testing.T, db *sql.DB, openid string) {
	t.Helper()
	var users, sessions int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM miniprogram_users WHERE openid=?", openid).Scan(&users); err != nil {
		t.Fatal("count rollback users failed")
	}
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM miniprogram_sessions s JOIN miniprogram_users u ON u.id=s.user_id WHERE u.openid=?", openid).Scan(&sessions); err != nil {
		t.Fatal("count rollback sessions failed")
	}
	if users != 0 || sessions != 0 {
		t.Fatalf("rollback rows = users %d, sessions %d", users, sessions)
	}
}

type mysqlStaticExchanger struct{ openid string }

func (exchanger mysqlStaticExchanger) Exchange(context.Context, string) (string, error) {
	return exchanger.openid, nil
}

type identitySchemaColumn struct {
	Name     string
	Type     string
	Nullable string
	Extra    string
}

type identitySchemaIndex struct {
	Name      string
	NonUnique bool
	Sequence  uint64
	Column    string
}

func assertIdentitySchema(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, table := range []string{"miniprogram_users", "miniprogram_sessions"} {
		var engine, collation string
		if err := db.QueryRowContext(context.Background(), "SELECT engine,table_collation FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?", table).Scan(&engine, &collation); err != nil {
			t.Fatalf("inspect %s table failed", table)
		}
		if engine != "InnoDB" || collation != "utf8mb4_0900_ai_ci" {
			t.Fatalf("%s engine/collation = %s/%s", table, engine, collation)
		}
	}

	wantColumns := map[string][]identitySchemaColumn{
		"miniprogram_users": {
			{Name: "id", Type: "bigint unsigned", Nullable: "NO", Extra: "auto_increment"},
			{Name: "openid", Type: "varbinary(128)", Nullable: "NO"},
			{Name: "created_at", Type: "timestamp(6)", Nullable: "NO"},
			{Name: "last_login_at", Type: "timestamp(6)", Nullable: "NO"},
			{Name: "primary_phone", Type: "varbinary(16)", Nullable: "YES"},
			{Name: "primary_phone_bound_at", Type: "timestamp(6)", Nullable: "YES"},
		},
		"miniprogram_sessions": {
			{Name: "token_hash", Type: "binary(32)", Nullable: "NO"},
			{Name: "user_id", Type: "bigint unsigned", Nullable: "NO"},
			{Name: "issued_at", Type: "timestamp(6)", Nullable: "NO"},
			{Name: "expires_at", Type: "timestamp(6)", Nullable: "NO"},
		},
	}
	for table, want := range wantColumns {
		rows, err := db.QueryContext(context.Background(), "SELECT column_name,column_type,is_nullable,extra FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? ORDER BY ordinal_position", table)
		if err != nil {
			t.Fatalf("read %s columns failed", table)
		}
		var got []identitySchemaColumn
		for rows.Next() {
			var column identitySchemaColumn
			if err := rows.Scan(&column.Name, &column.Type, &column.Nullable, &column.Extra); err != nil {
				rows.Close()
				t.Fatalf("scan %s columns failed", table)
			}
			got = append(got, column)
		}
		if err := rows.Close(); err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("%s columns = %#v, want %#v", table, got, want)
		}
	}

	wantIndexes := map[string][]identitySchemaIndex{
		"miniprogram_users": {
			{Name: "PRIMARY", Sequence: 1, Column: "id"},
			{Name: "uq_miniprogram_users_openid", Sequence: 1, Column: "openid"},
			{Name: "uq_miniprogram_users_primary_phone", Sequence: 1, Column: "primary_phone"},
		},
		"miniprogram_sessions": {
			{Name: "idx_miniprogram_sessions_user_expiry", NonUnique: true, Sequence: 1, Column: "user_id"},
			{Name: "idx_miniprogram_sessions_user_expiry", NonUnique: true, Sequence: 2, Column: "expires_at"},
			{Name: "PRIMARY", Sequence: 1, Column: "token_hash"},
		},
	}
	for table, want := range wantIndexes {
		rows, err := db.QueryContext(context.Background(), "SELECT index_name,non_unique,seq_in_index,column_name FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name=? ORDER BY index_name,seq_in_index", table)
		if err != nil {
			t.Fatalf("read %s indexes failed", table)
		}
		var got []identitySchemaIndex
		for rows.Next() {
			var index identitySchemaIndex
			if err := rows.Scan(&index.Name, &index.NonUnique, &index.Sequence, &index.Column); err != nil {
				rows.Close()
				t.Fatalf("scan %s indexes failed", table)
			}
			got = append(got, index)
		}
		if err := rows.Close(); err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("%s indexes = %#v, want %#v", table, got, want)
		}
	}

	var column, referencedTable, referencedColumn, updateRule, deleteRule string
	if err := db.QueryRowContext(context.Background(), `
		SELECT k.column_name,k.referenced_table_name,k.referenced_column_name,r.update_rule,r.delete_rule
		FROM information_schema.key_column_usage k
		JOIN information_schema.referential_constraints r
		  ON r.constraint_schema=k.constraint_schema AND r.constraint_name=k.constraint_name
		WHERE k.table_schema=DATABASE() AND k.table_name='miniprogram_sessions' AND k.constraint_name='fk_miniprogram_sessions_user'
	`).Scan(&column, &referencedTable, &referencedColumn, &updateRule, &deleteRule); err != nil {
		t.Fatal("inspect session user foreign key failed")
	}
	if column != "user_id" || referencedTable != "miniprogram_users" || referencedColumn != "id" || updateRule != "RESTRICT" || deleteRule != "RESTRICT" {
		t.Fatal("session user foreign key is not exact")
	}

	var checkClause string
	if err := db.QueryRowContext(context.Background(), "SELECT check_clause FROM information_schema.check_constraints WHERE constraint_schema=DATABASE() AND constraint_name='chk_miniprogram_sessions_expiry'").Scan(&checkClause); err != nil {
		t.Fatal("inspect session expiry check failed")
	}
	if checkClause != "(`expires_at` > `issued_at`)" {
		t.Fatalf("session expiry check = %q", checkClause)
	}

	if err := db.QueryRowContext(context.Background(), "SELECT check_clause FROM information_schema.check_constraints WHERE constraint_schema=DATABASE() AND constraint_name='chk_miniprogram_users_primary_phone_pair'").Scan(&checkClause); err != nil {
		t.Fatal("inspect primary phone pair check failed")
	}
	if checkClause != "(((`primary_phone` is null) and (`primary_phone_bound_at` is null)) or ((`primary_phone` is not null) and (`primary_phone_bound_at` is not null)))" {
		t.Fatalf("primary phone pair check = %q", checkClause)
	}
}

func withIdentitySchema(t *testing.T, run func(*sql.DB)) {
	t.Helper()
	serverConfig, ok := identityIntegrationConfig(t, "mysql")
	if !ok {
		t.Skip("identity MySQL integration environment not provided")
	}
	serverDB, err := database.Open(serverConfig)
	if err != nil {
		t.Fatal("open isolated MySQL server failed")
	}
	defer serverDB.Close()
	var version string
	if err := serverDB.QueryRowContext(context.Background(), "SELECT VERSION()").Scan(&version); err != nil || len(version) < 4 || version[:4] != "8.0." {
		t.Fatal("isolated database is not MySQL 8.0")
	}

	schemaName := randomIdentitySchemaName(t)
	if !identitySchemaPattern.MatchString(schemaName) {
		t.Fatal("generated schema name failed ownership validation")
	}
	if _, err := serverDB.ExecContext(context.Background(), "CREATE DATABASE `"+schemaName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal("create isolated identity schema failed")
	}
	defer func() {
		if !identitySchemaPattern.MatchString(schemaName) {
			t.Error("unsafe identity schema cleanup target")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := serverDB.ExecContext(ctx, "DROP DATABASE `"+schemaName+"`"); err != nil {
			t.Error("identity schema cleanup failed")
		}
	}()

	databaseConfig, _ := identityIntegrationConfig(t, schemaName)
	db, err := database.Open(databaseConfig)
	if err != nil {
		t.Fatal("open isolated identity schema failed")
	}
	defer db.Close()
	run(db)
}

func identityIntegrationConfig(t *testing.T, databaseName string) (database.ConnectionConfig, bool) {
	t.Helper()
	keys := []string{"ORDER_TEST_MYSQL_HOST", "ORDER_TEST_MYSQL_PORT", "ORDER_TEST_MYSQL_USER", "ORDER_TEST_MYSQL_PASSWORD", "ORDER_TEST_MYSQL_TLS_MODE", "ORDER_TEST_MYSQL_INSTANCE", "ORDER_TEST_MYSQL_ISOLATED"}
	present := 0
	for _, key := range keys {
		if _, ok := os.LookupEnv(key); ok {
			present++
		}
	}
	if present == 0 {
		return database.ConnectionConfig{}, false
	}
	if present != len(keys) || os.Getenv("ORDER_TEST_MYSQL_INSTANCE") != "order-mysql-w3" || os.Getenv("ORDER_TEST_MYSQL_ISOLATED") != "YES" {
		t.Fatal("identity integration environment is incomplete or not owned")
	}
	port, err := strconv.ParseUint(os.Getenv("ORDER_TEST_MYSQL_PORT"), 10, 16)
	if err != nil || port == 0 {
		t.Fatal("identity integration port is invalid")
	}
	return database.ConnectionConfig{
		Host: os.Getenv("ORDER_TEST_MYSQL_HOST"), Port: uint16(port), Database: databaseName,
		User: os.Getenv("ORDER_TEST_MYSQL_USER"), Password: os.Getenv("ORDER_TEST_MYSQL_PASSWORD"), TLSMode: os.Getenv("ORDER_TEST_MYSQL_TLS_MODE"),
	}, true
}

func randomIdentitySchemaName(t *testing.T) string {
	t.Helper()
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		t.Fatal("generate random identity schema name failed")
	}
	return fmt.Sprintf("order_identity_test_%s", hex.EncodeToString(value))
}
