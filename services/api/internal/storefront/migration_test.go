package storefront

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/gaofeng30/order/services/api/migrations"
)

func TestStorefrontMigrationContract(t *testing.T) {
	const name = "000011_create_storefront_settings.sql"
	data, err := fs.ReadFile(migrations.FS, name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	text := string(data)
	if strings.Count(strings.TrimSpace(text), ";") != 1 || !strings.HasSuffix(text, "\n") || strings.Contains(text, "\r") {
		t.Fatalf("%s must be one LF-terminated statement", name)
	}
	for _, fragment := range []string{
		"CREATE TABLE storefront_settings",
		"id TINYINT UNSIGNED NOT NULL",
		"store_name TEXT NOT NULL",
		"store_address TEXT NOT NULL",
		"pickup_point TEXT NOT NULL",
		"announcement TEXT NOT NULL",
		"business_status ENUM('open','closed','cutoff') NOT NULL",
		"launch_png_url TEXT NULL",
		"center_x DOUBLE NULL",
		"center_y DOUBLE NULL",
		"width_ratio DOUBLE NULL",
		"aspect_ratio DOUBLE NULL",
		"PRIMARY KEY (id)",
		"CHECK (id = 1)",
		"CHAR_LENGTH(announcement) <= 1000",
		"launch_png_url IS NULL AND center_x IS NULL AND center_y IS NULL AND width_ratio IS NULL AND aspect_ratio IS NULL",
		"launch_png_url IS NOT NULL AND center_x IS NOT NULL AND center_y IS NOT NULL AND width_ratio IS NOT NULL AND aspect_ratio IS NOT NULL",
		"center_x BETWEEN 0 AND 1",
		"center_y BETWEEN 0 AND 1",
		"width_ratio > 0 AND width_ratio <= 1",
		"aspect_ratio > 0",
		"ENGINE=InnoDB",
		"CHARACTER SET=utf8mb4",
		"COLLATE=utf8mb4_0900_ai_ci",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("%s missing %q", name, fragment)
		}
	}
	upper := strings.ToUpper(text)
	for _, forbidden := range []string{"INSERT ", "REPLACE ", "UPDATE ", "DELETE ", "DROP ", " SEED", " DOWN", " REPAIR", " FORCE", "LOAD DATA"} {
		if strings.Contains(upper, forbidden) {
			t.Fatalf("%s contains forbidden token %q", name, forbidden)
		}
	}
}
