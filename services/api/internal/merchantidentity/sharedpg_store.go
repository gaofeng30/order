package merchantidentity

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/gaofeng30/order/services/api/internal/cloudbasepg"
)

// SharedPGStore keeps the existing merchant Service/provider boundary and
// persists each of its short transactions with fixed ordinary-user RPCs.
type SharedPGStore struct{ client sharedPGIdentityExecutor }

func NewSharedPGStore(client sharedPGIdentityExecutor) *SharedPGStore {
	return &SharedPGStore{client: client}
}

func (store *SharedPGStore) ReadIdentity(ctx context.Context, userID uint64) (Identity, error) {
	if store == nil {
		return Identity{}, ErrUnavailable
	}
	return NewSharedPGIdentityReader(store.client).Identity(ctx, userID)
}

type sharedPGLoginFacts struct {
	Outcome       string `json:"outcome"`
	OpenIDHex     string `json:"openid_hex"`
	AccountID     string `json:"account_id"`
	Role          Role   `json:"role"`
	Enabled       bool   `json:"enabled"`
	RecordVersion string `json:"record_version"`
	AuthVersion   string `json:"auth_version"`
}

func (facts sharedPGLoginFacts) identity() (Identity, error) {
	parse := func(v string) (uint64, error) {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil || strconv.FormatUint(id, 10) != v {
			return 0, ErrUnavailable
		}
		return id, nil
	}
	id, e1 := parse(facts.AccountID)
	record, e2 := parse(facts.RecordVersion)
	auth, e3 := parse(facts.AuthVersion)
	account := accountState{ID: id, Role: facts.Role, Enabled: facts.Enabled, RecordVersion: record, AuthVersion: auth}
	if e1 != nil || e2 != nil || e3 != nil || !validAccount(account) || !account.Enabled {
		return Identity{}, ErrUnavailable
	}
	return accountIdentity(account), nil
}

func (store *SharedPGStore) StartLogin(ctx context.Context, userID uint64, codeHash LoginCodeHash, requestID string, at time.Time) (LoginStart, error) {
	if store == nil || store.client == nil || ctx == nil || userID == 0 || !validLoginCodeHash(codeHash) || !validRequestID(requestID) || at.IsZero() {
		return LoginStart{}, ErrUnavailable
	}
	requestHash := sha256.Sum256([]byte(requestID))
	var facts sharedPGLoginFacts
	if store.client.CallRPC(ctx, "order_merchant_login_start_v1", map[string]any{"p_user": strconv.FormatUint(userID, 10), "p_code_hash": hex.EncodeToString(codeHash[:]), "p_request_hash": hex.EncodeToString(requestHash[:]), "p_at": at.UTC().Truncate(time.Microsecond)}, &facts) != nil {
		return LoginStart{}, ErrUnavailable
	}
	switch facts.Outcome {
	case "ACCOUNT_NOT_AVAILABLE":
		return LoginStart{}, ErrMerchantAccountNotAvailable
	case "NEEDS_PROVIDER":
		openid, err := hex.DecodeString(facts.OpenIDHex)
		if err != nil || len(openid) == 0 {
			return LoginStart{}, ErrUnavailable
		}
		return LoginStart{OpenID: string(openid)}, nil
	case "SUCCEEDED":
		projection, err := facts.identity()
		if err != nil {
			return LoginStart{}, ErrUnavailable
		}
		return LoginStart{AlreadyBound: true, Existing: projection}, nil
	default:
		return LoginStart{}, ErrUnavailable
	}
}

func (store *SharedPGStore) CompleteLogin(ctx context.Context, userID uint64, phone string, codeHash LoginCodeHash, requestID string, at time.Time) (Identity, error) {
	if store == nil || store.client == nil || ctx == nil || userID == 0 || !canonicalPhone(phone) || !validLoginCodeHash(codeHash) || !validRequestID(requestID) || at.IsZero() {
		return Identity{}, ErrUnavailable
	}
	requestHash := sha256.Sum256([]byte(requestID))
	params := map[string]any{"p_user": strconv.FormatUint(userID, 10), "p_phone_hex": hex.EncodeToString([]byte(phone)), "p_code_hash": hex.EncodeToString(codeHash[:]), "p_request_hash": hex.EncodeToString(requestHash[:]), "p_at": at.UTC().Truncate(time.Microsecond)}
	for attempt := 0; attempt < 2; attempt++ {
		var facts sharedPGLoginFacts
		err := store.client.CallRPC(ctx, "order_merchant_login_complete_v1", params, &facts)
		if attempt == 0 && sharedPGRetryableTransaction(err) {
			continue
		}
		if err != nil {
			return Identity{}, ErrUnavailable
		}
		return sharedPGLoginResult(facts)
	}
	return Identity{}, ErrUnavailable
}

func sharedPGRetryableTransaction(err error) bool {
	var failure *cloudbasepg.Error
	return errors.As(err, &failure) && (failure.SQLState == "40P01" || failure.SQLState == "40001" || failure.SQLState == "55P03")
}

func sharedPGLoginResult(facts sharedPGLoginFacts) (Identity, error) {
	switch facts.Outcome {
	case "SUCCEEDED":
		return facts.identity()
	case "PRIMARY_PHONE_MISMATCH":
		return Identity{}, ErrPrimaryPhoneMismatch
	case "ACCOUNT_NOT_AVAILABLE":
		return Identity{}, ErrMerchantAccountNotAvailable
	case "PHONE_IN_USE":
		return Identity{}, ErrPhoneInUse
	case "PHONE_CODE_REJECTED":
		return Identity{}, ErrPhoneCodeRejected
	default:
		return Identity{}, ErrUnavailable
	}
}

func (store *SharedPGStore) RecoverRejectedLogin(ctx context.Context, userID uint64, codeHash LoginCodeHash, requestID string, startedAt, at time.Time) (Identity, error) {
	if store == nil || store.client == nil || ctx == nil || userID == 0 || !validLoginCodeHash(codeHash) || !validRequestID(requestID) || startedAt.IsZero() || at.Before(startedAt) {
		return Identity{}, ErrUnavailable
	}
	requestHash := sha256.Sum256([]byte(requestID))
	var facts sharedPGLoginFacts
	if store.client.CallRPC(ctx, "order_merchant_login_recover_v1", map[string]any{"p_user": strconv.FormatUint(userID, 10), "p_code_hash": hex.EncodeToString(codeHash[:]), "p_request_hash": hex.EncodeToString(requestHash[:]), "p_started_at": startedAt.UTC().Truncate(time.Microsecond), "p_at": at.UTC().Truncate(time.Microsecond)}, &facts) != nil {
		return Identity{}, ErrUnavailable
	}
	return sharedPGLoginResult(facts)
}

func (store *SharedPGStore) SetContactProfile(ctx context.Context, meta WriteMeta, command ContactProfileCommand) (ContactProfileResult, error) {
	command, key, ok := canonicalProfileCommand(command)
	if store == nil || store.client == nil || ctx == nil || !ok || !validIdentityWriteMeta(meta) {
		return ContactProfileResult{}, ErrInvalidInput
	}
	digest := digestContactProfileCommand(command)
	operation := sha256.Sum256([]byte(meta.IdempotencyKey))
	request := sha256.Sum256([]byte(meta.RequestID))
	params := map[string]any{"p_user": strconv.FormatUint(meta.ActorUserID, 10), "p_phone_hex": nil, "p_name": nil, "p_name_key_hex": nil, "p_operation_hash": hex.EncodeToString(operation[:]), "p_request_hash": hex.EncodeToString(request[:]), "p_request_digest": hex.EncodeToString(digest[:])}
	if command.Field == profileName {
		params["p_name"] = command.Value
		params["p_name_key_hex"] = hex.EncodeToString(key)
	} else {
		params["p_phone_hex"] = hex.EncodeToString([]byte(command.Value))
	}
	for attempt := 0; attempt < 2; attempt++ {
		var f struct {
			Outcome  string      `json:"outcome"`
			Kind     PricingKind `json:"kind"`
			Rate     uint8       `json:"rate_percent"`
			Evidence string      `json:"evidence_text"`
			Response string      `json:"response_text"`
			Phone    *string     `json:"current_phone_hex"`
			Name     *string     `json:"current_name"`
		}
		err := store.client.CallRPC(ctx, "order_identity_extra_phone_set_v1", params, &f)
		if attempt == 0 && sharedPGRetryableTransaction(err) {
			continue
		}
		if err != nil {
			return ContactProfileResult{}, ErrUnavailable
		}
		if f.Outcome == "PRIMARY_PHONE_REQUIRED" {
			return ContactProfileResult{}, ErrPrimaryPhoneRequired
		}
		if f.Outcome == "REPLAY" {
			got, err := decodeExtraPhoneEvidence([]byte(f.Evidence))
			if err != nil || subtle.ConstantTimeCompare(got[:], digest[:]) != 1 {
				return ContactProfileResult{}, ErrIdempotencyConflict
			}
			var receipt extraPhoneReceipt
			if !decodeExactJSON([]byte(f.Response), &receipt) || !validPricing(receipt.Pricing.Kind, receipt.Pricing.RatePercent) {
				return ContactProfileResult{}, ErrUnavailable
			}
			f.Kind, f.Rate = receipt.Pricing.Kind, receipt.Pricing.RatePercent
		} else if f.Outcome != "SUCCEEDED" {
			return ContactProfileResult{}, ErrUnavailable
		}
		if !validPricing(f.Kind, f.Rate) {
			return ContactProfileResult{}, ErrUnavailable
		}
		result := ContactProfileResult{Pricing: PricingProjection{Kind: f.Kind, RatePercent: f.Rate}}
		if f.Name != nil {
			_, _, valid := canonicalProfileName(*f.Name)
			if !valid {
				return ContactProfileResult{}, ErrUnavailable
			}
			result.DefaultContactName = *f.Name
		}
		if f.Phone != nil {
			raw, err := hex.DecodeString(*f.Phone)
			if err != nil || !canonicalPhone(string(raw)) {
				return ContactProfileResult{}, ErrUnavailable
			}
			result.ExtraPhone = &ExtraPhoneProjection{Phone: string(raw)}
		}
		if (command.Field == profileName && result.DefaultContactName != command.Value) || (command.Field == profilePhone && (result.ExtraPhone == nil || result.ExtraPhone.Phone != command.Value)) {
			return ContactProfileResult{}, ErrIdempotencyConflict
		}
		return result, nil
	}
	return ContactProfileResult{}, ErrUnavailable
}

var _ Store = (*SharedPGStore)(nil)
