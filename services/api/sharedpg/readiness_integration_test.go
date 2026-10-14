package sharedpg

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

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
}

type catalogProbe struct{ raw []byte }

func (p catalogProbe) CallRPC(_ context.Context, _ string, _ map[string]any, out any) error {
	return json.Unmarshal(p.raw, out)
}
