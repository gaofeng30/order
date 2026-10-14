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

}
