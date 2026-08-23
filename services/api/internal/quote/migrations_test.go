package quote

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/migrations"
)

func TestMigrationSetLoadsExactlyThroughV17(t *testing.T) {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	files := fstest.MapFS{}
	for _, entry := range entries {
		data, err := fs.ReadFile(migrations.FS, entry.Name())
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		files[entry.Name()] = &fstest.MapFile{Data: data}
		if _, err := migrate.Load(files); err != nil {
			t.Fatalf("migration prefix ending at %s is invalid", entry.Name())
		}
	}
	loaded, err := migrate.Load(migrations.FS)
	if err != nil || len(loaded) != 17 || loaded[16].Version != 17 || loaded[16].Name != "000017_create_quote_items.sql" {
		t.Fatalf("loaded migration set = count:%d err:%v", len(loaded), err)
	}
}

func TestStaffWhitelistMigrationContract(t *testing.T) {
	text := readSingleMigration(t, "000014_create_staff_whitelist.sql")
	for _, required := range []string{
		"CREATE TABLE staff_whitelist",
		"id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT",
		"phone VARBINARY(16) NOT NULL",
		"name TEXT NOT NULL",
		"name_key VARBINARY(400) NOT NULL",
		"enabled BOOLEAN NOT NULL DEFAULT TRUE",
		"record_version BIGINT UNSIGNED NOT NULL DEFAULT 1",
		"created_at TIMESTAMP(6) NOT NULL",
		"updated_at TIMESTAMP(6) NOT NULL",
		"UNIQUE KEY uq_staff_whitelist_phone (phone)",
		"REGEXP_LIKE(CONVERT(phone USING ascii)",
		"OCTET_LENGTH(name) = OCTET_LENGTH(TRIM(name))",
		"record_version > 0",
		"ENGINE=InnoDB",
		"COLLATE=utf8mb4_0900_ai_ci",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("v14 missing %q", required)
		}
	}
	if count := strings.Count(text, "\n  KEY "); count != 0 {
		t.Fatalf("v14 non-unique secondary index count = %d, want exact 0", count)
	}
}

func TestDiscountSettingsMigrationContract(t *testing.T) {
	text := readSingleMigration(t, "000015_create_discount_settings.sql")
	for _, required := range []string{
		"CREATE TABLE discount_settings",
		"id TINYINT UNSIGNED NOT NULL",
		"rate_percent TINYINT UNSIGNED NOT NULL",
		"discount_version BIGINT UNSIGNED NOT NULL",
		"whitelist_version BIGINT UNSIGNED NOT NULL",
		"updated_at TIMESTAMP(6) NOT NULL",
		"PRIMARY KEY (id)",
		"CHECK (id = 1)",
		"rate_percent >= 1 AND rate_percent <= 100",
		"discount_version > 0",
		"whitelist_version > 0",
		"ENGINE=InnoDB",
		"COLLATE=utf8mb4_0900_ai_ci",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("v15 missing %q", required)
		}
	}
}

func TestQuotesMigrationContract(t *testing.T) {
	text := readSingleMigration(t, "000016_create_quotes.sql")
	for _, required := range []string{
		"CREATE TABLE quotes",
		"id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT",
		"user_id BIGINT UNSIGNED NOT NULL",
		"contact_name_snapshot VARCHAR(64) NOT NULL",
		"contact_phone_snapshot VARBINARY(16) NOT NULL",
		"idempotency_key_hash BINARY(32) NOT NULL",
		"request_digest BINARY(32) NOT NULL",
		"identity_kind ENUM('STAFF','VISITOR') NOT NULL",
		"identity_source_version BIGINT UNSIGNED NOT NULL",
		"discount_rate_percent TINYINT UNSIGNED NOT NULL",
		"discount_version BIGINT UNSIGNED NOT NULL",
		"store_name_snapshot TEXT NOT NULL",
		"store_address_snapshot TEXT NOT NULL",
		"pickup_point_snapshot TEXT NOT NULL",
		"pickup_date DATE NOT NULL",
		"pickup_time TIME NOT NULL",
		"meal_period ENUM('lunch','dinner') NOT NULL",
		"order_note TEXT NOT NULL",
		"item_count SMALLINT UNSIGNED NOT NULL",
		"original_subtotal_cents BIGINT UNSIGNED NOT NULL",
		"discount_cents BIGINT UNSIGNED NOT NULL",
		"payable_cents BIGINT UNSIGNED NOT NULL",
		"snapshot_digest BINARY(32) NOT NULL",
		"created_at TIMESTAMP(6) NOT NULL",
		"expires_at TIMESTAMP(6) NOT NULL",
		"UNIQUE KEY uq_quotes_user_idempotency (user_id, idempotency_key_hash)",
		"FOREIGN KEY (user_id) REFERENCES miniprogram_users (id)",
		"OCTET_LENGTH(contact_name_snapshot) <= 64",
		"REGEXP_LIKE(CONVERT(contact_phone_snapshot USING ascii)",
		"discount_rate_percent >= 1 AND discount_rate_percent <= 100",
		"original_subtotal_cents = discount_cents + payable_cents",
		"payable_cents > 0",
		"expires_at > created_at",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("v16 missing %q", required)
		}
	}
	for _, forbidden := range []string{"ttl", "prepay", "out_trade_no", "order_id", "updated_at"} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Fatalf("v16 contains out-of-scope field %q", forbidden)
		}
	}
}

func TestQuoteItemsMigrationContract(t *testing.T) {
	text := readSingleMigration(t, "000017_create_quote_items.sql")
	for _, required := range []string{
		"CREATE TABLE quote_items",
		"quote_id BIGINT UNSIGNED NOT NULL",
		"line_number SMALLINT UNSIGNED NOT NULL",
		"product_id BIGINT UNSIGNED NOT NULL",
		"product_name_snapshot TEXT NOT NULL",
		"product_source_version BINARY(32) NOT NULL",
		"image_object_key_snapshot VARBINARY(1024) NULL",
		"original_unit_price_cents BIGINT UNSIGNED NOT NULL",
		"discounted_unit_price_cents BIGINT UNSIGNED NOT NULL",
		"quantity BIGINT UNSIGNED NOT NULL",
		"original_subtotal_cents BIGINT UNSIGNED NOT NULL",
		"payable_subtotal_cents BIGINT UNSIGNED NOT NULL",
		"flavors_json JSON NOT NULL",
		"line_note TEXT NOT NULL",
		"PRIMARY KEY (quote_id, line_number)",
		"FOREIGN KEY (quote_id) REFERENCES quotes (id)",
		"JSON_TYPE(flavors_json) = 'ARRAY'",
		"OCTET_LENGTH(image_object_key_snapshot) BETWEEN 1 AND 1024",
		"original_subtotal_cents = original_unit_price_cents * quantity",
		"payable_subtotal_cents = discounted_unit_price_cents * quantity",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("v17 missing %q", required)
		}
	}
	if strings.Contains(text, "REFERENCES products") {
		t.Fatal("v17 must preserve product snapshots after catalog deletion")
	}
}

func readSingleMigration(t *testing.T, name string) string {
	t.Helper()
	data, err := fs.ReadFile(migrations.FS, name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	text := string(data)
	trimmed := strings.TrimSpace(text)
	if strings.Count(trimmed, ";") != 1 || !strings.HasSuffix(text, "\n") || strings.Contains(text, "\r") {
		t.Fatalf("%s must be one LF-terminated statement", name)
	}
	upper := strings.ToUpper(trimmed)
	for _, forbidden := range []string{"DROP ", " SEED", " DOWN", " REPAIR", " FORCE", "LOAD DATA", "DELIMITER"} {
		if strings.Contains(upper, forbidden) {
			t.Fatalf("%s contains forbidden token %q", name, forbidden)
		}
	}
	return text
}
