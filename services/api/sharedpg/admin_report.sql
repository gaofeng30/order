-- Fixed report functions retain separate VOLATILE statement snapshots and native Scan stages.
CREATE OR REPLACE FUNCTION public.order_admin_report_get_v1(p_owner numeric,p_order numeric)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE owner numeric;scan_row record;rows jsonb:='[]';items jsonb:='[]';ids numeric[]:=ARRAY[]::numeric[];v_state text;v_date date;v_query text;v_like text;v_from date;v_to date;micros numeric;v_at timestamptz;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF p_owner IS NULL OR p_owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_owner)<>p_owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  SELECT id INTO owner FROM public.merchant_accounts WHERE bound_user_id=p_owner AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER';
  IF NOT FOUND THEN RETURN jsonb_build_object('outcome','FORBIDDEN');END IF;
  IF owner IS NULL OR owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(owner)<>owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF p_order IS NULL OR p_order NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_order)<>p_order THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 FOR scan_row IN SELECT id AS c0,convert_from(order_no,'UTF8') AS c1,(CASE WHEN state='RESERVED' AND pickup_at<=statement_timestamp()+interval '30 minutes' THEN 'PREPARING' ELSE state END) AS c2,to_char(pickup_date,'YYYY-MM-DD') AS c3,to_char(pickup_date+pickup_time,'HH24:MI') AS c4,meal_period AS c5,pickup_point_snapshot AS c6,LPAD(pickup_number::text,4,'0') AS c7,payable_cents AS c8,original_subtotal_cents AS c9,discount_cents AS c10,discount_rate_percent AS c11,contact_name_snapshot AS c12,convert_from(contact_phone_snapshot,'UTF8') AS c13,convert_from(transaction_id,'UTF8') AS c14,paid_at AS c15,materialized_at AS c16,order_note AS c17 FROM public.orders WHERE id=p_order LIMIT 1 LOOP
  IF scan_row.c0 IS NULL OR scan_row.c0 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c0)<>scan_row.c0 OR scan_row.c1 IS NULL OR scan_row.c2 IS NULL OR scan_row.c3 IS NULL OR scan_row.c4 IS NULL OR scan_row.c5 IS NULL OR scan_row.c6 IS NULL OR scan_row.c7 IS NULL OR scan_row.c8 IS NULL OR scan_row.c8 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c8)<>scan_row.c8 OR scan_row.c9 IS NULL OR scan_row.c9 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c9)<>scan_row.c9 OR scan_row.c10 IS NULL OR scan_row.c10 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c10)<>scan_row.c10 OR scan_row.c11 IS NULL OR scan_row.c11 NOT BETWEEN 0 AND 255 OR trunc(scan_row.c11)<>scan_row.c11 OR scan_row.c12 IS NULL OR scan_row.c13 IS NULL OR scan_row.c14 IS NULL OR scan_row.c15 IS NULL OR NOT isfinite(scan_row.c15) OR scan_row.c16 IS NULL OR NOT isfinite(scan_row.c16) OR scan_row.c17 IS NULL THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  rows:=rows||jsonb_build_array(jsonb_build_array(scan_row.c0::text,scan_row.c1,scan_row.c2,scan_row.c3,scan_row.c4,scan_row.c5,scan_row.c6,scan_row.c7,scan_row.c8::text,scan_row.c9::text,scan_row.c10::text,scan_row.c11::text,scan_row.c12,scan_row.c13,scan_row.c14,jsonb_build_object('seconds',floor(extract(epoch FROM scan_row.c15))::text,'nanosecond',((extract(epoch FROM scan_row.c15)-floor(extract(epoch FROM scan_row.c15)))*1000000000)::integer),jsonb_build_object('seconds',floor(extract(epoch FROM scan_row.c16))::text,'nanosecond',((extract(epoch FROM scan_row.c16)-floor(extract(epoch FROM scan_row.c16)))*1000000000)::integer),scan_row.c17));
  ids:=array_append(ids,scan_row.c0);
 END LOOP;
 IF jsonb_array_length(rows)=0 THEN RETURN jsonb_build_object('outcome','NOT_FOUND');END IF;
 FOR scan_row IN SELECT order_id AS c0,product_id AS c1,product_name_snapshot AS c2,quantity AS c3,payable_subtotal_cents AS c4,flavors_json AS c5,line_note AS c6 FROM public.order_items WHERE order_id=ANY(ids) ORDER BY order_id,line_number LOOP
  IF scan_row.c0 IS NULL OR scan_row.c0 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c0)<>scan_row.c0 OR scan_row.c1 IS NULL OR scan_row.c1 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c1)<>scan_row.c1 OR scan_row.c2 IS NULL OR scan_row.c3 IS NULL OR scan_row.c3 NOT BETWEEN 0 AND 4294967295 OR trunc(scan_row.c3)<>scan_row.c3 OR scan_row.c4 IS NULL OR scan_row.c4 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c4)<>scan_row.c4 OR scan_row.c5 IS NULL OR jsonb_typeof(scan_row.c5)<>'array' OR scan_row.c6 IS NULL THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  IF EXISTS(SELECT 1 FROM jsonb_array_elements(scan_row.c5) f WHERE jsonb_typeof(f) NOT IN ('string','null')) THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  items:=items||jsonb_build_array(jsonb_build_array(scan_row.c0::text,scan_row.c1::text,scan_row.c2,scan_row.c3::text,scan_row.c4::text,scan_row.c5,scan_row.c6));
 END LOOP;
 RETURN jsonb_build_object('outcome','FOUND','rows',rows,'items',items);
 EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_admin_report_search_v1(p_owner numeric,p_state_hex text,p_date_hex text,p_query_hex text,p_unclaimed boolean,p_after numeric,p_limit integer)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE owner numeric;scan_row record;rows jsonb:='[]';items jsonb:='[]';ids numeric[]:=ARRAY[]::numeric[];v_state text;v_date date;v_query text;v_like text;v_from date;v_to date;micros numeric;v_at timestamptz;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF p_owner IS NULL OR p_owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_owner)<>p_owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  SELECT id INTO owner FROM public.merchant_accounts WHERE bound_user_id=p_owner AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER';
  IF NOT FOUND THEN RETURN jsonb_build_object('outcome','FORBIDDEN');END IF;
  IF owner IS NULL OR owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(owner)<>owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF p_state_hex IS NULL OR p_state_hex !~ '^([0-9a-f]{2})*$' OR p_date_hex IS NULL OR p_date_hex !~ '^([0-9a-f]{2})*$' OR p_query_hex IS NULL OR p_query_hex !~ '^([0-9a-f]{2})*$' OR p_unclaimed IS NULL OR p_after IS NULL OR p_after NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_after)<>p_after OR p_limit IS NULL OR p_limit NOT BETWEEN 0 AND 65535 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 v_state:=convert_from(decode(p_state_hex,'hex'),'UTF8');
 IF p_date_hex<>'' THEN v_date:=convert_from(decode(p_date_hex,'hex'),'UTF8')::date;END IF;
 v_query:=convert_from(decode(p_query_hex,'hex'),'UTF8');
 v_like:='%'||replace(replace(replace(v_query,'=','=='),'%','=%'),'_','=_')||'%';
 FOR scan_row IN SELECT id AS c0,convert_from(order_no,'UTF8') AS c1,(CASE WHEN state='RESERVED' AND pickup_at<=statement_timestamp()+interval '30 minutes' THEN 'PREPARING' ELSE state END) AS c2,to_char(pickup_date,'YYYY-MM-DD') AS c3,to_char(pickup_date+pickup_time,'HH24:MI') AS c4,meal_period AS c5,pickup_point_snapshot AS c6,LPAD(pickup_number::text,4,'0') AS c7,payable_cents AS c8,original_subtotal_cents AS c9,discount_cents AS c10,discount_rate_percent AS c11,contact_name_snapshot AS c12,convert_from(contact_phone_snapshot,'UTF8') AS c13,convert_from(transaction_id,'UTF8') AS c14,paid_at AS c15,materialized_at AS c16,order_note AS c17 FROM public.orders WHERE id>p_after AND (p_state_hex='' OR state=CASE v_state WHEN '已预约' THEN 'RESERVED' WHEN '制作中' THEN 'PREPARING' WHEN '待取餐' THEN 'READY_FOR_PICKUP' WHEN '已完成' THEN 'COMPLETED' WHEN '退款中' THEN 'REFUNDING' WHEN '已退款' THEN 'REFUNDED' ELSE '' END) AND (p_date_hex='' OR pickup_date=v_date) AND (NOT p_unclaimed OR (state='READY_FOR_PICKUP' AND pickup_date<(clock_timestamp() AT TIME ZONE 'Asia/Shanghai')::date)) AND (p_query_hex='' OR convert_from(order_no,'UTF8') LIKE v_like ESCAPE '=' OR LPAD(pickup_number::text,4,'0')=v_query OR convert_from(contact_phone_snapshot,'UTF8') LIKE v_like ESCAPE '=') ORDER BY id LIMIT p_limit LOOP
  IF scan_row.c0 IS NULL OR scan_row.c0 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c0)<>scan_row.c0 OR scan_row.c1 IS NULL OR scan_row.c2 IS NULL OR scan_row.c3 IS NULL OR scan_row.c4 IS NULL OR scan_row.c5 IS NULL OR scan_row.c6 IS NULL OR scan_row.c7 IS NULL OR scan_row.c8 IS NULL OR scan_row.c8 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c8)<>scan_row.c8 OR scan_row.c9 IS NULL OR scan_row.c9 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c9)<>scan_row.c9 OR scan_row.c10 IS NULL OR scan_row.c10 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c10)<>scan_row.c10 OR scan_row.c11 IS NULL OR scan_row.c11 NOT BETWEEN 0 AND 255 OR trunc(scan_row.c11)<>scan_row.c11 OR scan_row.c12 IS NULL OR scan_row.c13 IS NULL OR scan_row.c14 IS NULL OR scan_row.c15 IS NULL OR NOT isfinite(scan_row.c15) OR scan_row.c16 IS NULL OR NOT isfinite(scan_row.c16) OR scan_row.c17 IS NULL THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  rows:=rows||jsonb_build_array(jsonb_build_array(scan_row.c0::text,scan_row.c1,scan_row.c2,scan_row.c3,scan_row.c4,scan_row.c5,scan_row.c6,scan_row.c7,scan_row.c8::text,scan_row.c9::text,scan_row.c10::text,scan_row.c11::text,scan_row.c12,scan_row.c13,scan_row.c14,jsonb_build_object('seconds',floor(extract(epoch FROM scan_row.c15))::text,'nanosecond',((extract(epoch FROM scan_row.c15)-floor(extract(epoch FROM scan_row.c15)))*1000000000)::integer),jsonb_build_object('seconds',floor(extract(epoch FROM scan_row.c16))::text,'nanosecond',((extract(epoch FROM scan_row.c16)-floor(extract(epoch FROM scan_row.c16)))*1000000000)::integer),scan_row.c17));
  ids:=array_append(ids,scan_row.c0);
 END LOOP;
 IF cardinality(ids)>0 THEN
 FOR scan_row IN SELECT order_id AS c0,product_id AS c1,product_name_snapshot AS c2,quantity AS c3,payable_subtotal_cents AS c4,flavors_json AS c5,line_note AS c6 FROM public.order_items WHERE order_id=ANY(ids) ORDER BY order_id,line_number LOOP
  IF scan_row.c0 IS NULL OR scan_row.c0 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c0)<>scan_row.c0 OR scan_row.c1 IS NULL OR scan_row.c1 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c1)<>scan_row.c1 OR scan_row.c2 IS NULL OR scan_row.c3 IS NULL OR scan_row.c3 NOT BETWEEN 0 AND 4294967295 OR trunc(scan_row.c3)<>scan_row.c3 OR scan_row.c4 IS NULL OR scan_row.c4 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c4)<>scan_row.c4 OR scan_row.c5 IS NULL OR jsonb_typeof(scan_row.c5)<>'array' OR scan_row.c6 IS NULL THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  IF EXISTS(SELECT 1 FROM jsonb_array_elements(scan_row.c5) f WHERE jsonb_typeof(f) NOT IN ('string','null')) THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  items:=items||jsonb_build_array(jsonb_build_array(scan_row.c0::text,scan_row.c1::text,scan_row.c2,scan_row.c3::text,scan_row.c4::text,scan_row.c5,scan_row.c6));
 END LOOP;
 END IF;
 RETURN jsonb_build_object('outcome','FOUND','rows',rows,'items',items);
 EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_admin_report_payments_v1(p_owner numeric,p_from_hex text,p_to_hex text,p_after numeric,p_limit integer)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE owner numeric;scan_row record;rows jsonb:='[]';items jsonb:='[]';ids numeric[]:=ARRAY[]::numeric[];v_state text;v_date date;v_query text;v_like text;v_from date;v_to date;micros numeric;v_at timestamptz;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF p_owner IS NULL OR p_owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_owner)<>p_owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  SELECT id INTO owner FROM public.merchant_accounts WHERE bound_user_id=p_owner AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER';
  IF NOT FOUND THEN RETURN jsonb_build_object('outcome','FORBIDDEN');END IF;
  IF owner IS NULL OR owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(owner)<>owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF p_from_hex IS NULL OR p_from_hex !~ '^([0-9a-f]{2})*$' OR p_to_hex IS NULL OR p_to_hex !~ '^([0-9a-f]{2})*$' OR p_after IS NULL OR p_after NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_after)<>p_after OR p_limit IS NULL OR p_limit NOT BETWEEN 0 AND 65535 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 v_from:=convert_from(decode(p_from_hex,'hex'),'UTF8')::date;v_to:=convert_from(decode(p_to_hex,'hex'),'UTF8')::date;
 FOR scan_row IN SELECT id AS c0,convert_from(order_no,'UTF8') AS c1,(CASE WHEN state='RESERVED' AND pickup_at<=statement_timestamp()+interval '30 minutes' THEN 'PREPARING' ELSE state END) AS c2,to_char(pickup_date,'YYYY-MM-DD') AS c3,to_char(pickup_date+pickup_time,'HH24:MI') AS c4,meal_period AS c5,pickup_point_snapshot AS c6,LPAD(pickup_number::text,4,'0') AS c7,payable_cents AS c8,original_subtotal_cents AS c9,discount_cents AS c10,discount_rate_percent AS c11,contact_name_snapshot AS c12,convert_from(contact_phone_snapshot,'UTF8') AS c13,convert_from(transaction_id,'UTF8') AS c14,paid_at AS c15,materialized_at AS c16,order_note AS c17 FROM public.orders WHERE id>p_after AND (paid_at AT TIME ZONE 'Asia/Shanghai')::date BETWEEN v_from AND v_to ORDER BY id LIMIT p_limit LOOP
  IF scan_row.c0 IS NULL OR scan_row.c0 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c0)<>scan_row.c0 OR scan_row.c1 IS NULL OR scan_row.c2 IS NULL OR scan_row.c3 IS NULL OR scan_row.c4 IS NULL OR scan_row.c5 IS NULL OR scan_row.c6 IS NULL OR scan_row.c7 IS NULL OR scan_row.c8 IS NULL OR scan_row.c8 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c8)<>scan_row.c8 OR scan_row.c9 IS NULL OR scan_row.c9 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c9)<>scan_row.c9 OR scan_row.c10 IS NULL OR scan_row.c10 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c10)<>scan_row.c10 OR scan_row.c11 IS NULL OR scan_row.c11 NOT BETWEEN 0 AND 255 OR trunc(scan_row.c11)<>scan_row.c11 OR scan_row.c12 IS NULL OR scan_row.c13 IS NULL OR scan_row.c14 IS NULL OR scan_row.c15 IS NULL OR NOT isfinite(scan_row.c15) OR scan_row.c16 IS NULL OR NOT isfinite(scan_row.c16) OR scan_row.c17 IS NULL THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  rows:=rows||jsonb_build_array(jsonb_build_array(scan_row.c0::text,scan_row.c1,scan_row.c2,scan_row.c3,scan_row.c4,scan_row.c5,scan_row.c6,scan_row.c7,scan_row.c8::text,scan_row.c9::text,scan_row.c10::text,scan_row.c11::text,scan_row.c12,scan_row.c13,scan_row.c14,jsonb_build_object('seconds',floor(extract(epoch FROM scan_row.c15))::text,'nanosecond',((extract(epoch FROM scan_row.c15)-floor(extract(epoch FROM scan_row.c15)))*1000000000)::integer),jsonb_build_object('seconds',floor(extract(epoch FROM scan_row.c16))::text,'nanosecond',((extract(epoch FROM scan_row.c16)-floor(extract(epoch FROM scan_row.c16)))*1000000000)::integer),scan_row.c17));
  ids:=array_append(ids,scan_row.c0);
 END LOOP;
 IF cardinality(ids)>0 THEN
 FOR scan_row IN SELECT order_id AS c0,product_id AS c1,product_name_snapshot AS c2,quantity AS c3,payable_subtotal_cents AS c4,flavors_json AS c5,line_note AS c6 FROM public.order_items WHERE order_id=ANY(ids) ORDER BY order_id,line_number LOOP
  IF scan_row.c0 IS NULL OR scan_row.c0 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c0)<>scan_row.c0 OR scan_row.c1 IS NULL OR scan_row.c1 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c1)<>scan_row.c1 OR scan_row.c2 IS NULL OR scan_row.c3 IS NULL OR scan_row.c3 NOT BETWEEN 0 AND 4294967295 OR trunc(scan_row.c3)<>scan_row.c3 OR scan_row.c4 IS NULL OR scan_row.c4 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c4)<>scan_row.c4 OR scan_row.c5 IS NULL OR jsonb_typeof(scan_row.c5)<>'array' OR scan_row.c6 IS NULL THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  IF EXISTS(SELECT 1 FROM jsonb_array_elements(scan_row.c5) f WHERE jsonb_typeof(f) NOT IN ('string','null')) THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  items:=items||jsonb_build_array(jsonb_build_array(scan_row.c0::text,scan_row.c1::text,scan_row.c2,scan_row.c3::text,scan_row.c4::text,scan_row.c5,scan_row.c6));
 END LOOP;
 END IF;
 RETURN jsonb_build_object('outcome','FOUND','rows',rows,'items',items);
 EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_admin_report_stats_v1(p_owner numeric,p_at jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE owner numeric;scan_row record;rows jsonb:='[]';items jsonb:='[]';ids numeric[]:=ARRAY[]::numeric[];v_state text;v_date date;v_query text;v_like text;v_from date;v_to date;micros numeric;v_at timestamptz;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF p_owner IS NULL OR p_owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_owner)<>p_owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  SELECT id INTO owner FROM public.merchant_accounts WHERE bound_user_id=p_owner AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER';
  IF NOT FOUND THEN RETURN jsonb_build_object('outcome','FORBIDDEN');END IF;
  IF owner IS NULL OR owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(owner)<>owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF jsonb_typeof(p_at) IS DISTINCT FROM 'object' OR NOT (p_at ?& ARRAY['epoch_seconds','nanosecond','zero','canonical_utc']) OR (p_at-ARRAY['epoch_seconds','nanosecond','zero','canonical_utc'])<>'{}'::jsonb OR jsonb_typeof(p_at->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_at->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_at->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_at->'nanosecond') IS DISTINCT FROM 'number' OR (p_at->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_at->>'nanosecond')::numeric)<>(p_at->>'nanosecond')::numeric OR jsonb_typeof(p_at->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_at->'canonical_utc') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 micros:=(p_at->>'epoch_seconds')::numeric*1000000+div((p_at->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN v_at:='infinity'::timestamptz;ELSIF micros=-9223372036854775808 THEN v_at:='-infinity'::timestamptz;
 ELSE v_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 FOR scan_row IN SELECT COALESCE(SUM(payable_cents) FILTER (WHERE (paid_at AT TIME ZONE 'Asia/Shanghai')::date=(v_at::timestamptz AT TIME ZONE 'Asia/Shanghai')::date AND state='COMPLETED'),0) AS c0,COUNT(*) FILTER (WHERE (paid_at AT TIME ZONE 'Asia/Shanghai')::date=(v_at::timestamptz AT TIME ZONE 'Asia/Shanghai')::date AND state='COMPLETED') AS c1,COALESCE(SUM(payable_cents) FILTER (WHERE date_trunc('month',paid_at AT TIME ZONE 'Asia/Shanghai')=date_trunc('month',v_at::timestamptz AT TIME ZONE 'Asia/Shanghai') AND state='COMPLETED'),0) AS c2,COUNT(*) FILTER (WHERE date_trunc('month',paid_at AT TIME ZONE 'Asia/Shanghai')=date_trunc('month',v_at::timestamptz AT TIME ZONE 'Asia/Shanghai') AND state='COMPLETED') AS c3,COUNT(*) FILTER (WHERE (CASE WHEN state='RESERVED' AND pickup_at<=v_at+interval '30 minutes' THEN 'PREPARING' ELSE state END)='PREPARING') AS c4 FROM public.orders WHERE (CASE WHEN state='RESERVED' AND pickup_at<=v_at+interval '30 minutes' THEN 'PREPARING' ELSE state END)='PREPARING' OR (state='COMPLETED' AND paid_at >= (date_trunc('month',v_at::timestamptz AT TIME ZONE 'Asia/Shanghai') AT TIME ZONE 'Asia/Shanghai') AND paid_at < ((date_trunc('month',v_at::timestamptz AT TIME ZONE 'Asia/Shanghai') + interval '1 month') AT TIME ZONE 'Asia/Shanghai')) LOOP
  IF scan_row.c0 IS NULL OR scan_row.c0 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c0)<>scan_row.c0 OR scan_row.c1 IS NULL OR scan_row.c1 NOT BETWEEN 0 AND 4294967295 OR trunc(scan_row.c1)<>scan_row.c1 OR scan_row.c2 IS NULL OR scan_row.c2 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c2)<>scan_row.c2 OR scan_row.c3 IS NULL OR scan_row.c3 NOT BETWEEN 0 AND 4294967295 OR trunc(scan_row.c3)<>scan_row.c3 OR scan_row.c4 IS NULL OR scan_row.c4 NOT BETWEEN 0 AND 4294967295 OR trunc(scan_row.c4)<>scan_row.c4 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  rows:=rows||jsonb_build_array(jsonb_build_array(scan_row.c0::text,scan_row.c1::text,scan_row.c2::text,scan_row.c3::text,scan_row.c4::text));
 END LOOP;
 FOR scan_row IN SELECT COALESCE(SUM(r.amount_cents),0) AS c0 FROM public.refunds r JOIN (SELECT refund_id,MIN(success_time) AS success_time FROM public.refund_observations WHERE provider_state='SUCCESS' AND validation='MATCH' AND apply_state='APPLIED' GROUP BY refund_id) observed ON observed.refund_id=r.id WHERE r.materialization_state='APPLIED' AND (observed.success_time AT TIME ZONE 'Asia/Shanghai')::date=(v_at::timestamptz AT TIME ZONE 'Asia/Shanghai')::date LOOP
  IF scan_row.c0 IS NULL OR scan_row.c0 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c0)<>scan_row.c0 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  rows:=rows||jsonb_build_array(jsonb_build_array(scan_row.c0::text));
 END LOOP;
 FOR scan_row IN SELECT oi.product_id AS c0,oi.product_name_snapshot AS c1,SUM(oi.quantity) AS c2 FROM public.order_items oi JOIN public.orders o ON o.id=oi.order_id WHERE (o.paid_at AT TIME ZONE 'Asia/Shanghai')::date=(v_at::timestamptz AT TIME ZONE 'Asia/Shanghai')::date AND o.state='COMPLETED' GROUP BY oi.product_id,oi.product_name_snapshot ORDER BY SUM(oi.quantity) DESC,oi.product_id LIMIT 10 LOOP
  IF scan_row.c0 IS NULL OR scan_row.c0 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c0)<>scan_row.c0 OR scan_row.c1 IS NULL OR scan_row.c2 IS NULL OR scan_row.c2 NOT BETWEEN 0 AND 4294967295 OR trunc(scan_row.c2)<>scan_row.c2 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  items:=items||jsonb_build_array(jsonb_build_array(scan_row.c0::text,scan_row.c1,scan_row.c2::text));
 END LOOP;
 RETURN jsonb_build_object('outcome','FOUND','rows',rows,'items',items);
 EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_admin_report_refunds_v1(p_owner numeric,p_from_hex text,p_to_hex text,p_after numeric,p_limit integer)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE owner numeric;scan_row record;rows jsonb:='[]';items jsonb:='[]';ids numeric[]:=ARRAY[]::numeric[];v_state text;v_date date;v_query text;v_like text;v_from date;v_to date;micros numeric;v_at timestamptz;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF p_owner IS NULL OR p_owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_owner)<>p_owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  SELECT id INTO owner FROM public.merchant_accounts WHERE bound_user_id=p_owner AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER';
  IF NOT FOUND THEN RETURN jsonb_build_object('outcome','FORBIDDEN');END IF;
  IF owner IS NULL OR owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(owner)<>owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF p_from_hex IS NULL OR p_from_hex !~ '^([0-9a-f]{2})*$' OR p_to_hex IS NULL OR p_to_hex !~ '^([0-9a-f]{2})*$' OR p_after IS NULL OR p_after NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_after)<>p_after OR p_limit IS NULL OR p_limit NOT BETWEEN 0 AND 65535 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 v_from:=convert_from(decode(p_from_hex,'hex'),'UTF8')::date;v_to:=convert_from(decode(p_to_hex,'hex'),'UTF8')::date;
 FOR scan_row IN SELECT r.id AS c0,COALESCE(r.order_id,0) AS c1,COALESCE(convert_from(o.order_no,'UTF8'),'') AS c2,r.provider_state AS c3,COALESCE(convert_from(r.provider_refund_id,'UTF8'),'') AS c4,COALESCE(convert_from(o.transaction_id,'UTF8'),'') AS c5,COALESCE(a.name,'系统') AS c6,r.reason_code AS c7,r.amount_cents AS c8,COALESCE(o.paid_at,r.created_at) AS c9,r.created_at AS c10 FROM public.refunds r LEFT JOIN public.orders o ON o.id=r.order_id LEFT JOIN public.merchant_accounts a ON a.id=r.requested_by_account_id LEFT JOIN (SELECT refund_id,MIN(success_time) AS success_time FROM public.refund_observations WHERE provider_state='SUCCESS' AND validation='MATCH' AND apply_state='APPLIED' GROUP BY refund_id) observed ON observed.refund_id=r.id WHERE r.id>p_after AND ((CASE WHEN r.materialization_state='APPLIED' THEN observed.success_time ELSE r.created_at END) AT TIME ZONE 'Asia/Shanghai')::date BETWEEN v_from AND v_to ORDER BY r.id LIMIT p_limit LOOP
  IF scan_row.c0 IS NULL OR scan_row.c0 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c0)<>scan_row.c0 OR scan_row.c1 IS NULL OR scan_row.c1 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c1)<>scan_row.c1 OR scan_row.c2 IS NULL OR scan_row.c3 IS NULL OR scan_row.c4 IS NULL OR scan_row.c5 IS NULL OR scan_row.c6 IS NULL OR scan_row.c7 IS NULL OR scan_row.c8 IS NULL OR scan_row.c8 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c8)<>scan_row.c8 OR scan_row.c9 IS NULL OR NOT isfinite(scan_row.c9) OR scan_row.c10 IS NULL OR NOT isfinite(scan_row.c10) THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  rows:=rows||jsonb_build_array(jsonb_build_array(scan_row.c0::text,scan_row.c1::text,scan_row.c2,scan_row.c3,scan_row.c4,scan_row.c5,scan_row.c6,scan_row.c7,scan_row.c8::text,jsonb_build_object('seconds',floor(extract(epoch FROM scan_row.c9))::text,'nanosecond',((extract(epoch FROM scan_row.c9)-floor(extract(epoch FROM scan_row.c9)))*1000000000)::integer),jsonb_build_object('seconds',floor(extract(epoch FROM scan_row.c10))::text,'nanosecond',((extract(epoch FROM scan_row.c10)-floor(extract(epoch FROM scan_row.c10)))*1000000000)::integer)));
 END LOOP;
 RETURN jsonb_build_object('outcome','FOUND','rows',rows,'items',items);
 EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_admin_report_summary_v1(p_owner numeric,p_from_hex text,p_to_hex text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE owner numeric;scan_row record;rows jsonb:='[]';items jsonb:='[]';ids numeric[]:=ARRAY[]::numeric[];v_state text;v_date date;v_query text;v_like text;v_from date;v_to date;micros numeric;v_at timestamptz;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF p_owner IS NULL OR p_owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_owner)<>p_owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  SELECT id INTO owner FROM public.merchant_accounts WHERE bound_user_id=p_owner AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER';
  IF NOT FOUND THEN RETURN jsonb_build_object('outcome','FORBIDDEN');END IF;
  IF owner IS NULL OR owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(owner)<>owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF p_from_hex IS NULL OR p_from_hex !~ '^([0-9a-f]{2})*$' OR p_to_hex IS NULL OR p_to_hex !~ '^([0-9a-f]{2})*$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 v_from:=convert_from(decode(p_from_hex,'hex'),'UTF8')::date;v_to:=convert_from(decode(p_to_hex,'hex'),'UTF8')::date;
 FOR scan_row IN SELECT COUNT(*) AS c0,COALESCE(SUM(payable_cents),0) AS c1,COUNT(*) FILTER (WHERE identity_kind='STAFF') AS c2,COALESCE(SUM(discount_cents),0) AS c3 FROM public.orders WHERE (paid_at AT TIME ZONE 'Asia/Shanghai')::date BETWEEN v_from AND v_to LOOP
  IF scan_row.c0 IS NULL OR scan_row.c0 NOT BETWEEN 0 AND 4294967295 OR trunc(scan_row.c0)<>scan_row.c0 OR scan_row.c1 IS NULL OR scan_row.c1 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c1)<>scan_row.c1 OR scan_row.c2 IS NULL OR scan_row.c2 NOT BETWEEN 0 AND 4294967295 OR trunc(scan_row.c2)<>scan_row.c2 OR scan_row.c3 IS NULL OR scan_row.c3 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c3)<>scan_row.c3 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  rows:=rows||jsonb_build_array(jsonb_build_array(scan_row.c0::text,scan_row.c1::text,scan_row.c2::text,scan_row.c3::text));
 END LOOP;
 FOR scan_row IN SELECT COUNT(*) AS c0,COALESCE(SUM(r.amount_cents),0) AS c1,COUNT(*) FILTER (WHERE r.materialization_state<>'APPLIED') AS c2 FROM public.refunds r LEFT JOIN (SELECT refund_id,MIN(success_time) AS success_time FROM public.refund_observations WHERE provider_state='SUCCESS' AND validation='MATCH' AND apply_state='APPLIED' GROUP BY refund_id) observed ON observed.refund_id=r.id WHERE ((CASE WHEN r.materialization_state='APPLIED' THEN observed.success_time ELSE r.created_at END) AT TIME ZONE 'Asia/Shanghai')::date BETWEEN v_from AND v_to LOOP
  IF scan_row.c0 IS NULL OR scan_row.c0 NOT BETWEEN 0 AND 4294967295 OR trunc(scan_row.c0)<>scan_row.c0 OR scan_row.c1 IS NULL OR scan_row.c1 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c1)<>scan_row.c1 OR scan_row.c2 IS NULL OR scan_row.c2 NOT BETWEEN 0 AND 4294967295 OR trunc(scan_row.c2)<>scan_row.c2 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  rows:=rows||jsonb_build_array(jsonb_build_array(scan_row.c0::text,scan_row.c1::text,scan_row.c2::text));
 END LOOP;
 FOR scan_row IN SELECT COUNT(*) AS c0,COALESCE(SUM(expected_amount_cents),0) AS c1 FROM public.prepayments WHERE materialization_state='PENDING_MANUAL' AND (created_at AT TIME ZONE 'Asia/Shanghai')::date BETWEEN v_from AND v_to LOOP
  IF scan_row.c0 IS NULL OR scan_row.c0 NOT BETWEEN 0 AND 4294967295 OR trunc(scan_row.c0)<>scan_row.c0 OR scan_row.c1 IS NULL OR scan_row.c1 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c1)<>scan_row.c1 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  rows:=rows||jsonb_build_array(jsonb_build_array(scan_row.c0::text,scan_row.c1::text));
 END LOOP;
 RETURN jsonb_build_object('outcome','FOUND','rows',rows,'items',items);
 EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_admin_report_pending_v1(p_owner numeric,p_after numeric,p_limit integer)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE owner numeric;scan_row record;rows jsonb:='[]';items jsonb:='[]';ids numeric[]:=ARRAY[]::numeric[];v_state text;v_date date;v_query text;v_like text;v_from date;v_to date;micros numeric;v_at timestamptz;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF p_owner IS NULL OR p_owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_owner)<>p_owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  SELECT id INTO owner FROM public.merchant_accounts WHERE bound_user_id=p_owner AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER';
  IF NOT FOUND THEN RETURN jsonb_build_object('outcome','FORBIDDEN');END IF;
  IF owner IS NULL OR owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(owner)<>owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF p_after IS NULL OR p_after NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_after)<>p_after OR p_limit IS NULL OR p_limit NOT BETWEEN 0 AND 65535 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 FOR scan_row IN SELECT p.id AS c0,p.quote_id AS c1,convert_from(p.out_trade_no,'UTF8') AS c2,p.materialization_state AS c3,p.pending_reason_code AS c4,to_char(q.pickup_date,'YYYY-MM-DD') AS c5,to_char(q.pickup_date+q.pickup_time,'HH24:MI') AS c6,q.meal_period AS c7,q.contact_name_snapshot AS c8,convert_from(q.contact_phone_snapshot,'UTF8') AS c9,p.expected_amount_cents AS c10,COALESCE((SELECT MIN(po.success_time) FROM public.payment_observations po WHERE po.prepayment_id=p.id AND po.provider_state='PAID' AND po.validation='MATCH'),p.updated_at) AS c11,COALESCE((SELECT convert_from(po.transaction_id,'UTF8') FROM public.payment_observations po WHERE po.prepayment_id=p.id AND po.provider_state='PAID' AND po.validation='MATCH' ORDER BY po.id LIMIT 1),'') AS c12 FROM public.prepayments p JOIN public.quotes q ON q.id=p.quote_id WHERE p.materialization_state='PENDING_MANUAL' AND p.id>p_after ORDER BY p.id LIMIT p_limit LOOP
  IF scan_row.c0 IS NULL OR scan_row.c0 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c0)<>scan_row.c0 OR scan_row.c1 IS NULL OR scan_row.c1 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c1)<>scan_row.c1 OR scan_row.c2 IS NULL OR scan_row.c3 IS NULL OR scan_row.c4 IS NULL OR scan_row.c5 IS NULL OR scan_row.c6 IS NULL OR scan_row.c7 IS NULL OR scan_row.c8 IS NULL OR scan_row.c9 IS NULL OR scan_row.c10 IS NULL OR scan_row.c10 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c10)<>scan_row.c10 OR scan_row.c11 IS NULL OR NOT isfinite(scan_row.c11) OR scan_row.c12 IS NULL THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  rows:=rows||jsonb_build_array(jsonb_build_array(scan_row.c0::text,scan_row.c1::text,scan_row.c2,scan_row.c3,scan_row.c4,scan_row.c5,scan_row.c6,scan_row.c7,scan_row.c8,scan_row.c9,scan_row.c10::text,jsonb_build_object('seconds',floor(extract(epoch FROM scan_row.c11))::text,'nanosecond',((extract(epoch FROM scan_row.c11)-floor(extract(epoch FROM scan_row.c11)))*1000000000)::integer),scan_row.c12));
  ids:=array_append(ids,scan_row.c1);
 END LOOP;
 IF cardinality(ids)>0 THEN
 FOR scan_row IN SELECT quote_id AS c0,product_id AS c1,product_name_snapshot AS c2,quantity AS c3,payable_subtotal_cents AS c4 FROM public.quote_items WHERE quote_id=ANY(ids) ORDER BY quote_id,line_number LOOP
  IF scan_row.c0 IS NULL OR scan_row.c0 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c0)<>scan_row.c0 OR scan_row.c1 IS NULL OR scan_row.c1 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c1)<>scan_row.c1 OR scan_row.c2 IS NULL OR scan_row.c3 IS NULL OR scan_row.c3 NOT BETWEEN 0 AND 4294967295 OR trunc(scan_row.c3)<>scan_row.c3 OR scan_row.c4 IS NULL OR scan_row.c4 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c4)<>scan_row.c4 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  items:=items||jsonb_build_array(jsonb_build_array(scan_row.c0::text,scan_row.c1::text,scan_row.c2,scan_row.c3::text,scan_row.c4::text));
 END LOOP;
 END IF;
 RETURN jsonb_build_object('outcome','FOUND','rows',rows,'items',items);
 EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_admin_report_export_v1(p_owner numeric,p_from_hex text,p_to_hex text,p_after numeric,p_limit integer)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE owner numeric;scan_row record;rows jsonb:='[]';v_from date;v_to date;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF p_owner IS NULL OR p_owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_owner)<>p_owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  SELECT id INTO owner FROM public.merchant_accounts WHERE bound_user_id=p_owner AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER';
  IF NOT FOUND THEN RETURN jsonb_build_object('outcome','FORBIDDEN');END IF;
  IF owner IS NULL OR owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(owner)<>owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF p_from_hex IS NULL OR p_from_hex !~ '^([0-9a-f]{2})*$' OR p_to_hex IS NULL OR p_to_hex !~ '^([0-9a-f]{2})*$' OR p_after IS NULL OR p_after NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_after)<>p_after OR p_limit IS NULL OR p_limit NOT BETWEEN 0 AND 65535 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 v_from:=convert_from(decode(p_from_hex,'hex'),'UTF8')::date;v_to:=convert_from(decode(p_to_hex,'hex'),'UTF8')::date;
 FOR scan_row IN SELECT id AS c0,convert_from(order_no,'UTF8') AS c1,LPAD(pickup_number::text,4,'0') AS c2,paid_at AS c3,convert_from(transaction_id,'UTF8') AS c4,payable_cents AS c5 FROM public.orders WHERE id>p_after AND (paid_at AT TIME ZONE 'Asia/Shanghai')::date BETWEEN v_from AND v_to ORDER BY id LIMIT p_limit LOOP
  IF scan_row.c0 IS NULL OR scan_row.c0 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c0)<>scan_row.c0 OR scan_row.c1 IS NULL OR scan_row.c2 IS NULL OR scan_row.c3 IS NULL OR NOT isfinite(scan_row.c3) OR scan_row.c4 IS NULL OR scan_row.c5 IS NULL OR scan_row.c5 NOT BETWEEN 0 AND 18446744073709551615 OR trunc(scan_row.c5)<>scan_row.c5 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  rows:=rows||jsonb_build_array(jsonb_build_array(scan_row.c0::text,scan_row.c1,scan_row.c2,jsonb_build_object('seconds',floor(extract(epoch FROM scan_row.c3))::text,'nanosecond',((extract(epoch FROM scan_row.c3)-floor(extract(epoch FROM scan_row.c3)))*1000000000)::integer),scan_row.c4,scan_row.c5::text));
 END LOOP;
 RETURN jsonb_build_object('outcome','FOUND','rows',rows);
 EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END;
END
$order_rpc$;
CREATE OR REPLACE FUNCTION public.order_admin_report_counts_v1(p_owner numeric)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE owner numeric;scan_row record;rows jsonb:='[]';
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF p_owner IS NULL OR p_owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_owner)<>p_owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  SELECT id INTO owner FROM public.merchant_accounts WHERE bound_user_id=p_owner AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER';
  IF NOT FOUND THEN RETURN jsonb_build_object('outcome','FORBIDDEN');END IF;
  IF owner IS NULL OR owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(owner)<>owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 FOR scan_row IN SELECT (CASE WHEN state='RESERVED' AND pickup_at<=statement_timestamp()+interval '30 minutes' THEN 'PREPARING' ELSE state END) AS state,COUNT(*) AS n FROM public.orders GROUP BY (CASE WHEN state='RESERVED' AND pickup_at<=statement_timestamp()+interval '30 minutes' THEN 'PREPARING' ELSE state END) LOOP
  IF scan_row.state NOT IN ('RESERVED','PREPARING','READY_FOR_PICKUP','COMPLETED','REFUNDING','REFUNDED') THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  rows:=rows||jsonb_build_array(jsonb_build_array(scan_row.state,scan_row.n::text));
 END LOOP;
 RETURN jsonb_build_object('outcome','FOUND','rows',rows);
 EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END;
END
$order_rpc$;
