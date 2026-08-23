package subscription

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	weChatSubscribeMessageEndpoint = "https://api.weixin.qq.com/cgi-bin/message/subscribe/send"
	maxWeChatResponseBodyBytes     = 64 << 10
)

type TokenSource interface {
	AccessToken(context.Context) (string, error)
}

type RecipientResolver interface {
	OpenID(context.Context, uint64) (string, error)
}

type TemplateResolver interface {
	ResolveTemplate(context.Context, Kind, uint64, Message) (ResolvedTemplate, error)
}

type ResolvedTemplate struct {
	TemplateID string
	Page       string
	Data       map[string]string
}

type WeChatProviderConfig struct {
	MiniProgramState string
	Language         string
}

type WeChatProvider struct {
	client            *http.Client
	tokenSource       TokenSource
	recipientResolver RecipientResolver
	templateResolver  TemplateResolver
	config            WeChatProviderConfig
}

func NewWeChatProvider(
	client *http.Client,
	tokenSource TokenSource,
	recipientResolver RecipientResolver,
	templateResolver TemplateResolver,
	config WeChatProviderConfig,
) *WeChatProvider {
	return &WeChatProvider{
		client:            client,
		tokenSource:       tokenSource,
		recipientResolver: recipientResolver,
		templateResolver:  templateResolver,
		config:            config,
	}
}

func (provider *WeChatProvider) SendSubscription(ctx context.Context, delivery Delivery) (SendResult, error) {
	if provider == nil || provider.client == nil || provider.tokenSource == nil || provider.recipientResolver == nil || provider.templateResolver == nil || !validProviderConfig(provider.config) || ctx == nil {
		return SendResult{}, weChatSendError("ADAPTER_UNAVAILABLE", false)
	}
	if !validDelivery(delivery) {
		return SendResult{}, weChatSendError("DELIVERY_INVALID", true)
	}
	accessToken, err := provider.tokenSource.AccessToken(ctx)
	if err != nil || !validOpaqueValue(accessToken, 1, 4096) {
		return SendResult{}, weChatSendError("TOKEN_UNAVAILABLE", false)
	}
	openID, err := provider.recipientResolver.OpenID(ctx, delivery.RecipientUserID)
	if err != nil {
		return SendResult{}, weChatSendError("RECIPIENT_UNAVAILABLE", false)
	}
	if !validOpaqueValue(openID, 1, 128) {
		return SendResult{}, weChatSendError("RECIPIENT_INVALID", true)
	}
	resolved, err := provider.templateResolver.ResolveTemplate(ctx, delivery.Kind, delivery.TemplateConfigVersion, delivery.Message)
	if err != nil {
		return SendResult{}, weChatSendError("TEMPLATE_UNAVAILABLE", false)
	}
	requestBody, err := buildWeChatRequest(openID, resolved, provider.config)
	if err != nil {
		return SendResult{}, err
	}
	encoded, err := json.Marshal(requestBody)
	if err != nil {
		return SendResult{}, weChatSendError("REQUEST_INVALID", true)
	}

	endpoint, err := url.Parse(weChatSubscribeMessageEndpoint)
	if err != nil {
		return SendResult{}, weChatSendError("ADAPTER_INVALID", true)
	}
	query := endpoint.Query()
	query.Set("access_token", accessToken)
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(encoded))
	if err != nil {
		return SendResult{}, weChatSendError("REQUEST_INVALID", true)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := provider.client.Do(request)
	if err != nil {
		return SendResult{}, weChatSendError("WECHAT_UNAVAILABLE", false)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return SendResult{}, weChatSendError("WECHAT_HTTP_STATUS", false)
	}
	if !validJSONContentType(response.Header) {
		return SendResult{}, weChatSendError("WECHAT_RESPONSE_INVALID", false)
	}
	result, err := decodeWeChatResponse(response.Body)
	if err != nil {
		return SendResult{}, err
	}
	if *result.ErrCode != 0 {
		return SendResult{}, classifyWeChatError(*result.ErrCode)
	}

	digest := sha256.Sum256(encoded)
	return SendResult{ProviderMessageID: "request_" + hex.EncodeToString(digest[:])}, nil
}

type weChatRequest struct {
	ToUser           string                     `json:"touser"`
	TemplateID       string                     `json:"template_id"`
	Page             string                     `json:"page,omitempty"`
	MiniProgramState string                     `json:"miniprogram_state"`
	Language         string                     `json:"lang"`
	Data             map[string]weChatDataValue `json:"data"`
}

type weChatDataValue struct {
	Value string `json:"value"`
}

func buildWeChatRequest(openID string, resolved ResolvedTemplate, config WeChatProviderConfig) (weChatRequest, error) {
	if !validOpaqueValue(resolved.TemplateID, 1, 128) || !validOptionalPage(resolved.Page) || len(resolved.Data) == 0 || len(resolved.Data) > 20 {
		return weChatRequest{}, weChatSendError("TEMPLATE_INVALID", true)
	}
	data := make(map[string]weChatDataValue, len(resolved.Data))
	for keyword, value := range resolved.Data {
		if !validTemplateKeyword(keyword) {
			return weChatRequest{}, weChatSendError("TEMPLATE_INVALID", true)
		}
		if !validTemplateValue(value) {
			return weChatRequest{}, weChatSendError("TEMPLATE_DATA_INVALID", true)
		}
		data[keyword] = weChatDataValue{Value: value}
	}
	return weChatRequest{
		ToUser:           openID,
		TemplateID:       resolved.TemplateID,
		Page:             resolved.Page,
		MiniProgramState: config.MiniProgramState,
		Language:         config.Language,
		Data:             data,
	}, nil
}

type weChatResponse struct {
	ErrCode *int64  `json:"errcode"`
	ErrMsg  *string `json:"errmsg"`
}

func decodeWeChatResponse(body io.Reader) (weChatResponse, error) {
	encoded, err := io.ReadAll(io.LimitReader(body, maxWeChatResponseBodyBytes+1))
	if err != nil || len(encoded) == 0 || len(encoded) > maxWeChatResponseBodyBytes || !utf8.Valid(encoded) {
		return weChatResponse{}, weChatSendError("WECHAT_RESPONSE_INVALID", false)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var response weChatResponse
	if err := decoder.Decode(&response); err != nil {
		return weChatResponse{}, weChatSendError("WECHAT_RESPONSE_INVALID", false)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return weChatResponse{}, weChatSendError("WECHAT_RESPONSE_INVALID", false)
	}
	if response.ErrCode == nil || response.ErrMsg == nil || *response.ErrCode < 0 || *response.ErrCode > 999999 || !utf8.ValidString(*response.ErrMsg) || len(*response.ErrMsg) > 1024 {
		return weChatResponse{}, weChatSendError("WECHAT_RESPONSE_INVALID", false)
	}
	return response, nil
}

func classifyWeChatError(code int64) error {
	switch code {
	case 40001, 40014:
		return weChatSendError("WECHAT_ACCESS_TOKEN_INVALID", false)
	case 40003:
		return weChatSendError("WECHAT_INVALID_RECIPIENT", true)
	case 40037:
		return weChatSendError("WECHAT_INVALID_TEMPLATE", true)
	case 43101:
		return weChatSendError("WECHAT_USER_NOT_SUBSCRIBED", true)
	case 43107:
		return weChatSendError("WECHAT_SUBSCRIPTION_BLOCKED", true)
	case 43108:
		return weChatSendError("WECHAT_CONCURRENT_SEND", false)
	case 45168:
		return weChatSendError("WECHAT_CONTENT_REJECTED", true)
	case 47003:
		return weChatSendError("WECHAT_INVALID_TEMPLATE_DATA", true)
	default:
		return weChatSendError("WECHAT_ERROR", false)
	}
}

func validProviderConfig(config WeChatProviderConfig) bool {
	validState := config.MiniProgramState == "developer" || config.MiniProgramState == "trial" || config.MiniProgramState == "formal"
	validLanguage := config.Language == "zh_CN" || config.Language == "en_US" || config.Language == "zh_HK" || config.Language == "zh_TW"
	return validState && validLanguage
}

func validDelivery(delivery Delivery) bool {
	return delivery.OutboxID > 0 && delivery.OrderID > 0 && delivery.RecipientUserID > 0 && validKind(delivery.Kind) && validMessage(delivery.Kind, delivery.Message) && delivery.TemplateConfigVersion > 0 && delivery.AttemptCount > 0
}

func validOpaqueValue(value string, min, max int) bool {
	return utf8.ValidString(value) && len(value) >= min && len(value) <= max && value == strings.TrimSpace(value) && !containsControl(value)
}

func validOptionalPage(page string) bool {
	return page == "" || validOpaqueValue(page, 1, 1024)
}

func validTemplateKeyword(keyword string) bool {
	if len(keyword) < 1 || len(keyword) > 64 || ((keyword[0] < 'a' || keyword[0] > 'z') && (keyword[0] < 'A' || keyword[0] > 'Z')) {
		return false
	}
	for _, character := range keyword {
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '_' {
			return false
		}
	}
	return true
}

func validTemplateValue(value string) bool {
	return validOpaqueValue(value, 1, 512)
}

func containsControl(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) {
			return true
		}
	}
	return false
}

func validJSONContentType(header http.Header) bool {
	values := header.Values("Content-Type")
	if len(values) != 1 {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(values[0])
	return err == nil && mediaType == "application/json"
}

func weChatSendError(code string, permanent bool) error {
	return &SendError{Code: code, Permanent: permanent}
}
