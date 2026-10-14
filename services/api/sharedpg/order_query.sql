-- Fixed, parameterized raw-fact reads for the reviewed Order service identity.

CREATE OR REPLACE FUNCTION public.order_query_list_user_v1(p_user numeric,p_state text,p_active boolean,p_after numeric,p_limit integer) RETURNS jsonb
LANGUAGE plpgsql SECURITY INVOKER VOLATILE
SET search_path=pg_catalog,public SET TimeZone='UTC'
AS $order_rpc$
DECLARE result jsonb;
BEGIN
IF current_user <> 'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid()::text IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='order_runtime_identity_required'; END IF;
IF p_user IS NULL OR p_user<>trunc(p_user) OR p_user<1 OR p_user>18446744073709551615 THEN
 RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='invalid_query'; END IF;
IF p_state IS NULL OR (p_state<>'' AND p_state NOT IN ('RESERVED','PREPARING','READY_FOR_PICKUP','COMPLETED','REFUNDING','REFUNDED')) OR p_after IS NULL OR p_after<>trunc(p_after) OR p_after<0 OR p_after>18446744073709551615 OR p_limit IS NULL OR p_limit<1 OR p_limit>100 THEN
 RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='invalid_query'; END IF;
IF p_active IS NULL OR (p_active AND p_state<>'') THEN RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='invalid_query'; END IF;
SELECT COALESCE(jsonb_agg(facts ORDER BY id DESC),'[]'::jsonb) INTO result FROM (
 SELECT o.id,jsonb_build_object('id',o.id::text,'order_no_hex',encode(o.order_no,'hex'),'state',(CASE WHEN o.state='RESERVED' AND o.pickup_at<=statement_timestamp()+interval '30 minutes' THEN 'PREPARING' ELSE o.state END),
 'pickup_date',to_char(o.pickup_date,'YYYY-MM-DD'),'pickup_time',to_char(o.pickup_date+o.pickup_time,'HH24:MI'),
 'pickup_point',o.pickup_point_snapshot,'pickup_number',o.pickup_number::text,'payable_cents',o.payable_cents::text,
 'materialized_at',o.materialized_at,'pickup_at',o.pickup_at) AS facts FROM public.orders o
 WHERE o.user_id=p_user AND (p_state='' OR (CASE WHEN o.state='RESERVED' AND o.pickup_at<=statement_timestamp()+interval '30 minutes' THEN 'PREPARING' ELSE o.state END)=p_state)
 AND (NOT p_active OR o.state IN ('RESERVED','PREPARING','READY_FOR_PICKUP'))
 AND (p_after=0 OR o.id<p_after) ORDER BY o.id DESC LIMIT p_limit+1
) rows; RETURN result;
END;
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_query_get_user_v1(p_user numeric,p_order numeric) RETURNS jsonb
LANGUAGE plpgsql SECURITY INVOKER VOLATILE
SET search_path=pg_catalog,public SET TimeZone='UTC'
AS $order_rpc$
DECLARE result jsonb; authorized boolean;
BEGIN
IF current_user <> 'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid()::text IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='order_runtime_identity_required'; END IF;
IF p_user IS NULL OR p_user<>trunc(p_user) OR p_user<1 OR p_user>18446744073709551615 THEN
 RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='invalid_query'; END IF;
IF p_order IS NULL OR p_order<>trunc(p_order) OR p_order<1 OR p_order>18446744073709551615 THEN
 RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='invalid_query'; END IF;

SELECT true, (SELECT jsonb_build_object('order',jsonb_build_object('id',o.id::text,'order_no_hex',encode(o.order_no,'hex'),'state',(CASE WHEN o.state='RESERVED' AND o.pickup_at<=statement_timestamp()+interval '30 minutes' THEN 'PREPARING' ELSE o.state END),
 'pickup_date',to_char(o.pickup_date,'YYYY-MM-DD'),'pickup_time',to_char(o.pickup_date+o.pickup_time,'HH24:MI'),
 'pickup_point',o.pickup_point_snapshot,'pickup_number',o.pickup_number::text,'payable_cents',o.payable_cents::text,
 'materialized_at',o.materialized_at,'pickup_at',o.pickup_at,
 'contact_name',o.contact_name_snapshot,'contact_phone_hex',encode(o.contact_phone_snapshot,'hex'),
 'identity_kind',o.identity_kind,'discount_rate',o.discount_rate_percent::text,'transaction_id_hex',encode(o.transaction_id,'hex'),
 'paid_at',o.paid_at,'order_note',o.order_note,'item_count',o.item_count::text,
 'preparing_at',(CASE WHEN o.state='RESERVED' AND o.pickup_at<=statement_timestamp()+interval '30 minutes' THEN GREATEST(o.materialized_at,o.pickup_at-interval '30 minutes') ELSE o.preparing_at END),'ready_at',o.ready_at,'completed_at',o.completed_at,'refunding_at',o.refunding_at,'refunded_at',o.refunded_at,
 'redemption_cipher_hex',encode(o.redemption_token_ciphertext,'hex'),'redemption_hash_hex',encode(o.redemption_token_hash,'hex'),
 'redemption_version',o.redemption_key_version::text,'redemption_issued_at',o.redemption_issued_at,
 'redeemed_by',o.redeemed_by_account_id::text,'redeemed_at',o.redeemed_at),'items',(SELECT COALESCE(jsonb_agg(jsonb_build_object(
 'line',line_number::text,'product_id',product_id::text,'name',product_name_snapshot,
 'unit_price_cents',discounted_unit_price_cents::text,'quantity',quantity::text,'line_total_cents',payable_subtotal_cents::text,
 'flavors',flavors_json::jsonb,'note',line_note
) ORDER BY line_number),'[]'::jsonb) FROM public.order_items WHERE order_id=o.id)) FROM public.orders o WHERE o.id=p_order AND o.user_id=p_user LIMIT 1) INTO authorized,result;
RETURN COALESCE(result,'null'::jsonb);
END;
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_query_search_merchant_v1(p_user numeric,p_state text,p_date text,p_search text,p_after numeric,p_limit integer) RETURNS jsonb
LANGUAGE plpgsql SECURITY INVOKER VOLATILE
SET search_path=pg_catalog,public SET TimeZone='UTC'
AS $order_rpc$
DECLARE result jsonb; escaped text;
BEGIN
IF current_user <> 'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid()::text IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='order_runtime_identity_required'; END IF;
IF p_user IS NULL OR p_user<>trunc(p_user) OR p_user<1 OR p_user>18446744073709551615 THEN
 RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='invalid_query'; END IF;
IF p_state IS NULL OR (p_state<>'' AND p_state NOT IN ('RESERVED','PREPARING','READY_FOR_PICKUP','COMPLETED','REFUNDING','REFUNDED')) OR p_after IS NULL OR p_after<>trunc(p_after) OR p_after<0 OR p_after>18446744073709551615 OR p_limit IS NULL OR p_limit<1 OR p_limit>100 THEN
 RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='invalid_query'; END IF;
IF p_date IS NULL OR p_search IS NULL OR octet_length(p_search)>64 OR btrim(p_search)<>p_search THEN RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='invalid_query'; END IF;
IF p_date<>'' THEN
 IF p_date !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' OR to_char(p_date::date,'YYYY-MM-DD')<>p_date THEN RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='invalid_query'; END IF;
END IF;
IF NOT EXISTS(SELECT 1 FROM public.merchant_accounts WHERE bound_user_id=p_user AND enabled=true AND deleted_at IS NULL AND role IN ('OWNER','SUBACCOUNT') AND auth_version>0) THEN
 RAISE EXCEPTION USING ERRCODE='P1002',MESSAGE='merchant_forbidden'; END IF;
escaped:=replace(replace(replace(p_search,'=','=='),'%','=%'),'_','=_');
SELECT COALESCE(jsonb_agg(facts ORDER BY id DESC),'[]'::jsonb) INTO result FROM (
 SELECT o.id,jsonb_build_object('id',o.id::text,'order_no_hex',encode(o.order_no,'hex'),'state',(CASE WHEN o.state='RESERVED' AND o.pickup_at<=statement_timestamp()+interval '30 minutes' THEN 'PREPARING' ELSE o.state END),
 'pickup_date',to_char(o.pickup_date,'YYYY-MM-DD'),'pickup_time',to_char(o.pickup_date+o.pickup_time,'HH24:MI'),
 'pickup_point',o.pickup_point_snapshot,'pickup_number',o.pickup_number::text,'payable_cents',o.payable_cents::text,
 'materialized_at',o.materialized_at,'pickup_at',o.pickup_at) AS facts FROM public.orders o
 WHERE (p_state='' OR (CASE WHEN o.state='RESERVED' AND o.pickup_at<=statement_timestamp()+interval '30 minutes' THEN 'PREPARING' ELSE o.state END)=p_state) AND (p_date='' OR o.pickup_date=p_date::date)
 AND (p_search='' OR o.order_no=convert_to(p_search,'UTF8') OR lpad(o.pickup_number::text,4,'0')=p_search OR convert_from(o.contact_phone_snapshot,'UTF8') LIKE '%'||escaped ESCAPE '=')
 AND (p_after=0 OR o.id<p_after) ORDER BY o.id DESC LIMIT p_limit+1
) rows; RETURN result;
END;
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_query_get_merchant_v1(p_user numeric,p_order numeric,p_state text) RETURNS jsonb
LANGUAGE plpgsql SECURITY INVOKER VOLATILE
SET search_path=pg_catalog,public SET TimeZone='UTC'
AS $order_rpc$
DECLARE result jsonb; authorized boolean;
BEGIN
IF current_user <> 'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid()::text IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='order_runtime_identity_required'; END IF;
IF p_user IS NULL OR p_user<>trunc(p_user) OR p_user<1 OR p_user>18446744073709551615 THEN
 RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='invalid_query'; END IF;
IF p_order IS NULL OR p_order<>trunc(p_order) OR p_order<1 OR p_order>18446744073709551615 THEN
 RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='invalid_query'; END IF;
IF p_state IS NULL OR (p_state<>'' AND p_state NOT IN ('RESERVED','PREPARING','READY_FOR_PICKUP','COMPLETED','REFUNDING','REFUNDED')) THEN RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='invalid_query'; END IF;
SELECT EXISTS(SELECT 1 FROM public.merchant_accounts WHERE bound_user_id=p_user AND enabled=true AND deleted_at IS NULL AND role IN ('OWNER','SUBACCOUNT') AND auth_version>0), (SELECT jsonb_build_object('order',jsonb_build_object('id',o.id::text,'order_no_hex',encode(o.order_no,'hex'),'state',(CASE WHEN o.state='RESERVED' AND o.pickup_at<=statement_timestamp()+interval '30 minutes' THEN 'PREPARING' ELSE o.state END),
 'pickup_date',to_char(o.pickup_date,'YYYY-MM-DD'),'pickup_time',to_char(o.pickup_date+o.pickup_time,'HH24:MI'),
 'pickup_point',o.pickup_point_snapshot,'pickup_number',o.pickup_number::text,'payable_cents',o.payable_cents::text,
 'materialized_at',o.materialized_at,'pickup_at',o.pickup_at,
 'contact_name',o.contact_name_snapshot,'contact_phone_hex',encode(o.contact_phone_snapshot,'hex'),
 'identity_kind',o.identity_kind,'discount_rate',o.discount_rate_percent::text,'transaction_id_hex',encode(o.transaction_id,'hex'),
 'paid_at',o.paid_at,'order_note',o.order_note,'item_count',o.item_count::text,
 'preparing_at',(CASE WHEN o.state='RESERVED' AND o.pickup_at<=statement_timestamp()+interval '30 minutes' THEN GREATEST(o.materialized_at,o.pickup_at-interval '30 minutes') ELSE o.preparing_at END),'ready_at',o.ready_at,'completed_at',o.completed_at,'refunding_at',o.refunding_at,'refunded_at',o.refunded_at,
 'redemption_cipher_hex',encode(o.redemption_token_ciphertext,'hex'),'redemption_hash_hex',encode(o.redemption_token_hash,'hex'),
 'redemption_version',o.redemption_key_version::text,'redemption_issued_at',o.redemption_issued_at,
 'redeemed_by',o.redeemed_by_account_id::text,'redeemed_at',o.redeemed_at),'items',(SELECT COALESCE(jsonb_agg(jsonb_build_object(
 'line',line_number::text,'product_id',product_id::text,'name',product_name_snapshot,
 'unit_price_cents',discounted_unit_price_cents::text,'quantity',quantity::text,'line_total_cents',payable_subtotal_cents::text,
 'flavors',flavors_json::jsonb,'note',line_note
) ORDER BY line_number),'[]'::jsonb) FROM public.order_items WHERE order_id=o.id)) FROM public.orders o WHERE o.id=p_order LIMIT 1) INTO authorized,result;
IF NOT authorized THEN RAISE EXCEPTION USING ERRCODE='P1002',MESSAGE='merchant_forbidden'; END IF;
RETURN COALESCE(result,'null'::jsonb);
END;
$order_rpc$;
