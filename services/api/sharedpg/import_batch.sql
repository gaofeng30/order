-- Fixed import ports. XLSX parsing and original Go plan normalization stay in Go.
CREATE OR REPLACE FUNCTION public.order_import_preview_facts_v1(p_kind text)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_products jsonb;v_categories jsonb;v_staff jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 IF p_kind='PRODUCT' THEN
  SELECT COALESCE(jsonb_agg(encode(name_key,'hex')),'[]'::jsonb) INTO v_products FROM public.products;
  SELECT COALESCE(jsonb_agg(encode(name_key,'hex')),'[]'::jsonb) INTO v_categories FROM public.categories;
  RETURN jsonb_build_object('outcome','SUCCEEDED','products',v_products,'categories',v_categories);
 ELSIF p_kind='STAFF' THEN
  -- Native convert_from must still fail on a corrupt UTF8 phone at this read stage.
  SELECT COALESCE(jsonb_agg(encode(convert_to(convert_from(phone,'UTF8'),'UTF8'),'hex')),'[]'::jsonb) INTO v_staff FROM public.staff_whitelist;
  RETURN jsonb_build_object('outcome','SUCCEEDED','staff',v_staff);
 END IF;
 RETURN jsonb_build_object('outcome','INVALID_INPUT');
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_import_preview_v1(
 p_user numeric,p_kind text,p_file_digest text,p_file_size numeric,p_row_count numeric,
 p_token_hash text,p_plan_json text,p_response_json text,p_captured_at timestamptz,
 p_operation_hash text,p_request_id_hash text,p_request_digest text,p_receipt_at timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_account numeric;v_role text;v_auth numeric;v_scope bytea;v_id numeric;
 v_plan jsonb;v_response jsonb;v_before text;v_old_response text;v_constraint text;v_stage text:='BUSINESS';
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 BEGIN
  SELECT id,role,auth_version INTO v_account,v_role,v_auth FROM public.merchant_accounts
   WHERE bound_user_id=p_user AND enabled AND deleted_at IS NULL AND role='OWNER' FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P5009',MESSAGE='ORDER_STORAGE_UNAVAILABLE';END IF;
  v_scope:=sha256(convert_to('merchant:'||p_user::text||':'||v_account::text,'UTF8'));
  v_plan:=p_plan_json::jsonb;v_response:=p_response_json::jsonb;
  IF v_plan->>'kind' IS DISTINCT FROM p_kind OR p_kind NOT IN('PRODUCT','STAFF')
   OR sha256(convert_to(v_response->>'Token','UTF8')) IS DISTINCT FROM decode(p_token_hash,'hex')
   OR (v_response->>'NewCount')::numeric IS DISTINCT FROM (v_plan->>'new_count')::numeric
   OR (v_response->>'UpdateCount')::numeric IS DISTINCT FROM (v_plan->>'update_count')::numeric
   OR (v_response->>'ErrorCount')::numeric IS DISTINCT FROM (v_plan->>'error_count')::numeric
   OR (v_response->>'ExpiresAt')::timestamptz IS DISTINCT FROM p_captured_at+interval '15 minutes' THEN
   RAISE EXCEPTION USING ERRCODE='P5009',MESSAGE='ORDER_STORAGE_UNAVAILABLE';
  END IF;
  IF p_receipt_at IS NULL THEN RETURN jsonb_build_object('outcome','NEED_RECEIPT');END IF;
  INSERT INTO public.import_batches(kind,actor_account_id,file_object_key,file_digest,file_size_bytes,row_count,
   preview_token_hash,preview_json,new_count,update_count,error_count,state,expires_at,record_version,created_at,updated_at)
   VALUES(p_kind,v_account,NULL,decode(p_file_digest,'hex'),p_file_size,p_row_count,decode(p_token_hash,'hex'),v_plan,
   (v_plan->>'new_count')::bigint,(v_plan->>'update_count')::bigint,(v_plan->>'error_count')::bigint,
   'PREVIEWED',p_captured_at+interval '15 minutes',1,p_captured_at,p_captured_at) RETURNING id INTO v_id;
  v_stage:='RECEIPT';
  INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,
   actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,target_type,target_id,target_key_hash,
   operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at)
   VALUES('COMMAND_RECEIPT','MERCHANT',v_scope,p_user,v_account,v_account,v_role,v_auth,'PREVIEW_IMPORT','IMPORT_BATCH',v_id,NULL,
   decode(p_operation_hash,'hex'),decode(p_request_id_hash,'hex'),'SUCCEEDED','OK',jsonb_build_object('request_digest',p_request_digest),NULL,v_response,p_receipt_at);
  RETURN jsonb_build_object('outcome','SUCCEEDED');
 EXCEPTION
  WHEN unique_violation THEN
   GET STACKED DIAGNOSTICS v_constraint=CONSTRAINT_NAME;
   IF v_stage='RECEIPT' AND v_constraint='uq_action_audits_command_receipt' THEN
    SELECT before_state_json::text,response_json::text INTO v_before,v_old_response FROM public.action_audits
     WHERE entry_kind='COMMAND_RECEIPT' AND actor_scope_hash=v_scope AND action='PREVIEW_IMPORT' AND operation_key_hash=decode(p_operation_hash,'hex') LIMIT 1;
    IF FOUND THEN RETURN jsonb_build_object('outcome','REPLAY','before_text',v_before,'response_text',v_old_response);END IF;
    RETURN jsonb_build_object('outcome','UNAVAILABLE');
   END IF;
   RETURN jsonb_build_object('outcome',CASE WHEN v_stage='BUSINESS' THEN 'INVALID_INPUT' ELSE 'UNAVAILABLE' END);
  WHEN foreign_key_violation OR check_violation THEN RETURN jsonb_build_object('outcome',CASE WHEN v_stage='BUSINESS' THEN 'INVALID_INPUT' ELSE 'UNAVAILABLE' END);
  WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_import_commit_v1(
 p_user numeric,p_token_hash text,p_skip boolean,p_operation_hash text,p_request_id_hash text,p_request_digest text,
 p_plan_digest text,p_material jsonb,p_observed_at timestamptz,p_receipt_at timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_account numeric;v_role text;v_auth numeric;v_scope bytea;
 v_id numeric;v_kind text;v_plan jsonb;v_state text;v_expires timestamptz;
 v_new bigint;v_update bigint;v_error bigint;v_committed_key bytea;
 v_before text;v_response text;v_constraint text;v_stage text:='READ';v_observed timestamptz;
 v_result jsonb;v_row jsonb;v_material jsonb;v_name text;v_key bytea;v_category numeric;v_categories jsonb:='{}'::jsonb;
 v_record numeric;v_rows jsonb:='[]'::jsonb;v_phone text;
 v_category_row record;v_index integer:=0;v_count bigint;v_singleton smallint;v_n integer;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 BEGIN
  SELECT id,role,auth_version INTO v_account,v_role,v_auth FROM public.merchant_accounts
   WHERE bound_user_id=p_user AND enabled AND deleted_at IS NULL AND role='OWNER' FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P5009',MESSAGE='ORDER_STORAGE_UNAVAILABLE';END IF;
  v_scope:=sha256(convert_to('merchant:'||p_user::text||':'||v_account::text,'UTF8'));
  SELECT id,kind,preview_json,state,expires_at,new_count,update_count,error_count,commit_idempotency_key_hash
   INTO v_id,v_kind,v_plan,v_state,v_expires,v_new,v_update,v_error,v_committed_key FROM public.import_batches
   WHERE preview_token_hash=decode(p_token_hash,'hex') AND actor_account_id=v_account FOR UPDATE;
  IF NOT FOUND THEN RETURN jsonb_build_object('outcome','PREVIEW_EXPIRED');END IF;
  IF v_state='COMMITTED' THEN
   IF v_committed_key IS DISTINCT FROM decode(p_operation_hash,'hex') THEN RETURN jsonb_build_object('outcome','IDEMPOTENCY_CONFLICT');END IF;
   SELECT before_state_json::text,response_json::text INTO v_before,v_response FROM public.action_audits
    WHERE entry_kind='COMMAND_RECEIPT' AND actor_scope_hash=v_scope AND action='COMMIT_IMPORT' AND operation_key_hash=decode(p_operation_hash,'hex') LIMIT 1;
   IF FOUND THEN RETURN jsonb_build_object('outcome','REPLAY','before_text',v_before,'response_text',v_response);END IF;
   RETURN jsonb_build_object('outcome','UNAVAILABLE');
  END IF;
  IF p_material IS NULL THEN v_observed:=clock_timestamp();ELSE v_observed:=p_observed_at;END IF;
  IF v_state<>'PREVIEWED' OR v_observed IS NULL OR NOT(v_observed<v_expires) THEN RETURN jsonb_build_object('outcome','PREVIEW_EXPIRED');END IF;
  IF v_error>0 AND NOT p_skip THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');END IF;
  IF p_material IS NULL THEN
   RETURN jsonb_build_object('outcome','NEED_MATERIAL','kind',v_kind,'plan_text',v_plan::text,
    'plan_digest',encode(sha256(convert_to(v_plan::text,'UTF8')),'hex'),'observed_at',v_observed);
  END IF;
  IF p_material->>'status' IS DISTINCT FROM 'VALID' OR p_plan_digest IS DISTINCT FROM encode(sha256(convert_to(v_plan::text,'UTF8')),'hex')
   OR p_receipt_at IS NULL THEN RAISE EXCEPTION USING ERRCODE='P5009',MESSAGE='ORDER_STORAGE_UNAVAILABLE';END IF;
  -- Go's tolerant original plan decoder controls validity; SQL consumes that complete immutable plan.
  IF jsonb_typeof(p_material->'rows') IS DISTINCT FROM 'array' OR jsonb_typeof(p_material->'new_categories') IS DISTINCT FROM 'array' THEN
   RAISE EXCEPTION USING ERRCODE='P5009',MESSAGE='ORDER_STORAGE_UNAVAILABLE';END IF;
  IF jsonb_array_length(p_material->'rows')<>(SELECT count(*) FROM jsonb_array_elements(COALESCE(NULLIF(v_plan->'rows','null'::jsonb),'[]'::jsonb)) r WHERE COALESCE(r->>'outcome','')<>'ERROR') THEN
   RAISE EXCEPTION USING ERRCODE='P5009',MESSAGE='ORDER_STORAGE_UNAVAILABLE';END IF;
  IF v_kind='STAFF' THEN
   SELECT id INTO v_singleton FROM public.discount_settings WHERE id=1 FOR UPDATE;
   IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P5009',MESSAGE='ORDER_STORAGE_UNAVAILABLE';END IF;
   PERFORM id FROM public.staff_whitelist ORDER BY id FOR UPDATE;
   FOR v_row IN SELECT value FROM jsonb_array_elements(COALESCE(NULLIF(v_plan->'rows','null'::jsonb),'[]'::jsonb)) LOOP
    IF v_row->>'outcome'='ERROR' THEN CONTINUE;END IF;
    v_material:=p_material->'rows'->v_index;v_index:=v_index+1;
    IF (v_material->>'row')::numeric IS DISTINCT FROM COALESCE((v_row->>'row')::numeric,0)
     OR decode(v_material->>'phone_hex','hex') IS DISTINCT FROM convert_to(COALESCE(v_row->>'phone',''),'UTF8') THEN
     RAISE EXCEPTION USING ERRCODE='P5009',MESSAGE='ORDER_STORAGE_UNAVAILABLE';END IF;
    v_name:=convert_from(decode(v_material->>'name_hex','hex'),'UTF8');v_key:=decode(v_material->>'name_key_hex','hex');v_stage:='BUSINESS';
    IF v_row->>'outcome'='ADD' THEN
     INSERT INTO public.staff_whitelist(phone,name,name_key,enabled,record_version,created_at,updated_at)
      VALUES(decode(v_material->>'phone_hex','hex'),v_name,v_key,true,1,clock_timestamp(),clock_timestamp()) RETURNING id INTO v_record;
    ELSE
     UPDATE public.staff_whitelist SET name=v_name,name_key=v_key,record_version=record_version+1,updated_at=clock_timestamp()
      WHERE phone=decode(v_material->>'phone_hex','hex') RETURNING id INTO v_record;
    END IF;
    IF v_record IS NULL THEN RAISE EXCEPTION USING ERRCODE='P5009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
    v_phone:=regexp_replace(v_row->>'phone','^[+]86','');
    v_rows:=v_rows||jsonb_build_array(jsonb_build_object('row',v_row->'row','id',v_record::text,'outcome',v_row->>'outcome','name',v_name,'phone_masked',CASE WHEN length(v_phone)>=7 THEN left(v_phone,3)||'****'||right(v_phone,4) ELSE '***' END));
    v_stage:='READ';
   END LOOP;
   UPDATE public.discount_settings SET whitelist_version=whitelist_version+1,updated_at=clock_timestamp() WHERE id=1;
  ELSE
   FOR v_category_row IN SELECT id,name,name_key FROM public.categories ORDER BY id FOR UPDATE LOOP
    IF v_category_row.id NOT BETWEEN 0 AND 18446744073709551615 THEN RAISE EXCEPTION USING ERRCODE='P5009',MESSAGE='ORDER_STORAGE_UNAVAILABLE';END IF;
    v_categories:=v_categories||jsonb_build_object(encode(v_category_row.name_key,'hex'),v_category_row.id::text);
   END LOOP;
   IF jsonb_array_length(p_material->'new_categories')<>jsonb_array_length(COALESCE(NULLIF(v_plan->'new_categories','null'::jsonb),'[]'::jsonb)) THEN
    RAISE EXCEPTION USING ERRCODE='P5009',MESSAGE='ORDER_STORAGE_UNAVAILABLE';END IF;
   v_n:=0;
   FOR v_row IN SELECT value FROM jsonb_array_elements(COALESCE(NULLIF(v_plan->'new_categories','null'::jsonb),'[]'::jsonb)) LOOP
    v_material:=p_material->'new_categories'->v_n;v_n:=v_n+1;
    v_name:=convert_from(decode(v_material->>'name_hex','hex'),'UTF8');v_key:=decode(v_material->>'name_key_hex','hex');
    IF v_name IS DISTINCT FROM (v_row#>>'{}') THEN RAISE EXCEPTION USING ERRCODE='P5009',MESSAGE='ORDER_STORAGE_UNAVAILABLE';END IF;
    v_stage:='BUSINESS';
    INSERT INTO public.categories(name,name_key,sort_order,is_active,record_version)
     -- Original pgx argument is string(catalogKey), so bytea's text input
     -- parser (including its backslash rules) is part of this late write.
     SELECT v_name,convert_from(v_key,'UTF8')::bytea,COALESCE(MAX(sort_order),0)+1,true,1 FROM public.categories RETURNING id INTO v_category;
    v_stage:='READ';v_categories:=v_categories||jsonb_build_object(encode(v_key,'hex'),v_category::text);
   END LOOP;
   PERFORM id FROM public.products ORDER BY id FOR UPDATE;
   FOR v_row IN SELECT value FROM jsonb_array_elements(COALESCE(NULLIF(v_plan->'rows','null'::jsonb),'[]'::jsonb)) LOOP
    IF v_row->>'outcome'='ERROR' THEN CONTINUE;END IF;
    v_material:=p_material->'rows'->v_index;v_index:=v_index+1;
    IF (v_material->>'row')::numeric IS DISTINCT FROM COALESCE((v_row->>'row')::numeric,0)
     OR decode(v_material->>'name_hex','hex') IS DISTINCT FROM convert_to(COALESCE(v_row->>'name',''),'UTF8') THEN
     RAISE EXCEPTION USING ERRCODE='P5009',MESSAGE='ORDER_STORAGE_UNAVAILABLE';END IF;
    v_category:=(v_categories->>(v_material->>'category_key_hex'))::numeric;
    IF COALESCE(v_category,0)=0 THEN RAISE EXCEPTION USING ERRCODE='P5009',MESSAGE='ORDER_STORAGE_UNAVAILABLE';END IF;
    v_stage:='BUSINESS';
    INSERT INTO public.products(category_id,name,name_key,description,specification,images_json,price_cents,sort_order,is_listed,meal_period,record_version)
     SELECT v_category,convert_from(decode(v_material->>'name_hex','hex'),'UTF8'),convert_from(decode(v_material->>'name_key_hex','hex'),'UTF8')::bytea,
      COALESCE(v_row->>'description',''),'',COALESCE(v_row->'images','[]'::jsonb),COALESCE((v_row->>'price_cents')::bigint,0),COALESCE(MAX(sort_order),0)+1,true,COALESCE(v_row->>'meal',''),1
      FROM public.products WHERE category_id=v_category RETURNING id INTO v_record;
    v_rows:=v_rows||jsonb_build_array(jsonb_build_object('row',v_row->'row','id',v_record::text,'outcome',v_row->>'outcome','name',v_row->>'name','category',v_row->>'category','meal',v_row->>'meal','price_cents',v_row->'price_cents','description',COALESCE(v_row->>'description',''),'images',COALESCE(v_row->'images','[]'::jsonb)));
    v_stage:='READ';
   END LOOP;
  END IF;
  v_result:=jsonb_build_object('Kind',v_kind,'Rows',v_rows,'BatchID',v_id,'NewCount',v_new,'UpdateCount',v_update,'SkippedCount',v_error,'Duplicate',false);
  v_stage:='BUSINESS';
  UPDATE public.import_batches SET preview_json=CASE WHEN v_kind='STAFF' THEN NULL ELSE v_plan END,state='COMMITTED',
   commit_idempotency_key_hash=decode(p_operation_hash,'hex'),skip_invalid=p_skip,result_json=v_result,
   committed_at=clock_timestamp(),record_version=record_version+1,updated_at=clock_timestamp() WHERE id=v_id AND state='PREVIEWED';
  GET DIAGNOSTICS v_count=ROW_COUNT;
  IF v_count<>1 THEN RAISE EXCEPTION USING ERRCODE='P5009',MESSAGE='ORDER_STORAGE_UNAVAILABLE';END IF;
  v_stage:='RECEIPT';
  INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,
   actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,target_type,target_id,target_key_hash,
   operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at)
   VALUES('COMMAND_RECEIPT','MERCHANT',v_scope,p_user,v_account,v_account,v_role,v_auth,'COMMIT_IMPORT','IMPORT_BATCH',v_id,NULL,
   decode(p_operation_hash,'hex'),decode(p_request_id_hash,'hex'),'SUCCEEDED','OK',jsonb_build_object('request_digest',p_request_digest),NULL,v_result,p_receipt_at);
  RETURN jsonb_build_object('outcome','SUCCEEDED','response_text',v_result::text);
 EXCEPTION
  WHEN unique_violation THEN
   GET STACKED DIAGNOSTICS v_constraint=CONSTRAINT_NAME;
   IF v_stage='RECEIPT' AND v_constraint='uq_action_audits_command_receipt' THEN
    SELECT before_state_json::text,response_json::text INTO v_before,v_response FROM public.action_audits
     WHERE entry_kind='COMMAND_RECEIPT' AND actor_scope_hash=v_scope AND action='COMMIT_IMPORT' AND operation_key_hash=decode(p_operation_hash,'hex') LIMIT 1;
    IF FOUND THEN RETURN jsonb_build_object('outcome','REPLAY','before_text',v_before,'response_text',v_response);END IF;
    RETURN jsonb_build_object('outcome','UNAVAILABLE');
   END IF;
   RETURN jsonb_build_object('outcome',CASE WHEN v_stage='BUSINESS' THEN 'INVALID_INPUT' ELSE 'UNAVAILABLE' END);
  WHEN foreign_key_violation OR check_violation THEN RETURN jsonb_build_object('outcome',CASE WHEN v_stage='BUSINESS' THEN 'INVALID_INPUT' ELSE 'UNAVAILABLE' END);
  WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_import_result_v1(p_user numeric,p_batch numeric)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_result jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 SELECT b.result_json INTO v_result FROM public.import_batches b JOIN public.merchant_accounts a ON a.id=b.actor_account_id
 WHERE b.id=p_batch AND b.state='COMMITTED' AND a.bound_user_id=p_user AND a.enabled AND a.deleted_at IS NULL AND a.role='OWNER';
 IF NOT FOUND OR jsonb_typeof(v_result->'Rows') IS DISTINCT FROM 'array' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 RETURN jsonb_build_object('outcome','SUCCEEDED','response_text',v_result::text);
END
$order_rpc$;
