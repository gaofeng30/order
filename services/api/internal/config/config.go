package config

import (
	"fmt"
	"os"
	"time"
)

const (
	defaultHTTPAddr        = ":8080"
	defaultShutdownTimeout = 10 * time.Second
)

// Config contains the complete runtime configuration for order-api.
type Config struct {
	HTTPAddr        string
	ShutdownTimeout time.Duration
}

// Load reads and validates the two supported order-api environment variables.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        defaultHTTPAddr,
		ShutdownTimeout: defaultShutdownTimeout,
	}

	if value, present := os.LookupEnv("ORDER_API_HTTP_ADDR"); present {
		if value == "" {
			return Config{}, fmt.Errorf("ORDER_API_HTTP_ADDR must not be empty")
		}
		cfg.HTTPAddr = value
	}

	if value, present := os.LookupEnv("ORDER_API_SHUTDOWN_TIMEOUT"); present {
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("ORDER_API_SHUTDOWN_TIMEOUT: %w", err)
		}
		if timeout <= 0 {
			return Config{}, fmt.Errorf("ORDER_API_SHUTDOWN_TIMEOUT must be greater than zero")
		}
		cfg.ShutdownTimeout = timeout
	}

	return cfg, nil
}
