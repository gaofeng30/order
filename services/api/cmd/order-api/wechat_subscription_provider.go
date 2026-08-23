package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/subscription"
	"github.com/gaofeng30/order/services/api/internal/wechat"
)

const (
	weChatSubscriptionVersionEnv              = "ORDER_WECHAT_SUBSCRIPTION_TEMPLATE_CONFIG_VERSION"
	weChatSubscriptionReadyTemplateIDEnv      = "ORDER_WECHAT_SUBSCRIPTION_READY_TEMPLATE_ID"
	weChatSubscriptionReadyOrderNumberKeyEnv  = "ORDER_WECHAT_SUBSCRIPTION_READY_ORDER_NUMBER_KEY"
	weChatSubscriptionReadyPickupDateKeyEnv   = "ORDER_WECHAT_SUBSCRIPTION_READY_PICKUP_DATE_KEY"
	weChatSubscriptionReadyPickupTimeKeyEnv   = "ORDER_WECHAT_SUBSCRIPTION_READY_PICKUP_TIME_KEY"
	weChatSubscriptionReadyPickupPointKeyEnv  = "ORDER_WECHAT_SUBSCRIPTION_READY_PICKUP_POINT_KEY"
	weChatSubscriptionRefundTemplateIDEnv     = "ORDER_WECHAT_SUBSCRIPTION_REFUND_RESULT_TEMPLATE_ID"
	weChatSubscriptionRefundOrderNumberKeyEnv = "ORDER_WECHAT_SUBSCRIPTION_REFUND_RESULT_ORDER_NUMBER_KEY"
	weChatSubscriptionRefundResultKeyEnv      = "ORDER_WECHAT_SUBSCRIPTION_REFUND_RESULT_RESULT_KEY"
	weChatSubscriptionMiniProgramStateEnv     = "ORDER_WECHAT_SUBSCRIPTION_MINIPROGRAM_STATE"
	weChatSubscriptionLanguageEnv             = "ORDER_WECHAT_SUBSCRIPTION_LANGUAGE"
	weChatSubscriptionOrderPage               = "pages/order-detail/index"
)

var (
	errWeChatSubscriptionConfig    = errors.New("wechat subscription configuration invalid")
	errWeChatSubscriptionProvider  = errors.New("wechat subscription provider unavailable")
	errWeChatSubscriptionRecipient = errors.New("wechat subscription recipient unavailable")
	errWeChatSubscriptionTemplate  = errors.New("wechat subscription template unavailable")
)

type readyTemplateKeys struct {
	orderNumber string
	pickupDate  string
	pickupTime  string
	pickupPoint string
}

type refundResultTemplateKeys struct {
	orderNumber string
	result      string
}

type weChatSubscriptionProviderConfig struct {
	templateConfigVersion  uint64
	readyTemplateID        string
	readyKeys              readyTemplateKeys
	refundResultTemplateID string
	refundResultKeys       refundResultTemplateKeys
	miniProgramState       string
	language               string
}

func loadWeChatSubscriptionProviderConfig(lookup func(string) (string, bool)) (weChatSubscriptionProviderConfig, error) {
	if lookup == nil {
		return weChatSubscriptionProviderConfig{}, errWeChatSubscriptionConfig
	}
	values := make(map[string]string, 11)
	for _, key := range []string{
		weChatSubscriptionVersionEnv,
		weChatSubscriptionReadyTemplateIDEnv,
		weChatSubscriptionReadyOrderNumberKeyEnv,
		weChatSubscriptionReadyPickupDateKeyEnv,
		weChatSubscriptionReadyPickupTimeKeyEnv,
		weChatSubscriptionReadyPickupPointKeyEnv,
		weChatSubscriptionRefundTemplateIDEnv,
		weChatSubscriptionRefundOrderNumberKeyEnv,
		weChatSubscriptionRefundResultKeyEnv,
		weChatSubscriptionMiniProgramStateEnv,
		weChatSubscriptionLanguageEnv,
	} {
		value, ok := lookup(key)
		if !ok || value == "" {
			return weChatSubscriptionProviderConfig{}, errWeChatSubscriptionConfig
		}
		values[key] = value
	}
	version, err := strconv.ParseUint(values[weChatSubscriptionVersionEnv], 10, 64)
	if err != nil || version == 0 || strconv.FormatUint(version, 10) != values[weChatSubscriptionVersionEnv] {
		return weChatSubscriptionProviderConfig{}, errWeChatSubscriptionConfig
	}
	config := weChatSubscriptionProviderConfig{
		templateConfigVersion: version,
		readyTemplateID:       values[weChatSubscriptionReadyTemplateIDEnv],
		readyKeys: readyTemplateKeys{
			orderNumber: values[weChatSubscriptionReadyOrderNumberKeyEnv],
			pickupDate:  values[weChatSubscriptionReadyPickupDateKeyEnv],
			pickupTime:  values[weChatSubscriptionReadyPickupTimeKeyEnv],
			pickupPoint: values[weChatSubscriptionReadyPickupPointKeyEnv],
		},
		refundResultTemplateID: values[weChatSubscriptionRefundTemplateIDEnv],
		refundResultKeys: refundResultTemplateKeys{
			orderNumber: values[weChatSubscriptionRefundOrderNumberKeyEnv],
			result:      values[weChatSubscriptionRefundResultKeyEnv],
		},
		miniProgramState: values[weChatSubscriptionMiniProgramStateEnv],
		language:         values[weChatSubscriptionLanguageEnv],
	}
	if !validWeChatSubscriptionProviderConfig(config) {
		return weChatSubscriptionProviderConfig{}, errWeChatSubscriptionConfig
	}
	return config, nil
}

func validWeChatSubscriptionProviderConfig(config weChatSubscriptionProviderConfig) bool {
	if config.templateConfigVersion == 0 ||
		!validWeChatTemplateID(config.readyTemplateID) ||
		!validWeChatTemplateID(config.refundResultTemplateID) ||
		config.readyTemplateID == config.refundResultTemplateID {
		return false
	}
	ready := []string{config.readyKeys.orderNumber, config.readyKeys.pickupDate, config.readyKeys.pickupTime, config.readyKeys.pickupPoint}
	refund := []string{config.refundResultKeys.orderNumber, config.refundResultKeys.result}
	if !validDistinctTemplateKeys(ready) || !validDistinctTemplateKeys(refund) {
		return false
	}
	validState := config.miniProgramState == "developer" || config.miniProgramState == "trial" || config.miniProgramState == "formal"
	validLanguage := config.language == "zh_CN" || config.language == "en_US" || config.language == "zh_HK" || config.language == "zh_TW"
	return validState && validLanguage
}

func validWeChatTemplateID(value string) bool {
	if len(value) < 1 || len(value) > 128 {
		return false
	}
	for index := 0; index < len(value); index++ {
		character := value[index]
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '_' && character != '-' {
			return false
		}
	}
	return true
}

func validDistinctTemplateKeys(keys []string) bool {
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if !validWeChatTemplateKeyword(key) {
			return false
		}
		if _, duplicate := seen[key]; duplicate {
			return false
		}
		seen[key] = struct{}{}
	}
	return true
}

func validWeChatTemplateKeyword(key string) bool {
	if len(key) < 1 || len(key) > 64 || ((key[0] < 'a' || key[0] > 'z') && (key[0] < 'A' || key[0] > 'Z')) {
		return false
	}
	for index := 1; index < len(key); index++ {
		character := key[index]
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '_' {
			return false
		}
	}
	return true
}

type phoneUserFinder interface {
	FindPhoneUser(context.Context, uint64) (identity.PhoneUser, error)
}

type identityRecipientResolver struct {
	repository phoneUserFinder
}

type staticWeChatSubscriptionTemplateResolver struct {
	config weChatSubscriptionProviderConfig
}

func newProductionWeChatSubscriptionProvider(
	client *http.Client,
	tokenSource *wechat.PhoneNumberClient,
	repository *identity.Repository,
	config weChatSubscriptionProviderConfig,
) (*subscription.WeChatProvider, error) {
	if client == nil || client.Timeout <= 0 || client.CheckRedirect == nil || tokenSource == nil || repository == nil || !validWeChatSubscriptionProviderConfig(config) {
		return nil, errWeChatSubscriptionProvider
	}
	return subscription.NewWeChatProvider(
		client,
		tokenSource,
		identityRecipientResolver{repository: repository},
		staticWeChatSubscriptionTemplateResolver{config: config},
		subscription.WeChatProviderConfig{MiniProgramState: config.miniProgramState, Language: config.language},
	), nil
}

func (resolver identityRecipientResolver) OpenID(ctx context.Context, userID uint64) (string, error) {
	if ctx == nil || userID == 0 || resolver.repository == nil {
		return "", errWeChatSubscriptionRecipient
	}
	user, err := resolver.repository.FindPhoneUser(ctx, userID)
	if err != nil || !validSubscriptionOpaqueValue(user.OpenID, 1, 128) {
		return "", errWeChatSubscriptionRecipient
	}
	return user.OpenID, nil
}

func (resolver staticWeChatSubscriptionTemplateResolver) ResolveTemplate(
	ctx context.Context,
	kind subscription.Kind,
	version uint64,
	message subscription.Message,
) (subscription.ResolvedTemplate, error) {
	if ctx == nil || version != resolver.config.templateConfigVersion || !validWeChatSubscriptionProviderConfig(resolver.config) {
		return subscription.ResolvedTemplate{}, errWeChatSubscriptionTemplate
	}
	switch kind {
	case subscription.KindReady:
		return subscription.ResolvedTemplate{
			TemplateID: resolver.config.readyTemplateID,
			Page:       weChatSubscriptionOrderPage,
			Data: map[string]string{
				resolver.config.readyKeys.orderNumber: message.OrderNumber,
				resolver.config.readyKeys.pickupDate:  message.PickupDate,
				resolver.config.readyKeys.pickupTime:  message.PickupTime,
				resolver.config.readyKeys.pickupPoint: message.PickupPoint,
			},
		}, nil
	case subscription.KindRefundResult:
		return subscription.ResolvedTemplate{
			TemplateID: resolver.config.refundResultTemplateID,
			Page:       weChatSubscriptionOrderPage,
			Data: map[string]string{
				resolver.config.refundResultKeys.orderNumber: message.OrderNumber,
				resolver.config.refundResultKeys.result:      message.RefundResult,
			},
		}, nil
	default:
		return subscription.ResolvedTemplate{}, errWeChatSubscriptionTemplate
	}
}

func validSubscriptionOpaqueValue(value string, min, max int) bool {
	if !utf8.ValidString(value) || len(value) < min || len(value) > max || value != strings.TrimSpace(value) {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}
