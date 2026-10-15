package config

import (
	"context"
	"errors"
	"os"
	"strings"
)

var errCloudRunSecretUnavailable = errors.New("cloudrun secret unavailable")

// cloudRunSecretSource reads only the selected environment service-version variables.
// Values are parsed and retained by the startup loaders, without a fallback.
type cloudRunSecretSource struct{}

func (cloudRunSecretSource) Get(ctx context.Context, name string) (string, error) {
	if ctx == nil || ctx.Err() != nil {
		return "", errCloudRunSecretUnavailable
	}
	environment := Environment(os.Getenv("ORDER_ENV"))
	if !environment.UsesCloudProviders() {
		return "", errCloudRunSecretUnavailable
	}
	prefix := "order-" + string(environment) + "-"
	if !strings.HasPrefix(name, prefix) {
		return "", errCloudRunSecretUnavailable
	}
	var variable string
	switch strings.TrimPrefix(name, prefix) {
	case databasePasswordPurpose:
		variable = "ORDER_DB_PASSWORD"
	case miniProgramPurpose:
		variable = "ORDER_WECHAT_MINIPROGRAM_APP_SECRET"
	case weChatPaySecretPurpose:
		variable = "ORDER_WECHATPAY_API_V3"
	case redemptionTokenPurpose:
		variable = "ORDER_REDEMPTION_TOKEN_KEY"
	default:
		return "", errCloudRunSecretUnavailable
	}
	value, present := os.LookupEnv(variable)
	if !present || value == "" {
		return "", errCloudRunSecretUnavailable
	}
	return value, nil
}
