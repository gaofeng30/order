CREATE OR REPLACE FUNCTION public.order_image_confirmed_v1(p_target_key_hashes text[])
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_hashes jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 IF p_target_key_hashes IS NULL OR cardinality(p_target_key_hashes) NOT BETWEEN 1 AND 3 OR EXISTS(SELECT 1 FROM unnest(p_target_key_hashes) h WHERE h IS NULL OR h !~ '^[0-9a-f]{64}$') THEN
  RAISE EXCEPTION USING ERRCODE='P1009',MESSAGE='ORDER_STORAGE_UNAVAILABLE';
 END IF;
 SELECT COALESCE(jsonb_agg(encode(target_key_hash,'hex') ORDER BY target_key_hash),'[]'::jsonb) INTO v_hashes
 FROM (SELECT DISTINCT target_key_hash FROM public.action_audits
 WHERE entry_kind='COMMAND_RECEIPT' AND action='object.image.confirm' AND target_type='IMAGE_OBJECT'
 AND target_key_hash IN(SELECT decode(h,'hex') FROM unnest(p_target_key_hashes) h) LIMIT 3) confirmed;
 RETURN jsonb_build_object('outcome','SUCCEEDED','target_key_hashes',v_hashes);
END
$order_rpc$;
