package sharedpg

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Real PostgreSQL verifies transactional state and capability CAS. The cron and
// function transport adapters below only record calls; platform delivery has its
// own live receipt and is not certified by this test.
func TestDurableCompensationCapabilities(t *testing.T) {
	bin := os.Getenv("ORDER_TEST_PG_BIN")
	if bin == "" {
		t.Skip("set ORDER_TEST_PG_BIN")
	}
	dir, err := os.MkdirTemp("", "order-wake-")
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
	run("pg_ctl", "-D", dir+"/data", "-l", dir+"/log", "-o", "-k "+dir+" -p 55495 -c listen_addresses=''", "-w", "start")
	defer run("pg_ctl", "-D", dir+"/data", "-m", "immediate", "-w", "stop")
	sql := func(source string) string {
		t.Helper()
		cmd := exec.Command(filepath.Join(bin, "psql"), "-X", "-qAt", "-h", dir, "-p", "55495", "-d", "postgres", "-v", "ON_ERROR_STOP=1")
		cmd.Stdin = strings.NewReader(source)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("sql: %v\n%s", err, out)
		}
		return strings.TrimSpace(string(out))
	}
	sql(`CREATE ROLE authenticated; CREATE ROLE anon; CREATE SCHEMA auth;
 CREATE FUNCTION auth.uid() RETURNS text LANGUAGE sql AS $$ SELECT current_setting('request.jwt.claim.sub',true) $$;
 CREATE FUNCTION auth.role() RETURNS text LANGUAGE sql AS $$ SELECT current_setting('request.jwt.claim.role',true) $$;
 GRANT USAGE ON SCHEMA auth TO authenticated,anon;
 CREATE FUNCTION auth.jwt() RETURNS jsonb LANGUAGE sql AS $$ SELECT current_setting('request.jwt.claims',true)::jsonb $$;
 CREATE EXTENSION pgcrypto;
 CREATE SCHEMA cron;CREATE TABLE cron.job(jobid bigserial PRIMARY KEY,jobname text UNIQUE,schedule text,command text);
 CREATE FUNCTION cron.schedule(text,text,text) RETURNS bigint LANGUAGE sql AS $$ INSERT INTO cron.job(jobname,schedule,command) VALUES($1,$2,$3) ON CONFLICT(jobname) DO UPDATE SET schedule=$2,command=$3 RETURNING jobid $$;
 CREATE FUNCTION cron.unschedule(text) RETURNS boolean LANGUAGE sql AS $$ DELETE FROM cron.job WHERE jobname=$1 RETURNING true $$;
 CREATE SCHEMA tencentdb_scf;CREATE TABLE tencentdb_scf.requests(id bigserial,path text,body jsonb);
 CREATE FUNCTION tencentdb_scf.tencentdb_scf_post(text,jsonb,text,boolean,integer) RETURNS bigint LANGUAGE sql AS $$ INSERT INTO tencentdb_scf.requests(path,body) VALUES($1,$3::jsonb) RETURNING id $$;
 CREATE TABLE tencentdb_scf.responses(id bigint PRIMARY KEY,result jsonb);
 CREATE FUNCTION tencentdb_scf.tencentdb_scf_result(bigint) RETURNS jsonb LANGUAGE sql AS $$ SELECT result FROM tencentdb_scf.responses WHERE id=$1 $$;
 CREATE TABLE tencentdb_scf.test_auth(key text);
 CREATE FUNCTION tencentdb_scf.set_cloudbase_env_id(text) RETURNS boolean LANGUAGE sql AS $$ SELECT $1='order-prod-d0g4xnw49c5ada5ce' $$;
 CREATE FUNCTION tencentdb_scf.set_cloudbase_api_key(text) RETURNS boolean LANGUAGE plpgsql AS $$ BEGIN DELETE FROM tencentdb_scf.test_auth;INSERT INTO tencentdb_scf.test_auth VALUES($1);RETURN true;END $$;
 CREATE TABLE prepayments(id bigint PRIMARY KEY,materialization_state text,next_reconcile_at timestamptz,lease_expires_at timestamptz);
 CREATE TABLE refunds(id bigint PRIMARY KEY,provider_state text,next_query_at timestamptz,lease_expires_at timestamptz);
 CREATE TABLE notification_outbox(id bigint PRIMARY KEY,state text,next_attempt_at timestamptz,lease_expires_at timestamptz,
 order_id bigint DEFAULT 1,recipient_user_id bigint DEFAULT 1,kind text DEFAULT 'READY',template_config_version bigint DEFAULT 1,record_version bigint DEFAULT 1,attempt_count integer DEFAULT 0,
 immutable_message_json jsonb DEFAULT '{"order_number":"test","pickup_point":"test","pickup_date":"2030-01-01","pickup_time":"12:00"}',lease_owner bytea,last_error_code text);
 GRANT SELECT,UPDATE ON notification_outbox TO authenticated;`)
	definition, err := CompensationSQL("123")
	if err != nil {
		t.Fatal(err)
	}
	// A late installation error must not leave a partial private schema,
	// triggers or publicly callable capability functions behind.
	broken := strings.Replace(definition, "public.order_compensation_readiness_v1()", "public.order_compensation_readiness_v1(INVALID_ARGUMENT)", 1)
	failed := exec.Command(filepath.Join(bin, "psql"), "-X", "-qAt", "-h", dir, "-p", "55495", "-d", "postgres", "-v", "ON_ERROR_STOP=1")
	failed.Stdin = strings.NewReader(broken)
	if out, err := failed.CombinedOutput(); err == nil {
		t.Fatalf("broken installation unexpectedly succeeded: %s", out)
	}
	if sql("SELECT to_regnamespace('order_compensation') IS NULL AND to_regprocedure('public.order_compensation_claim_v1(text)') IS NULL") != "t" {
		t.Fatal("late installation failure left partial scheduler state")
	}
	sql(definition)
	t.Run("credential-renewal-rpc-exists", func(t *testing.T) {
		if sql("SELECT to_regprocedure('public.order_compensation_authenticate_v1()') IS NOT NULL") != "t" {
			t.Fatal("durable transport has no credential renewal RPC")
		}
	})
	t.Run("credential-register-and-no-expiry-regression", func(t *testing.T) {
		sql(`BEGIN;DO $$ DECLARE c jsonb;token text;first_token text;exp bigint:=extract(epoch FROM clock_timestamp())::bigint+7200;BEGIN
 PERFORM set_config('request.jwt.claim.sub','123',true);PERFORM set_config('request.jwt.claim.role','authenticated',true);
 FOR n IN 0..1 LOOP
 c:=jsonb_build_object('sub','123','role','authenticated','aud','order-prod-d0g4xnw49c5ada5ce','exp',exp-n*3600);
 token:='e30.'||rtrim(translate(replace(encode(convert_to(c::text,'UTF8'),'base64'),E'\n',''),'+/','-_'),'=')||'.sig';
 PERFORM set_config('request.jwt.claims',c::text,true);PERFORM set_config('request.headers',jsonb_build_object('authorization','Bearer '||token)::text,true);
 IF n=0 THEN first_token:=token;END IF;
 SET LOCAL ROLE authenticated;
 IF public.order_compensation_authenticate_v1()->>'outcome'<>'REGISTERED' THEN RAISE EXCEPTION 'not registered';END IF;
 RESET ROLE;
 END LOOP;
 IF (SELECT key FROM tencentdb_scf.test_auth) IS DISTINCT FROM first_token THEN RAISE EXCEPTION 'older token overwrote new token';END IF;
 IF has_table_privilege('authenticated','order_compensation.authentication','SELECT') OR has_function_privilege('anon','public.order_compensation_authenticate_v1()','EXECUTE') THEN RAISE EXCEPTION 'excess credential access';END IF;
 END $$;ROLLBACK;`)
	})
	t.Run("ordinary-identity-can-inspect-scheduler-catalog-without-reading-capabilities", func(t *testing.T) {
		raw := sql(`BEGIN;SET LOCAL ROLE authenticated;SET LOCAL "request.jwt.claim.sub"='123';SET LOCAL "request.jwt.claim.role"='authenticated';SELECT public.order_compensation_readiness_v1();COMMIT;`)
		if err := verifyCompensationObjects([]byte(raw), definition, "123"); err != nil {
			t.Fatalf("scheduler catalog rejected: %v\n%s", err, raw)
		}
		if CheckCompensation(context.Background(), catalogProbe{raw: []byte(raw)}, "123") == nil {
			t.Fatal("transport stubs were accepted as installed platform extensions")
		}
		var catalog struct {
			Role, UID string
			Functions []function
			Triggers  []struct {
				Name, Table, Enabled, Function, Arguments string
				Type                                      int
			}
		}
		if json.Unmarshal([]byte(raw), &catalog) != nil || catalog.Role != "authenticated" || catalog.UID != "123" || len(catalog.Functions) != 8 || len(catalog.Triggers) != 3 {
			t.Fatal("incomplete authenticated scheduler catalog")
		}
		for _, f := range catalog.Functions {
			if f.Hash == "" || f.PublicExecute || f.AnonExecute || f.AuthenticatedExecute != strings.HasPrefix(f.Name, "public.") {
				t.Fatalf("catalog reported incorrect routine boundary: %s", f.Name)
			}
		}
		for _, trigger := range catalog.Triggers {
			if trigger.Enabled != "O" || trigger.Type != 29 || trigger.Function != "order_compensation.changed" || trigger.Arguments == "" {
				t.Fatalf("incorrect trigger catalog: %+v", trigger)
			}
		}
		for _, mutation := range []string{
			"GRANT USAGE ON SCHEMA cron TO authenticated;",
			"ALTER TABLE order_compensation.wakes ENABLE ROW LEVEL SECURITY; ALTER TABLE order_compensation.wakes FORCE ROW LEVEL SECURITY;",
			"CREATE FUNCTION public.unreviewed_trigger() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RETURN NEW;END $$; CREATE TRIGGER unreviewed AFTER UPDATE ON prepayments FOR EACH ROW EXECUTE FUNCTION public.unreviewed_trigger();",
			"GRANT USAGE ON SCHEMA order_compensation TO authenticated;",
			"GRANT SELECT ON order_compensation.wakes TO authenticated;",
			"GRANT SELECT ON order_compensation.authentication TO authenticated;",
			"GRANT EXECUTE ON FUNCTION order_compensation.dispatch(text,numeric) TO authenticated;",
			"ALTER FUNCTION order_compensation.dispatch(text,numeric) SET search_path=public;",
			"ALTER TABLE order_compensation.wakes DROP CONSTRAINT wakes_token_hash_check;",
			"ALTER TABLE order_compensation.wakes ALTER COLUMN claimed SET DEFAULT true;",
			"ALTER TABLE prepayments DISABLE TRIGGER order_compensation_payment;",
			"CREATE FUNCTION order_compensation.unreviewed() RETURNS void LANGUAGE sql AS $$ SELECT $$;",
		} {
			changed := sql(`BEGIN;` + mutation + `SET LOCAL ROLE authenticated;SET LOCAL "request.jwt.claim.sub"='123';SET LOCAL "request.jwt.claim.role"='authenticated';SELECT public.order_compensation_readiness_v1();ROLLBACK;`)
			if verifyCompensationObjects([]byte(changed), definition, "123") == nil {
				t.Fatalf("catalog accepted drift: %s", mutation)
			}
		}
	})
	t.Run("empty-normal-and-rollback-do-not-schedule", func(t *testing.T) {
		sql(`INSERT INTO prepayments VALUES(1,'AWAITING_PAYMENT',NULL,NULL);INSERT INTO refunds VALUES(1,'PROCESSING',NULL,NULL);
  BEGIN; INSERT INTO notification_outbox VALUES(1,'PENDING',now(),NULL); ROLLBACK;
  DO $$ BEGIN IF EXISTS(SELECT 1 FROM cron.job) OR EXISTS(SELECT 1 FROM tencentdb_scf.requests) THEN RAISE EXCEPTION 'normal or rolled back work scheduled'; END IF; END $$;`)
	})
	t.Run("initial-send-claim-and-stop", func(t *testing.T) {
		sql(`INSERT INTO notification_outbox VALUES(1,'PENDING',now(),NULL);
  DO $$ DECLARE token text; result jsonb; BEGIN
   SELECT body->>'token' INTO token FROM tencentdb_scf.requests ORDER BY id DESC LIMIT 1;
   IF token IS NULL THEN RAISE EXCEPTION 'initial notification not woken'; END IF;
   PERFORM set_config('request.jwt.claim.sub','123',true);PERFORM set_config('request.jwt.claim.role','authenticated',true);
   SET LOCAL ROLE authenticated;
   result:=public.order_compensation_claim_v1(token);
   IF result->>'kind'<>'subscription' OR result->>'outcome'<>'CLAIMED' THEN RAISE EXCEPTION 'claim failed %',result;END IF;
   IF public.order_compensation_claim_v1(token)->>'outcome'<>'REJECTED' THEN RAISE EXCEPTION 'replayed capability accepted';END IF;
   RESET ROLE;
   UPDATE notification_outbox SET state='SENT',next_attempt_at=NULL WHERE id=1;
   IF EXISTS(SELECT 1 FROM cron.job) THEN RAISE EXCEPTION 'completed work still scheduled';END IF;
  END $$;`)
	})
	t.Run("failure-waits-thirty-minutes-and-success-stops", func(t *testing.T) {
		sql(`INSERT INTO notification_outbox VALUES(2,'PENDING',clock_timestamp(),NULL);
  DO $$ DECLARE token text; started timestamptz:=clock_timestamp(); before_count bigint; BEGIN
   SELECT body->>'token' INTO token FROM tencentdb_scf.requests ORDER BY id DESC LIMIT 1;
   SELECT count(*) INTO before_count FROM tencentdb_scf.requests;
   PERFORM set_config('request.jwt.claim.sub','123',true);PERFORM set_config('request.jwt.claim.role','authenticated',true);
   SET LOCAL ROLE authenticated;
   IF public.order_compensation_claim_v1(token)->>'outcome'<>'CLAIMED' THEN RAISE EXCEPTION 'not claimed'; END IF;
   IF public.order_compensation_finish_v1(token,false)->>'outcome'<>'FINISHED' THEN RAISE EXCEPTION 'not finished';END IF;
   RESET ROLE;
   IF NOT EXISTS(SELECT 1 FROM order_compensation.wakes WHERE kind='subscription' AND retry_after>=started+interval '30 minutes') THEN RAISE EXCEPTION 'early retry'; END IF;
   PERFORM order_compensation.dispatch('subscription',2);
   IF (SELECT count(*) FROM tencentdb_scf.requests)<>before_count THEN RAISE EXCEPTION 'failure sent immediately';END IF;
   UPDATE notification_outbox SET state='SENT',next_attempt_at=NULL WHERE id=2;
   IF EXISTS(SELECT 1 FROM cron.job) THEN RAISE EXCEPTION 'terminal intent scheduled'; END IF;
  END $$;`)
	})

	t.Run("two-processes-cannot-consume-the-same-capability", func(t *testing.T) {
		sql(`INSERT INTO notification_outbox VALUES(3,'PENDING',clock_timestamp(),NULL);`)
		token := sql(`SELECT body->>'token' FROM tencentdb_scf.requests ORDER BY id DESC LIMIT 1`)
		results := make(chan string, 2)
		for i := 0; i < 2; i++ {
			go func() {
				cmd := exec.Command(filepath.Join(bin, "psql"), "-X", "-qAt", "-h", dir, "-p", "55495", "-d", "postgres", "-v", "ON_ERROR_STOP=1")
				cmd.Stdin = strings.NewReader(`BEGIN; SET LOCAL ROLE authenticated; SET LOCAL "request.jwt.claim.sub"='123'; SET LOCAL "request.jwt.claim.role"='authenticated'; SELECT public.order_compensation_claim_v1('` + token + `')->>'outcome'; COMMIT;`)
				out, err := cmd.CombinedOutput()
				if err != nil {
					results <- "ERROR " + string(out)
					return
				}
				results <- strings.TrimSpace(string(out))
			}()
		}
		outcomes := map[string]int{}
		for i := 0; i < 2; i++ {
			outcomes[<-results]++
		}
		if outcomes["CLAIMED"] != 1 || outcomes["REJECTED"] != 1 {
			t.Fatalf("outcomes=%v", outcomes)
		}
	})
	t.Run("database-restart-preserves-unfinished-delivery-and-recovers-expired-claim", func(t *testing.T) {
		old := sql(`SELECT body->>'token' FROM tencentdb_scf.requests ORDER BY id DESC LIMIT 1`)
		run("pg_ctl", "-D", dir+"/data", "-m", "immediate", "-w", "stop")
		run("pg_ctl", "-D", dir+"/data", "-l", dir+"/log", "-o", "-k "+dir+" -p 55495 -c listen_addresses=''", "-w", "start")
		sql(`DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM order_compensation.wakes WHERE kind='subscription' AND claimed AND next_at IS NOT NULL) OR NOT EXISTS(SELECT 1 FROM cron.job WHERE jobname='order-compensation-subscription-3') THEN RAISE EXCEPTION 'restart lost unfinished work';END IF;END $$;
  UPDATE order_compensation.wakes SET token_until=clock_timestamp()-interval '1 second' WHERE kind='subscription';SELECT order_compensation.dispatch('subscription',3);`)
		fresh := sql(`SELECT body->>'token' FROM tencentdb_scf.requests ORDER BY id DESC LIMIT 1`)
		if fresh == old {
			t.Fatal("expired delivery did not receive new capability")
		}
		result := sql(`BEGIN;SET LOCAL ROLE authenticated;SET LOCAL "request.jwt.claim.sub"='123';SET LOCAL "request.jwt.claim.role"='authenticated';SELECT public.order_compensation_claim_v1('` + old + `')->>'outcome';COMMIT;`)
		if result != "REJECTED" {
			t.Fatal("expired capability accepted")
		}
		sql(`UPDATE notification_outbox SET state='SENT',next_attempt_at=NULL WHERE id=3;DO $$ BEGIN IF EXISTS(SELECT 1 FROM cron.job) THEN RAISE EXCEPTION 'success did not remove timer'; END IF;END $$;`)
	})

	t.Run("new-notification-is-not-delayed-by-an-older-failed-delivery", func(t *testing.T) {
		sql(`INSERT INTO notification_outbox VALUES(4,'PENDING',clock_timestamp(),NULL);
  DO $$ DECLARE token text; BEGIN
   SELECT body->>'token' INTO token FROM tencentdb_scf.requests ORDER BY id DESC LIMIT 1;
   PERFORM set_config('request.jwt.claim.sub','123',true);PERFORM set_config('request.jwt.claim.role','authenticated',true);
   SET LOCAL ROLE authenticated;PERFORM public.order_compensation_claim_v1(token);PERFORM public.order_compensation_finish_v1(token,false);RESET ROLE;
  END $$;`)
		before := sql(`SELECT count(*) FROM tencentdb_scf.requests`)
		sql(`INSERT INTO notification_outbox VALUES(5,'PENDING',clock_timestamp(),NULL);`)
		after := sql(`SELECT count(*) FROM tencentdb_scf.requests`)
		if after == before {
			t.Fatal("old failure delayed new initial notification")
		}
	})

	t.Run("refund-before-provider-claim-is-durable", func(t *testing.T) {
		before := sql(`SELECT count(*) FROM tencentdb_scf.requests`)
		sql(`INSERT INTO refunds VALUES(2,'READY',NULL,NULL);`)
		if sql(`SELECT count(*) FROM tencentdb_scf.requests`) == before {
			t.Fatal("committed refund before provider claim has no recovery wake")
		}
	})

	t.Run("wrong-identity-cannot-consume-or-manage-schedules", func(t *testing.T) {
		sql(`DO $$ DECLARE token text; denied boolean:=false; BEGIN
   IF has_schema_privilege('authenticated','order_compensation','USAGE') OR has_table_privilege('authenticated','order_compensation.wakes','SELECT') OR has_function_privilege('anon','public.order_compensation_claim_v1(text)','EXECUTE') THEN RAISE EXCEPTION 'excess privilege';END IF;
   SELECT body->>'token' INTO token FROM tencentdb_scf.requests ORDER BY id DESC LIMIT 1;
   PERFORM set_config('request.jwt.claim.sub','999',true);PERFORM set_config('request.jwt.claim.role','authenticated',true);
   SET LOCAL ROLE authenticated;
   BEGIN PERFORM public.order_compensation_claim_v1(token);EXCEPTION WHEN insufficient_privilege THEN denied:=true;END;
   RESET ROLE;
   IF NOT denied THEN RAISE EXCEPTION 'wrong UID accepted';END IF;
   IF EXISTS(SELECT 1 FROM order_compensation.wakes WHERE kind='refund' AND entity_id=2 AND claimed) THEN RAISE EXCEPTION 'unauthorized claim mutated state';END IF;
  END $$;`)
	})

	t.Run("notification-claim-selects-exact-record-instead-of-older-due-work", func(t *testing.T) {
		sql(`DO $$ DECLARE at_time jsonb:='{"epoch_seconds":"1893456000","nanosecond":0,"zero":false,"canonical":"2030-01-01T00:00:00Z"}'::jsonb; result jsonb; BEGIN
   PERFORM set_config('request.jwt.claim.sub','123',true);PERFORM set_config('request.jwt.claim.role','authenticated',true);
   SET LOCAL ROLE authenticated;
   result:=public.order_subscription_claim_one_v1(5,at_time,at_time,repeat('ab',16));
   IF result->>'outcome'<>'CLAIMED' OR result->'delivery'->>'id'<>'5' THEN RAISE EXCEPTION 'wrong target claim %',result;END IF;
   RESET ROLE;
   IF (SELECT state FROM notification_outbox WHERE id=4)<>'PENDING' THEN RAISE EXCEPTION 'older failed notification was retried by new wake';END IF;
  END $$;`)
	})
	t.Run("transport-completion-without-business-finish-waits-thirty-minutes", func(t *testing.T) {
		sql(`INSERT INTO notification_outbox VALUES(8,'PENDING',clock_timestamp(),NULL);
  INSERT INTO tencentdb_scf.responses SELECT id,'{"status_code":503}'::jsonb FROM tencentdb_scf.requests ORDER BY id DESC LIMIT 1;
  DO $$ DECLARE observed_at timestamptz:=clock_timestamp(); before_count bigint; BEGIN
   SELECT count(*) INTO before_count FROM tencentdb_scf.requests;
   PERFORM order_compensation.dispatch('subscription',8);
   IF NOT EXISTS(SELECT 1 FROM order_compensation.wakes WHERE kind='subscription' AND entity_id=8 AND retry_after>=observed_at+interval '30 minutes') THEN RAISE EXCEPTION 'transport failure retry is not measured from observed failure'; END IF;
   PERFORM order_compensation.dispatch('subscription',8);
   IF (SELECT count(*) FROM tencentdb_scf.requests)<>before_count THEN RAISE EXCEPTION 'transport failure retried immediately'; END IF;
  END $$;`)
	})
}
