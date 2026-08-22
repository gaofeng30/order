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
		"000004_add_products_meal_period.sql",
		"000005_create_meal_periods.sql",
		"000006_initialize_meal_periods.sql",
		"000007_create_product_sold_out_dates.sql",
		"000008_create_miniprogram_users.sql",
		"000009_create_miniprogram_sessions.sql",
		"000010_add_miniprogram_primary_phone.sql",
		"000011_create_storefront_settings.sql",
		"000012_create_merchant_accounts.sql",
		"000013_create_merchant_action_audits.sql",
	}
	if len(loaded) != len(wantNames) {
		t.Fatalf("migration count = %d, want %d", len(loaded), len(wantNames))
	}
	for index, wantName := range wantNames {
		if loaded[index].Name != wantName || loaded[index].Version != uint64(index+1) {
			t.Fatalf("migration %d = %d/%q, want %d/%q", index, loaded[index].Version, loaded[index].Name, index+1, wantName)
		}
	}

	for _, name := range wantNames[1:3] {
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

func TestIdentityMigrationContracts(t *testing.T) {
	checks := map[string][]string{
		"000008_create_miniprogram_users.sql": {
			"CREATE TABLE miniprogram_users", "id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT", "openid VARBINARY(128) NOT NULL", "created_at TIMESTAMP(6) NOT NULL", "last_login_at TIMESTAMP(6) NOT NULL", "UNIQUE KEY uq_miniprogram_users_openid (openid)",
		},
		"000009_create_miniprogram_sessions.sql": {
			"CREATE TABLE miniprogram_sessions", "token_hash BINARY(32) NOT NULL", "user_id BIGINT UNSIGNED NOT NULL", "issued_at TIMESTAMP(6) NOT NULL", "expires_at TIMESTAMP(6) NOT NULL", "PRIMARY KEY (token_hash)", "KEY idx_miniprogram_sessions_user_expiry (user_id, expires_at)", "CONSTRAINT fk_miniprogram_sessions_user", "ON UPDATE RESTRICT ON DELETE RESTRICT", "CONSTRAINT chk_miniprogram_sessions_expiry CHECK (expires_at > issued_at)",
		},
	}
	for name, required := range checks {
		data, err := fs.ReadFile(migrations.FS, name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		text := string(data)
		if strings.Count(strings.TrimSpace(text), ";") != 1 || !strings.HasSuffix(text, "\n") {
			t.Fatalf("%s must remain one LF-terminated statement", name)
		}
		for _, fragment := range required {
			if !strings.Contains(text, fragment) {
				t.Fatalf("%s missing %q", name, fragment)
			}
		}
		for _, forbidden := range []string{"phone", "merchant", "role", "unionid", "session_key", " raw_token", " code "} {
			if strings.Contains(strings.ToLower(text), forbidden) {
				t.Fatalf("%s contains forbidden identity field %q", name, forbidden)
			}
		}
	}
}

func TestPrimaryPhoneMigrationContract(t *testing.T) {
	const name = "000010_add_miniprogram_primary_phone.sql"
	data, err := fs.ReadFile(migrations.FS, name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	text := string(data)
	if strings.Count(strings.TrimSpace(text), ";") != 1 || !strings.HasSuffix(text, "\n") {
		t.Fatalf("%s must remain one LF-terminated statement", name)
	}
	for _, fragment := range []string{
		"ALTER TABLE miniprogram_users",
		"ADD COLUMN primary_phone VARBINARY(16) NULL",
		"ADD COLUMN primary_phone_bound_at TIMESTAMP(6) NULL",
		"ADD UNIQUE KEY uq_miniprogram_users_primary_phone (primary_phone)",
		"ADD CONSTRAINT chk_miniprogram_users_primary_phone_pair CHECK",
		"primary_phone IS NULL AND primary_phone_bound_at IS NULL",
		"primary_phone IS NOT NULL AND primary_phone_bound_at IS NOT NULL",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("%s missing %q", name, fragment)
		}
	}
}

func TestMenuMigrationContracts(t *testing.T) {
	checks := map[string][]string{
		"000004_add_products_meal_period.sql": {
			"ALTER TABLE products", "meal_period ENUM('all','lunch','dinner') NOT NULL DEFAULT 'all'", "idx_products_menu",
		},
		"000005_create_meal_periods.sql": {
			"CREATE TABLE meal_periods", "code ENUM('lunch','dinner')", "cutoff_time TIME NOT NULL", "pickup_start_time TIME NOT NULL", "pickup_end_time TIME NOT NULL", "interval_minutes SMALLINT UNSIGNED NOT NULL", "SECOND(cutoff_time) = 0", "SECOND(pickup_start_time) = 0", "SECOND(pickup_end_time) = 0", "interval_minutes BETWEEN 1 AND 1440", "cutoff_time <= pickup_start_time", "pickup_start_time <= pickup_end_time",
		},
		"000006_initialize_meal_periods.sql": {
			"INSERT INTO meal_periods", "('lunch','11:30:00','11:30:00','13:30:00',30)", "('dinner','17:00:00','17:00:00','19:00:00',30)",
		},
		"000007_create_product_sold_out_dates.sql": {
			"CREATE TABLE product_sold_out_dates", "service_date DATE NOT NULL", "product_id BIGINT UNSIGNED NOT NULL", "PRIMARY KEY (service_date, product_id)", "ON UPDATE RESTRICT ON DELETE RESTRICT",
		},
	}
	for name, required := range checks {
		data, err := fs.ReadFile(migrations.FS, name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		text := string(data)
		if strings.Count(strings.TrimSpace(text), ";") != 1 || !strings.HasSuffix(text, "\n") {
			t.Fatalf("%s must remain one LF-terminated statement", name)
		}
		for _, fragment := range required {
			if !strings.Contains(text, fragment) {
				t.Fatalf("%s missing %q", name, fragment)
			}
		}
	}
}
