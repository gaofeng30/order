package database

import (
	"crypto/tls"
	"database/sql"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// openPostgreSQL supplies the single native runtime pool.
func openPostgreSQL(cfg ConnectionConfig) (*sql.DB, error) {
	c, err := newPostgreSQLConfig(cfg)
	if err != nil {
		return nil, err
	}
	db := stdlib.OpenDB(*c)
	db.SetMaxOpenConns(poolMaxOpen)
	db.SetMaxIdleConns(poolMaxIdle)
	db.SetConnMaxLifetime(poolMaxLifetime)
	db.SetConnMaxIdleTime(poolMaxIdleTime)
	return db, nil
}

func newPostgreSQLConfig(cfg ConnectionConfig) (*pgx.ConnConfig, error) {
	if cfg.Host == "" || strings.ContainsAny(cfg.Host, "@/, \t\r\n") || strings.Contains(cfg.Host, "://") {
		return nil, connectionError{reason: "invalid_connection_field", field: "host"}
	}
	if cfg.Port == 0 {
		return nil, connectionError{reason: "invalid_connection_field", field: "port"}
	}
	for _, field := range []struct{ name, value string }{
		{"database", cfg.Database}, {"user", cfg.User}, {"password", cfg.Password},
	} {
		if field.value == "" {
			return nil, connectionError{reason: "invalid_connection_field", field: field.name}
		}
	}
	if cfg.TLSMode != "required" && cfg.TLSMode != "disabled" {
		return nil, connectionError{reason: "invalid_connection_field", field: "tls_mode"}
	}
	// ParseConfig consumes implicit PG settings/service/pass files. Reject this
	// alternate source before parsing, rather than select a second credential.
	for _, name := range []string{
		"PGHOST", "PGPORT", "PGDATABASE", "PGUSER", "PGPASSWORD", "PGPASSFILE",
		"PGSERVICE", "PGSERVICEFILE", "PGSSLMODE", "PGSSLCERT", "PGSSLKEY",
		"PGSSLROOTCERT", "PGSSLPASSWORD", "PGOPTIONS", "PGAPPNAME", "PGCONNECT_TIMEOUT",
		"PGTARGETSESSIONATTRS", "PGTZ", "PGMINPROTOCOLVERSION", "PGMAXPROTOCOLVERSION",
		"PGSSLSNI", "PGSSLNEGOTIATION",
	} {
		if os.Getenv(name) != "" {
			return nil, connectionError{reason: "ambient_connection_environment_forbidden"}
		}
	}
	// No user values enter the parser or its errors; a nonempty bootstrap
	// password prevents reading the implicit default .pgpass file.
	c, err := pgx.ParseConfig("host=127.0.0.1 port=1 user=order password=unused dbname=order sslmode=disable connect_timeout=3 passfile=/dev/null")
	if err != nil {
		return nil, connectionError{reason: "connector_invalid"}
	}
	c.Host, c.Port, c.Database, c.User, c.Password = cfg.Host, cfg.Port, cfg.Database, cfg.User, cfg.Password
	c.ConnectTimeout = 3 * time.Second
	c.RuntimeParams = map[string]string{
		"timezone": "UTC", "client_encoding": "UTF8", "application_name": "order",
		"statement_timeout": "5000", "lock_timeout": "3000", "idle_in_transaction_session_timeout": "5000",
		"search_path": "public", "standard_conforming_strings": "on",
	}
	c.Fallbacks = nil
	if cfg.TLSMode == "required" {
		c.TLSConfig = &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}
	}
	return c, nil
}
