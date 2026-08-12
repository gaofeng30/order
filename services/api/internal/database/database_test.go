package database

import (
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

const databaseCanarySecret = "database-canary-secret-must-not-leak"

func TestDriverConfigIsStructuredAndFrozen(t *testing.T) {
	cfg := validConnectionConfig()
	driverConfig, err := newDriverConfig(cfg)
	if err != nil {
		t.Fatalf("newDriverConfig() error = %v", err)
	}

	if driverConfig.Net != "tcp" || driverConfig.Addr != "127.0.0.1:3306" {
		t.Fatalf("network = %q/%q, want tcp/127.0.0.1:3306", driverConfig.Net, driverConfig.Addr)
	}
	if driverConfig.User != cfg.User || driverConfig.Passwd != cfg.Password || driverConfig.DBName != cfg.Database {
		t.Fatal("structured identity fields were not preserved")
	}
	if !driverConfig.ParseTime || driverConfig.Loc != time.UTC {
		t.Fatalf("time config = ParseTime:%v Loc:%v, want true/UTC", driverConfig.ParseTime, driverConfig.Loc)
	}
	if driverConfig.Collation != "utf8mb4_0900_ai_ci" || driverConfig.Params["time_zone"] != "'+00:00'" || !strings.Contains(driverConfig.FormatDSN(), "charset=utf8mb4") {
		t.Fatalf("session config = collation:%q params:%v", driverConfig.Collation, driverConfig.Params)
	}
	if driverConfig.Timeout != 3*time.Second || driverConfig.ReadTimeout != 5*time.Second || driverConfig.WriteTimeout != 5*time.Second {
		t.Fatalf("timeouts = %s/%s/%s", driverConfig.Timeout, driverConfig.ReadTimeout, driverConfig.WriteTimeout)
	}
	if driverConfig.TLSConfig != "" {
		t.Fatalf("disabled TLS config = %q, want empty", driverConfig.TLSConfig)
	}
	if driverConfig.AllowAllFiles || driverConfig.AllowCleartextPasswords || driverConfig.AllowFallbackToPlaintext || driverConfig.AllowOldPasswords || driverConfig.InterpolateParams || driverConfig.MultiStatements {
		t.Fatal("unsafe driver option enabled")
	}
	if _, ok := driverConfig.Logger.(*mysql.NopLogger); !ok {
		t.Fatalf("logger = %#v, want mysql.NopLogger", driverConfig.Logger)
	}
}

func TestDriverConfigRequiresVerifiedTLSWhenRequested(t *testing.T) {
	cfg := validConnectionConfig()
	cfg.TLSMode = "required"

	driverConfig, err := newDriverConfig(cfg)
	if err != nil {
		t.Fatalf("newDriverConfig() error = %v", err)
	}
	if driverConfig.TLSConfig != "true" {
		t.Fatalf("TLSConfig = %q, want verified TLS", driverConfig.TLSConfig)
	}
}

func TestOpenBuildsBoundedLazyPool(t *testing.T) {
	cfg := validConnectionConfig()
	cfg.Port = 1

	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	stats := db.Stats()
	if stats.OpenConnections != 0 {
		t.Fatalf("OpenConnections = %d, want lazy zero", stats.OpenConnections)
	}
	if stats.MaxOpenConnections != 10 {
		t.Fatalf("MaxOpenConnections = %d, want 10", stats.MaxOpenConnections)
	}
	if poolMaxIdle != 10 || poolMaxLifetime != 3*time.Minute || poolMaxIdleTime != time.Minute {
		t.Fatalf("pool settings = %d/%s/%s", poolMaxIdle, poolMaxLifetime, poolMaxIdleTime)
	}
}

func TestOpenRejectsInvalidFieldsWithoutLeakingValues(t *testing.T) {
	tests := []struct {
		name  string
		alter func(*ConnectionConfig)
		field string
	}{
		{name: "host", alter: func(cfg *ConnectionConfig) { cfg.Host = "mysql://" + databaseCanarySecret }, field: "host"},
		{name: "port", alter: func(cfg *ConnectionConfig) { cfg.Port = 0 }, field: "port"},
		{name: "database", alter: func(cfg *ConnectionConfig) { cfg.Database = "" }, field: "database"},
		{name: "user", alter: func(cfg *ConnectionConfig) { cfg.User = "" }, field: "user"},
		{name: "password", alter: func(cfg *ConnectionConfig) { cfg.Password = "" }, field: "password"},
		{name: "tls", alter: func(cfg *ConnectionConfig) { cfg.TLSMode = "skip-verify-" + databaseCanarySecret }, field: "tls_mode"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validConnectionConfig()
			test.alter(&cfg)

			_, err := Open(cfg)
			if err == nil {
				t.Fatal("Open() error = nil, want rejection")
			}
			if Reason(err) != "invalid_connection_field" || !strings.Contains(err.Error(), test.field) {
				t.Fatalf("error = %v reason=%q, want named safe field error", err, Reason(err))
			}
			for _, forbidden := range []string{databaseCanarySecret, cfg.Password, "mysql://"} {
				if forbidden != "" && strings.Contains(err.Error(), forbidden) {
					t.Fatalf("error leaked %q: %v", forbidden, err)
				}
			}
		})
	}
}

func validConnectionConfig() ConnectionConfig {
	return ConnectionConfig{
		Host:     "127.0.0.1",
		Port:     3306,
		Database: "order_test",
		User:     "order_test",
		Password: databaseCanarySecret,
		TLSMode:  "disabled",
	}
}
