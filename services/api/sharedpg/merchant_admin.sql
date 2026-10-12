-- Fixed merchant account operations. Receipt bytes are decoded by the native Go contract.
CREATE OR REPLACE FUNCTION public.order_merchant_account_current_v1(p_user numeric)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE a public.merchant_accounts%ROWTYPE;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 SELECT * INTO a FROM public.merchant_accounts WHERE bound_user_id=p_user AND enabled AND deleted_at IS NULL AND role='OWNER';
 IF NOT FOUND THEN RETURN jsonb_build_object('outcome','NOT_FOUND');END IF;
 RETURN jsonb_build_object('outcome','SUCCEEDED','account',jsonb_build_object('id',a.id::text,'name',a.name,'phone',convert_from(a.phone,'UTF8'),'role',a.role,'enabled',a.enabled,'bound',a.bound_user_id IS NOT NULL));
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_merchant_account_list_v1(p_user numeric,p_query text)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE owner_id numeric; a record; accounts jsonb:='[]'::jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 SELECT id INTO owner_id FROM public.merchant_accounts WHERE bound_user_id=p_user AND enabled AND deleted_at IS NULL AND role='OWNER';
 IF NOT FOUND THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 FOR a IN SELECT id,name,convert_from(phone,'UTF8') AS phone,role,enabled,(bound_user_id IS NOT NULL) AS bound
  FROM public.merchant_accounts WHERE deleted_at IS NULL AND (p_query='' OR name LIKE '%'||p_query||'%' OR convert_from(phone,'UTF8') LIKE '%'||p_query||'%') ORDER BY id LOOP
  accounts:=accounts||jsonb_build_array(jsonb_build_object('id',a.id::text,'name',a.name,'phone',a.phone,'role',a.role,'enabled',a.enabled,'bound',a.bound));
 END LOOP;
 RETURN jsonb_build_object('outcome','SUCCEEDED','accounts',accounts);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_merchant_account_command_v1(p_actor numeric,p_kind text,p_account numeric,p_name text,p_phone text,p_role text,p_enabled boolean,p_operation_hash text,p_request_hash text,p_request_digest text,p_receipt_at timestamptz,p_input_status text)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE
 a public.merchant_accounts%ROWTYPE; actor public.merchant_accounts%ROWTYPE; target public.merchant_accounts%ROWTYPE;
 actor_found boolean:=false;target_found boolean:=false;owners bigint:=0;
 scope bytea;receipt public.action_audits%ROWTYPE;new_id numeric;v_phone text;digits text;response jsonb;account_json jsonb;
 target_id numeric; affected bigint;constraint_name text;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 -- Collection order and the conversion error boundary are the native contract.
 FOR a IN SELECT * FROM public.merchant_accounts WHERE deleted_at IS NULL ORDER BY id FOR UPDATE LOOP
  PERFORM convert_from(a.phone,'UTF8');
  IF a.bound_user_id=p_actor THEN actor:=a;actor_found:=true;END IF;
  IF a.id=p_account THEN target:=a;target_found:=true;END IF;
  IF a.role='OWNER' AND a.enabled AND a.id<>p_account THEN owners:=owners+1;END IF;
 END LOOP;
 IF NOT actor_found OR actor.role<>'OWNER' OR NOT actor.enabled OR actor.id=0 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 scope:=sha256(convert_to('merchant:'||p_actor::text||':'||actor.id::text,'UTF8'));
 SELECT * INTO receipt FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_scope_hash=scope AND action=p_kind AND operation_key_hash=decode(p_operation_hash,'hex') LIMIT 1;
 IF FOUND THEN RETURN jsonb_build_object('outcome','REPLAY','evidence_text',receipt.before_state_json::text,'response_text',receipt.response_json::text);END IF;
 IF p_kind='CREATE_ACCOUNT' THEN
  IF p_input_status='INVALID_INPUT' THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
 ELSIF p_kind='UPDATE_ACCOUNT' THEN
  IF NOT target_found THEN RETURN jsonb_build_object('outcome','NOT_FOUND');END IF;
  IF target.role='OWNER' AND target.enabled AND p_role IS DISTINCT FROM 'OWNER' AND owners=0 THEN RETURN jsonb_build_object('outcome','LAST_OWNER');END IF;
  IF p_input_status='INVALID_INPUT' THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
 ELSIF p_kind='SET_ACCOUNT_ENABLED' THEN
  IF p_enabled IS NULL THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
  IF NOT target_found THEN RETURN jsonb_build_object('outcome','NOT_FOUND');END IF;
  IF target.role='OWNER' AND target.enabled AND NOT p_enabled AND owners=0 THEN RETURN jsonb_build_object('outcome','LAST_OWNER');END IF;
 ELSIF p_kind='DELETE_ACCOUNT' THEN
  IF NOT target_found THEN RETURN jsonb_build_object('outcome','NOT_FOUND');END IF;
  IF target.role='OWNER' AND target.enabled AND owners=0 THEN RETURN jsonb_build_object('outcome','LAST_OWNER');END IF;
 ELSE RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
 IF p_input_status='STORAGE_UNAVAILABLE' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF p_input_status IS DISTINCT FROM 'VALID' THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
 IF p_kind IN ('CREATE_ACCOUNT','UPDATE_ACCOUNT') THEN
  v_phone:=p_phone;
  IF p_kind='UPDATE_ACCOUNT' AND v_phone IS NULL THEN v_phone:=convert_from(target.phone,'UTF8');END IF;
  IF v_phone IS NULL OR v_phone !~ '^[+][1-9][0-9]{0,14}$' OR p_name IS NULL OR p_name='' OR p_role IS NULL OR p_role NOT IN ('OWNER','SUBACCOUNT') THEN
   RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
 END IF;
 IF p_receipt_at IS NULL THEN RETURN jsonb_build_object('outcome','NEED_MATERIAL');END IF;
 BEGIN
  target_id:=p_account;
  IF p_kind='CREATE_ACCOUNT' THEN
   INSERT INTO public.merchant_accounts(phone,name,role,enabled,record_version,auth_version,bound_user_id,bound_at,created_at,updated_at,created_by,updated_by,deleted_at,deleted_by_account_id)
    VALUES(convert_to(v_phone,'UTF8'),p_name,p_role,TRUE,1,1,NULL,NULL,clock_timestamp(),clock_timestamp(),actor.id,actor.id,NULL,NULL) RETURNING id INTO new_id;
   target_id:=new_id;digits:=regexp_replace(v_phone,'^[+]86','');
   account_json:=jsonb_build_object('ID',new_id,'Name',p_name,'MaskedPhone',CASE WHEN length(digits)<7 THEN '***' ELSE left(digits,3)||'****'||right(digits,4) END,'Role',p_role,'Enabled',TRUE,'Bound',FALSE);
  ELSIF p_kind='UPDATE_ACCOUNT' THEN
   UPDATE public.merchant_accounts SET phone=convert_to(v_phone,'UTF8'),name=p_name,role=p_role,record_version=record_version+1,auth_version=auth_version+1,updated_at=clock_timestamp(),updated_by=actor.id WHERE id=p_account AND deleted_at IS NULL;
   GET DIAGNOSTICS affected=ROW_COUNT;IF affected<>1 THEN RETURN jsonb_build_object('outcome','NOT_FOUND');END IF;
   digits:=regexp_replace(v_phone,'^[+]86','');
   account_json:=jsonb_build_object('ID',p_account,'Name',p_name,'MaskedPhone',CASE WHEN length(digits)<7 THEN '***' ELSE left(digits,3)||'****'||right(digits,4) END,'Role',p_role,'Enabled',target.enabled,'Bound',target.bound_user_id IS NOT NULL);
  ELSIF p_kind='SET_ACCOUNT_ENABLED' THEN
   UPDATE public.merchant_accounts SET enabled=p_enabled,record_version=record_version+1,auth_version=auth_version+1,updated_at=clock_timestamp(),updated_by=actor.id WHERE id=p_account AND deleted_at IS NULL;
   GET DIAGNOSTICS affected=ROW_COUNT;IF affected<>1 THEN RETURN jsonb_build_object('outcome','NOT_FOUND');END IF;
   digits:=regexp_replace(convert_from(target.phone,'UTF8'),'^[+]86','');
   account_json:=jsonb_build_object('ID',p_account,'Name',target.name,'MaskedPhone',CASE WHEN length(digits)<7 THEN '***' ELSE left(digits,3)||'****'||right(digits,4) END,'Role',target.role,'Enabled',p_enabled,'Bound',target.bound_user_id IS NOT NULL);
  ELSE
   UPDATE public.merchant_accounts SET enabled=FALSE,bound_user_id=NULL,bound_at=NULL,deleted_at=clock_timestamp(),deleted_by_account_id=actor.id,record_version=record_version+1,auth_version=auth_version+1,updated_at=clock_timestamp(),updated_by=actor.id WHERE id=p_account AND deleted_at IS NULL;
   GET DIAGNOSTICS affected=ROW_COUNT;IF affected<>1 THEN RETURN jsonb_build_object('outcome','NOT_FOUND');END IF;
  END IF;
  IF p_kind='DELETE_ACCOUNT' THEN response:=jsonb_build_object('deleted',TRUE);ELSE response:=jsonb_build_object('account',account_json);END IF;
  INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,target_type,target_id,target_key_hash,operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at)
   VALUES('COMMAND_RECEIPT','MERCHANT',scope,p_actor,actor.id,actor.id,actor.role,actor.auth_version,p_kind,CASE WHEN target_id>0 THEN 'MERCHANT_ACCOUNT' END,CASE WHEN target_id>0 THEN target_id END,NULL,decode(p_operation_hash,'hex'),decode(p_request_hash,'hex'),'SUCCEEDED','OK',jsonb_build_object('request_digest',p_request_digest),NULL,response,p_receipt_at);
  RETURN jsonb_build_object('outcome','SUCCEEDED','response_text',response::text);
 EXCEPTION WHEN unique_violation THEN
  GET STACKED DIAGNOSTICS constraint_name=CONSTRAINT_NAME;
  IF constraint_name<>'uq_action_audits_command_receipt' THEN RAISE;END IF;
 END;
 SELECT * INTO receipt FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_scope_hash=scope AND action=p_kind AND operation_key_hash=decode(p_operation_hash,'hex') LIMIT 1;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_ADMIN_UNAVAILABLE';END IF;
 RETURN jsonb_build_object('outcome','REPLAY','evidence_text',receipt.before_state_json::text,'response_text',receipt.response_json::text);
END
$order_rpc$;
