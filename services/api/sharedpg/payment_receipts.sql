-- Fixed payment ABI; raw frozen aggregates stay inside each atomic RPC.

CREATE OR REPLACE FUNCTION public.order_payment_receipt_read_v1(p_user numeric,p_action text,p_key_hash text)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
<<payment_command>>
DECLARE audit_row public.action_audits%ROWTYPE;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF p_action NOT IN ('payment.prepare','payment.confirm','payment.materialize_pending') THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 SELECT * INTO audit_row FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='USER' AND actor_scope_hash=sha256(convert_to('USER','UTF8')||decode('00','hex')||int8send((CASE WHEN (p_user)::numeric>9223372036854775807 THEN (p_user)::numeric-18446744073709551616 ELSE (p_user)::numeric END)::bigint)) AND action=p_action AND operation_key_hash=decode(p_key_hash,'hex');
 IF NOT FOUND THEN RETURN jsonb_build_object('outcome','ABSENT');END IF;
 RETURN jsonb_build_object('outcome','FOUND','evidence_text',audit_row.before_state_json::text,'response_text',audit_row.response_json::text);

 END;
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_payment_receipt_append_v1(p_user numeric,p_action text,p_target numeric,p_key_hash text,p_request_hash text,p_response_json text,p_at timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
<<payment_command>>
DECLARE target_kind text;reason text;request_digest bytea;
 failure_state text;failure_constraint text;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
 RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';END IF;
 BEGIN
 IF p_action='payment.prepare' THEN target_kind:='QUOTE';reason:='PAYMENT_PREPARED';ELSIF p_action IN ('payment.confirm','payment.materialize_pending') THEN target_kind:='PREPAYMENT';reason:='PAYMENT_CONFIRMED';ELSE RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 request_digest:=sha256(convert_to(p_action,'UTF8')||decode('00','hex')||convert_to(p_target::text,'UTF8'));
 INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,action,target_type,target_id,operation_key_hash,request_id_hash,result,reason_code,before_state_json,response_json,occurred_at)
 VALUES('COMMAND_RECEIPT','USER',sha256(convert_to('USER','UTF8')||decode('00','hex')||int8send((CASE WHEN (p_user)::numeric>9223372036854775807 THEN (p_user)::numeric-18446744073709551616 ELSE (p_user)::numeric END)::bigint)),p_user,p_action,target_kind,p_target,decode(p_key_hash,'hex'),decode(p_request_hash,'hex'),'SUCCEEDED',reason,jsonb_build_object('request_digest',encode(request_digest,'hex')),p_response_json::jsonb,p_at);
 RETURN jsonb_build_object('outcome','APPENDED');
 EXCEPTION WHEN OTHERS THEN
 GET STACKED DIAGNOSTICS failure_state=RETURNED_SQLSTATE,failure_constraint=CONSTRAINT_NAME;
 IF failure_state IN ('40P01','40001','55P03') THEN RETURN jsonb_build_object('outcome','SQL_FAILURE','sqlstate',failure_state,'constraint','');END IF;
 IF failure_state='23505' AND failure_constraint IN ('uq_prepayments_quote','uq_prepayments_user_idempotency','uq_prepayments_out_trade_no','uq_action_audits_command_receipt') THEN RETURN jsonb_build_object('outcome','SQL_FAILURE','sqlstate',failure_state,'constraint',failure_constraint);END IF;
 IF failure_state='P1000' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 RAISE;
 END;
END
$order_rpc$;
