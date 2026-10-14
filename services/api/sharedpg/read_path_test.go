package sharedpg

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
)

type readinessProbe struct{ called bool }

func (p *readinessProbe) CallRPC(_ context.Context, name string, _ map[string]any, _ any) error {
	p.called = name == "order_shared_pg_readiness_v1"
	return errors.New("offline probe")
}
func TestReadPathInstallAndReadinessRegistry(t *testing.T) {
	definition, err := InstallSQL("123", "uat", "https://example.invalid/api/v1/payments/wechat/notify", "https://example.invalid/api/v1/refunds/wechat/notify")
	if err != nil {
		t.Fatal(err)
	}
	probe := &readinessProbe{}
	_ = Check(context.Background(), probe, "123", "uat", "https://example.invalid/api/v1/payments/wechat/notify", "https://example.invalid/api/v1/refunds/wechat/notify")
	if !probe.called {
		t.Fatal("local definition drift prevents runtime readiness before RPC")
	}
	raw, _ := source.ReadFile("redeem_order.sql")
	for _, match := range regexp.MustCompile(`CREATE OR REPLACE FUNCTION public\.(order_[a-z_0-9]+)\(`).FindAllStringSubmatch(definition, -1) {
		name := match[1]
		if !strings.Contains(string(raw), "'"+name+"'") {
			t.Errorf("function missing from readiness: %s", name)
		}
	}
}
