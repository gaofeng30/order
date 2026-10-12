CREATE OR REPLACE FUNCTION public.order_identity_projection_v1(p_user numeric)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
BEGIN
  IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
    RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
  END IF;
  IF p_user IS NULL OR p_user<=0 OR p_user<>trunc(p_user) OR p_user>18446744073709551615 THEN
    RAISE EXCEPTION USING ERRCODE='P1001',MESSAGE='ORDER_INPUT';
  END IF;
  RETURN (SELECT jsonb_build_object(
    'phone_hex',encode(u.primary_phone,'hex'),'phone_bound_at',u.primary_phone_bound_at,
    'extra_phone_hex',encode(u.extra_phone,'hex'),'extra_name',u.extra_name,
    'extra_name_key_hex',encode(u.extra_name_key,'hex'),'extra_phone_set_at',u.extra_phone_set_at,
    'record_version',u.record_version::text,'account_id',a.id::text,'account_role',a.role,
    'auth_version',a.auth_version::text,'rate_percent',d.rate_percent::text,
    'whitelist_version',d.whitelist_version::text,'primary_enabled',primary_staff.enabled,
    'extra_enabled',extra_staff.enabled,'whitelist_extra_name_key_hex',encode(extra_staff.name_key,'hex'))
    FROM public.miniprogram_users u JOIN public.discount_settings d ON d.id=1
    LEFT JOIN public.merchant_accounts a ON a.bound_user_id=u.id AND a.enabled=TRUE AND a.deleted_at IS NULL
    LEFT JOIN public.staff_whitelist primary_staff ON primary_staff.phone=u.primary_phone
    LEFT JOIN public.staff_whitelist extra_staff ON extra_staff.phone=u.extra_phone
    WHERE u.id=p_user);
END;
$order_rpc$;
