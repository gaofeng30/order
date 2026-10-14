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
		return run("psql", "-X", "-q", "-A", "-t", "-h", dir, "-p", "55492", "-d", "postgres", "-v", "ON_ERROR_STOP=1", "-c", s)
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
	install(schema)
	sql("CREATE ROLE anon; CREATE ROLE authenticated; CREATE SCHEMA auth; CREATE FUNCTION auth.uid() RETURNS text LANGUAGE sql AS $$ SELECT '123'::text $$; CREATE FUNCTION auth.role() RETURNS text LANGUAGE sql AS $$ SELECT current_user::text $$; GRANT USAGE ON SCHEMA auth TO authenticated,anon;")
	definition, err := InstallSQL("123", "uat", "https://example.invalid/api/v1/payments/wechat/notify", "https://example.invalid/api/v1/refunds/wechat/notify")
	if err != nil {
		t.Fatal(err)
	}
	install(definition)
	p := catalogProbe{raw: sql("SET ROLE authenticated; SELECT public.order_shared_pg_readiness_v1();")}
	if err := Check(context.Background(), p, "123", "uat", "https://example.invalid/api/v1/payments/wechat/notify", "https://example.invalid/api/v1/refunds/wechat/notify"); err != nil {
		t.Fatal(err)
	}
	t.Run("real-reconcile-receipt-reused-after-service-restart", func(t *testing.T) {
		rpc := billingSQLProbe{sql: sql}
		provider := billing.NewFakeBillProvider()
		date := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
		provider.SetCompleteBill(date, []wechatpay.BillEntry{})
		cfg := billing.Config{Environment: "uat", AppID: "wx15e136b256d4dc5a", MerchantID: "lab", PaymentNotifyURL: "https://example.invalid/api/v1/payments/wechat/notify", RefundNotifyURL: "https://example.invalid/api/v1/refunds/wechat/notify"}
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
