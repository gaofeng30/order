package main

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/gaofeng30/order/services/api/internal/config"
	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/migrations"
)

type commandFunc func(context.Context) (migrate.Result, error)

type commandError struct {
	reason  string
	version uint64
	cause   error
}

func (err commandError) Error() string { return err.reason }

func main() {
	os.Exit(execute(os.Args[1:], os.Stdout, os.Stderr, runMigrationCommand, time.Now))
}

func execute(args []string, stdout, stderr io.Writer, command commandFunc, now func() time.Time) int {
	if len(args) != 0 {
		_, _ = io.WriteString(stderr, "usage: order-migrate\n")
		return 2
	}
	started := now()
	result, err := command(context.Background())
	duration := now().Sub(started).Milliseconds()
	if err != nil {
		reason := "migration_failed"
		version := result.ToVersion
		if safe, ok := err.(commandError); ok {
			reason = safe.reason
			version = safe.version
		}
		logger := slog.New(slog.NewJSONHandler(stderr, nil))
		logger.Error("migration failed", "event", "migration_failed", "reason", reason, "version", version, "duration_ms", duration)
		return 1
	}
	logger := slog.New(slog.NewJSONHandler(stderr, nil))
	logger.Info("migration completed", "event", "migration_complete", "from_version", result.FromVersion, "to_version", result.ToVersion, "applied_count", result.AppliedCount, "duration_ms", duration)
	return 0
}

func runMigrationCommand(ctx context.Context) (migrate.Result, error) {
	cfg, err := config.Load()
	if err != nil {
		return migrate.Result{}, commandError{reason: config.Reason(err), cause: err}
	}
	db, err := database.Open(cfg.Database)
	if err != nil {
		return migrate.Result{}, commandError{reason: database.Reason(err), cause: err}
	}
	defer closeDatabase(db)
	set, err := migrate.Load(migrations.FS)
	if err != nil {
		return migrate.Result{}, commandError{reason: migrate.Reason(err), cause: err}
	}
	result, err := migrate.Run(ctx, db, set)
	if err != nil {
		return result, commandError{reason: migrate.Reason(err), version: migrate.Version(err), cause: err}
	}
	return result, nil
}

func closeDatabase(db *sql.DB) { _ = db.Close() }
