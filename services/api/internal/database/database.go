package database

import (
	"database/sql"
	"fmt"
	"time"
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

// Open constructs a bounded lazy PostgreSQL pool without touching the network.
func Open(cfg ConnectionConfig) (*sql.DB, error) { return openPostgreSQL(cfg) }
