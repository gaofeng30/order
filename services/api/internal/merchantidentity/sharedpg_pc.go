package merchantidentity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

func sharedPGPCSecretHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
func (a *SharedPGAdminApplication) BeginPCLogin(ctx context.Context) (PCLogin, error) {
	if a == nil || a.client == nil || a.newSecret == nil || a.now == nil {
		return PCLogin{}, ErrAdminUnavailable
	}
	approval, err := a.newSecret()
	if err != nil {
		return PCLogin{}, ErrAdminUnavailable
	}
	poll, err := a.newSecret()
	if err != nil {
		return PCLogin{}, ErrAdminUnavailable
	}
	now := a.now().UTC().Truncate(time.Microsecond)
	var facts struct {
		Outcome string `json:"outcome"`
		LoginID string `json:"login_id"`
	}
	if a.client.CallRPC(ctx, "order_pc_login_begin_v1", map[string]any{"p_approval_hash": sharedPGPCSecretHash(approval), "p_poll_hash": sharedPGPCSecretHash(poll), "p_now": now}, &facts) != nil || facts.Outcome != "SUCCEEDED" {
		return PCLogin{}, ErrAdminUnavailable
	}
	id, err := strconv.ParseUint(facts.LoginID, 10, 64)
	if err != nil || id == 0 || strconv.FormatUint(id, 10) != facts.LoginID {
		return PCLogin{}, ErrAdminUnavailable
	}
	return PCLogin{LoginID: facts.LoginID, PollSecret: poll, QRPayload: "order-admin-login://approve?login_id=" + facts.LoginID + "&approval_secret=" + approval, ExpiresAt: now.Add(2 * time.Minute)}, nil
}
func (a *SharedPGAdminApplication) ApprovePCLogin(ctx context.Context, userID uint64, loginID, approvalSecret, phoneCode string) error {
	id, err := parseAdminID(loginID)
	if err != nil {
		return ErrAdminInvalidInput
	}
	if a == nil || a.client == nil || a.now == nil {
		return ErrAdminUnavailable
	}
	hash := sharedPGPCSecretHash(approvalSecret)
	var facts struct {
		Outcome   string    `json:"outcome"`
		State     string    `json:"state"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	// The native first read maps both missing rows and storage faults to expired.
	if a.client.CallRPC(ctx, "order_pc_login_approval_read_v1", map[string]any{"p_login": strconv.FormatUint(id, 10), "p_approval_hash": hash}, &facts) != nil || facts.Outcome != "SUCCEEDED" || facts.State != "WAITING" || !a.now().Before(facts.ExpiresAt) {
		return ErrPCLoginExpired
	}
	if a.mini == nil {
		return ErrAdminUnavailable
	}
	projection, err := a.mini.Login(ctx, userID, phoneCode, "pc-approve-"+loginID)
	if err != nil {
		return err
	}
	if projection.Merchant == nil || projection.Merchant.Role != RoleOwner {
		return ErrForbidden
	}
	var result struct {
		Outcome string `json:"outcome"`
	}
	if a.client.CallRPC(ctx, "order_pc_login_approve_v1", map[string]any{"p_user": strconv.FormatUint(userID, 10), "p_login": strconv.FormatUint(id, 10), "p_approval_hash": hash}, &result) != nil {
		return ErrAdminUnavailable
	}
	switch result.Outcome {
	case "SUCCEEDED":
		return nil
	case "FORBIDDEN":
		return ErrForbidden
	case "LOGIN_EXPIRED":
		return ErrPCLoginExpired
	default:
		return ErrAdminUnavailable
	}
}
func (a *SharedPGAdminApplication) PollPCLogin(ctx context.Context, loginID, pollSecret string) (PCSession, error) {
	id, err := parseAdminID(loginID)
	if err != nil {
		return PCSession{}, ErrAdminInvalidInput
	}
	if a == nil || a.client == nil {
		return PCSession{}, ErrAdminUnavailable
	}
	params := map[string]any{"p_login": strconv.FormatUint(id, 10), "p_poll_hash": sharedPGPCSecretHash(pollSecret), "p_now": nil, "p_access_hash": nil, "p_entropy_failed": false}
	token := ""
	for attempt := 0; attempt < 2; attempt++ {
		var facts struct {
			Outcome    string     `json:"outcome"`
			ExpiresAt  time.Time  `json:"expires_at"`
			ObservedAt *time.Time `json:"observed_at"`
		}
		if a.client.CallRPC(ctx, "order_pc_login_poll_v1", params, &facts) != nil {
			return PCSession{}, ErrAdminUnavailable
		}
		switch facts.Outcome {
		case "WAITING":
			return PCSession{State: "WAITING"}, nil
		case "LOGIN_EXPIRED":
			return PCSession{}, ErrPCLoginExpired
		case "APPROVED":
			if attempt != 1 || token == "" {
				return PCSession{}, ErrAdminUnavailable
			}
			return PCSession{State: "APPROVED", Token: token, ExpiresAt: facts.ExpiresAt}, nil
		case "NEED_MATERIAL":
			if attempt != 0 || facts.ObservedAt == nil {
				return PCSession{}, ErrAdminUnavailable
			}
			params["p_now"] = facts.ObservedAt.UTC().Truncate(time.Microsecond)
			if a.newSecret == nil {
				params["p_entropy_failed"] = true
			} else {
				token, err = a.newSecret()
				if err != nil {
					params["p_entropy_failed"] = true
				} else {
					params["p_access_hash"] = sharedPGPCSecretHash(token)
				}
			}
		default:
			return PCSession{}, ErrAdminUnavailable
		}
	}
	return PCSession{}, ErrAdminUnavailable
}
func (a *SharedPGAdminApplication) AuthenticatePC(ctx context.Context, token string) (uint64, error) {
	if a == nil || a.client == nil || a.now == nil {
		return 0, ErrAdminUnavailable
	}
	var facts struct {
		Outcome string `json:"outcome"`
		UserID  string `json:"user_id"`
	}
	if a.client.CallRPC(ctx, "order_pc_session_authenticate_v1", map[string]any{"p_token_hash": sharedPGPCSecretHash(token), "p_now": a.now().UTC().Truncate(time.Microsecond)}, &facts) != nil {
		return 0, ErrAdminUnavailable
	}
	if facts.Outcome == "SESSION_EXPIRED" {
		return 0, ErrPCSessionExpired
	}
	if facts.Outcome != "SUCCEEDED" {
		return 0, ErrAdminUnavailable
	}
	id, err := strconv.ParseUint(facts.UserID, 10, 64)
	if err != nil || id == 0 || strconv.FormatUint(id, 10) != facts.UserID {
		return 0, ErrAdminUnavailable
	}
	return id, nil
}
func (a *SharedPGAdminApplication) RevokePC(ctx context.Context, token string) error {
	if a == nil || a.client == nil {
		return ErrAdminUnavailable
	}
	var facts struct {
		Outcome string `json:"outcome"`
	}
	if a.client.CallRPC(ctx, "order_pc_session_revoke_v1", map[string]any{"p_token_hash": sharedPGPCSecretHash(token)}, &facts) != nil || facts.Outcome != "SUCCEEDED" {
		return ErrAdminUnavailable
	}
	return nil
}

var _ AdminApplication = (*SharedPGAdminApplication)(nil)
