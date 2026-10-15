BEGIN ISOLATION LEVEL REPEATABLE READ;
DO $order_bootstrap$
DECLARE
 owner_phone bytea:=decode('__ORDER_OWNER_PHONE_HEX__','hex');
 owner_name bytea:=decode('__ORDER_OWNER_NAME_HEX__','hex');
 store_name bytea:=decode('__ORDER_STORE_NAME_HEX__','hex');
 store_address bytea:=decode('__ORDER_STORE_ADDRESS_HEX__','hex');
 pickup_point bytea:=decode('__ORDER_PICKUP_POINT_HEX__','hex');
 s public.storefront_settings%ROWTYPE;d public.discount_settings%ROWTYPE;a public.merchant_accounts%ROWTYPE;
 has_store boolean;has_discount boolean;owner_count bigint:=0;owner_matches boolean:=false;store_matches boolean:=false;discount_matches boolean:=false;
 flavors jsonb:=jsonb_build_array('少饭','加饭','少盐','加辣','酱汁分装','免葱蒜','打包分装','多双餐具');
BEGIN
 IF current_setting('transaction_isolation')<>'repeatable read' OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_BOOTSTRAP_UNAVAILABLE';END IF;
 SELECT * INTO s FROM public.storefront_settings WHERE id=1 FOR UPDATE;has_store:=FOUND;
 IF has_store THEN
  -- Native reads jsonb_array_length before its final exact comparison.
  PERFORM jsonb_array_length(s.flavor_options_json);
  store_matches:=convert_to(s.store_name,'UTF8')=store_name AND convert_to(s.store_address,'UTF8')=store_address AND convert_to(s.pickup_point,'UTF8')=pickup_point
   AND s.announcement='' AND s.business_status='closed' AND s.launch_image_object_key IS NULL AND s.center_x IS NULL AND s.center_y IS NULL AND s.width_ratio IS NULL AND s.aspect_ratio IS NULL
   AND s.flavor_options_json=flavors AND s.record_version=1;
 END IF;
 SELECT * INTO d FROM public.discount_settings WHERE id=1 FOR UPDATE;has_discount:=FOUND;
 IF has_discount THEN discount_matches:=d.rate_percent=100 AND d.discount_version=1 AND d.whitelist_version=1;END IF;
 FOR a IN SELECT * FROM public.merchant_accounts WHERE role='OWNER' ORDER BY id FOR UPDATE LOOP
  PERFORM convert_from(a.phone,'UTF8');owner_count:=owner_count+1;
  owner_matches:=a.phone=owner_phone AND convert_to(a.name,'UTF8')=owner_name AND a.enabled AND a.record_version=1 AND a.auth_version=1
   AND a.bound_user_id IS NULL AND a.bound_at IS NULL AND a.created_by IS NULL AND a.updated_by IS NULL AND a.deleted_at IS NULL AND a.deleted_by_account_id IS NULL AND a.created_at=a.updated_at;
 END LOOP;
 IF has_store OR has_discount OR owner_count<>0 THEN
  IF has_store AND has_discount AND owner_count=1 AND store_matches AND discount_matches AND owner_matches THEN
   PERFORM set_config('order.bootstrap_outcome','unchanged',true);RETURN;
  END IF;
  RAISE EXCEPTION USING ERRCODE='P1002',MESSAGE='ORDER_BOOTSTRAP_CONFLICT';
 END IF;
 INSERT INTO public.storefront_settings(id,store_name,store_address,pickup_point,announcement,business_status,launch_image_object_key,center_x,center_y,width_ratio,aspect_ratio,flavor_options_json,record_version)
  VALUES(1,convert_from(store_name,'UTF8'),convert_from(store_address,'UTF8'),convert_from(pickup_point,'UTF8'),'','closed',NULL,NULL,NULL,NULL,NULL,flavors,1);
 INSERT INTO public.discount_settings(id,rate_percent,discount_version,whitelist_version,updated_at) VALUES(1,100,1,1,CURRENT_TIMESTAMP);
 INSERT INTO public.merchant_accounts(phone,name,role,enabled,record_version,auth_version,bound_user_id,bound_at,deleted_at,deleted_by_account_id,created_at,updated_at,created_by,updated_by)
  VALUES(owner_phone,convert_from(owner_name,'UTF8'),'OWNER',TRUE,1,1,NULL,NULL,NULL,NULL,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,NULL,NULL);
 PERFORM set_config('order.bootstrap_outcome','created',true);
END
$order_bootstrap$;
SELECT jsonb_build_object('outcome',current_setting('order.bootstrap_outcome'),'isolation',current_setting('transaction_isolation'),'read_only',current_setting('transaction_read_only')) AS order_bootstrap_result;
COMMIT;
