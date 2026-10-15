// Package sharedpg owns the reviewed atomic command definitions separately
// from the immutable v1-v44 relational migration chain.
package sharedpg

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"net/url"

	"errors"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"regexp"
	"slices"
	"strings"
)

//go:embed redeem_order.sql fulfillment.sql identity_read.sql order_query.sql identity_commands.sql merchant_admin.sql pc_session.sql catalog.sql menu.sql storefront.sql image_confirmation.sql image_version_confirmation.sql display_pricing.sql merchant_soldout.sql quote.sql staff_admin.sql import_batch.sql payment_prepare.sql payment_observation.sql payment_materialize.sql payment_receipts.sql refund_commands.sql refund_observation.sql subscription.sql order_production.sql billing.sql admin_report.sql audit_search.sql store_status.sql access.sql payment_environment.sql
var source embed.FS
var ErrDefinitionMismatch = errors.New("shared PG command definition mismatch")

const identityMarker = "__ORDER_SERVICE_UID__"
const identityMarkerCount = 169
const schemaMarker = "__ORDER_SCHEMA_CATALOG_SQL__"

type Executor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}

// InstallSQL binds a management-approved CloudBase UID to the fixed reviewed
// definition. This UAT installation supports text UIDs of 1-64 ASCII digits;
// it never parses their numeric value or accepts caller-supplied SQL.
func InstallSQL(serviceUID, environment, paymentNotifyURL, refundNotifyURL string) (string, error) {
	if environment != "uat" && environment != "production" {
		return "", ErrDefinitionMismatch
	}
	payment, paymentErr := url.Parse(paymentNotifyURL)
	refund, refundErr := url.Parse(refundNotifyURL)
	if paymentErr != nil || refundErr != nil || payment.Scheme != "https" || refund.Scheme != "https" || payment.Host == "" || payment.Host != refund.Host || payment.User != nil || refund.User != nil || payment.Opaque != "" || refund.Opaque != "" || payment.RawQuery != "" || refund.RawQuery != "" || payment.ForceQuery || refund.ForceQuery || strings.Contains(paymentNotifyURL, "#") || strings.Contains(refundNotifyURL, "#") || payment.RawPath != "" || refund.RawPath != "" || payment.Path != "/api/v1/payments/wechat/notify" || refund.Path != "/api/v1/refunds/wechat/notify" {
		return "", ErrDefinitionMismatch
	}
	if len(serviceUID) < 1 || len(serviceUID) > 64 {
		return "", ErrDefinitionMismatch
	}
	for _, c := range serviceUID {
		if c < '0' || c > '9' {
			return "", ErrDefinitionMismatch
		}
	}

	var raw strings.Builder
	for _, file := range []struct {
		name    string
		markers int
	}{
		{"redeem_order.sql", 3}, {"fulfillment.sql", 4}, {"identity_read.sql", 1}, {"order_query.sql", 4}, {"identity_commands.sql", 9}, {"merchant_admin.sql", 3}, {"pc_session.sql", 6}, {"catalog.sql", 6}, {"menu.sql", 2}, {"storefront.sql", 5}, {"image_confirmation.sql", 1}, {"image_version_confirmation.sql", 3}, {"display_pricing.sql", 1}, {"merchant_soldout.sql", 2}, {"quote.sql", 2}, {"staff_admin.sql", 3}, {"import_batch.sql", 4}, {"payment_prepare.sql", 5}, {"payment_observation.sql", 4}, {"payment_materialize.sql", 4}, {"payment_receipts.sql", 2}, {"refund_commands.sql", 11}, {"refund_observation.sql", 1}, {"subscription.sql", 6}, {"order_production.sql", 2}, {"billing.sql", 7}, {"admin_report.sql", 9}, {"audit_search.sql", 1}, {"store_status.sql", 1}, {"access.sql", 52}, {"payment_environment.sql", 5},
	} {
		body, err := source.ReadFile(file.name)
		if err != nil || strings.Count(string(body), identityMarker) != file.markers {
			return "", ErrDefinitionMismatch
		}
		raw.Write(body)
		raw.WriteString("\n")
	}
	if strings.Count(raw.String(), identityMarker) != identityMarkerCount {
		return "", ErrDefinitionMismatch
	}
	catalog, err := migrate.SharedPostgreSQLCatalogReadinessSQL()
	if err != nil || strings.Count(raw.String(), schemaMarker) != 1 {
		return "", ErrDefinitionMismatch
	}
	if strings.Count(raw.String(), "__ORDER_ENVIRONMENT__") != 6 || strings.Count(raw.String(), "__ORDER_PAYMENT_NOTIFY_URL__") != 2 || strings.Count(raw.String(), "__ORDER_REFUND_NOTIFY_URL__") != 3 {
		return "", ErrDefinitionMismatch
	}
	definition := strings.NewReplacer(identityMarker, serviceUID, "__ORDER_ENVIRONMENT__", environment, "__ORDER_PAYMENT_NOTIFY_URL__", strings.ReplaceAll(paymentNotifyURL, "'", "''"), "__ORDER_REFUND_NOTIFY_URL__", strings.ReplaceAll(refundNotifyURL, "'", "''")).Replace(raw.String())
	return strings.ReplaceAll(definition, schemaMarker, catalog), nil
}

type function struct {
	Name, Hash, Language, Owner, Kind string
	SecurityDefiner                   bool `json:"security_definer"`
	ReturnsSet                        bool `json:"returns_set"`
	Strict                            bool
	Volatility, Parallel              string
	ResultType                        string   `json:"result_type"`
	ArgumentTypes                     string   `json:"argument_types"`
	ArgumentNames                     []string `json:"argument_names"`
	Config                            []string
	Defaults, Variadic                int
	PublicExecute                     bool `json:"public_execute"`
	AnonExecute                       bool `json:"anon_execute"`
	AuthenticatedExecute              bool `json:"authenticated_execute"`
}
type role struct {
	Name           string
	Superuser      bool
	CreateRole     bool     `json:"create_role"`
	CreateDatabase bool     `json:"create_database"`
	BypassRLS      bool     `json:"bypass_rls"`
	CreateSchemas  []string `json:"create_schemas"`
	Membership     []string
}
type policy struct {
	Name         string
	Permissive   bool
	Command      string
	Roles        []string
	Using, Check string
}
type table struct {
	Name, Owner                    string
	RLS                            bool
	PublicPrivileges               []string `json:"public_privileges"`
	AnonPrivileges                 []string `json:"anon_privileges"`
	AuthenticatedPrivileges        []string `json:"authenticated_privileges"`
	AuthenticatedUpdateColumns     []string `json:"authenticated_update_columns"`
	AnonColumnAccess               bool     `json:"anon_column_access"`
	PublicColumnAccess             bool     `json:"public_column_access"`
	AuthenticatedInsertColumns     bool     `json:"authenticated_insert_columns"`
	AuthenticatedInsertColumnNames []string `json:"authenticated_insert_column_names"`
	AuthenticatedReferenceColumns  bool     `json:"authenticated_reference_columns"`
	Policies                       []policy
}
type readiness struct {
	Role, UID           string
	Schema              json.RawMessage
	Roles               []role
	Functions           []function
	Tables              []table
	Sequences           []sequence
	UnreviewedRelations json.RawMessage `json:"unreviewed_relations"`
	UnreviewedDefiners  json.RawMessage `json:"unreviewed_definers"`
}
type sequence struct {
	Name, Owner             string
	PublicPrivileges        []string `json:"public_privileges"`
	AnonPrivileges          []string `json:"anon_privileges"`
	AuthenticatedPrivileges []string `json:"authenticated_privileges"`
}

// Check compares actual catalog metadata supplied by a fixed, authenticated
// read-only RPC with the reviewed definitions and least-privilege boundary.
// It does not certify that the remaining backend has been ported.
func Check(ctx context.Context, client Executor, serviceUID, environment, paymentNotifyURL, refundNotifyURL string) error {
	production := environment == "production"
	if err := checkBusinessCatalog(ctx, client, serviceUID, environment, paymentNotifyURL, refundNotifyURL, production); err != nil {
		return err
	}
	if production {
		return CheckCompensation(ctx, client, serviceUID)
	}
	return nil
}

func checkBusinessCatalog(ctx context.Context, client Executor, serviceUID, environment, paymentNotifyURL, refundNotifyURL string, compensationInstalled bool) error {
	definition, err := InstallSQL(serviceUID, environment, paymentNotifyURL, refundNotifyURL)
	if ctx == nil || client == nil || err != nil {
		return ErrDefinitionMismatch
	}
	matches := regexp.MustCompile(`(?s)CREATE OR REPLACE FUNCTION public\.(order_[a-z_0-9]+)\(.*?\).*?AS \$order_rpc\$(.*?)\$order_rpc\$;`).FindAllStringSubmatch(definition, -1)
	if len(matches) != 117 {
		return ErrDefinitionMismatch
	}
	bodies := make(map[string]string, len(matches))
	for _, match := range matches {
		if _, exists := bodies[match[1]]; exists {
			return ErrDefinitionMismatch
		}
		bodies[match[1]] = match[2]
	}
	var actual readiness
	callErr := client.CallRPC(ctx, "order_shared_pg_readiness_v1", map[string]any{}, &actual)
	if callErr != nil || actual.Role != "authenticated" || actual.UID != serviceUID || len(actual.Roles) != 2 || len(actual.Functions) != 117 || len(actual.Tables) != 26 || len(actual.Sequences) != 16 {
		return ErrDefinitionMismatch
	}
	var unreviewed []json.RawMessage
	if json.Unmarshal(actual.UnreviewedRelations, &unreviewed) != nil || unreviewed == nil || len(unreviewed) != 0 {
		return ErrDefinitionMismatch
	}
	var unreviewedDefiners []json.RawMessage
	if json.Unmarshal(actual.UnreviewedDefiners, &unreviewedDefiners) != nil || unreviewedDefiners == nil {
		return ErrDefinitionMismatch
	}
	if compensationInstalled {
		// These fixed credential and capability RPCs are checked in full by the additional
		// compensation catalog gate. No other definer may become callable.
		var definers []struct{ Signature, Role string }
		if json.Unmarshal(actual.UnreviewedDefiners, &definers) != nil || len(definers) != 3 {
			return ErrDefinitionMismatch
		}
		for i, name := range []string{"order_compensation_authenticate_v1()", "order_compensation_claim_v1(text)", "order_compensation_finish_v1(text,boolean)"} {
			if definers[i].Signature != name || definers[i].Role != "authenticated" {
				return ErrDefinitionMismatch
			}
		}
	} else if len(unreviewedDefiners) != 0 {
		return ErrDefinitionMismatch
	}
	for i, r := range actual.Roles {
		if r.Name != []string{"anon", "authenticated"}[i] || r.Superuser || r.CreateRole || r.CreateDatabase || r.BypassRLS || len(r.CreateSchemas) != 0 || len(r.Membership) != 0 {
			return ErrDefinitionMismatch
		}
	}
	expected := []struct {
		name, types string
		arguments   []string
	}{
		{"order_admin_report_counts_v1", "1700", []string{"p_owner"}},
		{"order_admin_report_export_v1", "1700 25 25 1700 23", []string{"p_owner", "p_from_hex", "p_to_hex", "p_after", "p_limit"}},
		{"order_admin_report_get_v1", "1700 1700", []string{"p_owner", "p_order"}},
		{"order_admin_report_payments_v1", "1700 25 25 1700 23", []string{"p_owner", "p_from_hex", "p_to_hex", "p_after", "p_limit"}},
		{"order_admin_report_pending_v1", "1700 1700 23", []string{"p_owner", "p_after", "p_limit"}},
		{"order_admin_report_refunds_v1", "1700 25 25 1700 23", []string{"p_owner", "p_from_hex", "p_to_hex", "p_after", "p_limit"}},
		{"order_admin_report_search_v1", "1700 25 25 25 16 1700 23", []string{"p_owner", "p_state_hex", "p_date_hex", "p_query_hex", "p_unclaimed", "p_after", "p_limit"}},
		{"order_admin_report_stats_v1", "1700 3802", []string{"p_owner", "p_at"}},
		{"order_admin_report_summary_v1", "1700 25 25", []string{"p_owner", "p_from_hex", "p_to_hex"}},
		{"order_audit_search_v1", "1700 25 25 25 25 1700 23", []string{"p_owner", "p_action_hex", "p_target_hex", "p_from_hex", "p_to_hex", "p_after", "p_limit"}},
		{"order_billing_completed_v1", "25 25 25 25", []string{"p_date", "p_environment", "p_appid", "p_merchant"}},
		{"order_billing_ownership_facts_v1", "25 3802 1700", []string{"p_environment", "p_entries", "p_evidence_cutoff"}},
		{"order_billing_payment_facts_v1", "3802 3802", []string{"p_from", "p_to"}},
		{"order_billing_payments_v1", "1700 3802 3802 1700 23", []string{"p_owner", "p_from", "p_to", "p_after", "p_limit"}},
		{"order_billing_reconcile_v1", "25 25 1700 3802 3802", []string{"p_date", "p_digest_hex", "p_matched", "p_provider_only", "p_system_only"}},
		{"order_billing_reconcile_v2", "25 25 1700 3802 3802 25 3802", []string{"p_date", "p_digest_hex", "p_matched", "p_provider_only", "p_system_only", "p_environment", "p_evidence"}},
		{"order_billing_refund_facts_v1", "3802 3802", []string{"p_from", "p_to"}},
		{"order_billing_refunds_v1", "1700 3802 3802 1700 23", []string{"p_owner", "p_from", "p_to", "p_after", "p_limit"}},
		{"order_billing_summary_v1", "1700 3802 3802", []string{"p_owner", "p_from", "p_to"}},
		{"order_catalog_browse_v1", "", []string{}},
		{"order_catalog_categories_v1", "", []string{}},
		{"order_catalog_command_v1", "1700 25 1700 1700 16 25 25 20 25 3802 16 16 25 16 1231 25 25 25 25 1184", []string{"p_user", "p_kind", "p_category", "p_product", "p_name_present", "p_name_hex", "p_description_hex", "p_price_cents", "p_meal_period_hex", "p_images", "p_enabled", "p_listed", "p_service_date_hex", "p_sold_out", "p_ordered_ids", "p_confirm_status", "p_operation_hash", "p_request_id_hash", "p_request_digest", "p_receipt_at"}},
		{"order_catalog_detail_v1", "1700 1082", []string{"p_product", "p_service_date"}},
		{"order_catalog_product_v1", "1700 1082", []string{"p_product", "p_service_date"}},
		{"order_catalog_products_v1", "1082", []string{"p_service_date"}},
		{"order_display_pricing_facts_v1", "1700", []string{"p_user"}},
		{"order_fulfillment_proof_v1", "1700 25 1700 25 25 1082 16 25", []string{"p_actor", "p_kind", "p_order", "p_key", "p_request", "p_locator_date", "p_locator_live", "p_token_hash"}},
		{"order_identity_extra_phone_set_v1", "1700 25 25 25 25 25 25", []string{"p_user", "p_phone_hex", "p_name", "p_name_key_hex", "p_operation_hash", "p_request_hash", "p_request_digest"}},
		{"order_identity_phone_bind_v1", "1700 25 1184", []string{"p_user", "p_phone_hex", "p_bound_at"}},
		{"order_identity_phone_read_v1", "1700", []string{"p_user"}},
		{"order_identity_projection_v1", "1700", []string{"p_user"}},
		{"order_identity_session_create_v1", "25 25 1184 1184", []string{"p_openid_hex", "p_token_hash", "p_issued_at", "p_expires_at"}},
		{"order_identity_session_find_v1", "25 1184", []string{"p_token_hash", "p_now"}},
		{"order_identity_session_revoke_v1", "25", []string{"p_token_hash"}},
		{"order_image_confirmation_commit_v1", "1700 25 25 25 25 25 25 1184", []string{"p_actor", "p_object_key", "p_key_hash", "p_request_hash", "p_request_digest", "p_published_key", "p_version_id", "p_at"}},
		{"order_image_confirmation_replay_v1", "1700 25", []string{"p_actor", "p_key_hash"}},
		{"order_image_confirmed_v1", "1009", []string{"p_target_key_hashes"}},
		{"order_image_versions_v1", "1009", []string{"p_object_keys"}},
		{"order_import_commit_v1", "1700 25 16 25 25 25 25 3802 1184 1184", []string{"p_user", "p_token_hash", "p_skip", "p_operation_hash", "p_request_id_hash", "p_request_digest", "p_plan_digest", "p_material", "p_observed_at", "p_receipt_at"}},
		{"order_import_preview_facts_v1", "25", []string{"p_kind"}},
		{"order_import_preview_v1", "1700 25 25 1700 1700 25 25 25 1184 25 25 25 1184", []string{"p_user", "p_kind", "p_file_digest", "p_file_size", "p_row_count", "p_token_hash", "p_plan_json", "p_response_json", "p_captured_at", "p_operation_hash", "p_request_id_hash", "p_request_digest", "p_receipt_at"}},
		{"order_import_result_v1", "1700 1700", []string{"p_user", "p_batch"}},
		{"order_mark_ready_v1", "1700 1700 25 25 1184 1184 25 3802 3802 1184", []string{"p_actor", "p_order", "p_key", "p_request_id", "p_now", "p_enqueue_at", "p_request", "p_proof", "p_token_material", "p_receipt_at"}},
		{"order_menu_pickup_facts_v1", "1082 1082", []string{"p_date_a", "p_date_b"}},
		{"order_menu_snapshot_v1", "1082", []string{"p_service_date"}},
		{"order_merchant_account_command_v1", "1700 25 1700 25 25 25 16 25 25 25 1184 25", []string{"p_actor", "p_kind", "p_account", "p_name", "p_phone", "p_role", "p_enabled", "p_operation_hash", "p_request_hash", "p_request_digest", "p_receipt_at", "p_input_status"}},
		{"order_merchant_account_current_v1", "1700", []string{"p_user"}},
		{"order_merchant_account_list_v1", "1700 25", []string{"p_user", "p_query"}},
		{"order_merchant_login_complete_v1", "1700 25 25 25 1184", []string{"p_user", "p_phone_hex", "p_code_hash", "p_request_hash", "p_at"}},
		{"order_merchant_login_recover_v1", "1700 25 25 1184 1184", []string{"p_user", "p_code_hash", "p_request_hash", "p_started_at", "p_at"}},
		{"order_merchant_login_start_v1", "1700 25 25 1184", []string{"p_user", "p_code_hash", "p_request_hash", "p_at"}},
		{"order_merchant_soldout_command_v1", "1700 1700 1082 16 1184 16 1700 25 25 25 1184", []string{"p_user", "p_product", "p_service_date", "p_sold_out", "p_now", "p_locator_found", "p_category", "p_operation_hash", "p_request_id_hash", "p_request_digest", "p_receipt_at"}},
		{"order_merchant_soldout_replay_v1", "1700 1700 1082 16 25 25 25", []string{"p_user", "p_product", "p_service_date", "p_sold_out", "p_operation_hash", "p_request_id_hash", "p_request_digest"}},
		{"order_payment_claim_query_v1", "1700 1184 16 25 1700", []string{"p_prepayment", "p_cutoff", "p_force", "p_owner_hex", "p_lease_ns"}},
		{"order_payment_due_v1", "1184 23", []string{"p_now", "p_limit"}},
		{"order_payment_existing_order_v1", "1700", []string{"p_prepayment"}},
		{"order_payment_finish_create_v1", "1700 25 1700 1184 1184 3802", []string{"p_prepayment", "p_owner_hex", "p_version", "p_now", "p_next", "p_result"}},
		{"order_payment_finish_query_v1", "1700 25 1700 1184 1184 16", []string{"p_prepayment", "p_owner_hex", "p_version", "p_now", "p_next", "p_close"}},
		{"order_payment_materialize_v1", "1700 16 1700", []string{"p_prepayment", "p_manual", "p_owner"}},
		{"order_payment_observe_v1", "1700 25 25 3802 3802 1184 1700", []string{"p_expected_prepayment", "p_source", "p_event_hex", "p_transaction", "p_claim", "p_received_at", "p_reconcile_ns"}},
		{"order_payment_openid_v1", "1700", []string{"p_user"}},
		{"order_payment_owner_v1", "1700", []string{"p_user"}},
		{"order_payment_pending_v1", "25 1700 23", []string{"p_reason_hex", "p_after", "p_limit"}},
		{"order_payment_prepare_replay_v1", "1700 1700 25", []string{"p_user", "p_quote", "p_key_hash"}},
		{"order_payment_prepare_v1", "1700 1700 25 25 25 1184 1700 3802", []string{"p_user", "p_quote", "p_key_hash", "p_openid_hex", "p_owner_hex", "p_now", "p_lease_ns", "p_config"}},
		{"order_payment_prepare_v2", "1700 1700 25 25 25 1184 1700 3802", []string{"p_user", "p_quote", "p_key_hash", "p_openid_hex", "p_owner_hex", "p_now", "p_lease_ns", "p_config"}},
		{"order_payment_read_v1", "1700", []string{"p_prepayment"}},
		{"order_payment_receipt_append_v1", "1700 25 1700 25 25 25 1184", []string{"p_user", "p_action", "p_target", "p_key_hash", "p_request_hash", "p_response_json", "p_at"}},
		{"order_payment_receipt_read_v1", "1700 25 25", []string{"p_user", "p_action", "p_key_hash"}},
		{"order_pc_login_approval_read_v1", "1700 25", []string{"p_login", "p_approval_hash"}},
		{"order_pc_login_approve_v1", "1700 1700 25", []string{"p_user", "p_login", "p_approval_hash"}},
		{"order_pc_login_begin_v1", "25 25 1184", []string{"p_approval_hash", "p_poll_hash", "p_now"}},
		{"order_pc_login_poll_v1", "1700 25 1184 25 16", []string{"p_login", "p_poll_hash", "p_now", "p_access_hash", "p_entropy_failed"}},
		{"order_pc_session_authenticate_v1", "25 1184", []string{"p_token_hash", "p_now"}},
		{"order_pc_session_revoke_v1", "25", []string{"p_token_hash"}},
		{"order_production_advance_v1", "1700 3802", []string{"p_order", "p_now"}},
		{"order_production_due_v1", "3802 23", []string{"p_now", "p_limit"}},
		{"order_query_get_merchant_v1", "1700 1700 25", []string{"p_user", "p_order", "p_state"}},
		{"order_query_get_user_v1", "1700 1700", []string{"p_user", "p_order"}},
		{"order_query_list_user_v1", "1700 25 16 1700 23", []string{"p_user", "p_state", "p_active", "p_after", "p_limit"}},
		{"order_query_search_merchant_v1", "1700 25 25 25 1700 23", []string{"p_user", "p_state", "p_date", "p_search", "p_after", "p_limit"}},
		{"order_quote_create_v1", "1700 25 25 25 25 3802 1184 1184", []string{"p_user", "p_key_hash", "p_request_digest", "p_operation_hash", "p_request_id_hash", "p_input", "p_captured_at", "p_receipt_at"}},
		{"order_quote_read_v1", "1700 1700", []string{"p_user", "p_quote"}},
		{"order_redeem_current_date_code_v1", "1700 1700 25 25 1184 25 16 1082 3802 1184", []string{"p_actor", "p_order", "p_key", "p_request_id", "p_now", "p_request", "p_locator_live", "p_locator_date", "p_proof", "p_receipt_at"}},
		{"order_redeem_order_proof_v1", "1700 1700 25", []string{"p_actor", "p_order", "p_key"}},
		{"order_redeem_order_v1", "1700 1700 25 25 1184 3802", []string{"p_actor", "p_order", "p_key", "p_request_id", "p_now", "p_proof"}},
		{"order_redeem_token_v1", "1700 1700 25 25 1184 25 16 1082 3802 25 1184", []string{"p_actor", "p_order", "p_key", "p_request_id", "p_now", "p_request", "p_locator_live", "p_locator_date", "p_proof", "p_token_hash", "p_receipt_at"}},
		{"order_refund_claim_create_v1", "1700 25 3802", []string{"p_refund", "p_owner_hex", "p_now"}},
		{"order_refund_claim_query_v1", "1700 25 3802 3802", []string{"p_refund", "p_owner_hex", "p_cutoff", "p_now"}},
		{"order_refund_due_v1", "3802 3802 23", []string{"p_cutoff", "p_lease_now", "p_limit"}},
		{"order_refund_finish_create_v1", "1700 25 1700 3802 16 3802 3802", []string{"p_refund", "p_owner_hex", "p_version", "p_request", "p_provider_error", "p_observed", "p_now"}},
		{"order_refund_observation_locator_v1", "25", []string{"p_out_refund_no_hex"}},
		{"order_refund_observe_v1", "3802 25 25 3802 25 3802 3802 16", []string{"p_locator", "p_source", "p_event_hex", "p_observed", "p_dedupe_hash", "p_claim", "p_now", "p_enqueue"}},
		{"order_refund_order_locator_v1", "1700", []string{"p_order"}},
		{"order_refund_pending_v1", "1700 1700 23", []string{"p_owner", "p_after", "p_limit"}},
		{"order_refund_read_v1", "1700", []string{"p_refund"}},
		{"order_refund_release_query_v1", "1700 25 1700 3802", []string{"p_refund", "p_owner_hex", "p_version", "p_now"}},
		{"order_refund_request_order_v1", "25 1700 3802 25 25 25 3802 25", []string{"p_actor_kind", "p_actor", "p_locator", "p_reason_hex", "p_key_hash", "p_request_hash", "p_now", "p_notify_url_hex"}},
		{"order_refund_request_order_v2", "25 1700 3802 25 25 25 3802 25 25", []string{"p_actor_kind", "p_actor", "p_locator", "p_reason_hex", "p_key_hash", "p_request_hash", "p_now", "p_notify_url_hex", "p_environment"}},
		{"order_refund_request_paid_v1", "1700 1700 25 25 25 3802 25", []string{"p_actor", "p_prepayment", "p_reason_hex", "p_key_hash", "p_request_hash", "p_now", "p_notify_url_hex"}},
		{"order_refund_request_paid_v2", "1700 1700 25 25 25 3802 25 25", []string{"p_actor", "p_prepayment", "p_reason_hex", "p_key_hash", "p_request_hash", "p_now", "p_notify_url_hex", "p_environment"}},
		{"order_shared_pg_readiness_v1", "", []string{}},
		{"order_staff_admin_command_v1", "1700 25 1700 25 25 25 16 23 25 25 25 25 1184", []string{"p_user", "p_kind", "p_staff", "p_name_hex", "p_name_key_hex", "p_phone_hex", "p_enabled", "p_rate", "p_input_status", "p_operation_hash", "p_request_id_hash", "p_request_digest", "p_receipt_at"}},
		{"order_staff_admin_list_v1", "25", []string{"p_query_hex"}},
		{"order_staff_admin_rate_v1", "", []string{}},
		{"order_store_status_apply_v1", "1700 25 25 25 25 3802", []string{"p_actor", "p_desired", "p_operation_hash", "p_request_hash", "p_request_digest", "p_now"}},
		{"order_storefront_admin_v1", "", []string{}},
		{"order_storefront_configure_v1", "1700 25 25 25 23 3802 3802 25 25 25 1184", []string{"p_user", "p_store_status_hex", "p_notice_hex", "p_pickup_point_hex", "p_pickup_step_min", "p_meal_periods", "p_service_dates", "p_operation_hash", "p_request_id_hash", "p_request_digest", "p_receipt_at"}},
		{"order_storefront_get_v1", "", []string{}},
		{"order_storefront_launch_configure_v1", "1700 16 16 25 25 25 25 25 25 25 25 25 25 1184", []string{"p_user", "p_layer_present", "p_enabled", "p_image_key_hex", "p_image_key_json_hex", "p_center_x", "p_center_y", "p_width_ratio", "p_aspect_ratio", "p_confirm_status", "p_operation_hash", "p_request_id_hash", "p_request_digest", "p_receipt_at"}},
		{"order_storefront_launch_read_v1", "", []string{}},
		{"order_subscription_claim_v1", "3802 3802 25", []string{"p_cutoff", "p_lease_now", "p_owner_hex"}},
		{"order_subscription_consent_read_v1", "1700 25", []string{"p_user", "p_key_hash"}},
		{"order_subscription_consent_record_v1", "1700 1700 25 25 1700 25 25 3802", []string{"p_user", "p_order", "p_kind", "p_decision", "p_template_version", "p_key_hash", "p_request_hash", "p_now"}},
		{"order_subscription_permanent_failure_v1", "1700 25 1700 25 3802", []string{"p_outbox", "p_owner_hex", "p_version", "p_code", "p_completed_at"}},
		{"order_subscription_sent_v1", "1700 25 1700 25 3802", []string{"p_outbox", "p_owner_hex", "p_version", "p_provider_message_hex", "p_completed_at"}},
		{"order_subscription_temporary_failure_v1", "1700 25 1700 25 3802 3802", []string{"p_outbox", "p_owner_hex", "p_version", "p_code", "p_completed_at", "p_next_at"}},
	}
	owner := actual.Tables[0].Owner
	if owner == "" || owner == "anon" || owner == "authenticated" {
		return ErrDefinitionMismatch
	}
	runtimeCallable := 0
	for i, r := range actual.Functions {
		executable := !slices.Contains([]string{"order_payment_prepare_v1", "order_refund_request_order_v1", "order_refund_request_paid_v1", "order_billing_reconcile_v1"}, r.Name)
		if r.AuthenticatedExecute {
			runtimeCallable++
		}
		body, ok := bodies[r.Name]
		if !ok {
			return ErrDefinitionMismatch
		}
		hash := sha256.Sum256([]byte(body))
		if r.Name != expected[i].name || r.Hash != hex.EncodeToString(hash[:]) || r.Owner != owner || r.SecurityDefiner || r.ReturnsSet || r.Strict || r.Volatility != "v" || r.Parallel != "u" || r.ResultType != "jsonb" || r.ArgumentTypes != expected[i].types || r.Language != "plpgsql" || r.Kind != "f" || r.Defaults != 0 || r.Variadic != 0 || !slices.Equal(r.ArgumentNames, expected[i].arguments) || len(r.Config) != 2 || !slices.Contains(r.Config, "search_path=pg_catalog, public") || !slices.Contains(r.Config, "TimeZone=UTC") || r.PublicExecute || r.AnonExecute || r.AuthenticatedExecute != executable {
			return ErrDefinitionMismatch
		}
	}
	if runtimeCallable != 113 {
		return ErrDefinitionMismatch
	}
	tableNames := []string{"action_audits", "categories", "discount_settings", "import_batches", "meal_periods", "merchant_accounts", "merchant_pc_sessions", "miniprogram_sessions", "miniprogram_users", "notification_consents", "notification_outbox", "order_items", "orders", "payment_observations", "pickup_sequences", "prepayments", "product_sold_out_dates", "products", "quote_items", "quotes", "refund_observations", "refunds", "schema_migrations", "service_dates", "staff_whitelist", "storefront_settings"}
	privileges := [][]string{{"INSERT", "SELECT"}, {"DELETE", "SELECT"}, {"SELECT"}, {"SELECT"}, {"SELECT"}, {"SELECT"}, {"DELETE", "SELECT"}, {"DELETE", "SELECT"}, {"SELECT"}, {"SELECT"}, {"INSERT", "SELECT"}, {"SELECT"}, {"SELECT"}, {"SELECT"}, {"SELECT"}, {"SELECT"}, {"DELETE", "SELECT"}, {"DELETE", "SELECT"}, {"SELECT"}, {"SELECT"}, {"SELECT"}, {"SELECT"}, {"SELECT"}, {"SELECT"}, {"DELETE", "SELECT"}, {"SELECT"}}
	columns := [][]string{{"id", "response_json"}, {"is_active", "name", "name_key", "record_version", "sort_order"}, {"discount_version", "rate_percent", "updated_at", "whitelist_version"}, {"commit_idempotency_key_hash", "committed_at", "preview_json", "record_version", "result_json", "skip_invalid", "state", "updated_at"}, {"cutoff_time", "interval_minutes", "pickup_end_time", "pickup_start_time"}, {"auth_version", "bound_at", "bound_user_id", "deleted_at", "deleted_by_account_id", "enabled", "name", "phone", "record_version", "role", "updated_at", "updated_by"}, {"access_expires_at", "access_issued_at", "access_token_hash", "approved_account_id", "approved_at", "approved_auth_version", "approved_user_id", "consumed_at", "state", "updated_at"}, {}, {"extra_name", "extra_name_key", "extra_phone", "extra_phone_set_at", "last_login_at", "primary_phone", "primary_phone_bound_at", "record_version"}, {"consumed_at"}, {"attempt_count", "id", "last_error_code", "lease_expires_at", "lease_owner", "next_attempt_at", "provider_message_id", "record_version", "sent_at", "state"}, {}, {"completed_at", "preparing_at", "ready_at", "record_version", "redeemed_at", "redeemed_by_account_id", "redemption_issued_at", "redemption_key_version", "redemption_token_ciphertext", "redemption_token_hash", "refunded_at", "refunding_at", "state", "updated_at"}, {"applied_at", "apply_reason_code", "apply_state", "record_version"}, {"last_number", "updated_at"}, {"create_attempted_at", "last_queried_at", "lease_expires_at", "lease_kind", "lease_owner", "materialization_state", "materialized_at", "next_reconcile_at", "pending_reason_code", "provider_prepay_id", "provider_state", "record_version", "updated_at", "wx_request_payment_json"}, {"product_id"}, {"category_id", "description", "images_json", "is_listed", "meal_period", "name", "name_key", "price_cents", "record_version", "sort_order"}, {"quote_id"}, {"id"}, {"applied_at", "apply_reason_code", "apply_state", "record_version"}, {"create_attempted_at", "lease_expires_at", "lease_kind", "lease_owner", "materialization_state", "materialized_at", "next_query_at", "pending_reason_code", "provider_refund_id", "provider_state", "record_version", "updated_at"}, {}, {"is_open", "record_version", "updated_at", "updated_by_account_id"}, {"enabled", "name", "name_key", "phone", "record_version", "updated_at"}, {"announcement", "aspect_ratio", "business_status", "center_x", "center_y", "launch_image_object_key", "pickup_point", "record_version", "width_ratio"}}
	insertColumns := [][]string{{"action", "actor_account_id", "actor_account_id_snapshot", "actor_auth_version_snapshot", "actor_kind", "actor_role_snapshot", "actor_scope_hash", "actor_user_id", "after_state_json", "before_state_json", "entry_kind", "id", "occurred_at", "operation_key_hash", "reason_code", "request_id_hash", "response_json", "result", "target_id", "target_key_hash", "target_type"}, {"is_active", "name", "name_key", "record_version", "sort_order"}, {}, {"actor_account_id", "created_at", "error_count", "expires_at", "file_digest", "file_object_key", "file_size_bytes", "kind", "new_count", "preview_json", "preview_token_hash", "record_version", "row_count", "state", "update_count", "updated_at"}, {}, {"auth_version", "bound_at", "bound_user_id", "created_at", "created_by", "deleted_at", "deleted_by_account_id", "enabled", "name", "phone", "record_version", "role", "updated_at", "updated_by"}, {"approval_secret_hash", "created_at", "login_expires_at", "poll_secret_hash", "state", "updated_at"}, {"expires_at", "issued_at", "token_hash", "user_id"}, {"created_at", "last_login_at", "openid"}, {"consumed_at", "decided_at", "decision", "grant_sequence", "idempotency_key_hash", "kind", "order_id", "template_config_version", "user_id"}, {"attempt_count", "consent_id", "created_at", "id", "immutable_message_json", "kind", "last_error_code", "lease_expires_at", "lease_owner", "next_attempt_at", "order_id", "provider_message_id", "recipient_user_id", "record_version", "sent_at", "state", "template_config_version"}, {"discounted_unit_price_cents", "flavors_json", "image_object_key_snapshot", "line_note", "line_number", "order_id", "original_subtotal_cents", "original_unit_price_cents", "payable_subtotal_cents", "product_id", "product_name_snapshot", "product_source_version", "quantity"}, {"completed_at", "contact_name_snapshot", "contact_phone_snapshot", "created_at", "discount_cents", "discount_rate_percent", "discount_version", "identity_kind", "identity_source_version", "item_count", "materialized_at", "meal_period", "order_no", "order_note", "original_subtotal_cents", "paid_at", "payable_cents", "payment_observation_id", "pickup_at", "pickup_date", "pickup_number", "pickup_point_snapshot", "pickup_time", "preparing_at", "prepayment_id", "quote_id", "ready_at", "record_version", "redeemed_at", "redeemed_by_account_id", "redemption_issued_at", "redemption_key_version", "redemption_token_ciphertext", "redemption_token_hash", "refunded_at", "refunding_at", "snapshot_digest", "state", "store_address_snapshot", "store_name_snapshot", "transaction_id", "updated_at", "user_id"}, {"amount_cents", "applied_at", "apply_reason_code", "apply_state", "currency", "dedupe_key", "materialization_mode", "mismatch_code", "out_trade_no", "prepayment_id", "provider_event_id", "provider_state", "received_at", "record_version", "source", "success_time", "transaction_id", "validation"}, {"last_number", "service_date", "updated_at"}, {"create_attempted_at", "created_at", "currency", "effective_deadline", "expected_amount_cents", "expected_appid", "expected_mchid", "idempotency_key_hash", "last_queried_at", "lease_expires_at", "lease_kind", "lease_owner", "materialization_state", "materialized_at", "next_reconcile_at", "out_trade_no", "pending_reason_code", "provider_create_request_digest", "provider_create_request_json", "provider_prepay_id", "provider_state", "quote_id", "record_version", "updated_at", "user_id", "wx_request_payment_json"}, {"product_id", "service_date"}, {"category_id", "description", "images_json", "is_listed", "meal_period", "name", "name_key", "price_cents", "record_version", "sort_order", "specification"}, {"discounted_unit_price_cents", "flavors_json", "image_object_key_snapshot", "line_note", "line_number", "original_subtotal_cents", "original_unit_price_cents", "payable_subtotal_cents", "product_id", "product_name_snapshot", "product_source_version", "quantity", "quote_id"}, {"contact_name_snapshot", "contact_phone_snapshot", "created_at", "discount_cents", "discount_rate_percent", "discount_version", "expires_at", "idempotency_key_hash", "identity_kind", "identity_source_version", "item_count", "meal_period", "order_note", "original_subtotal_cents", "payable_cents", "pickup_date", "pickup_point_snapshot", "pickup_time", "request_digest", "snapshot_digest", "store_address_snapshot", "store_name_snapshot", "user_id"}, {"applied_at", "apply_reason_code", "apply_state", "dedupe_key", "mismatch_code", "out_refund_no", "provider_event_id", "provider_refund_id", "provider_state", "received_at", "record_version", "refund_id", "source", "success_time", "validation"}, {"amount_cents", "create_attempted_at", "created_at", "currency", "idempotency_key_hash", "lease_expires_at", "lease_kind", "lease_owner", "materialization_state", "materialized_at", "next_query_at", "order_id", "out_refund_no", "pending_reason_code", "prepayment_id", "provider_refund_id", "provider_request_digest", "provider_request_json", "provider_state", "reason_code", "record_version", "requested_by_account_id", "requested_by_user_id", "updated_at"}, {}, {"is_open", "record_version", "service_date", "updated_at", "updated_by_account_id"}, {"created_at", "enabled", "name", "name_key", "phone", "record_version", "updated_at"}, {}}
	expression := "((CURRENT_USER = 'authenticated'::name) AND (auth.uid() = '" + serviceUID + "'::text))"
	for i, t := range actual.Tables {
		if t.Name != tableNames[i] || !t.RLS || t.Owner != owner || len(t.PublicPrivileges) != 0 || len(t.AnonPrivileges) != 0 || t.AnonColumnAccess || t.PublicColumnAccess || t.AuthenticatedReferenceColumns || t.AuthenticatedInsertColumns != (len(insertColumns[i]) > 0) || !slices.Equal(t.AuthenticatedInsertColumnNames, insertColumns[i]) || !slices.Equal(t.AuthenticatedPrivileges, privileges[i]) || !slices.Equal(t.AuthenticatedUpdateColumns, columns[i]) || len(t.Policies) != 1 {
			return ErrDefinitionMismatch
		}
		p := t.Policies[0]
		if p.Name != "order_runtime_identity_v1" || !p.Permissive || p.Command != "*" || !slices.Equal(p.Roles, []string{"authenticated"}) || p.Using != expression || p.Check != expression {
			return ErrDefinitionMismatch
		}
	}
	for i, sequence := range actual.Sequences {
		if sequence.Name != []string{"categories_id_seq", "import_batches_id_seq", "merchant_accounts_id_seq", "merchant_action_audits_id_seq", "merchant_pc_sessions_id_seq", "miniprogram_users_id_seq", "notification_consents_id_seq", "notification_outbox_id_seq", "orders_id_seq", "payment_observations_id_seq", "prepayments_id_seq", "products_id_seq", "quotes_id_seq", "refund_observations_id_seq", "refunds_id_seq", "staff_whitelist_id_seq"}[i] || sequence.Owner != owner || len(sequence.PublicPrivileges) != 0 || len(sequence.AnonPrivileges) != 0 || !slices.Equal(sequence.AuthenticatedPrivileges, []string{"USAGE"}) {
			return ErrDefinitionMismatch
		}
	}
	if migrate.VerifySharedPostgreSQLCatalogReadiness(actual.Schema) != nil {
		return ErrDefinitionMismatch
	}
	return nil
}
