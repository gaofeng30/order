package migrations

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/gaofeng30/order/services/api/internal/migrate"
)

func TestEmbeddedMigrationChainIsExactAndRecoverable(t *testing.T) {
	wantNames := []string{
		"000001_create_schema_migrations.sql",
		"000002_create_categories.sql",
		"000003_create_products.sql",
	}
	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != len(wantNames) {
		t.Fatalf("embedded migrations = %d, want exact v1-v3 chain", len(entries))
	}
	for index, wantName := range wantNames {
		if entries[index].IsDir() || entries[index].Name() != wantName {
			t.Fatalf("embedded migration %d = %q, want %q", index, entries[index].Name(), wantName)
		}
		data, err := fs.ReadFile(FS, wantName)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", wantName, err)
		}
		sql := string(data)
		trimmed := strings.TrimSpace(sql)
		if trimmed == "" || !strings.HasSuffix(sql, "\n") || strings.Contains(sql, "\r") {
			t.Fatalf("%s must be nonempty LF text with a final newline", wantName)
		}
		if strings.Count(trimmed, ";") != 1 || !strings.HasSuffix(trimmed, ";") || strings.Count(strings.ToUpper(trimmed), "CREATE TABLE ") != 1 {
			t.Fatalf("%s must contain exactly one terminated CREATE TABLE statement", wantName)
		}
		for _, forbidden := range []string{" seed", " down", " repair", " force", "drop table", "delimiter", "load data"} {
			if strings.Contains(strings.ToLower(trimmed), forbidden) {
				t.Fatalf("%s contains forbidden token %q", wantName, forbidden)
			}
		}
	}

	loaded, err := migrate.Load(FS)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded) != len(wantNames) {
		t.Fatalf("loaded migrations = %d, want %d", len(loaded), len(wantNames))
	}
	for index, migration := range loaded {
		if migration.Version != uint64(index+1) || migration.Name != wantNames[index] || len(migration.SQL) == 0 {
			t.Fatalf("loaded migration %d = %d/%q", index, migration.Version, migration.Name)
		}
	}

	foundationSQL := string(loaded[0].SQL)
	for _, required := range []string{"CREATE TABLE IF NOT EXISTS schema_migrations", "version BIGINT UNSIGNED", "name VARCHAR(255)", "checksum BINARY(32)", "dirty BOOLEAN", "applied_at TIMESTAMP(6) NULL", "ENGINE=InnoDB", "utf8mb4_0900_ai_ci"} {
		if !strings.Contains(foundationSQL, required) {
			t.Fatalf("foundation SQL missing %q", required)
		}
	}
	for _, forbidden := range []string{"orders", "users", "inventory", "payment", "seed", "DROP TABLE"} {
		if strings.Contains(strings.ToLower(foundationSQL), strings.ToLower(forbidden)) {
			t.Fatalf("foundation SQL contains forbidden token %q", forbidden)
		}
	}
}
