-- Fixed Staff administration ports; ordinary JWT service UID only.
CREATE OR REPLACE FUNCTION public.order_staff_admin_list_v1(p_query_hex text)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_query text;v_result jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 v_query:=convert_from(decode(p_query_hex,'hex'),'UTF8');
 SELECT COALESCE(jsonb_agg(value ORDER BY id DESC),'[]'::jsonb) INTO v_result FROM (
  SELECT s.id,jsonb_build_object('id',s.id::text,'phone',convert_from(s.phone,'UTF8'),'name',s.name,'enabled',s.enabled,'created_at',s.created_at,
   'bound',EXISTS(SELECT 1 FROM public.miniprogram_users u WHERE u.primary_phone=s.phone OR u.extra_phone=s.phone),
   'spend_cents',COALESCE((SELECT SUM(o.payable_cents) FROM public.orders o JOIN public.miniprogram_users u ON u.id=o.user_id WHERE (u.primary_phone=s.phone OR u.extra_phone=s.phone) AND o.identity_kind='STAFF' AND o.state<>'REFUNDED'),0)::text,
   'order_count',COALESCE((SELECT COUNT(*) FROM public.orders o JOIN public.miniprogram_users u ON u.id=o.user_id WHERE (u.primary_phone=s.phone OR u.extra_phone=s.phone) AND o.identity_kind='STAFF' AND o.state<>'REFUNDED'),0)::text) value
  FROM public.staff_whitelist s WHERE (v_query='' OR s.name LIKE '%'||v_query||'%' OR convert_from(s.phone,'UTF8') LIKE '%'||v_query||'%') ORDER BY s.id DESC LIMIT 5000
 ) rows;
 RETURN jsonb_build_object('outcome','SUCCEEDED','staff',v_result);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_staff_admin_command_v1(p_user numeric,p_kind text,p_staff numeric,p_name_hex text,p_name_key_hex text,p_phone_hex text,p_enabled boolean,p_rate integer,p_input_status text,p_operation_hash text,p_request_id_hash text,p_request_digest text,p_receipt_at timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE
 v_account numeric;v_role text;v_auth numeric;v_scope bytea;v_before text;v_response text;
 v_stage text:='BUSINESS';v_constraint text;v_settings integer;v_id numeric;v_enabled boolean;v_name text;v_phone text;v_name_key bytea;v_count bigint;
 v_result jsonb:=jsonb_build_object('Staff',NULL,'RatePercent',0);v_staff jsonb;v_mask text;v_target_type text;v_target numeric;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 BEGIN
  SELECT id,role,auth_version INTO v_account,v_role,v_auth FROM public.merchant_accounts WHERE bound_user_id=p_user AND enabled AND deleted_at IS NULL AND role='OWNER' FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P2009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
  SELECT id INTO v_settings FROM public.discount_settings WHERE id=1 FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P2009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
  v_scope:=sha256(convert_to('merchant:'||p_user::text||':'||v_account::text,'UTF8'));
  SELECT before_state_json::text,response_json::text INTO v_before,v_response FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_scope_hash=v_scope AND action=p_kind AND operation_key_hash=decode(p_operation_hash,'hex') LIMIT 1;
  IF FOUND THEN RETURN jsonb_build_object('outcome','REPLAY','before_text',v_before,'response_text',v_response); END IF;
  CASE p_kind
  WHEN 'SET_DISCOUNT_RATE' THEN
   IF p_rate IS NULL OR p_rate NOT BETWEEN 1 AND 100 THEN RAISE EXCEPTION USING ERRCODE='P2001',MESSAGE='ORDER_INVALID_INPUT'; END IF;
   IF p_receipt_at IS NULL THEN RETURN jsonb_build_object('outcome','NEED_RECEIPT'); END IF;
   UPDATE public.discount_settings SET rate_percent=p_rate,discount_version=discount_version+1,updated_at=clock_timestamp() WHERE id=1;
   v_result:=jsonb_build_object('Staff',NULL,'RatePercent',p_rate);
  WHEN 'CREATE_STAFF','UPDATE_STAFF' THEN
   IF p_kind='UPDATE_STAFF' THEN
    BEGIN
     SELECT convert_from(phone,'UTF8'),enabled INTO v_phone,v_enabled FROM public.staff_whitelist WHERE id=p_staff FOR UPDATE;
     IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P2002',MESSAGE='ORDER_NOT_FOUND'; END IF;
    EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION USING ERRCODE='P2002',MESSAGE='ORDER_NOT_FOUND';
    END;
   END IF;
   IF p_input_status='INVALID_INPUT' THEN RAISE EXCEPTION USING ERRCODE='P2001',MESSAGE='ORDER_INVALID_INPUT'; END IF;
   IF p_input_status IS DISTINCT FROM 'VALID' THEN RAISE EXCEPTION USING ERRCODE='P2009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
   IF p_receipt_at IS NULL THEN RETURN jsonb_build_object('outcome','NEED_RECEIPT'); END IF;
   v_name:=convert_from(decode(p_name_hex,'hex'),'UTF8');v_name_key:=decode(p_name_key_hex,'hex');
   IF p_kind='CREATE_STAFF' OR p_phone_hex IS NOT NULL THEN v_phone:=convert_from(decode(p_phone_hex,'hex'),'UTF8'); END IF;
   IF p_kind='CREATE_STAFF' THEN
    INSERT INTO public.staff_whitelist(phone,name,name_key,enabled,record_version,created_at,updated_at) VALUES(convert_to(v_phone,'UTF8'),v_name,v_name_key,true,1,clock_timestamp(),clock_timestamp()) RETURNING id INTO v_id;
    v_enabled:=true;
   ELSE
    v_id:=p_staff;
    UPDATE public.staff_whitelist SET phone=convert_to(v_phone,'UTF8'),name=v_name,name_key=v_name_key,record_version=record_version+1,updated_at=clock_timestamp() WHERE id=p_staff;
   END IF;
   v_stage:='VERSION';
   UPDATE public.discount_settings SET whitelist_version=whitelist_version+1,updated_at=clock_timestamp() WHERE id=1;
  WHEN 'SET_STAFF_ENABLED' THEN
   IF p_enabled IS NULL THEN RAISE EXCEPTION USING ERRCODE='P2001',MESSAGE='ORDER_INVALID_INPUT'; END IF;
   BEGIN
    SELECT name,convert_from(phone,'UTF8') INTO v_name,v_phone FROM public.staff_whitelist WHERE id=p_staff FOR UPDATE;
    IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P2002',MESSAGE='ORDER_NOT_FOUND'; END IF;
   EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION USING ERRCODE='P2002',MESSAGE='ORDER_NOT_FOUND';
   END;
   IF p_receipt_at IS NULL THEN RETURN jsonb_build_object('outcome','NEED_RECEIPT'); END IF;
   v_id:=p_staff;v_enabled:=p_enabled;
   UPDATE public.staff_whitelist SET enabled=p_enabled,record_version=record_version+1,updated_at=clock_timestamp() WHERE id=p_staff;
   v_stage:='VERSION';
   UPDATE public.discount_settings SET whitelist_version=whitelist_version+1,updated_at=clock_timestamp() WHERE id=1;
  WHEN 'DELETE_STAFF' THEN
   PERFORM id FROM public.staff_whitelist WHERE id=p_staff FOR UPDATE;
   IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P2002',MESSAGE='ORDER_NOT_FOUND'; END IF;
   IF p_receipt_at IS NULL THEN RETURN jsonb_build_object('outcome','NEED_RECEIPT'); END IF;
   DELETE FROM public.staff_whitelist WHERE id=p_staff;
   GET DIAGNOSTICS v_count=ROW_COUNT;
   IF v_count<>1 THEN RAISE EXCEPTION USING ERRCODE='P2002',MESSAGE='ORDER_NOT_FOUND'; END IF;
   v_stage:='VERSION';
   UPDATE public.discount_settings SET whitelist_version=whitelist_version+1,updated_at=clock_timestamp() WHERE id=1;
  ELSE RAISE EXCEPTION USING ERRCODE='P2001',MESSAGE='ORDER_INVALID_INPUT';
  END CASE;
  IF p_kind IN('CREATE_STAFF','UPDATE_STAFF','SET_STAFF_ENABLED') THEN
   v_mask:=CASE WHEN left(v_phone,3)='+86' THEN substring(v_phone FROM 4) ELSE v_phone END;
   v_mask:=CASE WHEN length(v_mask)<7 THEN '***' ELSE left(v_mask,3)||'****'||right(v_mask,4) END;
   v_staff:=jsonb_build_object('ID',v_id,'Name',v_name,'MaskedPhone',v_mask,'Enabled',v_enabled,'Bound',false,'SpendCents',0,'OrderCount',0,'CreatedAt','0001-01-01T00:00:00Z');
   v_result:=jsonb_build_object('Staff',v_staff,'RatePercent',0);
  END IF;
  v_target_type:=CASE WHEN p_staff>0 THEN 'STAFF' ELSE 'DISCOUNT_SETTINGS' END;
  v_target:=CASE WHEN p_staff>0 THEN p_staff ELSE 1 END;
  v_stage:='RECEIPT';
  INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,target_type,target_id,target_key_hash,operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at)
  VALUES('COMMAND_RECEIPT','MERCHANT',v_scope,p_user,v_account,v_account,v_role,v_auth,p_kind,v_target_type,v_target,NULL,decode(p_operation_hash,'hex'),CASE WHEN p_request_id_hash IS NULL THEN NULL ELSE decode(p_request_id_hash,'hex') END,'SUCCEEDED','OK',jsonb_build_object('request_digest',p_request_digest),NULL,v_result,p_receipt_at);
  RETURN jsonb_build_object('outcome','SUCCEEDED','response_text',v_result::text);
 EXCEPTION
  WHEN SQLSTATE 'P2001' THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');
  WHEN SQLSTATE 'P2002' THEN RETURN jsonb_build_object('outcome','NOT_FOUND');
  WHEN SQLSTATE 'P2009' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');
  WHEN unique_violation THEN
   GET STACKED DIAGNOSTICS v_constraint=CONSTRAINT_NAME;
   IF v_stage='RECEIPT' AND v_constraint='uq_action_audits_command_receipt' THEN
    SELECT before_state_json::text,response_json::text INTO v_before,v_response FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_scope_hash=v_scope AND action=p_kind AND operation_key_hash=decode(p_operation_hash,'hex') LIMIT 1;
    IF FOUND THEN RETURN jsonb_build_object('outcome','REPLAY','before_text',v_before,'response_text',v_response); END IF;
   END IF;
   RETURN jsonb_build_object('outcome',CASE WHEN v_stage='BUSINESS' THEN 'CONFLICT' ELSE 'UNAVAILABLE' END);
  WHEN foreign_key_violation OR check_violation THEN RETURN jsonb_build_object('outcome',CASE WHEN v_stage='BUSINESS' THEN 'CONFLICT' ELSE 'UNAVAILABLE' END);
  WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_staff_admin_rate_v1()
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_rate integer;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 SELECT rate_percent INTO v_rate FROM public.discount_settings WHERE id=1;
 RETURN jsonb_build_object('outcome','SUCCEEDED','rate_percent',v_rate);
END
$order_rpc$;
