package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/migrate"
)

const cliCanary = "cli-canary-password-dsn-sql-server-error"

func TestExecuteSuccessAndCurrentExitZero(t *testing.T) {
	for _, result := range []migrate.Result{
		{FromVersion: 0, ToVersion: 1, AppliedCount: 1},
		{FromVersion: 1, ToVersion: 1, AppliedCount: 0},
	} {
		var stdout, stderr bytes.Buffer
		code := execute(nil, &stdout, &stderr, func(context.Context) (migrate.Result, error) {
			return result, nil
		}, func() time.Time { return time.Unix(0, 0) })
		if code != 0 || stderr.Len() != 0 {
			t.Fatalf("code/stderr = %d/%q, want 0/empty", code, stderr.String())
		}
		entry := decodeSingleJSONLog(t, stdout.Bytes())
		assertExactKeys(t, entry, "time", "level", "msg", "event", "from_version", "to_version", "applied_count", "duration_ms")
		if entry["event"] != "migration_completed" {
			t.Fatalf("event = %v", entry["event"])
		}
	}
}

func TestExecuteFailureIsSanitizedAndExitOne(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := execute(nil, &stdout, &stderr, func(context.Context) (migrate.Result, error) {
		return migrate.Result{FromVersion: 1, ToVersion: 1}, commandError{reason: "schema_dirty", cause: errors.New(cliCanary)}
	}, func() time.Time { return time.Unix(0, 0) })
	if code != 1 || stdout.Len() != 0 {
		t.Fatalf("code/stdout = %d/%q, want 1/empty", code, stdout.String())
	}
	entry := decodeSingleJSONLog(t, stderr.Bytes())
	assertExactKeys(t, entry, "time", "level", "msg", "event", "reason", "version", "duration_ms")
	if entry["reason"] != "schema_dirty" || strings.Contains(stderr.String(), cliCanary) {
		t.Fatalf("failure log was not sanitized: %s", stderr.String())
	}
}

func TestExecuteArgumentsExitTwoWithoutConnecting(t *testing.T) {
	for _, args := range [][]string{{"up"}, {"--help"}, {"status", "extra"}} {
		var stdout, stderr bytes.Buffer
		called := false
		code := execute(args, &stdout, &stderr, func(context.Context) (migrate.Result, error) {
			called = true
			return migrate.Result{}, nil
		}, time.Now)
		if code != 2 || called || stdout.Len() != 0 || strings.TrimSpace(stderr.String()) != "usage: order-migrate" {
			t.Fatalf("args=%v code=%d called=%v stdout=%q stderr=%q", args, code, called, stdout.String(), stderr.String())
		}
	}
}

func decodeSingleJSONLog(t *testing.T, data []byte) map[string]any {
	t.Helper()
	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	if len(lines) != 1 {
		t.Fatalf("log lines = %d, want one: %q", len(lines), data)
	}
	var entry map[string]any
	if err := json.Unmarshal(lines[0], &entry); err != nil {
		t.Fatalf("decode log: %v: %s", err, lines[0])
	}
	return entry
}

func assertExactKeys(t *testing.T, value map[string]any, keys ...string) {
	t.Helper()
	if len(value) != len(keys) {
		t.Fatalf("keys = %#v, want %v", value, keys)
	}
	for _, key := range keys {
		if _, ok := value[key]; !ok {
			t.Fatalf("missing key %q in %#v", key, value)
		}
	}
}
