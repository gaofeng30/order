package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

type sharedPGExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}

func (repository *SharedPGRepository) FindActiveUser(ctx context.Context, tokenHash [sha256.Size]byte, at time.Time) (uint64, error) {
	if repository == nil || repository.client == nil || ctx == nil {
		return 0, errPersistence
	}
	var facts struct {
		Outcome string `json:"outcome"`
		UserID  string `json:"user_id"`
	}
	if repository.client.CallRPC(ctx, "order_identity_session_find_v1", map[string]any{
		"p_token_hash": hex.EncodeToString(tokenHash[:]), "p_now": sharedPGTime(at),
	}, &facts) != nil {
		return 0, errPersistence
	}
	if facts.Outcome == "NOT_FOUND" {
		return 0, ErrSessionNotFound
	}
	if facts.Outcome != "SUCCEEDED" {
		return 0, errPersistence
	}
	id, err := strconv.ParseUint(facts.UserID, 10, 64)
	if err != nil || id == 0 || strconv.FormatUint(id, 10) != facts.UserID {
		return 0, errPersistence
	}
	return id, nil
}

func (repository *SharedPGRepository) RevokeSession(ctx context.Context, tokenHash [sha256.Size]byte) error {
	if repository == nil || repository.client == nil || ctx == nil {
		return errPersistence
	}
	var facts struct {
		Outcome string `json:"outcome"`
	}
	if repository.client.CallRPC(ctx, "order_identity_session_revoke_v1", map[string]any{"p_token_hash": hex.EncodeToString(tokenHash[:])}, &facts) != nil || facts.Outcome != "SUCCEEDED" {
		return errPersistence
	}
	return nil
}

var _ SessionStore = (*SharedPGRepository)(nil)

func (repository *SharedPGRepository) FindPhoneUser(ctx context.Context, userID uint64) (PhoneUser, error) {
	if repository == nil || repository.client == nil || ctx == nil || userID == 0 {
		return PhoneUser{}, errPersistence
	}
	var facts struct {
		Outcome   string  `json:"outcome"`
		OpenIDHex string  `json:"openid_hex"`
		PhoneHex  *string `json:"primary_phone_hex"`
	}
	if repository.client.CallRPC(ctx, "order_identity_phone_read_v1", map[string]any{"p_user": strconv.FormatUint(userID, 10)}, &facts) != nil || facts.Outcome != "SUCCEEDED" {
		return PhoneUser{}, errPersistence
	}
	openid, err := hex.DecodeString(facts.OpenIDHex)
	if err != nil || len(openid) == 0 {
		return PhoneUser{}, errPersistence
	}
	user := PhoneUser{OpenID: string(openid), PrimaryPhoneBound: facts.PhoneHex != nil}
	if facts.PhoneHex != nil {
		phone, err := hex.DecodeString(*facts.PhoneHex)
		if err != nil {
			return PhoneUser{}, errPersistence
		}
		user.PrimaryPhone = string(phone)
	}
	return user, nil
}

func (repository *SharedPGRepository) BindPrimaryPhone(ctx context.Context, userID uint64, phone string, boundAt time.Time) (string, error) {
	if repository == nil || repository.client == nil || ctx == nil {
		return "", errPersistence
	}
	var facts struct {
		Outcome  string `json:"outcome"`
		PhoneHex string `json:"phone_hex"`
	}
	if repository.client.CallRPC(ctx, "order_identity_phone_bind_v1", map[string]any{"p_user": strconv.FormatUint(userID, 10), "p_phone_hex": hex.EncodeToString([]byte(phone)), "p_bound_at": sharedPGTime(boundAt)}, &facts) != nil {
		return "", errPersistence
	}
	switch facts.Outcome {
	case "PHONE_IN_USE":
		return "", ErrPhoneInUse
	case "PRIMARY_PHONE_ALREADY_BOUND":
		return "", ErrPrimaryPhoneAlreadyBound
	case "SUCCEEDED":
		stored, err := hex.DecodeString(facts.PhoneHex)
		if err != nil || string(stored) != phone {
			return "", errPersistence
		}
		return string(stored), nil
	default:
		return "", errPersistence
	}
}

var _ PhoneStore = (*SharedPGRepository)(nil)

// SharedPGRepository persists Order identity through fixed ordinary-user RPCs.
type SharedPGRepository struct{ client sharedPGExecutor }

func NewSharedPGRepository(client sharedPGExecutor) *SharedPGRepository {
	return &SharedPGRepository{client: client}
}

func (repository *SharedPGRepository) CreateSession(ctx context.Context, params CreateSessionParams) error {
	if repository == nil || repository.client == nil || ctx == nil {
		return errPersistence
	}
	var response struct {
		Outcome string `json:"outcome"`
	}
	err := repository.client.CallRPC(ctx, "order_identity_session_create_v1", map[string]any{
		"p_openid_hex": hex.EncodeToString([]byte(params.OpenID)),
		"p_token_hash": hex.EncodeToString(params.TokenHash[:]),
		"p_issued_at":  sharedPGTime(params.IssuedAt), "p_expires_at": sharedPGTime(params.ExpiresAt),
	}, &response)
	if err != nil || response.Outcome != "SUCCEEDED" {
		return errPersistence
	}
	return nil
}

// Native pgx binary timestamptz binding truncates nanoseconds to microseconds.
func sharedPGTime(at time.Time) time.Time { return at.UTC().Truncate(time.Microsecond) }
