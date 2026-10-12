-- Fixed catalog RPCs; ordinary JWT service UID only.
CREATE OR REPLACE FUNCTION public.order_catalog_browse_v1()
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_categories jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 SELECT COALESCE(jsonb_agg(category ORDER BY sort_order,id),'[]'::jsonb) INTO v_categories FROM (
  SELECT c.id,c.sort_order,jsonb_build_object('id',c.id::text,'name',c.name,'products',
   COALESCE((SELECT jsonb_agg(jsonb_build_object('id',p.id::text,'category_id',p.category_id::text,'name',p.name,'description',p.description,'specification',p.specification,'meal_period',p.meal_period,'images',p.images_json,'listed',p.is_listed,'sold_out',false,'price_cents',p.price_cents) ORDER BY p.sort_order,p.id) FROM public.products p WHERE p.category_id=c.id AND p.is_listed),'[]'::jsonb)) category
  FROM public.categories c WHERE c.is_active
 ) rows;
 RETURN jsonb_build_object('outcome','SUCCEEDED','categories',v_categories);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_catalog_detail_v1(p_product numeric,p_service_date date)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_status text;v_date text;v_open boolean;v_periods jsonb;v_product jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 SELECT s.business_status,to_char(d.service_date,'YYYY-MM-DD'),d.is_open,
  (SELECT COALESCE(jsonb_agg(jsonb_build_object('Code',code,'CutoffTime',cutoff_time::text,'PickupStartTime',pickup_start_time::text,'PickupEndTime',pickup_end_time::text,'IntervalMinutes',interval_minutes) ORDER BY code),'[]'::jsonb) FROM public.meal_periods),
  (SELECT jsonb_build_object('id',p.id::text,'category_id',p.category_id::text,'name',p.name,'description',p.description,'specification',p.specification,'meal_period',p.meal_period,'images',p.images_json,'listed',p.is_listed,'sold_out',sold.product_id IS NOT NULL,'price_cents',p.price_cents)
   FROM public.products p INNER JOIN public.categories c ON c.id=p.category_id AND c.is_active
   LEFT JOIN public.product_sold_out_dates sold ON sold.product_id=p.id AND sold.service_date=p_service_date
   WHERE p.id=p_product AND p.is_listed LIMIT 1)
 INTO v_status,v_date,v_open,v_periods,v_product
 FROM public.storefront_settings s LEFT JOIN public.service_dates d ON d.service_date=p_service_date WHERE s.id=1;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P1009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
 IF v_product IS NULL THEN RETURN jsonb_build_object('outcome','NOT_FOUND'); END IF;
 RETURN jsonb_build_object('outcome','SUCCEEDED','product',v_product,'business_status',v_status,'service_date',v_date,'is_open',v_open,'meal_periods',v_periods);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_catalog_categories_v1()
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_result jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 SELECT COALESCE(jsonb_agg(value ORDER BY sort_order,id),'[]'::jsonb) INTO v_result FROM(SELECT c.id,c.sort_order,jsonb_build_object('id',c.id::text,'name',c.name,'sort_order',c.sort_order,'enabled',c.is_active,'product_count',COUNT(p.id)) value FROM public.categories c LEFT JOIN public.products p ON p.category_id=c.id GROUP BY c.id,c.name,c.sort_order,c.is_active) rows;
 RETURN jsonb_build_object('outcome','SUCCEEDED','categories',v_result);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_catalog_products_v1(p_service_date date)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_result jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 SELECT COALESCE(jsonb_agg(jsonb_build_object('id',p.id::text,'category_id',p.category_id::text,'category_name',c.name,'name',p.name,'description',p.description,'meal_period',p.meal_period,'images',p.images_json,'price_cents',p.price_cents,'listed',p.is_listed,'sold_out',s.product_id IS NOT NULL) ORDER BY p.category_id,p.sort_order,p.id),'[]'::jsonb) INTO v_result FROM public.products p JOIN public.categories c ON c.id=p.category_id LEFT JOIN public.product_sold_out_dates s ON s.product_id=p.id AND s.service_date=p_service_date;
 RETURN jsonb_build_object('outcome','SUCCEEDED','products',v_result);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_catalog_product_v1(p_product numeric,p_service_date date)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_result jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 SELECT jsonb_build_object('id',p.id::text,'category_id',p.category_id::text,'category_name',c.name,'name',p.name,'description',p.description,'meal_period',p.meal_period,'images',p.images_json,'price_cents',p.price_cents,'listed',p.is_listed,'sold_out',s.product_id IS NOT NULL) INTO v_result FROM public.products p JOIN public.categories c ON c.id=p.category_id LEFT JOIN public.product_sold_out_dates s ON s.product_id=p.id AND s.service_date=p_service_date WHERE p.id=p_product;
 RETURN jsonb_build_object('outcome','SUCCEEDED','found',EXISTS(SELECT 1 FROM public.products WHERE id=p_product),'category_id',(SELECT category_id::text FROM public.products WHERE id=p_product),'product',v_result);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_catalog_command_v1(p_user numeric,p_kind text,p_category numeric,p_product numeric,p_name_present boolean,p_name_hex text,p_description_hex text,p_price_cents bigint,p_meal_period_hex text,p_images jsonb,p_enabled boolean,p_listed boolean,p_service_date_hex text,p_sold_out boolean,p_ordered_ids numeric[],p_confirm_status text,p_operation_hash text,p_request_id_hash text,p_request_digest text,p_receipt_at timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE
 v_account numeric;v_role text;v_auth numeric;v_scope bytea;v_before text;v_response text;
 v_stage text:='BUSINESS';v_constraint text;v_name text;v_name_bytes bytea;v_description text;v_meal text;
 v_id numeric;v_sort bigint;v_enabled boolean;v_category_name text;v_count bigint;
 v_existing numeric[];v_item jsonb;v_key text;v_keys text[];v_stored jsonb;
 v_result jsonb:=jsonb_build_object('Category',NULL,'Product',NULL);
 v_images_response jsonb;v_target_type text;v_target numeric;v_index integer;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 BEGIN
  SELECT id,role,auth_version INTO v_account,v_role,v_auth FROM public.merchant_accounts WHERE bound_user_id=p_user AND enabled AND deleted_at IS NULL AND role='OWNER' FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P2009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
  v_scope:=sha256(convert_to('merchant:'||p_user::text||':'||v_account::text,'UTF8'));
  SELECT before_state_json::text,response_json::text INTO v_before,v_response FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_scope_hash=v_scope AND action=p_kind AND operation_key_hash=decode(p_operation_hash,'hex') LIMIT 1;
  IF FOUND THEN RETURN jsonb_build_object('outcome','REPLAY','before_text',v_before,'response_text',v_response); END IF;
  IF p_kind IN('CREATE_PRODUCT','UPDATE_PRODUCT') THEN
   IF p_confirm_status='UNAVAILABLE' THEN RAISE EXCEPTION USING ERRCODE='P2009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
   IF p_confirm_status='NOT_CONFIRMED' THEN RAISE EXCEPTION USING ERRCODE='P2001',MESSAGE='ORDER_INVALID_INPUT'; END IF;
   IF p_confirm_status IS DISTINCT FROM 'CONFIRMED' THEN RAISE EXCEPTION USING ERRCODE='P2009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
  END IF;
  CASE p_kind
  WHEN 'CREATE_CATEGORY' THEN
   v_name_bytes:=decode(p_name_hex,'hex');
   IF v_name_bytes IS NULL OR octet_length(v_name_bytes) NOT BETWEEN 1 AND 400 THEN RAISE EXCEPTION USING ERRCODE='P2001',MESSAGE='ORDER_INVALID_INPUT'; END IF;
   SELECT sort_order INTO v_sort FROM public.categories ORDER BY sort_order DESC,id DESC LIMIT 1 FOR UPDATE;
   v_sort:=CASE WHEN FOUND THEN (v_sort+1)%4294967296 ELSE 1 END;
   IF p_receipt_at IS NULL THEN RETURN jsonb_build_object('outcome','NEED_MATERIAL'); END IF;
   v_name:=convert_from(v_name_bytes,'UTF8');
   INSERT INTO public.categories(name,name_key,sort_order,is_active,record_version) VALUES(v_name,v_name_bytes,v_sort,true,1) RETURNING id INTO v_id;
   v_result:=jsonb_build_object('Category',jsonb_build_object('ID',v_id,'Name',v_name,'SortOrder',v_sort,'Enabled',true,'ProductCount',0),'Product',NULL);
  WHEN 'UPDATE_CATEGORY' THEN
   SELECT id,name,sort_order,is_active INTO v_id,v_name,v_sort,v_enabled FROM public.categories WHERE id=p_category FOR UPDATE;
   IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P2002',MESSAGE='ORDER_NOT_FOUND'; END IF;
   IF p_name_present THEN
    v_name_bytes:=decode(p_name_hex,'hex');
    IF v_name_bytes IS NULL OR octet_length(v_name_bytes) NOT BETWEEN 1 AND 400 THEN RAISE EXCEPTION USING ERRCODE='P2001',MESSAGE='ORDER_INVALID_INPUT'; END IF;
   END IF;
   v_enabled:=COALESCE(p_enabled,v_enabled);
   IF p_receipt_at IS NULL THEN RETURN jsonb_build_object('outcome','NEED_MATERIAL'); END IF;
   IF p_name_present THEN
    v_name:=convert_from(v_name_bytes,'UTF8');
    UPDATE public.categories SET name=v_name,name_key=v_name_bytes,is_active=v_enabled,record_version=record_version+1 WHERE id=p_category;
   ELSE UPDATE public.categories SET is_active=v_enabled,record_version=record_version+1 WHERE id=p_category; END IF;
   v_result:=jsonb_build_object('Category',jsonb_build_object('ID',v_id,'Name',v_name,'SortOrder',v_sort,'Enabled',v_enabled,'ProductCount',0),'Product',NULL);
  WHEN 'DELETE_CATEGORY' THEN
   SELECT id INTO v_id FROM public.categories WHERE id=p_category FOR UPDATE;
   IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P2002',MESSAGE='ORDER_NOT_FOUND'; END IF;
   PERFORM id FROM public.products WHERE category_id=p_category ORDER BY id LIMIT 1 FOR UPDATE;
   IF FOUND THEN RAISE EXCEPTION USING ERRCODE='P2003',MESSAGE='ORDER_CONFLICT'; END IF;
   IF p_receipt_at IS NULL THEN RETURN jsonb_build_object('outcome','NEED_MATERIAL'); END IF;
   DELETE FROM public.categories WHERE id=v_id;
   GET DIAGNOSTICS v_count=ROW_COUNT;
   IF v_count<>1 THEN RAISE EXCEPTION USING ERRCODE='P2002',MESSAGE='ORDER_NOT_FOUND'; END IF;
  WHEN 'REORDER_CATEGORIES','REORDER_PRODUCTS' THEN
   IF COALESCE(cardinality(p_ordered_ids),0)=0 THEN RAISE EXCEPTION USING ERRCODE='P2001',MESSAGE='ORDER_INVALID_INPUT'; END IF;
   IF p_kind='REORDER_CATEGORIES' THEN
    SELECT COALESCE(array_agg(id ORDER BY id),ARRAY[]::numeric[]) INTO v_existing FROM(SELECT id FROM public.categories ORDER BY id FOR UPDATE) locked;
   ELSE
    SELECT COALESCE(array_agg(id ORDER BY id),ARRAY[]::numeric[]) INTO v_existing FROM(SELECT id FROM public.products WHERE category_id=p_category ORDER BY id FOR UPDATE) locked;
   END IF;
   IF cardinality(v_existing)<>cardinality(p_ordered_ids) OR EXISTS(SELECT 1 FROM unnest(p_ordered_ids) x WHERE NOT(x=ANY(v_existing))) THEN RAISE EXCEPTION USING ERRCODE='P2003',MESSAGE='ORDER_CONFLICT'; END IF;
   IF p_receipt_at IS NULL THEN RETURN jsonb_build_object('outcome','NEED_MATERIAL'); END IF;
   FOR v_index IN 1..cardinality(p_ordered_ids) LOOP
    IF p_kind='REORDER_CATEGORIES' THEN UPDATE public.categories SET sort_order=v_index,record_version=record_version+1 WHERE id=p_ordered_ids[v_index];
    ELSE UPDATE public.products SET sort_order=v_index,record_version=record_version+1 WHERE id=p_ordered_ids[v_index]; END IF;
   END LOOP;
  WHEN 'CREATE_PRODUCT','UPDATE_PRODUCT' THEN
   v_name_bytes:=decode(p_name_hex,'hex');
   IF v_name_bytes IS NULL OR octet_length(v_name_bytes) NOT BETWEEN 1 AND 400 OR p_price_cents IS NULL OR p_price_cents=0 OR p_category=0 OR (p_images IS NOT NULL AND (jsonb_typeof(p_images)<>'array' OR jsonb_array_length(p_images)>3)) THEN RAISE EXCEPTION USING ERRCODE='P2001',MESSAGE='ORDER_INVALID_INPUT'; END IF;
   SELECT name INTO v_category_name FROM public.categories WHERE id=p_category FOR UPDATE;
   IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P2003',MESSAGE='ORDER_CONFLICT'; END IF;
   v_keys:=ARRAY[]::text[];
   FOR v_item IN SELECT value FROM jsonb_array_elements(COALESCE(p_images,'[]'::jsonb)) LOOP
    IF jsonb_typeof(v_item)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(v_item))<>4 OR NOT(v_item ?& ARRAY['object_key_hex','object_key_json_hex','url_json_hex','sort_order']) OR jsonb_typeof(v_item->'object_key_hex')<>'string' THEN RAISE EXCEPTION USING ERRCODE='P2009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
    v_key:=v_item->>'object_key_hex';
    IF octet_length(decode(v_key,'hex')) NOT BETWEEN 1 AND 1024 OR v_key=ANY(v_keys) THEN RAISE EXCEPTION USING ERRCODE='P2001',MESSAGE='ORDER_INVALID_INPUT'; END IF;
    v_keys:=array_append(v_keys,v_key);
   END LOOP;
   IF p_kind='CREATE_PRODUCT' THEN
    SELECT sort_order INTO v_sort FROM public.products WHERE category_id=p_category ORDER BY sort_order DESC,id DESC LIMIT 1 FOR UPDATE;
    v_sort:=CASE WHEN FOUND THEN (v_sort+1)%4294967296 ELSE 1 END;
   END IF;
   IF p_receipt_at IS NULL THEN RETURN jsonb_build_object('outcome','NEED_MATERIAL'); END IF;
   v_name:=convert_from(v_name_bytes,'UTF8');
   v_description:=convert_from(decode(p_description_hex,'hex'),'UTF8');
   v_meal:=convert_from(decode(p_meal_period_hex,'hex'),'UTF8');
   SELECT COALESCE(jsonb_agg(jsonb_build_object('object_key',convert_from(decode(value->>'object_key_json_hex','hex'),'UTF8')::jsonb) ORDER BY ordinal),'[]'::jsonb) INTO v_stored FROM jsonb_array_elements(COALESCE(p_images,'[]'::jsonb)) WITH ORDINALITY AS images(value,ordinal);
   IF p_kind='CREATE_PRODUCT' THEN
    INSERT INTO public.products(category_id,name,name_key,description,specification,images_json,price_cents,sort_order,is_listed,meal_period,record_version) VALUES(p_category,v_name,v_name_bytes,v_description,'',v_stored,p_price_cents,v_sort,true,v_meal,1) RETURNING id INTO v_id;
   ELSE
    v_id:=p_product;
    UPDATE public.products SET category_id=p_category,name=v_name,name_key=v_name_bytes,description=v_description,images_json=v_stored,price_cents=p_price_cents,meal_period=v_meal,record_version=record_version+1 WHERE id=p_product;
    GET DIAGNOSTICS v_count=ROW_COUNT;
    IF v_count<>1 THEN RAISE EXCEPTION USING ERRCODE='P2002',MESSAGE='ORDER_NOT_FOUND'; END IF;
   END IF;
   v_stage:='RECEIPT';
   SELECT CASE WHEN p_images IS NULL THEN NULL ELSE COALESCE(jsonb_agg(jsonb_build_object('ObjectKey',convert_from(decode(value->>'object_key_json_hex','hex'),'UTF8')::jsonb,'URL',convert_from(decode(value->>'url_json_hex','hex'),'UTF8')::jsonb,'SortOrder',value->'sort_order') ORDER BY ordinal),'[]'::jsonb) END INTO v_images_response FROM jsonb_array_elements(COALESCE(p_images,'[]'::jsonb)) WITH ORDINALITY AS images(value,ordinal);
   v_result:=jsonb_build_object('Category',NULL,'Product',jsonb_build_object('ID',v_id,'CategoryID',p_category,'Name',v_name,'Description',convert_from(decode(p_description_hex,'hex'),'UTF8'),'CategoryName',v_category_name,'MealPeriod',convert_from(decode(p_meal_period_hex,'hex'),'UTF8'),'Images',v_images_response,'PriceCents',p_price_cents,'Listed',true,'SoldOut',false));
  WHEN 'DELETE_PRODUCT' THEN
   PERFORM id FROM public.products WHERE id=p_product FOR UPDATE;
   IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P2002',MESSAGE='ORDER_NOT_FOUND'; END IF;
   IF p_receipt_at IS NULL THEN RETURN jsonb_build_object('outcome','NEED_MATERIAL'); END IF;
   DELETE FROM public.products WHERE id=p_product;
   GET DIAGNOSTICS v_count=ROW_COUNT;
   IF v_count<>1 THEN RAISE EXCEPTION USING ERRCODE='P2002',MESSAGE='ORDER_NOT_FOUND'; END IF;
  WHEN 'SET_PRODUCT_STATUS' THEN
   IF p_listed IS NULL THEN RAISE EXCEPTION USING ERRCODE='P2001',MESSAGE='ORDER_INVALID_INPUT'; END IF;
   PERFORM id FROM public.products WHERE id=p_product FOR UPDATE;
   IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P2002',MESSAGE='ORDER_NOT_FOUND'; END IF;
   IF p_receipt_at IS NULL THEN RETURN jsonb_build_object('outcome','NEED_MATERIAL'); END IF;
   UPDATE public.products SET is_listed=p_listed,record_version=record_version+1 WHERE id=p_product;
   GET DIAGNOSTICS v_count=ROW_COUNT;
   IF v_count<>1 THEN RAISE EXCEPTION USING ERRCODE='P2002',MESSAGE='ORDER_NOT_FOUND'; END IF;
  WHEN 'SET_PRODUCT_SOLD_OUT' THEN
   IF p_sold_out IS NULL OR p_service_date_hex IS NULL OR p_service_date_hex='' THEN RAISE EXCEPTION USING ERRCODE='P2001',MESSAGE='ORDER_INVALID_INPUT'; END IF;
   IF p_receipt_at IS NULL THEN RETURN jsonb_build_object('outcome','NEED_MATERIAL'); END IF;
   IF p_sold_out THEN INSERT INTO public.product_sold_out_dates(service_date,product_id) VALUES(convert_from(decode(p_service_date_hex,'hex'),'UTF8')::date,p_product) ON CONFLICT(service_date,product_id) DO NOTHING;
   ELSE DELETE FROM public.product_sold_out_dates WHERE service_date=convert_from(decode(p_service_date_hex,'hex'),'UTF8')::date AND product_id=p_product; END IF;
  ELSE RAISE EXCEPTION USING ERRCODE='P2001',MESSAGE='ORDER_INVALID_INPUT';
  END CASE;
  v_stage:='RECEIPT';
  IF p_request_digest IS NULL THEN RAISE EXCEPTION USING ERRCODE='P2009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
  IF p_product>0 THEN v_target_type:='PRODUCT';v_target:=p_product;ELSIF p_category>0 THEN v_target_type:='CATEGORY';v_target:=p_category;END IF;
  INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,target_type,target_id,target_key_hash,operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at)
  VALUES('COMMAND_RECEIPT','MERCHANT',v_scope,p_user,v_account,v_account,v_role,v_auth,p_kind,v_target_type,v_target,NULL,decode(p_operation_hash,'hex'),decode(p_request_id_hash,'hex'),'SUCCEEDED','OK',jsonb_build_object('request_digest',p_request_digest),NULL,v_result,p_receipt_at);
  RETURN jsonb_build_object('outcome','SUCCEEDED','response_text',v_result::text);
 EXCEPTION
  WHEN SQLSTATE 'P2001' THEN RETURN jsonb_build_object('outcome','INVALID_INPUT');
  WHEN SQLSTATE 'P2002' THEN RETURN jsonb_build_object('outcome','NOT_FOUND');
  WHEN SQLSTATE 'P2003' THEN RETURN jsonb_build_object('outcome','CONFLICT');
  WHEN unique_violation THEN
   GET STACKED DIAGNOSTICS v_constraint=CONSTRAINT_NAME;
   IF v_constraint='uq_action_audits_command_receipt' THEN
    SELECT before_state_json::text,response_json::text INTO v_before,v_response FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_scope_hash=v_scope AND action=p_kind AND operation_key_hash=decode(p_operation_hash,'hex') LIMIT 1;
    IF FOUND THEN RETURN jsonb_build_object('outcome','REPLAY','before_text',v_before,'response_text',v_response); END IF;
    RETURN jsonb_build_object('outcome','UNAVAILABLE');
   END IF;
   RETURN jsonb_build_object('outcome',CASE WHEN v_stage='BUSINESS' THEN 'CONFLICT' ELSE 'UNAVAILABLE' END);
  WHEN foreign_key_violation OR check_violation THEN RETURN jsonb_build_object('outcome',CASE WHEN v_stage='BUSINESS' THEN 'CONFLICT' ELSE 'UNAVAILABLE' END);
  WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END;
END
$order_rpc$;
