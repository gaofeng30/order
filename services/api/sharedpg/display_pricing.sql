-- One SQL statement gives the native repeatable-read fact set one snapshot.
CREATE OR REPLACE FUNCTION public.order_display_pricing_facts_v1(p_user numeric)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_result jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 SELECT jsonb_build_object('outcome','SUCCEEDED','user',jsonb_build_object('primary_phone',convert_from(u.primary_phone,'UTF8'),'extra_phone',convert_from(u.extra_phone,'UTF8'),'extra_name',u.extra_name,'extra_name_key_hex',encode(u.extra_name_key,'hex')),
 'settings',CASE WHEN u.primary_phone IS NULL OR d.id IS NULL THEN NULL ELSE jsonb_build_object('rate_percent',d.rate_percent,'whitelist_version',d.whitelist_version::text) END,
 'entries',CASE WHEN u.primary_phone IS NULL THEN '[]'::jsonb ELSE
  (SELECT COALESCE(jsonb_agg(jsonb_build_object('phone',convert_from(w.phone,'UTF8'),'name',w.name,'name_key_hex',encode(w.name_key,'hex'),'enabled',w.enabled) ORDER BY w.id),'[]'::jsonb) FROM public.staff_whitelist w WHERE w.phone=u.primary_phone OR (u.extra_phone IS NOT NULL AND u.extra_phone<>u.primary_phone AND w.phone=u.extra_phone)) END)
 INTO v_result FROM public.miniprogram_users u LEFT JOIN public.discount_settings d ON d.id=1 WHERE u.id=p_user;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P1009',MESSAGE='ORDER_STORAGE_UNAVAILABLE'; END IF;
 RETURN v_result;
END
$order_rpc$;
