CREATE OR REPLACE FUNCTION public.order_redeem_order_proof_v1(p_actor numeric,p_order numeric,p_key text)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE a public.merchant_accounts%ROWTYPE; o public.orders%ROWTYPE; scope bytea; proof jsonb;
BEGIN
  IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
  END IF;
  IF p_actor IS NULL OR p_order IS NULL OR p_actor<>trunc(p_actor) OR p_order<>trunc(p_order)
    OR p_actor<=0 OR p_order<=0 OR p_actor>18446744073709551615 OR p_order>18446744073709551615
    OR p_key IS NULL OR p_key !~ '^[!-~]{1,128}$' THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
  p_actor:=trunc(p_actor); p_order:=trunc(p_order);
  SELECT * INTO a FROM public.merchant_accounts WHERE bound_user_id=p_actor FOR SHARE;
  IF NOT FOUND OR NOT a.enabled THEN RAISE EXCEPTION USING ERRCODE='P1002',MESSAGE='ORDER_FORBIDDEN'; END IF;
  IF a.id<=0 OR a.role NOT IN ('OWNER','SUBACCOUNT') OR a.record_version<=0 OR a.auth_version<=0 THEN
    RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_UNAVAILABLE';
  END IF;
  SELECT * INTO o FROM public.orders WHERE id=p_order FOR SHARE;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P1003',MESSAGE='ORDER_NOT_FOUND'; END IF;
  IF o.id<=0 OR o.record_version<=0 OR o.pickup_number NOT BETWEEN 1 AND 9999 THEN
    RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_UNAVAILABLE';
  END IF;
  scope:=sha256(convert_to('merchant:'||p_actor::text||':'||a.id::text,'UTF8'));
  proof:=jsonb_build_object('id',o.id::text,'state',o.state,'version',o.record_version::text,
    'pickup_date',to_char(o.pickup_date,'YYYY-MM-DD'),'pickup_number',o.pickup_number,
    'preparing_at',o.preparing_at,'ready_at',o.ready_at,'completed_at',o.completed_at,
    'hash',encode(o.redemption_token_hash,'hex'),'ciphertext',encode(o.redemption_token_ciphertext,'hex'),
    'key_version',o.redemption_key_version,'issued_at',o.redemption_issued_at,
    'redeemed_by',o.redeemed_by_account_id::text,'redeemed_at',o.redeemed_at);
  RETURN jsonb_build_object('proof',proof,'receipt_exists',EXISTS(
    SELECT 1 FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='MERCHANT'
    AND actor_scope_hash=scope AND action='fulfillment.redeem_order'
    AND operation_key_hash=sha256(convert_to('operation:'||p_key,'UTF8'))));
END;
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_redeem_order_v1(p_actor numeric,p_order numeric,p_key text,p_request_id text,p_now timestamptz,p_proof jsonb)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE a public.merchant_accounts%ROWTYPE; o public.orders%ROWTYPE; receipt public.action_audits%ROWTYPE;
  scope bytea; operation bytea; request_digest text; response jsonb; proof jsonb; affected bigint; constraint_name text;
BEGIN
  IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
  END IF;
  IF p_actor IS NULL OR p_order IS NULL OR p_actor<>trunc(p_actor) OR p_order<>trunc(p_order)
    OR p_actor<=0 OR p_order<=0 OR p_actor>18446744073709551615 OR p_order>18446744073709551615
    OR p_key IS NULL OR p_key !~ '^[!-~]{1,128}$' OR p_request_id IS NULL OR p_request_id !~ '^[!-~]{1,64}$'
    OR p_now IS NULL OR NOT isfinite(p_now) THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
  p_actor:=trunc(p_actor); p_order:=trunc(p_order);
  SELECT * INTO a FROM public.merchant_accounts WHERE bound_user_id=p_actor FOR SHARE;
  IF NOT FOUND OR NOT a.enabled THEN RAISE EXCEPTION USING ERRCODE='P1002',MESSAGE='ORDER_FORBIDDEN'; END IF;
  IF a.id<=0 OR a.role NOT IN ('OWNER','SUBACCOUNT') OR a.record_version<=0 OR a.auth_version<=0 THEN
    RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_UNAVAILABLE';
  END IF;
  SELECT * INTO o FROM public.orders WHERE id=p_order FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P1003',MESSAGE='ORDER_NOT_FOUND'; END IF;
  IF o.id<=0 OR o.record_version<=0 OR o.pickup_number NOT BETWEEN 1 AND 9999 THEN
    RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_UNAVAILABLE';
  END IF;
  scope:=sha256(convert_to('merchant:'||p_actor::text||':'||a.id::text,'UTF8'));
  operation:=sha256(convert_to('operation:'||p_key,'UTF8'));
  request_digest:=encode(sha256(convert_to('{"kind":"REDEEM_ORDER","order_id":'||p_order::text||'}','UTF8')),'hex');
  SELECT * INTO receipt FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='MERCHANT'
    AND actor_scope_hash=scope AND action='fulfillment.redeem_order' AND operation_key_hash=operation FOR SHARE;
  IF FOUND THEN
    IF jsonb_typeof(receipt.before_state_json) IS DISTINCT FROM 'object' OR octet_length(receipt.before_state_json::text)>256
      OR (SELECT count(*) FROM jsonb_object_keys(receipt.before_state_json))<>1
      OR receipt.before_state_json->>'request_digest' IS NULL
      OR jsonb_typeof(receipt.before_state_json->'request_digest') IS DISTINCT FROM 'string'
      OR receipt.before_state_json->>'request_digest' !~ '^[0-9a-fA-F]{64}$' THEN
      RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_RECEIPT_CORRUPT';
    END IF;
    IF lower(receipt.before_state_json->>'request_digest')<>request_digest THEN
      RAISE EXCEPTION USING ERRCODE='P1006',MESSAGE='ORDER_IDEMPOTENCY_CONFLICT';
    END IF;
    response:=receipt.response_json;
    IF jsonb_typeof(response) IS DISTINCT FROM 'object' OR octet_length(response::text)>1024
      OR (SELECT count(*) FROM jsonb_object_keys(response))<>3
      OR jsonb_typeof(response->'order_id') IS DISTINCT FROM 'number'
      OR response->>'order_id' IS NULL OR response->>'order_id' !~ '^[1-9][0-9]{0,19}$'
      OR response->>'order_id' IS DISTINCT FROM p_order::text
      OR response->>'state' IS DISTINCT FROM 'COMPLETED' OR response->'changed' IS DISTINCT FROM 'true'::jsonb THEN
      RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_RECEIPT_CORRUPT';
    END IF;
    RETURN jsonb_build_object('order_id',p_order::text,'state','COMPLETED','changed',true,'replay',true);
  END IF;
  IF o.state<>'READY_FOR_PICKUP' THEN RAISE EXCEPTION USING ERRCODE='P1004',MESSAGE='ORDER_TRANSITION'; END IF;
  IF o.preparing_at IS NULL OR o.ready_at IS NULL OR o.ready_at<o.preparing_at OR o.ready_at>p_now OR o.completed_at IS NOT NULL
    OR o.redemption_token_hash IS NULL OR octet_length(o.redemption_token_hash)<>32
    OR o.redemption_token_ciphertext IS NULL OR octet_length(o.redemption_token_ciphertext) NOT BETWEEN 1 AND 192
    OR o.redemption_key_version IS NULL OR o.redemption_key_version NOT BETWEEN 1 AND 65535
    OR o.redemption_issued_at IS NULL OR o.redemption_issued_at<o.ready_at OR o.redemption_issued_at>p_now
    OR o.redeemed_by_account_id IS NOT NULL OR o.redeemed_at IS NOT NULL THEN
    RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_UNAVAILABLE';
  END IF;
  proof:=jsonb_build_object('id',o.id::text,'state',o.state,'version',o.record_version::text,
    'pickup_date',to_char(o.pickup_date,'YYYY-MM-DD'),'pickup_number',o.pickup_number,
    'preparing_at',o.preparing_at,'ready_at',o.ready_at,'completed_at',o.completed_at,
    'hash',encode(o.redemption_token_hash,'hex'),'ciphertext',encode(o.redemption_token_ciphertext,'hex'),
    'key_version',o.redemption_key_version,'issued_at',o.redemption_issued_at,
    'redeemed_by',o.redeemed_by_account_id::text,'redeemed_at',o.redeemed_at);
  IF p_proof IS DISTINCT FROM proof THEN RAISE EXCEPTION USING ERRCODE='P1005',MESSAGE='ORDER_STALE_PROOF'; END IF;
  UPDATE public.orders SET state='COMPLETED',completed_at=p_now,redeemed_by_account_id=a.id,redeemed_at=p_now,
    redemption_token_ciphertext=NULL,redemption_key_version=NULL,record_version=record_version+1,updated_at=p_now WHERE id=p_order;
  GET DIAGNOSTICS affected=ROW_COUNT;
  IF affected<>1 THEN RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_WRITE_COUNT'; END IF;
  response:=jsonb_build_object('order_id',p_order,'state','COMPLETED','changed',true);
  BEGIN
  INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,
    actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,target_type,target_id,
    target_key_hash,operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at)
  VALUES('COMMAND_RECEIPT','MERCHANT',scope,p_actor,a.id,a.id,a.role,a.auth_version,'fulfillment.redeem_order','ORDER',p_order,
    NULL,operation,sha256(convert_to('request:'||p_request_id,'UTF8')),'SUCCEEDED','OK',
    jsonb_build_object('request_digest',request_digest),NULL,response,p_now);
  EXCEPTION WHEN unique_violation THEN
    GET STACKED DIAGNOSTICS constraint_name=CONSTRAINT_NAME;
    IF constraint_name='uq_action_audits_command_receipt' THEN
      RAISE EXCEPTION USING ERRCODE='P1008',MESSAGE='ORDER_RECEIPT_RACE';
    END IF;
    RAISE;
  END;
  RETURN jsonb_build_object('order_id',p_order::text,'state','COMPLETED','changed',true,'replay',false);
END;
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_shared_pg_readiness_v1()
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
BEGIN
  IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
  END IF;
  RETURN jsonb_build_object(
    'role',current_user,'uid',auth.uid(),'schema',__ORDER_SCHEMA_CATALOG_SQL__,
    'roles',(SELECT jsonb_agg(jsonb_build_object('name',r.rolname,'superuser',r.rolsuper,
      'create_role',r.rolcreaterole,'create_database',r.rolcreatedb,'bypass_rls',r.rolbypassrls,
      'create_schemas',(SELECT coalesce(jsonb_agg(n.nspname ORDER BY n.nspname),'[]'::jsonb) FROM pg_namespace n
        WHERE n.nspname !~ '^pg_temp_' AND has_schema_privilege(r.rolname,n.oid,'CREATE')),
      'membership',(SELECT coalesce(jsonb_agg(z.rolname ORDER BY z.rolname),'[]'::jsonb) FROM pg_roles z
        WHERE z.oid<>r.oid AND pg_has_role(r.rolname,z.oid,'MEMBER'))) ORDER BY r.rolname)
      FROM pg_roles r WHERE r.rolname IN ('anon','authenticated')),
    'functions',(SELECT jsonb_agg(jsonb_build_object('name',p.proname,'hash',encode(sha256(convert_to(p.prosrc,'UTF8')),'hex'),
      'security_definer',p.prosecdef,'returns_set',p.proretset,'strict',p.proisstrict,'volatility',p.provolatile::text,
      'parallel',p.proparallel::text,'result_type',p.prorettype::regtype::text,'argument_types',p.proargtypes::text,
      'argument_names',p.proargnames,'language',l.lanname,'config',p.proconfig,
      'kind',p.prokind::text,'defaults',p.pronargdefaults,'variadic',p.provariadic::bigint,
      'owner',pg_get_userbyid(p.proowner),
      'public_execute',EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE a.grantee=0 AND a.privilege_type='EXECUTE'),
      'anon_execute',has_function_privilege('anon',p.oid,'EXECUTE'),
      'authenticated_execute',has_function_privilege('authenticated',p.oid,'EXECUTE')) ORDER BY p.proname)
      FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace JOIN pg_language l ON l.oid=p.prolang
      WHERE n.nspname='public' AND p.proname IN ('order_admin_report_get_v1','order_admin_report_payments_v1','order_admin_report_pending_v1','order_admin_report_refunds_v1','order_admin_report_search_v1','order_admin_report_stats_v1','order_admin_report_summary_v1','order_audit_search_v1','order_billing_ownership_facts_v1','order_billing_payment_facts_v1','order_billing_payments_v1','order_billing_reconcile_v1','order_billing_reconcile_v2','order_billing_refund_facts_v1','order_billing_refunds_v1','order_billing_summary_v1','order_catalog_browse_v1','order_catalog_categories_v1','order_catalog_command_v1','order_catalog_detail_v1','order_catalog_product_v1','order_catalog_products_v1','order_display_pricing_facts_v1','order_fulfillment_proof_v1','order_identity_extra_phone_set_v1','order_identity_phone_bind_v1','order_identity_phone_read_v1','order_identity_projection_v1','order_identity_session_create_v1','order_identity_session_find_v1','order_identity_session_revoke_v1','order_image_confirmation_commit_v1','order_image_confirmation_replay_v1','order_image_confirmed_v1','order_image_versions_v1','order_import_commit_v1','order_import_preview_facts_v1','order_import_preview_v1','order_mark_ready_v1','order_menu_pickup_facts_v1','order_menu_snapshot_v1','order_merchant_account_command_v1','order_merchant_account_current_v1','order_merchant_account_list_v1','order_merchant_login_complete_v1','order_merchant_login_recover_v1','order_merchant_login_start_v1','order_merchant_soldout_command_v1','order_merchant_soldout_replay_v1','order_payment_claim_query_v1','order_payment_due_v1','order_payment_existing_order_v1','order_payment_finish_create_v1','order_payment_finish_query_v1','order_payment_materialize_v1','order_payment_observe_v1','order_payment_openid_v1','order_payment_owner_v1','order_payment_pending_v1','order_payment_prepare_replay_v1','order_payment_prepare_v1','order_payment_prepare_v2','order_payment_read_v1','order_payment_receipt_append_v1','order_payment_receipt_read_v1','order_pc_login_approval_read_v1','order_pc_login_approve_v1','order_pc_login_begin_v1','order_pc_login_poll_v1','order_pc_session_authenticate_v1','order_pc_session_revoke_v1','order_production_advance_v1','order_production_due_v1','order_query_get_merchant_v1','order_query_get_user_v1','order_query_list_user_v1','order_query_search_merchant_v1','order_quote_create_v1','order_quote_read_v1','order_redeem_current_date_code_v1','order_redeem_order_proof_v1','order_redeem_order_v1','order_redeem_token_v1','order_refund_claim_create_v1','order_refund_claim_query_v1','order_refund_due_v1','order_refund_finish_create_v1','order_refund_observation_locator_v1','order_refund_observe_v1','order_refund_order_locator_v1','order_refund_pending_v1','order_refund_read_v1','order_refund_release_query_v1','order_refund_request_order_v1','order_refund_request_order_v2','order_refund_request_paid_v1','order_refund_request_paid_v2','order_shared_pg_readiness_v1','order_staff_admin_command_v1','order_staff_admin_list_v1','order_staff_admin_rate_v1','order_store_status_apply_v1','order_storefront_admin_v1','order_storefront_configure_v1','order_storefront_get_v1','order_storefront_launch_configure_v1','order_storefront_launch_read_v1','order_subscription_claim_v1','order_subscription_consent_read_v1','order_subscription_consent_record_v1','order_subscription_permanent_failure_v1','order_subscription_sent_v1','order_subscription_temporary_failure_v1')),
    'tables',(SELECT jsonb_agg(jsonb_build_object('name',c.relname,'rls',c.relrowsecurity,'owner',pg_get_userbyid(c.relowner),
      'public_privileges',(SELECT coalesce(jsonb_agg(a.privilege_type ORDER BY a.privilege_type),'[]'::jsonb)
        FROM aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a WHERE a.grantee=0),
      'anon_privileges',(SELECT coalesce(jsonb_agg(x ORDER BY x),'[]'::jsonb) FROM unnest(ARRAY['SELECT','INSERT','UPDATE','DELETE','TRUNCATE','REFERENCES','TRIGGER']) x WHERE has_table_privilege('anon',c.oid,x)),
      'authenticated_privileges',(SELECT coalesce(jsonb_agg(x ORDER BY x),'[]'::jsonb) FROM unnest(ARRAY['SELECT','INSERT','UPDATE','DELETE','TRUNCATE','REFERENCES','TRIGGER']) x WHERE has_table_privilege('authenticated',c.oid,x)),
      'anon_column_access',has_any_column_privilege('anon',c.oid,'SELECT,INSERT,UPDATE,REFERENCES'),
      'public_column_access',EXISTS(SELECT 1 FROM pg_attribute z,aclexplode(z.attacl) a WHERE z.attrelid=c.oid AND a.grantee=0),
      'authenticated_insert_columns',has_any_column_privilege('authenticated',c.oid,'INSERT'),
      'authenticated_insert_column_names',(SELECT coalesce(jsonb_agg(a.attname ORDER BY a.attname),'[]'::jsonb) FROM pg_attribute a WHERE a.attrelid=c.oid AND a.attnum>0 AND NOT a.attisdropped AND has_column_privilege('authenticated',c.oid,a.attname,'INSERT')),
      'authenticated_reference_columns',has_any_column_privilege('authenticated',c.oid,'REFERENCES'),
      'authenticated_update_columns',(SELECT coalesce(jsonb_agg(a.attname ORDER BY a.attname),'[]'::jsonb) FROM pg_attribute a WHERE a.attrelid=c.oid AND a.attnum>0 AND NOT a.attisdropped AND has_column_privilege('authenticated',c.oid,a.attname,'UPDATE')),
      'policies',(SELECT coalesce(jsonb_agg(jsonb_build_object('name',p.polname,'permissive',p.polpermissive,'command',p.polcmd::text,
        'roles',(SELECT jsonb_agg(pg_get_userbyid(r) ORDER BY pg_get_userbyid(r)) FROM unnest(p.polroles) r),
        'using',pg_get_expr(p.polqual,p.polrelid),'check',pg_get_expr(p.polwithcheck,p.polrelid)) ORDER BY p.polname),'[]'::jsonb)
        FROM pg_policy p WHERE p.polrelid=c.oid)) ORDER BY c.relname)
      FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname IN ('action_audits','categories','discount_settings','import_batches','meal_periods','merchant_accounts','merchant_pc_sessions','miniprogram_sessions','miniprogram_users','notification_consents','notification_outbox','order_items','orders','payment_observations','pickup_sequences','prepayments','product_sold_out_dates','products','quote_items','quotes','refund_observations','refunds','schema_migrations','service_dates','staff_whitelist','storefront_settings')),
    'unreviewed_definers',(SELECT coalesce(jsonb_agg(jsonb_build_object('signature',p.oid::regprocedure::text,'role',r.name) ORDER BY p.oid::regprocedure::text,r.name),'[]'::jsonb)
      FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace CROSS JOIN (VALUES('anon'),('authenticated')) AS r(name)
      WHERE n.nspname='public' AND p.prosecdef AND p.oid NOT IN ('public.order_fulfillment_proof_v1(numeric,text,numeric,text,text,date,boolean,text)'::regprocedure,'public.order_identity_projection_v1(numeric)'::regprocedure,'public.order_mark_ready_v1(numeric,numeric,text,text,timestamptz,timestamptz,text,jsonb,jsonb,timestamptz)'::regprocedure,'public.order_query_get_merchant_v1(numeric,numeric,text)'::regprocedure,'public.order_query_get_user_v1(numeric,numeric)'::regprocedure,'public.order_query_list_user_v1(numeric,text,boolean,numeric,integer)'::regprocedure,'public.order_query_search_merchant_v1(numeric,text,text,text,numeric,integer)'::regprocedure,'public.order_redeem_current_date_code_v1(numeric,numeric,text,text,timestamptz,text,boolean,date,jsonb,timestamptz)'::regprocedure,'public.order_redeem_order_proof_v1(numeric,numeric,text)'::regprocedure,'public.order_redeem_order_v1(numeric,numeric,text,text,timestamptz,jsonb)'::regprocedure,'public.order_redeem_token_v1(numeric,numeric,text,text,timestamptz,text,boolean,date,jsonb,text,timestamptz)'::regprocedure,'public.order_shared_pg_readiness_v1()'::regprocedure,'public.order_identity_session_create_v1(text,text,timestamptz,timestamptz)'::regprocedure,'public.order_identity_session_find_v1(text,timestamptz)'::regprocedure,'public.order_identity_session_revoke_v1(text)'::regprocedure,'public.order_identity_phone_read_v1(numeric)'::regprocedure,'public.order_identity_phone_bind_v1(numeric,text,timestamptz)'::regprocedure,'public.order_merchant_login_start_v1(numeric,text,text,timestamptz)'::regprocedure,'public.order_merchant_login_complete_v1(numeric,text,text,text,timestamptz)'::regprocedure,'public.order_merchant_login_recover_v1(numeric,text,text,timestamptz,timestamptz)'::regprocedure,'public.order_identity_extra_phone_set_v1(numeric,text,text,text,text,text,text)'::regprocedure,'public.order_merchant_account_current_v1(numeric)'::regprocedure,'public.order_merchant_account_list_v1(numeric,text)'::regprocedure,'public.order_merchant_account_command_v1(numeric,text,numeric,text,text,text,boolean,text,text,text,timestamptz,text)'::regprocedure,'public.order_pc_login_begin_v1(text,text,timestamptz)'::regprocedure,'public.order_pc_login_approval_read_v1(numeric,text)'::regprocedure,'public.order_pc_login_approve_v1(numeric,numeric,text)'::regprocedure,'public.order_pc_login_poll_v1(numeric,text,timestamptz,text,boolean)'::regprocedure,'public.order_pc_session_authenticate_v1(text,timestamptz)'::regprocedure,'public.order_pc_session_revoke_v1(text)'::regprocedure,'public.order_catalog_browse_v1()'::regprocedure,'public.order_catalog_categories_v1()'::regprocedure,'public.order_catalog_command_v1(numeric,text,numeric,numeric,boolean,text,text,bigint,text,jsonb,boolean,boolean,text,boolean,numeric[],text,text,text,text,timestamptz)'::regprocedure,'public.order_catalog_detail_v1(numeric,date)'::regprocedure,'public.order_catalog_product_v1(numeric,date)'::regprocedure,'public.order_catalog_products_v1(date)'::regprocedure,'public.order_display_pricing_facts_v1(numeric)'::regprocedure,'public.order_image_confirmation_commit_v1(numeric,text,text,text,text,text,text,timestamptz)'::regprocedure,'public.order_image_confirmation_replay_v1(numeric,text)'::regprocedure,'public.order_image_confirmed_v1(text[])'::regprocedure,'public.order_image_versions_v1(text[])'::regprocedure,'public.order_menu_pickup_facts_v1(date,date)'::regprocedure,'public.order_menu_snapshot_v1(date)'::regprocedure,'public.order_merchant_soldout_command_v1(numeric,numeric,date,boolean,timestamptz,boolean,numeric,text,text,text,timestamptz)'::regprocedure,'public.order_merchant_soldout_replay_v1(numeric,numeric,date,boolean,text,text,text)'::regprocedure,'public.order_quote_create_v1(numeric,text,text,text,text,jsonb,timestamptz,timestamptz)'::regprocedure,'public.order_quote_read_v1(numeric,numeric)'::regprocedure,'public.order_storefront_admin_v1()'::regprocedure,'public.order_storefront_configure_v1(numeric,text,text,text,integer,jsonb,jsonb,text,text,text,timestamptz)'::regprocedure,'public.order_storefront_get_v1()'::regprocedure,'public.order_storefront_launch_configure_v1(numeric,boolean,boolean,text,text,text,text,text,text,text,text,text,text,timestamptz)'::regprocedure,'public.order_storefront_launch_read_v1()'::regprocedure,'public.order_staff_admin_list_v1(text)'::regprocedure,'public.order_staff_admin_rate_v1()'::regprocedure,'public.order_staff_admin_command_v1(numeric,text,numeric,text,text,text,boolean,integer,text,text,text,text,timestamptz)'::regprocedure,'public.order_import_preview_facts_v1(text)'::regprocedure,'public.order_import_preview_v1(numeric,text,text,numeric,numeric,text,text,text,timestamptz,text,text,text,timestamptz)'::regprocedure,'public.order_import_commit_v1(numeric,text,boolean,text,text,text,text,jsonb,timestamptz,timestamptz)'::regprocedure,'public.order_payment_receipt_read_v1(numeric,text,text)'::regprocedure,'public.order_payment_prepare_replay_v1(numeric,numeric,text)'::regprocedure,'public.order_payment_openid_v1(numeric)'::regprocedure,'public.order_payment_prepare_v1(numeric,numeric,text,text,text,timestamptz,numeric,jsonb)'::regprocedure,'public.order_payment_finish_create_v1(numeric,text,numeric,timestamptz,timestamptz,jsonb)'::regprocedure,'public.order_payment_read_v1(numeric)'::regprocedure,'public.order_payment_existing_order_v1(numeric)'::regprocedure,'public.order_payment_due_v1(timestamptz,integer)'::regprocedure,'public.order_payment_claim_query_v1(numeric,timestamptz,boolean,text,numeric)'::regprocedure,'public.order_payment_finish_query_v1(numeric,text,numeric,timestamptz,timestamptz,boolean)'::regprocedure,'public.order_payment_observe_v1(numeric,text,text,jsonb,jsonb,timestamptz,numeric)'::regprocedure,'public.order_payment_materialize_v1(numeric,boolean,numeric)'::regprocedure,'public.order_payment_pending_v1(text,numeric,integer)'::regprocedure,'public.order_payment_receipt_append_v1(numeric,text,numeric,text,text,text,timestamptz)'::regprocedure,'public.order_payment_owner_v1(numeric)'::regprocedure,'public.order_refund_order_locator_v1(numeric)'::regprocedure,'public.order_refund_request_order_v1(text,numeric,jsonb,text,text,text,jsonb,text)'::regprocedure,'public.order_refund_request_paid_v1(numeric,numeric,text,text,text,jsonb,text)'::regprocedure,'public.order_refund_claim_create_v1(numeric,text,jsonb)'::regprocedure,'public.order_refund_finish_create_v1(numeric,text,numeric,jsonb,boolean,jsonb,jsonb)'::regprocedure,'public.order_refund_read_v1(numeric)'::regprocedure,'public.order_refund_observation_locator_v1(text)'::regprocedure,'public.order_refund_observe_v1(jsonb,text,text,jsonb,text,jsonb,jsonb,boolean)'::regprocedure,'public.order_refund_due_v1(jsonb,jsonb,integer)'::regprocedure,'public.order_refund_claim_query_v1(numeric,text,jsonb,jsonb)'::regprocedure,'public.order_refund_release_query_v1(numeric,text,numeric,jsonb)'::regprocedure,'public.order_refund_pending_v1(numeric,numeric,integer)'::regprocedure,'public.order_subscription_consent_read_v1(numeric,text)'::regprocedure,'public.order_subscription_consent_record_v1(numeric,numeric,text,text,numeric,text,text,jsonb)'::regprocedure,'public.order_subscription_claim_v1(jsonb,jsonb,text)'::regprocedure,'public.order_subscription_sent_v1(numeric,text,numeric,text,jsonb)'::regprocedure,'public.order_subscription_temporary_failure_v1(numeric,text,numeric,text,jsonb,jsonb)'::regprocedure,'public.order_subscription_permanent_failure_v1(numeric,text,numeric,text,jsonb)'::regprocedure,'public.order_production_due_v1(jsonb,integer)'::regprocedure,'public.order_production_advance_v1(numeric,jsonb)'::regprocedure,'public.order_billing_summary_v1(numeric,jsonb,jsonb)'::regprocedure,'public.order_billing_payments_v1(numeric,jsonb,jsonb,numeric,integer)'::regprocedure,'public.order_billing_refunds_v1(numeric,jsonb,jsonb,numeric,integer)'::regprocedure,'public.order_billing_payment_facts_v1(jsonb,jsonb)'::regprocedure,'public.order_billing_refund_facts_v1(jsonb,jsonb)'::regprocedure,'public.order_billing_reconcile_v1(text,text,numeric,jsonb,jsonb)'::regprocedure,'public.order_admin_report_get_v1(numeric,numeric)'::regprocedure,'public.order_admin_report_search_v1(numeric,text,text,text,boolean,numeric,integer)'::regprocedure,'public.order_admin_report_stats_v1(numeric,jsonb)'::regprocedure,'public.order_admin_report_payments_v1(numeric,text,text,numeric,integer)'::regprocedure,'public.order_admin_report_refunds_v1(numeric,text,text,numeric,integer)'::regprocedure,'public.order_admin_report_summary_v1(numeric,text,text)'::regprocedure,'public.order_admin_report_pending_v1(numeric,numeric,integer)'::regprocedure,'public.order_audit_search_v1(numeric,text,text,text,text,numeric,integer)'::regprocedure,'public.order_store_status_apply_v1(numeric,text,text,text,text,jsonb)'::regprocedure) AND has_function_privilege(r.name,p.oid,'EXECUTE')),
    'unreviewed_relations',(SELECT coalesce(jsonb_agg(jsonb_build_object('name',c.relname,'kind',c.relkind::text,'role',r.name) ORDER BY c.relname,r.name),'[]'::jsonb)
      FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace CROSS JOIN (VALUES('anon'),('authenticated')) AS r(name)
      WHERE n.nspname='public' AND CASE
       WHEN c.relkind IN ('r','p','v','m','f') AND c.relname NOT IN ('action_audits','categories','discount_settings','import_batches','meal_periods','merchant_accounts','merchant_pc_sessions','miniprogram_sessions','miniprogram_users','notification_consents','notification_outbox','order_items','orders','payment_observations','pickup_sequences','prepayments','product_sold_out_dates','products','quote_items','quotes','refund_observations','refunds','schema_migrations','service_dates','staff_whitelist','storefront_settings') THEN
        has_table_privilege(r.name,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') OR has_any_column_privilege(r.name,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')
       WHEN c.relkind='S' AND c.relname NOT IN ('categories_id_seq','import_batches_id_seq','merchant_accounts_id_seq','merchant_action_audits_id_seq','merchant_pc_sessions_id_seq','miniprogram_users_id_seq','notification_consents_id_seq','notification_outbox_id_seq','orders_id_seq','payment_observations_id_seq','prepayments_id_seq','products_id_seq','quotes_id_seq','refund_observations_id_seq','refunds_id_seq','staff_whitelist_id_seq') THEN
        has_sequence_privilege(r.name,c.oid,'USAGE,SELECT,UPDATE')
       ELSE false END),
    'sequences',(SELECT jsonb_agg(jsonb_build_object('name',c.relname,'owner',pg_get_userbyid(c.relowner),
      'public_privileges',(SELECT coalesce(jsonb_agg(a.privilege_type ORDER BY a.privilege_type),'[]'::jsonb)
        FROM aclexplode(coalesce(c.relacl,acldefault('S',c.relowner))) a WHERE a.grantee=0),
      'anon_privileges',(SELECT coalesce(jsonb_agg(x ORDER BY x),'[]'::jsonb) FROM unnest(ARRAY['SELECT','UPDATE','USAGE']) x WHERE has_sequence_privilege('anon',c.oid,x)),
      'authenticated_privileges',(SELECT coalesce(jsonb_agg(x ORDER BY x),'[]'::jsonb) FROM unnest(ARRAY['SELECT','UPDATE','USAGE']) x WHERE has_sequence_privilege('authenticated',c.oid,x))) ORDER BY c.relname)
      FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname IN ('categories_id_seq','import_batches_id_seq','merchant_accounts_id_seq','merchant_action_audits_id_seq','merchant_pc_sessions_id_seq','miniprogram_users_id_seq','notification_consents_id_seq','notification_outbox_id_seq','orders_id_seq','payment_observations_id_seq','prepayments_id_seq','products_id_seq','quotes_id_seq','refund_observations_id_seq','refunds_id_seq','staff_whitelist_id_seq')));
END;
$order_rpc$;
