-- Fixed redacted audit search; every owner query/Scan failure is Forbidden.
CREATE OR REPLACE FUNCTION public.order_audit_search_v1(p_owner numeric,p_action_hex text,p_target_hex text,p_from_hex text,p_to_hex text,p_after numeric,p_limit integer)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE owner numeric;scan_row record;rows jsonb:='[]';v_action text;v_target text;v_from timestamptz;v_to timestamptz;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF p_owner IS NULL OR p_owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_owner)<>p_owner THEN RETURN jsonb_build_object('outcome','FORBIDDEN');END IF;
  SELECT id INTO owner FROM public.merchant_accounts WHERE bound_user_id=p_owner AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER';
  IF NOT FOUND THEN RETURN jsonb_build_object('outcome','FORBIDDEN');END IF;
  IF owner IS NULL OR owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(owner)<>owner THEN RETURN jsonb_build_object('outcome','FORBIDDEN');END IF;
 EXCEPTION WHEN query_canceled THEN RETURN jsonb_build_object('outcome','FORBIDDEN');
 WHEN OTHERS THEN RETURN jsonb_build_object('outcome','FORBIDDEN');END;
 BEGIN
 IF p_action_hex IS NULL OR p_action_hex !~ '^([0-9a-f]{2})*$' OR p_target_hex IS NULL OR p_target_hex !~ '^([0-9a-f]{2})*$' OR p_from_hex IS NULL OR p_from_hex !~ '^([0-9a-f]{2})*$' OR p_to_hex IS NULL OR p_to_hex !~ '^([0-9a-f]{2})*$' OR p_after IS NULL OR p_after NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_after)<>p_after OR p_limit IS NULL OR p_limit NOT BETWEEN 0 AND 65535 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 v_action:=convert_from(decode(p_action_hex,'hex'),'UTF8');v_target:=convert_from(decode(p_target_hex,'hex'),'UTF8');
 IF p_from_hex<>'' THEN v_from:=convert_from(decode(p_from_hex,'hex'),'UTF8')::timestamptz;END IF;
 IF p_to_hex<>'' THEN v_to:=(convert_from(decode(p_to_hex,'hex'),'UTF8')||' 23:59:59.999999')::timestamptz;END IF;
 FOR scan_row IN SELECT id AS c0,action AS c1,COALESCE(target_type,'') AS c2,COALESCE(target_id::text,'') AS c3,result AS c4,reason_code AS c5,COALESCE(encode(request_id_hash,'hex'),'') AS c6,COALESCE(actor_account_id,0) AS c7,occurred_at AS c8 FROM public.action_audits WHERE id>p_after AND (p_action_hex='' OR action=v_action) AND (p_target_hex='' OR target_type=v_target) AND (p_from_hex='' OR occurred_at>=v_from) AND (p_to_hex='' OR occurred_at<v_to) ORDER BY id LIMIT p_limit LOOP
  IF scan_row.c0 IS NULL OR scan_row.c0 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c0)<>scan_row.c0 OR scan_row.c1 IS NULL OR scan_row.c2 IS NULL OR scan_row.c3 IS NULL OR scan_row.c4 IS NULL OR scan_row.c5 IS NULL OR scan_row.c6 IS NULL OR scan_row.c7 IS NULL OR scan_row.c7 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c7)<>scan_row.c7 OR scan_row.c8 IS NULL OR NOT isfinite(scan_row.c8) THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  rows:=rows||jsonb_build_array(jsonb_build_array(scan_row.c0::text,scan_row.c1,scan_row.c2,scan_row.c3,scan_row.c4,scan_row.c5,scan_row.c6,scan_row.c7::text,jsonb_build_object('seconds',floor(extract(epoch FROM scan_row.c8))::text,'nanosecond',((extract(epoch FROM scan_row.c8)-floor(extract(epoch FROM scan_row.c8)))*1000000000)::integer)));
 END LOOP;
 RETURN jsonb_build_object('outcome','FOUND','rows',rows);
 EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END;
END
$order_rpc$;
