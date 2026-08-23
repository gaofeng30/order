package migrations

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/gaofeng30/order/services/api/internal/migrate"
)

func TestEmbeddedMigrationChainIsExactAndRecoverable(t *testing.T) {
	want := []struct {
		name   string
		prefix string
	}{
		{name: "000001_create_schema_migrations.sql", prefix: "CREATE TABLE "},
		{name: "000002_create_categories.sql", prefix: "CREATE TABLE "},
		{name: "000003_create_products.sql", prefix: "CREATE TABLE "},
		{name: "000004_add_products_meal_period.sql", prefix: "ALTER TABLE "},
		{name: "000005_create_meal_periods.sql", prefix: "CREATE TABLE "},
		{name: "000006_initialize_meal_periods.sql", prefix: "INSERT INTO "},
		{name: "000007_create_product_sold_out_dates.sql", prefix: "CREATE TABLE "},
		{name: "000008_create_miniprogram_users.sql", prefix: "CREATE TABLE "},
		{name: "000009_create_miniprogram_sessions.sql", prefix: "CREATE TABLE "},
		{name: "000010_add_miniprogram_primary_phone.sql", prefix: "ALTER TABLE "},
		{name: "000011_create_storefront_settings.sql", prefix: "CREATE TABLE "},
		{name: "000012_create_merchant_accounts.sql", prefix: "CREATE TABLE "},
		{name: "000013_create_merchant_action_audits.sql", prefix: "CREATE TABLE "},
		{name: "000014_create_staff_whitelist.sql", prefix: "CREATE TABLE "},
		{name: "000015_create_discount_settings.sql", prefix: "CREATE TABLE "},
		{name: "000016_create_quotes.sql", prefix: "CREATE TABLE "},
		{name: "000017_create_quote_items.sql", prefix: "CREATE TABLE "},
		{name: "000018_extend_miniprogram_users.sql", prefix: "ALTER TABLE "},
		{name: "000019_add_category_name_key.sql", prefix: "ALTER TABLE "},
		{name: "000020_backfill_category_name_keys.sql", prefix: "UPDATE "},
		{name: "000021_constrain_category_name_key.sql", prefix: "ALTER TABLE "},
		{name: "000022_add_product_name_key_images.sql", prefix: "ALTER TABLE "},
		{name: "000023_backfill_product_name_keys.sql", prefix: "UPDATE "},
		{name: "000024_constrain_product_catalog_fields.sql", prefix: "ALTER TABLE "},
		{name: "000025_extend_storefront_settings.sql", prefix: "ALTER TABLE "},
		{name: "000026_clear_legacy_launch_layer.sql", prefix: "UPDATE "},
		{name: "000027_constrain_storefront_settings.sql", prefix: "ALTER TABLE "},
		{name: "000028_soft_delete_merchant_accounts.sql", prefix: "ALTER TABLE "},
		{name: "000029_create_service_dates.sql", prefix: "CREATE TABLE "},
		{name: "000030_create_merchant_pc_sessions.sql", prefix: "CREATE TABLE "},
		{name: "000031_create_prepayments.sql", prefix: "CREATE TABLE "},
		{name: "000032_create_payment_observations.sql", prefix: "CREATE TABLE "},
		{name: "000033_create_pickup_sequences.sql", prefix: "CREATE TABLE "},
		{name: "000034_create_orders.sql", prefix: "CREATE TABLE "},
		{name: "000035_create_order_items.sql", prefix: "CREATE TABLE "},
		{name: "000036_create_refunds.sql", prefix: "CREATE TABLE "},
		{name: "000037_create_refund_observations.sql", prefix: "CREATE TABLE "},
		{name: "000038_create_notification_consents.sql", prefix: "CREATE TABLE "},
		{name: "000039_create_notification_outbox.sql", prefix: "CREATE TABLE "},
		{name: "000040_create_import_batches.sql", prefix: "CREATE TABLE "},
		{name: "000041_rename_action_audits.sql", prefix: "RENAME TABLE "},
		{name: "000042_generalize_action_audits.sql", prefix: "ALTER TABLE "},
		{name: "000043_backfill_action_audits.sql", prefix: "UPDATE "},
		{name: "000044_constrain_action_audits.sql", prefix: "ALTER TABLE "},
	}
	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != len(want) {
		t.Fatalf("embedded migrations = %d, want exact v1-v44 chain", len(entries))
	}
	for index, expected := range want {
		if entries[index].IsDir() || entries[index].Name() != expected.name {
			t.Fatalf("embedded migration %d = %q, want %q", index, entries[index].Name(), expected.name)
		}
		data, err := fs.ReadFile(FS, expected.name)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", expected.name, err)
		}
		sql := string(data)
		trimmed := strings.TrimSpace(sql)
		if trimmed == "" || !strings.HasSuffix(sql, "\n") || strings.Contains(sql, "\r") {
			t.Fatalf("%s must be nonempty LF text with a final newline", expected.name)
		}
		upper := strings.ToUpper(trimmed)
		if strings.Count(trimmed, ";") != 1 || !strings.HasSuffix(trimmed, ";") || !strings.HasPrefix(upper, expected.prefix) {
			t.Fatalf("%s must contain exactly one terminated %s statement", expected.name, strings.TrimSpace(expected.prefix))
		}
		for _, forbidden := range []string{" seed", " down", " repair", " force", "drop table", "delimiter", "load data"} {
			if strings.Contains(strings.ToLower(trimmed), forbidden) {
				t.Fatalf("%s contains forbidden token %q", expected.name, forbidden)
			}
		}
	}

	loaded, err := migrate.Load(FS)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded) != len(want) {
		t.Fatalf("loaded migrations = %d, want %d", len(loaded), len(want))
	}
	for index, migration := range loaded {
		if migration.Version != uint64(index+1) || migration.Name != want[index].name || len(migration.SQL) == 0 {
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
