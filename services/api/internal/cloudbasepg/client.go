// Package cloudbasepg calls the reviewed Order UAT RPCs as an ordinary service user.
package cloudbasepg

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"
)

const maxResponseBytes = 8 << 20
const uatEnvironment = "order-uat-d1gtzpzj50615c300"
const refreshAhead = 30 * time.Second
const failedAuthenticationBackoff = time.Second
const requestTimeout = 15 * time.Second

var ErrUnavailable = errors.New("shared PG unavailable")

type Error struct {
	SQLState, RequestID string
	Status              int
}

func (e *Error) Error() string {
	return fmt.Sprintf("shared PG request failed (status=%d sqlstate=%s)", e.Status, e.SQLState)
}

type Credentials struct {
	Username, Password, PublishableKey, ServiceUID string
}

type Client struct {
	origin      string
	credentials Credentials
	http        *http.Client
	now         func() time.Time
	mu          sync.Mutex
	cached      accessToken
	flight      *authentication
	authError   error
	retryAfter  time.Time
}

type accessToken struct {
	value       string
	usableUntil time.Time
}

type authentication struct {
	done  chan struct{}
	token accessToken
	err   error
}

func New(envID string, credentials Credentials) (*Client, error) {
	if envID != uatEnvironment {
		return nil, ErrUnavailable
	}
	for _, value := range []string{credentials.Username, credentials.Password, credentials.PublishableKey, credentials.ServiceUID} {
		if strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\r\n") {
			return nil, ErrUnavailable
		}
	}
	keyClaims, err := tokenClaims(credentials.PublishableKey)
	if err != nil || len(credentials.PublishableKey) > 4096 || keyClaims.Role != "anon" || keyClaims.Audience != envID || (keyClaims.ProjectID != "" && keyClaims.ProjectID != envID) {
		return nil, ErrUnavailable
	}
	return newClient("https://"+envID+".api.tcloudbasegateway.com", credentials, &http.Client{Timeout: requestTimeout}), nil
}

func newClient(origin string, credentials Credentials, transport *http.Client) *Client {
	h := *transport
	h.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{origin: origin, credentials: credentials, http: &h, now: time.Now}
}

func (c *Client) CallRPC(ctx context.Context, name string, parameters map[string]any, out any) error {
	if c == nil || ctx == nil || out == nil || reflect.ValueOf(out).Kind() != reflect.Pointer || reflect.ValueOf(out).IsNil() || !allowedRPC(name) {
		return ErrUnavailable
	}
	if parameters == nil {
		parameters = map[string]any{}
	}
	body, err := json.Marshal(parameters)
	if err != nil {
		return ErrUnavailable
	}
	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}
	raw, err := c.post(ctx, "/v1/rdb/rest/rpc/"+name, token, body, false)
	if err != nil {
		return err
	}
	return decodeJSON(raw, out)
}

func (c *Client) accessToken(ctx context.Context) (string, error) {
	if ctx.Err() != nil {
		return "", ErrUnavailable
	}
	c.mu.Lock()
	if c.cached.value != "" && c.now().Before(c.cached.usableUntil) {
		token := c.cached.value
		c.mu.Unlock()
		return token, nil
	}
	if c.authError != nil && c.now().Before(c.retryAfter) {
		err := c.authError
		c.mu.Unlock()
		return "", err
	}
	flight := c.flight
	if flight == nil {
		flight = &authentication{done: make(chan struct{})}
		c.flight = flight
		go c.authenticate(flight)
	}
	c.mu.Unlock()
	select {
	case <-ctx.Done():
		return "", ErrUnavailable
	case <-flight.done:
		if ctx.Err() != nil {
			return "", ErrUnavailable
		}
		return flight.token.value, flight.err
	}
}

// Authentication belongs to the shared flight, not its first caller. Its own
// deadline bounds cleanup even when every caller cancels their wait.
func (c *Client) authenticate(flight *authentication) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	token, err := c.signIn(ctx)
	c.mu.Lock()
	if err == nil {
		c.cached = token
		c.authError = nil
	} else {
		c.authError, c.retryAfter = err, c.now().Add(failedAuthenticationBackoff)
	}
	flight.token, flight.err = token, err
	close(flight.done)
	c.flight = nil
	c.mu.Unlock()
}

func allowedRPC(name string) bool {
	switch name {
	case "order_redeem_order_proof_v1", "order_redeem_order_v1", "order_shared_pg_readiness_v1",
		"order_fulfillment_proof_v1", "order_mark_ready_v1", "order_redeem_token_v1", "order_redeem_current_date_code_v1",
		"order_identity_projection_v1", "order_query_list_user_v1", "order_query_get_user_v1", "order_query_search_merchant_v1", "order_query_get_merchant_v1",
		"order_identity_session_create_v1", "order_identity_session_find_v1", "order_identity_session_revoke_v1",
		"order_identity_phone_read_v1", "order_identity_phone_bind_v1", "order_merchant_login_start_v1",
		"order_merchant_login_complete_v1", "order_merchant_login_recover_v1", "order_identity_extra_phone_set_v1",
		"order_merchant_account_current_v1", "order_merchant_account_list_v1", "order_merchant_account_command_v1",
		"order_pc_login_begin_v1", "order_pc_login_approval_read_v1", "order_pc_login_approve_v1",
		"order_pc_login_poll_v1", "order_pc_session_authenticate_v1", "order_pc_session_revoke_v1",
		"order_catalog_browse_v1", "order_catalog_detail_v1", "order_catalog_categories_v1",
		"order_catalog_products_v1", "order_catalog_product_v1", "order_catalog_command_v1",
		"order_menu_snapshot_v1", "order_menu_pickup_facts_v1",
		"order_storefront_get_v1", "order_storefront_admin_v1", "order_storefront_configure_v1",
		"order_storefront_launch_read_v1", "order_storefront_launch_configure_v1",
		"order_image_confirmed_v1", "order_image_confirmation_replay_v1", "order_image_confirmation_commit_v1", "order_image_versions_v1", "order_display_pricing_facts_v1",
		"order_merchant_soldout_replay_v1", "order_merchant_soldout_command_v1",
		"order_quote_create_v1", "order_quote_read_v1",
		"order_staff_admin_list_v1", "order_staff_admin_rate_v1", "order_staff_admin_command_v1",
		"order_import_preview_facts_v1", "order_import_preview_v1", "order_import_commit_v1", "order_import_result_v1",
		"order_payment_receipt_read_v1",
		"order_payment_prepare_replay_v1",
		"order_payment_openid_v1",
		"order_payment_prepare_v2",
		"order_payment_finish_create_v1",
		"order_payment_read_v1",
		"order_payment_existing_order_v1",
		"order_payment_due_v1",
		"order_payment_claim_query_v1",
		"order_payment_finish_query_v1",
		"order_payment_observe_v1",
		"order_payment_materialize_v1",
		"order_payment_pending_v1",
		"order_payment_receipt_append_v1",
		"order_payment_owner_v1",
		"order_refund_order_locator_v1",
		"order_refund_request_order_v2",
		"order_refund_request_paid_v2",
		"order_refund_claim_create_v1",
		"order_refund_finish_create_v1",
		"order_refund_read_v1",
		"order_refund_observation_locator_v1",
		"order_refund_observe_v1",
		"order_refund_due_v1",
		"order_refund_claim_query_v1",
		"order_refund_release_query_v1",
		"order_refund_pending_v1",
		"order_subscription_consent_read_v1",
		"order_subscription_consent_record_v1",
		"order_subscription_claim_v1",
		"order_subscription_sent_v1",
		"order_subscription_temporary_failure_v1",
		"order_subscription_permanent_failure_v1",
		"order_production_due_v1",
		"order_production_advance_v1",
		"order_billing_summary_v1",
		"order_billing_payments_v1",
		"order_billing_refunds_v1",
		"order_billing_payment_facts_v1",
		"order_billing_refund_facts_v1",
		"order_billing_reconcile_v2",
		"order_billing_ownership_facts_v1",
		"order_admin_report_counts_v1",
		"order_admin_report_export_v1",
		"order_admin_report_get_v1",
		"order_admin_report_search_v1",
		"order_admin_report_stats_v1",
		"order_admin_report_payments_v1",
		"order_admin_report_refunds_v1",
		"order_admin_report_summary_v1",
		"order_admin_report_pending_v1",
		"order_audit_search_v1",
		"order_store_status_apply_v1":
		return true
	}
	return false
}

func (c *Client) signIn(ctx context.Context) (accessToken, error) {
	started := c.now()
	body, err := json.Marshal(map[string]string{"username": c.credentials.Username, "password": c.credentials.Password})
	if err != nil {
		return accessToken{}, ErrUnavailable
	}
	raw, err := c.post(ctx, "/auth/v1/signin", c.credentials.PublishableKey, body, true)
	if err != nil {
		return accessToken{}, err
	}
	var response struct {
		TokenType   string `json:"token_type"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		Sub         string `json:"sub"`
	}
	if decodeJSON(raw, &response) != nil || response.TokenType != "Bearer" || response.Sub != c.credentials.ServiceUID || response.ExpiresIn <= 0 || response.ExpiresIn > 2147483647 || len(response.AccessToken) > 4096 || strings.ContainsAny(response.AccessToken, "\r\n") {
		return accessToken{}, ErrUnavailable
	}
	claims, err := tokenClaims(response.AccessToken)
	if err != nil || claims.Sub != c.credentials.ServiceUID || claims.Role != "authenticated" {
		return accessToken{}, ErrUnavailable
	}
	expires := started.Add(time.Duration(response.ExpiresIn) * time.Second)
	if jwtExpiry := time.Unix(claims.Exp, 0); jwtExpiry.Before(expires) {
		expires = jwtExpiry
	}
	if !c.now().Before(expires) {
		return accessToken{}, ErrUnavailable
	}
	ahead := min(refreshAhead, expires.Sub(started)/10)
	return accessToken{value: response.AccessToken, usableUntil: expires.Add(-ahead)}, nil
}

type claims struct {
	Sub       string `json:"sub"`
	Role      string `json:"role"`
	Exp       int64  `json:"exp"`
	Audience  string `json:"aud"`
	ProjectID string `json:"project_id"`
}

// These claims checks reject an unexpected identity; gateway signature verification
// remains the authentication authority.
func tokenClaims(token string) (claims, error) {
	var result claims
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] == "" || parts[2] == "" {
		return result, ErrUnavailable
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || decodeJSON(payload, &result) != nil {
		return result, ErrUnavailable
	}
	return result, nil
}

// Only reviewed, exact RPC routes may return a growing native success result.
// New RPCs retain the default response limit until their contract is reviewed.
func growingSuccessResponse(path string) bool {
	switch path {
	case "/v1/rdb/rest/rpc/order_fulfillment_proof_v1",
		"/v1/rdb/rest/rpc/order_query_list_user_v1",
		"/v1/rdb/rest/rpc/order_query_get_user_v1",
		"/v1/rdb/rest/rpc/order_query_search_merchant_v1",
		"/v1/rdb/rest/rpc/order_query_get_merchant_v1",
		"/v1/rdb/rest/rpc/order_merchant_account_current_v1",
		"/v1/rdb/rest/rpc/order_merchant_account_list_v1",
		"/v1/rdb/rest/rpc/order_merchant_account_command_v1",
		"/v1/rdb/rest/rpc/order_catalog_browse_v1",
		"/v1/rdb/rest/rpc/order_catalog_categories_v1",
		"/v1/rdb/rest/rpc/order_catalog_products_v1",
		"/v1/rdb/rest/rpc/order_catalog_command_v1",
		"/v1/rdb/rest/rpc/order_menu_snapshot_v1",
		"/v1/rdb/rest/rpc/order_storefront_get_v1",
		"/v1/rdb/rest/rpc/order_storefront_admin_v1",
		"/v1/rdb/rest/rpc/order_storefront_configure_v1",
		"/v1/rdb/rest/rpc/order_display_pricing_facts_v1",
		"/v1/rdb/rest/rpc/order_quote_create_v1",
		"/v1/rdb/rest/rpc/order_quote_read_v1",
		"/v1/rdb/rest/rpc/order_staff_admin_list_v1",
		"/v1/rdb/rest/rpc/order_staff_admin_command_v1",
		"/v1/rdb/rest/rpc/order_import_preview_facts_v1",
		"/v1/rdb/rest/rpc/order_import_preview_v1",
		"/v1/rdb/rest/rpc/order_import_commit_v1",
		"/v1/rdb/rest/rpc/order_billing_payment_facts_v1",
		"/v1/rdb/rest/rpc/order_billing_refund_facts_v1",
		"/v1/rdb/rest/rpc/order_admin_report_export_v1",
		"/v1/rdb/rest/rpc/order_admin_report_get_v1",
		"/v1/rdb/rest/rpc/order_admin_report_search_v1",
		"/v1/rdb/rest/rpc/order_admin_report_stats_v1",
		"/v1/rdb/rest/rpc/order_admin_report_payments_v1",
		"/v1/rdb/rest/rpc/order_admin_report_refunds_v1",
		"/v1/rdb/rest/rpc/order_admin_report_pending_v1",
		"/v1/rdb/rest/rpc/order_audit_search_v1":
		return true
	}
	return false
}

func (c *Client) post(ctx context.Context, path, token string, body []byte, signin bool) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.origin+path, bytes.NewReader(body))
	if err != nil {
		return nil, ErrUnavailable
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	if signin {
		req.Header.Set("x-device-id", "order-uat-sharedpg-runtime")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		c.mu.Lock()
		if c.cached.value == token {
			c.cached = accessToken{}
		}
		c.mu.Unlock()
	}
	// These fixed RPCs return native collections or growing original text.
	// Preserve their successful results without adding a business byte limit.
	limited := signin || resp.StatusCode != http.StatusOK || !growingSuccessResponse(path)
	var responseBody io.Reader = resp.Body
	if limited {
		responseBody = io.LimitReader(resp.Body, maxResponseBytes+1)
	}
	raw, err := io.ReadAll(responseBody)
	if err != nil || (limited && len(raw) > maxResponseBytes) {
		return nil, ErrUnavailable
	}
	var envelope struct {
		Code      json.RawMessage `json:"code"`
		Message   json.RawMessage `json:"message"`
		RequestID string          `json:"requestId"`
	}
	_ = decodeJSON(raw, &envelope)
	if resp.StatusCode != http.StatusOK || (len(envelope.Code) > 0 && len(envelope.Message) > 0) {
		if len(raw) > maxResponseBytes {
			return nil, ErrUnavailable
		}
		var code string
		_ = json.Unmarshal(envelope.Code, &code)
		state := strings.TrimPrefix(code, "DATABASE_")
		if !validSQLState(state) {
			state = ""
		}
		requestID := safeRequestID(envelope.RequestID)
		if requestID == "" {
			requestID = safeRequestID(resp.Header.Get("X-Request-Id"))
		}
		for _, secret := range []string{c.credentials.Username, c.credentials.Password, c.credentials.PublishableKey, token} {
			if secret != "" && strings.Contains(requestID, secret) {
				requestID = ""
				break
			}
		}
		return nil, &Error{Status: resp.StatusCode, SQLState: state, RequestID: requestID}
	}
	return raw, nil
}

func decodeJSON(raw []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(out) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return ErrUnavailable
	}
	return nil
}

func validSQLState(value string) bool {
	if len(value) != 5 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9') && !(r >= 'A' && r <= 'Z') {
			return false
		}
	}
	return true
}

func safeRequestID(value string) string {
	if len(value) > 128 {
		return ""
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !strings.ContainsRune("-_", r) {
			return ""
		}
	}
	return value
}
