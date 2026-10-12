-- Fixed subscription transactions. Provider I/O remains outside these functions.

CREATE OR REPLACE FUNCTION public.order_subscription_consent_read_v1(p_user numeric,p_key_hash text)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE cr public.notification_consents%ROWTYPE;failure_state text;failure_constraint text;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 SELECT * INTO cr FROM public.notification_consents WHERE user_id=p_user AND idempotency_key_hash=decode(p_key_hash,'hex');
 IF NOT FOUND THEN RETURN jsonb_build_object('outcome','NOT_FOUND');END IF;
 IF cr.order_id NOT BETWEEN 0 AND 18446744073709551615 OR cr.grant_sequence NOT BETWEEN 0 AND 18446744073709551615 OR cr.template_config_version NOT BETWEEN 0 AND 18446744073709551615 OR NOT isfinite(cr.decided_at) THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 RETURN jsonb_build_object('outcome','FOUND','consent',jsonb_build_object('order',cr.order_id::text,'kind',cr.kind,'decision',cr.decision,'sequence',cr.grant_sequence::text,'template',cr.template_config_version::text,'decided_seconds',floor(extract(epoch FROM cr.decided_at))::text,'decided_nanosecond',((extract(epoch FROM cr.decided_at)-floor(extract(epoch FROM cr.decided_at)))*1000000000)::integer));
 EXCEPTION WHEN OTHERS THEN
 GET STACKED DIAGNOSTICS failure_state=RETURNED_SQLSTATE,failure_constraint=CONSTRAINT_NAME;
 IF failure_state IN ('40P01','40001','55P03') OR (failure_state='23505' AND failure_constraint IN ('uq_notification_consents_user_idempotency','uq_action_audits_command_receipt')) THEN
 RETURN jsonb_build_object('outcome','SQL_FAILURE','sqlstate',failure_state,'constraint',CASE WHEN failure_state='23505' THEN failure_constraint ELSE '' END);END IF;
 RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_subscription_consent_record_v1(p_user numeric,p_order numeric,p_kind text,p_decision text,p_template_version numeric,p_key_hash text,p_request_hash text,p_now jsonb)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE owner_id numeric;latest numeric;now_at timestamptz;micros numeric;response jsonb;scope bytea;failure_state text;failure_constraint text;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 SELECT user_id INTO owner_id FROM public.orders WHERE id=p_order FOR UPDATE;
 IF NOT FOUND OR owner_id<>p_user THEN RETURN jsonb_build_object('outcome','NOT_FOUND');END IF;
 IF owner_id NOT BETWEEN 0 AND 18446744073709551615 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 SELECT grant_sequence INTO latest FROM public.notification_consents WHERE order_id=p_order AND kind=p_kind ORDER BY grant_sequence DESC LIMIT 1 FOR UPDATE;
 IF NOT FOUND THEN latest:=0;END IF;
 IF latest NOT BETWEEN 0 AND 18446744073709551614 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 -- Validate the closed Go Time4 only at this original bind consumption.
 IF jsonb_typeof(p_now) IS DISTINCT FROM 'object' OR NOT (p_now ?& ARRAY['epoch_seconds','nanosecond','zero','canonical']) OR (p_now-ARRAY['epoch_seconds','nanosecond','zero','canonical'])<>'{}'::jsonb OR jsonb_typeof(p_now->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_now->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_now->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_now->'nanosecond') IS DISTINCT FROM 'number' OR (p_now->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_now->>'nanosecond')::numeric)<>(p_now->>'nanosecond')::numeric OR jsonb_typeof(p_now->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_now->'canonical') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 -- Match pgx binary timestamptz's signed int64 microseconds, including sentinel values.
 micros:=((p_now->>'epoch_seconds')::numeric+0)*1000000+div((p_now->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN now_at:='infinity'::timestamptz;
 ELSIF micros=-9223372036854775808 THEN now_at:='-infinity'::timestamptz;
 ELSE now_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 INSERT INTO public.notification_consents(order_id,user_id,kind,grant_sequence,decision,template_config_version,idempotency_key_hash,decided_at,consumed_at)
 VALUES(p_order,p_user,p_kind,latest+1,p_decision,p_template_version,decode(p_key_hash,'hex'),now_at,NULL);
 response:=jsonb_build_object('subscription',jsonb_build_object('kind',p_kind,'decision',p_decision,'available',p_decision='ACCEPTED'));
 scope:=sha256(decode('5553455200','hex')||int8send((CASE WHEN p_user>9223372036854775807 THEN p_user-18446744073709551616 ELSE p_user END)::bigint));
 INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,action,target_type,target_id,operation_key_hash,request_id_hash,result,reason_code,after_state_json,response_json,occurred_at)
 VALUES('COMMAND_RECEIPT','USER',scope,p_user,'subscription.record_consent','ORDER',p_order,decode(p_key_hash,'hex'),decode(p_request_hash,'hex'),'SUCCEEDED','CONSENT_RECORDED',response,response,now_at);
 RETURN jsonb_build_object('outcome','RECORDED','consent',jsonb_build_object('sequence',(latest+1)::text));
 EXCEPTION WHEN OTHERS THEN
 GET STACKED DIAGNOSTICS failure_state=RETURNED_SQLSTATE,failure_constraint=CONSTRAINT_NAME;
 IF failure_state IN ('40P01','40001','55P03') OR (failure_state='23505' AND failure_constraint IN ('uq_notification_consents_user_idempotency','uq_action_audits_command_receipt')) THEN
 RETURN jsonb_build_object('outcome','SQL_FAILURE','sqlstate',failure_state,'constraint',CASE WHEN failure_state='23505' THEN failure_constraint ELSE '' END);END IF;
 RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_subscription_claim_v1(p_cutoff jsonb,p_lease_now jsonb,p_owner_hex text)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE ob public.notification_outbox%ROWTYPE;cutoff_at timestamptz;lease_at timestamptz;expiry_at timestamptz;micros numeric;field record;name text;message jsonb;value text;affected bigint;failure_state text;failure_constraint text;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 -- Validate the closed Go Time4 only at this original bind consumption.
 IF jsonb_typeof(p_cutoff) IS DISTINCT FROM 'object' OR NOT (p_cutoff ?& ARRAY['epoch_seconds','nanosecond','zero','canonical']) OR (p_cutoff-ARRAY['epoch_seconds','nanosecond','zero','canonical'])<>'{}'::jsonb OR jsonb_typeof(p_cutoff->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_cutoff->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_cutoff->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_cutoff->'nanosecond') IS DISTINCT FROM 'number' OR (p_cutoff->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_cutoff->>'nanosecond')::numeric)<>(p_cutoff->>'nanosecond')::numeric OR jsonb_typeof(p_cutoff->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_cutoff->'canonical') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 -- Match pgx binary timestamptz's signed int64 microseconds, including sentinel values.
 micros:=((p_cutoff->>'epoch_seconds')::numeric+0)*1000000+div((p_cutoff->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN cutoff_at:='infinity'::timestamptz;
 ELSIF micros=-9223372036854775808 THEN cutoff_at:='-infinity'::timestamptz;
 ELSE cutoff_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 -- Validate the closed Go Time4 only at this original bind consumption.
 IF jsonb_typeof(p_lease_now) IS DISTINCT FROM 'object' OR NOT (p_lease_now ?& ARRAY['epoch_seconds','nanosecond','zero','canonical']) OR (p_lease_now-ARRAY['epoch_seconds','nanosecond','zero','canonical'])<>'{}'::jsonb OR jsonb_typeof(p_lease_now->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_lease_now->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_lease_now->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_lease_now->'nanosecond') IS DISTINCT FROM 'number' OR (p_lease_now->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_lease_now->>'nanosecond')::numeric)<>(p_lease_now->>'nanosecond')::numeric OR jsonb_typeof(p_lease_now->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_lease_now->'canonical') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 -- Match pgx binary timestamptz's signed int64 microseconds, including sentinel values.
 micros:=((p_lease_now->>'epoch_seconds')::numeric+0)*1000000+div((p_lease_now->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN lease_at:='infinity'::timestamptz;
 ELSIF micros=-9223372036854775808 THEN lease_at:='-infinity'::timestamptz;
 ELSE lease_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 SELECT * INTO ob FROM public.notification_outbox WHERE (state='PENDING' AND next_attempt_at<=cutoff_at) OR (state='IN_FLIGHT' AND lease_expires_at<=lease_at) ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED;
 IF NOT FOUND THEN RETURN jsonb_build_object('outcome','EMPTY');END IF;
 IF ob.id NOT BETWEEN 0 AND 18446744073709551615 OR ob.order_id NOT BETWEEN 0 AND 18446744073709551615 OR ob.recipient_user_id NOT BETWEEN 0 AND 18446744073709551615 OR ob.template_config_version NOT BETWEEN 0 AND 18446744073709551615 OR ob.record_version NOT BETWEEN 0 AND 18446744073709551615 OR ob.attempt_count NOT BETWEEN 0 AND 65534 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 -- encoding/json accepts unknown members and folds ASCII names (plus Kelvin K and long S).
 message:='{}'::jsonb;
 FOR field IN SELECT fields.key,fields.value,fields.ordinality FROM json_each(ob.immutable_message_json::text::json) WITH ORDINALITY AS fields(key,value,ordinality) ORDER BY ordinality LOOP
 name:=translate(field.key,'ABCDEFGHIJKLMNOPQRSTUVWXYZſK','abcdefghijklmnopqrstuvwxyzsk');
 IF name NOT IN ('order_number','pickup_date','pickup_time','pickup_point','refund_result') THEN CONTINUE;END IF;
 IF json_typeof(field.value)='null' THEN CONTINUE;END IF;
 IF json_typeof(field.value)<>'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 message:=message||jsonb_build_object(name,field.value::jsonb);
 END LOOP;
 value:=coalesce(message->>'order_number','');
 IF octet_length(value) NOT BETWEEN 1 AND 64 OR value<>btrim(value,E' \t\n\r\f'||chr(11)||chr(133)||chr(160)||chr(5760)||chr(8192)||chr(8193)||chr(8194)||chr(8195)||chr(8196)||chr(8197)||chr(8198)||chr(8199)||chr(8200)||chr(8201)||chr(8202)||chr(8232)||chr(8233)||chr(8239)||chr(8287)||chr(12288)) THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF ob.kind='READY' THEN
 value:=coalesce(message->>'pickup_point','');
 IF octet_length(value) NOT BETWEEN 1 AND 256 OR value<>btrim(value,E' \t\n\r\f'||chr(11)||chr(133)||chr(160)||chr(5760)||chr(8192)||chr(8193)||chr(8194)||chr(8195)||chr(8196)||chr(8197)||chr(8198)||chr(8199)||chr(8200)||chr(8201)||chr(8202)||chr(8232)||chr(8233)||chr(8239)||chr(8287)||chr(12288)) OR coalesce(message->>'refund_result','')<>'' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 value:=coalesce(message->>'pickup_date','');
 IF value !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' OR substring(value,6,2)::integer NOT BETWEEN 1 AND 12 OR substring(value,9,2)::integer<1 OR substring(value,9,2)::integer>(CASE substring(value,6,2)::integer WHEN 2 THEN CASE WHEN mod(substring(value,1,4)::integer,400)=0 OR (mod(substring(value,1,4)::integer,4)=0 AND mod(substring(value,1,4)::integer,100)<>0) THEN 29 ELSE 28 END WHEN 4 THEN 30 WHEN 6 THEN 30 WHEN 9 THEN 30 WHEN 11 THEN 30 ELSE 31 END) THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF coalesce(message->>'pickup_time','') !~ '^([01][0-9]|2[0-3]):[0-5][0-9]$' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 ELSIF ob.kind='REFUND_RESULT' THEN
 IF coalesce(message->>'refund_result','')<>'REFUNDED' OR coalesce(message->>'pickup_date','')<>'' OR coalesce(message->>'pickup_time','')<>'' OR coalesce(message->>'pickup_point','')<>'' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 ELSE RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 -- Match pgx binary timestamptz's signed int64 microseconds, including sentinel values.
 micros:=((p_lease_now->>'epoch_seconds')::numeric+30)*1000000+div((p_lease_now->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN expiry_at:='infinity'::timestamptz;
 ELSIF micros=-9223372036854775808 THEN expiry_at:='-infinity'::timestamptz;
 ELSE expiry_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 UPDATE public.notification_outbox SET state='IN_FLIGHT',attempt_count=attempt_count+1,next_attempt_at=NULL,lease_owner=decode(p_owner_hex,'hex'),lease_expires_at=expiry_at,record_version=record_version+1,last_error_code=NULL
 WHERE id=ob.id AND record_version=ob.record_version AND ((state='PENDING' AND next_attempt_at<=cutoff_at) OR (state='IN_FLIGHT' AND lease_expires_at<=lease_at));
 GET DIAGNOSTICS affected=ROW_COUNT;IF affected<>1 THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_SUBSCRIPTION_UNAVAILABLE';END IF;
 RETURN jsonb_build_object('outcome','CLAIMED','delivery',jsonb_build_object('id',ob.id::text,'order',ob.order_id::text,'recipient',ob.recipient_user_id::text,'kind',ob.kind,'message',ob.immutable_message_json::text,'template',ob.template_config_version::text,'attempts',ob.attempt_count+1,'version',(ob.record_version+1)::text));
 EXCEPTION WHEN OTHERS THEN
 GET STACKED DIAGNOSTICS failure_state=RETURNED_SQLSTATE,failure_constraint=CONSTRAINT_NAME;
 IF failure_state IN ('40P01','40001','55P03') OR (failure_state='23505' AND failure_constraint IN ('uq_notification_consents_user_idempotency','uq_action_audits_command_receipt')) THEN
 RETURN jsonb_build_object('outcome','SQL_FAILURE','sqlstate',failure_state,'constraint',CASE WHEN failure_state='23505' THEN failure_constraint ELSE '' END);END IF;
 RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_subscription_sent_v1(p_outbox numeric,p_owner_hex text,p_version numeric,p_provider_message_hex text,p_completed_at jsonb)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE completed_at timestamptz;next_at timestamptz;micros numeric;affected bigint;failure_state text;failure_constraint text;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 -- Validate the closed Go Time4 only at this original bind consumption.
 IF jsonb_typeof(p_completed_at) IS DISTINCT FROM 'object' OR NOT (p_completed_at ?& ARRAY['epoch_seconds','nanosecond','zero','canonical']) OR (p_completed_at-ARRAY['epoch_seconds','nanosecond','zero','canonical'])<>'{}'::jsonb OR jsonb_typeof(p_completed_at->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_completed_at->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_completed_at->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_completed_at->'nanosecond') IS DISTINCT FROM 'number' OR (p_completed_at->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_completed_at->>'nanosecond')::numeric)<>(p_completed_at->>'nanosecond')::numeric OR jsonb_typeof(p_completed_at->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_completed_at->'canonical') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 -- Match pgx binary timestamptz's signed int64 microseconds, including sentinel values.
 micros:=((p_completed_at->>'epoch_seconds')::numeric+0)*1000000+div((p_completed_at->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN completed_at:='infinity'::timestamptz;
 ELSIF micros=-9223372036854775808 THEN completed_at:='-infinity'::timestamptz;
 ELSE completed_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 UPDATE public.notification_outbox SET state='SENT',next_attempt_at=NULL,provider_message_id=decode(p_provider_message_hex,'hex'),last_error_code=NULL,sent_at=completed_at,lease_owner=NULL,lease_expires_at=NULL,record_version=record_version+1 WHERE id=p_outbox AND state='IN_FLIGHT' AND lease_owner=decode(p_owner_hex,'hex') AND record_version=p_version AND lease_expires_at>completed_at;
 GET DIAGNOSTICS affected=ROW_COUNT;IF affected<>1 THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_SUBSCRIPTION_UNAVAILABLE';END IF;
 RETURN jsonb_build_object('outcome','FINISHED');
 EXCEPTION WHEN OTHERS THEN
 GET STACKED DIAGNOSTICS failure_state=RETURNED_SQLSTATE,failure_constraint=CONSTRAINT_NAME;
 IF failure_state IN ('40P01','40001','55P03') OR (failure_state='23505' AND failure_constraint IN ('uq_notification_consents_user_idempotency','uq_action_audits_command_receipt')) THEN
 RETURN jsonb_build_object('outcome','SQL_FAILURE','sqlstate',failure_state,'constraint',CASE WHEN failure_state='23505' THEN failure_constraint ELSE '' END);END IF;
 RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_subscription_temporary_failure_v1(p_outbox numeric,p_owner_hex text,p_version numeric,p_code text,p_completed_at jsonb,p_next_at jsonb)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE completed_at timestamptz;next_at timestamptz;micros numeric;affected bigint;failure_state text;failure_constraint text;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 -- Validate the closed Go Time4 only at this original bind consumption.
 IF jsonb_typeof(p_completed_at) IS DISTINCT FROM 'object' OR NOT (p_completed_at ?& ARRAY['epoch_seconds','nanosecond','zero','canonical']) OR (p_completed_at-ARRAY['epoch_seconds','nanosecond','zero','canonical'])<>'{}'::jsonb OR jsonb_typeof(p_completed_at->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_completed_at->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_completed_at->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_completed_at->'nanosecond') IS DISTINCT FROM 'number' OR (p_completed_at->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_completed_at->>'nanosecond')::numeric)<>(p_completed_at->>'nanosecond')::numeric OR jsonb_typeof(p_completed_at->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_completed_at->'canonical') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 -- Match pgx binary timestamptz's signed int64 microseconds, including sentinel values.
 micros:=((p_completed_at->>'epoch_seconds')::numeric+0)*1000000+div((p_completed_at->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN completed_at:='infinity'::timestamptz;
 ELSIF micros=-9223372036854775808 THEN completed_at:='-infinity'::timestamptz;
 ELSE completed_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 -- Validate the closed Go Time4 only at this original bind consumption.
 IF jsonb_typeof(p_next_at) IS DISTINCT FROM 'object' OR NOT (p_next_at ?& ARRAY['epoch_seconds','nanosecond','zero','canonical']) OR (p_next_at-ARRAY['epoch_seconds','nanosecond','zero','canonical'])<>'{}'::jsonb OR jsonb_typeof(p_next_at->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_next_at->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_next_at->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_next_at->'nanosecond') IS DISTINCT FROM 'number' OR (p_next_at->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_next_at->>'nanosecond')::numeric)<>(p_next_at->>'nanosecond')::numeric OR jsonb_typeof(p_next_at->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_next_at->'canonical') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 -- Match pgx binary timestamptz's signed int64 microseconds, including sentinel values.
 micros:=((p_next_at->>'epoch_seconds')::numeric+0)*1000000+div((p_next_at->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN next_at:='infinity'::timestamptz;
 ELSIF micros=-9223372036854775808 THEN next_at:='-infinity'::timestamptz;
 ELSE next_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 UPDATE public.notification_outbox SET state='PENDING',next_attempt_at=next_at,provider_message_id=NULL,last_error_code=p_code,sent_at=NULL,lease_owner=NULL,lease_expires_at=NULL,record_version=record_version+1 WHERE id=p_outbox AND state='IN_FLIGHT' AND lease_owner=decode(p_owner_hex,'hex') AND record_version=p_version AND lease_expires_at>completed_at;
 GET DIAGNOSTICS affected=ROW_COUNT;IF affected<>1 THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_SUBSCRIPTION_UNAVAILABLE';END IF;
 RETURN jsonb_build_object('outcome','FINISHED');
 EXCEPTION WHEN OTHERS THEN
 GET STACKED DIAGNOSTICS failure_state=RETURNED_SQLSTATE,failure_constraint=CONSTRAINT_NAME;
 IF failure_state IN ('40P01','40001','55P03') OR (failure_state='23505' AND failure_constraint IN ('uq_notification_consents_user_idempotency','uq_action_audits_command_receipt')) THEN
 RETURN jsonb_build_object('outcome','SQL_FAILURE','sqlstate',failure_state,'constraint',CASE WHEN failure_state='23505' THEN failure_constraint ELSE '' END);END IF;
 RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_subscription_permanent_failure_v1(p_outbox numeric,p_owner_hex text,p_version numeric,p_code text,p_completed_at jsonb)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE completed_at timestamptz;next_at timestamptz;micros numeric;affected bigint;failure_state text;failure_constraint text;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 -- Validate the closed Go Time4 only at this original bind consumption.
 IF jsonb_typeof(p_completed_at) IS DISTINCT FROM 'object' OR NOT (p_completed_at ?& ARRAY['epoch_seconds','nanosecond','zero','canonical']) OR (p_completed_at-ARRAY['epoch_seconds','nanosecond','zero','canonical'])<>'{}'::jsonb OR jsonb_typeof(p_completed_at->'epoch_seconds') IS DISTINCT FROM 'string' OR (p_completed_at->>'epoch_seconds') !~ '^(0|[1-9][0-9]*|-[1-9][0-9]*)$' OR (p_completed_at->>'epoch_seconds')::numeric NOT BETWEEN -9223372036854775808 AND 9223372036854775807 OR jsonb_typeof(p_completed_at->'nanosecond') IS DISTINCT FROM 'number' OR (p_completed_at->>'nanosecond')::numeric NOT BETWEEN 0 AND 999999999 OR trunc((p_completed_at->>'nanosecond')::numeric)<>(p_completed_at->>'nanosecond')::numeric OR jsonb_typeof(p_completed_at->'zero') IS DISTINCT FROM 'boolean' OR jsonb_typeof(p_completed_at->'canonical') IS DISTINCT FROM 'string' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 -- Match pgx binary timestamptz's signed int64 microseconds, including sentinel values.
 micros:=((p_completed_at->>'epoch_seconds')::numeric+0)*1000000+div((p_completed_at->>'nanosecond')::numeric,1000)-946684800000000;
 micros:=mod(mod(micros+9223372036854775808,18446744073709551616)+18446744073709551616,18446744073709551616)-9223372036854775808;
 IF micros=9223372036854775807 THEN completed_at:='infinity'::timestamptz;
 ELSIF micros=-9223372036854775808 THEN completed_at:='-infinity'::timestamptz;
 ELSE completed_at:='2000-01-01T00:00:00Z'::timestamptz+make_interval(days=>div(micros,86400000000)::integer)+(mod(micros,86400000000)::text||' microseconds')::interval;END IF;
 UPDATE public.notification_outbox SET state='FAILED_PERMANENT',next_attempt_at=NULL,provider_message_id=NULL,last_error_code=p_code,sent_at=NULL,lease_owner=NULL,lease_expires_at=NULL,record_version=record_version+1 WHERE id=p_outbox AND state='IN_FLIGHT' AND lease_owner=decode(p_owner_hex,'hex') AND record_version=p_version AND lease_expires_at>completed_at;
 GET DIAGNOSTICS affected=ROW_COUNT;IF affected<>1 THEN RAISE EXCEPTION USING ERRCODE='P1000',MESSAGE='ORDER_SUBSCRIPTION_UNAVAILABLE';END IF;
 RETURN jsonb_build_object('outcome','FINISHED');
 EXCEPTION WHEN OTHERS THEN
 GET STACKED DIAGNOSTICS failure_state=RETURNED_SQLSTATE,failure_constraint=CONSTRAINT_NAME;
 IF failure_state IN ('40P01','40001','55P03') OR (failure_state='23505' AND failure_constraint IN ('uq_notification_consents_user_idempotency','uq_action_audits_command_receipt')) THEN
 RETURN jsonb_build_object('outcome','SQL_FAILURE','sqlstate',failure_state,'constraint',CASE WHEN failure_state='23505' THEN failure_constraint ELSE '' END);END IF;
 RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END;
END
$order_rpc$;
