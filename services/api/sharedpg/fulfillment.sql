-- Fixed fulfillment RPCs. Token request JSON stays text: PostgreSQL must not decode an allowed NUL token.

CREATE OR REPLACE FUNCTION public.order_fulfillment_proof_v1(p_actor numeric,p_kind text,p_order numeric,p_key text,p_request text,p_locator_date date,p_locator_live boolean,p_token_hash text)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE
  a public.merchant_accounts%ROWTYPE; o public.orders%ROWTYPE; receipt public.action_audits%ROWTYPE;
  scope bytea; operation bytea; request_digest text; response jsonb; proof jsonb;
  token_json text; pickup_code integer; affected bigint; constraint_name text;
  command_action text; live boolean; receipt_count integer:=0;
BEGIN
  IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
  END IF;
  IF p_actor IS NULL OR p_actor<>trunc(p_actor) OR p_actor<=0 OR p_actor>18446744073709551615
    OR p_key IS NULL OR p_key !~ '^[!-~]{1,128}$' THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
  p_actor:=trunc(p_actor);
  IF p_kind IS NULL OR p_kind NOT IN ('MARK_READY','REDEEM_TOKEN','REDEEM_CURRENT_DATE_CODE')
    OR (p_order IS NOT NULL AND (p_order<>trunc(p_order) OR p_order<=0 OR p_order>18446744073709551615))
    OR ((p_order IS NULL)<>(p_locator_live IS NULL)) THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
  p_order:=trunc(p_order);
  CASE p_kind
  WHEN 'MARK_READY' THEN
    IF p_order IS NULL OR p_locator_live IS DISTINCT FROM true OR p_locator_date IS NOT NULL OR p_token_hash IS NOT NULL THEN
      RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
    END IF;
  IF p_request IS DISTINCT FROM '{"kind":"MARK_READY","order_id":'||p_order::text||'}' THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
    command_action:='fulfillment.mark_ready';
  WHEN 'REDEEM_TOKEN' THEN
  IF p_request IS NULL OR octet_length(p_request)>1024
    OR left(p_request,length('{"kind":"REDEEM_TOKEN","token":'))<>'{"kind":"REDEEM_TOKEN","token":'
    OR right(p_request,1)<>'}' OR p_token_hash IS NULL OR p_token_hash !~ '^[0-9a-f]{64}$'
    OR p_locator_date IS NOT NULL THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
  token_json:=substring(p_request FROM length('{"kind":"REDEEM_TOKEN","token":')+1
    FOR length(p_request)-length('{"kind":"REDEEM_TOKEN","token":')-1);
  IF octet_length(token_json) NOT BETWEEN 3 AND 770 OR left(token_json,1)<>'"' OR right(token_json,1)<>'"' THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
  BEGIN
    IF json_typeof(token_json::json) IS DISTINCT FROM 'string' THEN
      RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
    END IF;
  EXCEPTION WHEN invalid_text_representation THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END;
    command_action:='fulfillment.redeem_token';
  WHEN 'REDEEM_CURRENT_DATE_CODE' THEN
    IF p_token_hash IS NOT NULL THEN RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT'; END IF;
  IF p_request IS NULL OR p_request !~ '^\{"kind":"REDEEM_CURRENT_DATE_CODE","pickup_number":"[0-9]{4}"\}$'
    OR p_locator_date IS NULL OR NOT isfinite(p_locator_date)
    OR to_char(p_locator_date,'YYYY-MM-DD') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
  pickup_code:=substring(p_request FROM length('{"kind":"REDEEM_CURRENT_DATE_CODE","pickup_number":"')+1 FOR 4)::integer;
  IF pickup_code NOT BETWEEN 1 AND 9999 THEN RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT'; END IF;
    command_action:='fulfillment.redeem_current_date_code';
  END CASE;
  request_digest:=encode(sha256(convert_to(p_request,'UTF8')),'hex');
  SELECT * INTO a FROM public.merchant_accounts WHERE bound_user_id=p_actor FOR SHARE;
  IF NOT FOUND OR NOT a.enabled THEN RAISE EXCEPTION USING ERRCODE='P1002',MESSAGE='ORDER_FORBIDDEN'; END IF;
  IF a.id<=0 OR a.role NOT IN ('OWNER','SUBACCOUNT') OR a.record_version<=0 OR a.auth_version<=0 THEN
    RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_UNAVAILABLE';
  END IF;
  scope:=sha256(convert_to('merchant:'||p_actor::text||':'||a.id::text,'UTF8'));
  operation:=sha256(convert_to('operation:'||p_key,'UTF8'));
  live:=p_locator_live;
  IF p_order IS NULL THEN
    IF p_kind='REDEEM_TOKEN' THEN
      SELECT id INTO p_order FROM public.orders WHERE redemption_token_hash=decode(p_token_hash,'hex') LIMIT 1;
    ELSE
      SELECT id INTO p_order FROM public.orders WHERE pickup_date=p_locator_date AND pickup_number=pickup_code LIMIT 1;
    END IF;
    live:=FOUND AND p_order>0;
    IF NOT live THEN
      p_order:=NULL;
      FOR receipt IN SELECT * FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='MERCHANT'
        AND actor_scope_hash=scope AND action=command_action AND operation_key_hash=operation ORDER BY id LIMIT 2 LOOP
        receipt_count:=receipt_count+1;
    IF jsonb_typeof(receipt.before_state_json) IS DISTINCT FROM 'object' OR octet_length(receipt.before_state_json::text)>256
      OR (SELECT count(*) FROM jsonb_object_keys(receipt.before_state_json))<>1
      OR jsonb_typeof(receipt.before_state_json->'request_digest') IS DISTINCT FROM 'string'
      OR receipt.before_state_json->>'request_digest' IS NULL OR receipt.before_state_json->>'request_digest' !~ '^[0-9a-fA-F]{64}$' THEN
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
      OR (response->>'order_id')::numeric>18446744073709551615
      OR response->>'state' IS DISTINCT FROM 'COMPLETED' OR response->'changed' IS DISTINCT FROM 'true'::jsonb
      OR response->>'order_id' IS DISTINCT FROM receipt.target_id::text THEN
      RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_RECEIPT_CORRUPT';
    END IF;
        p_order:=receipt.target_id;
      END LOOP;
      IF receipt_count>1 THEN RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_RECEIPT_CORRUPT'; END IF;
      IF receipt_count=0 OR p_order IS NULL OR p_order<=0 THEN RAISE EXCEPTION USING ERRCODE='P1009',MESSAGE='ORDER_REDEMPTION_INVALID'; END IF;
    END IF;
  END IF;
  SELECT * INTO o FROM public.orders WHERE id=p_order FOR SHARE;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P1003',MESSAGE='ORDER_NOT_FOUND'; END IF;
  IF o.id<=0 OR o.record_version<=0 OR o.pickup_number NOT BETWEEN 1 AND 9999
    OR to_char(o.pickup_date,'YYYY-MM-DD') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
    RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_UNAVAILABLE';
  END IF;
  proof:=jsonb_build_object('id',o.id::text,'state',o.state,'version',o.record_version::text,
    'pickup_date',to_char(o.pickup_date,'YYYY-MM-DD'),'pickup_number',o.pickup_number,
    'preparing_at',o.preparing_at,'ready_at',o.ready_at,'completed_at',o.completed_at,
    'hash',encode(o.redemption_token_hash,'hex'),'ciphertext',encode(o.redemption_token_ciphertext,'hex'),
    'key_version',o.redemption_key_version,'issued_at',o.redemption_issued_at,
    'redeemed_by',o.redeemed_by_account_id::text,'redeemed_at',o.redeemed_at,
    'user_id',o.user_id::text,'order_number_hex',encode(o.order_no,'hex'),'pickup_time',to_char(o.pickup_date+o.pickup_time,'HH24:MI'),
    'pickup_point',o.pickup_point_snapshot);
  RETURN jsonb_build_object('proof',proof,'locator_live',live,'receipt_exists',EXISTS(
    SELECT 1 FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='MERCHANT'
    AND actor_scope_hash=scope AND action=command_action AND operation_key_hash=operation));
END;
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_redeem_token_v1(p_actor numeric,p_order numeric,p_key text,p_request_id text,p_now timestamptz,p_request text,p_locator_live boolean,p_locator_date date,p_proof jsonb,p_token_hash text,p_receipt_at timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE
  a public.merchant_accounts%ROWTYPE; o public.orders%ROWTYPE; receipt public.action_audits%ROWTYPE;
  scope bytea; operation bytea; request_digest text; response jsonb; proof jsonb;
  token_json text; pickup_code integer; affected bigint; constraint_name text;
  replaying boolean:=false;
BEGIN
  IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
  END IF;
  IF p_actor IS NULL OR p_actor<>trunc(p_actor) OR p_actor<=0 OR p_actor>18446744073709551615
    OR p_key IS NULL OR p_key !~ '^[!-~]{1,128}$' THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
  p_actor:=trunc(p_actor);
  IF p_order IS NULL OR p_order<>trunc(p_order) OR p_order<=0 OR p_order>18446744073709551615
    OR p_request_id IS NULL OR p_request_id !~ '^[!-~]{1,64}$' OR (p_now IS NOT NULL AND NOT isfinite(p_now)) THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
  p_order:=trunc(p_order);
  IF p_request IS NULL OR octet_length(p_request)>1024
    OR left(p_request,length('{"kind":"REDEEM_TOKEN","token":'))<>'{"kind":"REDEEM_TOKEN","token":'
    OR right(p_request,1)<>'}' OR p_token_hash IS NULL OR p_token_hash !~ '^[0-9a-f]{64}$'
    OR p_locator_date IS NOT NULL THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
  token_json:=substring(p_request FROM length('{"kind":"REDEEM_TOKEN","token":')+1
    FOR length(p_request)-length('{"kind":"REDEEM_TOKEN","token":')-1);
  IF octet_length(token_json) NOT BETWEEN 3 AND 770 OR left(token_json,1)<>'"' OR right(token_json,1)<>'"' THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
  BEGIN
    IF json_typeof(token_json::json) IS DISTINCT FROM 'string' THEN
      RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
    END IF;
  EXCEPTION WHEN invalid_text_representation THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END;
  IF p_locator_live IS NULL THEN RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT'; END IF;
  request_digest:=encode(sha256(convert_to(p_request,'UTF8')),'hex');
  SELECT * INTO a FROM public.merchant_accounts WHERE bound_user_id=p_actor FOR SHARE;
  IF NOT FOUND OR NOT a.enabled THEN RAISE EXCEPTION USING ERRCODE='P1002',MESSAGE='ORDER_FORBIDDEN'; END IF;
  IF a.id<=0 OR a.role NOT IN ('OWNER','SUBACCOUNT') OR a.record_version<=0 OR a.auth_version<=0 THEN
    RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_UNAVAILABLE';
  END IF;
  scope:=sha256(convert_to('merchant:'||p_actor::text||':'||a.id::text,'UTF8'));
  operation:=sha256(convert_to('operation:'||p_key,'UTF8'));
  SELECT * INTO o FROM public.orders WHERE id=p_order FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P1003',MESSAGE='ORDER_NOT_FOUND'; END IF;
  IF o.id<=0 OR o.record_version<=0 OR o.pickup_number NOT BETWEEN 1 AND 9999
    OR to_char(o.pickup_date,'YYYY-MM-DD') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
    RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_UNAVAILABLE';
  END IF;
  SELECT * INTO receipt FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='MERCHANT'
    AND actor_scope_hash=scope AND action='fulfillment.redeem_token' AND operation_key_hash=operation FOR SHARE;
  IF FOUND THEN
    IF jsonb_typeof(receipt.before_state_json) IS DISTINCT FROM 'object' OR octet_length(receipt.before_state_json::text)>256
      OR (SELECT count(*) FROM jsonb_object_keys(receipt.before_state_json))<>1
      OR jsonb_typeof(receipt.before_state_json->'request_digest') IS DISTINCT FROM 'string'
      OR receipt.before_state_json->>'request_digest' IS NULL OR receipt.before_state_json->>'request_digest' !~ '^[0-9a-fA-F]{64}$' THEN
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
      OR (response->>'order_id')::numeric>18446744073709551615
      OR response->>'state' IS DISTINCT FROM 'COMPLETED' OR response->'changed' IS DISTINCT FROM 'true'::jsonb
      OR response->>'order_id' IS DISTINCT FROM p_order::text THEN
      RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_RECEIPT_CORRUPT';
    END IF;
    RETURN jsonb_build_object('order_id',p_order::text,'state','COMPLETED','changed',true,'replay',true);
  END IF;
  IF p_now IS NULL THEN RAISE EXCEPTION USING ERRCODE='P1005',MESSAGE='ORDER_STALE_PROOF'; END IF;
  IF NOT p_locator_live OR o.redemption_token_hash IS NULL OR octet_length(o.redemption_token_hash)<>32 OR o.redemption_token_hash<>decode(p_token_hash,'hex') THEN
    RAISE EXCEPTION USING ERRCODE='P1009',MESSAGE='ORDER_REDEMPTION_INVALID';
  END IF;
  IF o.state='COMPLETED' AND o.preparing_at IS NOT NULL AND o.ready_at IS NOT NULL AND o.completed_at IS NOT NULL
    AND o.ready_at>=o.preparing_at AND o.completed_at>=o.ready_at AND o.completed_at<=p_now
    AND o.redemption_token_hash IS NOT NULL AND octet_length(o.redemption_token_hash)=32
    AND coalesce(octet_length(o.redemption_token_ciphertext),0)=0 AND o.redemption_key_version IS NULL
    AND o.redemption_issued_at IS NOT NULL AND o.redemption_issued_at>=o.ready_at AND o.redemption_issued_at<=o.completed_at
    AND o.redeemed_by_account_id IS NOT NULL AND o.redeemed_by_account_id>0
    AND o.redeemed_at IS NOT NULL AND o.redeemed_at=o.completed_at THEN
    replaying:=true;
  ELSE
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
    'redeemed_by',o.redeemed_by_account_id::text,'redeemed_at',o.redeemed_at,
    'user_id',o.user_id::text,'order_number_hex',encode(o.order_no,'hex'),'pickup_time',to_char(o.pickup_date+o.pickup_time,'HH24:MI'),
    'pickup_point',o.pickup_point_snapshot);
    IF p_proof IS DISTINCT FROM proof THEN RAISE EXCEPTION USING ERRCODE='P1005',MESSAGE='ORDER_STALE_PROOF'; END IF;
    UPDATE public.orders SET state='COMPLETED',completed_at=p_now,redeemed_by_account_id=a.id,redeemed_at=p_now,
      redemption_token_ciphertext=NULL,redemption_key_version=NULL,record_version=record_version+1,updated_at=p_now
      WHERE id=p_order AND state='READY_FOR_PICKUP' AND record_version=o.record_version;
    GET DIAGNOSTICS affected=ROW_COUNT;
    IF affected<>1 THEN RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_WRITE_COUNT'; END IF;
  END IF;
  IF p_receipt_at IS NULL THEN RAISE EXCEPTION USING ERRCODE='P1005',MESSAGE='ORDER_STALE_PROOF'; END IF;
  IF NOT isfinite(p_receipt_at) THEN RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT'; END IF;
  response:=jsonb_build_object('order_id',p_order,'state','COMPLETED','changed',true);
  BEGIN
    INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,
      actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,target_type,target_id,
      target_key_hash,operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at)
    VALUES('COMMAND_RECEIPT','MERCHANT',scope,p_actor,a.id,a.id,a.role,a.auth_version,'fulfillment.redeem_token','ORDER',p_order,
      NULL,operation,sha256(convert_to('request:'||p_request_id,'UTF8')),'SUCCEEDED','OK',
      jsonb_build_object('request_digest',request_digest),NULL,response,p_receipt_at);
  EXCEPTION WHEN unique_violation THEN
    GET STACKED DIAGNOSTICS constraint_name=CONSTRAINT_NAME;
    IF constraint_name='uq_action_audits_command_receipt' THEN
      RAISE EXCEPTION USING ERRCODE='P1008',MESSAGE='ORDER_RECEIPT_RACE';
    END IF;
    RAISE;
  END;
  RETURN jsonb_build_object('order_id',p_order::text,'state','COMPLETED','changed',true,'replay',replaying);
END;
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_redeem_current_date_code_v1(p_actor numeric,p_order numeric,p_key text,p_request_id text,p_now timestamptz,p_request text,p_locator_live boolean,p_locator_date date,p_proof jsonb,p_receipt_at timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE
  a public.merchant_accounts%ROWTYPE; o public.orders%ROWTYPE; receipt public.action_audits%ROWTYPE;
  scope bytea; operation bytea; request_digest text; response jsonb; proof jsonb;
  token_json text; pickup_code integer; affected bigint; constraint_name text;
  replaying boolean:=false;
BEGIN
  IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
  END IF;
  IF p_actor IS NULL OR p_actor<>trunc(p_actor) OR p_actor<=0 OR p_actor>18446744073709551615
    OR p_key IS NULL OR p_key !~ '^[!-~]{1,128}$' THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
  p_actor:=trunc(p_actor);
  IF p_order IS NULL OR p_order<>trunc(p_order) OR p_order<=0 OR p_order>18446744073709551615
    OR p_request_id IS NULL OR p_request_id !~ '^[!-~]{1,64}$' OR (p_now IS NOT NULL AND NOT isfinite(p_now)) THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
  p_order:=trunc(p_order);
  IF p_request IS NULL OR p_request !~ '^\{"kind":"REDEEM_CURRENT_DATE_CODE","pickup_number":"[0-9]{4}"\}$'
    OR p_locator_date IS NULL OR NOT isfinite(p_locator_date)
    OR to_char(p_locator_date,'YYYY-MM-DD') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
  pickup_code:=substring(p_request FROM length('{"kind":"REDEEM_CURRENT_DATE_CODE","pickup_number":"')+1 FOR 4)::integer;
  IF pickup_code NOT BETWEEN 1 AND 9999 THEN RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT'; END IF;
  IF p_locator_live IS NULL THEN RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT'; END IF;
  request_digest:=encode(sha256(convert_to(p_request,'UTF8')),'hex');
  SELECT * INTO a FROM public.merchant_accounts WHERE bound_user_id=p_actor FOR SHARE;
  IF NOT FOUND OR NOT a.enabled THEN RAISE EXCEPTION USING ERRCODE='P1002',MESSAGE='ORDER_FORBIDDEN'; END IF;
  IF a.id<=0 OR a.role NOT IN ('OWNER','SUBACCOUNT') OR a.record_version<=0 OR a.auth_version<=0 THEN
    RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_UNAVAILABLE';
  END IF;
  scope:=sha256(convert_to('merchant:'||p_actor::text||':'||a.id::text,'UTF8'));
  operation:=sha256(convert_to('operation:'||p_key,'UTF8'));
  SELECT * INTO o FROM public.orders WHERE id=p_order FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P1003',MESSAGE='ORDER_NOT_FOUND'; END IF;
  IF o.id<=0 OR o.record_version<=0 OR o.pickup_number NOT BETWEEN 1 AND 9999
    OR to_char(o.pickup_date,'YYYY-MM-DD') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
    RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_UNAVAILABLE';
  END IF;
  SELECT * INTO receipt FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='MERCHANT'
    AND actor_scope_hash=scope AND action='fulfillment.redeem_current_date_code' AND operation_key_hash=operation FOR SHARE;
  IF FOUND THEN
    IF jsonb_typeof(receipt.before_state_json) IS DISTINCT FROM 'object' OR octet_length(receipt.before_state_json::text)>256
      OR (SELECT count(*) FROM jsonb_object_keys(receipt.before_state_json))<>1
      OR jsonb_typeof(receipt.before_state_json->'request_digest') IS DISTINCT FROM 'string'
      OR receipt.before_state_json->>'request_digest' IS NULL OR receipt.before_state_json->>'request_digest' !~ '^[0-9a-fA-F]{64}$' THEN
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
      OR (response->>'order_id')::numeric>18446744073709551615
      OR response->>'state' IS DISTINCT FROM 'COMPLETED' OR response->'changed' IS DISTINCT FROM 'true'::jsonb
      OR response->>'order_id' IS DISTINCT FROM p_order::text THEN
      RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_RECEIPT_CORRUPT';
    END IF;
    RETURN jsonb_build_object('order_id',p_order::text,'state','COMPLETED','changed',true,'replay',true);
  END IF;
  IF p_now IS NULL THEN RAISE EXCEPTION USING ERRCODE='P1005',MESSAGE='ORDER_STALE_PROOF'; END IF;
  IF NOT p_locator_live OR o.pickup_date<>p_locator_date OR o.pickup_number<>pickup_code THEN
    RAISE EXCEPTION USING ERRCODE='P1009',MESSAGE='ORDER_REDEMPTION_INVALID';
  END IF;
  IF o.state='COMPLETED' AND o.preparing_at IS NOT NULL AND o.ready_at IS NOT NULL AND o.completed_at IS NOT NULL
    AND o.ready_at>=o.preparing_at AND o.completed_at>=o.ready_at AND o.completed_at<=p_now
    AND o.redemption_token_hash IS NOT NULL AND octet_length(o.redemption_token_hash)=32
    AND coalesce(octet_length(o.redemption_token_ciphertext),0)=0 AND o.redemption_key_version IS NULL
    AND o.redemption_issued_at IS NOT NULL AND o.redemption_issued_at>=o.ready_at AND o.redemption_issued_at<=o.completed_at
    AND o.redeemed_by_account_id IS NOT NULL AND o.redeemed_by_account_id>0
    AND o.redeemed_at IS NOT NULL AND o.redeemed_at=o.completed_at THEN
    replaying:=true;
  ELSE
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
    'redeemed_by',o.redeemed_by_account_id::text,'redeemed_at',o.redeemed_at,
    'user_id',o.user_id::text,'order_number_hex',encode(o.order_no,'hex'),'pickup_time',to_char(o.pickup_date+o.pickup_time,'HH24:MI'),
    'pickup_point',o.pickup_point_snapshot);
    IF p_proof IS DISTINCT FROM proof THEN RAISE EXCEPTION USING ERRCODE='P1005',MESSAGE='ORDER_STALE_PROOF'; END IF;
    UPDATE public.orders SET state='COMPLETED',completed_at=p_now,redeemed_by_account_id=a.id,redeemed_at=p_now,
      redemption_token_ciphertext=NULL,redemption_key_version=NULL,record_version=record_version+1,updated_at=p_now
      WHERE id=p_order AND state='READY_FOR_PICKUP' AND record_version=o.record_version;
    GET DIAGNOSTICS affected=ROW_COUNT;
    IF affected<>1 THEN RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_WRITE_COUNT'; END IF;
  END IF;
  IF p_receipt_at IS NULL THEN RAISE EXCEPTION USING ERRCODE='P1005',MESSAGE='ORDER_STALE_PROOF'; END IF;
  IF NOT isfinite(p_receipt_at) THEN RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT'; END IF;
  response:=jsonb_build_object('order_id',p_order,'state','COMPLETED','changed',true);
  BEGIN
    INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,
      actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,target_type,target_id,
      target_key_hash,operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at)
    VALUES('COMMAND_RECEIPT','MERCHANT',scope,p_actor,a.id,a.id,a.role,a.auth_version,'fulfillment.redeem_current_date_code','ORDER',p_order,
      NULL,operation,sha256(convert_to('request:'||p_request_id,'UTF8')),'SUCCEEDED','OK',
      jsonb_build_object('request_digest',request_digest),NULL,response,p_receipt_at);
  EXCEPTION WHEN unique_violation THEN
    GET STACKED DIAGNOSTICS constraint_name=CONSTRAINT_NAME;
    IF constraint_name='uq_action_audits_command_receipt' THEN
      RAISE EXCEPTION USING ERRCODE='P1008',MESSAGE='ORDER_RECEIPT_RACE';
    END IF;
    RAISE;
  END;
  RETURN jsonb_build_object('order_id',p_order::text,'state','COMPLETED','changed',true,'replay',replaying);
END;
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_mark_ready_v1(p_actor numeric,p_order numeric,p_key text,p_request_id text,p_now timestamptz,p_enqueue_at timestamptz,p_request text,p_proof jsonb,p_token_material jsonb,p_receipt_at timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE
  a public.merchant_accounts%ROWTYPE; o public.orders%ROWTYPE; receipt public.action_audits%ROWTYPE;
  scope bytea; operation bytea; request_digest text; response jsonb; proof jsonb;
  token_json text; pickup_code integer; affected bigint; constraint_name text;
  consent public.notification_consents%ROWTYPE; outbox public.notification_outbox%ROWTYPE; message jsonb; whitespace text;
BEGIN
  IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
  END IF;
  IF p_actor IS NULL OR p_actor<>trunc(p_actor) OR p_actor<=0 OR p_actor>18446744073709551615
    OR p_key IS NULL OR p_key !~ '^[!-~]{1,128}$' THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
  p_actor:=trunc(p_actor);
  IF p_order IS NULL OR p_order<>trunc(p_order) OR p_order<=0 OR p_order>18446744073709551615
    OR p_request_id IS NULL OR p_request_id !~ '^[!-~]{1,64}$' OR (p_now IS NOT NULL AND NOT isfinite(p_now)) THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
  p_order:=trunc(p_order);
  IF p_request IS DISTINCT FROM '{"kind":"MARK_READY","order_id":'||p_order::text||'}' THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
  request_digest:=encode(sha256(convert_to(p_request,'UTF8')),'hex');
  SELECT * INTO a FROM public.merchant_accounts WHERE bound_user_id=p_actor FOR SHARE;
  IF NOT FOUND OR NOT a.enabled THEN RAISE EXCEPTION USING ERRCODE='P1002',MESSAGE='ORDER_FORBIDDEN'; END IF;
  IF a.id<=0 OR a.role NOT IN ('OWNER','SUBACCOUNT') OR a.record_version<=0 OR a.auth_version<=0 THEN
    RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_UNAVAILABLE';
  END IF;
  scope:=sha256(convert_to('merchant:'||p_actor::text||':'||a.id::text,'UTF8'));
  operation:=sha256(convert_to('operation:'||p_key,'UTF8'));
  SELECT * INTO o FROM public.orders WHERE id=p_order FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P1003',MESSAGE='ORDER_NOT_FOUND'; END IF;
  IF o.id<=0 OR o.record_version<=0 OR o.pickup_number NOT BETWEEN 1 AND 9999
    OR to_char(o.pickup_date,'YYYY-MM-DD') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
    RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_UNAVAILABLE';
  END IF;
  SELECT * INTO receipt FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='MERCHANT'
    AND actor_scope_hash=scope AND action='fulfillment.mark_ready' AND operation_key_hash=operation FOR SHARE;
  IF FOUND THEN
    IF jsonb_typeof(receipt.before_state_json) IS DISTINCT FROM 'object' OR octet_length(receipt.before_state_json::text)>256
      OR (SELECT count(*) FROM jsonb_object_keys(receipt.before_state_json))<>1
      OR jsonb_typeof(receipt.before_state_json->'request_digest') IS DISTINCT FROM 'string'
      OR receipt.before_state_json->>'request_digest' IS NULL OR receipt.before_state_json->>'request_digest' !~ '^[0-9a-fA-F]{64}$' THEN
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
      OR (response->>'order_id')::numeric>18446744073709551615
      OR response->>'state' IS DISTINCT FROM 'READY_FOR_PICKUP' OR response->'changed' IS DISTINCT FROM 'true'::jsonb
      OR response->>'order_id' IS DISTINCT FROM p_order::text THEN
      RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_RECEIPT_CORRUPT';
    END IF;
    RETURN jsonb_build_object('order_id',p_order::text,'state','READY_FOR_PICKUP','changed',true,'replay',true);
  END IF;
  IF p_now IS NULL THEN RAISE EXCEPTION USING ERRCODE='P1005',MESSAGE='ORDER_STALE_PROOF'; END IF;
  IF NOT (o.state='PREPARING' OR (o.state='RESERVED' AND o.pickup_at<=p_now+interval '30 minutes')) THEN RAISE EXCEPTION USING ERRCODE='P1004',MESSAGE='ORDER_TRANSITION'; END IF;
  IF (o.state='PREPARING' AND (o.preparing_at IS NULL OR o.preparing_at>p_now))
    OR (o.state='RESERVED' AND (o.preparing_at IS NOT NULL OR o.materialized_at>p_now)) OR o.ready_at IS NOT NULL OR o.completed_at IS NOT NULL
    OR coalesce(octet_length(o.redemption_token_hash),0)<>0 OR coalesce(octet_length(o.redemption_token_ciphertext),0)<>0
    OR o.redemption_key_version IS NOT NULL OR o.redemption_issued_at IS NOT NULL OR o.redeemed_by_account_id IS NOT NULL OR o.redeemed_at IS NOT NULL THEN
    RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_UNAVAILABLE';
  END IF;
  proof:=jsonb_build_object('id',o.id::text,'state',o.state,'version',o.record_version::text,
    'pickup_date',to_char(o.pickup_date,'YYYY-MM-DD'),'pickup_number',o.pickup_number,
    'preparing_at',o.preparing_at,'ready_at',o.ready_at,'completed_at',o.completed_at,
    'hash',encode(o.redemption_token_hash,'hex'),'ciphertext',encode(o.redemption_token_ciphertext,'hex'),
    'key_version',o.redemption_key_version,'issued_at',o.redemption_issued_at,
    'redeemed_by',o.redeemed_by_account_id::text,'redeemed_at',o.redeemed_at,
    'user_id',o.user_id::text,'order_number_hex',encode(o.order_no,'hex'),'pickup_time',to_char(o.pickup_date+o.pickup_time,'HH24:MI'),
    'pickup_point',o.pickup_point_snapshot);
  IF p_proof IS DISTINCT FROM proof OR p_token_material IS NULL THEN RAISE EXCEPTION USING ERRCODE='P1005',MESSAGE='ORDER_STALE_PROOF'; END IF;
  IF jsonb_typeof(p_token_material) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_token_material))<>3
    OR jsonb_typeof(p_token_material->'hash') IS DISTINCT FROM 'string' OR p_token_material->>'hash' IS NULL OR p_token_material->>'hash' !~ '^[0-9a-f]{64}$'
    OR jsonb_typeof(p_token_material->'ciphertext') IS DISTINCT FROM 'string' OR p_token_material->>'ciphertext' IS NULL
    OR p_token_material->>'ciphertext' !~ '^([0-9a-f]{2}){1,192}$'
    OR jsonb_typeof(p_token_material->'key_version') IS DISTINCT FROM 'number' OR p_token_material->>'key_version' IS NULL
    OR p_token_material->>'key_version' !~ '^[1-9][0-9]{0,4}$' OR (p_token_material->>'key_version')::integer>65535 THEN
    RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_TOKEN_MATERIAL';
  END IF;
  UPDATE public.orders SET state='READY_FOR_PICKUP',
    preparing_at=COALESCE(o.preparing_at,GREATEST(o.materialized_at,o.pickup_at-interval '30 minutes')),
    ready_at=p_now,redemption_issued_at=p_now,
    redemption_token_hash=decode(p_token_material->>'hash','hex'),redemption_token_ciphertext=decode(p_token_material->>'ciphertext','hex'),
    redemption_key_version=(p_token_material->>'key_version')::integer,record_version=record_version+1,updated_at=p_now
    WHERE id=p_order AND state=o.state AND record_version=o.record_version;
  GET DIAGNOSTICS affected=ROW_COUNT;
  IF affected<>1 THEN RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_WRITE_COUNT'; END IF;
  whitespace:=E' \t\n\r\v\f'||chr(133)||chr(160)||chr(5760)||chr(8192)||chr(8193)||chr(8194)||chr(8195)||chr(8196)||chr(8197)||chr(8198)||chr(8199)||chr(8200)||chr(8201)||chr(8202)||chr(8232)||chr(8233)||chr(8239)||chr(8287)||chr(12288);
  IF o.user_id<=0 OR octet_length(o.order_no) NOT BETWEEN 1 AND 64
    OR octet_length(o.pickup_point_snapshot) NOT BETWEEN 1 AND 256 OR o.pickup_point_snapshot<>btrim(o.pickup_point_snapshot,whitespace) THEN
    RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_NOTIFICATION_INVALID';
  END IF;
  SELECT * INTO consent FROM public.notification_consents WHERE order_id=p_order AND kind='READY' AND decision='ACCEPTED'
    ORDER BY grant_sequence LIMIT 1 FOR UPDATE;
  IF FOUND THEN
    IF consent.user_id<>o.user_id THEN RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_NOTIFICATION_UNAVAILABLE'; END IF;
    message:=jsonb_build_object('order_number',convert_from(o.order_no,'UTF8'),'pickup_date',to_char(o.pickup_date,'YYYY-MM-DD'),
      'pickup_time',to_char(o.pickup_date+o.pickup_time,'HH24:MI'),'pickup_point',o.pickup_point_snapshot);
    SELECT * INTO outbox FROM public.notification_outbox WHERE order_id=p_order AND kind='READY' FOR UPDATE;
    IF FOUND THEN
      IF outbox.consent_id<>consent.id OR outbox.recipient_user_id<>o.user_id
        OR outbox.template_config_version<>consent.template_config_version OR outbox.immutable_message_json IS DISTINCT FROM message THEN
        RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_NOTIFICATION_UNAVAILABLE';
      END IF;
    ELSE
      IF consent.consumed_at IS NOT NULL THEN RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_NOTIFICATION_UNAVAILABLE'; END IF;
      IF p_enqueue_at IS NULL OR NOT isfinite(p_enqueue_at) THEN RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_NOTIFICATION_UNAVAILABLE'; END IF;
      INSERT INTO public.notification_outbox(order_id,consent_id,kind,recipient_user_id,immutable_message_json,template_config_version,
        state,attempt_count,next_attempt_at,lease_owner,lease_expires_at,record_version,provider_message_id,last_error_code,created_at,sent_at)
      VALUES(p_order,consent.id,'READY',o.user_id,message,consent.template_config_version,'PENDING',0,p_now,NULL,NULL,1,NULL,NULL,p_enqueue_at,NULL);
      UPDATE public.notification_consents SET consumed_at=p_enqueue_at WHERE id=consent.id AND consumed_at IS NULL;
      GET DIAGNOSTICS affected=ROW_COUNT;
      IF affected<>1 THEN RAISE EXCEPTION USING ERRCODE='P1007',MESSAGE='ORDER_NOTIFICATION_UNAVAILABLE'; END IF;
    END IF;
  END IF;
  IF p_receipt_at IS NULL THEN RAISE EXCEPTION USING ERRCODE='P1005',MESSAGE='ORDER_STALE_PROOF'; END IF;
  IF NOT isfinite(p_receipt_at) THEN RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT'; END IF;
  response:=jsonb_build_object('order_id',p_order,'state','READY_FOR_PICKUP','changed',true);
  BEGIN
    INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,
      actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,target_type,target_id,
      target_key_hash,operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at)
    VALUES('COMMAND_RECEIPT','MERCHANT',scope,p_actor,a.id,a.id,a.role,a.auth_version,'fulfillment.mark_ready','ORDER',p_order,
      NULL,operation,sha256(convert_to('request:'||p_request_id,'UTF8')),'SUCCEEDED','OK',
      jsonb_build_object('request_digest',request_digest),NULL,response,p_receipt_at);
  EXCEPTION WHEN unique_violation THEN
    GET STACKED DIAGNOSTICS constraint_name=CONSTRAINT_NAME;
    IF constraint_name='uq_action_audits_command_receipt' THEN
      RAISE EXCEPTION USING ERRCODE='P1008',MESSAGE='ORDER_RECEIPT_RACE';
    END IF;
    RAISE;
  END;
  RETURN jsonb_build_object('order_id',p_order::text,'state','READY_FOR_PICKUP','changed',true,'replay',false);
END;
$order_rpc$;
