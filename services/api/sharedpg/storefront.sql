CREATE OR REPLACE FUNCTION public.order_storefront_get_v1()
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_result jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 SELECT jsonb_build_object('outcome','SUCCEEDED','settings',jsonb_build_object('StoreName',store_name,'StoreAddress',store_address,'PickupPoint',pickup_point,'Announcement',announcement,'BusinessStatus',business_status,'Flavors',flavor_options_json),
 'launch_key',convert_from(launch_image_object_key,'UTF8'),'center_x',center_x,'center_y',center_y,'width_ratio',width_ratio,'aspect_ratio',aspect_ratio)
 INTO v_result FROM public.storefront_settings WHERE id=1;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P1009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
 RETURN v_result;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_storefront_admin_v1()
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_result jsonb;v_count bigint;v_min integer;v_max integer;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 SELECT COUNT(*),MIN(interval_minutes),MAX(interval_minutes) INTO v_count,v_min,v_max FROM public.meal_periods;
 IF v_count<>2 OR v_min<>v_max THEN RAISE EXCEPTION USING ERRCODE='P1009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
 SELECT jsonb_build_object('outcome','SUCCEEDED','settings',jsonb_build_object('StoreName',s.store_name,'PickupPoint',s.pickup_point,'Notice',s.announcement,'StoreStatus',s.business_status,'ServiceDate',to_char(clock_timestamp() AT TIME ZONE 'Asia/Shanghai','YYYY-MM-DD'),'PickupStepMin',v_min,
 'MealPeriods',(SELECT jsonb_agg(jsonb_build_object('Code',code,'Name',CASE code WHEN 'lunch' THEN '午餐' WHEN 'dinner' THEN '晚餐' ELSE '' END,'CutoffTime',to_char(DATE '2000-01-01'+cutoff_time,'HH24:MI'),'PickupFrom',to_char(DATE '2000-01-01'+pickup_start_time,'HH24:MI'),'PickupTo',to_char(DATE '2000-01-01'+pickup_end_time,'HH24:MI')) ORDER BY CASE code WHEN 'lunch' THEN 1 WHEN 'dinner' THEN 2 ELSE 3 END) FROM public.meal_periods),
 'ServiceDates',(SELECT jsonb_agg(jsonb_build_object('Date',to_char(service_date,'YYYY-MM-DD'),'Status',CASE WHEN is_open THEN 'open' ELSE 'closed' END) ORDER BY service_date) FROM(SELECT service_date,is_open FROM public.service_dates WHERE service_date>=(clock_timestamp() AT TIME ZONE 'Asia/Shanghai')::date ORDER BY service_date LIMIT 31) dates)))
 INTO v_result FROM public.storefront_settings s WHERE s.id=1;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P1009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
 RETURN v_result;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_storefront_launch_read_v1()
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_result jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 SELECT jsonb_build_object('outcome','SUCCEEDED','launch_key',convert_from(launch_image_object_key,'UTF8'),'center_x',center_x::text,'center_y',center_y::text,'width_ratio',width_ratio::text,'aspect_ratio',aspect_ratio::text) INTO v_result FROM public.storefront_settings WHERE id=1;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P1009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
 RETURN v_result;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_storefront_configure_v1(p_user numeric,p_store_status_hex text,p_notice_hex text,p_pickup_point_hex text,p_pickup_step_min integer,p_meal_periods jsonb,p_service_dates jsonb,p_operation_hash text,p_request_id_hash text,p_request_digest text,p_receipt_at timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_account numeric;v_role text;v_auth numeric;v_scope bytea;v_item jsonb;v_count bigint;v_result jsonb;v_before text;v_response text;v_constraint text;v_stage text:='BUSINESS';
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 IF p_pickup_step_min IS NULL OR p_pickup_step_min=0 OR p_meal_periods IS NULL OR jsonb_typeof(p_meal_periods)<>'array' OR jsonb_array_length(p_meal_periods)<>2 THEN RETURN jsonb_build_object('outcome','INVALID_INPUT'); END IF;
 BEGIN
  SELECT id,role,auth_version INTO v_account,v_role,v_auth FROM public.merchant_accounts WHERE bound_user_id=p_user AND enabled AND deleted_at IS NULL AND role='OWNER' FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P2009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
  v_scope:=sha256(convert_to('merchant:'||p_user::text||':'||v_account::text,'UTF8'));
  PERFORM id FROM public.storefront_settings WHERE id=1 FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P2009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
  -- No replay here: the native path reaches receipt uniqueness after all writes.
  IF p_receipt_at IS NULL THEN RETURN jsonb_build_object('outcome','NEED_MATERIAL'); END IF;
  FOR v_item IN SELECT value FROM jsonb_array_elements(p_service_dates) LOOP
   IF v_item->>'date_hex'='' OR v_item->>'status_hex' NOT IN('6f70656e','636c6f736564') THEN RAISE EXCEPTION USING ERRCODE='P2001',MESSAGE='ORDER_INVALID_INPUT'; END IF;
   INSERT INTO public.service_dates(service_date,is_open,record_version,updated_by_account_id,updated_at)
   VALUES(convert_from(decode(v_item->>'date_hex','hex'),'UTF8')::date,v_item->>'status_hex'='6f70656e',1,v_account,clock_timestamp())
   ON CONFLICT(service_date) DO UPDATE SET is_open=EXCLUDED.is_open,record_version=service_dates.record_version+1,updated_by_account_id=EXCLUDED.updated_by_account_id,updated_at=EXCLUDED.updated_at;
  END LOOP;
  PERFORM code FROM public.meal_periods WHERE code IN('lunch','dinner') ORDER BY CASE code WHEN 'lunch' THEN 1 WHEN 'dinner' THEN 2 ELSE 3 END FOR UPDATE;
  GET DIAGNOSTICS v_count=ROW_COUNT;
  IF v_count<>2 THEN RAISE EXCEPTION USING ERRCODE='P2009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
  UPDATE public.storefront_settings SET pickup_point=convert_from(decode(p_pickup_point_hex,'hex'),'UTF8'),announcement=convert_from(decode(p_notice_hex,'hex'),'UTF8'),business_status=convert_from(decode(p_store_status_hex,'hex'),'UTF8'),record_version=record_version+1 WHERE id=1;
  FOR v_item IN SELECT value FROM jsonb_array_elements(p_meal_periods) LOOP
   IF v_item->>'code_hex' NOT IN('6c756e6368','64696e6e6572') THEN RAISE EXCEPTION USING ERRCODE='P2001',MESSAGE='ORDER_INVALID_INPUT'; END IF;
   UPDATE public.meal_periods SET cutoff_time=convert_from(decode(v_item->>'cutoff_hex','hex'),'UTF8')::time,pickup_start_time=convert_from(decode(v_item->>'pickup_from_hex','hex'),'UTF8')::time,pickup_end_time=convert_from(decode(v_item->>'pickup_to_hex','hex'),'UTF8')::time,interval_minutes=p_pickup_step_min WHERE code=convert_from(decode(v_item->>'code_hex','hex'),'UTF8');
  END LOOP;
  v_result:=public.order_storefront_admin_v1()->'settings';
  v_stage:='RECEIPT';
  IF p_request_digest IS NULL THEN RAISE EXCEPTION USING ERRCODE='P2009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
  INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,target_type,target_id,target_key_hash,operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at)
  VALUES('COMMAND_RECEIPT','MERCHANT',v_scope,p_user,v_account,v_account,v_role,v_auth,'CONFIGURE_STOREFRONT','STOREFRONT',1,NULL,decode(p_operation_hash,'hex'),decode(p_request_id_hash,'hex'),'SUCCEEDED','OK',jsonb_build_object('request_digest',p_request_digest),NULL,v_result,p_receipt_at);
  RETURN jsonb_build_object('outcome','SUCCEEDED','response_text',v_result::text);
 EXCEPTION
  WHEN SQLSTATE 'P2001' THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');
  WHEN unique_violation THEN
   GET STACKED DIAGNOSTICS v_constraint=CONSTRAINT_NAME;
   IF v_constraint='uq_action_audits_command_receipt' THEN
    SELECT before_state_json::text,response_json::text INTO v_before,v_response FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_scope_hash=v_scope AND action='CONFIGURE_STOREFRONT' AND operation_key_hash=decode(p_operation_hash,'hex') LIMIT 1;
    IF FOUND THEN RETURN jsonb_build_object('outcome','REPLAY','before_text',v_before,'response_text',v_response); END IF;
    RETURN jsonb_build_object('outcome','UNAVAILABLE');
   END IF;
   RETURN jsonb_build_object('outcome',CASE WHEN v_stage='BUSINESS' THEN 'CONFLICT' ELSE 'UNAVAILABLE' END);
  WHEN foreign_key_violation OR check_violation THEN RETURN jsonb_build_object('outcome',CASE WHEN v_stage='BUSINESS' THEN 'CONFLICT' ELSE 'UNAVAILABLE' END);
  WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_storefront_launch_configure_v1(p_user numeric,p_layer_present boolean,p_enabled boolean,p_image_key_hex text,p_image_key_json_hex text,p_center_x text,p_center_y text,p_width_ratio text,p_aspect_ratio text,p_confirm_status text,p_operation_hash text,p_request_id_hash text,p_request_digest text,p_receipt_at timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_account numeric;v_role text;v_auth numeric;v_scope bytea;v_result jsonb;v_before text;v_response text;v_constraint text;v_stage text:='BUSINESS';v_cx double precision;v_cy double precision;v_width double precision;v_aspect double precision;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 BEGIN
  SELECT id,role,auth_version INTO v_account,v_role,v_auth FROM public.merchant_accounts WHERE bound_user_id=p_user AND enabled AND deleted_at IS NULL AND role='OWNER' FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P2009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
  v_scope:=sha256(convert_to('merchant:'||p_user::text||':'||v_account::text,'UTF8'));
  IF p_layer_present AND p_enabled THEN
   IF p_confirm_status='UNAVAILABLE' THEN RAISE EXCEPTION USING ERRCODE='P2009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
   IF p_confirm_status='NOT_CONFIRMED' THEN RAISE EXCEPTION USING ERRCODE='P2001',MESSAGE='ORDER_INVALID_INPUT'; END IF;
   IF p_confirm_status IS DISTINCT FROM 'CONFIRMED' THEN RAISE EXCEPTION USING ERRCODE='P2009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
  END IF;
  PERFORM id FROM public.storefront_settings WHERE id=1 FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P2009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
  -- No early replay: confirmation, writes and constraints precede uniqueness.
  IF p_receipt_at IS NULL THEN RETURN jsonb_build_object('outcome','NEED_MATERIAL'); END IF;
  IF NOT COALESCE(p_layer_present AND p_enabled,false) THEN
   UPDATE public.storefront_settings SET launch_image_object_key=NULL,center_x=NULL,center_y=NULL,width_ratio=NULL,aspect_ratio=NULL,record_version=record_version+1 WHERE id=1;
   v_result:=jsonb_build_object('ImageObjectKey','','Enabled',false,'SizeRatio',0,'CenterX',0,'CenterY',0,'AspectRatio',0);
  ELSE
   v_cx:=p_center_x::double precision;v_cy:=p_center_y::double precision;v_width:=p_width_ratio::double precision;v_aspect:=p_aspect_ratio::double precision;
   UPDATE public.storefront_settings SET launch_image_object_key=decode(p_image_key_hex,'hex'),center_x=v_cx,center_y=v_cy,width_ratio=v_width,aspect_ratio=v_aspect,record_version=record_version+1 WHERE id=1;
   v_stage:='RECEIPT';
   -- Go response Marshal rejects all non-finite floats after the business UPDATE.
   IF v_cx::text IN('NaN','Infinity','-Infinity') OR v_cy::text IN('NaN','Infinity','-Infinity') OR v_width::text IN('NaN','Infinity','-Infinity') OR v_aspect::text IN('NaN','Infinity','-Infinity') THEN RAISE EXCEPTION USING ERRCODE='P2009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
   v_result:=jsonb_build_object('ImageObjectKey',convert_from(decode(p_image_key_json_hex,'hex'),'UTF8')::jsonb,'Enabled',true,'SizeRatio',v_width,'CenterX',v_cx,'CenterY',v_cy,'AspectRatio',v_aspect);
  END IF;
  v_stage:='RECEIPT';
  IF p_request_digest IS NULL THEN RAISE EXCEPTION USING ERRCODE='P2009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
  INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,target_type,target_id,target_key_hash,operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at)
  VALUES('COMMAND_RECEIPT','MERCHANT',v_scope,p_user,v_account,v_account,v_role,v_auth,'CONFIGURE_LAUNCH_LAYER','STOREFRONT',1,NULL,decode(p_operation_hash,'hex'),decode(p_request_id_hash,'hex'),'SUCCEEDED','OK',jsonb_build_object('request_digest',p_request_digest),NULL,v_result,p_receipt_at);
  RETURN jsonb_build_object('outcome','SUCCEEDED','response_text',v_result::text);
 EXCEPTION
  WHEN SQLSTATE 'P2001' THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');
  WHEN unique_violation THEN
   GET STACKED DIAGNOSTICS v_constraint=CONSTRAINT_NAME;
   IF v_constraint='uq_action_audits_command_receipt' THEN
    SELECT before_state_json::text,response_json::text INTO v_before,v_response FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_scope_hash=v_scope AND action='CONFIGURE_LAUNCH_LAYER' AND operation_key_hash=decode(p_operation_hash,'hex') LIMIT 1;
    IF FOUND THEN RETURN jsonb_build_object('outcome','REPLAY','before_text',v_before,'response_text',v_response); END IF;
    RETURN jsonb_build_object('outcome','UNAVAILABLE');
   END IF;
   RETURN jsonb_build_object('outcome',CASE WHEN v_stage='BUSINESS' THEN 'CONFLICT' ELSE 'UNAVAILABLE' END);
  WHEN foreign_key_violation OR check_violation THEN RETURN jsonb_build_object('outcome',CASE WHEN v_stage='BUSINESS' THEN 'CONFLICT' ELSE 'UNAVAILABLE' END);
  WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END;
END
$order_rpc$;
