-- Order identity storage; no provider codes, raw session tokens or credentials.
CREATE OR REPLACE FUNCTION public.order_identity_session_create_v1(p_openid_hex text,p_token_hash text,p_issued_at timestamptz,p_expires_at timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE user_id numeric;
BEGIN
  IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
  END IF;
  IF p_openid_hex IS NULL OR p_openid_hex !~ '^([0-9a-f]{2})*$' OR p_token_hash IS NULL OR p_token_hash !~ '^[0-9a-f]{64}$' THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_IDENTITY_INPUT';
  END IF;
  INSERT INTO public.miniprogram_users(openid,created_at,last_login_at)
    VALUES(decode(p_openid_hex,'hex'),p_issued_at,p_issued_at)
    ON CONFLICT(openid) DO UPDATE SET last_login_at=EXCLUDED.last_login_at RETURNING id INTO user_id;
  IF user_id IS NULL OR user_id=0 THEN
    RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';
  END IF;
  INSERT INTO public.miniprogram_sessions(token_hash,user_id,issued_at,expires_at)
    VALUES(decode(p_token_hash,'hex'),user_id,p_issued_at,p_expires_at);
  RETURN jsonb_build_object('outcome','SUCCEEDED');
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_identity_extra_phone_set_v1(p_user numeric,p_phone_hex text,p_name text,p_name_key_hex text,p_operation_hash text,p_request_hash text,p_request_digest text)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE
  u public.miniprogram_users%ROWTYPE; d public.discount_settings%ROWTYPE; candidate record;
  command_phone bytea; command_name_key bytea; primary_seen boolean:=false; extra_seen boolean:=false;
  primary_staff boolean:=false; extra_staff boolean:=false; pricing_kind text:='VISITOR';rate integer:=100;
  scope bytea; affected bigint; constraint_name text; receipt public.action_audits%ROWTYPE;
  current_extra bytea; current_name text;
BEGIN
  IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
  END IF;
  command_phone:=decode(p_phone_hex,'hex');command_name_key:=decode(p_name_key_hex,'hex');
  BEGIN
  SELECT * INTO u FROM public.miniprogram_users WHERE id=p_user FOR UPDATE;
  IF NOT FOUND OR u.record_version<=0 OR (u.primary_phone IS NULL)<>(u.primary_phone_bound_at IS NULL)
    OR (u.primary_phone IS NOT NULL AND convert_from(u.primary_phone,'UTF8') !~ '^[+][1-9][0-9]{0,14}$') THEN
    RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';
  END IF;
  IF u.primary_phone IS NULL THEN RETURN jsonb_build_object('outcome','PRIMARY_PHONE_REQUIRED');END IF;
  SELECT * INTO d FROM public.discount_settings WHERE id=1 FOR UPDATE;
  IF NOT FOUND OR d.rate_percent<1 OR d.rate_percent>100 OR d.whitelist_version<=0 THEN
    RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';
  END IF;
  FOR candidate IN SELECT phone,name_key,enabled FROM public.staff_whitelist
    WHERE phone=u.primary_phone OR phone=command_phone ORDER BY id FOR UPDATE
  LOOP
    IF convert_from(candidate.phone,'UTF8') !~ '^[+][1-9][0-9]{0,14}$' OR octet_length(candidate.name_key) NOT BETWEEN 1 AND 400 THEN
      RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';
    END IF;
    IF candidate.phone=u.primary_phone THEN
      IF primary_seen THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';END IF;
      primary_seen:=true;primary_staff:=candidate.enabled;
    END IF;
    IF candidate.phone=command_phone THEN
      IF extra_seen THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';END IF;
      extra_seen:=true;extra_staff:=candidate.enabled AND candidate.name_key=command_name_key;
    END IF;
  END LOOP;
  IF primary_staff OR extra_staff THEN pricing_kind:='STAFF';rate:=d.rate_percent;END IF;
  UPDATE public.miniprogram_users SET extra_phone=command_phone,extra_name=p_name,extra_name_key=command_name_key,
    extra_phone_set_at=clock_timestamp(),record_version=record_version+1 WHERE id=p_user AND record_version=u.record_version;
  GET DIAGNOSTICS affected=ROW_COUNT;
  IF affected<>1 THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';END IF;
  scope:=sha256(convert_to('USER','UTF8')||decode('00','hex')||
    decode(lpad(to_hex(div(p_user,4294967296)::bigint),8,'0')||lpad(to_hex(mod(p_user,4294967296)::bigint),8,'0'),'hex'));
  INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,action,target_type,target_id,
    operation_key_hash,request_id_hash,result,reason_code,before_state_json,response_json,occurred_at)
    VALUES('COMMAND_RECEIPT','USER',scope,p_user,'user.extra_phone.set','miniprogram_user',p_user,
      decode(p_operation_hash,'hex'),decode(p_request_hash,'hex'),'SUCCEEDED','EXTRA_PHONE_SET',
      jsonb_build_object('request_digest',p_request_digest),jsonb_build_object('pricing_identity',jsonb_build_object('kind',pricing_kind,'rate_percent',rate)),clock_timestamp());
  RETURN jsonb_build_object('outcome','SUCCEEDED','kind',pricing_kind,'rate_percent',rate);
  EXCEPTION WHEN unique_violation THEN
    GET STACKED DIAGNOSTICS constraint_name=CONSTRAINT_NAME;
    IF constraint_name<>'uq_action_audits_command_receipt' THEN RAISE;END IF;
  END;
  -- The controlled subtransaction rolls back the attempted update/receipt
  -- before the original strict Go decoder examines committed evidence.
  SELECT * INTO receipt FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='USER'
    AND actor_scope_hash=scope AND action='user.extra_phone.set' AND operation_key_hash=decode(p_operation_hash,'hex') LIMIT 1;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';END IF;
  SELECT extra_phone,extra_name INTO current_extra,current_name FROM public.miniprogram_users WHERE id=p_user;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';END IF;
  RETURN jsonb_build_object('outcome','REPLAY','evidence_text',receipt.before_state_json::text,'response_text',receipt.response_json::text,
    'current_phone_hex',encode(current_extra,'hex'),'current_name',current_name);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_merchant_login_recover_v1(p_user numeric,p_code_hash text,p_request_hash text,p_started_at timestamptz,p_at timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE
  u public.miniprogram_users%ROWTYPE; a public.merchant_accounts%ROWTYPE; has_account boolean; confirmed boolean:=false;
  result text; reason text; state text; scope bytea;
BEGIN
  IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
  END IF;
  SELECT * INTO u FROM public.miniprogram_users WHERE id=p_user FOR SHARE;
  IF NOT FOUND OR (u.primary_phone IS NULL)<>(u.primary_phone_bound_at IS NULL)
    OR (u.primary_phone IS NOT NULL AND convert_from(u.primary_phone,'UTF8') !~ '^[+][1-9][0-9]{0,14}$') THEN
    RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';
  END IF;
  SELECT * INTO a FROM public.merchant_accounts WHERE bound_user_id=p_user FOR SHARE;
  has_account:=FOUND;
  IF has_account AND (a.id<=0 OR a.role NOT IN ('OWNER','SUBACCOUNT') OR a.record_version<=0 OR a.auth_version<=0) THEN
    RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';
  END IF;
  IF has_account AND a.enabled AND u.primary_phone IS NOT NULL THEN
    SELECT EXISTS(SELECT 1 FROM public.action_audits evidence WHERE evidence.actor_user_id=p_user
      AND evidence.actor_account_id_snapshot=a.id AND evidence.actor_auth_version_snapshot=a.auth_version
      AND evidence.target_type='merchant_login_code' AND evidence.target_key_hash=decode(p_code_hash,'hex')
      AND evidence.action='merchant.login' AND evidence.result='SUCCEEDED'
      AND evidence.reason_code IN ('FIRST_BINDING','CONCURRENT_BINDING_CONFIRMED') AND evidence.occurred_at>=p_started_at) INTO confirmed;
  END IF;
  IF confirmed THEN result:='SUCCEEDED';reason:='CONCURRENT_BINDING_CONFIRMED';state:='BOUND_ENABLED';
  ELSE
    result:='REJECTED';reason:='PHONE_CODE_REJECTED';
    state:=CASE WHEN NOT has_account THEN 'UNRESOLVED' WHEN a.enabled THEN 'BOUND_UNCONFIRMED' ELSE 'BOUND_DISABLED' END;
  END IF;
  scope:=sha256(convert_to('MERCHANT_LOGIN','UTF8')||decode('00','hex')||
    decode(lpad(to_hex(div(p_user,4294967296)::bigint),8,'0')||lpad(to_hex(mod(p_user,4294967296)::bigint),8,'0'),'hex'));
  INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,
    actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,result,reason_code,
    target_type,target_key_hash,request_id_hash,before_state_json,after_state_json,occurred_at)
    VALUES('LEGACY_EVIDENCE','MERCHANT',scope,p_user,CASE WHEN confirmed THEN a.id END,CASE WHEN confirmed THEN a.id END,
      CASE WHEN confirmed THEN a.role END,CASE WHEN confirmed THEN a.auth_version END,'merchant.login',result,reason,
      'merchant_login_code',decode(p_code_hash,'hex'),decode(p_request_hash,'hex'),jsonb_build_object('state',state),jsonb_build_object('state',state),p_at);
  IF NOT confirmed THEN RETURN jsonb_build_object('outcome','PHONE_CODE_REJECTED');END IF;
  RETURN jsonb_build_object('outcome','SUCCEEDED','account_id',a.id::text,'role',a.role,'enabled',a.enabled,
    'record_version',a.record_version::text,'auth_version',a.auth_version::text);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_merchant_login_complete_v1(p_user numeric,p_phone_hex text,p_code_hash text,p_request_hash text,p_at timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE
  u public.miniprogram_users%ROWTYPE; a public.merchant_accounts%ROWTYPE; has_account boolean:=false;
  command_phone bytea; outcome text; result text; reason text; before_state text; after_state text;
  scope bytea; affected bigint; constraint_name text;
BEGIN
  IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
  END IF;
  command_phone:=decode(p_phone_hex,'hex');
  SELECT * INTO u FROM public.miniprogram_users WHERE id=p_user FOR UPDATE;
  IF NOT FOUND OR (u.primary_phone IS NULL)<>(u.primary_phone_bound_at IS NULL)
    OR (u.primary_phone IS NOT NULL AND convert_from(u.primary_phone,'UTF8') !~ '^[+][1-9][0-9]{0,14}$') THEN
    RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';
  END IF;
  <<resolve>>
  BEGIN
    SELECT * INTO a FROM public.merchant_accounts WHERE bound_user_id=p_user FOR UPDATE;
    IF FOUND THEN
      has_account:=true;
      IF a.id<=0 OR a.role NOT IN ('OWNER','SUBACCOUNT') OR a.record_version<=0 OR a.auth_version<=0 OR u.primary_phone IS NULL THEN
        RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';
      END IF;
      before_state:=CASE WHEN a.enabled THEN 'BOUND_ENABLED' ELSE 'BOUND_DISABLED' END;
      after_state:=before_state;
      IF u.primary_phone<>command_phone THEN outcome:='PRIMARY_PHONE_MISMATCH';reason:='PRIMARY_PHONE_MISMATCH';result:='REJECTED';
      ELSIF NOT a.enabled THEN outcome:='ACCOUNT_NOT_AVAILABLE';reason:='ACCOUNT_NOT_AVAILABLE';result:='REJECTED';
      ELSE outcome:='SUCCEEDED';reason:='CONCURRENT_BINDING_CONFIRMED';result:='SUCCEEDED';END IF;
      EXIT resolve;
    END IF;
    SELECT * INTO a FROM public.merchant_accounts WHERE merchant_accounts.phone=command_phone FOR UPDATE;
    IF NOT FOUND THEN
      before_state:='UNRESOLVED';after_state:=before_state;result:='REJECTED';
      IF u.primary_phone IS NOT NULL AND u.primary_phone<>command_phone THEN outcome:='PRIMARY_PHONE_MISMATCH';reason:='PRIMARY_PHONE_MISMATCH';
      ELSE outcome:='ACCOUNT_NOT_AVAILABLE';reason:='ACCOUNT_NOT_AVAILABLE';END IF;
      EXIT resolve;
    END IF;
    has_account:=true;
    IF a.id<=0 OR a.role NOT IN ('OWNER','SUBACCOUNT') OR a.record_version<=0 OR a.auth_version<=0 THEN
      RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';
    END IF;
    IF u.primary_phone IS NOT NULL AND u.primary_phone<>command_phone THEN
      before_state:=CASE WHEN a.bound_user_id IS NOT NULL THEN 'BOUND_OTHER' WHEN NOT a.enabled THEN 'UNBOUND_DISABLED' ELSE 'UNBOUND' END;
      after_state:=before_state;outcome:='PRIMARY_PHONE_MISMATCH';reason:='PRIMARY_PHONE_MISMATCH';result:='REJECTED';EXIT resolve;
    END IF;
    IF NOT a.enabled THEN
      before_state:='UNBOUND_DISABLED';after_state:=before_state;outcome:='ACCOUNT_NOT_AVAILABLE';reason:='ACCOUNT_NOT_AVAILABLE';result:='REJECTED';EXIT resolve;
    END IF;
    IF a.bound_user_id IS NOT NULL THEN
      before_state:='BOUND_OTHER';after_state:=before_state;outcome:='ACCOUNT_NOT_AVAILABLE';reason:='ACCOUNT_NOT_AVAILABLE';result:='REJECTED';EXIT resolve;
    END IF;
    IF u.primary_phone IS NULL THEN
      BEGIN
        UPDATE public.miniprogram_users SET primary_phone=command_phone,primary_phone_bound_at=p_at
          WHERE id=p_user AND primary_phone IS NULL AND primary_phone_bound_at IS NULL;
      EXCEPTION WHEN unique_violation THEN
        GET STACKED DIAGNOSTICS constraint_name=CONSTRAINT_NAME;
        IF constraint_name<>'uq_miniprogram_users_primary_phone' THEN RAISE; END IF;
        outcome:='PHONE_IN_USE';reason:='PHONE_IN_USE';result:='REJECTED';before_state:='UNBOUND';after_state:=before_state;
      END;
      IF outcome='PHONE_IN_USE' THEN EXIT resolve; END IF;
    END IF;
    UPDATE public.merchant_accounts SET bound_user_id=p_user,bound_at=p_at,record_version=record_version+1,
      auth_version=auth_version+1,updated_at=p_at
      WHERE id=a.id AND bound_user_id IS NULL AND record_version=a.record_version AND auth_version=a.auth_version;
    GET DIAGNOSTICS affected=ROW_COUNT;
    IF affected<>1 THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';END IF;
    a.record_version:=a.record_version+1;a.auth_version:=a.auth_version+1;
    outcome:='SUCCEEDED';result:='SUCCEEDED';reason:='FIRST_BINDING';before_state:='UNBOUND';after_state:='BOUND_ENABLED';
  END resolve;
  scope:=sha256(convert_to('MERCHANT_LOGIN','UTF8')||decode('00','hex')||
    decode(lpad(to_hex(div(p_user,4294967296)::bigint),8,'0')||lpad(to_hex(mod(p_user,4294967296)::bigint),8,'0'),'hex'));
  INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,
    actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,result,reason_code,
    target_type,target_key_hash,request_id_hash,before_state_json,after_state_json,occurred_at)
    VALUES('LEGACY_EVIDENCE','MERCHANT',scope,p_user,CASE WHEN has_account THEN a.id END,CASE WHEN has_account THEN a.id END,
      CASE WHEN has_account THEN a.role END,CASE WHEN has_account THEN a.auth_version END,'merchant.login',result,reason,
      'merchant_login_code',decode(p_code_hash,'hex'),decode(p_request_hash,'hex'),jsonb_build_object('state',before_state),jsonb_build_object('state',after_state),p_at);
  IF outcome<>'SUCCEEDED' THEN RETURN jsonb_build_object('outcome',outcome);END IF;
  RETURN jsonb_build_object('outcome','SUCCEEDED','account_id',a.id::text,'role',a.role,'enabled',a.enabled,
    'record_version',a.record_version::text,'auth_version',a.auth_version::text);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_merchant_login_start_v1(p_user numeric,p_code_hash text,p_request_hash text,p_at timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE
  u public.miniprogram_users%ROWTYPE; a public.merchant_accounts%ROWTYPE;
  result text; reason text; state text; scope bytea;
BEGIN
  IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
  END IF;
  SELECT * INTO u FROM public.miniprogram_users WHERE id=p_user FOR SHARE;
  IF NOT FOUND OR octet_length(u.openid)=0 OR (u.primary_phone IS NULL)<>(u.primary_phone_bound_at IS NULL)
    OR (u.primary_phone IS NOT NULL AND convert_from(u.primary_phone,'UTF8') !~ '^[+][1-9][0-9]{0,14}$') THEN
    RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';
  END IF;
  SELECT * INTO a FROM public.merchant_accounts WHERE bound_user_id=p_user FOR SHARE;
  IF NOT FOUND THEN RETURN jsonb_build_object('outcome','NEEDS_PROVIDER','openid_hex',encode(u.openid,'hex')); END IF;
  IF a.id<=0 OR a.role NOT IN ('OWNER','SUBACCOUNT') OR a.record_version<=0 OR a.auth_version<=0 THEN
    RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';
  END IF;
  IF NOT a.enabled THEN result:='REJECTED';reason:='ACCOUNT_NOT_AVAILABLE';state:='BOUND_DISABLED';
  ELSE
    IF u.primary_phone IS NULL THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE'; END IF;
    result:='SUCCEEDED';reason:='ALREADY_BOUND';state:='BOUND_ENABLED';
  END IF;
  scope:=sha256(convert_to('MERCHANT_LOGIN','UTF8')||decode('00','hex')||
    decode(lpad(to_hex(div(p_user,4294967296)::bigint),8,'0')||lpad(to_hex(mod(p_user,4294967296)::bigint),8,'0'),'hex'));
  INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,
    actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,result,reason_code,
    target_type,target_key_hash,request_id_hash,before_state_json,after_state_json,occurred_at)
    VALUES('LEGACY_EVIDENCE','MERCHANT',scope,p_user,a.id,a.id,a.role,a.auth_version,'merchant.login',result,reason,
      'merchant_login_code',decode(p_code_hash,'hex'),decode(p_request_hash,'hex'),jsonb_build_object('state',state),jsonb_build_object('state',state),p_at);
  IF NOT a.enabled THEN RETURN jsonb_build_object('outcome','ACCOUNT_NOT_AVAILABLE'); END IF;
  RETURN jsonb_build_object('outcome','SUCCEEDED','account_id',a.id::text,'role',a.role,'enabled',a.enabled,
    'record_version',a.record_version::text,'auth_version',a.auth_version::text);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_identity_phone_read_v1(p_user numeric)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE u public.miniprogram_users%ROWTYPE;
BEGIN
  IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
  END IF;
  SELECT * INTO u FROM public.miniprogram_users WHERE id=p_user;
  IF NOT FOUND OR p_user=0 OR octet_length(u.openid)=0 THEN
    RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';
  END IF;
  RETURN jsonb_build_object('outcome','SUCCEEDED','openid_hex',encode(u.openid,'hex'),'primary_phone_hex',encode(u.primary_phone,'hex'));
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_identity_phone_bind_v1(p_user numeric,p_phone_hex text,p_bound_at timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE current_phone bytea; phone bytea; affected bigint; constraint_name text;
BEGIN
  IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
  END IF;
  IF p_phone_hex IS NULL OR p_phone_hex !~ '^([0-9a-f]{2})*$' THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_IDENTITY_INPUT';
  END IF;
  phone:=decode(p_phone_hex,'hex');
  SELECT primary_phone INTO current_phone FROM public.miniprogram_users WHERE id=p_user FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE'; END IF;
  IF current_phone IS NOT NULL THEN
    IF convert_from(current_phone,'UTF8') !~ '^[+][1-9][0-9]{0,14}$' THEN
      RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE';
    END IF;
    IF current_phone<>phone THEN RETURN jsonb_build_object('outcome','PRIMARY_PHONE_ALREADY_BOUND'); END IF;
  ELSE
    BEGIN
      UPDATE public.miniprogram_users SET primary_phone=phone,primary_phone_bound_at=p_bound_at
        WHERE id=p_user AND primary_phone IS NULL AND primary_phone_bound_at IS NULL;
      GET DIAGNOSTICS affected=ROW_COUNT;
      IF affected<>1 THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE'; END IF;
    EXCEPTION WHEN unique_violation THEN
      GET STACKED DIAGNOSTICS constraint_name=CONSTRAINT_NAME;
      IF constraint_name='uq_miniprogram_users_primary_phone' THEN RETURN jsonb_build_object('outcome','PHONE_IN_USE'); END IF;
      RAISE;
    END;
  END IF;
  RETURN jsonb_build_object('outcome','SUCCEEDED','phone_hex',p_phone_hex);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_identity_session_find_v1(p_token_hash text,p_now timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE user_id numeric;
BEGIN
  IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
  END IF;
  IF p_token_hash IS NULL OR p_token_hash !~ '^[0-9a-f]{64}$' THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_IDENTITY_INPUT';
  END IF;
  SELECT s.user_id INTO user_id FROM public.miniprogram_sessions s
    WHERE s.token_hash=decode(p_token_hash,'hex') AND s.issued_at<=p_now AND s.expires_at>p_now;
  IF NOT FOUND THEN RETURN jsonb_build_object('outcome','NOT_FOUND'); END IF;
  IF user_id IS NULL OR user_id=0 THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_IDENTITY_UNAVAILABLE'; END IF;
  RETURN jsonb_build_object('outcome','SUCCEEDED','user_id',user_id::text);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_identity_session_revoke_v1(p_token_hash text)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
BEGIN
  IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
  END IF;
  IF p_token_hash IS NULL OR p_token_hash !~ '^[0-9a-f]{64}$' THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_IDENTITY_INPUT';
  END IF;
  DELETE FROM public.miniprogram_sessions WHERE token_hash=decode(p_token_hash,'hex');
  RETURN jsonb_build_object('outcome','SUCCEEDED');
END
$order_rpc$;
