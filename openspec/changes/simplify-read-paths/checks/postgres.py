"""Run changed read contracts in an isolated, Unix-socket-only PostgreSQL.
No credentials/environment DSN, no cloud connection, no persistent database.
"""
import json, pathlib, re, shutil, subprocess, tempfile
ROOT = pathlib.Path(__file__).resolve().parents[4]
PG = pathlib.Path('/opt/homebrew/opt/postgresql@16/bin')
BASE = '0aba16b086bd3d2770e7dce41ccb86311e697587'

def run(args):
    return subprocess.run([str(a) for a in args], check=True, text=True, capture_output=True).stdout

def function(source, name):
    return re.search(r'CREATE OR REPLACE FUNCTION public\.' + name + r'\(.*?\$order_rpc\$;', source, re.S).group()

with tempfile.TemporaryDirectory(prefix='order-read-pg-') as tmp:
    tmp=pathlib.Path(tmp); data=tmp/'data'; sock=tmp/'socket'; sock.mkdir()
    run([PG/'initdb','-D',data,'--auth=trust','--no-locale','--encoding=UTF8'])
    run([PG/'pg_ctl','-D',data,'-l',tmp/'log','-o',f"-k {sock} -p 55491 -c listen_addresses=''",'-w','start'])
    def sql(text):
        return run([PG/'psql','-X','-q','-A','-t','-h',sock,'-p','55491','-d','postgres','-v','ON_ERROR_STOP=1','-c',text]).strip()
    def rpc(expr):
        return json.loads(sql('SET ROLE authenticated; SELECT '+expr))
    try:
        sql("""
        CREATE ROLE authenticated; CREATE ROLE anon; CREATE SCHEMA auth;
        CREATE FUNCTION auth.role() RETURNS text LANGUAGE sql AS $$ SELECT current_user::text $$;
        CREATE FUNCTION auth.uid() RETURNS text LANGUAGE sql AS $$ SELECT '123'::text $$;
        CREATE TABLE storefront_settings(id int,business_status text);
        CREATE TABLE meal_periods(code text,cutoff_time time,pickup_start_time time,pickup_end_time time,interval_minutes int);
        CREATE TABLE service_dates(service_date date,is_open boolean);
        CREATE TABLE categories(id bigint,is_active boolean);
        CREATE TABLE products(id bigint,category_id bigint,is_listed boolean,meal_period text);
        CREATE TABLE product_sold_out_dates(product_id bigint,service_date date);
        CREATE TABLE merchant_accounts(id bigint,bound_user_id bigint,enabled boolean,deleted_at timestamptz,role text);
        CREATE TABLE orders(id bigint,paid_at timestamptz,state text,payable_cents bigint,pickup_at timestamptz,order_no bytea,pickup_number int,transaction_id bytea);
        CREATE INDEX idx_orders_state_pickup ON orders(state,pickup_at,id);
        CREATE TABLE refunds(id bigint,amount_cents bigint,materialization_state text);
        CREATE TABLE refund_observations(refund_id bigint,success_time timestamptz,provider_state text,validation text,apply_state text);
        CREATE TABLE order_items(order_id bigint,product_id bigint,product_name_snapshot text,quantity bigint);
        CREATE TABLE action_audits(id bigint,entry_kind text,action text,target_key_hash bytea,response_json jsonb,actor_kind text,actor_user_id bigint,actor_account_id bigint,actor_account_id_snapshot bigint,actor_role_snapshot text,actor_auth_version_snapshot bigint,actor_scope_hash bytea,target_type text,target_id bigint,result text,reason_code text,before_state_json jsonb);
        GRANT USAGE ON SCHEMA auth TO authenticated,anon;
        GRANT SELECT ON ALL TABLES IN SCHEMA public TO authenticated;
        INSERT INTO storefront_settings VALUES(1,'open');
        INSERT INTO meal_periods VALUES('lunch','10:00','11:00','12:00',30),('dinner','16:00','17:00','18:00',30);
        INSERT INTO service_dates VALUES('2026-10-13',true),('2026-10-14',true);
        INSERT INTO categories VALUES(1,true),(2,false);
        INSERT INTO products VALUES(1,1,true,'lunch'),(2,1,true,'dinner'),(3,2,true,'all'),(4,1,false,'all');
        INSERT INTO product_sold_out_dates VALUES(1,'2026-10-13');
        INSERT INTO merchant_accounts VALUES(1,7,true,NULL,'OWNER'),(2,8,true,NULL,'STAFF');
        """)
        shared=ROOT/'services/api/sharedpg'
        for file,names in [('menu.sql',['order_menu_pickup_facts_v1']),('admin_report.sql',['order_admin_report_counts_v1','order_admin_report_export_v1','order_admin_report_stats_v1']),('image_version_confirmation.sql',['order_image_versions_v1'])]:
            source=(shared/file).read_text()
            for name in names: sql(function(source,name).replace('__ORDER_SERVICE_UID__','123'))
        old_images=run(['git','-C',ROOT,'show',BASE+':services/api/sharedpg/image_version_confirmation.sql'])
        sql(function(old_images,'order_image_versions_v1').replace('order_image_versions_v1','lab_image_versions_baseline',1).replace('__ORDER_SERVICE_UID__','123'))
        def pins_equal():
            keys="ARRAY['images/'||lpad(to_hex(1),64,'0')||'.png','images/'||lpad(to_hex(2),64,'0')||'.png','images/'||repeat('f',64)||'.png']"
            assert rpc('order_image_versions_v1('+keys+')')==rpc('lab_image_versions_baseline('+keys+')')
        options=rpc("order_menu_pickup_facts_v1('2026-10-13','2026-10-14')")
        assert options['service_dates'][0]['available_meals']==['dinner'], options
        assert options['service_dates'][1]['available_meals']==['dinner','lunch'], options
        sql("UPDATE products SET is_listed=false WHERE id=2")
        assert rpc("order_menu_pickup_facts_v1('2026-10-13','2026-10-14')")['service_dates'][0]['available_meals']==[]
        native=(ROOT/'services/api/internal/menu/repository.go').read_text()
        query=re.search(r'const pickupDatesQuery = `(.*?)`',native,re.S).group(1).replace('$1',"'2026-10-13'").replace('$2',"'2026-10-14'")
        assert sql(query).splitlines()==['2026-10-13|t|f|f','2026-10-14|t|t|f']
        assert rpc('order_admin_report_counts_v1(8)')['outcome']=='FORBIDDEN'
        assert rpc("order_admin_report_export_v1(8,'323032362d31302d3133','323032362d31302d3133',0,100)")['outcome']=='FORBIDDEN'
        # 16 distinct pins; verify SQL batch cap and audit validation still apply.
        sql("""INSERT INTO action_audits SELECT g,'COMMAND_RECEIPT','object.image.confirm',sha256(convert_to('target:images/'||lpad(to_hex(g),64,'0')||'.png','UTF8')),jsonb_build_object('object_key','images/'||lpad(to_hex(g),64,'0')||'.png','published_key','images/published/'||repeat('a',64)||'/'||lpad(to_hex(g),64,'0')||'.png','version_id','fixed-version'),'MERCHANT',7,1,1,'OWNER',1,sha256(convert_to('merchant:7:1','UTF8')),'IMAGE_OBJECT',NULL,'SUCCEEDED','OK',jsonb_build_object('request_digest',repeat('b',64)) FROM generate_series(1,16) g""")
        pins=rpc("order_image_versions_v1(ARRAY(SELECT 'images/'||lpad(to_hex(g),64,'0')||'.png' FROM generate_series(1,16) g))")
        assert len(pins['versions'])==16,pins
        pins_equal()
        duplicate=rpc("order_image_versions_v1(ARRAY['images/'||lpad(to_hex(1),64,'0')||'.png','images/'||lpad(to_hex(1),64,'0')||'.png','images/'||repeat('f',64)||'.png'])")
        assert len(duplicate['versions'])==1,duplicate
        sql("INSERT INTO action_audits SELECT (jsonb_populate_record(NULL::action_audits,to_jsonb(a)||jsonb_build_object('id',17,'response_json',response_json||jsonb_build_object('version_id','conflicting-version')))).* FROM action_audits a WHERE id=1")
        assert rpc("order_image_versions_v1(ARRAY['images/'||lpad(to_hex(1),64,'0')||'.png'])")['outcome']=='UNAVAILABLE'
        pins_equal()
        sql("DELETE FROM action_audits WHERE id=17")
        sql("UPDATE action_audits SET response_json=jsonb_set(response_json,'{object_key}',to_jsonb('images/'||repeat('e',64)||'.png')) WHERE id=1")
        pins_equal()
        assert rpc("order_image_versions_v1(ARRAY['images/'||lpad(to_hex(1),64,'0')||'.png'])")['outcome']=='UNAVAILABLE'
        sql("UPDATE action_audits SET response_json=jsonb_set(response_json,'{object_key}',to_jsonb('images/'||lpad(to_hex(1),64,'0')||'.png')) WHERE id=1")
        sql("UPDATE action_audits SET actor_role_snapshot='STAFF' WHERE id=1")
        assert rpc("order_image_versions_v1(ARRAY['images/'||lpad(to_hex(1),64,'0')||'.png'])")['outcome']=='UNAVAILABLE'
        pins_equal()
        seed="""INSERT INTO orders SELECT g,'2020-01-01'::timestamptz+g*interval '10 minutes',CASE WHEN g%97=0 THEN 'REFUNDED' ELSE 'COMPLETED' END,1000,'2020-01-01',convert_to('old-'||g,'UTF8'),1,convert_to('tx-'||g,'UTF8') FROM generate_series(1,200000) g;
        INSERT INTO orders SELECT 200000+g,'2026-10-01'::timestamptz+g*interval '1 hour',CASE WHEN g%5=0 THEN 'PREPARING' ELSE 'COMPLETED' END,1500,'2026-10-01',convert_to('new-'||g,'UTF8'),1,convert_to('tx-'||g,'UTF8') FROM generate_series(1,500) g;
        INSERT INTO orders VALUES(300001,'2020-01-01','PREPARING',1000,'2020-01-01','old-pending',1,'tx'),(300002,'2026-09-30T15:59:59Z','COMPLETED',1234,'2026-10-01','before-month',1,'tx'),(300003,'2026-09-30T16:00:00Z','COMPLETED',1234,'2026-10-01','month-start',1,'tx'),(300004,'2026-10-31T15:59:59Z','COMPLETED',1234,'2026-10-31','month-end',1,'tx'),(300005,'2026-10-31T16:00:00Z','COMPLETED',1234,'2026-11-01','after-month',1,'tx');"""
        sql(seed);sql('VACUUM ANALYZE orders')
        source_path='services/api/internal/adminreport/postgresql.go'
        baseline=run(['git','-C',ROOT,'show',BASE+':'+source_path])
        current=(ROOT/source_path).read_text()
        results=[]
        for label,source in [('baseline',baseline),('current',current)]:
            query=re.search(r'err := a.db.QueryRowContext\(ctx, `(SELECT COALESCE\(SUM\(payable_cents\).*?)`',source).group(1).replace('$1',"'2026-10-13T12:00:00+08:00'")
            value=sql(query);plans=[json.loads(sql('EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) '+query))[0] for _ in range(3)]
            results.append({'variant':label,'value':value,'execution_ms':[p['Execution Time'] for p in plans]})
        assert results[0]['value']==results[1]['value'],results
        params=json.dumps({'epoch_seconds':'1791864000','nanosecond':0,'zero':False,'canonical_utc':'2026-10-13T04:00:00Z'})
        stats=rpc("order_admin_report_stats_v1(7,'"+params+"'::jsonb)")
        assert stats['outcome']=='FOUND' and '|'.join(stats['rows'][0])==results[1]['value'],stats
        counts=rpc('order_admin_report_counts_v1(7)');assert counts['outcome']=='FOUND' and sum(int(row[1]) for row in counts['rows'])==200505,counts
        hx=lambda x:x.encode().hex()
        page=rpc(f"order_admin_report_export_v1(7,'{hx('2026-10-13')}','{hx('2026-10-13')}',0,100)")
        assert page['outcome']=='FOUND' and len(page['rows'])==24 and all(len(row)==6 for row in page['rows']) and 'items' not in page,page
        print(json.dumps({'scope':'isolated PG16, synthetic data, unchanged indexes; not UAT latency','contracts':'menu date/meal/listed/category/sold-out; pins/invalid receipt; owner denial; export narrow rows; stats full-month and all-time pending','stats':results},ensure_ascii=False,indent=2))
    finally:
        run([PG/'pg_ctl','-D',data,'-m','fast','-w','stop'])
