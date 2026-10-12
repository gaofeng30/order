-- Fixed production scan and per-order atomic transition; original Go time facts remain separate from pgx persistence.

CREATE OR REPLACE FUNCTION public.order_production_due_v1(p_now jsonb,p_limit integer)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE ids jsonb;cutoff_wire jsonb;cutoff_at timestamptz;micros numeric;failure_state text;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF p_limit IS NULL OR p_limit NOT BETWEEN 1 AND 100 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF jsonb_typeof(p_now) IS DISTINCT FROM 'object' OR NOT (p_now ?& ARRAY['epoch_seconds','nanosecond','zero','canonical']) OR (p_now-ARRAY['epoch_seconds','nanosecond','zero','canonical','cutoff'])<>'{}'::jsonb OR jsonb_typeof(p_now->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_now->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_now->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_now->'nanosecond') IS DISTINCT FROM 'number' OR (p_now->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_now->>'nanosecond')::numeric)<>(p_now->>'nanosecond')::numeric OR jsonb_typeof(p_now->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_now->'canonical') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF NOT (p_now ? 'cutoff') THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 -- Go has already performed UTC, Truncate(microsecond), then Add(30 minutes).
 cutoff_wire:=p_now->'cutoff';
 IF jsonb_typeof(cutoff_wire) IS DISTINCT FROM 'object' OR NOT (cutoff_wire ?& ARRAY['epoch_seconds','nanosecond','zero','canonical']) OR (cutoff_wire-ARRAY['epoch_seconds','nanosecond','zero','canonical'])<>'{}'::jsonb OR jsonb_typeof(cutoff_wire->'epoch_seconds') IS DISTINCT FROM 'string' OR (cutoff_wire->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (cutoff_wire->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(cutoff_wire->'nanosecond') IS DISTINCT FROM 'number' OR (cutoff_wire->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((cutoff_wire->>'nanosecond')::numeric)<>(cutoff_wire->>'nanosecond')::numeric OR jsonb_typeof(cutoff_wire->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(cutoff_wire->'canonical') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 micros:=(cutoff_wire->>'epoch_seconds')::numeric*1000000+div((cutoff_wire->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN cutoff_at:='infinity'::timestamptz;
 ELSIF micros=-9223372036854775808 THEN cutoff_at:='-infinity'::timestamptz;
 ELSE cutoff_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 SELECT coalesce(jsonb_agg(id::text ORDER BY pickup_at,id),'[]'::jsonb) INTO ids FROM (SELECT id,pickup_at FROM public.orders WHERE state='RESERVED' AND pickup_at<=cutoff_at ORDER BY pickup_at,id LIMIT p_limit) due;
 RETURN jsonb_build_object('outcome','FOUND','ids',ids);
 EXCEPTION WHEN OTHERS THEN
 GET STACKED DIAGNOSTICS failure_state=RETURNED_SQLSTATE;
 IF failure_state IN ('40P01','40001','55P03') THEN RETURN jsonb_build_object('outcome','SQL_FAILURE','sqlstate',failure_state);END IF;
 RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_production_advance_v1(p_order numeric,p_now jsonb)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE ob public.orders%ROWTYPE;affected bigint;after_json jsonb;now_at timestamptz;micros numeric;raw_nanos numeric;raw_seconds numeric;failure_state text;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 SELECT * INTO ob FROM public.orders WHERE id=p_order FOR UPDATE;
 IF NOT FOUND THEN RETURN jsonb_build_object('outcome','ADVANCED','advanced',false);END IF;
 -- The original Scan consumes all four columns before testing state.
 IF ob.state IS NULL OR ob.pickup_at IS NULL OR ob.materialized_at IS NULL OR ob.record_version IS NULL OR NOT isfinite(ob.pickup_at) OR NOT isfinite(ob.materialized_at) OR ob.record_version NOT BETWEEN 0 AND 18446744073709551615 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF ob.state<>'RESERVED' THEN RETURN jsonb_build_object('outcome','ADVANCED','advanced',false);END IF;
 IF jsonb_typeof(p_now) IS DISTINCT FROM 'object' OR NOT (p_now ?& ARRAY['epoch_seconds','nanosecond','zero','canonical']) OR (p_now-ARRAY['epoch_seconds','nanosecond','zero','canonical','after_json_hex'])<>'{}'::jsonb OR jsonb_typeof(p_now->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_now->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_now->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_now->'nanosecond') IS DISTINCT FROM 'number' OR (p_now->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_now->>'nanosecond')::numeric)<>(p_now->>'nanosecond')::numeric OR jsonb_typeof(p_now->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_now->'canonical') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 -- UTC strips monotonic data. Match Go's signed seconds since year 1 for Before/Advance,
 -- retaining original facts rather than the wrapped PG timestamp.
 raw_seconds:=(p_now->>'epoch_seconds')::numeric+62135596800;
 raw_seconds:=mod(mod(raw_seconds+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 raw_nanos:=raw_seconds*1000000000+(p_now->>'nanosecond')::numeric;
 IF (p_now->>'zero')::boolean OR ob.pickup_at='0001-01-01T00:00:00Z'::timestamptz OR raw_nanos<(extract(epoch FROM ob.materialized_at)+62135596800)*1000000000 OR ob.record_version=0 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF raw_nanos<(extract(epoch FROM ob.pickup_at)+62135596800-1800)*1000000000 THEN RETURN jsonb_build_object('outcome','ADVANCED','advanced',false);END IF;
 -- Only the original UPDATE bind now consumes the pgx signed microsecond value.
 micros:=(p_now->>'epoch_seconds')::numeric*1000000+div((p_now->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN now_at:='infinity'::timestamptz;
 ELSIF micros=-9223372036854775808 THEN now_at:='-infinity'::timestamptz;
 ELSE now_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 UPDATE public.orders SET state='PREPARING',preparing_at=now_at,record_version=record_version+1,updated_at=now_at WHERE id=p_order AND state='RESERVED' AND record_version=ob.record_version;
 GET DIAGNOSTICS affected=ROW_COUNT;IF affected<>1 THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_PRODUCTION_UNAVAILABLE';END IF;
 -- Closed after-material is consumed only at the original audit stage. An
 -- explicit JSON null retains json.Marshal's ignored error / SQL NULL.
 IF NOT (p_now ? 'after_json_hex') OR jsonb_typeof(p_now->'after_json_hex') NOT IN ('null','string') OR (jsonb_typeof(p_now->'after_json_hex')='string' AND ((p_now->>'after_json_hex') !~ '^([0-9a-f]{2})*$')) THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_PRODUCTION_UNAVAILABLE';END IF;
 IF p_now->>'after_json_hex' IS NULL THEN after_json:=NULL;ELSE after_json:=convert_from(decode(p_now->>'after_json_hex','hex'),'UTF8')::jsonb;END IF;
 INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,action,target_type,target_id,operation_key_hash,result,reason_code,before_state_json,after_state_json,occurred_at)
 VALUES('SYSTEM_EVIDENCE','SYSTEM',sha256(convert_to('SYSTEM:ORDER_PRODUCTION','UTF8')),NULL,NULL,'order.production_due','ORDER',p_order,NULL,'SUCCEEDED','PRODUCTION_STARTED',jsonb_build_object('state','RESERVED'),after_json,now_at);
 RETURN jsonb_build_object('outcome','ADVANCED','advanced',true);
 EXCEPTION WHEN OTHERS THEN
 GET STACKED DIAGNOSTICS failure_state=RETURNED_SQLSTATE;
 IF failure_state IN ('40P01','40001','55P03') THEN RETURN jsonb_build_object('outcome','SQL_FAILURE','sqlstate',failure_state);END IF;
 RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END;
END
$order_rpc$;
