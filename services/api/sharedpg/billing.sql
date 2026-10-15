-- Six fixed billing ports. Each VOLATILE read statement and its Scan checks retain the original stage order.
-- All calls use the ordinary service UID; no helper RPC, new table or generic SQL capability.

CREATE OR REPLACE FUNCTION public.order_billing_summary_v1(p_owner numeric,p_from jsonb,p_to jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE owner_id numeric;micros numeric;from_at timestamptz;to_at timestamptz;revenue numeric;orders_count numeric;refunds_cents numeric;refunds_count numeric;pending_count numeric;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF p_owner IS NULL OR p_owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_owner)<>p_owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 SELECT id INTO owner_id FROM public.merchant_accounts WHERE bound_user_id=p_owner AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER';
 IF NOT FOUND THEN RETURN jsonb_build_object('outcome','FORBIDDEN');END IF;
 IF owner_id IS NULL OR owner_id NOT BETWEEN 0 AND 18446744073709551615 OR trunc(owner_id)<>owner_id THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 -- Consume the four raw Go time facts at the original SQL bind, preserving pgx int64 microsecond wrapping.
 IF jsonb_typeof(p_from) IS DISTINCT FROM 'object' OR NOT (p_from ?& ARRAY['epoch_seconds','nanosecond','zero','canonical_utc']) OR (p_from-ARRAY['epoch_seconds','nanosecond','zero','canonical_utc'])<>'{}'::jsonb OR jsonb_typeof(p_from->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_from->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_from->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_from->'nanosecond') IS DISTINCT FROM 'number' OR (p_from->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_from->>'nanosecond')::numeric)<>(p_from->>'nanosecond')::numeric OR jsonb_typeof(p_from->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_from->'canonical_utc') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 micros:=(p_from->>'epoch_seconds')::numeric*1000000+div((p_from->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN from_at:='infinity'::timestamptz;ELSIF micros=-9223372036854775808 THEN from_at:='-infinity'::timestamptz;
 ELSE from_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 -- Consume the four raw Go time facts at the original SQL bind, preserving pgx int64 microsecond wrapping.
 IF jsonb_typeof(p_to) IS DISTINCT FROM 'object' OR NOT (p_to ?& ARRAY['epoch_seconds','nanosecond','zero','canonical_utc']) OR (p_to-ARRAY['epoch_seconds','nanosecond','zero','canonical_utc'])<>'{}'::jsonb OR jsonb_typeof(p_to->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_to->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_to->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_to->'nanosecond') IS DISTINCT FROM 'number' OR (p_to->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_to->>'nanosecond')::numeric)<>(p_to->>'nanosecond')::numeric OR jsonb_typeof(p_to->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_to->'canonical_utc') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 micros:=(p_to->>'epoch_seconds')::numeric*1000000+div((p_to->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN to_at:='infinity'::timestamptz;ELSIF micros=-9223372036854775808 THEN to_at:='-infinity'::timestamptz;
 ELSE to_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 -- Stage 1: completed order revenue. Scan must succeed before reading refunds.
 SELECT COALESCE(SUM(payable_cents) FILTER (WHERE state='COMPLETED'),0),COUNT(*) FILTER (WHERE state='COMPLETED') INTO revenue,orders_count FROM public.orders WHERE paid_at>=from_at AND paid_at<to_at;
 IF revenue IS NULL OR revenue NOT BETWEEN 0 AND 18446744073709551615 OR trunc(revenue)<>revenue OR orders_count IS NULL OR orders_count NOT BETWEEN 0 AND 18446744073709551615 OR trunc(orders_count)<>orders_count THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 -- Stage 2: original requested-at refund range and materialization counters.
 SELECT COALESCE(SUM(amount_cents) FILTER (WHERE materialization_state='APPLIED'),0),COUNT(*) FILTER (WHERE materialization_state='APPLIED'),COUNT(*) FILTER (WHERE materialization_state<>'APPLIED') INTO refunds_cents,refunds_count,pending_count FROM public.refunds WHERE created_at>=from_at AND created_at<to_at;
 IF refunds_cents IS NULL OR refunds_cents NOT BETWEEN 0 AND 18446744073709551615 OR trunc(refunds_cents)<>refunds_cents OR refunds_count IS NULL OR refunds_count NOT BETWEEN 0 AND 18446744073709551615 OR trunc(refunds_count)<>refunds_count OR pending_count IS NULL OR pending_count NOT BETWEEN 0 AND 18446744073709551615 OR trunc(pending_count)<>pending_count THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 RETURN jsonb_build_object('outcome','FOUND','summary',jsonb_build_object('effective_revenue_cents',revenue::text,'effective_orders',orders_count::text,'refund_cents',refunds_cents::text,'refund_count',refunds_count::text,'pending_refunds',pending_count::text));
 EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_billing_payments_v1(p_owner numeric,p_from jsonb,p_to jsonb,p_after numeric,p_limit integer)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE owner_id numeric;micros numeric;from_at timestamptz;to_at timestamptz;item record;result jsonb:='[]'::jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF p_owner IS NULL OR p_owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_owner)<>p_owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 SELECT id INTO owner_id FROM public.merchant_accounts WHERE bound_user_id=p_owner AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER';
 IF NOT FOUND THEN RETURN jsonb_build_object('outcome','FORBIDDEN');END IF;
 IF owner_id IS NULL OR owner_id NOT BETWEEN 0 AND 18446744073709551615 OR trunc(owner_id)<>owner_id THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF p_after IS NULL OR p_after NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_after)<>p_after THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF p_limit IS NULL OR p_limit NOT BETWEEN 1 AND 100 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 -- Consume the four raw Go time facts at the original SQL bind, preserving pgx int64 microsecond wrapping.
 IF jsonb_typeof(p_from) IS DISTINCT FROM 'object' OR NOT (p_from ?& ARRAY['epoch_seconds','nanosecond','zero','canonical_utc']) OR (p_from-ARRAY['epoch_seconds','nanosecond','zero','canonical_utc'])<>'{}'::jsonb OR jsonb_typeof(p_from->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_from->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_from->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_from->'nanosecond') IS DISTINCT FROM 'number' OR (p_from->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_from->>'nanosecond')::numeric)<>(p_from->>'nanosecond')::numeric OR jsonb_typeof(p_from->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_from->'canonical_utc') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 micros:=(p_from->>'epoch_seconds')::numeric*1000000+div((p_from->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN from_at:='infinity'::timestamptz;ELSIF micros=-9223372036854775808 THEN from_at:='-infinity'::timestamptz;
 ELSE from_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 -- Consume the four raw Go time facts at the original SQL bind, preserving pgx int64 microsecond wrapping.
 IF jsonb_typeof(p_to) IS DISTINCT FROM 'object' OR NOT (p_to ?& ARRAY['epoch_seconds','nanosecond','zero','canonical_utc']) OR (p_to-ARRAY['epoch_seconds','nanosecond','zero','canonical_utc'])<>'{}'::jsonb OR jsonb_typeof(p_to->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_to->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_to->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_to->'nanosecond') IS DISTINCT FROM 'number' OR (p_to->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_to->>'nanosecond')::numeric)<>(p_to->>'nanosecond')::numeric OR jsonb_typeof(p_to->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_to->'canonical_utc') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 micros:=(p_to->>'epoch_seconds')::numeric*1000000+div((p_to->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN to_at:='infinity'::timestamptz;ELSIF micros=-9223372036854775808 THEN to_at:='-infinity'::timestamptz;
 ELSE to_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 FOR item IN SELECT o.id,convert_from(o.order_no,'UTF8') AS order_no,convert_from(p.out_trade_no,'UTF8') AS out_trade_no,convert_from(o.transaction_id,'UTF8') AS transaction_id,o.state,o.payable_cents,o.paid_at FROM public.orders o JOIN public.prepayments p ON p.id=o.prepayment_id WHERE o.id>p_after AND o.paid_at>=from_at AND o.paid_at<to_at ORDER BY o.id LIMIT p_limit LOOP
 IF item.id IS NULL OR item.id NOT BETWEEN 0 AND 18446744073709551615 OR trunc(item.id)<>item.id OR item.payable_cents IS NULL OR item.payable_cents NOT BETWEEN 0 AND 18446744073709551615 OR trunc(item.payable_cents)<>item.payable_cents THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF item.order_no IS NULL OR item.out_trade_no IS NULL OR item.transaction_id IS NULL OR item.state IS NULL OR item.paid_at IS NULL OR NOT isfinite(item.paid_at) THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  result:=result||jsonb_build_array(jsonb_build_object('order_id',item.id::text,'order_no_hex',encode(convert_to(item.order_no,'UTF8'),'hex'),'out_trade_no_hex',encode(convert_to(item.out_trade_no,'UTF8'),'hex'),'transaction_id_hex',encode(convert_to(item.transaction_id,'UTF8'),'hex'),'state',item.state,'amount_cents',item.payable_cents::text,'paid_at',jsonb_build_object('kind','finite','microseconds',((extract(epoch FROM item.paid_at)-946684800)*1000000)::numeric(30,0)::text)));
 END LOOP;
 RETURN jsonb_build_object('outcome','FOUND','payments',result);
 EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_billing_refunds_v1(p_owner numeric,p_from jsonb,p_to jsonb,p_after numeric,p_limit integer)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE owner_id numeric;micros numeric;from_at timestamptz;to_at timestamptz;item record;result jsonb:='[]'::jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF p_owner IS NULL OR p_owner NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_owner)<>p_owner THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 SELECT id INTO owner_id FROM public.merchant_accounts WHERE bound_user_id=p_owner AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER';
 IF NOT FOUND THEN RETURN jsonb_build_object('outcome','FORBIDDEN');END IF;
 IF owner_id IS NULL OR owner_id NOT BETWEEN 0 AND 18446744073709551615 OR trunc(owner_id)<>owner_id THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF p_after IS NULL OR p_after NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_after)<>p_after THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF p_limit IS NULL OR p_limit NOT BETWEEN 1 AND 100 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 -- Consume the four raw Go time facts at the original SQL bind, preserving pgx int64 microsecond wrapping.
 IF jsonb_typeof(p_from) IS DISTINCT FROM 'object' OR NOT (p_from ?& ARRAY['epoch_seconds','nanosecond','zero','canonical_utc']) OR (p_from-ARRAY['epoch_seconds','nanosecond','zero','canonical_utc'])<>'{}'::jsonb OR jsonb_typeof(p_from->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_from->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_from->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_from->'nanosecond') IS DISTINCT FROM 'number' OR (p_from->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_from->>'nanosecond')::numeric)<>(p_from->>'nanosecond')::numeric OR jsonb_typeof(p_from->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_from->'canonical_utc') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 micros:=(p_from->>'epoch_seconds')::numeric*1000000+div((p_from->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN from_at:='infinity'::timestamptz;ELSIF micros=-9223372036854775808 THEN from_at:='-infinity'::timestamptz;
 ELSE from_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 -- Consume the four raw Go time facts at the original SQL bind, preserving pgx int64 microsecond wrapping.
 IF jsonb_typeof(p_to) IS DISTINCT FROM 'object' OR NOT (p_to ?& ARRAY['epoch_seconds','nanosecond','zero','canonical_utc']) OR (p_to-ARRAY['epoch_seconds','nanosecond','zero','canonical_utc'])<>'{}'::jsonb OR jsonb_typeof(p_to->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_to->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_to->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_to->'nanosecond') IS DISTINCT FROM 'number' OR (p_to->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_to->>'nanosecond')::numeric)<>(p_to->>'nanosecond')::numeric OR jsonb_typeof(p_to->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_to->'canonical_utc') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 micros:=(p_to->>'epoch_seconds')::numeric*1000000+div((p_to->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN to_at:='infinity'::timestamptz;ELSIF micros=-9223372036854775808 THEN to_at:='-infinity'::timestamptz;
 ELSE to_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 FOR item IN SELECT id,COALESCE(order_id,0) AS order_id,convert_from(out_refund_no,'UTF8') AS out_refund_no,COALESCE(convert_from(provider_refund_id,'UTF8'),'') AS provider_refund_id,provider_state,amount_cents,created_at FROM public.refunds WHERE id>p_after AND created_at>=from_at AND created_at<to_at ORDER BY id LIMIT p_limit LOOP
 IF item.id IS NULL OR item.id NOT BETWEEN 0 AND 18446744073709551615 OR trunc(item.id)<>item.id OR item.order_id IS NULL OR item.order_id NOT BETWEEN 0 AND 18446744073709551615 OR trunc(item.order_id)<>item.order_id OR item.amount_cents IS NULL OR item.amount_cents NOT BETWEEN 0 AND 18446744073709551615 OR trunc(item.amount_cents)<>item.amount_cents THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF item.out_refund_no IS NULL OR item.provider_refund_id IS NULL OR item.provider_state IS NULL OR item.created_at IS NULL OR NOT isfinite(item.created_at) THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  result:=result||jsonb_build_array(jsonb_build_object('id',item.id::text,'order_id',item.order_id::text,'out_refund_no_hex',encode(convert_to(item.out_refund_no,'UTF8'),'hex'),'provider_refund_id_hex',encode(convert_to(item.provider_refund_id,'UTF8'),'hex'),'state',item.provider_state,'amount_cents',item.amount_cents::text,'requested_at',jsonb_build_object('kind','finite','microseconds',((extract(epoch FROM item.created_at)-946684800)*1000000)::numeric(30,0)::text)));
 END LOOP;
 RETURN jsonb_build_object('outcome','FOUND','refunds',result);
 EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_billing_payment_facts_v1(p_from jsonb,p_to jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE micros numeric;from_at timestamptz;to_at timestamptz;item record;result jsonb:='[]'::jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 -- Consume the four raw Go time facts at the original SQL bind, preserving pgx int64 microsecond wrapping.
 IF jsonb_typeof(p_from) IS DISTINCT FROM 'object' OR NOT (p_from ?& ARRAY['epoch_seconds','nanosecond','zero','canonical_utc']) OR (p_from-ARRAY['epoch_seconds','nanosecond','zero','canonical_utc'])<>'{}'::jsonb OR jsonb_typeof(p_from->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_from->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_from->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_from->'nanosecond') IS DISTINCT FROM 'number' OR (p_from->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_from->>'nanosecond')::numeric)<>(p_from->>'nanosecond')::numeric OR jsonb_typeof(p_from->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_from->'canonical_utc') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 micros:=(p_from->>'epoch_seconds')::numeric*1000000+div((p_from->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN from_at:='infinity'::timestamptz;ELSIF micros=-9223372036854775808 THEN from_at:='-infinity'::timestamptz;
 ELSE from_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 -- Consume the four raw Go time facts at the original SQL bind, preserving pgx int64 microsecond wrapping.
 IF jsonb_typeof(p_to) IS DISTINCT FROM 'object' OR NOT (p_to ?& ARRAY['epoch_seconds','nanosecond','zero','canonical_utc']) OR (p_to-ARRAY['epoch_seconds','nanosecond','zero','canonical_utc'])<>'{}'::jsonb OR jsonb_typeof(p_to->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_to->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_to->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_to->'nanosecond') IS DISTINCT FROM 'number' OR (p_to->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_to->>'nanosecond')::numeric)<>(p_to->>'nanosecond')::numeric OR jsonb_typeof(p_to->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_to->'canonical_utc') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 micros:=(p_to->>'epoch_seconds')::numeric*1000000+div((p_to->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN to_at:='infinity'::timestamptz;ELSIF micros=-9223372036854775808 THEN to_at:='-infinity'::timestamptz;
 ELSE to_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 FOR item IN SELECT convert_from(p.out_trade_no,'UTF8') AS number,convert_from(o.transaction_id,'UTF8') AS provider_id,o.payable_cents AS amount_cents,p.currency,o.paid_at AS occurred_at FROM public.orders o JOIN public.prepayments p ON p.id=o.prepayment_id WHERE o.paid_at>=from_at AND o.paid_at<to_at ORDER BY p.out_trade_no LOOP
 IF item.amount_cents IS NULL OR item.amount_cents NOT BETWEEN 0 AND 18446744073709551615 OR trunc(item.amount_cents)<>item.amount_cents THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF item.number IS NULL OR item.provider_id IS NULL OR item.currency IS NULL OR item.occurred_at IS NULL OR NOT isfinite(item.occurred_at) THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  result:=result||jsonb_build_array(jsonb_build_object('number_hex',encode(convert_to(item.number,'UTF8'),'hex'),'provider_id_hex',encode(convert_to(item.provider_id,'UTF8'),'hex'),'amount_cents',item.amount_cents::text,'currency',item.currency,'occurred_at',jsonb_build_object('kind','finite','microseconds',((extract(epoch FROM item.occurred_at)-946684800)*1000000)::numeric(30,0)::text)));
 END LOOP;
 RETURN jsonb_build_object('outcome','FOUND','entries',result);
 EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_billing_refund_facts_v1(p_from jsonb,p_to jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE micros numeric;from_at timestamptz;to_at timestamptz;item record;result jsonb:='[]'::jsonb;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 -- Consume the four raw Go time facts at the original SQL bind, preserving pgx int64 microsecond wrapping.
 IF jsonb_typeof(p_from) IS DISTINCT FROM 'object' OR NOT (p_from ?& ARRAY['epoch_seconds','nanosecond','zero','canonical_utc']) OR (p_from-ARRAY['epoch_seconds','nanosecond','zero','canonical_utc'])<>'{}'::jsonb OR jsonb_typeof(p_from->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_from->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_from->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_from->'nanosecond') IS DISTINCT FROM 'number' OR (p_from->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_from->>'nanosecond')::numeric)<>(p_from->>'nanosecond')::numeric OR jsonb_typeof(p_from->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_from->'canonical_utc') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 micros:=(p_from->>'epoch_seconds')::numeric*1000000+div((p_from->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN from_at:='infinity'::timestamptz;ELSIF micros=-9223372036854775808 THEN from_at:='-infinity'::timestamptz;
 ELSE from_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 -- Consume the four raw Go time facts at the original SQL bind, preserving pgx int64 microsecond wrapping.
 IF jsonb_typeof(p_to) IS DISTINCT FROM 'object' OR NOT (p_to ?& ARRAY['epoch_seconds','nanosecond','zero','canonical_utc']) OR (p_to-ARRAY['epoch_seconds','nanosecond','zero','canonical_utc'])<>'{}'::jsonb OR jsonb_typeof(p_to->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_to->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_to->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_to->'nanosecond') IS DISTINCT FROM 'number' OR (p_to->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_to->>'nanosecond')::numeric)<>(p_to->>'nanosecond')::numeric OR jsonb_typeof(p_to->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_to->'canonical_utc') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 micros:=(p_to->>'epoch_seconds')::numeric*1000000+div((p_to->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN to_at:='infinity'::timestamptz;ELSIF micros=-9223372036854775808 THEN to_at:='-infinity'::timestamptz;
 ELSE to_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 FOR item IN SELECT convert_from(r.out_refund_no,'UTF8') AS number,convert_from(r.provider_refund_id,'UTF8') AS provider_id,r.amount_cents,r.currency,observed.success_time AS occurred_at FROM public.refunds r JOIN (SELECT refund_id,MIN(success_time) AS success_time FROM public.refund_observations WHERE provider_state='SUCCESS' AND validation='MATCH' AND apply_state='APPLIED' GROUP BY refund_id) observed ON observed.refund_id=r.id WHERE r.materialization_state='APPLIED' AND observed.success_time>=from_at AND observed.success_time<to_at ORDER BY r.out_refund_no LOOP
 IF item.amount_cents IS NULL OR item.amount_cents NOT BETWEEN 0 AND 18446744073709551615 OR trunc(item.amount_cents)<>item.amount_cents THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF item.number IS NULL OR item.provider_id IS NULL OR item.currency IS NULL OR item.occurred_at IS NULL OR NOT isfinite(item.occurred_at) THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  result:=result||jsonb_build_array(jsonb_build_object('number_hex',encode(convert_to(item.number,'UTF8'),'hex'),'provider_id_hex',encode(convert_to(item.provider_id,'UTF8'),'hex'),'amount_cents',item.amount_cents::text,'currency',item.currency,'occurred_at',jsonb_build_object('kind','finite','microseconds',((extract(epoch FROM item.occurred_at)-946684800)*1000000)::numeric(30,0)::text)));
 END LOOP;
 RETURN jsonb_build_object('outcome','FOUND','entries',result);
 EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_billing_reconcile_v1(p_date text,p_digest_hex text,p_matched numeric,p_provider_only jsonb,p_system_only jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE intrinsic bytea;lock_unsigned numeric;position integer;existing_id numeric;entry jsonb;target_id numeric;payment_ids numeric[]:='{}'::numeric[];refund_ids numeric[]:='{}'::numeric[];evidence jsonb;audit_result text;reason text;audit_at timestamptz;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 -- Go constructed date.Format + NUL + digest before BeginTx. Date is hash material, not a new calendar parser.
 IF p_date IS NULL OR p_digest_hex IS NULL OR p_digest_hex !~ '^[0-9a-f]{64}$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 intrinsic:=sha256(convert_to(p_date,'UTF8')||decode('00','hex')||decode(p_digest_hex,'hex'));
 lock_unsigned:=0;FOR position IN 0..7 LOOP lock_unsigned:=lock_unsigned*256+get_byte(intrinsic,position);END LOOP;
 IF lock_unsigned>=9223372036854775808 THEN lock_unsigned:=lock_unsigned-18446744073709551616;END IF;
 PERFORM set_config('lock_timeout','5s',true);
 PERFORM pg_advisory_xact_lock(lock_unsigned::bigint);
 SELECT id INTO existing_id FROM public.action_audits WHERE entry_kind='SYSTEM_EVIDENCE' AND actor_kind='PROVIDER' AND action='billing.reconcile' AND target_key_hash=intrinsic ORDER BY id LIMIT 1;
 IF FOUND THEN
 IF existing_id IS NULL OR existing_id NOT BETWEEN 0 AND 18446744073709551615 OR trunc(existing_id)<>existing_id THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  -- Original duplicate commits before projecting entries or sampling the independent audit clock.
  RETURN jsonb_build_object('outcome','RECONCILED');
 END IF;
 IF jsonb_typeof(p_provider_only) IS DISTINCT FROM 'array' OR jsonb_typeof(p_system_only) IS DISTINCT FROM 'array' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 FOR entry IN SELECT value FROM jsonb_array_elements(p_provider_only||p_system_only) LOOP
  IF jsonb_typeof(entry) IS DISTINCT FROM 'object' OR NOT (entry ?& ARRAY['kind','out_trade_no_hex','out_refund_no_hex']) OR (entry-ARRAY['kind','out_trade_no_hex','out_refund_no_hex'])<>'{}'::jsonb OR jsonb_typeof(entry->'kind') IS DISTINCT FROM 'string' OR jsonb_typeof(entry->'out_trade_no_hex') IS DISTINCT FROM 'string' OR jsonb_typeof(entry->'out_refund_no_hex') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  -- Only the original target field is consumed. Raw bytea deliberately keeps invalid UTF8 and NUL.
  IF entry->>'kind'='PAYMENT' THEN
   IF (entry->>'out_trade_no_hex') !~ '^([0-9a-f]{2})*$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
   SELECT id INTO target_id FROM public.prepayments WHERE out_trade_no=decode(entry->>'out_trade_no_hex','hex');
  ELSE
   IF (entry->>'out_refund_no_hex') !~ '^([0-9a-f]{2})*$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
   SELECT id INTO target_id FROM public.refunds WHERE out_refund_no=decode(entry->>'out_refund_no_hex','hex');
  END IF;
  IF NOT FOUND THEN CONTINUE;END IF;
 IF target_id IS NULL OR target_id NOT BETWEEN 0 AND 18446744073709551615 OR trunc(target_id)<>target_id THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  IF entry->>'kind'='PAYMENT' THEN payment_ids:=array_append(payment_ids,target_id);ELSE refund_ids:=array_append(refund_ids,target_id);END IF;
 END LOOP;
 -- Original map deduplication and sorted unsigned numeric id order, payments before refunds.
 FOR target_id IN SELECT DISTINCT value FROM unnest(payment_ids) AS ids(value) ORDER BY value LOOP
  UPDATE public.prepayments SET materialization_state='PENDING_MANUAL',pending_reason_code='BILL_MISMATCH',materialized_at=NULL,record_version=record_version+1,updated_at=clock_timestamp() WHERE id=target_id AND materialization_state IN ('AWAITING_PAYMENT','READY','PENDING_MANUAL');
 END LOOP;
 FOR target_id IN SELECT DISTINCT value FROM unnest(refund_ids) AS ids(value) ORDER BY value LOOP
  UPDATE public.refunds SET materialization_state='PENDING_MANUAL',pending_reason_code='BILL_MISMATCH',materialized_at=NULL,record_version=record_version+1,updated_at=clock_timestamp() WHERE id=target_id AND materialization_state<>'APPLIED';
 END LOOP;
 -- All failures after a write must raise into this block's exception handler, rolling its subtransaction back.
 IF p_matched IS NULL OR p_matched NOT BETWEEN 0 AND 18446744073709551615 OR trunc(p_matched)<>p_matched THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_BILLING_UNAVAILABLE';END IF;
 evidence:=jsonb_build_object('date',p_date,'digest',p_digest_hex,'matched',p_matched,'provider_only',jsonb_array_length(p_provider_only)::numeric,'system_only',jsonb_array_length(p_system_only)::numeric);
 audit_result:='SUCCEEDED';reason:='BILL_MATCHED';
 IF jsonb_array_length(p_provider_only)>0 OR jsonb_array_length(p_system_only)>0 THEN audit_result:='REJECTED';reason:='BILL_MISMATCH';END IF;
 -- Approved host-clock change: sample at the original audit INSERT consumption point, after every pending UPDATE.
 audit_at:=clock_timestamp();
 INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,action,target_type,target_id,target_key_hash,operation_key_hash,result,reason_code,after_state_json,occurred_at)
 VALUES('SYSTEM_EVIDENCE','PROVIDER',sha256(convert_to('PROVIDER:BILLING','UTF8')),NULL,NULL,'billing.reconcile','BILL',NULL,intrinsic,NULL,audit_result,reason,evidence,audit_at);
 RETURN jsonb_build_object('outcome','RECONCILED');
 EXCEPTION WHEN OTHERS THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END;
END
$order_rpc$;

-- Successful reconciliation is durable across process restarts; do not download the same completed bill again.
CREATE OR REPLACE FUNCTION public.order_billing_completed_v1(p_date text,p_environment text,p_appid text,p_merchant text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE receipt public.action_audits%ROWTYPE;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 IF p_date IS NULL OR p_date !~ '^\d{4}-\d{2}-\d{2}$' OR p_environment IS DISTINCT FROM '__ORDER_ENVIRONMENT__' OR coalesce(p_appid,'')='' OR coalesce(p_merchant,'')='' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 SELECT * INTO receipt FROM public.action_audits WHERE entry_kind='SYSTEM_EVIDENCE' AND actor_kind='PROVIDER' AND action='billing.reconcile' AND after_state_json->>'date'=p_date AND after_state_json->>'environment'=p_environment AND after_state_json->>'appid'=p_appid AND after_state_json->>'merchant_id'=p_merchant ORDER BY id DESC LIMIT 1;
 IF NOT FOUND OR receipt.result<>'SUCCEEDED' OR receipt.reason_code<>'BILL_MATCHED' THEN RETURN jsonb_build_object('outcome','SUCCEEDED','completed',false);END IF;
 RETURN jsonb_build_object('outcome','SUCCEEDED','completed',true,'matched',receipt.after_state_json->>'matched','digest',receipt.after_state_json->>'bill_digest');
END;$order_rpc$;
REVOKE ALL ON FUNCTION public.order_billing_completed_v1(text,text,text,text) FROM PUBLIC,anon,authenticated;
GRANT EXECUTE ON FUNCTION public.order_billing_completed_v1(text,text,text,text) TO authenticated;
