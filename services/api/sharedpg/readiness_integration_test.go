package sharedpg

import (
	"context"
	"encoding/json"
	"github.com/gaofeng30/order/services/api/internal/billing"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/migrate"
)

// Exercise the complete readiness contract against a fresh native catalog,
// including registry order, grants and the authenticated RPC response.
func TestInstalledCatalogPassesReadiness(t *testing.T) {
	for _, environment := range []string{"uat", "production"} {
		t.Run(environment, func(t *testing.T) { testInstalledCatalog(t, environment) })
	}
}

func testInstalledCatalog(t *testing.T, environment string) {
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
	run("pg_ctl", "-D", dir+"/data", "-l", dir+"/log", "-o", "-k "+dir+" -p 55492 -c listen_addresses=''", "-w", "start")
	defer run("pg_ctl", "-D", dir+"/data", "-m", "immediate", "-w", "stop")
	sql := func(s string) []byte {
		return run("psql", "-X", "-q", "-A", "-t", "-h", dir, "-p", "55492", "-d", "postgres", "-v", "ON_ERROR_STOP=1", "-c", strings.ReplaceAll(s, "'uat'", "'"+environment+"'"))
	}
	schema, err := migrate.SharedPostgreSQLInstallSQL()
	if err != nil {
		t.Fatal(err)
	}
	// Use stdin for the large schema/install statements; avoid command-size limits.
	install := func(s string) {
		t.Helper()
		cmd := exec.Command(filepath.Join(bin, "psql"), "-X", "-q", "-h", dir, "-p", "55492", "-d", "postgres", "-v", "ON_ERROR_STOP=1")
		f := dir + "/install.sql"
		if err := os.WriteFile(f, []byte(s), 0600); err != nil {
			t.Fatal(err)
		}
		cmd.Args = append(cmd.Args, "-f", f)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("install: %v\n%s", err, out)
		}
	}
	if environment == "production" {
		// Prod already has the verified platform pgcrypto dependency installed.
		// Initial installation must preserve it without accepting unrelated objects.
		sql("CREATE EXTENSION pgcrypto;")
		sql("CREATE FUNCTION public.pgcrypto_unreviewed() RETURNS integer LANGUAGE sql AS $$ SELECT 1 $$;")
		cmd := exec.Command(filepath.Join(bin, "psql"), "-X", "-q", "-h", dir, "-p", "55492", "-d", "postgres", "-v", "ON_ERROR_STOP=1")
		cmd.Stdin = strings.NewReader(schema)
		out, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(out), "shared_install_public_not_empty") {
			t.Fatalf("unreviewed function accepted: %v %s", err, out)
		}
		if string(sql("SELECT to_regclass('public.schema_migrations') IS NULL;")) != "t\n" {
			t.Fatal("rejected installation left partial migration state")
		}
		sql("DROP FUNCTION public.pgcrypto_unreviewed();")
	}
	install(schema)
	sql("CREATE ROLE anon; CREATE ROLE authenticated; CREATE SCHEMA auth; CREATE FUNCTION auth.uid() RETURNS text LANGUAGE sql AS $$ SELECT '123'::text $$; CREATE FUNCTION auth.role() RETURNS text LANGUAGE sql AS $$ SELECT current_user::text $$; GRANT USAGE ON SCHEMA auth TO authenticated,anon;")
	definition, err := InstallSQL("123", environment, "https://example.invalid/api/v1/payments/wechat/notify", "https://example.invalid/api/v1/refunds/wechat/notify")
	if err != nil {
		t.Fatal(err)
	}
	install(definition)
	p := catalogProbe{raw: sql("SET ROLE authenticated; SELECT public.order_shared_pg_readiness_v1();")}
	if environment == "production" && Check(context.Background(), p, "123", environment, "https://example.invalid/api/v1/payments/wechat/notify", "https://example.invalid/api/v1/refunds/wechat/notify") == nil {
		t.Fatal("production became ready without the durable compensation installation")
	}
	// This fixture validates the base business catalog independently. The public
	// production gate additionally requires the scheduler and real extensions.
	if err := checkBusinessCatalog(context.Background(), p, "123", environment, "https://example.invalid/api/v1/payments/wechat/notify", "https://example.invalid/api/v1/refunds/wechat/notify", false); err != nil {
		t.Fatal(err)
	}
	otherEnvironment := "production"
	if environment == "production" {
		otherEnvironment = "uat"
	}
	if Check(context.Background(), p, "123", otherEnvironment, "https://example.invalid/api/v1/payments/wechat/notify", "https://example.invalid/api/v1/refunds/wechat/notify") == nil {
		t.Fatal("catalog accepted wrong environment")
	}
	if Check(context.Background(), p, "999", environment, "https://example.invalid/api/v1/payments/wechat/notify", "https://example.invalid/api/v1/refunds/wechat/notify") == nil {
		t.Fatal("catalog accepted wrong service UID")
	}
	t.Run("real-reconcile-receipt-reused-after-service-restart", func(t *testing.T) {
		rpc := billingSQLProbe{sql: sql}
		provider := billing.NewFakeBillProvider()
		date := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
		provider.SetCompleteBill(date, []wechatpay.BillEntry{})
		cfg := billing.Config{Environment: environment, AppID: "wx15e136b256d4dc5a", MerchantID: "lab", PaymentNotifyURL: "https://example.invalid/api/v1/payments/wechat/notify", RefundNotifyURL: "https://example.invalid/api/v1/refunds/wechat/notify"}
		first, err := billing.NewSharedPG(rpc, provider, cfg).RunReconcile(context.Background(), date, 100)
		if err != nil {
			t.Fatal(err)
		}
		provider.SetUnavailable(true)
		second, err := billing.NewSharedPG(rpc, provider, cfg).RunReconcile(context.Background(), date, 100)
		if err != nil || second.Digest != first.Digest || second.Matched != first.Matched || !second.BillDate.Equal(first.BillDate) {
			t.Fatalf("durable result lost: %+v err %v", second, err)
		}
	})

	t.Run("completed-bill-is-scoped-and-latest-result-wins", func(t *testing.T) {
		completed := func(merchant string) bool {
			var reply struct {
				Completed bool   `json:"completed"`
				Outcome   string `json:"outcome"`
			}
			raw := sql("SET ROLE authenticated; SELECT public.order_billing_completed_v1('2026-10-13','uat','wx15e136b256d4dc5a','" + merchant + "');")
			if json.Unmarshal(raw, &reply) != nil || reply.Outcome != "SUCCEEDED" {
				t.Fatalf("invalid reply %s", raw)
			}
			return reply.Completed
		}
		if completed("lab") {
			t.Fatal("missing receipt completed")
		}
		sql(`INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,action,target_type,target_key_hash,result,reason_code,after_state_json,occurred_at) VALUES('SYSTEM_EVIDENCE','PROVIDER',sha256(convert_to('PROVIDER:BILLING','UTF8')),'billing.reconcile','BILL',sha256(convert_to('bill1','UTF8')),'SUCCEEDED','BILL_MATCHED',jsonb_build_object('environment','uat','appid','wx15e136b256d4dc5a','merchant_id','lab','date','2026-10-13','matched',3,'bill_digest',repeat('01',32)),clock_timestamp());`)
		if !completed("lab") || completed("other") {
			t.Fatal("completion scope mismatch")
		}
		sql(`INSERT INTO public.action_audits(entry_kind,actor_kind,actor_scope_hash,action,target_type,target_key_hash,result,reason_code,after_state_json,occurred_at) VALUES('SYSTEM_EVIDENCE','PROVIDER',sha256(convert_to('PROVIDER:BILLING','UTF8')),'billing.reconcile','BILL',sha256(convert_to('bill2','UTF8')),'REJECTED','BILL_MISMATCH',jsonb_build_object('environment','uat','appid','wx15e136b256d4dc5a','merchant_id','lab','date','2026-10-13','matched',3,'bill_digest',repeat('01',32)),clock_timestamp());`)
		if completed("lab") {
			t.Fatal("newer mismatch hidden by old success")
		}
	})
	if environment == "production" {
		t.Run("combined-business-and-compensation-catalog", func(t *testing.T) {
			// Native PG has no Tencent transport. These are explicit transactional
			// adapters; the real extension receipt is validated separately in Prod.
			install(`CREATE SCHEMA cron;CREATE TABLE cron.job(jobid bigserial PRIMARY KEY,jobname text UNIQUE,schedule text,command text);
CREATE FUNCTION cron.schedule(text,text,text) RETURNS bigint LANGUAGE sql AS $$ INSERT INTO cron.job(jobname,schedule,command) VALUES($1,$2,$3) ON CONFLICT(jobname) DO UPDATE SET schedule=$2,command=$3 RETURNING jobid $$;
CREATE FUNCTION cron.unschedule(text) RETURNS boolean LANGUAGE sql AS $$ DELETE FROM cron.job WHERE jobname=$1 RETURNING true $$;
CREATE SCHEMA tencentdb_scf;
CREATE FUNCTION tencentdb_scf.tencentdb_scf_post(text,jsonb,text,boolean,integer) RETURNS bigint LANGUAGE sql AS $$ SELECT 1::bigint $$;
CREATE FUNCTION tencentdb_scf.tencentdb_scf_result(bigint) RETURNS jsonb LANGUAGE sql AS $$ SELECT NULL::jsonb $$;`)
			compensation, err := CompensationSQL("123")
			if err != nil {
				t.Fatal(err)
			}
			install(compensation)
			business := sql("SET ROLE authenticated; SELECT public.order_shared_pg_readiness_v1();")
			if err := checkBusinessCatalog(context.Background(), catalogProbe{raw: business}, "123", environment, "https://example.invalid/api/v1/payments/wechat/notify", "https://example.invalid/api/v1/refunds/wechat/notify", true); err != nil {
				t.Fatal("base catalog rejected the reviewed capability functions", err)
			}
			actual := sql("SET ROLE authenticated; SELECT public.order_compensation_readiness_v1();")
			if verifyCompensationObjects(actual, compensation, "123") != nil {
				t.Fatal("full business schema scheduler mismatch")
			}
			probe := multipleCatalogProbe{"order_shared_pg_readiness_v1": business, "order_compensation_readiness_v1": actual}
			if Check(context.Background(), probe, "123", environment, "https://example.invalid/api/v1/payments/wechat/notify", "https://example.invalid/api/v1/refunds/wechat/notify") == nil {
				t.Fatal("native stubs passed platform gate")
			}
			// Only the platform-extension metadata is a fixture in this final gate
			// check. All business functions, privileges, constraints and triggers
			// above were installed and queried from the isolated real PG instance.
			var c map[string]json.RawMessage
			if json.Unmarshal(actual, &c) != nil {
				t.Fatal("invalid catalog")
			}
			var functions []function
			if json.Unmarshal(c["functions"], &functions) != nil || len(functions) == 0 {
				t.Fatal("missing routines")
			}
			c["extensions"], _ = json.Marshal([]map[string]string{
				{"name": "pg_cron", "version": "1.6", "schema": "pg_catalog", "owner": functions[0].Owner},
				{"name": "pgcrypto", "version": "1.3", "schema": "public", "owner": functions[0].Owner},
				{"name": "tencentdb_scf", "version": "1.1", "schema": "tencentdb_scf", "owner": functions[0].Owner},
			})
			probe["order_compensation_readiness_v1"], _ = json.Marshal(c)
			if err := Check(context.Background(), probe, "123", environment, "https://example.invalid/api/v1/payments/wechat/notify", "https://example.invalid/api/v1/refunds/wechat/notify"); err != nil {
				t.Fatal("combined catalog gate rejected valid fixture", err)
			}
		})
	}
}

type multipleCatalogProbe map[string][]byte

func (p multipleCatalogProbe) CallRPC(_ context.Context, name string, _ map[string]any, out any) error {
	return json.Unmarshal(p[name], out)
}

type catalogProbe struct{ raw []byte }

func (p catalogProbe) CallRPC(_ context.Context, _ string, _ map[string]any, out any) error {
	return json.Unmarshal(p.raw, out)
}

// Local test adapter calls the real installed RPCs; only the external bill provider is fake.
type billingSQLProbe struct{ sql func(string) []byte }

func (p billingSQLProbe) CallRPC(_ context.Context, name string, args map[string]any, out any) error {
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		value, ok := args[k].(string)
		suffix := ""
		if !ok {
			raw, err := json.Marshal(args[k])
			if err != nil {
				return err
			}
			value = string(raw)
			suffix = "::jsonb"
		}
		parts = append(parts, k+"=>'"+strings.ReplaceAll(value, "'", "''")+"'"+suffix)
	}
	raw := p.sql("SET ROLE authenticated; SELECT public." + name + "(" + strings.Join(parts, ",") + ");")
	return json.Unmarshal(raw, out)
}
