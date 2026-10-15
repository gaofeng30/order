-- Fixed store-status command; original current authorization and singleton locks.
CREATE OR REPLACE FUNCTION public.order_store_status_apply_v1(p_actor numeric,p_desired text,p_operation_hash text,p_request_hash text,p_request_digest text,p_now jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE account record;receipt record;current_status text;scope bytea;changed boolean;reason text;affected bigint;micros numeric;at timestamptz;failure_state text;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
  IF p_actor IS NULL OR p_actor NOT BETWEEN 1 AND 18446744073709551615 OR trunc(p_actor)<>p_actor OR p_desired IS NULL OR p_desired NOT IN ('open','closed','cutoff') THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  BEGIN
   SELECT id,role,enabled,record_version,auth_version INTO account FROM public.merchant_accounts WHERE bound_user_id=p_actor FOR SHARE;
   IF NOT FOUND THEN RETURN jsonb_build_object('outcome','ACCOUNT_NOT_AVAILABLE');END IF;
   IF account.id IS NULL OR account.role IS NULL OR account.enabled IS NULL OR account.record_version IS NULL OR account.auth_version IS NULL OR account.id NOT BETWEEN 0 AND 18446744073709551615 OR account.record_version NOT BETWEEN 0 AND 18446744073709551615 OR account.auth_version NOT BETWEEN 0 AND 18446744073709551615 OR trunc(account.id)<>account.id OR trunc(account.record_version)<>account.record_version OR trunc(account.auth_version)<>account.auth_version THEN RETURN jsonb_build_object('outcome','AUTHORIZATION_UNAVAILABLE');END IF;
   IF NOT account.enabled THEN RETURN jsonb_build_object('outcome','ACCOUNT_NOT_AVAILABLE');END IF;
   IF account.id=0 OR account.record_version=0 OR account.auth_version=0 OR account.role NOT IN ('OWNER','SUBACCOUNT') THEN RETURN jsonb_build_object('outcome','AUTHORIZATION_UNAVAILABLE');END IF;
  EXCEPTION WHEN query_canceled THEN RETURN jsonb_build_object('outcome','AUTHORIZATION_UNAVAILABLE');
  WHEN OTHERS THEN RETURN jsonb_build_object('outcome','AUTHORIZATION_UNAVAILABLE');END;
  SELECT business_status INTO current_status FROM public.storefront_settings WHERE id=1 FOR UPDATE;
  IF NOT FOUND OR current_status IS NULL OR current_status NOT IN ('open','closed','cutoff') THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  scope:=sha256(convert_to('merchant:'||p_actor::text||':'||account.id::text,'UTF8'));
  IF p_operation_hash IS NULL OR p_operation_hash !~ '^[0-9a-f]{64}$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  SELECT actor_user_id,actor_account_id,actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,target_type,target_id,result,reason_code,before_state_json,after_state_json,response_json INTO receipt FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='MERCHANT' AND actor_scope_hash=scope AND action='merchant.store.operating_status.write' AND operation_key_hash=decode(p_operation_hash,'hex') LIMIT 1;
  IF FOUND THEN
   -- Replay is read-only; the unchanged Go strict decoder consumes its bytes.
   IF receipt.actor_account_id IS DISTINCT FROM account.id THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
   RETURN jsonb_build_object('outcome','REPLAY','Actor',receipt.actor_user_id::text,'Account',receipt.actor_account_id::text,'Snapshot',receipt.actor_account_id_snapshot::text,'Auth',receipt.actor_auth_version_snapshot::text,'Target',receipt.target_id::text,'Role',receipt.actor_role_snapshot,'TargetType',receipt.target_type,'ResultValue',receipt.result,'Reason',receipt.reason_code,'BeforeHex',encode(convert_to(receipt.before_state_json::text,'UTF8'),'hex'),'AfterHex',encode(convert_to(receipt.after_state_json::text,'UTF8'),'hex'),'ResponseHex',encode(convert_to(receipt.response_json::text,'UTF8'),'hex'));
  END IF;
  changed:=current_status<>p_desired;reason:='OPERATING_STATUS_UNCHANGED';IF changed THEN reason:='OPERATING_STATUS_CHANGED';
   UPDATE public.storefront_settings SET business_status=p_desired WHERE id=1;GET DIAGNOSTICS affected=ROW_COUNT;
   IF affected<>1 THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_STORE_STATUS_UNAVAILABLE';END IF;
  END IF;
  -- Consume the original clock bind only at the audit INSERT.
 IF jsonb_typeof(p_now) IS DISTINCT FROM 'object' OR NOT (p_now ?& ARRAY['epoch_seconds','nanosecond','zero','canonical_utc']) OR (p_now-ARRAY['epoch_seconds','nanosecond','zero','canonical_utc'])<>'{}'::jsonb OR jsonb_typeof(p_now->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_now->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_now->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_now->'nanosecond') IS DISTINCT FROM 'number' OR (p_now->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_now->>'nanosecond')::numeric)<>(p_now->>'nanosecond')::numeric OR jsonb_typeof(p_now->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_now->'canonical_utc') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_STORE_STATUS_UNAVAILABLE';END IF;
  micros:=(p_now->>'epoch_seconds')::numeric*1000000+div((p_now->>'nanosecond')::numeric,1000)-946684800000000;
  micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
  IF micros=9223372036854775807 THEN at:='infinity'::timestamptz;ELSIF micros=-9223372036854775808 THEN at:='-infinity'::timestamptz;
  ELSE at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
  IF p_request_hash IS NULL OR p_request_hash !~ '^[0-9a-f]{64}$' OR p_request_digest IS NULL OR p_request_digest !~ '^[0-9a-f]{64}$' THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_STORE_STATUS_UNAVAILABLE';END IF;
  INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,target_type,target_id,target_key_hash,operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at)
  VALUES('COMMAND_RECEIPT','MERCHANT',scope,p_actor,account.id,account.id,account.role,account.auth_version,'merchant.store.operating_status.write','storefront_settings',1,NULL,decode(p_operation_hash,'hex'),decode(p_request_hash,'hex'),'SUCCEEDED',reason,jsonb_build_object('request_digest',p_request_digest),jsonb_build_object('business_status',p_desired),jsonb_build_object('before',current_status,'after',p_desired,'changed',changed),at);
  RETURN jsonb_build_object('outcome','APPLIED','result',jsonb_build_object('Before',current_status,'After',p_desired,'Changed',changed));
 EXCEPTION WHEN OTHERS THEN GET STACKED DIAGNOSTICS failure_state=RETURNED_SQLSTATE;
  IF failure_state IN ('40P01','40001','55P03') THEN RETURN jsonb_build_object('outcome','SQL_FAILURE','sqlstate',failure_state);END IF;
  RETURN jsonb_build_object('outcome','UNAVAILABLE');END;
END
$order_rpc$;
