package main

import (
	"strings"
	"testing"
)

func TestGenerateFunctionInstall(t *testing.T) {
	args := []string{"--functions", "--service-uid", "123", "--environment", "production", "--origin", "https://orders.example.invalid"}
	sql, err := generate(args)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(sql, "CREATE OR REPLACE FUNCTION public.order_") != 117 {
		t.Fatal("wrong function entity count")
	}
	if strings.Contains(sql, "__ORDER_") || strings.Contains(sql, "order-uat-") {
		t.Fatal("unbound marker or UAT binding")
	}
	for _, want := range []string{"https://orders.example.invalid/api/v1/payments/wechat/notify", "https://orders.example.invalid/api/v1/refunds/wechat/notify", "'production'"} {
		if !strings.Contains(sql, want) {
			t.Fatal("missing production binding")
		}
	}
}
func TestGenerateRejectsUnboundInputs(t *testing.T) {
	for _, args := range [][]string{
		{"--functions"},
		{"--service-uid", "123"},
		{"--functions", "--service-uid", "1';DROP", "--environment", "production", "--origin", "https://orders.example.invalid"},
		{"--functions", "--service-uid", "123", "--environment", "production", "--origin", "https://orders.example.invalid?x=1"},
		{"--functions", "--service-uid", "123", "--environment", "production", "--origin", "https://orders.example.invalid/"},
	} {
		if _, err := generate(args); err == nil {
			t.Fatal("invalid generator inputs accepted")
		}
	}
}
func TestGenerateRelationsPreservesExistingEntry(t *testing.T) {
	sql, err := generate(nil)
	if err != nil || !strings.Contains(sql, "schema_migrations") {
		t.Fatal("relation generator unavailable")
	}
}

func TestGenerateProductionCompensationSeparately(t *testing.T) {
	query, err := generate([]string{"--compensation", "--service-uid", "123", "--environment", "production"})
	if err != nil || !strings.Contains(query, "CREATE SCHEMA order_compensation;") || !strings.Contains(query, "public.order_compensation_readiness_v1()") || strings.Contains(query, "__ORDER_") {
		t.Fatalf("production scheduler generation failed: %v", err)
	}
	for _, args := range [][]string{
		{"--compensation", "--service-uid", "123", "--environment", "uat"},
		{"--compensation", "--functions", "--service-uid", "123", "--environment", "production"},
		{"--compensation", "--environment", "production"},
		{"--compensation", "--service-uid", "123", "--environment", "production", "--origin", "https://other.invalid"},
	} {
		if _, err := generate(args); err == nil {
			t.Fatal("invalid scheduler inputs accepted")
		}
	}
}
