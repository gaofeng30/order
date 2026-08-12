package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
)

const migrationLockName = "order_schema_migrate"

// Result is the sanitized migration outcome.
type Result struct {
	FromVersion  uint64
	ToVersion    uint64
	AppliedCount int
}

type migrateError struct {
	reason  string
	version uint64
}

func (err migrateError) Error() string { return err.reason }

// Reason returns a stable non-sensitive migration failure category.
func Reason(err error) string {
	if value, ok := err.(migrateError); ok {
		return value.reason
	}
	return ReasonMigrationFailed
}

// Version returns the migration version associated with a failure, if any.
func Version(err error) uint64 {
	if value, ok := err.(migrateError); ok {
		return value.version
	}
	return 0
}

// Run applies pending forward-only migrations under a connection-scoped named lock.
func Run(ctx context.Context, db *sql.DB, migrations []Migration) (Result, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return Result{}, migrateError{reason: ReasonDatabaseUnreachable}
	}
	defer conn.Close()
	return runLocked(ctx, &mysqlRunnerConnection{conn: conn}, migrations)
}

type runnerConnection interface {
	acquireLock(context.Context) (*bool, error)
	releaseLock(context.Context) error
	validateSession(context.Context) error
	history(context.Context) (bool, []historyRow, error)
	execute(context.Context, Migration) error
	verifyHistoryShape(context.Context) error
	insertClean(context.Context, Migration) error
	insertDirty(context.Context, Migration) error
	markClean(context.Context, Migration) error
}

func runLocked(ctx context.Context, connection runnerConnection, migrations []Migration) (result Result, returnErr error) {
	locked, err := connection.acquireLock(ctx)
	if err != nil || locked == nil || !*locked {
		return Result{}, migrateError{reason: ReasonMigrationLockUnavailable}
	}
	defer func() {
		if err := connection.releaseLock(context.WithoutCancel(ctx)); err != nil && returnErr == nil {
			returnErr = migrateError{reason: ReasonMigrationFailed}
		}
	}()

	if err := connection.validateSession(ctx); err != nil {
		return Result{}, normalizeError(err, ReasonDatabaseIncompatible, 0)
	}
	exists, rows, err := connection.history(ctx)
	if err != nil {
		return Result{}, normalizeError(err, ReasonDatabaseIncompatible, 0)
	}
	if len(rows) > 0 {
		state := classifyHistory(exists, rows, migrations)
		if state.Reason != "" && state.Reason != ReasonSchemaBehind {
			return Result{FromVersion: state.CurrentVersion, ToVersion: state.CurrentVersion}, migrateError{reason: state.Reason, version: state.CurrentVersion}
		}
		result.FromVersion = state.CurrentVersion
		result.ToVersion = state.CurrentVersion
	}

	start := len(rows)
	for index := start; index < len(migrations); index++ {
		migration := migrations[index]
		if migration.Version == 1 {
			if err := connection.execute(ctx, migration); err != nil {
				return result, migrateError{reason: ReasonStatementFailed, version: migration.Version}
			}
			if err := connection.verifyHistoryShape(ctx); err != nil {
				return result, migrateError{reason: ReasonDatabaseIncompatible, version: migration.Version}
			}
			if err := connection.insertClean(ctx, migration); err != nil {
				return result, migrateError{reason: ReasonMigrationFailed, version: migration.Version}
			}
		} else {
			if err := connection.insertDirty(ctx, migration); err != nil {
				return result, migrateError{reason: ReasonMigrationFailed, version: migration.Version}
			}
			if err := connection.execute(ctx, migration); err != nil {
				return result, migrateError{reason: ReasonStatementFailed, version: migration.Version}
			}
			if err := connection.markClean(ctx, migration); err != nil {
				return result, migrateError{reason: ReasonMigrationFailed, version: migration.Version}
			}
		}
		result.AppliedCount++
		result.ToVersion = migration.Version
	}
	return result, nil
}

func normalizeError(err error, fallback string, version uint64) error {
	if reason := Reason(err); reason != ReasonMigrationFailed {
		return migrateError{reason: reason, version: version}
	}
	return migrateError{reason: fallback, version: version}
}

type mysqlRunnerConnection struct {
	conn *sql.Conn
}

func (connection *mysqlRunnerConnection) acquireLock(ctx context.Context) (*bool, error) {
	var value sql.NullInt64
	if err := connection.conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 30)", migrationLockName).Scan(&value); err != nil {
		return nil, err
	}
	if !value.Valid {
		return nil, nil
	}
	locked := value.Int64 == 1
	return &locked, nil
}

func (connection *mysqlRunnerConnection) releaseLock(ctx context.Context) error {
	var value sql.NullInt64
	if err := connection.conn.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?)", migrationLockName).Scan(&value); err != nil || !value.Valid || value.Int64 != 1 {
		return fmt.Errorf("release lock failed")
	}
	return nil
}

func (connection *mysqlRunnerConnection) validateSession(ctx context.Context) error {
	var version, timezone, connectionCharset, serverCharset string
	if err := connection.conn.QueryRowContext(ctx, "SELECT VERSION(), @@session.time_zone, @@character_set_connection, @@character_set_server").Scan(&version, &timezone, &connectionCharset, &serverCharset); err != nil {
		return migrateError{reason: ReasonDatabaseUnreachable}
	}
	if !compatibleSession(version, timezone, connectionCharset, serverCharset) {
		return migrateError{reason: ReasonDatabaseIncompatible}
	}
	return nil
}

func (connection *mysqlRunnerConnection) history(ctx context.Context) (bool, []historyRow, error) {
	var exists int
	if err := connection.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='schema_migrations'").Scan(&exists); err != nil {
		return false, nil, migrateError{reason: ReasonDatabaseUnreachable}
	}
	if exists == 0 {
		return false, nil, nil
	}
	if err := connection.verifyHistoryShape(ctx); err != nil {
		return true, nil, err
	}
	rows, err := connection.conn.QueryContext(ctx, "SELECT version,name,checksum,dirty FROM schema_migrations ORDER BY version")
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
	if err := rows.Err(); err != nil {
		return true, nil, migrateError{reason: ReasonDatabaseUnreachable}
	}
	return true, result, nil
}

func (connection *mysqlRunnerConnection) execute(ctx context.Context, migration Migration) error {
	_, err := connection.conn.ExecContext(ctx, string(migration.SQL))
	return err
}

func (connection *mysqlRunnerConnection) verifyHistoryShape(ctx context.Context) error {
	var engine, collation string
	if err := connection.conn.QueryRowContext(ctx, "SELECT engine,table_collation FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='schema_migrations'").Scan(&engine, &collation); err != nil || engine != "InnoDB" || collation != "utf8mb4_0900_ai_ci" {
		return migrateError{reason: ReasonDatabaseIncompatible}
	}
	rows, err := connection.conn.QueryContext(ctx, "SELECT column_name,column_type,is_nullable,column_key FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='schema_migrations' ORDER BY ordinal_position")
	if err != nil {
		return migrateError{reason: ReasonDatabaseUnreachable}
	}
	defer rows.Close()
	type column struct{ name, dataType, nullable, key string }
	want := []column{{"version", "bigint unsigned", "NO", "PRI"}, {"name", "varchar(255)", "NO", ""}, {"checksum", "binary(32)", "NO", ""}, {"dirty", "tinyint(1)", "NO", ""}, {"applied_at", "timestamp(6)", "YES", ""}}
	index := 0
	for rows.Next() {
		if index >= len(want) {
			return migrateError{reason: ReasonDatabaseIncompatible}
		}
		var got column
		if err := rows.Scan(&got.name, &got.dataType, &got.nullable, &got.key); err != nil || got != want[index] {
			return migrateError{reason: ReasonDatabaseIncompatible}
		}
		index++
	}
	if rows.Err() != nil || index != len(want) {
		return migrateError{reason: ReasonDatabaseIncompatible}
	}
	return nil
}

func (connection *mysqlRunnerConnection) insertClean(ctx context.Context, migration Migration) error {
	_, err := connection.conn.ExecContext(ctx, "INSERT INTO schema_migrations(version,name,checksum,dirty,applied_at) VALUES (?,?,?,FALSE,CURRENT_TIMESTAMP(6))", migration.Version, migration.Name, migration.Checksum[:])
	return err
}

func (connection *mysqlRunnerConnection) insertDirty(ctx context.Context, migration Migration) error {
	_, err := connection.conn.ExecContext(ctx, "INSERT INTO schema_migrations(version,name,checksum,dirty,applied_at) VALUES (?,?,?,TRUE,NULL)", migration.Version, migration.Name, migration.Checksum[:])
	return err
}

func (connection *mysqlRunnerConnection) markClean(ctx context.Context, migration Migration) error {
	result, err := connection.conn.ExecContext(ctx, "UPDATE schema_migrations SET dirty=FALSE,applied_at=CURRENT_TIMESTAMP(6) WHERE version=? AND dirty=TRUE", migration.Version)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		return fmt.Errorf("unexpected clean update")
	}
	return nil
}

func versionString(version uint64) string { return strconv.FormatUint(version, 10) }
