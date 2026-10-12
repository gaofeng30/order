-- Fixed payment ABI; raw frozen aggregates stay inside each atomic RPC.

CREATE OR REPLACE FUNCTION public.order_payment_due_v1(p_now timestamptz,p_limit integer)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
<<payment_command>>
DECLARE ids jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 SELECT coalesce(jsonb_agg(id::text ORDER BY id),'[]'::jsonb) INTO ids FROM (SELECT id FROM public.prepayments WHERE materialization_state='READY' OR (materialization_state='AWAITING_PAYMENT' AND next_reconcile_at<=p_now) OR (materialization_state='AWAITING_PAYMENT' AND lease_expires_at<=p_now) ORDER BY id LIMIT p_limit) due;RETURN jsonb_build_object('outcome','FOUND','ids',ids);

 END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_payment_claim_query_v1(p_prepayment numeric,p_cutoff timestamptz,p_force boolean,p_owner_hex text,p_lease_ns numeric)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
<<payment_command>>
DECLARE pr public.prepayments%ROWTYPE;req jsonb;request_json text;request_field record;canonical_fields jsonb;folded_field text;lease_now timestamptz;lease_expiry timestamptz;expiry_nanos numeric;expiry_text text;next_state text;affected bigint;
 failure_state text;failure_constraint text;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
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

 lease_now:=date_trunc('microseconds',clock_timestamp());
 IF pr.provider_state IN ('PAID','CLOSED') OR pr.materialization_state IN ('APPLIED','PENDING_MANUAL') THEN RETURN jsonb_build_object('outcome','NOT_CLAIMED');END IF;
 IF pr.provider_state='READY' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF pr.lease_kind IS NOT NULL AND pr.lease_expires_at IS NOT NULL AND pr.lease_expires_at>lease_now THEN RETURN jsonb_build_object('outcome','NOT_CLAIMED');END IF;
 IF NOT p_force AND pr.next_reconcile_at IS NOT NULL AND pr.next_reconcile_at>p_cutoff THEN RETURN jsonb_build_object('outcome','NOT_CLAIMED');END IF;
 IF p_lease_ns IS NULL OR p_lease_ns<>trunc(p_lease_ns) OR p_lease_ns NOT BETWEEN 1 AND 9223372036854775807 OR p_owner_hex !~ '^[0-9a-f]{32}$' OR p_owner_hex=repeat('0',32) THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 lease_expiry:=(lease_now+make_interval(days=>div(p_lease_ns,86400000000000)::integer,secs=>div(mod(p_lease_ns,86400000000000),1000000000)::double precision)+(div(mod(p_lease_ns,1000000000),1000)::text||' microseconds')::interval);
 expiry_nanos:=mod(extract(epoch FROM lease_now)*1000000000+p_lease_ns,1000000000);
 expiry_text:=to_char(lease_expiry,'YYYY-MM-DD"T"HH24:MI:SS')||CASE WHEN expiry_nanos=0 THEN '' ELSE '.'||rtrim(lpad(trunc(expiry_nanos)::text,9,'0'),'0') END||'Z';
 next_state:=CASE WHEN pr.provider_state='CREATE_CLAIMED' THEN 'CREATE_UNKNOWN' ELSE pr.provider_state END;
 UPDATE public.prepayments SET provider_state=next_state,lease_kind='QUERY',lease_owner=decode(p_owner_hex,'hex'),lease_expires_at=lease_expiry,record_version=record_version+1,next_reconcile_at=lease_expiry,updated_at=lease_now WHERE id=pr.id AND record_version=pr.record_version;
 GET DIAGNOSTICS affected=ROW_COUNT;IF affected<>1 THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_PAYMENT_UNAVAILABLE';END IF;
 -- The original claim returns the old row plus only its state/version changes.
 pr.provider_state:=next_state;pr.record_version:=pr.record_version+1;
 RETURN jsonb_build_object('outcome','CLAIMED','record',jsonb_build_object('id',pr.id::text,'user_id',pr.user_id::text,'quote_id',pr.quote_id::text,'idempotency_key_hash',encode(pr.idempotency_key_hash,'hex'),'out_trade_no',encode(pr.out_trade_no,'hex'),'expected_appid',encode(pr.expected_appid,'hex'),'expected_mchid',encode(pr.expected_mchid,'hex'),'expected_amount_cents',pr.expected_amount_cents::text,'currency',pr.currency,'provider_create_request_json',pr.provider_create_request_json::text,'provider_create_request_digest',encode(pr.provider_create_request_digest,'hex'),'effective_deadline',pr.effective_deadline,'provider_state',pr.provider_state,'create_attempted_at',pr.create_attempted_at,'wx_request_payment_json',pr.wx_request_payment_json::text,'provider_prepay_id',encode(pr.provider_prepay_id,'hex'),'last_queried_at',pr.last_queried_at,'materialization_state',pr.materialization_state,'pending_reason_code',pr.pending_reason_code,'materialized_at',pr.materialized_at,'lease_kind',pr.lease_kind,'lease_owner',encode(pr.lease_owner,'hex'),'lease_expires_at',pr.lease_expires_at,'record_version',pr.record_version::text,'next_reconcile_at',pr.next_reconcile_at,'created_at',pr.created_at,'updated_at',pr.updated_at),'owner_hex',p_owner_hex,'version',pr.record_version::text,'expires_at',expiry_text);
 EXCEPTION WHEN OTHERS THEN
 GET STACKED DIAGNOSTICS failure_state=RETURNED_SQLSTATE,failure_constraint=CONSTRAINT_NAME;
 IF failure_state IN ('40P01','40001','55P03') THEN RETURN jsonb_build_object('outcome','SQL_FAILURE','sqlstate',failure_state,'constraint','');END IF;

 IF failure_state='P1000' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 RAISE;
 END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_payment_finish_query_v1(p_prepayment numeric,p_owner_hex text,p_version numeric,p_now timestamptz,p_next timestamptz,p_close boolean)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
<<payment_command>>
DECLARE
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF p_close THEN
 UPDATE public.prepayments SET provider_state='CLOSED',lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,record_version=record_version+1,next_reconcile_at=NULL,updated_at=p_now WHERE id=p_prepayment AND lease_kind='QUERY' AND lease_owner=decode(p_owner_hex,'hex') AND record_version=p_version AND lease_expires_at>p_now;
 ELSE
 UPDATE public.prepayments SET lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,record_version=record_version+1,next_reconcile_at=p_next,updated_at=p_now WHERE id=p_prepayment AND lease_kind='QUERY' AND lease_owner=decode(p_owner_hex,'hex') AND record_version=p_version AND lease_expires_at>p_now;
 END IF;RETURN jsonb_build_object('outcome','FINISHED');

 END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_payment_observe_v1(p_expected_prepayment numeric,p_source text,p_event_hex text,p_transaction jsonb,p_claim jsonb,p_received_at timestamptz,p_reconcile_ns numeric)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
<<payment_command>>
DECLARE pr public.prepayments%ROWTYPE;req jsonb;request_json text;request_field record;canonical_fields jsonb;folded_field text;total numeric;secs numeric;nanos numeric;success_zero boolean;success_at timestamptz;pg_micros numeric;success_text text;claim_matches boolean;
state text;mismatch text;validation text;dedupe bytea;mode text;reason text;affected bigint;existing_prepayment numeric;
next_state text;next_material text;pending_reason text;next_reconcile timestamptz;last_queried timestamptz;preserve_lease boolean;
 failure_state text;failure_constraint text;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF jsonb_typeof(p_transaction) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(p_transaction))<>18 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(p_transaction->'appid_hex') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(p_transaction->'mchid_hex') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(p_transaction->'out_trade_no_hex') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(p_transaction->'transaction_id_hex') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(p_transaction->'trade_type_hex') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(p_transaction->'trade_state_hex') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(p_transaction->'trade_state_desc_hex') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(p_transaction->'bank_type_hex') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(p_transaction->'attach_hex') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(p_transaction->'payer_openid_hex') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(p_transaction->'currency_hex') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(p_transaction->'payer_currency_hex') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(p_transaction->'total') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(p_transaction->'payer_total') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(p_transaction->'success_zero') IS DISTINCT FROM 'boolean' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(p_transaction->'success_epoch_seconds') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(p_transaction->'success_nanosecond') IS DISTINCT FROM 'number' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF jsonb_typeof(p_transaction->'success_rfc3339nano') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF p_transaction->>'appid_hex' !~ '^([0-9a-f]{2})*$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF p_transaction->>'mchid_hex' !~ '^([0-9a-f]{2})*$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF p_transaction->>'out_trade_no_hex' !~ '^([0-9a-f]{2})*$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF p_transaction->>'transaction_id_hex' !~ '^([0-9a-f]{2})*$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF p_transaction->>'trade_type_hex' !~ '^([0-9a-f]{2})*$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF p_transaction->>'trade_state_hex' !~ '^([0-9a-f]{2})*$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF p_transaction->>'trade_state_desc_hex' !~ '^([0-9a-f]{2})*$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF p_transaction->>'bank_type_hex' !~ '^([0-9a-f]{2})*$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF p_transaction->>'attach_hex' !~ '^([0-9a-f]{2})*$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF p_transaction->>'payer_openid_hex' !~ '^([0-9a-f]{2})*$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF p_transaction->>'currency_hex' !~ '^([0-9a-f]{2})*$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF p_transaction->>'payer_currency_hex' !~ '^([0-9a-f]{2})*$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
IF p_transaction->>'total' !~ '^(0|-?[1-9][0-9]*)$' OR p_transaction->>'payer_total' !~ '^(0|-?[1-9][0-9]*)$' OR p_transaction->>'success_epoch_seconds' !~ '^(0|-?[1-9][0-9]*)$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 total:=(p_transaction->>'total')::numeric;secs:=(p_transaction->>'success_epoch_seconds')::numeric;nanos:=(p_transaction->>'success_nanosecond')::numeric;success_zero:=(p_transaction->>'success_zero')::boolean;
 IF total NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR (p_transaction->>'payer_total')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR secs NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR nanos<>trunc(nanos) OR nanos NOT BETWEEN 0 AND 999999999 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 -- The fixed Go encoder derives this canonical digest text and numeric instant from one UTC time.Time.
 success_text:=p_transaction->>'success_rfc3339nano';
 IF success_zero<>(secs=-62135596800 AND nanos=0) THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF p_expected_prepayment>0 THEN SELECT * INTO pr FROM public.prepayments WHERE id=p_expected_prepayment FOR UPDATE;ELSE SELECT * INTO pr FROM public.prepayments WHERE out_trade_no=decode(p_transaction->>'out_trade_no_hex','hex') FOR UPDATE;END IF;
 IF NOT FOUND THEN RETURN jsonb_build_object('outcome','NOT_FOUND');END IF;
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

 claim_matches:=false;
 IF p_claim IS NOT NULL THEN
 IF jsonb_typeof(p_claim) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(p_claim))<>2 OR jsonb_typeof(p_claim->'owner_hex') IS DISTINCT FROM 'string' OR jsonb_typeof(p_claim->'version') IS DISTINCT FROM 'string' OR p_claim->>'owner_hex' !~ '^[0-9a-f]{32}$' OR p_claim->>'version' !~ '^[1-9][0-9]*$' OR (p_claim->>'version')::numeric>18446744073709551615 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 claim_matches:=pr.lease_kind='QUERY' AND pr.lease_owner=decode(p_claim->>'owner_hex','hex') AND pr.record_version=(p_claim->>'version')::numeric;
 IF NOT coalesce(claim_matches,false) OR pr.lease_expires_at IS NULL OR NOT p_received_at<pr.lease_expires_at THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 END IF;
 IF pr.expected_appid=''::bytea OR pr.expected_mchid=''::bytea OR pr.out_trade_no=''::bytea OR position(decode('00','hex') IN pr.expected_appid)>0 OR position(decode('00','hex') IN pr.expected_mchid)>0 OR position(decode('00','hex') IN pr.out_trade_no)>0 OR p_source NOT IN ('CALLBACK','ACTIVE_QUERY') THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
 IF position(decode('00','hex') IN decode(p_transaction->>'appid_hex','hex'))>0 THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
IF position(decode('00','hex') IN decode(p_transaction->>'mchid_hex','hex'))>0 THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
IF position(decode('00','hex') IN decode(p_transaction->>'out_trade_no_hex','hex'))>0 THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
IF position(decode('00','hex') IN decode(p_transaction->>'transaction_id_hex','hex'))>0 THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
IF position(decode('00','hex') IN decode(p_transaction->>'trade_type_hex','hex'))>0 THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
IF position(decode('00','hex') IN decode(p_transaction->>'trade_state_hex','hex'))>0 THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
IF position(decode('00','hex') IN decode(p_transaction->>'trade_state_desc_hex','hex'))>0 THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
IF position(decode('00','hex') IN decode(p_transaction->>'bank_type_hex','hex'))>0 THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
IF position(decode('00','hex') IN decode(p_transaction->>'attach_hex','hex'))>0 THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
IF position(decode('00','hex') IN decode(p_transaction->>'payer_openid_hex','hex'))>0 THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
IF position(decode('00','hex') IN decode(p_transaction->>'currency_hex','hex'))>0 THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
IF position(decode('00','hex') IN decode(p_transaction->>'payer_currency_hex','hex'))>0 THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
IF p_transaction->>'appid_hex'='' OR p_transaction->>'mchid_hex'='' OR p_transaction->>'out_trade_no_hex'='' OR p_transaction->>'trade_state_hex'='' THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
 state:=CASE p_transaction->>'trade_state_hex' WHEN '53554343455353' THEN 'PAID' WHEN '4e4f54504159' THEN 'NOT_PAID' WHEN '434c4f534544' THEN 'CLOSED' ELSE NULL END;
 IF state IS NULL THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
 IF state='PAID' THEN IF p_transaction->>'transaction_id_hex'='' OR success_zero OR total<=0 OR p_transaction->>'currency_hex'='' THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
 ELSE IF total<0 OR (total=0)<>(p_transaction->>'currency_hex'='') THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;END IF;
 IF p_source='CALLBACK' AND state<>'PAID' THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
 -- Match pgx's binary timestamptz encoding: integer microseconds since 2000, including signed int64 wrap.
 -- Normalize and the digest below continue to use the original Go seconds/nanoseconds and canonical text.
 IF state='PAID' THEN
 pg_micros:=mod(secs*1000000+div(nanos,1000)-946684800000000,18446744073709551616);
 IF pg_micros<0 THEN pg_micros:=pg_micros+18446744073709551616;END IF;
 IF pg_micros>=9223372036854775808 THEN pg_micros:=pg_micros-18446744073709551616;END IF;
 IF pg_micros=9223372036854775807 THEN success_at:='infinity'::timestamptz;
 ELSIF pg_micros=-9223372036854775808 THEN success_at:='-infinity'::timestamptz;
 ELSE success_at:='2000-01-01T00:00:00Z'::timestamptz+(div(pg_micros,86400000000)::text||' days '||mod(pg_micros,86400000000)::text||' microseconds')::interval;END IF;
 END IF;
 mismatch:=CASE WHEN pr.expected_appid<>decode(p_transaction->>'appid_hex','hex') THEN 'APP_ID' WHEN pr.expected_mchid<>decode(p_transaction->>'mchid_hex','hex') THEN 'MERCHANT_ID' WHEN pr.out_trade_no<>decode(p_transaction->>'out_trade_no_hex','hex') THEN 'OUT_TRADE_NO' WHEN (state='PAID' OR total<>0) AND pr.expected_amount_cents<>total THEN 'TOTAL_AMOUNT' WHEN (state='PAID' OR p_transaction->>'currency_hex'<>'') AND convert_to(pr.currency,'UTF8')<>decode(p_transaction->>'currency_hex','hex') THEN 'CURRENCY' ELSE 'NONE' END;
 validation:=CASE WHEN mismatch='NONE' THEN 'ACCEPTED' ELSE 'REJECTED_MISMATCH' END;
 dedupe:=sha256(convert_to('order.payment-observation.v1','UTF8')||decode('00','hex')||convert_to(validation,'UTF8')||decode('00','hex')||convert_to(mismatch,'UTF8')||decode('00','hex')||convert_to(state,'UTF8')||decode('00','hex')||pr.expected_appid||decode('00','hex')||decode(p_transaction->>'appid_hex','hex')||decode('00','hex')||pr.expected_mchid||decode('00','hex')||decode(p_transaction->>'mchid_hex','hex')||decode('00','hex')||pr.out_trade_no||decode('00','hex')||decode(p_transaction->>'out_trade_no_hex','hex')||decode('00','hex')||convert_to(pr.expected_amount_cents::text,'UTF8')||decode('00','hex')||convert_to(CASE WHEN total=0 THEN '' ELSE total::text END,'UTF8')||decode('00','hex')||convert_to(pr.currency,'UTF8')||decode('00','hex')||decode(p_transaction->>'currency_hex','hex')||decode('00','hex')||decode(p_transaction->>'transaction_id_hex','hex')||decode('00','hex')||convert_to(CASE WHEN success_zero THEN '' ELSE success_text END,'UTF8'));
mode:=CASE WHEN validation='ACCEPTED' AND state='PAID' AND NOT success_zero AND secs*1000000000+nanos<extract(epoch FROM pr.effective_deadline)*1000000000 THEN 'AUTO' ELSE 'DELAYED_MANUAL' END;
 reason:=CASE WHEN validation<>'ACCEPTED' THEN 'PAYMENT_'||mismatch||'_MISMATCH' WHEN state<>'PAID' THEN 'PAYMENT_NOT_SUCCESSFUL' ELSE 'PAYMENT_AT_OR_AFTER_DEADLINE' END;
 INSERT INTO public.payment_observations(prepayment_id,dedupe_key,source,provider_event_id,out_trade_no,transaction_id,provider_state,validation,mismatch_code,amount_cents,currency,success_time,received_at,materialization_mode,apply_state,apply_reason_code,applied_at,record_version)
 VALUES(pr.id,dedupe,p_source,CASE WHEN p_event_hex='' THEN NULL ELSE decode(p_event_hex,'hex') END,pr.out_trade_no,CASE WHEN state='PAID' THEN decode(p_transaction->>'transaction_id_hex','hex') ELSE NULL END,state,CASE WHEN validation='ACCEPTED' THEN 'MATCH' ELSE 'MISMATCH' END,CASE WHEN mismatch='NONE' THEN NULL ELSE mismatch END,CASE WHEN state='PAID' THEN total ELSE NULL END,CASE WHEN state='PAID' THEN convert_from(decode(p_transaction->>'currency_hex','hex'),'UTF8') ELSE NULL END,CASE WHEN state='PAID' THEN success_at ELSE NULL END,p_received_at,mode,CASE WHEN mode='AUTO' THEN 'NEW' ELSE 'DEFERRED' END,CASE WHEN mode='AUTO' THEN NULL ELSE reason END,NULL,1)
 ON CONFLICT ON CONSTRAINT uq_payment_observations_dedupe DO NOTHING;
 GET DIAGNOSTICS affected=ROW_COUNT;
 IF affected=0 THEN SELECT prepayment_id INTO existing_prepayment FROM public.payment_observations WHERE dedupe_key=dedupe;IF NOT FOUND OR existing_prepayment<>pr.id THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_PAYMENT_UNAVAILABLE';END IF;END IF;
 next_state:=CASE WHEN pr.provider_state='PAID' THEN 'PAID' WHEN state='PAID' THEN 'PAID' WHEN pr.provider_state='CLOSED' THEN 'CLOSED' WHEN state='CLOSED' THEN 'CLOSED' ELSE 'NOT_PAID' END;
 next_material:=pr.materialization_state;pending_reason:=pr.pending_reason_code;
 IF next_material<>'APPLIED' THEN IF mode='AUTO' THEN next_material:='READY';pending_reason:=NULL;ELSIF state='PAID' OR validation<>'ACCEPTED' THEN next_material:='PENDING_MANUAL';pending_reason:=reason;END IF;END IF;
 next_reconcile:=NULL;IF next_material='READY' THEN next_reconcile:=p_received_at;ELSIF next_material='AWAITING_PAYMENT' AND next_state<>'CLOSED' THEN next_reconcile:=(p_received_at+make_interval(days=>div(p_reconcile_ns,86400000000000)::integer,secs=>div(mod(p_reconcile_ns,86400000000000),1000000000)::double precision)+(div(mod(p_reconcile_ns,1000000000),1000)::text||' microseconds')::interval);END IF;
 last_queried:=pr.last_queried_at;IF p_source='ACTIVE_QUERY' THEN last_queried:=p_received_at;END IF;
 preserve_lease:=false;
 IF p_source='ACTIVE_QUERY' THEN IF pr.lease_kind IS NOT NULL AND pr.lease_kind<>'QUERY' THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_PAYMENT_UNAVAILABLE';END IF;preserve_lease:=NOT coalesce(claim_matches,false) AND pr.lease_kind IS NOT NULL;END IF;
 IF preserve_lease THEN next_reconcile:=pr.next_reconcile_at;END IF;
 UPDATE public.prepayments SET provider_state=next_state,last_queried_at=last_queried,materialization_state=next_material,pending_reason_code=pending_reason,lease_kind=CASE WHEN preserve_lease THEN pr.lease_kind ELSE NULL END,lease_owner=CASE WHEN preserve_lease THEN pr.lease_owner ELSE NULL END,lease_expires_at=CASE WHEN preserve_lease THEN pr.lease_expires_at ELSE NULL END,record_version=record_version+1,next_reconcile_at=next_reconcile,updated_at=p_received_at WHERE id=pr.id AND record_version=pr.record_version;
 GET DIAGNOSTICS affected=ROW_COUNT;IF affected<>1 THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_PAYMENT_UNAVAILABLE';END IF;
 RETURN jsonb_build_object('outcome','OBSERVED');
 EXCEPTION WHEN OTHERS THEN
 GET STACKED DIAGNOSTICS failure_state=RETURNED_SQLSTATE,failure_constraint=CONSTRAINT_NAME;
 IF failure_state IN ('40P01','40001','55P03') THEN RETURN jsonb_build_object('outcome','SQL_FAILURE','sqlstate',failure_state,'constraint','');END IF;

 IF failure_state='P1000' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 RAISE;
 END;
END
$order_rpc$;
