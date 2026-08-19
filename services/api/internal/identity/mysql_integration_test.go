package identity

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/migrations"
)

var identitySchemaPattern = regexp.MustCompile(`^order_identity_test_[0-9a-f]{32}$`)

func TestMiniprogramSessionMySQLIntegration(t *testing.T) {
	withIdentitySchema(t, func(db *sql.DB) {
		migrationSet, err := migrate.Load(migrations.FS)
		if err != nil {
			t.Fatal("load migrations failed")
		}
		if len(migrationSet) != 9 {
			t.Fatalf("migration count = %d, want 9", len(migrationSet))
		}

		foundation, err := migrate.Run(context.Background(), db, migrationSet[:1])
		if err != nil || foundation.AppliedCount != 1 || foundation.ToVersion != 1 {
			t.Fatal("foundation migration did not establish version 1")
		}
		identityResult, err := migrate.Run(context.Background(), db, migrationSet)
		if err != nil || identityResult.FromVersion != 1 || identityResult.ToVersion != 9 || identityResult.AppliedCount != 8 {
			t.Fatal("identity migrations did not advance version 1 to version 9")
		}
		repeat, err := migrate.Run(context.Background(), db, migrationSet)
		if err != nil || repeat.FromVersion != 9 || repeat.ToVersion != 9 || repeat.AppliedCount != 0 {
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
