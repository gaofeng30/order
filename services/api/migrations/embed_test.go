package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestEmbeddedFoundationMigrationIsRecoverableAndContainsNoBusinessTables(t *testing.T) {
	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "000001_create_schema_migrations.sql" {
		t.Fatalf("embedded migrations = %#v, want only foundation migration", entries)
	}
	data, err := fs.ReadFile(FS, entries[0].Name())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(data)
	for _, required := range []string{"CREATE TABLE IF NOT EXISTS schema_migrations", "version BIGINT UNSIGNED", "name VARCHAR(255)", "checksum BINARY(32)", "dirty BOOLEAN", "applied_at TIMESTAMP(6) NULL", "ENGINE=InnoDB", "utf8mb4_0900_ai_ci"} {
		if !strings.Contains(sql, required) {
			t.Fatalf("foundation SQL missing %q", required)
		}
	}
	for _, forbidden := range []string{"orders", "catalog", "users", "inventory", "payment", "seed", "DROP TABLE"} {
		if strings.Contains(strings.ToLower(sql), strings.ToLower(forbidden)) {
			t.Fatalf("foundation SQL contains forbidden token %q", forbidden)
		}
	}
}
