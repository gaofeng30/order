package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	unsetEnv(t, "ORDER_API_HTTP_ADDR")
	unsetEnv(t, "ORDER_API_SHUTDOWN_TIMEOUT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Fatalf("ShutdownTimeout = %s, want 10s", cfg.ShutdownTimeout)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("ORDER_API_HTTP_ADDR", "127.0.0.1:9090")
	t.Setenv("ORDER_API_SHUTDOWN_TIMEOUT", "3s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != "127.0.0.1:9090" {
		t.Fatalf("HTTPAddr = %q, want override", cfg.HTTPAddr)
	}
	if cfg.ShutdownTimeout != 3*time.Second {
		t.Fatalf("ShutdownTimeout = %s, want 3s", cfg.ShutdownTimeout)
	}
}

func TestLoadRejectsInvalidShutdownTimeout(t *testing.T) {
	t.Setenv("ORDER_API_SHUTDOWN_TIMEOUT", "not-a-duration")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "ORDER_API_SHUTDOWN_TIMEOUT") {
		t.Fatalf("Load() error = %v, want named configuration error", err)
	}
}

func TestLoadRejectsNonPositiveShutdownTimeout(t *testing.T) {
	for _, value := range []string{"0s", "-1s"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("ORDER_API_SHUTDOWN_TIMEOUT", value)

			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), "greater than zero") {
				t.Fatalf("Load() error = %v, want positive-duration error", err)
			}
		})
	}
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	value, present := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}
	t.Cleanup(func() {
		if present {
			_ = os.Setenv(key, value)
			return
		}
		_ = os.Unsetenv(key)
	})
}
