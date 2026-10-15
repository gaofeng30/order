-- Fixed merchant dated sold-out transactions; ordinary service JWT only.
CREATE OR REPLACE FUNCTION public.order_merchant_soldout_replay_v1(p_user numeric,p_product numeric,p_service_date date,p_sold_out boolean,p_operation_hash text,p_request_id_hash text,p_request_digest text)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_account numeric;v_role text;v_enabled boolean;v_version numeric;v_auth numeric;v_receipt jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 BEGIN
  SELECT id,role,enabled,record_version,auth_version INTO v_account,v_role,v_enabled,v_version,v_auth FROM public.merchant_accounts WHERE bound_user_id=p_user FOR SHARE;
 EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('outcome','AUTH_UNAVAILABLE'); END;
 IF v_account IS NULL OR NOT v_enabled THEN RETURN jsonb_build_object('outcome','FORBIDDEN'); END IF;
 IF v_account<=0 OR v_role NOT IN('OWNER','SUBACCOUNT') OR v_version<=0 OR v_auth<=0 THEN RETURN jsonb_build_object('outcome','AUTH_UNAVAILABLE'); END IF;
 SELECT jsonb_build_object('target_type',target_type,'target_id',target_id::text,'before_text',before_state_json::text,'after_text',after_state_json::text,'response_text',response_json::text) INTO v_receipt FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='MERCHANT' AND actor_scope_hash=sha256(convert_to('merchant:'||p_user::text||':'||v_account::text,'UTF8')) AND action='merchant.product.sold_out.write' AND operation_key_hash=decode(p_operation_hash,'hex') LIMIT 1 FOR SHARE;
 RETURN jsonb_build_object('outcome','SUCCEEDED','receipt',v_receipt);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_merchant_soldout_command_v1(p_user numeric,p_product numeric,p_service_date date,p_sold_out boolean,p_now timestamptz,p_locator_found boolean,p_category numeric,p_operation_hash text,p_request_id_hash text,p_request_digest text,p_receipt_at timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_account numeric;v_role text;v_enabled boolean;v_version numeric;v_auth numeric;v_category numeric;v_sold boolean;v_changed boolean;v_rows bigint;v_constraint text;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 BEGIN
  SELECT id,role,enabled,record_version,auth_version INTO v_account,v_role,v_enabled,v_version,v_auth FROM public.merchant_accounts WHERE bound_user_id=p_user FOR SHARE;
 EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('outcome','AUTH_UNAVAILABLE'); END;
 IF v_account IS NULL OR NOT v_enabled THEN RETURN jsonb_build_object('outcome','FORBIDDEN'); END IF;
 IF v_account<=0 OR v_role NOT IN('OWNER','SUBACCOUNT') OR v_version<=0 OR v_auth<=0 THEN RETURN jsonb_build_object('outcome','AUTH_UNAVAILABLE'); END IF;
 BEGIN
  SELECT record_version INTO v_version FROM public.service_dates WHERE service_date=p_service_date FOR UPDATE;
  IF NOT FOUND THEN RETURN jsonb_build_object('outcome','NOT_FOUND'); END IF;
  IF v_version<=0 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE'); END IF;
  IF NOT p_locator_found THEN RETURN jsonb_build_object('outcome','NOT_FOUND'); END IF;
  SELECT record_version INTO v_version FROM public.categories WHERE id=p_category FOR UPDATE;
  IF NOT FOUND THEN RETURN jsonb_build_object('outcome','NOT_FOUND'); END IF;
  IF v_version<=0 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE'); END IF;
  SELECT category_id,record_version INTO v_category,v_version FROM public.products WHERE id=p_product FOR UPDATE;
  IF NOT FOUND THEN RETURN jsonb_build_object('outcome','NOT_FOUND'); END IF;
  IF v_category IS DISTINCT FROM p_category OR v_version<=0 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE'); END IF;
  PERFORM product_id FROM public.product_sold_out_dates WHERE service_date=p_service_date AND product_id=p_product FOR UPDATE;
  v_sold:=FOUND;v_changed:=v_sold IS DISTINCT FROM p_sold_out;
  IF p_receipt_at IS NULL THEN RETURN jsonb_build_object('outcome','NEED_MATERIAL'); END IF;
  IF v_changed AND p_sold_out THEN
   INSERT INTO public.product_sold_out_dates(service_date,product_id) VALUES(p_service_date,p_product);GET DIAGNOSTICS v_rows=ROW_COUNT;
   IF v_rows<>1 THEN RAISE EXCEPTION USING ERRCODE='P1009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
  ELSIF v_changed AND NOT p_sold_out THEN
   DELETE FROM public.product_sold_out_dates WHERE service_date=p_service_date AND product_id=p_product;GET DIAGNOSTICS v_rows=ROW_COUNT;
   IF v_rows<>1 THEN RAISE EXCEPTION USING ERRCODE='P1009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
  END IF;
  INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,target_type,target_id,target_key_hash,operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at)
  VALUES('COMMAND_RECEIPT','MERCHANT',sha256(convert_to('merchant:'||p_user::text||':'||v_account::text,'UTF8')),p_user,v_account,v_account,v_role,v_auth,'merchant.product.sold_out.write','PRODUCT',p_product,NULL,decode(p_operation_hash,'hex'),decode(p_request_id_hash,'hex'),'SUCCEEDED',CASE WHEN v_changed THEN 'SOLD_OUT_CHANGED' ELSE 'SOLD_OUT_UNCHANGED' END,jsonb_build_object('request_digest',p_request_digest,'sold_out',v_sold),jsonb_build_object('sold_out',p_sold_out),jsonb_build_object('product_id',p_product,'service_date',to_char(p_service_date,'YYYY-MM-DD'),'sold_out',p_sold_out),p_receipt_at);
 EXCEPTION WHEN unique_violation THEN
  GET STACKED DIAGNOSTICS v_constraint=CONSTRAINT_NAME;
  IF v_constraint='uq_action_audits_command_receipt' THEN RETURN jsonb_build_object('outcome','DUPLICATE'); END IF;
  RAISE;
 END;
 RETURN jsonb_build_object('outcome','SUCCEEDED');
END
$order_rpc$;
