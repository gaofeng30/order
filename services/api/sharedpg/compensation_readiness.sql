CREATE OR REPLACE FUNCTION public.order_compensation_readiness_v1()
RETURNS jsonb LANGUAGE plpgsql SECURITY INVOKER
SET search_path=pg_catalog,public SET TimeZone='UTC' AS $order_rpc$
BEGIN
 IF current_user<>'authenticated' OR auth.role() IS DISTINCT FROM 'authenticated' OR auth.uid() IS DISTINCT FROM '__ORDER_SERVICE_UID__' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='ORDER_RUNTIME_FORBIDDEN';
 END IF;
 -- Catalog metadata only: never return wake rows, capabilities or extension
 -- credentials. Ordinary runtime identities retain no private schema access.
 RETURN jsonb_build_object(
  'role',current_user,'uid',auth.uid(),
  'functions',(SELECT coalesce(jsonb_agg(jsonb_build_object(
   'name',n.nspname||'.'||p.proname,'hash',encode(sha256(convert_to(p.prosrc,'UTF8')),'hex'),
   'security_definer',p.prosecdef,'returns_set',p.proretset,'strict',p.proisstrict,
   'volatility',p.provolatile::text,'parallel',p.proparallel::text,'result_type',p.prorettype::regtype::text,
   'argument_types',p.proargtypes::text,'argument_names',p.proargnames,'language',l.lanname,'config',p.proconfig,
   'kind',p.prokind::text,'defaults',p.pronargdefaults,'variadic',p.provariadic::bigint,'owner',pg_get_userbyid(p.proowner),
   'public_execute',EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE a.grantee=0 AND a.privilege_type='EXECUTE'),
   'anon_execute',has_function_privilege('anon',p.oid,'EXECUTE'),
   'authenticated_execute',has_function_privilege('authenticated',p.oid,'EXECUTE')) ORDER BY n.nspname,p.proname,p.oid),'[]'::jsonb)
   FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace JOIN pg_language l ON l.oid=p.prolang
   WHERE n.nspname='order_compensation' OR (n.nspname='public' AND (p.proname LIKE 'order_compensation_%' OR p.proname='order_subscription_claim_one_v1'))),
  'schemas',(SELECT coalesce(jsonb_agg(jsonb_build_object('name',n.nspname,'owner',pg_get_userbyid(n.nspowner),
   'public_access',EXISTS(SELECT 1 FROM aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a WHERE a.grantee=0),
   'anon_access',has_schema_privilege('anon',n.oid,'USAGE,CREATE'),
   'authenticated_access',has_schema_privilege('authenticated',n.oid,'USAGE,CREATE'))),'[]'::jsonb) FROM pg_namespace n WHERE n.nspname='order_compensation'),
  'platform_schemas',(SELECT coalesce(jsonb_agg(jsonb_build_object('name',n.nspname,'owner',pg_get_userbyid(n.nspowner),
   'public_access',EXISTS(SELECT 1 FROM aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a WHERE a.grantee=0),
   'anon_access',has_schema_privilege('anon',n.oid,'USAGE,CREATE'),
   'authenticated_access',has_schema_privilege('authenticated',n.oid,'USAGE,CREATE')) ORDER BY n.nspname),'[]'::jsonb) FROM pg_namespace n WHERE n.nspname IN ('cron','tencentdb_scf')),
  'relations',(SELECT coalesce(jsonb_agg(jsonb_build_object('name',c.relname,'kind',c.relkind::text,'owner',pg_get_userbyid(c.relowner),'persistence',c.relpersistence::text,'rls',c.relrowsecurity,'force_rls',c.relforcerowsecurity,
   'public_access',EXISTS(SELECT 1 FROM aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a WHERE a.grantee=0),
   'anon_access',has_table_privilege('anon',c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') OR has_any_column_privilege('anon',c.oid,'SELECT,INSERT,UPDATE,REFERENCES'),
   'authenticated_access',has_table_privilege('authenticated',c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') OR has_any_column_privilege('authenticated',c.oid,'SELECT,INSERT,UPDATE,REFERENCES'),
   'columns',(SELECT jsonb_agg(jsonb_build_object('name',a.attname,'type',format_type(a.atttypid,a.atttypmod),'not_null',a.attnotnull,'default',pg_get_expr(d.adbin,d.adrelid)) ORDER BY a.attnum) FROM pg_attribute a LEFT JOIN pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum WHERE a.attrelid=c.oid AND a.attnum>0 AND NOT a.attisdropped),
   'constraints',(SELECT coalesce(jsonb_agg(pg_get_constraintdef(k.oid) ORDER BY k.conname),'[]'::jsonb) FROM pg_constraint k WHERE k.conrelid=c.oid),
   'indexes',(SELECT coalesce(jsonb_agg(pg_get_indexdef(i.indexrelid) ORDER BY ic.relname),'[]'::jsonb) FROM pg_index i JOIN pg_class ic ON ic.oid=i.indexrelid WHERE i.indrelid=c.oid)
   ) ORDER BY c.relname),'[]'::jsonb) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='order_compensation' AND c.relkind NOT IN ('i','I')),
  'triggers',(SELECT coalesce(jsonb_agg(jsonb_build_object('name',t.tgname,'table',c.relname,'enabled',t.tgenabled::text,'type',t.tgtype,'function',fn.nspname||'.'||p.proname,'arguments',encode(t.tgargs,'hex'),'when',pg_get_expr(t.tgqual,t.tgrelid)) ORDER BY t.tgname),'[]'::jsonb)
   FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace JOIN pg_proc p ON p.oid=t.tgfoid JOIN pg_namespace fn ON fn.oid=p.pronamespace
   WHERE NOT t.tgisinternal AND n.nspname='public' AND (c.relname IN ('prepayments','refunds','notification_outbox') OR t.tgname LIKE 'order_compensation_%' OR fn.nspname='order_compensation')),
  'extensions',(SELECT coalesce(jsonb_agg(jsonb_build_object('name',e.extname,'version',e.extversion,'schema',n.nspname,'owner',pg_get_userbyid(e.extowner)) ORDER BY e.extname),'[]'::jsonb) FROM pg_extension e JOIN pg_namespace n ON n.oid=e.extnamespace WHERE e.extname IN ('pgcrypto','pg_cron','tencentdb_scf'))
 );
END
$order_rpc$;
REVOKE ALL ON FUNCTION public.order_compensation_readiness_v1() FROM PUBLIC,anon;
GRANT EXECUTE ON FUNCTION public.order_compensation_readiness_v1() TO authenticated;
