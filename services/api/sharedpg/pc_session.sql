-- PC intent and opaque-session operations; raw secrets remain in Go memory.
CREATE OR REPLACE FUNCTION public.order_pc_login_begin_v1(p_approval_hash text,p_poll_hash text,p_now timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE login_id numeric;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 INSERT INTO public.merchant_pc_sessions(approval_secret_hash,poll_secret_hash,state,login_expires_at,created_at,updated_at)
  VALUES(decode(p_approval_hash,'hex'),decode(p_poll_hash,'hex'),'WAITING',p_now+interval '2 minutes',p_now,p_now) RETURNING id INTO login_id;
 RETURN jsonb_build_object('outcome','SUCCEEDED','login_id',login_id::text);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_pc_login_approval_read_v1(p_login numeric,p_approval_hash text)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE s public.merchant_pc_sessions%ROWTYPE;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 SELECT * INTO s FROM public.merchant_pc_sessions WHERE id=p_login AND approval_secret_hash=decode(p_approval_hash,'hex');
 IF NOT FOUND THEN RETURN jsonb_build_object('outcome','LOGIN_EXPIRED');END IF;
 RETURN jsonb_build_object('outcome','SUCCEEDED','state',s.state,'expires_at',s.login_expires_at);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_pc_login_approve_v1(p_user numeric,p_login numeric,p_approval_hash text)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE account_id numeric;auth_version numeric;affected bigint;observed_at timestamptz;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 SELECT id,merchant_accounts.auth_version INTO account_id,auth_version FROM public.merchant_accounts
  WHERE bound_user_id=p_user AND role='OWNER' AND enabled AND deleted_at IS NULL FOR UPDATE;
 IF NOT FOUND THEN RETURN jsonb_build_object('outcome','FORBIDDEN');END IF;
 observed_at:=clock_timestamp();
 UPDATE public.merchant_pc_sessions SET state='APPROVED',approved_account_id=account_id,approved_user_id=p_user,approved_auth_version=auth_version,approved_at=observed_at,updated_at=observed_at
  WHERE id=p_login AND approval_secret_hash=decode(p_approval_hash,'hex') AND state='WAITING' AND login_expires_at>observed_at;
 GET DIAGNOSTICS affected=ROW_COUNT;
 IF affected<>1 THEN RETURN jsonb_build_object('outcome','LOGIN_EXPIRED');END IF;
 RETURN jsonb_build_object('outcome','SUCCEEDED');
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_pc_login_poll_v1(p_login numeric,p_poll_hash text,p_now timestamptz,p_access_hash text,p_entropy_failed boolean)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE s public.merchant_pc_sessions%ROWTYPE;affected bigint;observed_at timestamptz;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 SELECT * INTO s FROM public.merchant_pc_sessions WHERE id=p_login AND poll_secret_hash=decode(p_poll_hash,'hex') FOR UPDATE;
 IF NOT FOUND THEN RETURN jsonb_build_object('outcome','LOGIN_EXPIRED');END IF;
 observed_at:=COALESCE(p_now,clock_timestamp());
 IF observed_at>=s.login_expires_at THEN RETURN jsonb_build_object('outcome','LOGIN_EXPIRED');END IF;
 IF s.state='WAITING' THEN RETURN jsonb_build_object('outcome','WAITING');END IF;
 IF s.state<>'APPROVED' THEN RETURN jsonb_build_object('outcome','LOGIN_EXPIRED');END IF;
 IF p_now IS NULL AND (p_access_hash IS NOT NULL OR p_entropy_failed) THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF p_now IS NOT NULL AND p_access_hash IS NULL AND NOT p_entropy_failed THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF p_entropy_failed THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF p_access_hash IS NULL THEN RETURN jsonb_build_object('outcome','NEED_MATERIAL','observed_at',observed_at);END IF;
 IF p_access_hash !~ '^[0-9a-f]{64}$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 UPDATE public.merchant_pc_sessions SET state='CONSUMED',consumed_at=observed_at,access_token_hash=decode(p_access_hash,'hex'),access_issued_at=observed_at,access_expires_at=observed_at+interval '12 hours',updated_at=observed_at
  WHERE id=p_login AND state='APPROVED';
 GET DIAGNOSTICS affected=ROW_COUNT;
 IF affected<>1 THEN RETURN jsonb_build_object('outcome','LOGIN_EXPIRED');END IF;
 RETURN jsonb_build_object('outcome','APPROVED','expires_at',observed_at+interval '12 hours');
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_pc_session_authenticate_v1(p_token_hash text,p_now timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE user_id numeric;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 SELECT s.approved_user_id INTO user_id FROM public.merchant_pc_sessions s JOIN public.merchant_accounts a
  ON a.id=s.approved_account_id AND a.bound_user_id=s.approved_user_id AND a.auth_version=s.approved_auth_version
  WHERE s.access_token_hash=decode(p_token_hash,'hex') AND s.state='CONSUMED' AND s.access_expires_at>p_now AND a.enabled AND a.deleted_at IS NULL AND a.role='OWNER';
 IF NOT FOUND THEN RETURN jsonb_build_object('outcome','SESSION_EXPIRED');END IF;
 IF user_id=0 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 RETURN jsonb_build_object('outcome','SUCCEEDED','user_id',user_id::text);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_pc_session_revoke_v1(p_token_hash text)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 DELETE FROM public.merchant_pc_sessions WHERE access_token_hash=decode(p_token_hash,'hex');
 RETURN jsonb_build_object('outcome','SUCCEEDED');
END
$order_rpc$;
