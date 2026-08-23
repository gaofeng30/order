package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestFrozenV18ToV44LedgerContracts(t *testing.T) {
	want := []string{
		"000018_extend_miniprogram_users.sql",
		"000019_add_category_name_key.sql",
		"000020_backfill_category_name_keys.sql",
		"000021_constrain_category_name_key.sql",
		"000022_add_product_name_key_images.sql",
		"000023_backfill_product_name_keys.sql",
		"000024_constrain_product_catalog_fields.sql",
		"000025_extend_storefront_settings.sql",
		"000026_clear_legacy_launch_layer.sql",
		"000027_constrain_storefront_settings.sql",
		"000028_soft_delete_merchant_accounts.sql",
		"000029_create_service_dates.sql",
		"000030_create_merchant_pc_sessions.sql",
		"000031_create_prepayments.sql",
		"000032_create_payment_observations.sql",
		"000033_create_pickup_sequences.sql",
		"000034_create_orders.sql",
		"000035_create_order_items.sql",
		"000036_create_refunds.sql",
		"000037_create_refund_observations.sql",
		"000038_create_notification_consents.sql",
		"000039_create_notification_outbox.sql",
		"000040_create_import_batches.sql",
		"000041_rename_action_audits.sql",
		"000042_generalize_action_audits.sql",
		"000043_backfill_action_audits.sql",
		"000044_constrain_action_audits.sql",
	}
	var ledger strings.Builder
	for _, name := range want {
		data, err := fs.ReadFile(FS, name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		text := string(data)
		trimmed := strings.TrimSpace(text)
		if !strings.HasSuffix(text, "\n") || strings.Contains(text, "\r") || strings.Count(trimmed, ";") != 1 || !strings.HasSuffix(trimmed, ";") {
			t.Fatalf("%s must be one LF-terminated statement", name)
		}
		ledger.WriteString(text)
	}

	all := ledger.String()
	for _, required := range []string{
		"CREATE TABLE service_dates",
		"CREATE TABLE merchant_pc_sessions",
		"CREATE TABLE prepayments",
		"CREATE TABLE payment_observations",
		"CREATE TABLE pickup_sequences",
		"CREATE TABLE orders",
		"CREATE TABLE order_items",
		"CREATE TABLE refunds",
		"CREATE TABLE refund_observations",
		"CREATE TABLE notification_consents",
		"CREATE TABLE notification_outbox",
		"CREATE TABLE import_batches",
		"RENAME TABLE merchant_action_audits TO action_audits",
		"provider_state ENUM('READY','CREATE_CLAIMED','CREATE_UNKNOWN'",
		"UNIQUE KEY uq_orders_quote (quote_id)",
		"UNIQUE KEY uq_orders_pickup (pickup_date,pickup_number)",
		"operation_key_hash=NULL",
		"entry_kind='COMMAND_RECEIPT' AND operation_key_hash IS NOT NULL",
	} {
		if !strings.Contains(all, required) {
			t.Fatalf("frozen ledger missing %q", required)
		}
	}
	if got := strings.Count(all, "KEY idx_"); got != 3 {
		t.Fatalf("non-unique range indexes = %d, want only R1-R3", got)
	}
	if foreign, restrict := strings.Count(all, " REFERENCES "), strings.Count(all, "ON UPDATE RESTRICT ON DELETE RESTRICT"); foreign != restrict {
		t.Fatalf("foreign keys=%d but exact RESTRICT clauses=%d", foreign, restrict)
	}
	for _, forbidden := range []string{"inventory", "coupon", "member_level", "order_state_events", "dashboard_stats", "finance_daily", "import_rows", "refund_items"} {
		if strings.Contains(strings.ToLower(all), forbidden) {
			t.Fatalf("frozen ledger contains out-of-scope model %q", forbidden)
		}
	}
}
