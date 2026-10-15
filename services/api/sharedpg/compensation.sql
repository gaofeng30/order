-- Durable wake state is private. Business callers cannot schedule arbitrary SQL
-- or choose a function URL. cron/function extensions are installed separately.
CREATE SCHEMA order_compensation;
REVOKE ALL ON SCHEMA order_compensation FROM PUBLIC,anon,authenticated;
CREATE TABLE order_compensation.wakes (
 kind text NOT NULL CHECK(kind IN ('payment','refund','subscription')),
 entity_id numeric(20,0) NOT NULL CHECK(entity_id BETWEEN 1 AND 18446744073709551615),
 PRIMARY KEY(kind,entity_id),
 next_at timestamptz,
 token_hash bytea CHECK(token_hash IS NULL OR octet_length(token_hash)=32),
 token_until timestamptz,
 request_id bigint,
 claimed boolean NOT NULL DEFAULT false,
 retry_after timestamptz
);
CREATE UNIQUE INDEX order_compensation_token_unique ON order_compensation.wakes(token_hash) WHERE token_hash IS NOT NULL;
REVOKE ALL ON ALL TABLES IN SCHEMA order_compensation FROM PUBLIC,anon,authenticated;

CREATE FUNCTION order_compensation.dispatch(p_kind text,p_id numeric) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $wake$
DECLARE w order_compensation.wakes%ROWTYPE; token text; queued_id bigint; now_at timestamptz:=clock_timestamp();
BEGIN
 SELECT * INTO w FROM order_compensation.wakes WHERE kind=p_kind AND entity_id=p_id FOR UPDATE;
 IF NOT FOUND THEN RETURN;END IF;
 -- A completed transport with this capability still outstanding means no
 -- durable business completion was confirmed. Start backoff at observation,
 -- never at request issuance (which could precede failure by the timeout).
 -- Successful Finish clears request_id or removes this row transactionally.
 IF w.request_id IS NOT NULL AND tencentdb_scf.tencentdb_scf_result(w.request_id) IS NOT NULL THEN
  UPDATE order_compensation.wakes SET request_id=NULL,token_hash=NULL,token_until=NULL,claimed=false,retry_after=clock_timestamp()+interval '30 minutes' WHERE kind=p_kind AND entity_id=p_id;
  RETURN;
 END IF;
 IF w.next_at IS NULL OR w.next_at>now_at OR w.token_until>now_at OR w.retry_after>now_at THEN RETURN; END IF;
 token:=encode(public.gen_random_bytes(32),'hex');
 BEGIN
  UPDATE order_compensation.wakes SET token_hash=sha256(decode(token,'hex')),token_until=now_at+interval '30 minutes',claimed=false,retry_after=NULL WHERE kind=p_kind AND entity_id=p_id;
  queued_id:=tencentdb_scf.tencentdb_scf_post('/v1/functions/order-compensation-wake','{}'::jsonb,jsonb_build_object('token',token)::text,false,30000);
  UPDATE order_compensation.wakes SET request_id=queued_id WHERE kind=p_kind AND entity_id=p_id;
 EXCEPTION WHEN OTHERS THEN
  -- Queue insertion and capability issuance roll back together. Keep a durable
  -- retry even when the transport extension cannot accept the request.
  UPDATE order_compensation.wakes SET request_id=NULL,token_hash=NULL,token_until=NULL,claimed=false,retry_after=clock_timestamp()+interval '30 minutes' WHERE kind=p_kind AND entity_id=p_id;
 END;
END $wake$;

CREATE FUNCTION order_compensation.refresh(p_kind text,p_id numeric) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $wake$
DECLARE w order_compensation.wakes%ROWTYPE; due_at timestamptz; name text;
BEGIN
 IF p_kind NOT IN ('payment','refund','subscription') OR p_id IS NULL OR p_id NOT BETWEEN 1 AND 18446744073709551615 THEN RAISE EXCEPTION 'invalid compensation target';END IF;
 name:='order-compensation-'||p_kind||'-'||p_id::text;
 IF p_kind='payment' THEN
  SELECT CASE WHEN materialization_state='READY' THEN coalesce(next_reconcile_at,clock_timestamp())
   WHEN materialization_state='AWAITING_PAYMENT' THEN greatest(next_reconcile_at,lease_expires_at) END
   INTO due_at FROM public.prepayments WHERE id=p_id;
 ELSIF p_kind='refund' THEN
  SELECT CASE WHEN provider_state='READY' THEN clock_timestamp() WHEN provider_state='CREATE_CLAIMED' THEN lease_expires_at
   WHEN provider_state IN ('CREATE_UNKNOWN','PROCESSING') THEN greatest(next_query_at,lease_expires_at) END
   INTO due_at FROM public.refunds WHERE id=p_id;
 ELSE
  SELECT CASE WHEN state='PENDING' THEN next_attempt_at WHEN state='IN_FLIGHT' THEN lease_expires_at END
   INTO due_at FROM public.notification_outbox WHERE id=p_id;
 END IF;
 IF due_at IS NOT NULL THEN
  INSERT INTO order_compensation.wakes(kind,entity_id) VALUES(p_kind,p_id) ON CONFLICT DO NOTHING;
 END IF;
 SELECT * INTO w FROM order_compensation.wakes WHERE kind=p_kind AND entity_id=p_id FOR UPDATE;
 IF NOT FOUND THEN RETURN;END IF;
 IF due_at IS NULL THEN
  DELETE FROM order_compensation.wakes WHERE kind=p_kind AND entity_id=p_id;
  IF EXISTS(SELECT 1 FROM cron.job WHERE jobname=name) THEN PERFORM cron.unschedule(name);END IF;
  RETURN;
 END IF;
 UPDATE order_compensation.wakes SET next_at=due_at,
  token_until=CASE WHEN w.next_at IS NULL OR due_at<w.next_at THEN NULL ELSE token_until END,
  retry_after=CASE WHEN w.next_at IS NULL OR due_at<w.next_at THEN NULL ELSE retry_after END
 WHERE kind=p_kind AND entity_id=p_id;
 IF NOT EXISTS(SELECT 1 FROM cron.job WHERE jobname=name) THEN
  -- This timer exists only while durable work exists. It reads this one wake,
  -- never periodically scans empty business tables. dispatch gates actual
  -- attempts by persisted due/retry times, with no early retry.
  PERFORM cron.schedule(name,'* * * * *',format('SELECT order_compensation.dispatch(%L,%s)',p_kind,p_id));
 END IF;
 PERFORM order_compensation.dispatch(p_kind,p_id);
END $wake$;

CREATE FUNCTION order_compensation.changed() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $wake$
BEGIN
 PERFORM order_compensation.refresh(TG_ARGV[0],CASE WHEN TG_OP='DELETE' THEN OLD.id ELSE NEW.id END);
 RETURN NULL;
END $wake$;
CREATE TRIGGER order_compensation_payment AFTER INSERT OR UPDATE OR DELETE ON public.prepayments FOR EACH ROW EXECUTE FUNCTION order_compensation.changed('payment');
CREATE TRIGGER order_compensation_refund AFTER INSERT OR UPDATE OR DELETE ON public.refunds FOR EACH ROW EXECUTE FUNCTION order_compensation.changed('refund');
CREATE TRIGGER order_compensation_subscription AFTER INSERT OR UPDATE OR DELETE ON public.notification_outbox FOR EACH ROW EXECUTE FUNCTION order_compensation.changed('subscription');

CREATE FUNCTION public.order_compensation_claim_v1(p_token text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $wake$
DECLARE claimed_kind text;claimed_id numeric;
BEGIN
 IF auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION 'ORDER_RUNTIME_FORBIDDEN' USING ERRCODE='42501';
 END IF;
 IF p_token IS NULL OR p_token !~ '^[0-9a-f]{64}$' THEN RETURN jsonb_build_object('outcome','REJECTED');END IF;
 UPDATE order_compensation.wakes SET claimed=true
 WHERE token_hash=sha256(decode(p_token,'hex')) AND NOT claimed AND token_until>clock_timestamp() AND next_at IS NOT NULL
 RETURNING kind,entity_id INTO claimed_kind,claimed_id;
 IF NOT FOUND THEN RETURN jsonb_build_object('outcome','REJECTED');END IF;
 RETURN jsonb_build_object('outcome','CLAIMED','kind',claimed_kind,'id',claimed_id::text);
END $wake$;
CREATE FUNCTION public.order_compensation_finish_v1(p_token text,p_success boolean) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $wake$
DECLARE w order_compensation.wakes%ROWTYPE;
BEGIN
 IF auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION 'ORDER_RUNTIME_FORBIDDEN' USING ERRCODE='42501';
 END IF;
 IF p_token IS NULL OR p_token !~ '^[0-9a-f]{64}$' OR p_success IS NULL THEN RETURN jsonb_build_object('outcome','REJECTED');END IF;
 SELECT * INTO w FROM order_compensation.wakes WHERE token_hash=sha256(decode(p_token,'hex')) AND claimed FOR UPDATE;
 -- A terminal business update may already have removed this wake, or newer
 -- work may have issued a new capability. Never overwrite that newer state.
 IF NOT FOUND THEN RETURN jsonb_build_object('outcome','FINISHED');END IF;
 UPDATE order_compensation.wakes SET request_id=NULL,token_hash=NULL,token_until=NULL,claimed=false,
  retry_after=CASE WHEN p_success THEN NULL ELSE clock_timestamp()+interval '30 minutes' END WHERE kind=w.kind AND entity_id=w.entity_id;
 PERFORM order_compensation.refresh(w.kind,w.entity_id);
 RETURN jsonb_build_object('outcome','FINISHED');
END $wake$;
REVOKE ALL ON FUNCTION public.order_compensation_finish_v1(text,boolean) FROM PUBLIC,anon;
GRANT EXECUTE ON FUNCTION public.order_compensation_finish_v1(text,boolean) TO authenticated;

REVOKE ALL ON ALL FUNCTIONS IN SCHEMA order_compensation FROM PUBLIC,anon,authenticated;
REVOKE ALL ON FUNCTION public.order_compensation_claim_v1(text) FROM PUBLIC,anon;
GRANT EXECUTE ON FUNCTION public.order_compensation_claim_v1(text) TO authenticated;
SELECT order_compensation.refresh('payment',id) FROM public.prepayments;
SELECT order_compensation.refresh('refund',id) FROM public.refunds;
SELECT order_compensation.refresh('subscription',id) FROM public.notification_outbox;
