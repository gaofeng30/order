-- Version-pinned image RPCs reuse the existing reviewed relation privileges.
REVOKE ALL ON FUNCTION public.order_image_confirmation_commit_v1(numeric,text,text,text,text,text,text,timestamptz),public.order_image_confirmation_replay_v1(numeric,text),public.order_image_versions_v1(text[]) FROM PUBLIC,anon,authenticated;
-- Eighth fixed read/report entries reuse the exact existing v44 table and sequence grants.
REVOKE ALL ON FUNCTION public.order_billing_summary_v1(numeric,jsonb,jsonb),public.order_billing_payments_v1(numeric,jsonb,jsonb,numeric,integer),public.order_billing_refunds_v1(numeric,jsonb,jsonb,numeric,integer),public.order_billing_payment_facts_v1(jsonb,jsonb),public.order_billing_refund_facts_v1(jsonb,jsonb),public.order_billing_reconcile_v1(text,text,numeric,jsonb,jsonb),public.order_admin_report_get_v1(numeric,numeric),public.order_admin_report_search_v1(numeric,text,text,text,boolean,numeric,integer),public.order_admin_report_stats_v1(numeric,jsonb),public.order_admin_report_payments_v1(numeric,text,text,numeric,integer),public.order_admin_report_refunds_v1(numeric,text,text,numeric,integer),public.order_admin_report_summary_v1(numeric,text,text),public.order_admin_report_pending_v1(numeric,numeric,integer),public.order_audit_search_v1(numeric,text,text,text,text,numeric,integer),public.order_store_status_apply_v1(numeric,text,text,text,text,jsonb) FROM PUBLIC,anon,authenticated;
-- Seventh-slice fixed refund, subscription, production privileges.
REVOKE ALL ON FUNCTION public.order_refund_order_locator_v1(numeric),public.order_refund_request_order_v1(text,numeric,jsonb,text,text,text,jsonb,text),public.order_refund_request_paid_v1(numeric,numeric,text,text,text,jsonb,text),public.order_refund_claim_create_v1(numeric,text,jsonb),public.order_refund_finish_create_v1(numeric,text,numeric,jsonb,boolean,jsonb,jsonb),public.order_refund_read_v1(numeric),public.order_refund_observation_locator_v1(text),public.order_refund_observe_v1(jsonb,text,text,jsonb,text,jsonb,jsonb,boolean),public.order_refund_due_v1(jsonb,jsonb,integer),public.order_refund_claim_query_v1(numeric,text,jsonb,jsonb),public.order_refund_release_query_v1(numeric,text,numeric,jsonb),public.order_refund_pending_v1(numeric,numeric,integer),public.order_subscription_consent_read_v1(numeric,text),public.order_subscription_consent_record_v1(numeric,numeric,text,text,numeric,text,text,jsonb),public.order_subscription_claim_v1(jsonb,jsonb,text),public.order_subscription_sent_v1(numeric,text,numeric,text,jsonb),public.order_subscription_temporary_failure_v1(numeric,text,numeric,text,jsonb,jsonb),public.order_subscription_permanent_failure_v1(numeric,text,numeric,text,jsonb),public.order_production_due_v1(jsonb,integer),public.order_production_advance_v1(numeric,jsonb) FROM PUBLIC,anon,authenticated;
-- Sixth-slice fixed payment function privileges.
REVOKE ALL ON FUNCTION public.order_payment_receipt_read_v1(numeric,text,text),public.order_payment_prepare_replay_v1(numeric,numeric,text),public.order_payment_openid_v1(numeric),public.order_payment_prepare_v1(numeric,numeric,text,text,text,timestamptz,numeric,jsonb),public.order_payment_finish_create_v1(numeric,text,numeric,timestamptz,timestamptz,jsonb),public.order_payment_read_v1(numeric),public.order_payment_existing_order_v1(numeric),public.order_payment_due_v1(timestamptz,integer),public.order_payment_claim_query_v1(numeric,timestamptz,boolean,text,numeric),public.order_payment_finish_query_v1(numeric,text,numeric,timestamptz,timestamptz,boolean),public.order_payment_observe_v1(numeric,text,text,jsonb,jsonb,timestamptz,numeric),public.order_payment_materialize_v1(numeric,boolean,numeric),public.order_payment_pending_v1(text,numeric,integer),public.order_payment_receipt_append_v1(numeric,text,numeric,text,text,text,timestamptz),public.order_payment_owner_v1(numeric) FROM PUBLIC,anon,authenticated;
-- Fifth-slice exact function privileges.
REVOKE ALL ON FUNCTION public.order_import_commit_v1(numeric,text,boolean,text,text,text,text,jsonb,timestamptz,timestamptz),public.order_import_preview_facts_v1(text),public.order_import_preview_v1(numeric,text,text,numeric,numeric,text,text,text,timestamptz,text,text,text,timestamptz),public.order_staff_admin_command_v1(numeric,text,numeric,text,text,text,boolean,integer,text,text,text,text,timestamptz),public.order_staff_admin_list_v1(text),public.order_staff_admin_rate_v1() FROM PUBLIC,anon,authenticated;
-- Fourth-slice exact function privileges.
REVOKE ALL ON FUNCTION public.order_catalog_browse_v1(),public.order_catalog_categories_v1(),public.order_catalog_command_v1(numeric,text,numeric,numeric,boolean,text,text,bigint,text,jsonb,boolean,boolean,text,boolean,numeric[],text,text,text,text,timestamptz),public.order_catalog_detail_v1(numeric,date),public.order_catalog_product_v1(numeric,date),public.order_catalog_products_v1(date),public.order_display_pricing_facts_v1(numeric),public.order_image_confirmed_v1(text[]),public.order_menu_pickup_facts_v1(date,date),public.order_menu_snapshot_v1(date),public.order_merchant_soldout_command_v1(numeric,numeric,date,boolean,timestamptz,boolean,numeric,text,text,text,timestamptz),public.order_merchant_soldout_replay_v1(numeric,numeric,date,boolean,text,text,text),public.order_quote_create_v1(numeric,text,text,text,text,jsonb,timestamptz,timestamptz),public.order_quote_read_v1(numeric,numeric),public.order_storefront_admin_v1(),public.order_storefront_configure_v1(numeric,text,text,text,integer,jsonb,jsonb,text,text,text,timestamptz),public.order_storefront_get_v1(),public.order_storefront_launch_configure_v1(numeric,boolean,boolean,text,text,text,text,text,text,text,text,text,text,timestamptz),public.order_storefront_launch_read_v1() FROM PUBLIC,anon,authenticated;
-- Fixed minimum privileges for the reviewed service UID.
REVOKE ALL ON FUNCTION public.order_fulfillment_proof_v1(numeric,text,numeric,text,text,date,boolean,text),public.order_identity_projection_v1(numeric),public.order_mark_ready_v1(numeric,numeric,text,text,timestamptz,timestamptz,text,jsonb,jsonb,timestamptz),public.order_query_get_merchant_v1(numeric,numeric,text),public.order_query_get_user_v1(numeric,numeric),public.order_query_list_user_v1(numeric,text,boolean,numeric,integer),public.order_query_search_merchant_v1(numeric,text,text,text,numeric,integer),public.order_redeem_current_date_code_v1(numeric,numeric,text,text,timestamptz,text,boolean,date,jsonb,timestamptz),public.order_redeem_order_proof_v1(numeric,numeric,text),public.order_redeem_order_v1(numeric,numeric,text,text,timestamptz,jsonb),public.order_redeem_token_v1(numeric,numeric,text,text,timestamptz,text,boolean,date,jsonb,text,timestamptz),public.order_shared_pg_readiness_v1(),public.order_identity_session_create_v1(text,text,timestamptz,timestamptz),public.order_identity_session_find_v1(text,timestamptz),public.order_identity_session_revoke_v1(text),public.order_identity_phone_read_v1(numeric),public.order_identity_phone_bind_v1(numeric,text,timestamptz),public.order_merchant_login_start_v1(numeric,text,text,timestamptz),public.order_merchant_login_complete_v1(numeric,text,text,text,timestamptz),public.order_merchant_login_recover_v1(numeric,text,text,timestamptz,timestamptz),public.order_identity_extra_phone_set_v1(numeric,text,text,text,text,text,text),public.order_merchant_account_current_v1(numeric),public.order_merchant_account_list_v1(numeric,text),public.order_merchant_account_command_v1(numeric,text,numeric,text,text,text,boolean,text,text,text,timestamptz,text),public.order_pc_login_begin_v1(text,text,timestamptz),public.order_pc_login_approval_read_v1(numeric,text),public.order_pc_login_approve_v1(numeric,numeric,text),public.order_pc_login_poll_v1(numeric,text,timestamptz,text,boolean),public.order_pc_session_authenticate_v1(text,timestamptz),public.order_pc_session_revoke_v1(text) FROM PUBLIC,anon,authenticated;
REVOKE ALL ON public.action_audits,public.categories,public.discount_settings,public.import_batches,public.meal_periods,public.merchant_accounts,public.merchant_pc_sessions,public.miniprogram_sessions,public.miniprogram_users,public.notification_consents,public.notification_outbox,public.order_items,public.orders,public.payment_observations,public.pickup_sequences,public.prepayments,public.product_sold_out_dates,public.products,public.quote_items,public.quotes,public.refund_observations,public.refunds,public.schema_migrations,public.service_dates,public.staff_whitelist,public.storefront_settings FROM PUBLIC,anon,authenticated;
REVOKE ALL ON SEQUENCE public.categories_id_seq,public.import_batches_id_seq,public.merchant_accounts_id_seq,public.merchant_action_audits_id_seq,public.merchant_pc_sessions_id_seq,public.miniprogram_users_id_seq,public.notification_consents_id_seq,public.notification_outbox_id_seq,public.orders_id_seq,public.payment_observations_id_seq,public.prepayments_id_seq,public.products_id_seq,public.quotes_id_seq,public.refund_observations_id_seq,public.refunds_id_seq,public.staff_whitelist_id_seq FROM PUBLIC,anon,authenticated;
-- Only known v44 columns are reset; unknown objects and columns are never revoked.
REVOKE ALL(action,actor_account_id,actor_account_id_snapshot,actor_auth_version_snapshot,actor_kind,actor_role_snapshot,actor_scope_hash,actor_user_id,after_state_json,before_state_json,entry_kind,id,occurred_at,operation_key_hash,reason_code,request_id_hash,response_json,result,target_id,target_key_hash,target_type) ON public.action_audits FROM PUBLIC,anon,authenticated;
REVOKE ALL(id,is_active,name,name_key,record_version,sort_order) ON public.categories FROM PUBLIC,anon,authenticated;
REVOKE ALL(discount_version,id,rate_percent,updated_at,whitelist_version) ON public.discount_settings FROM PUBLIC,anon,authenticated;
REVOKE ALL(actor_account_id,commit_idempotency_key_hash,committed_at,created_at,error_count,expires_at,file_digest,file_object_key,file_size_bytes,id,kind,new_count,preview_json,preview_token_hash,record_version,result_json,row_count,skip_invalid,state,update_count,updated_at) ON public.import_batches FROM PUBLIC,anon,authenticated;
REVOKE ALL(code,cutoff_time,interval_minutes,pickup_end_time,pickup_start_time) ON public.meal_periods FROM PUBLIC,anon,authenticated;
REVOKE ALL(auth_version,bound_at,bound_user_id,created_at,created_by,deleted_at,deleted_by_account_id,enabled,id,name,phone,record_version,role,updated_at,updated_by) ON public.merchant_accounts FROM PUBLIC,anon,authenticated;
REVOKE ALL(access_expires_at,access_issued_at,access_token_hash,approval_secret_hash,approved_account_id,approved_at,approved_auth_version,approved_user_id,consumed_at,created_at,id,login_expires_at,poll_secret_hash,state,updated_at) ON public.merchant_pc_sessions FROM PUBLIC,anon,authenticated;
REVOKE ALL(expires_at,issued_at,token_hash,user_id) ON public.miniprogram_sessions FROM PUBLIC,anon,authenticated;
REVOKE ALL(created_at,extra_name,extra_name_key,extra_phone,extra_phone_set_at,id,last_login_at,openid,primary_phone,primary_phone_bound_at,record_version) ON public.miniprogram_users FROM PUBLIC,anon,authenticated;
REVOKE ALL(consumed_at,decided_at,decision,grant_sequence,id,idempotency_key_hash,kind,order_id,template_config_version,user_id) ON public.notification_consents FROM PUBLIC,anon,authenticated;
REVOKE ALL(attempt_count,consent_id,created_at,id,immutable_message_json,kind,last_error_code,lease_expires_at,lease_owner,next_attempt_at,order_id,provider_message_id,recipient_user_id,record_version,sent_at,state,template_config_version) ON public.notification_outbox FROM PUBLIC,anon,authenticated;
REVOKE ALL(discounted_unit_price_cents,flavors_json,image_object_key_snapshot,line_note,line_number,order_id,original_subtotal_cents,original_unit_price_cents,payable_subtotal_cents,product_id,product_name_snapshot,product_source_version,quantity) ON public.order_items FROM PUBLIC,anon,authenticated;
REVOKE ALL(completed_at,contact_name_snapshot,contact_phone_snapshot,created_at,discount_cents,discount_rate_percent,discount_version,id,identity_kind,identity_source_version,item_count,materialized_at,meal_period,order_no,order_note,original_subtotal_cents,paid_at,payable_cents,payment_observation_id,pickup_at,pickup_date,pickup_number,pickup_point_snapshot,pickup_time,preparing_at,prepayment_id,quote_id,ready_at,record_version,redeemed_at,redeemed_by_account_id,redemption_issued_at,redemption_key_version,redemption_token_ciphertext,redemption_token_hash,refunded_at,refunding_at,snapshot_digest,state,store_address_snapshot,store_name_snapshot,transaction_id,updated_at,user_id) ON public.orders FROM PUBLIC,anon,authenticated;
REVOKE ALL(amount_cents,applied_at,apply_reason_code,apply_state,currency,dedupe_key,id,materialization_mode,mismatch_code,out_trade_no,prepayment_id,provider_event_id,provider_state,received_at,record_version,source,success_time,transaction_id,validation) ON public.payment_observations FROM PUBLIC,anon,authenticated;
REVOKE ALL(last_number,service_date,updated_at) ON public.pickup_sequences FROM PUBLIC,anon,authenticated;
REVOKE ALL(create_attempted_at,created_at,currency,effective_deadline,expected_amount_cents,expected_appid,expected_mchid,id,idempotency_key_hash,last_queried_at,lease_expires_at,lease_kind,lease_owner,materialization_state,materialized_at,next_reconcile_at,out_trade_no,pending_reason_code,provider_create_request_digest,provider_create_request_json,provider_prepay_id,provider_state,quote_id,record_version,updated_at,user_id,wx_request_payment_json) ON public.prepayments FROM PUBLIC,anon,authenticated;
REVOKE ALL(product_id,service_date) ON public.product_sold_out_dates FROM PUBLIC,anon,authenticated;
REVOKE ALL(category_id,description,id,images_json,is_listed,meal_period,name,name_key,price_cents,record_version,sort_order,specification) ON public.products FROM PUBLIC,anon,authenticated;
REVOKE ALL(discounted_unit_price_cents,flavors_json,image_object_key_snapshot,line_note,line_number,original_subtotal_cents,original_unit_price_cents,payable_subtotal_cents,product_id,product_name_snapshot,product_source_version,quantity,quote_id) ON public.quote_items FROM PUBLIC,anon,authenticated;
REVOKE ALL(contact_name_snapshot,contact_phone_snapshot,created_at,discount_cents,discount_rate_percent,discount_version,expires_at,id,idempotency_key_hash,identity_kind,identity_source_version,item_count,meal_period,order_note,original_subtotal_cents,payable_cents,pickup_date,pickup_point_snapshot,pickup_time,request_digest,snapshot_digest,store_address_snapshot,store_name_snapshot,user_id) ON public.quotes FROM PUBLIC,anon,authenticated;
REVOKE ALL(applied_at,apply_reason_code,apply_state,dedupe_key,id,mismatch_code,out_refund_no,provider_event_id,provider_refund_id,provider_state,received_at,record_version,refund_id,source,success_time,validation) ON public.refund_observations FROM PUBLIC,anon,authenticated;
REVOKE ALL(amount_cents,create_attempted_at,created_at,currency,id,idempotency_key_hash,lease_expires_at,lease_kind,lease_owner,materialization_state,materialized_at,next_query_at,order_id,out_refund_no,pending_reason_code,prepayment_id,provider_refund_id,provider_request_digest,provider_request_json,provider_state,reason_code,record_version,requested_by_account_id,requested_by_user_id,updated_at) ON public.refunds FROM PUBLIC,anon,authenticated;
REVOKE ALL(applied_at,checksum,dirty,name,version) ON public.schema_migrations FROM PUBLIC,anon,authenticated;
REVOKE ALL(is_open,record_version,service_date,updated_at,updated_by_account_id) ON public.service_dates FROM PUBLIC,anon,authenticated;
REVOKE ALL(created_at,enabled,id,name,name_key,phone,record_version,updated_at) ON public.staff_whitelist FROM PUBLIC,anon,authenticated;
REVOKE ALL(announcement,aspect_ratio,business_status,center_x,center_y,flavor_options_json,id,launch_image_object_key,pickup_point,record_version,store_address,store_name,width_ratio) ON public.storefront_settings FROM PUBLIC,anon,authenticated;
ALTER TABLE public.action_audits ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.action_audits;
CREATE POLICY order_runtime_identity_v1 ON public.action_audits FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
ALTER TABLE public.discount_settings ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.discount_settings;
CREATE POLICY order_runtime_identity_v1 ON public.discount_settings FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
ALTER TABLE public.merchant_accounts ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.merchant_accounts;
CREATE POLICY order_runtime_identity_v1 ON public.merchant_accounts FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
ALTER TABLE public.miniprogram_users ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.miniprogram_users;
CREATE POLICY order_runtime_identity_v1 ON public.miniprogram_users FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
ALTER TABLE public.notification_consents ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.notification_consents;
CREATE POLICY order_runtime_identity_v1 ON public.notification_consents FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
ALTER TABLE public.notification_outbox ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.notification_outbox;
CREATE POLICY order_runtime_identity_v1 ON public.notification_outbox FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
ALTER TABLE public.order_items ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.order_items;
CREATE POLICY order_runtime_identity_v1 ON public.order_items FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
ALTER TABLE public.orders ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.orders;
CREATE POLICY order_runtime_identity_v1 ON public.orders FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
ALTER TABLE public.schema_migrations ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.schema_migrations;
CREATE POLICY order_runtime_identity_v1 ON public.schema_migrations FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
ALTER TABLE public.staff_whitelist ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.staff_whitelist;
CREATE POLICY order_runtime_identity_v1 ON public.staff_whitelist FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
ALTER TABLE public.merchant_pc_sessions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.merchant_pc_sessions;
CREATE POLICY order_runtime_identity_v1 ON public.merchant_pc_sessions FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
ALTER TABLE public.miniprogram_sessions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.miniprogram_sessions;
CREATE POLICY order_runtime_identity_v1 ON public.miniprogram_sessions FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
GRANT SELECT ON public.action_audits,public.discount_settings,public.merchant_accounts,public.merchant_pc_sessions,public.miniprogram_sessions,public.miniprogram_users,public.notification_consents,public.notification_outbox,public.order_items,public.orders,public.schema_migrations,public.staff_whitelist TO authenticated;
GRANT UPDATE(id) ON public.action_audits TO authenticated;
GRANT UPDATE(discount_version,rate_percent,updated_at,whitelist_version) ON public.discount_settings TO authenticated;
GRANT UPDATE(auth_version,bound_at,bound_user_id,deleted_at,deleted_by_account_id,enabled,name,phone,record_version,role,updated_at,updated_by) ON public.merchant_accounts TO authenticated;
GRANT UPDATE(access_expires_at,access_issued_at,access_token_hash,approved_account_id,approved_at,approved_auth_version,approved_user_id,consumed_at,state,updated_at) ON public.merchant_pc_sessions TO authenticated;
GRANT UPDATE(extra_name,extra_name_key,extra_phone,extra_phone_set_at,last_login_at,primary_phone,primary_phone_bound_at,record_version) ON public.miniprogram_users TO authenticated;
GRANT UPDATE(consumed_at) ON public.notification_consents TO authenticated;
GRANT UPDATE(id) ON public.notification_outbox TO authenticated;
GRANT UPDATE(completed_at,ready_at,record_version,redeemed_at,redeemed_by_account_id,redemption_issued_at,redemption_key_version,redemption_token_ciphertext,redemption_token_hash,state,updated_at) ON public.orders TO authenticated;
GRANT UPDATE(enabled,name,name_key,phone,record_version,updated_at) ON public.staff_whitelist TO authenticated;
GRANT INSERT ON public.action_audits,public.notification_outbox TO authenticated;
GRANT INSERT(auth_version,bound_at,bound_user_id,created_at,created_by,deleted_at,deleted_by_account_id,enabled,name,phone,record_version,role,updated_at,updated_by) ON public.merchant_accounts TO authenticated;
GRANT INSERT(approval_secret_hash,created_at,login_expires_at,poll_secret_hash,state,updated_at) ON public.merchant_pc_sessions TO authenticated;
GRANT INSERT(expires_at,issued_at,token_hash,user_id) ON public.miniprogram_sessions TO authenticated;
GRANT INSERT(created_at,last_login_at,openid) ON public.miniprogram_users TO authenticated;
GRANT DELETE ON public.merchant_pc_sessions,public.miniprogram_sessions TO authenticated;
GRANT USAGE ON SEQUENCE public.merchant_accounts_id_seq,public.merchant_action_audits_id_seq,public.merchant_pc_sessions_id_seq,public.miniprogram_users_id_seq,public.notification_outbox_id_seq TO authenticated;
GRANT EXECUTE ON FUNCTION public.order_fulfillment_proof_v1(numeric,text,numeric,text,text,date,boolean,text),public.order_identity_projection_v1(numeric),public.order_mark_ready_v1(numeric,numeric,text,text,timestamptz,timestamptz,text,jsonb,jsonb,timestamptz),public.order_query_get_merchant_v1(numeric,numeric,text),public.order_query_get_user_v1(numeric,numeric),public.order_query_list_user_v1(numeric,text,boolean,numeric,integer),public.order_query_search_merchant_v1(numeric,text,text,text,numeric,integer),public.order_redeem_current_date_code_v1(numeric,numeric,text,text,timestamptz,text,boolean,date,jsonb,timestamptz),public.order_redeem_order_proof_v1(numeric,numeric,text),public.order_redeem_order_v1(numeric,numeric,text,text,timestamptz,jsonb),public.order_redeem_token_v1(numeric,numeric,text,text,timestamptz,text,boolean,date,jsonb,text,timestamptz),public.order_shared_pg_readiness_v1(),public.order_identity_session_create_v1(text,text,timestamptz,timestamptz),public.order_identity_session_find_v1(text,timestamptz),public.order_identity_session_revoke_v1(text),public.order_identity_phone_read_v1(numeric),public.order_identity_phone_bind_v1(numeric,text,timestamptz),public.order_merchant_login_start_v1(numeric,text,text,timestamptz),public.order_merchant_login_complete_v1(numeric,text,text,text,timestamptz),public.order_merchant_login_recover_v1(numeric,text,text,timestamptz,timestamptz),public.order_identity_extra_phone_set_v1(numeric,text,text,text,text,text,text),public.order_merchant_account_current_v1(numeric),public.order_merchant_account_list_v1(numeric,text),public.order_merchant_account_command_v1(numeric,text,numeric,text,text,text,boolean,text,text,text,timestamptz,text),public.order_pc_login_begin_v1(text,text,timestamptz),public.order_pc_login_approval_read_v1(numeric,text),public.order_pc_login_approve_v1(numeric,numeric,text),public.order_pc_login_poll_v1(numeric,text,timestamptz,text,boolean),public.order_pc_session_authenticate_v1(text,timestamptz),public.order_pc_session_revoke_v1(text) TO authenticated;
ALTER TABLE public.categories ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.categories;
CREATE POLICY order_runtime_identity_v1 ON public.categories FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
ALTER TABLE public.meal_periods ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.meal_periods;
CREATE POLICY order_runtime_identity_v1 ON public.meal_periods FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
ALTER TABLE public.product_sold_out_dates ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.product_sold_out_dates;
CREATE POLICY order_runtime_identity_v1 ON public.product_sold_out_dates FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
ALTER TABLE public.products ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.products;
CREATE POLICY order_runtime_identity_v1 ON public.products FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
ALTER TABLE public.quote_items ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.quote_items;
CREATE POLICY order_runtime_identity_v1 ON public.quote_items FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
ALTER TABLE public.quotes ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.quotes;
CREATE POLICY order_runtime_identity_v1 ON public.quotes FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
ALTER TABLE public.service_dates ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.service_dates;
CREATE POLICY order_runtime_identity_v1 ON public.service_dates FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
ALTER TABLE public.storefront_settings ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.storefront_settings;
CREATE POLICY order_runtime_identity_v1 ON public.storefront_settings FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
GRANT SELECT ON public.categories,public.meal_periods,public.product_sold_out_dates,public.products,public.quote_items,public.quotes,public.service_dates,public.storefront_settings TO authenticated;
GRANT INSERT(is_active,name,name_key,record_version,sort_order) ON public.categories TO authenticated;
GRANT UPDATE(is_active,name,name_key,record_version,sort_order) ON public.categories TO authenticated;
GRANT UPDATE(cutoff_time,interval_minutes,pickup_end_time,pickup_start_time) ON public.meal_periods TO authenticated;
GRANT INSERT(product_id,service_date) ON public.product_sold_out_dates TO authenticated;
GRANT UPDATE(product_id) ON public.product_sold_out_dates TO authenticated;
GRANT INSERT(category_id,description,images_json,is_listed,meal_period,name,name_key,price_cents,record_version,sort_order,specification) ON public.products TO authenticated;
GRANT UPDATE(category_id,description,images_json,is_listed,meal_period,name,name_key,price_cents,record_version,sort_order) ON public.products TO authenticated;
GRANT INSERT(discounted_unit_price_cents,flavors_json,image_object_key_snapshot,line_note,line_number,original_subtotal_cents,original_unit_price_cents,payable_subtotal_cents,product_id,product_name_snapshot,product_source_version,quantity,quote_id) ON public.quote_items TO authenticated;
GRANT INSERT(contact_name_snapshot,contact_phone_snapshot,created_at,discount_cents,discount_rate_percent,discount_version,expires_at,idempotency_key_hash,identity_kind,identity_source_version,item_count,meal_period,order_note,original_subtotal_cents,payable_cents,pickup_date,pickup_point_snapshot,pickup_time,request_digest,snapshot_digest,store_address_snapshot,store_name_snapshot,user_id) ON public.quotes TO authenticated;
GRANT INSERT(is_open,record_version,service_date,updated_at,updated_by_account_id) ON public.service_dates TO authenticated;
GRANT UPDATE(is_open,record_version,updated_at,updated_by_account_id) ON public.service_dates TO authenticated;
GRANT UPDATE(announcement,aspect_ratio,business_status,center_x,center_y,launch_image_object_key,pickup_point,record_version,width_ratio) ON public.storefront_settings TO authenticated;
GRANT DELETE ON public.categories,public.products,public.product_sold_out_dates TO authenticated;
GRANT USAGE ON SEQUENCE public.categories_id_seq,public.products_id_seq,public.quotes_id_seq TO authenticated;
GRANT EXECUTE ON FUNCTION public.order_catalog_browse_v1(),public.order_catalog_categories_v1(),public.order_catalog_command_v1(numeric,text,numeric,numeric,boolean,text,text,bigint,text,jsonb,boolean,boolean,text,boolean,numeric[],text,text,text,text,timestamptz),public.order_catalog_detail_v1(numeric,date),public.order_catalog_product_v1(numeric,date),public.order_catalog_products_v1(date),public.order_display_pricing_facts_v1(numeric),public.order_image_confirmed_v1(text[]),public.order_menu_pickup_facts_v1(date,date),public.order_menu_snapshot_v1(date),public.order_merchant_soldout_command_v1(numeric,numeric,date,boolean,timestamptz,boolean,numeric,text,text,text,timestamptz),public.order_merchant_soldout_replay_v1(numeric,numeric,date,boolean,text,text,text),public.order_quote_create_v1(numeric,text,text,text,text,jsonb,timestamptz,timestamptz),public.order_quote_read_v1(numeric,numeric),public.order_storefront_admin_v1(),public.order_storefront_configure_v1(numeric,text,text,text,integer,jsonb,jsonb,text,text,text,timestamptz),public.order_storefront_get_v1(),public.order_storefront_launch_configure_v1(numeric,boolean,boolean,text,text,text,text,text,text,text,text,text,text,timestamptz),public.order_storefront_launch_read_v1() TO authenticated;

-- Staff administration and import persistence privileges for the reviewed UID.
ALTER TABLE public.import_batches ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.import_batches;
CREATE POLICY order_runtime_identity_v1 ON public.import_batches FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
GRANT SELECT ON public.import_batches TO authenticated;
GRANT DELETE ON public.staff_whitelist TO authenticated;
GRANT INSERT(created_at,enabled,name,name_key,phone,record_version,updated_at) ON public.staff_whitelist TO authenticated;
GRANT INSERT(actor_account_id,created_at,error_count,expires_at,file_digest,file_object_key,file_size_bytes,kind,new_count,preview_json,preview_token_hash,record_version,row_count,state,update_count,updated_at) ON public.import_batches TO authenticated;
GRANT UPDATE(commit_idempotency_key_hash,committed_at,preview_json,record_version,result_json,skip_invalid,state,updated_at) ON public.import_batches TO authenticated;
GRANT USAGE ON SEQUENCE public.import_batches_id_seq,public.staff_whitelist_id_seq TO authenticated;
GRANT EXECUTE ON FUNCTION public.order_import_commit_v1(numeric,text,boolean,text,text,text,text,jsonb,timestamptz,timestamptz),public.order_import_preview_facts_v1(text),public.order_import_preview_v1(numeric,text,text,numeric,numeric,text,text,text,timestamptz,text,text,text,timestamptz),public.order_staff_admin_command_v1(numeric,text,numeric,text,text,text,boolean,integer,text,text,text,text,timestamptz),public.order_staff_admin_list_v1(text),public.order_staff_admin_rate_v1() TO authenticated;

GRANT EXECUTE ON FUNCTION public.order_payment_receipt_read_v1(numeric,text,text),public.order_payment_prepare_replay_v1(numeric,numeric,text),public.order_payment_openid_v1(numeric),public.order_payment_prepare_v1(numeric,numeric,text,text,text,timestamptz,numeric,jsonb),public.order_payment_finish_create_v1(numeric,text,numeric,timestamptz,timestamptz,jsonb),public.order_payment_read_v1(numeric),public.order_payment_existing_order_v1(numeric),public.order_payment_due_v1(timestamptz,integer),public.order_payment_claim_query_v1(numeric,timestamptz,boolean,text,numeric),public.order_payment_finish_query_v1(numeric,text,numeric,timestamptz,timestamptz,boolean),public.order_payment_observe_v1(numeric,text,text,jsonb,jsonb,timestamptz,numeric),public.order_payment_materialize_v1(numeric,boolean,numeric),public.order_payment_pending_v1(text,numeric,integer),public.order_payment_receipt_append_v1(numeric,text,numeric,text,text,text,timestamptz),public.order_payment_owner_v1(numeric) TO authenticated;

-- Sixth-slice payment relations and exact native write columns.
ALTER TABLE public.payment_observations ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.payment_observations;
CREATE POLICY order_runtime_identity_v1 ON public.payment_observations FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
GRANT SELECT ON public.payment_observations TO authenticated;
ALTER TABLE public.pickup_sequences ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.pickup_sequences;
CREATE POLICY order_runtime_identity_v1 ON public.pickup_sequences FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
GRANT SELECT ON public.pickup_sequences TO authenticated;
ALTER TABLE public.prepayments ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.prepayments;
CREATE POLICY order_runtime_identity_v1 ON public.prepayments FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
GRANT SELECT ON public.prepayments TO authenticated;
GRANT INSERT(create_attempted_at,created_at,currency,effective_deadline,expected_amount_cents,expected_appid,expected_mchid,idempotency_key_hash,last_queried_at,lease_expires_at,lease_kind,lease_owner,materialization_state,materialized_at,next_reconcile_at,out_trade_no,pending_reason_code,provider_create_request_digest,provider_create_request_json,provider_prepay_id,provider_state,quote_id,record_version,updated_at,user_id,wx_request_payment_json) ON public.prepayments TO authenticated;
GRANT UPDATE(create_attempted_at,last_queried_at,lease_expires_at,lease_kind,lease_owner,materialization_state,materialized_at,next_reconcile_at,pending_reason_code,provider_prepay_id,provider_state,record_version,updated_at,wx_request_payment_json) ON public.prepayments TO authenticated;
GRANT INSERT(amount_cents,applied_at,apply_reason_code,apply_state,currency,dedupe_key,materialization_mode,mismatch_code,out_trade_no,prepayment_id,provider_event_id,provider_state,received_at,record_version,source,success_time,transaction_id,validation) ON public.payment_observations TO authenticated;
GRANT UPDATE(applied_at,apply_reason_code,apply_state,record_version) ON public.payment_observations TO authenticated;
GRANT INSERT(last_number,service_date,updated_at) ON public.pickup_sequences TO authenticated;
GRANT UPDATE(last_number,updated_at) ON public.pickup_sequences TO authenticated;
GRANT INSERT(completed_at,contact_name_snapshot,contact_phone_snapshot,created_at,discount_cents,discount_rate_percent,discount_version,identity_kind,identity_source_version,item_count,materialized_at,meal_period,order_no,order_note,original_subtotal_cents,paid_at,payable_cents,payment_observation_id,pickup_at,pickup_date,pickup_number,pickup_point_snapshot,pickup_time,preparing_at,prepayment_id,quote_id,ready_at,record_version,redeemed_at,redeemed_by_account_id,redemption_issued_at,redemption_key_version,redemption_token_ciphertext,redemption_token_hash,refunded_at,refunding_at,snapshot_digest,state,store_address_snapshot,store_name_snapshot,transaction_id,updated_at,user_id) ON public.orders TO authenticated;
GRANT INSERT(discounted_unit_price_cents,flavors_json,image_object_key_snapshot,line_note,line_number,order_id,original_subtotal_cents,original_unit_price_cents,payable_subtotal_cents,product_id,product_name_snapshot,product_source_version,quantity) ON public.order_items TO authenticated;
GRANT UPDATE(id) ON public.quotes TO authenticated;
GRANT UPDATE(quote_id) ON public.quote_items TO authenticated;
GRANT USAGE ON SEQUENCE public.orders_id_seq,public.payment_observations_id_seq,public.prepayments_id_seq TO authenticated;

ALTER TABLE public.refund_observations ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.refund_observations;
CREATE POLICY order_runtime_identity_v1 ON public.refund_observations FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');

ALTER TABLE public.refunds ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_runtime_identity_v1 ON public.refunds;
CREATE POLICY order_runtime_identity_v1 ON public.refunds FOR ALL TO authenticated
 USING (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__')
 WITH CHECK (current_user='authenticated' AND auth.uid()='__ORDER_SERVICE_UID__');
GRANT SELECT ON public.refund_observations,public.refunds TO authenticated;
GRANT INSERT(applied_at,apply_reason_code,apply_state,dedupe_key,mismatch_code,out_refund_no,provider_event_id,provider_refund_id,provider_state,received_at,record_version,refund_id,source,success_time,validation) ON public.refund_observations TO authenticated;
GRANT INSERT(amount_cents,create_attempted_at,created_at,currency,idempotency_key_hash,lease_expires_at,lease_kind,lease_owner,materialization_state,materialized_at,next_query_at,order_id,out_refund_no,pending_reason_code,prepayment_id,provider_refund_id,provider_request_digest,provider_request_json,provider_state,reason_code,record_version,requested_by_account_id,requested_by_user_id,updated_at) ON public.refunds TO authenticated;
GRANT INSERT(consumed_at,decided_at,decision,grant_sequence,idempotency_key_hash,kind,order_id,template_config_version,user_id) ON public.notification_consents TO authenticated;
GRANT UPDATE(applied_at,apply_reason_code,apply_state,record_version) ON public.refund_observations TO authenticated;
GRANT UPDATE(create_attempted_at,lease_expires_at,lease_kind,lease_owner,materialization_state,materialized_at,next_query_at,pending_reason_code,provider_refund_id,provider_state,record_version,updated_at) ON public.refunds TO authenticated;
GRANT UPDATE(completed_at,preparing_at,ready_at,record_version,redeemed_at,redeemed_by_account_id,redemption_issued_at,redemption_key_version,redemption_token_ciphertext,redemption_token_hash,refunded_at,refunding_at,state,updated_at) ON public.orders TO authenticated;
GRANT UPDATE(attempt_count,id,last_error_code,lease_expires_at,lease_owner,next_attempt_at,provider_message_id,record_version,sent_at,state) ON public.notification_outbox TO authenticated;
GRANT UPDATE(id,response_json) ON public.action_audits TO authenticated;
GRANT USAGE ON SEQUENCE public.notification_consents_id_seq,public.refund_observations_id_seq,public.refunds_id_seq TO authenticated;
GRANT EXECUTE ON FUNCTION public.order_refund_order_locator_v1(numeric),public.order_refund_request_order_v1(text,numeric,jsonb,text,text,text,jsonb,text),public.order_refund_request_paid_v1(numeric,numeric,text,text,text,jsonb,text),public.order_refund_claim_create_v1(numeric,text,jsonb),public.order_refund_finish_create_v1(numeric,text,numeric,jsonb,boolean,jsonb,jsonb),public.order_refund_read_v1(numeric),public.order_refund_observation_locator_v1(text),public.order_refund_observe_v1(jsonb,text,text,jsonb,text,jsonb,jsonb,boolean),public.order_refund_due_v1(jsonb,jsonb,integer),public.order_refund_claim_query_v1(numeric,text,jsonb,jsonb),public.order_refund_release_query_v1(numeric,text,numeric,jsonb),public.order_refund_pending_v1(numeric,numeric,integer),public.order_subscription_consent_read_v1(numeric,text),public.order_subscription_consent_record_v1(numeric,numeric,text,text,numeric,text,text,jsonb),public.order_subscription_claim_v1(jsonb,jsonb,text),public.order_subscription_sent_v1(numeric,text,numeric,text,jsonb),public.order_subscription_temporary_failure_v1(numeric,text,numeric,text,jsonb,jsonb),public.order_subscription_permanent_failure_v1(numeric,text,numeric,text,jsonb),public.order_production_due_v1(jsonb,integer),public.order_production_advance_v1(numeric,jsonb) TO authenticated;

GRANT EXECUTE ON FUNCTION public.order_billing_summary_v1(numeric,jsonb,jsonb),public.order_billing_payments_v1(numeric,jsonb,jsonb,numeric,integer),public.order_billing_refunds_v1(numeric,jsonb,jsonb,numeric,integer),public.order_billing_payment_facts_v1(jsonb,jsonb),public.order_billing_refund_facts_v1(jsonb,jsonb),public.order_billing_reconcile_v1(text,text,numeric,jsonb,jsonb),public.order_admin_report_get_v1(numeric,numeric),public.order_admin_report_search_v1(numeric,text,text,text,boolean,numeric,integer),public.order_admin_report_stats_v1(numeric,jsonb),public.order_admin_report_payments_v1(numeric,text,text,numeric,integer),public.order_admin_report_refunds_v1(numeric,text,text,numeric,integer),public.order_admin_report_summary_v1(numeric,text,text),public.order_admin_report_pending_v1(numeric,numeric,integer),public.order_audit_search_v1(numeric,text,text,text,text,numeric,integer),public.order_store_status_apply_v1(numeric,text,text,text,text,jsonb) TO authenticated;

GRANT EXECUTE ON FUNCTION public.order_image_confirmation_commit_v1(numeric,text,text,text,text,text,text,timestamptz),public.order_image_confirmation_replay_v1(numeric,text),public.order_image_versions_v1(text[]) TO authenticated;

REVOKE ALL ON FUNCTION public.order_import_result_v1(numeric,numeric) FROM PUBLIC,anon,authenticated;
GRANT EXECUTE ON FUNCTION public.order_import_result_v1(numeric,numeric) TO authenticated;

REVOKE ALL ON FUNCTION public.order_admin_report_counts_v1(numeric),public.order_admin_report_export_v1(numeric,text,text,numeric,integer) FROM PUBLIC,anon;
GRANT EXECUTE ON FUNCTION public.order_admin_report_counts_v1(numeric),public.order_admin_report_export_v1(numeric,text,text,numeric,integer) TO authenticated;
