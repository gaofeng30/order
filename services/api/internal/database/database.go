package database

import (
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

const (
	poolMaxOpen     = 10
	poolMaxIdle     = 10
	poolMaxLifetime = 3 * time.Minute
	poolMaxIdleTime = time.Minute
)

// ConnectionConfig is an in-memory structured connection description.
type ConnectionConfig struct {
	Host     string
	Port     uint16
	Database string
	User     string
	Password string
	TLSMode  string
}

type connectionError struct {
	reason string
	field  string
}

func (err connectionError) Error() string {
	if err.field == "" {
		return err.reason
	}
	return fmt.Sprintf("%s: %s", err.reason, err.field)
}

// Reason returns a stable non-sensitive connection failure category.
func Reason(err error) string {
	if value, ok := err.(connectionError); ok {
		return value.reason
	}
	return "database_unavailable"
}

// Open constructs a bounded lazy MySQL pool without touching the network.
func Open(cfg ConnectionConfig) (*sql.DB, error) {
	driverConfig, err := newDriverConfig(cfg)
	if err != nil {
		return nil, err
	}
	connector, err := mysql.NewConnector(driverConfig)
	if err != nil {
		return nil, connectionError{reason: "connector_invalid"}
	}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(poolMaxOpen)
	db.SetMaxIdleConns(poolMaxIdle)
	db.SetConnMaxLifetime(poolMaxLifetime)
	db.SetConnMaxIdleTime(poolMaxIdleTime)
	return db, nil
}

func newDriverConfig(cfg ConnectionConfig) (*mysql.Config, error) {
	if cfg.Host == "" || strings.ContainsAny(cfg.Host, "@/") || strings.Contains(cfg.Host, "://") {
		return nil, connectionError{reason: "invalid_connection_field", field: "host"}
	}
	if cfg.Port == 0 {
		return nil, connectionError{reason: "invalid_connection_field", field: "port"}
	}
	if cfg.Database == "" {
		return nil, connectionError{reason: "invalid_connection_field", field: "database"}
	}
	if cfg.User == "" {
		return nil, connectionError{reason: "invalid_connection_field", field: "user"}
	}
	if cfg.Password == "" {
		return nil, connectionError{reason: "invalid_connection_field", field: "password"}
	}
	if cfg.TLSMode != "required" && cfg.TLSMode != "disabled" {
		return nil, connectionError{reason: "invalid_connection_field", field: "tls_mode"}
	}

	driverConfig := mysql.NewConfig()
	if err := driverConfig.Apply(mysql.Charset("utf8mb4", "utf8mb4_0900_ai_ci")); err != nil {
		return nil, connectionError{reason: "connector_invalid"}
	}
	driverConfig.Net = "tcp"
	driverConfig.Addr = net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port)))
	driverConfig.User = cfg.User
	driverConfig.Passwd = cfg.Password
	driverConfig.DBName = cfg.Database
	driverConfig.ParseTime = true
	driverConfig.Loc = time.UTC
	driverConfig.Params = map[string]string{"time_zone": "'+00:00'"}
	driverConfig.Timeout = 3 * time.Second
	driverConfig.ReadTimeout = 5 * time.Second
	driverConfig.WriteTimeout = 5 * time.Second
	driverConfig.Logger = &mysql.NopLogger{}
	if cfg.TLSMode == "required" {
		driverConfig.TLSConfig = "true"
	}
	return driverConfig, nil
}
