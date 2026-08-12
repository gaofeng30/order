package migrate

import (
	"crypto/sha256"
	"testing"
)

func TestClassifyHistoryStates(t *testing.T) {
	migrations := []Migration{
		{Version: 1, Name: "000001_create_schema_migrations.sql", Checksum: sha256.Sum256([]byte("one"))},
		{Version: 2, Name: "000002_probe.sql", Checksum: sha256.Sum256([]byte("two"))},
	}

	tests := []struct {
		name   string
		exists bool
		rows   []historyRow
		want   string
	}{
		{name: "table absent", want: ReasonSchemaUninitialized},
		{name: "empty table", exists: true, want: ReasonSchemaUninitialized},
		{name: "clean prefix", exists: true, rows: []historyRow{rowFor(migrations[0], false)}, want: ReasonSchemaBehind},
		{name: "current", exists: true, rows: []historyRow{rowFor(migrations[0], false), rowFor(migrations[1], false)}, want: ""},
		{name: "dirty", exists: true, rows: []historyRow{rowFor(migrations[0], false), rowFor(migrations[1], true)}, want: ReasonSchemaDirty},
		{name: "too new", exists: true, rows: []historyRow{rowFor(migrations[0], false), rowFor(migrations[1], false), {Version: 3}}, want: ReasonSchemaTooNew},
		{name: "missing history", exists: true, rows: []historyRow{{Version: 2}}, want: ReasonSchemaTooNew},
		{name: "name drift", exists: true, rows: []historyRow{{Version: 1, Name: "drift", Checksum: migrations[0].Checksum}}, want: ReasonSchemaChecksumMismatch},
		{name: "checksum drift", exists: true, rows: []historyRow{{Version: 1, Name: migrations[0].Name}}, want: ReasonSchemaChecksumMismatch},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := classifyHistory(test.exists, test.rows, migrations)
			if state.Reason != test.want || state.Ready != (test.want == "") {
				t.Fatalf("state = %#v, want reason %q", state, test.want)
			}
		})
	}
}

func TestCompatibleSessionRequiresMySQL80UTCAndUTF8MB4(t *testing.T) {
	for _, test := range []struct {
		version, timezone, connectionCharset, serverCharset string
		want                                                bool
	}{
		{version: "8.0.46", timezone: "+00:00", connectionCharset: "utf8mb4", serverCharset: "utf8mb4", want: true},
		{version: "8.4.0", timezone: "+00:00", connectionCharset: "utf8mb4", serverCharset: "utf8mb4"},
		{version: "8.0.46", timezone: "SYSTEM", connectionCharset: "utf8mb4", serverCharset: "utf8mb4"},
		{version: "8.0.46", timezone: "+00:00", connectionCharset: "latin1", serverCharset: "utf8mb4"},
		{version: "8.0.46", timezone: "+00:00", connectionCharset: "utf8mb4", serverCharset: "latin1"},
	} {
		if got := compatibleSession(test.version, test.timezone, test.connectionCharset, test.serverCharset); got != test.want {
			t.Fatalf("compatibleSession(%q,%q,%q,%q) = %v, want %v", test.version, test.timezone, test.connectionCharset, test.serverCharset, got, test.want)
		}
	}
}

func rowFor(migration Migration, dirty bool) historyRow {
	return historyRow{Version: migration.Version, Name: migration.Name, Checksum: migration.Checksum, Dirty: dirty}
}
