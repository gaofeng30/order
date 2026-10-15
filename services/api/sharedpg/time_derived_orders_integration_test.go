package sharedpg

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaofeng30/order/services/api/internal/migrate"
)

// Native PostgreSQL RPC regression. Historical paid-order fixtures bypass only
// foreign-key triggers during fixture insertion, then creates all referenced facts;
// every constraint and trigger is active during operations under test.
// This does not simulate WeChat payment acceptance or real device E2E.
func TestTimeDerivedOrdersWithoutProductionWorker(t *testing.T) {
	bin := os.Getenv("ORDER_TEST_PG_BIN")
	if bin == "" {
		t.Skip("set ORDER_TEST_PG_BIN to a PostgreSQL 16/17 bin directory")
	}
	dir, err := os.MkdirTemp("", "order-ready-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	run := func(name string, args ...string) []byte {
		t.Helper()
		out, err := exec.Command(filepath.Join(bin, name), args...).CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v\n%s", name, err, out)
		}
		return out
	}
	run("initdb", "-D", dir+"/data", "--auth=trust", "--no-locale", "--encoding=UTF8")
	run("pg_ctl", "-D", dir+"/data", "-l", dir+"/log", "-o", "-k "+dir+" -p 55493 -c listen_addresses=''", "-w", "start")
	defer run("pg_ctl", "-D", dir+"/data", "-m", "immediate", "-w", "stop")
	sql := func(s string) []byte {
		return run("psql", "-X", "-q", "-A", "-t", "-h", dir, "-p", "55493", "-d", "postgres", "-v", "ON_ERROR_STOP=1", "-c", s)
	}
	schema, err := migrate.SharedPostgreSQLInstallSQL()
	if err != nil {
		t.Fatal(err)
	}
	// Use stdin for the large schema/install statements; avoid command-size limits.
	install := func(s string) {
		t.Helper()
		cmd := exec.Command(filepath.Join(bin, "psql"), "-X", "-q", "-h", dir, "-p", "55493", "-d", "postgres", "-v", "ON_ERROR_STOP=1")
		f := dir + "/install.sql"
		if err := os.WriteFile(f, []byte(s), 0600); err != nil {
			t.Fatal(err)
		}
		cmd.Args = append(cmd.Args, "-f", f)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("install: %v\n%s", err, out)
		}
	}
	install(schema)
	sql("CREATE ROLE anon; CREATE ROLE authenticated; CREATE SCHEMA auth; CREATE FUNCTION auth.uid() RETURNS text LANGUAGE sql AS $$ SELECT '123'::text $$; CREATE FUNCTION auth.role() RETURNS text LANGUAGE sql AS $$ SELECT current_user::text $$; GRANT USAGE ON SCHEMA auth TO authenticated,anon;")
	definition, err := InstallSQL("123", "uat", "https://example.invalid/api/v1/payments/wechat/notify", "https://example.invalid/api/v1/refunds/wechat/notify")
	if err != nil {
		t.Fatal(err)
	}
	install(definition)

	install(`
 SET session_replication_role=replica;
 INSERT INTO miniprogram_users(id,openid,created_at,last_login_at) VALUES(1,convert_to('test-user','UTF8'),now(),now());
 INSERT INTO merchant_accounts(id,phone,name,role,bound_user_id,bound_at,created_at,updated_at)
 VALUES(1,convert_to('+8613800000000','UTF8'),'Test owner','OWNER',1,now(),now(),now());
 INSERT INTO orders(id,order_no,user_id,quote_id,prepayment_id,payment_observation_id,
 contact_name_snapshot,contact_phone_snapshot,identity_kind,identity_source_version,discount_rate_percent,discount_version,
 store_name_snapshot,store_address_snapshot,pickup_point_snapshot,pickup_date,pickup_time,pickup_at,meal_period,order_note,
 item_count,original_subtotal_cents,discount_cents,payable_cents,snapshot_digest,transaction_id,paid_at,materialized_at,pickup_number,
 state,created_at,updated_at)
 SELECT i,convert_to('test-order-'||i,'UTF8'),1,i,i,i,'Test',convert_to('+8613800000000','UTF8'),'VISITOR',1,100,1,
 'Store','Address','Pickup',CURRENT_DATE,'12:00',now()+interval '1 hour','lunch','',1,100,0,100,decode(repeat('00',32),'hex'),
 convert_to('transaction-'||i,'UTF8'),now()-interval '1 hour',now()-interval '1 hour',i,'RESERVED',now()-interval '1 hour',now()-interval '1 hour'
 FROM generate_series(1,3) i;
 INSERT INTO quotes(id,user_id,contact_name_snapshot,contact_phone_snapshot,idempotency_key_hash,request_digest,identity_kind,identity_source_version,discount_rate_percent,discount_version,store_name_snapshot,store_address_snapshot,pickup_point_snapshot,pickup_date,pickup_time,meal_period,order_note,item_count,original_subtotal_cents,discount_cents,payable_cents,snapshot_digest,created_at,expires_at)
 SELECT id,user_id,contact_name_snapshot,contact_phone_snapshot,sha256(order_no),snapshot_digest,identity_kind,identity_source_version,discount_rate_percent,discount_version,store_name_snapshot,store_address_snapshot,pickup_point_snapshot,pickup_date,pickup_time,meal_period,order_note,item_count,original_subtotal_cents,discount_cents,payable_cents,snapshot_digest,created_at,created_at+interval '5 minutes' FROM orders;
 INSERT INTO prepayments(id,user_id,quote_id,idempotency_key_hash,out_trade_no,expected_appid,expected_mchid,expected_amount_cents,currency,provider_create_request_json,provider_create_request_digest,effective_deadline,provider_state,create_attempted_at,materialization_state,materialized_at,created_at,updated_at)
 SELECT id,user_id,quote_id,sha256(order_no),order_no,convert_to('test-app','UTF8'),convert_to('test-merchant','UTF8'),payable_cents,'CNY','{}',snapshot_digest,created_at+interval '5 minutes','PAID',created_at,'APPLIED',materialized_at,created_at,updated_at FROM orders;
 INSERT INTO payment_observations(id,prepayment_id,dedupe_key,source,out_trade_no,transaction_id,provider_state,validation,amount_cents,currency,success_time,received_at,materialization_mode,apply_state,applied_at)
 SELECT id,prepayment_id,sha256(order_no),'CALLBACK',order_no,transaction_id,'PAID','MATCH',payable_cents,'CNY',paid_at,paid_at,'AUTO','APPLIED',materialized_at FROM orders;
 SET session_replication_role=origin;
 `)

	t.Run("successful-prepayment-does-not-enter-compensation", func(t *testing.T) {
		sql(`BEGIN;
 UPDATE prepayments SET provider_state='CREATE_CLAIMED',materialization_state='AWAITING_PAYMENT',materialized_at=NULL,lease_kind='CREATE',lease_owner=decode(repeat('ab',16),'hex'),lease_expires_at=now()+interval '1 minute',record_version=10,next_reconcile_at=now() WHERE id=1;
 SET LOCAL ROLE authenticated;
 SELECT public.order_payment_finish_create_v1(1,repeat('ab',16),10,now(),now()+interval '15 seconds','{"prepay_id_hex":"706179","request_payment_json":"{}"}'::jsonb);
 DO $check$ BEGIN
 IF EXISTS(SELECT 1 FROM public.prepayments WHERE id=1 AND next_reconcile_at IS NOT NULL) THEN RAISE EXCEPTION 'normal payment has compensation scheduled'; END IF;
 IF public.order_payment_due_v1(now()+interval '1 hour',100)->'ids' @> '["1"]'::jsonb THEN RAISE EXCEPTION 'normal payment selected as exception'; END IF;
 END $check$;
 ROLLBACK;`)
	})

	t.Run("confirmed-notpay-stops-compensation", func(t *testing.T) {
		sql(`BEGIN;
 DO $test$
 DECLARE req text; tx jsonb; result jsonb;
 BEGIN
 req := '{"appid":"test-app","mchid":"test-merchant","description":"test","out_trade_no":"test-order-1","time_expire":"2026-10-15T12:00:00+08:00","notify_url":"https://example.invalid/api/v1/payments/wechat/notify","amount_cents":100,"currency":"CNY","payer_openid":"test-user","quote_id":"1","quote_digest":"' || repeat('00',32) || '"}';
 UPDATE prepayments SET provider_state='NOT_PAID',materialization_state='AWAITING_PAYMENT',materialized_at=NULL,lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,next_reconcile_at=now(),provider_create_request_json=req::jsonb,provider_create_request_digest=sha256(convert_to(req,'UTF8')) WHERE id=1;
 PERFORM setval(pg_get_serial_sequence('public.payment_observations','id'),100);
 tx:=jsonb_build_object('appid_hex',encode(convert_to('test-app','UTF8'),'hex'),'mchid_hex',encode(convert_to('test-merchant','UTF8'),'hex'),'out_trade_no_hex',encode(convert_to('test-order-1','UTF8'),'hex'),'transaction_id_hex','','trade_type_hex','','trade_state_hex','4e4f54504159','trade_state_desc_hex','','bank_type_hex','','attach_hex','','payer_openid_hex','','currency_hex','','payer_currency_hex','','total','0','payer_total','0','success_zero',true,'success_epoch_seconds','-62135596800','success_nanosecond',0,'success_rfc3339nano','0001-01-01T00:00:00Z');
 SET LOCAL ROLE authenticated;
 result:=public.order_payment_observe_v1(1,'ACTIVE_QUERY','',tx,NULL,now(),15000000000);
 IF result->>'outcome'<>'OBSERVED' THEN RAISE EXCEPTION 'observation rejected: %',result; END IF;
 IF EXISTS(SELECT 1 FROM public.prepayments WHERE id=1 AND next_reconcile_at IS NOT NULL) THEN RAISE EXCEPTION 'confirmed NOTPAY has compensation scheduled'; END IF;
 IF public.order_payment_due_v1(now()+interval '1 hour',100)->'ids' @> '["1"]'::jsonb THEN RAISE EXCEPTION 'confirmed NOTPAY selected as exception'; END IF;
 END $test$;
 ROLLBACK;`)
	})

	t.Run("boundary-and-filters", func(t *testing.T) {
		sql(`DO $test$
 DECLARE rows jsonb; counts jsonb; n timestamptz:=statement_timestamp();
 BEGIN
 UPDATE orders SET pickup_at=n+interval '30 minutes' WHERE id=1;
 UPDATE orders SET pickup_at=n+interval '30 minutes 1 microsecond' WHERE id=2;
 UPDATE orders SET pickup_at=n+interval '29 minutes' WHERE id=3;
 SET LOCAL ROLE authenticated;
 rows:=order_query_list_user_v1(1,'PREPARING',false,0,100);
 IF jsonb_array_length(rows)<>2 THEN RAISE EXCEPTION 'due preparing list: expected 2, got %',rows; END IF;
 IF rows->0->>'id'<>'3' OR rows->1->>'id'<>'1' THEN RAISE EXCEPTION 'wrong paging order: %',rows; END IF;
 rows:=order_query_list_user_v1(1,'RESERVED',false,0,100);
 IF jsonb_array_length(rows)<>1 OR rows->0->>'id'<>'2' THEN RAISE EXCEPTION 'reserved boundary: %',rows; END IF;
 rows:=order_query_search_merchant_v1(1,'PREPARING','','',0,100);
 IF jsonb_array_length(rows)<>2 THEN RAISE EXCEPTION 'merchant list mismatch: %',rows; END IF;
 rows:=order_query_get_user_v1(1,1);
 IF rows->'order'->>'state'<>'PREPARING' OR (rows->'order'->>'preparing_at')::timestamptz<>n THEN RAISE EXCEPTION 'detail mismatch: %',rows; END IF;
 counts:=order_admin_report_counts_v1(1);
 IF NOT (counts->'rows' @> '[ ["PREPARING","2"], ["RESERVED","1"] ]'::jsonb) THEN RAISE EXCEPTION 'admin counts mismatch: %',counts; END IF;
 RESET ROLE;
 IF EXISTS(SELECT 1 FROM orders WHERE state<>'RESERVED' OR preparing_at IS NOT NULL OR record_version<>1) THEN RAISE EXCEPTION 'read wrote orders'; END IF;
 END $test$;`)
	})
	t.Run("mark-ready-without-scheduler-and-replay", func(t *testing.T) {
		sql(`DO $test$
 DECLARE proof jsonb; result jsonb; n timestamptz:=statement_timestamp(); req text:='{"kind":"MARK_READY","order_id":1}';
 BEGIN
 UPDATE orders SET pickup_at=n+interval '30 minutes' WHERE id=1;
 SET LOCAL ROLE authenticated;
 proof:=order_fulfillment_proof_v1(1,'MARK_READY',1,'ready-1',req,NULL,true,NULL);
 result:=order_mark_ready_v1(1,1,'ready-1','request-1',n,n,req,proof->'proof',jsonb_build_object('hash',repeat('ab',32),'ciphertext','abcd','key_version',1),n);
 IF result->>'state'<>'READY_FOR_PICKUP' OR result->>'replay'<>'false' THEN RAISE EXCEPTION 'ready failed: %',result; END IF;
 result:=order_mark_ready_v1(1,1,'ready-1','request-2',NULL,NULL,req,NULL,NULL,NULL);
 IF result->>'replay'<>'true' THEN RAISE EXCEPTION 'replay failed: %',result; END IF;
 RESET ROLE;
 IF NOT EXISTS(SELECT 1 FROM orders WHERE id=1 AND state='READY_FOR_PICKUP' AND preparing_at=n AND ready_at=n AND record_version=2) THEN RAISE EXCEPTION 'ready persistence/history failed'; END IF;
 END $test$;`)
	})

	t.Run("future-order-cannot-be-marked-ready", func(t *testing.T) {
		sql(`DO $test$
 DECLARE proof jsonb; rejected boolean:=false; n timestamptz:=statement_timestamp(); req text:='{"kind":"MARK_READY","order_id":2}';
 BEGIN
 UPDATE orders SET pickup_at=n+interval '31 minutes' WHERE id=2;
 SET LOCAL ROLE authenticated;
 proof:=order_fulfillment_proof_v1(1,'MARK_READY',2,'early-2',req,NULL,true,NULL);
 BEGIN
 PERFORM order_mark_ready_v1(1,2,'early-2','early-request',n,n,req,proof->'proof',jsonb_build_object('hash',repeat('cd',32),'ciphertext','abcd','key_version',1),n);
 EXCEPTION WHEN SQLSTATE 'P1004' THEN rejected:=true; END;
 IF NOT rejected THEN RAISE EXCEPTION 'early ready allowed'; END IF;
 RESET ROLE;
 IF NOT EXISTS(SELECT 1 FROM orders WHERE id=2 AND state='RESERVED' AND record_version=1 AND preparing_at IS NULL) THEN RAISE EXCEPTION 'rejected write mutated order'; END IF;
 END $test$;`)
	})
	t.Run("cancel-boundary-does-not-rely-on-physical-state", func(t *testing.T) {
		sql(`DO $test$
 DECLARE result jsonb; n timestamptz:=date_trunc('second',statement_timestamp()); at_time timestamptz; locator jsonb; wire jsonb;
 BEGIN
 at_time:=n+interval '30 minutes';
 UPDATE orders SET pickup_at=at_time WHERE id=3;
 locator:=jsonb_build_object('order_id','3','prepayment_id','3','user_id','1','pickup_at',jsonb_build_object('epoch_seconds',extract(epoch FROM at_time)::bigint::text,'nanosecond',0,'zero',false,'canonical',to_char(at_time,'YYYY-MM-DD"T"HH24:MI:SS"Z"')));
 wire:=jsonb_build_object('epoch_seconds',extract(epoch FROM n)::bigint::text,'nanosecond',0,'zero',false,'canonical',to_char(n,'YYYY-MM-DD"T"HH24:MI:SS"Z"'));
 SET LOCAL ROLE authenticated;
 result:=order_refund_request_order_v2('USER',1,locator,encode(convert_to('USER_CANCEL','UTF8'),'hex'),repeat('01',32),repeat('02',32),wire,encode(convert_to('https://example.invalid/api/v1/refunds/wechat/notify','UTF8'),'hex'),'uat');
 IF result->>'outcome'<>'TRANSITION_NOT_ALLOWED' THEN RAISE EXCEPTION 'cancel boundary not enforced: %',result; END IF;
 RESET ROLE;
 IF NOT EXISTS(SELECT 1 FROM orders WHERE id=3 AND state='RESERVED' AND record_version=1) THEN RAISE EXCEPTION 'cancel wrote order'; END IF;
 UPDATE orders SET state='REFUNDING',refunding_at=n,updated_at=n WHERE id=3;
 SET LOCAL ROLE authenticated;
 result:=order_query_get_user_v1(1,3);
 IF result->'order'->>'state'<>'REFUNDING' THEN RAISE EXCEPTION 'refund state overwritten'; END IF;
 END $test$;`)
	})
	t.Run("concurrent-duplicate-ready", func(t *testing.T) {
		sql("UPDATE orders SET pickup_at=statement_timestamp()+interval '29 minutes' WHERE id=2")
		raw := strings.TrimSpace(string(sql(`SET ROLE authenticated; SELECT order_fulfillment_proof_v1(1,'MARK_READY',2,'concurrent-2','{"kind":"MARK_READY","order_id":2}',NULL,true,NULL)->'proof';`)))
		proof := "'" + strings.ReplaceAll(raw, "'", "''") + "'::jsonb"
		query := `SET ROLE authenticated; SELECT order_mark_ready_v1(1,2,'concurrent-2','concurrent-request',statement_timestamp(),statement_timestamp(),'{"kind":"MARK_READY","order_id":2}',` + proof + `,jsonb_build_object('hash',repeat('cd',32),'ciphertext','abcd','key_version',1),statement_timestamp());`
		type response struct {
			raw []byte
			err error
		}
		done := make(chan response, 2)
		for range 2 {
			go func() {
				out, err := exec.Command(filepath.Join(bin, "psql"), "-X", "-q", "-A", "-t", "-h", dir, "-p", "55493", "-d", "postgres", "-v", "ON_ERROR_STOP=1", "-c", query).CombinedOutput()
				done <- response{out, err}
			}()
		}
		replays := 0
		for range 2 {
			r := <-done
			if r.err != nil {
				t.Fatalf("concurrent ready: %v %s", r.err, r.raw)
			}
			var v struct {
				Replay bool   `json:"replay"`
				State  string `json:"state"`
			}
			if json.Unmarshal(r.raw, &v) != nil || v.State != "READY_FOR_PICKUP" {
				t.Fatalf("bad result: %s", r.raw)
			}
			if v.Replay {
				replays++
			}
		}
		if replays != 1 {
			t.Fatalf("expected one actual transition, got %d replays", replays)
		}
		sql(`DO $test$ BEGIN IF NOT EXISTS(SELECT 1 FROM orders WHERE id=2 AND state='READY_FOR_PICKUP' AND record_version=2) THEN RAISE EXCEPTION 'duplicate version increment'; END IF; END $test$;`)
	})

	t.Run("refund-due-excludes-normal-processing", func(t *testing.T) {
		sql(`BEGIN;
 DO $test$
 DECLARE n timestamptz:=date_trunc('second',statement_timestamp()); wire jsonb; result jsonb;
 BEGIN
 INSERT INTO refunds(id,prepayment_id,order_id,out_refund_no,idempotency_key_hash,amount_cents,currency,reason_code,requested_by_user_id,provider_request_json,provider_request_digest,provider_state,provider_refund_id,create_attempted_at,materialization_state,record_version,created_at,updated_at)
 VALUES(1,3,3,convert_to('refund-processing-test','UTF8'),decode(repeat('11',32),'hex'),100,'CNY','USER_CANCEL',1,'{}',decode(repeat('22',32),'hex'),'PROCESSING',convert_to('provider-refund','UTF8'),n,'AWAITING_PROVIDER',1,n,n);
 wire:=jsonb_build_object('epoch_seconds',extract(epoch FROM n)::bigint::text,'nanosecond',0,'zero',false,'canonical',to_char(n,'YYYY-MM-DD"T"HH24:MI:SS"Z"'));
 SET LOCAL ROLE authenticated;
 result:=order_refund_due_v1(wire,wire,100);
 IF result->'ids' IS DISTINCT FROM '[]'::jsonb THEN RAISE EXCEPTION 'normal processing selected: %',result; END IF;
 RESET ROLE;
 -- A crashed query worker must still be recoverable even when no next query was set.
 UPDATE refunds SET provider_state='CREATE_UNKNOWN',provider_refund_id=NULL,lease_kind='QUERY',lease_owner=decode(repeat('ab',16),'hex'),lease_expires_at=n-interval '1 second' WHERE id=1;
 SET LOCAL ROLE authenticated;
 result:=order_refund_due_v1(wire,wire,100);
 IF result->'ids' IS DISTINCT FROM '["1"]'::jsonb THEN RAISE EXCEPTION 'expired query not recovered: %',result; END IF;
 END $test$;
 ROLLBACK;`)
	})

	t.Run("refund-create-error-waits-thirty-minutes", func(t *testing.T) {
		sql(`BEGIN;
 DO $test$
 DECLARE n timestamptz:=date_trunc('second',statement_timestamp()); wire jsonb; result jsonb; req jsonb; obs jsonb;
 BEGIN
 INSERT INTO refunds(id,prepayment_id,order_id,out_refund_no,idempotency_key_hash,amount_cents,currency,reason_code,requested_by_user_id,provider_request_json,provider_request_digest,provider_state,create_attempted_at,materialization_state,lease_kind,lease_owner,lease_expires_at,record_version,created_at,updated_at)
 VALUES(1,3,3,convert_to('refund-create-test','UTF8'),decode(repeat('11',32),'hex'),100,'CNY','USER_CANCEL',1,'{}',decode(repeat('22',32),'hex'),'CREATE_CLAIMED',n,'AWAITING_PROVIDER','CREATE',decode(repeat('ab',16),'hex'),n+interval '30 seconds',1,n,n);
 INSERT INTO action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,action,target_type,target_id,operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at) VALUES('COMMAND_RECEIPT','USER',decode(repeat('33',32),'hex'),1,'refund.request','REFUND',1,decode(repeat('44',32),'hex'),decode(repeat('55',32),'hex'),'SUCCEEDED','REFUND_REQUESTED','{}','{}','{}',n);
 wire:=jsonb_build_object('epoch_seconds',extract(epoch FROM n)::bigint::text,'nanosecond',0,'zero',false,'canonical',to_char(n,'YYYY-MM-DD"T"HH24:MI:SS"Z"'),'receipt_time',jsonb_build_object('stored_kind','FINITE','stored_microseconds',(extract(epoch FROM n)*1000000-946684800000000)::bigint::text,'codec_status','VALID','json_hex',encode(convert_to(to_json(to_char(n,'YYYY-MM-DD"T"HH24:MI:SS"Z"'))::text,'UTF8'),'hex')));
 SET LOCAL ROLE authenticated;
 result:=order_refund_finish_create_v1(1,repeat('ab',16),1,NULL,true,NULL,wire);
 -- The same expired writer completion must not extend the durable retry or version.
 PERFORM order_refund_finish_create_v1(1,repeat('ab',16),1,NULL,true,NULL,wire);
 RESET ROLE;
 IF result->>'outcome'='UNAVAILABLE' THEN RAISE EXCEPTION 'create completion rejected: %',result; END IF;
 IF NOT EXISTS(SELECT 1 FROM refunds WHERE id=1 AND provider_state='CREATE_UNKNOWN' AND next_query_at=n+interval '30 minutes' AND record_version=2 AND lease_kind IS NULL) THEN RAISE EXCEPTION 'create failure does not wait 30 minutes: %',result; END IF;
 UPDATE refunds SET provider_state='CREATE_CLAIMED',materialization_state='AWAITING_PROVIDER',pending_reason_code=NULL,next_query_at=NULL,lease_kind='CREATE',lease_owner=decode(repeat('ab',16),'hex'),lease_expires_at=n+interval '30 seconds',record_version=3 WHERE id=1;
 req:=jsonb_build_object('out_trade_no_hex','61','transaction_id_hex','','out_refund_no_hex','62','reason_hex','63','notify_url_hex','64','amount_cents','100','total_cents','100','currency','CNY');
 obs:=jsonb_build_object('merchant_id_hex','65','out_trade_no_hex','61','transaction_id_hex','','out_refund_no_hex','62','refund_id_hex','66','state','PROCESSING','amount_cents','100','total_cents','100','currency','CNY','success_time',jsonb_build_object('epoch_seconds','-62135596800','nanosecond',0,'zero',true,'canonical','0001-01-01T00:00:00Z'));
 SET LOCAL ROLE authenticated;
 result:=order_refund_finish_create_v1(1,repeat('ab',16),3,req,false,obs,wire);
 RESET ROLE;
 IF result->>'outcome'<>'FINISHED' THEN RAISE EXCEPTION 'normal create completion rejected: %',result; END IF;
 IF NOT EXISTS(SELECT 1 FROM refunds WHERE id=1 AND provider_state='PROCESSING' AND next_query_at IS NULL AND record_version=4 AND lease_kind IS NULL) THEN RAISE EXCEPTION 'normal refund create entered compensation'; END IF;
 obs:=obs||jsonb_build_object('merchant_id_hex',encode(convert_to('test-merchant','UTF8'),'hex'),'out_trade_no_hex',encode(convert_to('test-order-3','UTF8'),'hex'),'transaction_id_hex',encode(convert_to('transaction-3','UTF8'),'hex'),'out_refund_no_hex',encode(convert_to('refund-create-test','UTF8'),'hex'));
 SET LOCAL ROLE authenticated;
 result:=order_refund_observe_v1('{"refund_id":"1","prepayment_id":"3","order_id":"3"}'::jsonb,'ACTIVE_QUERY','',obs,repeat('77',32),NULL,wire-'receipt_time',false);
 RESET ROLE;
 IF result->>'outcome'<>'OBSERVED' THEN RAISE EXCEPTION 'processing observation rejected: %',result; END IF;
 IF EXISTS(SELECT 1 FROM refunds WHERE id=1 AND next_query_at IS NOT NULL) THEN RAISE EXCEPTION 'processing observation scheduled compensation'; END IF;
 UPDATE refunds SET lease_kind='QUERY',lease_owner=decode(repeat('ab',16),'hex'),lease_expires_at=n+interval '30 seconds',record_version=6 WHERE id=1;
 SET LOCAL ROLE authenticated;
 result:=order_refund_observe_v1('{"refund_id":"1","prepayment_id":"3","order_id":"3"}'::jsonb,'ACTIVE_QUERY','',obs,repeat('77',32),'{"refund_id":"1","owner_hex":"abababababababababababababababab","version":"6"}'::jsonb,wire-'receipt_time',false);
 RESET ROLE;
 IF result->>'outcome'<>'OBSERVED' THEN RAISE EXCEPTION 'duplicate processing rejected: %',result; END IF;
 IF EXISTS(SELECT 1 FROM refunds WHERE id=1 AND (next_query_at IS NOT NULL OR lease_kind IS NOT NULL)) THEN RAISE EXCEPTION 'duplicate processing left compensation'; END IF;
 UPDATE refunds SET provider_state='CREATE_CLAIMED',provider_refund_id=NULL,lease_kind='CREATE',lease_owner=decode(repeat('ab',16),'hex'),lease_expires_at=n+interval '30 seconds',record_version=8 WHERE id=1;
 SET LOCAL ROLE authenticated;
 result:=order_refund_finish_create_v1(1,repeat('ab',16),8,req,false,obs||'{"amount_cents":"101"}'::jsonb,wire);
 RESET ROLE;
 IF result->>'outcome'<>'FINISHED' THEN RAISE EXCEPTION 'mismatch completion rejected: %',result; END IF;
 IF NOT EXISTS(SELECT 1 FROM refunds WHERE id=1 AND materialization_state='PENDING_MANUAL' AND pending_reason_code IS NOT NULL AND next_query_at IS NULL) THEN RAISE EXCEPTION 'manual mismatch requeued automatically'; END IF;


 END $test$;
 ROLLBACK;`)
	})

	t.Run("refund-failed-query-retry-persisted", func(t *testing.T) {
		sql(`DO $test$
 DECLARE n timestamptz:=date_trunc('second',statement_timestamp()); wire jsonb; result jsonb;
 BEGIN
 INSERT INTO refunds(id,prepayment_id,order_id,out_refund_no,idempotency_key_hash,amount_cents,currency,reason_code,requested_by_user_id,provider_request_json,provider_request_digest,provider_state,create_attempted_at,materialization_state,lease_kind,lease_owner,lease_expires_at,record_version,created_at,updated_at)
 VALUES(1,3,3,convert_to('refund-retry-test','UTF8'),decode(repeat('11',32),'hex'),100,'CNY','USER_CANCEL',1,'{}',decode(repeat('22',32),'hex'),'CREATE_UNKNOWN',n,'AWAITING_PROVIDER','QUERY',decode(repeat('ab',16),'hex'),n+interval '30 seconds',1,n,n);
 wire:=jsonb_build_object('epoch_seconds',extract(epoch FROM n)::bigint::text,'nanosecond',0,'zero',false,'canonical',to_char(n,'YYYY-MM-DD"T"HH24:MI:SS"Z"'));
 SET LOCAL ROLE authenticated;
 result:=order_refund_release_query_v1(1,repeat('ab',16),1,wire);
 IF result->>'outcome'<>'RELEASED' THEN RAISE EXCEPTION 'failed-query release: %',result; END IF;
 -- Duplicate completion from a stale instance must not extend retry or increment version.
 PERFORM order_refund_release_query_v1(1,repeat('ab',16),1,wire);
 RESET ROLE;
 IF NOT EXISTS(SELECT 1 FROM refunds WHERE id=1 AND next_query_at=n+interval '30 minutes' AND record_version=2 AND lease_kind IS NULL) THEN RAISE EXCEPTION 'retry is not persisted at failure + 30 minutes'; END IF;
 END $test$;`)
		run("pg_ctl", "-D", dir+"/data", "-l", dir+"/log", "-m", "fast", "-w", "restart")
		sql(`DO $test$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM refunds WHERE id=1 AND next_query_at=updated_at+interval '30 minutes' AND record_version=2 AND lease_kind IS NULL) THEN RAISE EXCEPTION 'retry lost after database restart'; END IF;
 END $test$;`)
	})

}
