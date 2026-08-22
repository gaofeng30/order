package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/wechat"
)

const (
	defaultHTTPAddr        = ":8080"
	defaultShutdownTimeout = 10 * time.Second
	secretLoadTimeout      = 5 * time.Second
	databasePasswordSecret = "order-production-db-password"
	miniProgramSecret      = "order-production-wechat-miniprogram-app-secret"
	maxDatabasePassword    = 4096
	maxMiniProgramAppID    = 128
	maxMiniProgramSecret   = 256
)

type Environment string

const (
	Development Environment = "development"
	Test        Environment = "test"
	Production  Environment = "production"
)

// Config contains the complete runtime configuration for order-api.
type Config struct {
	HTTPAddr        string
	ShutdownTimeout time.Duration
	Environment     Environment
	Database        database.ConnectionConfig
	MiniProgram     wechat.Credentials
}

// SecretSource retrieves production secrets without exposing provider details
// to the runtime configuration contract.
type SecretSource interface {
	Get(context.Context, string) (string, error)
}

type loadError struct {
	reason string
	field  string
}

func (err loadError) Error() string {
	if err.field == "" {
		return err.reason
	}
	return fmt.Sprintf("%s: %s", err.field, err.reason)
}

// Reason returns the stable, non-sensitive configuration failure category.
func Reason(err error) string {
	if value, ok := err.(loadError); ok {
		return value.reason
	}
	return "configuration_invalid"
}

// Load reads and validates order-api's structured runtime configuration.
func Load() (Config, error) {
	if Environment(os.Getenv("ORDER_ENV")) == Production {
		return load(newProductionSecretSource(os.Getenv("ORDER_TENCENT_REGION")))
	}
	return load(nil)
}

// LoadWithSecretSource loads configuration through an explicit production
// secret boundary. It is intended for provider adapters and their tests.
func LoadWithSecretSource(source SecretSource) (Config, error) {
	return load(source)
}

func load(source SecretSource) (Config, error) {
	cfg := Config{
		HTTPAddr:        defaultHTTPAddr,
		ShutdownTimeout: defaultShutdownTimeout,
		Environment:     Development,
	}

	if value, present := os.LookupEnv("ORDER_ENV"); present {
		cfg.Environment = Environment(value)
	}
	if cfg.Environment != Development && cfg.Environment != Test && cfg.Environment != Production {
		return Config{}, loadError{reason: "invalid_environment", field: "ORDER_ENV"}
	}

	if cfg.Environment == Production {
		for _, field := range []string{
			"ORDER_DB_PASSWORD",
			"ORDER_DB_DSN",
			"ORDER_WECHAT_MINIPROGRAM_APP_SECRET",
			"TENCENTCLOUD_SECRET_ID",
			"TENCENTCLOUD_SECRET_KEY",
			"TENCENTCLOUD_TOKEN",
			"TENCENTCLOUD_CREDENTIALS_FILE",
		} {
			if _, present := os.LookupEnv(field); present {
				return Config{}, loadError{reason: "production_secret_environment_forbidden", field: field}
			}
		}
		if !validTencentRegion(os.Getenv("ORDER_TENCENT_REGION")) {
			return Config{}, loadError{reason: "invalid_tencent_region", field: "ORDER_TENCENT_REGION"}
		}
	}
	if _, present := os.LookupEnv("ORDER_DB_DSN"); present {
		return Config{}, loadError{reason: "raw_dsn_unsupported", field: "ORDER_DB_DSN"}
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

	port, err := parsePort(os.Getenv("ORDER_DB_PORT"))
	if err != nil {
		return Config{}, err
	}
	databasePassword := os.Getenv("ORDER_DB_PASSWORD")
	miniProgramAppSecret := os.Getenv("ORDER_WECHAT_MINIPROGRAM_APP_SECRET")
	if cfg.Environment == Production {
		if source == nil {
			return Config{}, loadError{reason: "production_secret_source_unavailable"}
		}
		ctx, cancel := context.WithTimeout(context.Background(), secretLoadTimeout)
		defer cancel()
		databasePassword, err = source.Get(ctx, databasePasswordSecret)
		if err != nil {
			return Config{}, loadError{reason: "production_secret_source_unavailable"}
		}
		if !validProductionSecret(databasePassword, maxDatabasePassword) {
			return Config{}, loadError{reason: "production_secret_value_invalid", field: "ORDER_DB_PASSWORD"}
		}
		miniProgramAppSecret, err = source.Get(ctx, miniProgramSecret)
		if err != nil {
			return Config{}, loadError{reason: "production_secret_source_unavailable"}
		}
		if !validProductionSecret(miniProgramAppSecret, maxMiniProgramSecret) || validateMiniProgramField("ORDER_WECHAT_MINIPROGRAM_APP_SECRET", miniProgramAppSecret, maxMiniProgramSecret) != nil {
			return Config{}, loadError{reason: "production_secret_value_invalid", field: "ORDER_WECHAT_MINIPROGRAM_APP_SECRET"}
		}
	}
	cfg.Database = database.ConnectionConfig{
		Host:     os.Getenv("ORDER_DB_HOST"),
		Port:     port,
		Database: os.Getenv("ORDER_DB_NAME"),
		User:     os.Getenv("ORDER_DB_USER"),
		Password: databasePassword,
		TLSMode:  os.Getenv("ORDER_DB_TLS_MODE"),
	}
	if err := validateDatabase(cfg.Database); err != nil {
		return Config{}, err
	}
	cfg.MiniProgram = wechat.Credentials{
		AppID:     os.Getenv("ORDER_WECHAT_MINIPROGRAM_APP_ID"),
		AppSecret: miniProgramAppSecret,
	}
	if err := validateMiniProgramField("ORDER_WECHAT_MINIPROGRAM_APP_ID", cfg.MiniProgram.AppID, maxMiniProgramAppID); err != nil {
		return Config{}, err
	}
	if err := validateMiniProgramField("ORDER_WECHAT_MINIPROGRAM_APP_SECRET", cfg.MiniProgram.AppSecret, maxMiniProgramSecret); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func validProductionSecret(value string, maximum int) bool {
	if value == "" || len(value) > maximum {
		return false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < ' ' || value[index] == 0x7f {
			return false
		}
	}
	return true
}

func validTencentRegion(value string) bool {
	if len(value) < len("ap-a") || len(value) > 64 || !strings.HasPrefix(value, "ap-") || value[len(value)-1] == '-' {
		return false
	}
	for index := 0; index < len(value); index++ {
		if (value[index] < 'a' || value[index] > 'z') && (value[index] < '0' || value[index] > '9') && value[index] != '-' {
			return false
		}
	}
	return true
}

func validateMiniProgramField(field, value string, maximum int) error {
	if value == "" || len(value) > maximum {
		return loadError{reason: "invalid_miniprogram_field", field: field}
	}
	for index := 0; index < len(value); index++ {
		if value[index] < '!' || value[index] > '~' {
			return loadError{reason: "invalid_miniprogram_field", field: field}
		}
	}
	return nil
}

func parsePort(value string) (uint16, error) {
	port, err := strconv.ParseUint(value, 10, 16)
	if err != nil || port == 0 {
		return 0, loadError{reason: "invalid_database_field", field: "ORDER_DB_PORT"}
	}
	return uint16(port), nil
}

func validateDatabase(cfg database.ConnectionConfig) error {
	if cfg.Host == "" || strings.ContainsAny(cfg.Host, "@/") || strings.Contains(cfg.Host, "://") {
		return loadError{reason: "invalid_database_field", field: "ORDER_DB_HOST"}
	}
	if cfg.Database == "" {
		return loadError{reason: "invalid_database_field", field: "ORDER_DB_NAME"}
	}
	if cfg.User == "" {
		return loadError{reason: "invalid_database_field", field: "ORDER_DB_USER"}
	}
	if cfg.Password == "" {
		return loadError{reason: "invalid_database_field", field: "ORDER_DB_PASSWORD"}
	}
	if cfg.TLSMode != "required" && cfg.TLSMode != "disabled" {
		return loadError{reason: "invalid_database_field", field: "ORDER_DB_TLS_MODE"}
	}
	return nil
}
