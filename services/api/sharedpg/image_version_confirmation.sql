-- Confirmed immutable objects and existing UAT native-version pins share durable command receipts.
-- No business table, historical migration or runtime management permission is added.
CREATE OR REPLACE FUNCTION public.order_image_versions_v1(p_object_keys text[])
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_key text;v_target bytea;v_pin jsonb;v_candidate jsonb;v_published text;v_version text;
 v_versions jsonb:='[]'::jsonb;v_pins jsonb:='{}'::jsonb;v_targets bytea[];r public.action_audits%ROWTYPE;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 IF p_object_keys IS NULL OR cardinality(p_object_keys) NOT BETWEEN 1 AND 128
  OR array_ndims(p_object_keys) IS DISTINCT FROM 1
  OR EXISTS(SELECT 1 FROM unnest(p_object_keys) k WHERE k IS NULL OR k !~ '^images/([0-9a-f]{64}/)?[0-9a-f]{64}[.](png|jpg)$') THEN
  RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END IF;
 -- Fetch all matching receipts in one statement; validate every match as before.
 SELECT array_agg(sha256(convert_to('target:'||key,'UTF8'))) INTO v_targets FROM unnest(p_object_keys) key;
 FOR r IN SELECT * FROM public.action_audits
  WHERE entry_kind='COMMAND_RECEIPT' AND action='object.image.confirm'
   AND (target_key_hash=ANY(v_targets) OR response_json->>'object_key'=ANY(p_object_keys))
  ORDER BY id LOOP
  v_key:=r.response_json->>'object_key';
  IF v_key IS NULL OR NOT (v_key=ANY(p_object_keys)) THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  v_target:=sha256(convert_to('target:'||v_key,'UTF8'));
  v_pin:=v_pins->v_key;
   IF r.actor_kind IS DISTINCT FROM 'MERCHANT' OR r.actor_user_id IS NULL OR r.actor_user_id<=0
    OR r.actor_account_id IS NULL OR r.actor_account_id<=0 OR r.actor_account_id_snapshot IS DISTINCT FROM r.actor_account_id
    OR r.actor_role_snapshot IS DISTINCT FROM 'OWNER' OR r.actor_auth_version_snapshot IS NULL OR r.actor_auth_version_snapshot<=0
    OR r.actor_scope_hash IS DISTINCT FROM sha256(convert_to('merchant:'||r.actor_user_id::text||':'||r.actor_account_id::text,'UTF8'))
    OR r.target_type IS DISTINCT FROM 'IMAGE_OBJECT' OR r.target_id IS NOT NULL OR r.target_key_hash IS DISTINCT FROM v_target
    OR r.result IS DISTINCT FROM 'SUCCEEDED' OR r.reason_code IS DISTINCT FROM 'OK'
    OR jsonb_typeof(r.before_state_json) IS DISTINCT FROM 'object'
    OR jsonb_typeof(r.before_state_json->'request_digest') IS DISTINCT FROM 'string'
    OR r.before_state_json->>'request_digest' !~ '^[0-9a-f]{64}$'
    OR jsonb_typeof(r.response_json) IS DISTINCT FROM 'object'
    OR jsonb_typeof(r.response_json->'object_key') IS DISTINCT FROM 'string'
    OR r.response_json->>'object_key' IS DISTINCT FROM v_key
    OR jsonb_typeof(r.response_json->'published_key') IS DISTINCT FROM 'string'
    OR jsonb_typeof(r.response_json->'version_id') IS DISTINCT FROM 'string' THEN
    RETURN jsonb_build_object('outcome','UNAVAILABLE');
   END IF;
   v_published:=r.response_json->>'published_key';v_version:=r.response_json->>'version_id';
   IF NOT (v_published=v_key AND v_version='') AND (v_published !~ '^images/published/[0-9a-f]{64}/[0-9a-f]{64}[.](png|jpg)$'
    OR substring(v_published FROM '[^/]+$') IS DISTINCT FROM substring(v_key FROM '[^/]+$')
    OR v_version='' OR v_version='null' OR v_version ~ '[[:space:][:cntrl:]]'
    OR v_version ~ U&'[\007f-\009f\00a0\1680\2000-\200a\2028\2029\202f\205f\3000]') THEN
    RETURN jsonb_build_object('outcome','UNAVAILABLE');
   END IF;
   v_candidate:=jsonb_build_object('object_key',v_key,'published_key',v_published,'version_id',v_version);
   IF v_pin IS NOT NULL AND v_pin IS DISTINCT FROM v_candidate THEN
    RETURN jsonb_build_object('outcome','UNAVAILABLE');
   END IF;
  v_pins:=jsonb_set(v_pins,ARRAY[v_key],v_candidate,true);
 END LOOP;
 SELECT COALESCE(jsonb_agg(value ORDER BY key),'[]'::jsonb) INTO v_versions FROM jsonb_each(v_pins);
 RETURN jsonb_build_object('outcome','SUCCEEDED','versions',v_versions);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_image_confirmation_replay_v1(p_actor numeric,p_key_hash text)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_account numeric;v_auth numeric;v_scope bytea;v_pin jsonb;v_versions jsonb;v_lock_timeout text;
 r public.action_audits%ROWTYPE;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 IF p_actor IS NULL OR p_actor<1 OR p_actor>18446744073709551615 OR p_actor<>trunc(p_actor)
  OR p_key_hash IS NULL OR p_key_hash !~ '^[0-9a-f]{64}$' THEN
  RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END IF;
 v_lock_timeout:=current_setting('lock_timeout');
 PERFORM set_config('lock_timeout','5s',true);
 BEGIN
  SELECT id,auth_version INTO v_account,v_auth FROM public.merchant_accounts
   WHERE bound_user_id=p_actor AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER' FOR UPDATE;
 EXCEPTION WHEN OTHERS THEN
  PERFORM set_config('lock_timeout',v_lock_timeout,true);
  RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END;
 PERFORM set_config('lock_timeout',v_lock_timeout,true);
 IF v_account IS NULL OR v_account<=0 OR v_auth IS NULL OR v_auth<=0 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 v_scope:=sha256(convert_to('merchant:'||trunc(p_actor)::text||':'||v_account::text,'UTF8'));
 SELECT * INTO r FROM public.action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_scope_hash=v_scope
  AND action='object.image.confirm' AND operation_key_hash=decode(p_key_hash,'hex');
 IF NOT FOUND THEN RETURN jsonb_build_object('outcome','ABSENT');END IF;
 IF jsonb_typeof(r.before_state_json) IS DISTINCT FROM 'object'
  OR jsonb_typeof(r.before_state_json->'request_digest') IS DISTINCT FROM 'string'
  OR r.before_state_json->>'request_digest' !~ '^[0-9a-f]{64}$'
  OR jsonb_typeof(r.response_json) IS DISTINCT FROM 'object'
  OR jsonb_typeof(r.response_json->'object_key') IS DISTINCT FROM 'string' THEN
  RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END IF;
 v_versions:=public.order_image_versions_v1(ARRAY[r.response_json->>'object_key']);
 IF v_versions->>'outcome' IS DISTINCT FROM 'SUCCEEDED' OR jsonb_array_length(v_versions->'versions') IS DISTINCT FROM 1 THEN
  RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END IF;
 v_pin:=v_versions->'versions'->0;
 RETURN jsonb_build_object('outcome','SUCCEEDED','request_digest',r.before_state_json->>'request_digest','response',v_pin);
END
$order_rpc$;

CREATE OR REPLACE FUNCTION public.order_image_confirmation_commit_v1(p_actor numeric,p_object_key text,p_key_hash text,p_request_hash text,p_request_digest text,p_published_key text,p_version_id text,p_at timestamptz)
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
DECLARE v_account numeric;v_auth numeric;v_scope bytea;v_target bytea;v_pin jsonb;v_replay jsonb;v_versions jsonb;
 v_lock_timeout text;v_scope_lock bigint;v_target_lock bigint;v_constraint text;v_rows bigint;
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 IF p_actor IS NULL OR p_actor<1 OR p_actor>18446744073709551615 OR p_actor<>trunc(p_actor)
  OR p_object_key IS NULL OR p_object_key !~ '^images/([0-9a-f]{64}/)?[0-9a-f]{64}[.](png|jpg)$'
  OR p_key_hash IS NULL OR p_key_hash !~ '^[0-9a-f]{64}$'
  OR (p_request_hash IS NOT NULL AND p_request_hash !~ '^[0-9a-f]{64}$')
  OR p_request_digest IS NULL OR p_request_digest !~ '^[0-9a-f]{64}$'
  OR p_published_key IS NULL OR p_version_id IS NULL
  OR (NOT (p_published_key=p_object_key AND p_version_id='') AND (p_published_key !~ '^images/published/[0-9a-f]{64}/[0-9a-f]{64}[.](png|jpg)$'
  OR substring(p_published_key FROM '[^/]+$') IS DISTINCT FROM substring(p_object_key FROM '[^/]+$')
  OR p_version_id IS NULL OR p_version_id='' OR p_version_id='null' OR p_version_id ~ '[[:space:][:cntrl:]]'
  OR p_version_id ~ U&'[\007f-\009f\00a0\1680\2000-\200a\2028\2029\202f\205f\3000]'))
  OR p_at IS NULL OR NOT isfinite(p_at) THEN
  RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END IF;
 v_lock_timeout:=current_setting('lock_timeout');
 PERFORM set_config('lock_timeout','5s',true);
 BEGIN
  SELECT id,auth_version INTO v_account,v_auth FROM public.merchant_accounts
   WHERE bound_user_id=p_actor AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER' FOR UPDATE;
 EXCEPTION WHEN OTHERS THEN
  PERFORM set_config('lock_timeout',v_lock_timeout,true);
  RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END;
 PERFORM set_config('lock_timeout',v_lock_timeout,true);
 IF v_account IS NULL OR v_account<=0 OR v_auth IS NULL OR v_auth<=0 THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 v_scope:=sha256(convert_to('merchant:'||trunc(p_actor)::text||':'||v_account::text,'UTF8'));
 v_target:=sha256(convert_to('target:'||p_object_key,'UTF8'));
 v_scope_lock:=('x'||substr(encode(sha256(convert_to('order_image_scope:'||encode(v_scope,'hex')||':'||p_key_hash,'UTF8')),'hex'),1,16))::bit(64)::bigint;
 v_target_lock:=('x'||substr(encode(sha256(convert_to('order_image_target:'||encode(v_target,'hex'),'UTF8')),'hex'),1,16))::bit(64)::bigint;
 PERFORM set_config('lock_timeout','5s',true);
 BEGIN
  PERFORM pg_advisory_xact_lock(v_scope_lock);
  PERFORM pg_advisory_xact_lock(v_target_lock);
 EXCEPTION WHEN OTHERS THEN
  PERFORM set_config('lock_timeout',v_lock_timeout,true);
  RETURN jsonb_build_object('outcome','UNAVAILABLE');
 END;
 PERFORM set_config('lock_timeout',v_lock_timeout,true);
 v_replay:=public.order_image_confirmation_replay_v1(p_actor,p_key_hash);
 IF v_replay->>'outcome'='SUCCEEDED' THEN
  IF v_replay->>'request_digest' IS DISTINCT FROM p_request_digest OR v_replay->'response'->>'object_key' IS DISTINCT FROM p_object_key THEN
   RETURN jsonb_build_object('outcome','CONFLICT');
  END IF;
  RETURN v_replay;
 ELSIF v_replay->>'outcome' IS DISTINCT FROM 'ABSENT' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 v_versions:=public.order_image_versions_v1(ARRAY[p_object_key]);
 IF v_versions->>'outcome' IS DISTINCT FROM 'SUCCEEDED' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
 IF jsonb_array_length(v_versions->'versions')=1 THEN
  v_pin:=v_versions->'versions'->0;
 ELSE
  v_pin:=jsonb_build_object('object_key',p_object_key,'published_key',p_published_key,'version_id',p_version_id);
 END IF;
 BEGIN
  INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,target_type,target_id,target_key_hash,operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at)
  VALUES('COMMAND_RECEIPT','MERCHANT',v_scope,p_actor,v_account,v_account,'OWNER',v_auth,'object.image.confirm','IMAGE_OBJECT',NULL,v_target,decode(p_key_hash,'hex'),decode(p_request_hash,'hex'),'SUCCEEDED','OK',jsonb_build_object('request_digest',p_request_digest),NULL,v_pin,p_at);
  GET DIAGNOSTICS v_rows=ROW_COUNT;
  IF v_rows<>1 THEN RAISE EXCEPTION USING ERRCODE='P1009',MESSAGE='ORDER_STORAGE_UNAVAILABLE';END IF;
 EXCEPTION WHEN unique_violation THEN
  GET STACKED DIAGNOSTICS v_constraint=CONSTRAINT_NAME;
  IF v_constraint IS DISTINCT FROM 'uq_action_audits_command_receipt' THEN RAISE;END IF;
  v_replay:=public.order_image_confirmation_replay_v1(p_actor,p_key_hash);
  IF v_replay->>'outcome' IS DISTINCT FROM 'SUCCEEDED' THEN RETURN jsonb_build_object('outcome','UNAVAILABLE');END IF;
  IF v_replay->>'request_digest' IS DISTINCT FROM p_request_digest OR v_replay->'response'->>'object_key' IS DISTINCT FROM p_object_key THEN RETURN jsonb_build_object('outcome','CONFLICT');END IF;
  RETURN v_replay;
 END;
 RETURN jsonb_build_object('outcome','SUCCEEDED','request_digest',p_request_digest,'response',v_pin);
END
$order_rpc$;

REVOKE ALL ON FUNCTION public.order_image_confirmation_replay_v1(numeric,text),public.order_image_confirmation_commit_v1(numeric,text,text,text,text,text,text,timestamptz),public.order_image_versions_v1(text[]) FROM PUBLIC,anon,authenticated;
GRANT EXECUTE ON FUNCTION public.order_image_confirmation_replay_v1(numeric,text),public.order_image_confirmation_commit_v1(numeric,text,text,text,text,text,text,timestamptz),public.order_image_versions_v1(text[]) TO authenticated;
