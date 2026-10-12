package config

import (
	"context"
	"errors"
	"os"
)

var errCloudRunSecretUnavailable = errors.New("cloudrun secret unavailable")

// cloudRunSecretSource reads only UAT service-version business variables.
// Values are parsed and retained by the startup loaders, without a fallback.
type cloudRunSecretSource struct{}

func (cloudRunSecretSource) Get(ctx context.Context, name string) (string, error) {
	if ctx == nil || ctx.Err() != nil {
		return "", errCloudRunSecretUnavailable
	}
	var variable string
	switch name {
	case "order-uat-" + databasePasswordPurpose:
		variable = "ORDER_DB_PASSWORD"
	case "order-uat-" + miniProgramPurpose:
		variable = "ORDER_WECHAT_MINIPROGRAM_APP_SECRET"
	case "order-uat-" + weChatPaySecretPurpose:
		variable = "ORDER_WECHATPAY_API_V3"
	case "order-uat-" + redemptionTokenPurpose:
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
