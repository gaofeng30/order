-- Fixed payment ABI; raw frozen aggregates stay inside each atomic RPC.

CREATE OR REPLACE FUNCTION public.order_payment_existing_order_v1(p_prepayment numeric)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
<<payment_command>>
DECLARE order_id numeric;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 SELECT id INTO order_id FROM public.orders WHERE prepayment_id=p_prepayment;IF NOT FOUND THEN RETURN jsonb_build_object('outcome','ABSENT');END IF;RETURN jsonb_build_object('outcome','FOUND','order_id',order_id::text);

 END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_payment_owner_v1(p_user numeric)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
<<payment_command>>
DECLARE enabled boolean;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 SELECT count(*)>0 INTO enabled FROM public.merchant_accounts m WHERE m.bound_user_id=p_user AND m.role='OWNER' AND m.enabled=TRUE AND m.deleted_at IS NULL;RETURN jsonb_build_object('outcome','FOUND','enabled',enabled);

 END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_payment_pending_v1(p_reason_hex text,p_after numeric,p_limit integer)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
<<payment_command>>
DECLARE payments jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 SELECT coalesce(jsonb_agg(jsonb_build_object('id',id::text,'reason',pending_reason_code,'created_at',created_at) ORDER BY id),'[]'::jsonb) INTO payments FROM (SELECT id,pending_reason_code,created_at FROM public.prepayments WHERE materialization_state='PENDING_MANUAL' AND id>p_after AND (p_reason_hex='' OR convert_to(pending_reason_code,'UTF8')=decode(p_reason_hex,'hex')) ORDER BY id LIMIT p_limit) pending;RETURN jsonb_build_object('outcome','FOUND','payments',payments);

 END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_payment_materialize_v1(p_prepayment numeric,p_manual boolean,p_owner numeric)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
<<payment_command>>
DECLARE pr public.prepayments%ROWTYPE;req jsonb;request_json text;request_field record;canonical_fields jsonb;folded_field text;h public.quotes%ROWTYPE;qi public.quote_items%ROWTYPE;items public.quote_items[];line_count integer;flavor jsonb;seen_flavors text[];
snapshot_valid boolean; snapshot_read_unavailable boolean; snapshot_material bytea;material bytea;raw_value bytea;part text;
pickup_at timestamptz;original_total numeric;payable_total numeric;discounted_unit numeric;maximum_int64 CONSTANT numeric:=9223372036854775807;
v_go_space CONSTANT text:=convert_from(decode('090a0b0c0d20c285c2a0e19a80e28080e28081e28082e28083e28084e28085e28086e28087e28088e28089e2808ae280a8e280a9e280afe2819fe38080','hex'),'UTF8');obs public.payment_observations%ROWTYPE;observation_id numeric;order_id numeric;account_id numeric;defer_reason text;materialized_at timestamptz;state text;pickup_number integer;affected bigint;
 failure_state text;failure_constraint text;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF p_manual THEN SELECT id INTO observation_id FROM public.payment_observations WHERE prepayment_id=p_prepayment AND provider_state='PAID' AND validation='MATCH' AND apply_state IN ('NEW','DEFERRED') ORDER BY id LIMIT 1;
 ELSE SELECT id INTO observation_id FROM public.payment_observations WHERE prepayment_id=p_prepayment AND apply_state='NEW' AND materialization_mode='AUTO' ORDER BY id LIMIT 1;END IF;
 IF NOT FOUND THEN SELECT id INTO order_id FROM public.orders WHERE prepayment_id=p_prepayment;IF FOUND THEN RETURN jsonb_build_object('outcome','APPLIED','order_id',order_id::text);END IF;RETURN jsonb_build_object('outcome','PENDING');END IF;
 IF p_manual THEN SELECT id INTO account_id FROM public.merchant_accounts WHERE bound_user_id=p_owner AND role='OWNER' AND enabled=TRUE AND deleted_at IS NULL ORDER BY id LIMIT 1 FOR UPDATE;
 IF NOT FOUND THEN RETURN jsonb_build_object('outcome','FORBIDDEN');END IF;IF account_id IS NULL OR account_id=0 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;END IF;
 SELECT * INTO pr FROM public.prepayments WHERE id=p_prepayment FOR UPDATE;IF NOT FOUND THEN RETURN jsonb_build_object('outcome','NOT_FOUND');END IF;
 IF pr.id IS NULL OR pr.user_id IS NULL OR pr.quote_id IS NULL OR pr.record_version IS NULL OR pr.id NOT BETWEEN 1 AND 18446744073709551615 OR pr.user_id NOT BETWEEN 1 AND 18446744073709551615 OR pr.quote_id NOT BETWEEN 1 AND 18446744073709551615 OR pr.record_version NOT BETWEEN 1 AND 18446744073709551615 OR pr.expected_amount_cents IS NULL OR pr.expected_amount_cents NOT BETWEEN 1 AND 9223372036854775807 OR pr.currency IS DISTINCT FROM 'CNY' OR pr.out_trade_no IS NULL OR octet_length(pr.out_trade_no)=0 OR pr.expected_appid IS NULL OR octet_length(pr.expected_appid)=0 OR pr.expected_mchid IS NULL OR octet_length(pr.expected_mchid)=0 OR pr.effective_deadline IS NULL OR pr.effective_deadline='0001-01-01T00:00:00Z'::timestamptz OR pr.created_at IS NULL OR pr.updated_at IS NULL OR pr.provider_state IS NULL OR pr.materialization_state IS NULL OR octet_length(pr.idempotency_key_hash) IS DISTINCT FROM 32 OR octet_length(pr.provider_create_request_digest) IS DISTINCT FROM 32 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 req:=pr.provider_create_request_json;
 IF jsonb_typeof(req) IS DISTINCT FROM 'object' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
-- Match encoding/json foldName and Decode error/null behavior over the actual JSONB serialized scan order.
canonical_fields:='{}'::jsonb;
FOR request_field IN SELECT key,value,ordinality FROM json_each(req::text::json) WITH ORDINALITY AS fields(key,value,ordinality) ORDER BY ordinality LOOP
folded_field:=translate(request_field.key,'ABCDEFGHIJKLMNOPQRSTUVWXYZſ','abcdefghijklmnopqrstuvwxyzs');
IF folded_field NOT IN ('appid','mchid','description','out_trade_no','time_expire','notify_url','amount_cents','currency','payer_openid','quote_id','quote_digest') THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF json_typeof(request_field.value)='null' THEN CONTINUE;END IF;
IF folded_field='amount_cents' THEN IF json_typeof(request_field.value) IS DISTINCT FROM 'number' OR request_field.value::text !~ '^(0|-?[1-9][0-9]*)$' OR request_field.value::text::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
ELSIF json_typeof(request_field.value) IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
canonical_fields:=canonical_fields||jsonb_build_object(folded_field,request_field.value::jsonb);END LOOP;
req:=canonical_fields;IF (SELECT count(*) FROM jsonb_object_keys(req))<>11 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(req->'appid') IS DISTINCT FROM 'string' OR req->>'appid'='' OR octet_length(req->>'appid')>64 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(req->'mchid') IS DISTINCT FROM 'string' OR req->>'mchid'='' OR octet_length(req->>'mchid')>64 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(req->'description') IS DISTINCT FROM 'string' OR req->>'description'='' OR octet_length(req->>'description')>128 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(req->'out_trade_no') IS DISTINCT FROM 'string' OR req->>'out_trade_no'='' OR octet_length(req->>'out_trade_no')>64 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(req->'time_expire') IS DISTINCT FROM 'string' OR req->>'time_expire'='' OR octet_length(req->>'time_expire')>64 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(req->'notify_url') IS DISTINCT FROM 'string' OR req->>'notify_url'='' OR octet_length(req->>'notify_url')>2048 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(req->'amount_cents') IS DISTINCT FROM 'number' OR (req->>'amount_cents') !~ '^[1-9][0-9]*$' OR (req->>'amount_cents')::numeric>9223372036854775807 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(req->'currency') IS DISTINCT FROM 'string' OR req->>'currency'='' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(req->'payer_openid') IS DISTINCT FROM 'string' OR req->>'payer_openid'='' OR octet_length(req->>'payer_openid')>128 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(req->'quote_id') IS DISTINCT FROM 'string' OR req->>'quote_id'='' OR octet_length(req->>'quote_id')>20 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(req->'quote_digest') IS DISTINCT FROM 'string' OR req->>'quote_digest'='' OR octet_length(req->>'quote_digest')>64 OR octet_length(req->>'quote_digest')<>64 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF req->>'currency'<>'CNY' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
request_json:='{"appid":'||replace(replace(replace(replace(replace(to_json(req->>'appid')::text,'<',E'\\u003c'),'>',E'\\u003e'),'&',E'\\u0026'),chr(8232),E'\\u2028'),chr(8233),E'\\u2029')||',"mchid":'||replace(replace(replace(replace(replace(to_json(req->>'mchid')::text,'<',E'\\u003c'),'>',E'\\u003e'),'&',E'\\u0026'),chr(8232),E'\\u2028'),chr(8233),E'\\u2029')||',"description":'||replace(replace(replace(replace(replace(to_json(req->>'description')::text,'<',E'\\u003c'),'>',E'\\u003e'),'&',E'\\u0026'),chr(8232),E'\\u2028'),chr(8233),E'\\u2029')||',"out_trade_no":'||replace(replace(replace(replace(replace(to_json(req->>'out_trade_no')::text,'<',E'\\u003c'),'>',E'\\u003e'),'&',E'\\u0026'),chr(8232),E'\\u2028'),chr(8233),E'\\u2029')||',"time_expire":'||replace(replace(replace(replace(replace(to_json(req->>'time_expire')::text,'<',E'\\u003c'),'>',E'\\u003e'),'&',E'\\u0026'),chr(8232),E'\\u2028'),chr(8233),E'\\u2029')||',"notify_url":'||replace(replace(replace(replace(replace(to_json(req->>'notify_url')::text,'<',E'\\u003c'),'>',E'\\u003e'),'&',E'\\u0026'),chr(8232),E'\\u2028'),chr(8233),E'\\u2029')||',"amount_cents":'||(req->>'amount_cents')||',"currency":'||replace(replace(replace(replace(replace(to_json(req->>'currency')::text,'<',E'\\u003c'),'>',E'\\u003e'),'&',E'\\u0026'),chr(8232),E'\\u2028'),chr(8233),E'\\u2029')||',"payer_openid":'||replace(replace(replace(replace(replace(to_json(req->>'payer_openid')::text,'<',E'\\u003c'),'>',E'\\u003e'),'&',E'\\u0026'),chr(8232),E'\\u2028'),chr(8233),E'\\u2029')||',"quote_id":'||replace(replace(replace(replace(replace(to_json(req->>'quote_id')::text,'<',E'\\u003c'),'>',E'\\u003e'),'&',E'\\u0026'),chr(8232),E'\\u2028'),chr(8233),E'\\u2029')||',"quote_digest":'||replace(replace(replace(replace(replace(to_json(req->>'quote_digest')::text,'<',E'\\u003c'),'>',E'\\u003e'),'&',E'\\u0026'),chr(8232),E'\\u2028'),chr(8233),E'\\u2029')||'}';
IF sha256(convert_to(request_json,'UTF8'))<>pr.provider_create_request_digest THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;

 SELECT * INTO obs FROM public.payment_observations WHERE id=observation_id FOR UPDATE;
 IF NOT FOUND OR obs.id IS NULL OR obs.prepayment_id IS NULL OR obs.out_trade_no IS NULL OR obs.transaction_id IS NULL OR obs.provider_state IS NULL OR obs.validation IS NULL OR obs.amount_cents IS NULL OR obs.currency IS NULL OR obs.success_time IS NULL OR NOT isfinite(obs.success_time) OR obs.materialization_mode IS NULL OR obs.apply_state IS NULL OR obs.record_version IS NULL OR obs.id NOT BETWEEN 0 AND 18446744073709551615 OR obs.prepayment_id NOT BETWEEN 0 AND 18446744073709551615 OR obs.record_version NOT BETWEEN 0 AND 18446744073709551615 OR obs.amount_cents NOT BETWEEN -9223372036854775808 AND 9223372036854775807 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 defer_reason:=NULL;
 <<materialization_facts>> BEGIN
 IF obs.prepayment_id<>pr.id OR obs.provider_state<>'PAID' OR obs.validation<>'MATCH' OR octet_length(obs.transaction_id)=0 OR obs.amount_cents<=0 OR obs.currency<>'CNY' OR obs.success_time='0001-01-01T00:00:00Z'::timestamptz THEN defer_reason:='PAYMENT_FACT_INVALID';EXIT materialization_facts;END IF;
 IF NOT p_manual AND (obs.materialization_mode<>'AUTO' OR obs.apply_state<>'NEW') THEN RETURN jsonb_build_object('outcome','PENDING');END IF;
 SELECT * INTO h FROM public.quotes WHERE id=pr.quote_id;
 IF NOT FOUND THEN defer_reason:='QUOTE_SNAPSHOT_INVALID';EXIT materialization_facts;END IF;
 snapshot_valid:=false;snapshot_read_unavailable:=false;items:=ARRAY[]::public.quote_items[];
 <<snapshot_check>> BEGIN
 IF h.id IS NULL OR h.user_id IS NULL OR h.identity_source_version IS NULL OR h.discount_version IS NULL OR h.item_count IS NULL OR h.original_subtotal_cents IS NULL OR h.discount_cents IS NULL OR h.payable_cents IS NULL OR h.contact_name_snapshot IS NULL OR h.contact_phone_snapshot IS NULL OR h.identity_kind IS NULL OR h.discount_rate_percent IS NULL OR h.store_name_snapshot IS NULL OR h.store_address_snapshot IS NULL OR h.pickup_point_snapshot IS NULL OR h.pickup_date IS NULL OR h.pickup_time IS NULL OR h.meal_period IS NULL OR h.order_note IS NULL OR h.created_at IS NULL OR h.expires_at IS NULL OR h.id NOT BETWEEN 0 AND 18446744073709551615 OR h.user_id NOT BETWEEN 0 AND 18446744073709551615 OR h.identity_source_version NOT BETWEEN 0 AND 18446744073709551615 OR h.discount_version NOT BETWEEN 0 AND 18446744073709551615 OR h.item_count NOT BETWEEN 0 AND 65535 OR h.original_subtotal_cents NOT BETWEEN -9223372036854775808 AND maximum_int64 OR h.discount_cents NOT BETWEEN -9223372036854775808 AND maximum_int64 OR h.payable_cents NOT BETWEEN -9223372036854775808 AND maximum_int64 THEN snapshot_read_unavailable:=true;EXIT snapshot_check;END IF;
 IF octet_length(h.request_digest) IS DISTINCT FROM 32 OR octet_length(h.snapshot_digest) IS DISTINCT FROM 32 OR h.id=0 OR h.user_id=0 OR h.item_count=0 OR h.identity_source_version=0 OR h.discount_version=0 OR h.discount_rate_percent NOT BETWEEN 1 AND 100 OR h.identity_kind NOT IN ('STAFF','VISITOR') OR h.meal_period NOT IN ('lunch','dinner') OR h.pickup_date::text !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' OR h.pickup_time::text !~ '^([01][0-9]|2[0-3]):[0-5][0-9]:00$' OR h.created_at='0001-01-01T00:00:00Z'::timestamptz THEN EXIT snapshot_check;END IF;
 IF octet_length(h.contact_name_snapshot)>64 OR encode(h.contact_phone_snapshot,'hex') !~ '^2b3[1-9](3[0-9]){0,14}$' THEN EXIT snapshot_check;END IF;
 FOREACH part IN ARRAY ARRAY[h.contact_name_snapshot,h.store_name_snapshot,h.store_address_snapshot,h.pickup_point_snapshot] LOOP IF part='' OR convert_to(btrim(part,v_go_space),'UTF8')<>convert_to(part,'UTF8') THEN EXIT snapshot_check;END IF;END LOOP;
 pickup_at:=((h.pickup_date+h.pickup_time)-interval '8 hours') AT TIME ZONE 'UTC';
 IF pickup_at<=h.created_at OR h.expires_at<>least(h.created_at+interval '10 minutes',pickup_at) THEN EXIT snapshot_check;END IF;
 original_total:=0;payable_total:=0;line_count:=0;
snapshot_material:=int8send(octet_length(convert_to('order.quote.snapshot.v1','UTF8'))::bigint)||(convert_to('order.quote.snapshot.v1','UTF8'))||int8send((CASE WHEN (h.user_id)::numeric>9223372036854775807 THEN (h.user_id)::numeric-18446744073709551616 ELSE (h.user_id)::numeric END)::bigint)||int8send(octet_length(convert_to(h.contact_name_snapshot,'UTF8'))::bigint)||(convert_to(h.contact_name_snapshot,'UTF8'))||int8send(octet_length(h.contact_phone_snapshot)::bigint)||(h.contact_phone_snapshot)||int8send(octet_length(convert_to(h.identity_kind,'UTF8'))::bigint)||(convert_to(h.identity_kind,'UTF8'))||int8send((CASE WHEN (h.identity_source_version)::numeric>9223372036854775807 THEN (h.identity_source_version)::numeric-18446744073709551616 ELSE (h.identity_source_version)::numeric END)::bigint)||int8send((CASE WHEN (h.discount_rate_percent)::numeric>9223372036854775807 THEN (h.discount_rate_percent)::numeric-18446744073709551616 ELSE (h.discount_rate_percent)::numeric END)::bigint)||int8send((CASE WHEN (h.discount_version)::numeric>9223372036854775807 THEN (h.discount_version)::numeric-18446744073709551616 ELSE (h.discount_version)::numeric END)::bigint)||int8send(octet_length(convert_to(h.store_name_snapshot,'UTF8'))::bigint)||(convert_to(h.store_name_snapshot,'UTF8'))||int8send(octet_length(convert_to(h.store_address_snapshot,'UTF8'))::bigint)||(convert_to(h.store_address_snapshot,'UTF8'))||int8send(octet_length(convert_to(h.pickup_date::text,'UTF8'))::bigint)||(convert_to(h.pickup_date::text,'UTF8'))||int8send(octet_length(convert_to(substring(h.pickup_time::text,1,5),'UTF8'))::bigint)||(convert_to(substring(h.pickup_time::text,1,5),'UTF8'))||int8send(octet_length(convert_to(h.meal_period,'UTF8'))::bigint)||(convert_to(h.meal_period,'UTF8'))||int8send(octet_length(convert_to(h.pickup_point_snapshot,'UTF8'))::bigint)||(convert_to(h.pickup_point_snapshot,'UTF8'))||int8send(octet_length(convert_to(h.order_note,'UTF8'))::bigint)||(convert_to(h.order_note,'UTF8'))||int8send((CASE WHEN (h.item_count)::numeric>9223372036854775807 THEN (h.item_count)::numeric-18446744073709551616 ELSE (h.item_count)::numeric END)::bigint);
FOR qi IN SELECT * FROM public.quote_items WHERE quote_id=h.id ORDER BY line_number LOOP

 line_count:=line_count+1;
 IF qi.line_number IS NULL OR qi.product_id IS NULL OR qi.product_name_snapshot IS NULL OR qi.original_unit_price_cents IS NULL OR qi.discounted_unit_price_cents IS NULL OR qi.quantity IS NULL OR qi.original_subtotal_cents IS NULL OR qi.payable_subtotal_cents IS NULL OR qi.line_note IS NULL OR qi.line_number<>line_count OR qi.line_number NOT BETWEEN 1 AND 65535 OR qi.product_id NOT BETWEEN 1 AND 18446744073709551615 OR octet_length(qi.product_source_version) IS DISTINCT FROM 32 OR qi.product_source_version=decode(repeat('00',32),'hex') OR qi.original_unit_price_cents NOT BETWEEN 0 AND maximum_int64 OR qi.discounted_unit_price_cents NOT BETWEEN 0 AND maximum_int64 OR qi.quantity NOT BETWEEN 1 AND maximum_int64 OR qi.original_subtotal_cents NOT BETWEEN 0 AND maximum_int64 OR qi.payable_subtotal_cents NOT BETWEEN 0 AND maximum_int64 OR qi.product_name_snapshot='' OR convert_to(btrim(qi.product_name_snapshot,v_go_space),'UTF8')<>convert_to(qi.product_name_snapshot,'UTF8') OR jsonb_typeof(qi.flavors_json) IS DISTINCT FROM 'array' THEN EXIT snapshot_check;END IF;
 raw_value:=coalesce(qi.image_object_key_snapshot,''::bytea);
 IF qi.image_object_key_snapshot IS NOT NULL THEN
  IF octet_length(raw_value)=0 OR octet_length(raw_value)>1024 THEN EXIT snapshot_check;END IF;
  BEGIN part:=convert_from(raw_value,'UTF8');EXCEPTION WHEN character_not_in_repertoire THEN EXIT snapshot_check;END;
  IF convert_to(btrim(part,v_go_space),'UTF8')<>raw_value THEN EXIT snapshot_check;END IF;
 END IF;
 IF qi.original_unit_price_cents*h.discount_rate_percent>maximum_int64 THEN EXIT snapshot_check;END IF;
 discounted_unit:=div(qi.original_unit_price_cents*h.discount_rate_percent,100);
 IF mod(qi.original_unit_price_cents*h.discount_rate_percent,100)>=50 THEN discounted_unit:=discounted_unit+1;END IF;
 IF discounted_unit<>qi.discounted_unit_price_cents OR qi.original_unit_price_cents*qi.quantity>maximum_int64 OR discounted_unit*qi.quantity>maximum_int64 OR qi.original_subtotal_cents<>qi.original_unit_price_cents*qi.quantity OR qi.payable_subtotal_cents<>discounted_unit*qi.quantity OR original_total+qi.original_subtotal_cents>maximum_int64 OR payable_total+qi.payable_subtotal_cents>maximum_int64 THEN EXIT snapshot_check;END IF;
 original_total:=original_total+qi.original_subtotal_cents;payable_total:=payable_total+qi.payable_subtotal_cents;seen_flavors:=ARRAY[]::text[];
snapshot_material:=snapshot_material||int8send((CASE WHEN (qi.line_number)::numeric>9223372036854775807 THEN (qi.line_number)::numeric-18446744073709551616 ELSE (qi.line_number)::numeric END)::bigint)||int8send((CASE WHEN (qi.product_id)::numeric>9223372036854775807 THEN (qi.product_id)::numeric-18446744073709551616 ELSE (qi.product_id)::numeric END)::bigint)||int8send(octet_length(convert_to(qi.product_name_snapshot,'UTF8'))::bigint)||(convert_to(qi.product_name_snapshot,'UTF8'))||int8send(octet_length(qi.product_source_version)::bigint)||(qi.product_source_version)||int8send((CASE WHEN (CASE WHEN octet_length(raw_value)>0 THEN 1 ELSE 0 END)::numeric>9223372036854775807 THEN (CASE WHEN octet_length(raw_value)>0 THEN 1 ELSE 0 END)::numeric-18446744073709551616 ELSE (CASE WHEN octet_length(raw_value)>0 THEN 1 ELSE 0 END)::numeric END)::bigint)||int8send(octet_length(raw_value)::bigint)||(raw_value)||int8send((CASE WHEN (qi.original_unit_price_cents)::numeric>9223372036854775807 THEN (qi.original_unit_price_cents)::numeric-18446744073709551616 ELSE (qi.original_unit_price_cents)::numeric END)::bigint)||int8send((CASE WHEN (qi.discounted_unit_price_cents)::numeric>9223372036854775807 THEN (qi.discounted_unit_price_cents)::numeric-18446744073709551616 ELSE (qi.discounted_unit_price_cents)::numeric END)::bigint)||int8send((CASE WHEN (qi.quantity)::numeric>9223372036854775807 THEN (qi.quantity)::numeric-18446744073709551616 ELSE (qi.quantity)::numeric END)::bigint)||int8send((CASE WHEN (qi.original_subtotal_cents)::numeric>9223372036854775807 THEN (qi.original_subtotal_cents)::numeric-18446744073709551616 ELSE (qi.original_subtotal_cents)::numeric END)::bigint)||int8send((CASE WHEN (qi.payable_subtotal_cents)::numeric>9223372036854775807 THEN (qi.payable_subtotal_cents)::numeric-18446744073709551616 ELSE (qi.payable_subtotal_cents)::numeric END)::bigint)||int8send((CASE WHEN (jsonb_array_length(qi.flavors_json))::numeric>9223372036854775807 THEN (jsonb_array_length(qi.flavors_json))::numeric-18446744073709551616 ELSE (jsonb_array_length(qi.flavors_json))::numeric END)::bigint);
FOR flavor IN SELECT value FROM jsonb_array_elements(qi.flavors_json) LOOP
 IF jsonb_typeof(flavor) IS DISTINCT FROM 'string' THEN EXIT snapshot_check;END IF;
 part:=flavor#>>'{}';IF part='' OR convert_to(btrim(part,v_go_space),'UTF8')<>convert_to(part,'UTF8') OR encode(convert_to(part,'UTF8'),'hex')=ANY(seen_flavors) THEN EXIT snapshot_check;END IF;
 seen_flavors:=array_append(seen_flavors,encode(convert_to(part,'UTF8'),'hex'));
 snapshot_material:=snapshot_material||int8send(octet_length(convert_to(part,'UTF8'))::bigint)||convert_to(part,'UTF8');
 END LOOP;
 snapshot_material:=snapshot_material||int8send(octet_length(convert_to(qi.line_note,'UTF8'))::bigint)||convert_to(qi.line_note,'UTF8');
 items:=array_append(items,qi);
 END LOOP;
 IF line_count<>h.item_count OR original_total<>h.original_subtotal_cents OR original_total-payable_total<>h.discount_cents OR payable_total<>h.payable_cents THEN EXIT snapshot_check;END IF;
snapshot_material:=snapshot_material||int8send((CASE WHEN (h.original_subtotal_cents)::numeric>9223372036854775807 THEN (h.original_subtotal_cents)::numeric-18446744073709551616 ELSE (h.original_subtotal_cents)::numeric END)::bigint)||int8send((CASE WHEN (h.discount_cents)::numeric>9223372036854775807 THEN (h.discount_cents)::numeric-18446744073709551616 ELSE (h.discount_cents)::numeric END)::bigint)||int8send((CASE WHEN (h.payable_cents)::numeric>9223372036854775807 THEN (h.payable_cents)::numeric-18446744073709551616 ELSE (h.payable_cents)::numeric END)::bigint)||int8send((CASE WHEN (extract(epoch FROM h.created_at)*1000000)::numeric>9223372036854775807 THEN (extract(epoch FROM h.created_at)*1000000)::numeric-18446744073709551616 ELSE (extract(epoch FROM h.created_at)*1000000)::numeric END)::bigint)||int8send((CASE WHEN (extract(epoch FROM h.expires_at)*1000000)::numeric>9223372036854775807 THEN (extract(epoch FROM h.expires_at)*1000000)::numeric-18446744073709551616 ELSE (extract(epoch FROM h.expires_at)*1000000)::numeric END)::bigint);
IF sha256(snapshot_material)<>h.snapshot_digest THEN EXIT snapshot_check;END IF;
 snapshot_valid:=true;
 END;

 IF snapshot_read_unavailable THEN RETURN jsonb_build_object('outcome','QUOTE_UNAVAILABLE_RETRY');END IF;
 IF NOT snapshot_valid THEN defer_reason:='QUOTE_SNAPSHOT_INVALID';EXIT materialization_facts;END IF;
 IF h.id<>pr.quote_id OR h.user_id<>pr.user_id OR h.payable_cents<>pr.expected_amount_cents OR obs.prepayment_id<>pr.id OR obs.out_trade_no<>pr.out_trade_no OR obs.amount_cents<>pr.expected_amount_cents OR obs.currency<>pr.currency OR NOT obs.success_time<pickup_at OR (NOT p_manual AND NOT obs.success_time<pr.effective_deadline) OR decode(encode(convert_to(req->>'appid','UTF8'),'hex'),'hex')<>pr.expected_appid OR convert_to(req->>'mchid','UTF8')<>pr.expected_mchid OR convert_to(req->>'out_trade_no','UTF8')<>pr.out_trade_no OR (req->>'amount_cents')::numeric<>pr.expected_amount_cents OR req->>'currency'<>pr.currency OR req->>'quote_id'<>h.id::text OR req->>'quote_digest'<>encode(h.snapshot_digest,'hex') THEN defer_reason:='PAYMENT_FACT_INVALID';EXIT materialization_facts;END IF;
 materialized_at:=greatest(date_trunc('microseconds',clock_timestamp()),obs.success_time);
 IF pickup_at='0001-01-01T00:00:00Z'::timestamptz OR materialized_at='0001-01-01T00:00:00Z'::timestamptz OR NOT pickup_at>obs.success_time THEN defer_reason:='PAYMENT_TIME_INVALID';EXIT materialization_facts;END IF;
 state:=CASE WHEN pickup_at-obs.success_time<interval '30 minutes' THEN 'PREPARING' ELSE 'RESERVED' END;
 IF state='RESERVED' AND materialized_at>=pickup_at-interval '30 minutes' THEN state:='PREPARING';END IF;
 END;
 IF defer_reason IS NOT NULL THEN
 IF obs.apply_state<>'DEFERRED' OR obs.apply_reason_code IS DISTINCT FROM defer_reason THEN
 UPDATE public.payment_observations SET apply_state='DEFERRED',apply_reason_code=defer_reason,applied_at=NULL,record_version=record_version+1 WHERE id=obs.id AND record_version=obs.record_version AND apply_state IN ('NEW','DEFERRED');END IF;
 IF pr.materialization_state<>'APPLIED' THEN
 UPDATE public.prepayments SET materialization_state='PENDING_MANUAL',pending_reason_code=defer_reason,materialized_at=NULL,lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,record_version=record_version+1,next_reconcile_at=NULL,updated_at=date_trunc('microseconds',clock_timestamp()) WHERE id=pr.id AND record_version=pr.record_version;END IF;
 RETURN jsonb_build_object('outcome','DEFERRED');
 END IF;
 INSERT INTO public.pickup_sequences(service_date,last_number,updated_at) VALUES(h.pickup_date,1,materialized_at)
 ON CONFLICT(service_date) DO UPDATE SET last_number=public.pickup_sequences.last_number+1,updated_at=EXCLUDED.updated_at WHERE public.pickup_sequences.last_number<9999 RETURNING last_number INTO pickup_number;
 IF NOT FOUND OR pickup_number IS NULL OR pickup_number NOT BETWEEN 1 AND 9999 THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_PAYMENT_UNAVAILABLE';END IF;
 INSERT INTO public.orders(order_no,user_id,quote_id,prepayment_id,payment_observation_id,contact_name_snapshot,contact_phone_snapshot,identity_kind,identity_source_version,discount_rate_percent,discount_version,store_name_snapshot,store_address_snapshot,pickup_point_snapshot,pickup_date,pickup_time,pickup_at,meal_period,order_note,item_count,original_subtotal_cents,discount_cents,payable_cents,snapshot_digest,transaction_id,paid_at,materialized_at,pickup_number,state,preparing_at,ready_at,completed_at,refunding_at,refunded_at,redemption_token_ciphertext,redemption_token_hash,redemption_key_version,redemption_issued_at,redeemed_by_account_id,redeemed_at,record_version,created_at,updated_at)
 VALUES(convert_to(replace(h.pickup_date::text,'-','')||'-'||lpad(pickup_number::text,4,'0'),'UTF8'),h.user_id,h.id,pr.id,obs.id,h.contact_name_snapshot,h.contact_phone_snapshot,h.identity_kind,h.identity_source_version,h.discount_rate_percent,h.discount_version,h.store_name_snapshot,h.store_address_snapshot,h.pickup_point_snapshot,h.pickup_date,h.pickup_time,pickup_at,h.meal_period,h.order_note,h.item_count,h.original_subtotal_cents,h.discount_cents,h.payable_cents,h.snapshot_digest,obs.transaction_id,obs.success_time,materialized_at,pickup_number,state,CASE WHEN state='PREPARING' THEN materialized_at ELSE NULL END,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,1,materialized_at,materialized_at) RETURNING id INTO order_id;
 IF order_id IS NULL OR order_id=0 THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_PAYMENT_UNAVAILABLE';END IF;
 FOREACH qi IN ARRAY items LOOP
 INSERT INTO public.order_items(order_id,line_number,product_id,product_name_snapshot,product_source_version,image_object_key_snapshot,original_unit_price_cents,discounted_unit_price_cents,quantity,original_subtotal_cents,payable_subtotal_cents,flavors_json,line_note)
 VALUES(order_id,qi.line_number,qi.product_id,qi.product_name_snapshot,qi.product_source_version,qi.image_object_key_snapshot,qi.original_unit_price_cents,qi.discounted_unit_price_cents,qi.quantity,qi.original_subtotal_cents,qi.payable_subtotal_cents,qi.flavors_json,qi.line_note);
 END LOOP;
 UPDATE public.prepayments SET provider_state='PAID',materialization_state='APPLIED',pending_reason_code=NULL,materialized_at=payment_command.materialized_at,lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,record_version=record_version+1,next_reconcile_at=NULL,updated_at=payment_command.materialized_at WHERE id=pr.id AND record_version=pr.record_version AND materialization_state IN ('READY','PENDING_MANUAL');
 GET DIAGNOSTICS affected=ROW_COUNT;IF affected<>1 THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_PAYMENT_UNAVAILABLE';END IF;
 UPDATE public.payment_observations SET apply_state='APPLIED',apply_reason_code=NULL,applied_at=materialized_at,record_version=record_version+1 WHERE id=obs.id AND record_version=obs.record_version AND apply_state IN ('NEW','DEFERRED');
 GET DIAGNOSTICS affected=ROW_COUNT;IF affected<>1 THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_PAYMENT_UNAVAILABLE';END IF;
 RETURN jsonb_build_object('outcome','APPLIED','order_id',order_id::text);
 EXCEPTION WHEN OTHERS THEN
 GET STACKED DIAGNOSTICS failure_state=RETURNED_SQLSTATE,failure_constraint=CONSTRAINT_NAME;
 IF failure_state IN ('40P01','40001','55P03') THEN RETURN jsonb_build_object('outcome','SQL_FAILURE','sqlstate',failure_state,'constraint','');END IF;

 IF failure_state='P1000' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 RAISE;
 END;
END
$order_rpc$;
