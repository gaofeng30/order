package migrate

import (
	"context"
	"crypto/sha256"
	"errors"
	"reflect"
	"testing"
)

func TestRunLockedUsesOneConnectionAndIsIdempotent(t *testing.T) {
	migration := Migration{Version: 1, Name: "000001_create_schema_migrations.sql", SQL: []byte("CREATE TABLE IF NOT EXISTS schema_migrations (...);\n"), Checksum: sha256.Sum256([]byte("v1"))}
	connection := &fakeRunnerConnection{historyExists: false}

	result, err := runLocked(context.Background(), connection, []Migration{migration})
	if err != nil {
		t.Fatalf("runLocked() error = %v", err)
	}
	if result.AppliedCount != 1 {
		t.Fatalf("result = %#v, want one applied", result)
	}
	wantCalls := []string{"get_lock", "session", "history", "exec_v1", "shape", "insert_clean_1", "release_lock"}
	if !reflect.DeepEqual(connection.calls, wantCalls) {
		t.Fatalf("calls = %#v, want %#v", connection.calls, wantCalls)
	}

	connection.calls = nil
	connection.historyExists = true
	connection.rows = []historyRow{rowFor(migration, false)}
	result, err = runLocked(context.Background(), connection, []Migration{migration})
	if err != nil || result.AppliedCount != 0 {
		t.Fatalf("repeat result/error = %#v/%v, want zero/nil", result, err)
	}
	if !reflect.DeepEqual(connection.calls, []string{"get_lock", "session", "history", "release_lock"}) {
		t.Fatalf("repeat calls = %#v", connection.calls)
	}
}

func TestRunLockedRecoversAfterHistoryTableCreateCrash(t *testing.T) {
	migration := Migration{Version: 1, Name: "000001_create_schema_migrations.sql", SQL: []byte("CREATE TABLE IF NOT EXISTS schema_migrations (...);\n"), Checksum: sha256.Sum256([]byte("v1"))}
	connection := &fakeRunnerConnection{historyExists: true}

	result, err := runLocked(context.Background(), connection, []Migration{migration})
	if err != nil || result.AppliedCount != 1 {
		t.Fatalf("runLocked() = %#v, %v; want recoverable one migration", result, err)
	}
	if !reflect.DeepEqual(connection.calls, []string{"get_lock", "session", "history", "exec_v1", "shape", "insert_clean_1", "release_lock"}) {
		t.Fatalf("crash recovery calls = %#v", connection.calls)
	}
}

func TestRunLockedFailureLeavesDirtyAndStops(t *testing.T) {
	migrations := []Migration{
		{Version: 1, Name: "000001_create_schema_migrations.sql", Checksum: sha256.Sum256([]byte("v1"))},
		{Version: 2, Name: "000002_fail.sql", SQL: []byte("INVALID SQL;\n"), Checksum: sha256.Sum256([]byte("v2"))},
		{Version: 3, Name: "000003_never.sql", SQL: []byte("CREATE TABLE never (id BIGINT);\n"), Checksum: sha256.Sum256([]byte("v3"))},
	}
	connection := &fakeRunnerConnection{historyExists: true, rows: []historyRow{rowFor(migrations[0], false)}, execErrorAt: 2}

	_, err := runLocked(context.Background(), connection, migrations)
	if Reason(err) != ReasonStatementFailed {
		t.Fatalf("Reason(error) = %q, want %q", Reason(err), ReasonStatementFailed)
	}
	if !reflect.DeepEqual(connection.calls, []string{"get_lock", "session", "history", "insert_dirty_2", "exec_2", "release_lock"}) {
		t.Fatalf("failure calls = %#v", connection.calls)
	}
}

func TestRunLockedRejectsLockOutcomes(t *testing.T) {
	for _, test := range []struct {
		name       string
		lock       *bool
		returnNull bool
		err        error
	}{
		{name: "timeout", lock: boolPointer(false)},
		{name: "null", returnNull: true},
		{name: "error", err: errors.New("canary database error")},
	} {
		t.Run(test.name, func(t *testing.T) {
			connection := &fakeRunnerConnection{lock: test.lock, returnNullLock: test.returnNull, lockError: test.err}
			_, err := runLocked(context.Background(), connection, nil)
			if Reason(err) != ReasonMigrationLockUnavailable {
				t.Fatalf("Reason(error) = %q, want lock unavailable", Reason(err))
			}
			if len(connection.calls) != 1 || connection.calls[0] != "get_lock" {
				t.Fatalf("calls = %#v, want get_lock only", connection.calls)
			}
		})
	}
}

func TestRunLockedReportsReleaseFailure(t *testing.T) {
	connection := &fakeRunnerConnection{returnReleaseError: true}
	_, err := runLocked(context.Background(), connection, nil)
	if Reason(err) != ReasonMigrationFailed {
		t.Fatalf("Reason(error) = %q, want migration failure", Reason(err))
	}
	if !reflect.DeepEqual(connection.calls, []string{"get_lock", "session", "history", "release_lock"}) {
		t.Fatalf("calls = %#v", connection.calls)
	}
}

type fakeRunnerConnection struct {
	calls              []string
	lock               *bool
	returnNullLock     bool
	lockError          error
	historyExists      bool
	rows               []historyRow
	execErrorAt        uint64
	returnReleaseError bool
}

func (connection *fakeRunnerConnection) acquireLock(context.Context) (*bool, error) {
	connection.calls = append(connection.calls, "get_lock")
	if connection.lockError != nil {
		return nil, connection.lockError
	}
	if connection.returnNullLock {
		return nil, nil
	}
	if connection.lock != nil {
		return connection.lock, nil
	}
	value := true
	return &value, nil
}

func (connection *fakeRunnerConnection) releaseLock(context.Context) error {
	connection.calls = append(connection.calls, "release_lock")
	if connection.returnReleaseError {
		return errors.New("release canary")
	}
	return nil
}

func (connection *fakeRunnerConnection) validateSession(context.Context) error {
	connection.calls = append(connection.calls, "session")
	return nil
}

func (connection *fakeRunnerConnection) history(context.Context) (bool, []historyRow, error) {
	connection.calls = append(connection.calls, "history")
	return connection.historyExists, connection.rows, nil
}

func (connection *fakeRunnerConnection) execute(_ context.Context, migration Migration) error {
	if migration.Version == 1 {
		connection.calls = append(connection.calls, "exec_v1")
	} else {
		connection.calls = append(connection.calls, "exec_"+versionString(migration.Version))
	}
	if connection.execErrorAt == migration.Version {
		return errors.New("statement canary")
	}
	return nil
}

func (connection *fakeRunnerConnection) verifyHistoryShape(context.Context) error {
	connection.calls = append(connection.calls, "shape")
	return nil
}

func (connection *fakeRunnerConnection) insertClean(_ context.Context, migration Migration) error {
	connection.calls = append(connection.calls, "insert_clean_"+versionString(migration.Version))
	return nil
}

func (connection *fakeRunnerConnection) insertDirty(_ context.Context, migration Migration) error {
	connection.calls = append(connection.calls, "insert_dirty_"+versionString(migration.Version))
	return nil
}

func (connection *fakeRunnerConnection) markClean(_ context.Context, migration Migration) error {
	connection.calls = append(connection.calls, "mark_clean_"+versionString(migration.Version))
	return nil
}

func boolPointer(value bool) *bool { return &value }
