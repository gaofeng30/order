package migrate

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"time"
)

// Stable signed bigint namespace; the lock is scoped to the current database.
const postgresMigrationLockID int64 = 0x6f726465725f6d69

// Run applies each PostgreSQL migration and its history atomically on one locked session.
func Run(ctx context.Context, db *sql.DB, migrations []Migration) (result Result, returnErr error) {
	// Migration is public: callers cannot bypass the loader's boundary checks.
	for index, migration := range migrations {
		if migration.Version != uint64(index+1) || !validPostgreSQLMigrationBytes(migration.SQL) || sha256.Sum256(migration.SQL) != migration.Checksum {
			return result, migrateError{reason: ReasonMigrationFailed, version: migration.Version}
		}
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return result, migrateError{reason: ReasonDatabaseUnreachable}
	}
	defer conn.Close()
	if err := validatePostgreSQLSession(ctx, conn); err != nil {
		return result, err
	}
	// Retain the original bounded concurrent-runner serialization contract.
	lockCtx, cancelLock := context.WithTimeout(ctx, 30*time.Second)
	var locked bool
	err = conn.QueryRowContext(lockCtx, "SELECT true FROM pg_advisory_lock($1)", postgresMigrationLockID).Scan(&locked)
	cancelLock()
	if err != nil {
		// The server may have acquired the lock before we lost its result.
		// Returning this healthy-but-uncertain physical session is unsafe.
		_ = conn.Raw(func(any) error { return driver.ErrBadConn })
		return result, migrateError{reason: ReasonMigrationLockUnavailable}
	}
	if !locked {
		return result, migrateError{reason: ReasonMigrationLockUnavailable}
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		var released bool
		if err := conn.QueryRowContext(releaseCtx, "SELECT pg_advisory_unlock($1)", postgresMigrationLockID).Scan(&released); err != nil || !released {
			// A failed/unconfirmed unlock must never return this session to the pool.
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
			if returnErr == nil {
				returnErr = migrateError{reason: ReasonMigrationFailed}
			}
		}
	}()
	exists, rows, err := postgreSQLHistory(ctx, conn)
	if err != nil {
		return result, err
	}
	state := classifyHistory(exists, rows, migrations)
	if state.Reason != "" && state.Reason != ReasonSchemaUninitialized && state.Reason != ReasonSchemaBehind {
		return result, migrateError{reason: state.Reason, version: state.CurrentVersion}
	}
	result.FromVersion, result.ToVersion = state.CurrentVersion, state.CurrentVersion
	for _, migration := range migrations[len(rows):] {
		if err := applyPostgreSQLMigration(ctx, conn, migration); err != nil {
			return result, err
		}
		result.AppliedCount++
		result.ToVersion = migration.Version
	}
	return result, nil
}

// Check verifies PostgreSQL readiness without changing schema or business data.
func Check(ctx context.Context, db *sql.DB, migrations []Migration) State {
	conn, err := db.Conn(ctx)
	if err != nil {
		return State{Reason: ReasonDatabaseUnreachable}
	}
	defer conn.Close()
	if err := validatePostgreSQLSession(ctx, conn); err != nil {
		return State{Reason: Reason(err)}
	}
	tx, err := conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return State{Reason: ReasonDatabaseUnreachable}
	}
	defer tx.Rollback()
	exists, rows, err := postgreSQLHistory(ctx, tx)
	if err != nil {
		return State{Reason: Reason(err)}
	}
	state := classifyHistory(exists, rows, migrations)
	if state.Ready {
		if err := verifyPostgreSQLBusinessSchema(ctx, tx, migrations); err != nil {
			return State{Reason: Reason(err), CurrentVersion: state.CurrentVersion}
		}
	}
	return state
}

type postgresQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func validatePostgreSQLSession(ctx context.Context, q postgresQueryer) error {
	var version int
	var zone, encoding, readOnly, path, standardStrings string
	if err := q.QueryRowContext(ctx, "SELECT current_setting('server_version_num')::integer, current_setting('TimeZone'), current_setting('client_encoding'), current_setting('transaction_read_only'), current_setting('search_path'), current_setting('standard_conforming_strings')").Scan(&version, &zone, &encoding, &readOnly, &path, &standardStrings); err != nil {
		return migrateError{reason: ReasonDatabaseUnreachable}
	}
	if version < 160000 || version >= 180000 || zone != "UTC" || encoding != "UTF8" || readOnly != "off" || path != "public" || standardStrings != "on" {
		return migrateError{reason: ReasonDatabaseIncompatible}
	}
	return nil
}

func postgreSQLHistory(ctx context.Context, q postgresQueryer) (bool, []historyRow, error) {
	var exists bool
	if err := q.QueryRowContext(ctx, "SELECT to_regclass('public.schema_migrations') IS NOT NULL").Scan(&exists); err != nil {
		return false, nil, migrateError{reason: ReasonDatabaseUnreachable}
	}
	if !exists {
		return false, nil, nil
	}
	if err := verifyPostgreSQLHistoryShape(ctx, q); err != nil {
		return true, nil, err
	}
	rows, err := q.QueryContext(ctx, "SELECT version,name,checksum,dirty FROM public.schema_migrations ORDER BY version")
	if err != nil {
		return true, nil, migrateError{reason: ReasonDatabaseUnreachable}
	}
	defer rows.Close()
	result := make([]historyRow, 0)
	for rows.Next() {
		var row historyRow
		var checksum []byte
		if err := rows.Scan(&row.Version, &row.Name, &checksum, &row.Dirty); err != nil || len(checksum) != 32 {
			return true, nil, migrateError{reason: ReasonDatabaseIncompatible}
		}
		copy(row.Checksum[:], checksum)
		result = append(result, row)
	}
	if rows.Err() != nil {
		return true, nil, migrateError{reason: ReasonDatabaseUnreachable}
	}
	return true, result, nil
}

func verifyPostgreSQLHistoryShape(ctx context.Context, q postgresQueryer) error {
	rows, err := q.QueryContext(ctx, "SELECT column_name,data_type,is_nullable,COALESCE(character_maximum_length,0) FROM information_schema.columns WHERE table_schema='public' AND table_name='schema_migrations' ORDER BY ordinal_position")
	if err != nil {
		return migrateError{reason: ReasonDatabaseUnreachable}
	}
	type column struct {
		name, dataType, nullable string
		length                   int
	}
	want := []column{{"version", "bigint", "NO", 0}, {"name", "character varying", "NO", 255}, {"checksum", "bytea", "NO", 0}, {"dirty", "boolean", "NO", 0}, {"applied_at", "timestamp with time zone", "YES", 0}}
	index := 0
	for rows.Next() {
		var got column
		if index >= len(want) || rows.Scan(&got.name, &got.dataType, &got.nullable, &got.length) != nil || got != want[index] {
			rows.Close()
			return migrateError{reason: ReasonDatabaseIncompatible}
		}
		index++
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil || index != len(want) {
		return migrateError{reason: ReasonDatabaseIncompatible}
	}
	var primary bool
	if err := q.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.schema_migrations'::regclass AND contype='p' AND conkey=ARRAY[1]::smallint[])").Scan(&primary); err != nil || !primary {
		return migrateError{reason: ReasonDatabaseIncompatible}
	}
	return nil
}

func applyPostgreSQLMigration(ctx context.Context, conn *sql.Conn, migration Migration) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return migrateError{reason: ReasonMigrationFailed, version: migration.Version}
	}
	defer tx.Rollback()
	if migration.Version == 20 || migration.Version == 23 {
		table := "categories"
		if migration.Version == 23 {
			table = "products"
		}
		if _, err := tx.ExecContext(ctx, "LOCK TABLE "+table+" IN EXCLUSIVE MODE"); err != nil {
			return migrateError{reason: ReasonMigrationPreflightFailed, version: migration.Version}
		}
		rows, err := tx.QueryContext(ctx, "SELECT id,name FROM "+table+" ORDER BY id")
		if err != nil {
			return migrateError{reason: ReasonMigrationPreflightFailed, version: migration.Version}
		}
		names := make([]legacyCatalogName, 0)
		for rows.Next() {
			var row legacyCatalogName
			if rows.Scan(&row.id, &row.name) != nil {
				rows.Close()
				return migrateError{reason: ReasonMigrationPreflightFailed, version: migration.Version}
			}
			names = append(names, row)
		}
		rowsErr := rows.Err()
		rows.Close()
		if rowsErr != nil || validateLegacyCatalogNames(names) != nil {
			return migrateError{reason: ReasonMigrationPreflightFailed, version: migration.Version}
		}
	}
	if _, err := tx.ExecContext(ctx, string(migration.SQL)); err != nil {
		return migrateError{reason: ReasonStatementFailed, version: migration.Version}
	}
	if err := validatePostgreSQLSession(ctx, tx); err != nil {
		return err
	}
	if migration.Version == 1 {
		if err := verifyPostgreSQLHistoryShape(ctx, tx); err != nil {
			return migrateError{reason: ReasonDatabaseIncompatible, version: 1}
		}
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO public.schema_migrations(version,name,checksum,dirty,applied_at) VALUES($1,$2,$3,FALSE,CURRENT_TIMESTAMP)", migration.Version, migration.Name, migration.Checksum[:]); err != nil {
		return migrateError{reason: ReasonMigrationFailed, version: migration.Version}
	}
	if err := tx.Commit(); err != nil {
		return migrateError{reason: ReasonMigrationFailed, version: migration.Version}
	}
	return nil
}
