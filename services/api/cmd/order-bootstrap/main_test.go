package main

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestBootstrapFlavorOptionsMatchCanonicalPRDOrder(t *testing.T) {
	want := []string{"少饭", "加饭", "少盐", "加辣", "酱汁分装", "免葱蒜", "打包分装", "多双餐具"}
	if !slices.Equal(bootstrapFlavorOptions[:], want) {
		t.Fatalf("bootstrap flavor options = %#v, want %#v", bootstrapFlavorOptions, want)
	}
}

func TestLoadBootstrapInputAcceptsOnlyExplicitCanonicalValues(t *testing.T) {
	values := map[string]string{
		"ORDER_BOOTSTRAP_OWNER_PHONE":   "+8613800000000",
		"ORDER_BOOTSTRAP_OWNER_NAME":    "甲方管理员",
		"ORDER_BOOTSTRAP_STORE_NAME":    "甲方门店",
		"ORDER_BOOTSTRAP_STORE_ADDRESS": "甲方地址",
		"ORDER_BOOTSTRAP_PICKUP_POINT":  "甲方取餐点",
	}
	input, err := loadBootstrapInput(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("loadBootstrapInput() error = %v", err)
	}
	if input.OwnerPhone != "+8613800000000" || input.OwnerName != "甲方管理员" || input.StoreName != "甲方门店" || input.StoreAddress != "甲方地址" || input.PickupPoint != "甲方取餐点" {
		t.Fatalf("input = %#v", input)
	}

	invalid := []struct {
		name, key, value string
	}{
		{name: "missing", key: "ORDER_BOOTSTRAP_OWNER_NAME", value: ""},
		{name: "phone without country prefix", key: "ORDER_BOOTSTRAP_OWNER_PHONE", value: "13800000000"},
		{name: "phone leading zero", key: "ORDER_BOOTSTRAP_OWNER_PHONE", value: "+08613800000000"},
		{name: "phone too long", key: "ORDER_BOOTSTRAP_OWNER_PHONE", value: "+1234567890123456"},
		{name: "untrimmed owner name", key: "ORDER_BOOTSTRAP_OWNER_NAME", value: " 管理员"},
		{name: "untrimmed store name", key: "ORDER_BOOTSTRAP_STORE_NAME", value: "门店 "},
		{name: "invalid utf8", key: "ORDER_BOOTSTRAP_STORE_ADDRESS", value: string([]byte{0xff})},
		{name: "blank pickup", key: "ORDER_BOOTSTRAP_PICKUP_POINT", value: "\t"},
		{name: "text exceeds mysql text capacity", key: "ORDER_BOOTSTRAP_OWNER_NAME", value: strings.Repeat("x", 65536)},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			candidate := make(map[string]string, len(values))
			for key, value := range values {
				candidate[key] = value
			}
			if test.name == "missing" {
				delete(candidate, test.key)
			} else {
				candidate[test.key] = test.value
			}
			if _, err := loadBootstrapInput(func(key string) (string, bool) {
				value, ok := candidate[key]
				return value, ok
			}); err == nil {
				t.Fatal("loadBootstrapInput() error = nil")
			}
		})
	}
}

func TestExecuteHasSanitizedStableProcessBoundary(t *testing.T) {
	const secretCanary = "bootstrap-owner-phone-name-address-dsn-canary"
	for _, outcome := range []bootstrapOutcome{outcomeCreated, outcomeUnchanged} {
		var stdout, stderr bytes.Buffer
		code := execute(nil, &stdout, &stderr, func(context.Context) (bootstrapOutcome, error) {
			return outcome, nil
		})
		if code != 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), `"event":"bootstrap_complete"`) || !strings.Contains(stderr.String(), `"outcome":"`+string(outcome)+`"`) {
			t.Fatalf("success code/stdout/stderr = %d/%q/%q", code, stdout.String(), stderr.String())
		}
	}

	var stdout, stderr bytes.Buffer
	code := execute(nil, &stdout, &stderr, func(context.Context) (bootstrapOutcome, error) {
		return "", commandError{reason: reasonBootstrapConflict, cause: errors.New(secretCanary)}
	})
	if code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), `"reason":"bootstrap_conflict"`) || strings.Contains(stderr.String(), secretCanary) {
		t.Fatalf("failure code/stdout/stderr = %d/%q/%q", code, stdout.String(), stderr.String())
	}

	called := false
	stdout.Reset()
	stderr.Reset()
	code = execute([]string{"--force"}, &stdout, &stderr, func(context.Context) (bootstrapOutcome, error) {
		called = true
		return outcomeCreated, nil
	})
	if code != 2 || called || stdout.Len() != 0 || strings.TrimSpace(stderr.String()) != "usage: order-bootstrap" {
		t.Fatalf("argument rejection = code %d called %v stdout %q stderr %q", code, called, stdout.String(), stderr.String())
	}
}
