-- Only expiry metadata is stored here; the extension encrypts the credential.
CREATE TABLE order_compensation.authentication (
 singleton boolean PRIMARY KEY CHECK(singleton),
 expires_at timestamptz NOT NULL
);
REVOKE ALL ON order_compensation.authentication FROM PUBLIC,anon,authenticated;
INSERT INTO order_compensation.authentication VALUES(true,'epoch');

CREATE FUNCTION public.order_compensation_authenticate_v1() RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $wake$
DECLARE claims jsonb:=auth.jwt(); token text; payload text; expires timestamptz; previous timestamptz;
BEGIN
 IF auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__'
  OR claims->>'aud' IS DISTINCT FROM 'order-prod-d0g4xnw49c5ada5ce' THEN
  RAISE EXCEPTION 'ORDER_RUNTIME_FORBIDDEN' USING ERRCODE='42501';
 END IF;
 -- Use the bearer already authenticated by the gateway, never caller-supplied
 -- parameters or a management key. Nothing secret is returned or logged.
 token:=nullif(current_setting('request.headers',true),'')::jsonb->>'authorization';
 IF token IS NULL OR token !~ '^Bearer [A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$' OR length(token)>4103 THEN
  RAISE EXCEPTION 'ORDER_RUNTIME_AUTH_INVALID';
 END IF;
 token:=substr(token,8);
 payload:=split_part(token,'.',2);
 IF convert_from(decode(translate(payload,'-_','+/')||repeat('=',(4-length(payload)%4)%4),'base64'),'UTF8')::jsonb IS DISTINCT FROM claims THEN
  RAISE EXCEPTION 'ORDER_RUNTIME_AUTH_INVALID';
 END IF;
 expires:=to_timestamp((claims->>'exp')::double precision);
 IF expires IS NULL OR expires<=clock_timestamp()+interval '15 seconds' THEN RAISE EXCEPTION 'ORDER_RUNTIME_AUTH_EXPIRED';END IF;
 SELECT expires_at INTO STRICT previous FROM order_compensation.authentication WHERE singleton FOR UPDATE;
 -- Concurrent instances may complete logins in a different order. Never
 -- replace a longer-lived credential with one expiring sooner.
 IF expires>previous THEN
  IF NOT tencentdb_scf.set_cloudbase_env_id('order-prod-d0g4xnw49c5ada5ce') OR NOT tencentdb_scf.set_cloudbase_api_key(token) THEN
   RAISE EXCEPTION 'ORDER_RUNTIME_AUTH_UNAVAILABLE';
  END IF;
  UPDATE order_compensation.authentication SET expires_at=expires WHERE singleton;
 END IF;
 RETURN jsonb_build_object('outcome','REGISTERED');
END $wake$;
REVOKE ALL ON FUNCTION public.order_compensation_authenticate_v1() FROM PUBLIC,anon;
GRANT EXECUTE ON FUNCTION public.order_compensation_authenticate_v1() TO authenticated;
