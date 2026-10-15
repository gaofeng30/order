package sharedpg

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gaofeng30/order/services/api/internal/migrate"
)

// Uses the complete relational schema and production RPC/ACL/RLS definitions in
// a disposable local PostgreSQL. No cloud credentials or network listener.
func TestImmutableImageDurableConfirmation(t *testing.T) {
	bin := os.Getenv("ORDER_TEST_PG_BIN")
	if bin == "" {
		t.Skip("set ORDER_TEST_PG_BIN")
	}
	dir, err := os.MkdirTemp("/tmp", "order-image-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	run := func(name string, args ...string) {
		t.Helper()
		if out, e := exec.Command(filepath.Join(bin, name), args...).CombinedOutput(); e != nil {
			t.Fatalf("%s %v %s", name, e, out)
		}
	}
	run("initdb", "-D", dir+"/db", "--auth=trust", "--no-locale", "--encoding=UTF8")
	opts := "-k " + dir + " -p 55497 -c listen_addresses=''"
	run("pg_ctl", "-D", dir+"/db", "-l", dir+"/log", "-o", opts, "-w", "start")
	defer run("pg_ctl", "-D", dir+"/db", "-m", "immediate", "-w", "stop")
	query := func(s string) (string, error) {
		cmd := exec.Command(filepath.Join(bin, "psql"), "-X", "-qAt", "-h", dir, "-p", "55497", "-d", "postgres", "-v", "ON_ERROR_STOP=1")
		cmd.Stdin = strings.NewReader(s)
		out, e := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), e
	}
	sql := func(s string) string {
		t.Helper()
		out, e := query(s)
		if e != nil {
			t.Fatalf("SQL %v %s", e, out)
		}
		return out
	}
	schema, e := migrate.SharedPostgreSQLInstallSQL()
	if e != nil {
		t.Fatal(e)
	}
	sql(schema)
	sql("CREATE ROLE anon;CREATE ROLE authenticated;CREATE SCHEMA auth;CREATE FUNCTION auth.uid() RETURNS text LANGUAGE sql AS $$ SELECT '123'::text $$;CREATE FUNCTION auth.role() RETURNS text LANGUAGE sql AS $$ SELECT current_user::text $$;GRANT USAGE ON SCHEMA auth TO authenticated,anon;")
	defs, e := InstallSQL("123", "production", "https://example.invalid/api/v1/payments/wechat/notify", "https://example.invalid/api/v1/refunds/wechat/notify")
	if e != nil {
		t.Fatal(e)
	}
	sql(defs)
	sql("INSERT INTO miniprogram_users(id,openid,created_at,last_login_at) VALUES(1,convert_to('isolated-image-user','UTF8'),now(),now());INSERT INTO merchant_accounts(id,phone,name,role,enabled,bound_user_id,bound_at,created_at,updated_at) VALUES(1,convert_to('+10000000000','UTF8'),'isolated owner','OWNER',true,1,now(),now(),now());")
	key := "images/" + strings.Repeat("a", 64) + "/" + strings.Repeat("b", 64) + ".png"
	hash := strings.Repeat("c", 64)
	digest := strings.Repeat("d", 64)
	call := func(actor, keyArg, digestArg, published, version string) string {
		return "SET ROLE authenticated;SELECT order_image_confirmation_commit_v1(" + actor + ",'" + keyArg + "','" + hash + "',NULL,'" + digestArg + "','" + published + "','" + version + "',now());"
	}
	assertOutcome := func(raw, want string) {
		t.Helper()
		var v struct{ Outcome string }
		if json.Unmarshal([]byte(raw), &v) != nil || v.Outcome != want {
			t.Fatalf("want %s got %s", want, raw)
		}
	}
	var wg sync.WaitGroup
	errs := make(chan string, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, e := query(call("1", key, digest, key, ""))
			if e != nil || !strings.Contains(out, `"outcome": "SUCCEEDED"`) {
				errs <- out
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if sql("SELECT count(*) FROM action_audits WHERE action='object.image.confirm'") != "1" {
		t.Fatal("concurrent duplicate receipts")
	}
	run("pg_ctl", "-D", dir+"/db", "-m", "fast", "-l", dir+"/log", "-w", "restart", "-o", opts)
	assertOutcome(sql("SET ROLE authenticated;SELECT order_image_confirmation_replay_v1(1,'"+hash+"');"), "SUCCEEDED")
	assertOutcome(sql(call("1", key, strings.Repeat("e", 64), key, "")), "CONFLICT")
	assertOutcome(sql(call("2", key, digest, key, "")), "UNAVAILABLE")
	assertOutcome(sql(call("1", key, digest, "images/published/"+strings.Repeat("a", 64)+"/"+strings.Repeat("b", 64)+".png", "")), "UNAVAILABLE")
	if sql("SELECT has_function_privilege('anon','order_image_confirmation_commit_v1(numeric,text,text,text,text,text,text,timestamptz)','EXECUTE')") != "f" {
		t.Fatal("anon executable")
	}
	if sql("SELECT count(*) FROM schema_migrations WHERE dirty") != "0" {
		t.Fatal("migration dirty")
	}
	t.Log("real local PG: 16 concurrent confirmations -> 1 durable receipt; database restart replay succeeds; conflicting digest/unbound actor/invalid pin rejected; anon has no EXECUTE")
}
