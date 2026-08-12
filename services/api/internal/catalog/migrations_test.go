package catalog

import (
	"io/fs"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/migrations"
)

func TestCatalogMigrationSet(t *testing.T) {
	loaded, err := migrate.Load(migrations.FS)
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	wantNames := []string{
		"000001_create_schema_migrations.sql",
		"000002_create_categories.sql",
		"000003_create_products.sql",
	}
	if len(loaded) != len(wantNames) {
		t.Fatalf("migration count = %d, want %d", len(loaded), len(wantNames))
	}
	for index, wantName := range wantNames {
		if loaded[index].Name != wantName || loaded[index].Version != uint64(index+1) {
			t.Fatalf("migration %d = %d/%q, want %d/%q", index, loaded[index].Version, loaded[index].Name, index+1, wantName)
		}
	}

	for _, name := range wantNames[1:] {
		data, err := fs.ReadFile(migrations.FS, name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		text := string(data)
		trimmed := strings.TrimSpace(text)
		if !utf8.Valid(data) || !strings.HasSuffix(text, "\n") || strings.Contains(text, "\r") {
			t.Fatalf("%s must use LF and end with newline", name)
		}
		if strings.Count(trimmed, ";") != 1 || !strings.HasSuffix(trimmed, ";") {
			t.Fatalf("%s must contain one terminated statement", name)
		}
		upper := strings.ToUpper(trimmed)
		if !strings.HasPrefix(upper, "CREATE TABLE ") {
			t.Fatalf("%s must be one CREATE TABLE statement", name)
		}
		if strings.Count(upper, "CREATE TABLE ") != 1 {
			t.Fatalf("%s must contain exactly one CREATE TABLE", name)
		}
		for _, forbidden := range []string{" SEED", " DOWN", " REPAIR", " FORCE", "DELIMITER", "SOURCE ", "LOAD DATA"} {
			if strings.Contains(upper, forbidden) {
				t.Fatalf("%s contains forbidden token %q", name, forbidden)
			}
		}
	}
}
