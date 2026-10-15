-- Fixed menu snapshot RPCs; one statement per snapshot.
CREATE OR REPLACE FUNCTION public.order_menu_snapshot_v1(p_service_date date)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_result jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 SELECT jsonb_build_object('outcome','SUCCEEDED','business_status',s.business_status,'service_date',to_char(d.service_date,'YYYY-MM-DD'),'is_open',d.is_open,'meal_periods',(SELECT COALESCE(jsonb_agg(jsonb_build_object('Code',code,'CutoffTime',cutoff_time::text,'PickupStartTime',pickup_start_time::text,'PickupEndTime',pickup_end_time::text,'IntervalMinutes',interval_minutes) ORDER BY code),'[]'::jsonb) FROM public.meal_periods),'categories',(SELECT COALESCE(jsonb_agg(category ORDER BY sort_order,id),'[]'::jsonb) FROM (
 SELECT c.id,c.sort_order,jsonb_build_object('id',c.id::text,'name',c.name,'products',COALESCE((SELECT jsonb_agg(jsonb_build_object('id',p.id::text,'category_id',p.category_id::text,'name',p.name,'description',p.description,'specification',p.specification,'meal_period',p.meal_period,'images',p.images_json,'listed',p.is_listed,'sold_out',sold.product_id IS NOT NULL,'price_cents',p.price_cents) ORDER BY p.sort_order,p.id) FROM public.products p LEFT JOIN public.product_sold_out_dates sold ON sold.product_id=p.id AND sold.service_date=p_service_date WHERE p.category_id=c.id AND p.is_listed),'[]'::jsonb)) category FROM public.categories c WHERE c.is_active AND EXISTS(SELECT 1 FROM public.products p WHERE p.category_id=c.id AND p.is_listed)) rows)) INTO v_result
 FROM public.storefront_settings s LEFT JOIN public.service_dates d ON d.service_date=p_service_date WHERE s.id=1;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P1009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
 RETURN v_result;
END
$order_rpc$;
CREATE OR REPLACE FUNCTION public.order_menu_pickup_facts_v1(p_date_a date,p_date_b date)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_result jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 IF p_date_a IS NULL OR p_date_b IS NULL OR p_date_a=p_date_b THEN RAISE EXCEPTION USING ERRCODE='P1009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
 SELECT jsonb_build_object('outcome','SUCCEEDED','business_status',s.business_status,'meal_periods',(SELECT COALESCE(jsonb_agg(jsonb_build_object('Code',code,'CutoffTime',cutoff_time::text,'PickupStartTime',pickup_start_time::text,'PickupEndTime',pickup_end_time::text,'IntervalMinutes',interval_minutes) ORDER BY code),'[]'::jsonb) FROM public.meal_periods),'service_dates',
  (SELECT COALESCE(jsonb_agg(jsonb_build_object('date',to_char(service_date,'YYYY-MM-DD'),'is_open',is_open,'available_meals',(SELECT COALESCE(jsonb_agg(m.code ORDER BY m.code),'[]'::jsonb) FROM public.meal_periods m WHERE EXISTS (SELECT 1 FROM public.products p JOIN public.categories c ON c.id=p.category_id AND c.is_active WHERE p.is_listed AND p.meal_period IN ('all',m.code) AND NOT EXISTS (SELECT 1 FROM public.product_sold_out_dates sold WHERE sold.product_id=p.id AND sold.service_date=d.service_date)))) ORDER BY service_date),'[]'::jsonb) FROM public.service_dates d WHERE service_date IN(p_date_a,p_date_b))) INTO v_result FROM public.storefront_settings s WHERE s.id=1;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P1009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
 RETURN v_result;
END
$order_rpc$;
