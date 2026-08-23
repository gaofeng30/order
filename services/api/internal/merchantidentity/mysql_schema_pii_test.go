package merchantidentity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gin-gonic/gin"
)

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

	if err := insertMerchantAuditFixture(ctx, db, accountID, RoleSubaccount, 1, userB, ActionOrderRead, "SUCCEEDED", "AUTHORIZED", "order", 91, "internal-request-c", now); err != nil {
		t.Fatal("insert resolved retention audit failed")
	}
	partialScopeHash := sha256.Sum256([]byte("partial-snapshot"))
	if _, err := db.ExecContext(ctx, `
		INSERT INTO action_audits(entry_kind,actor_kind,actor_scope_hash,actor_account_id_snapshot,actor_user_id,action,result,reason_code,request_id_hash,occurred_at)
		VALUES ('LEGACY_EVIDENCE','MERCHANT',?,?,?,?,?,?,?,?)
	`, partialScopeHash[:], accountID, userB, ActionOrderRead, "REJECTED", "INVALID", merchantRequestIDHash("internal-request-d"), now); err == nil {
		t.Fatal("audit schema accepted a partial account snapshot")
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM merchant_accounts WHERE id=?", accountID); err == nil {
		t.Fatal("current audit foreign key allowed hard deletion")
	}
	if _, err := db.ExecContext(ctx, `UPDATE merchant_accounts SET enabled=FALSE,deleted_at=?,deleted_by_account_id=?,record_version=record_version+1,auth_version=auth_version+1,updated_at=? WHERE id=?`, now.Add(time.Minute), firstBoundID, now.Add(time.Minute), accountID); err != nil {
		t.Fatal("soft delete of audited account failed")
	}
	var liveAccount uint64
	var snapshotID, snapshotAuth uint64
	var snapshotRole Role
	var action, result, reason, targetType string
	var targetID, actorID uint64
	var occurredAt time.Time
	if err := db.QueryRowContext(ctx, `
		SELECT actor_account_id,actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,
		       actor_user_id,action,result,reason_code,target_type,target_id,occurred_at
		FROM action_audits WHERE request_id_hash=?
	`, merchantRequestIDHash("internal-request-c")).Scan(&liveAccount, &snapshotID, &snapshotRole, &snapshotAuth, &actorID, &action, &result, &reason, &targetType, &targetID, &occurredAt); err != nil {
		t.Fatal("read retained audit after soft delete failed")
	}
	if liveAccount != accountID || snapshotID != accountID || snapshotRole != RoleSubaccount || snapshotAuth != 1 || actorID != userB || action != string(ActionOrderRead) || result != "SUCCEEDED" || reason != "AUTHORIZED" || targetType != "order" || targetID != 91 || !occurredAt.Equal(now) {
		t.Fatal("soft delete did not retain the complete non-PII audit snapshot")
	}
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
	if identityResponse.Code != http.StatusOK || identityResponse.Body.String() != `{"identity":{"primary_phone":{"bound":true,"masked_phone":"***"},"extra_phone":{"set":false},"pricing_identity":{"kind":"VISITOR","rate_percent":100},"merchant":{"bound":true,"role":"OWNER"}}}` {
		t.Fatalf("real-session identity response = %d %s", identityResponse.Code, identityResponse.Body.String())
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
		SELECT CONCAT_WS('|',actor_account_id,actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,
		       actor_user_id,action,result,reason_code,target_type,target_id,HEX(request_id_hash),before_state_json,after_state_json,occurred_at)
		FROM action_audits
		WHERE actor_account_id=? AND request_id_hash=?
	`, accountID, merchantRequestIDHash("internal-pii-request")).Scan(&auditText); err != nil {
		t.Fatal("read PII boundary audit failed")
	}
	for _, canary := range []string{providerSubjectCanary, accountNameCanary, "+81", providerCodeCanary, sessionToken} {
		if strings.Contains(auditText, canary) {
			t.Fatal("durable merchant audit contained a PII canary")
		}
	}
	assertMerchantLoginAuditTarget(t, db, "internal-pii-request", uint64(accountID))
	var storedCodeHash []byte
	if err := db.QueryRowContext(ctx, `
		SELECT target_key_hash
		FROM action_audits
		WHERE actor_account_id=? AND request_id_hash=?
	`, accountID, merchantRequestIDHash("internal-pii-request")).Scan(&storedCodeHash); err != nil {
		t.Fatal("read merchant login code hash failed")
	}
	wantCodeHash := hashLoginCode(providerCodeCanary)
	if !bytes.Equal(storedCodeHash, wantCodeHash[:]) || bytes.Contains(storedCodeHash, []byte(providerCodeCanary)) {
		t.Fatal("merchant login audit did not retain only the domain-separated code hash")
	}
}

type unusedSessionExchanger struct{}

func (unusedSessionExchanger) Exchange(context.Context, string) (string, error) {
	return "", errors.New("session issuance is outside this test")
}
