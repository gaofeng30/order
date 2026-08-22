package merchantidentity

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/gaofeng30/order/services/api/migrations"
)

func TestMerchantIdentityMigrationContracts(t *testing.T) {
	checks := map[string][]string{
		"000012_create_merchant_accounts.sql": {
			"CREATE TABLE merchant_accounts",
			"id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT",
			"phone VARBINARY(16) NOT NULL",
			"name TEXT NOT NULL",
			"role ENUM('OWNER','SUBACCOUNT') NOT NULL",
			"enabled BOOLEAN NOT NULL DEFAULT TRUE",
			"record_version BIGINT UNSIGNED NOT NULL DEFAULT 1",
			"auth_version BIGINT UNSIGNED NOT NULL DEFAULT 1",
			"bound_user_id BIGINT UNSIGNED NULL",
			"bound_at TIMESTAMP(6) NULL",
			"created_at TIMESTAMP(6) NOT NULL",
			"updated_at TIMESTAMP(6) NOT NULL",
			"created_by BIGINT UNSIGNED NULL",
			"updated_by BIGINT UNSIGNED NULL",
			"UNIQUE KEY uq_merchant_accounts_phone (phone)",
			"UNIQUE KEY uq_merchant_accounts_bound_user (bound_user_id)",
			"CONSTRAINT fk_merchant_accounts_bound_user",
			"CONSTRAINT chk_merchant_accounts_phone CHECK",
			"CONSTRAINT chk_merchant_accounts_name CHECK",
			"CONSTRAINT chk_merchant_accounts_enabled CHECK",
			"CONSTRAINT chk_merchant_accounts_versions CHECK",
			"record_version > 0 AND auth_version > 0",
			"CONSTRAINT chk_merchant_accounts_binding_pair CHECK",
			"ON UPDATE RESTRICT ON DELETE RESTRICT",
		},
		"000013_create_merchant_action_audits.sql": {
			"CREATE TABLE merchant_action_audits",
			"merchant_account_id BIGINT UNSIGNED NULL",
			"account_id_snapshot BIGINT UNSIGNED NULL",
			"role_snapshot ENUM('OWNER','SUBACCOUNT') NULL",
			"auth_version_snapshot BIGINT UNSIGNED NULL",
			"actor_user_id BIGINT UNSIGNED NOT NULL",
			"action VARCHAR(64) NOT NULL",
			"result ENUM('SUCCEEDED','REJECTED') NOT NULL",
			"reason VARCHAR(64) NOT NULL",
			"target_type VARCHAR(64) NULL",
			"target_id BIGINT UNSIGNED NULL",
			"request_id VARBINARY(64) NOT NULL",
			"idempotency_key_hash BINARY(32) NULL",
			"state_before VARCHAR(64) NULL",
			"state_after VARCHAR(64) NULL",
			"occurred_at TIMESTAMP(6) NOT NULL",
			"CONSTRAINT fk_merchant_action_audits_account",
			"ON UPDATE RESTRICT ON DELETE SET NULL",
			"CONSTRAINT chk_merchant_action_audits_snapshot CHECK",
			"CONSTRAINT chk_merchant_action_audits_target CHECK",
			"CONSTRAINT chk_merchant_action_audits_actor CHECK",
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
		for _, forbidden := range []string{"openid", "code", "token", "phone_snapshot", "name_snapshot", "staff_whitelist"} {
			if strings.Contains(strings.ToLower(text), forbidden) {
				t.Fatalf("%s contains forbidden sensitive or cross-list token %q", name, forbidden)
			}
		}
	}
}
