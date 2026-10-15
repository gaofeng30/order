package merchantidentity

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/cloudbasepg"
)

// SharedPGAdminApplication retains the existing admin/provider contract.
type SharedPGAdminApplication struct {
	client     sharedPGIdentityExecutor
	mini       Application
	now        func() time.Time
	receiptNow func() time.Time
	newSecret  func() (string, error)
}

func NewSharedPGAdminApplication(client sharedPGIdentityExecutor, mini Application) *SharedPGAdminApplication {
	return &SharedPGAdminApplication{client: client, mini: mini, now: time.Now, receiptNow: time.Now, newSecret: secret}
}

type sharedPGAccountFacts struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Role    Role   `json:"role"`
	Enabled bool   `json:"enabled"`
	Bound   bool   `json:"bound"`
}

func (facts sharedPGAccountFacts) account() (Account, error) {
	id, err := strconv.ParseUint(facts.ID, 10, 64)
	if err != nil || id == 0 || strconv.FormatUint(id, 10) != facts.ID {
		return Account{}, ErrAdminUnavailable
	}
	return Account{ID: id, Name: facts.Name, MaskedPhone: maskMerchantPhone(facts.Phone), Role: facts.Role, Enabled: facts.Enabled, Bound: facts.Bound}, nil
}
func (a *SharedPGAdminApplication) CurrentAccount(ctx context.Context, userID uint64) (Account, error) {
	if a == nil || a.client == nil {
		return Account{}, ErrAdminUnavailable
	}
	var facts struct {
		Outcome string               `json:"outcome"`
		Account sharedPGAccountFacts `json:"account"`
	}
	if a.client.CallRPC(ctx, "order_merchant_account_current_v1", map[string]any{"p_user": strconv.FormatUint(userID, 10)}, &facts) != nil {
		return Account{}, ErrAdminUnavailable
	}
	if facts.Outcome == "NOT_FOUND" {
		return Account{}, ErrAdminNotFound
	}
	if facts.Outcome != "SUCCEEDED" {
		return Account{}, ErrAdminUnavailable
	}
	return facts.Account.account()
}
func (a *SharedPGAdminApplication) ListAccounts(ctx context.Context, userID uint64, q string) ([]Account, error) {
	if a == nil || a.client == nil || !utf8.ValidString(q) {
		return nil, ErrAdminUnavailable
	}
	var facts struct {
		Outcome  string                 `json:"outcome"`
		Accounts []sharedPGAccountFacts `json:"accounts"`
	}
	if a.client.CallRPC(ctx, "order_merchant_account_list_v1", map[string]any{"p_user": strconv.FormatUint(userID, 10), "p_query": q}, &facts) != nil || facts.Outcome != "SUCCEEDED" {
		return nil, ErrAdminUnavailable
	}
	out := make([]Account, 0, len(facts.Accounts))
	for _, item := range facts.Accounts {
		account, err := item.account()
		if err != nil {
			return nil, err
		}
		out = append(out, account)
	}
	return out, nil
}
func sharedPGAdminHash(prefix, value string) string {
	sum := sha256.Sum256([]byte(prefix + ":" + value))
	return hex.EncodeToString(sum[:])
}
func sharedPGAdminParameters(meta AdminWriteMeta, cmd AccountCommand, digest [32]byte) map[string]any {
	params := map[string]any{"p_actor": strconv.FormatUint(meta.ActorUserID, 10), "p_kind": string(cmd.Kind), "p_account": strconv.FormatUint(cmd.AccountID, 10), "p_name": nil, "p_phone": nil, "p_role": nil, "p_enabled": nil, "p_operation_hash": sharedPGAdminHash("operation", meta.IdempotencyKey), "p_request_hash": nil, "p_request_digest": hex.EncodeToString(digest[:]), "p_receipt_at": nil, "p_input_status": "VALID"}
	if meta.RequestID != "" {
		params["p_request_hash"] = sharedPGAdminHash("request", meta.RequestID)
	}
	switch cmd.Kind {
	case CreateAccount, UpdateAccount:
		name := strings.TrimSpace(cmd.Name)
		phone, valid := canonicalAdminPhone(cmd.Phone)
		if cmd.Kind == UpdateAccount && cmd.Phone == "" {
			valid = true
		} else {
			params["p_phone"] = phone
		}
		params["p_role"] = string(cmd.Role)
		if !valid || name == "" || (cmd.Role != RoleOwner && cmd.Role != RoleSubaccount) {
			params["p_input_status"] = "INVALID_INPUT"
		}
		if !utf8.ValidString(name) || strings.ContainsRune(name, 0) {
			if params["p_input_status"] == "VALID" {
				params["p_input_status"] = "STORAGE_UNAVAILABLE"
			}
		} else {
			params["p_name"] = name
		}
		// Invalid ignored/used role bytes cannot be encoded into PostgreSQL text.
		if !utf8.ValidString(string(cmd.Role)) || strings.ContainsRune(string(cmd.Role), 0) {
			params["p_role"] = nil
		}
		if !valid {
			params["p_phone"] = nil
		}
	case SetAccountEnabled:
		if cmd.Enabled != nil {
			params["p_enabled"] = *cmd.Enabled
		}
	}
	return params
}
func sharedPGAdminStorageError(err error) error {
	var pg *cloudbasepg.Error
	if errors.As(err, &pg) && (pg.SQLState == "23505" || pg.SQLState == "23503" || pg.SQLState == "23514") {
		return ErrAdminConflict
	}
	return ErrAdminUnavailable
}
func (a *SharedPGAdminApplication) ExecuteAccount(ctx context.Context, meta AdminWriteMeta, cmd AccountCommand) (*Account, error) {
	if a == nil || a.client == nil {
		return nil, ErrAdminUnavailable
	}
	raw, err := json.Marshal(cmd)
	if err != nil {
		return nil, ErrAdminUnavailable
	}
	digest := sha256.Sum256(raw)
	params := sharedPGAdminParameters(meta, cmd, digest)
	for attempt := 0; attempt < 2; attempt++ {
		var facts struct {
			Outcome      string `json:"outcome"`
			EvidenceText string `json:"evidence_text"`
			ResponseText string `json:"response_text"`
		}
		if err = a.client.CallRPC(ctx, "order_merchant_account_command_v1", params, &facts); err != nil {
			return nil, sharedPGAdminStorageError(err)
		}
		switch facts.Outcome {
		case "NEED_MATERIAL":
			if attempt != 0 || a.receiptNow == nil {
				return nil, ErrAdminUnavailable
			}
			params["p_receipt_at"] = a.receiptNow().UTC().Truncate(time.Microsecond)
			continue
		case "INVALID_INPUT":
			return nil, ErrAdminInvalidInput
		case "NOT_FOUND":
			return nil, ErrAdminNotFound
		case "LAST_OWNER":
			return nil, ErrLastOwner
		case "REPLAY":
			var evidence struct {
				RequestDigest string `json:"request_digest"`
			}
			if !json.Valid([]byte(facts.EvidenceText)) || json.Unmarshal([]byte(facts.EvidenceText), &evidence) != nil {
				return nil, ErrAdminUnavailable
			}
			stored, e := hex.DecodeString(evidence.RequestDigest)
			if e != nil || len(stored) != len(digest) || subtle.ConstantTimeCompare(stored, digest[:]) != 1 {
				return nil, ErrAdminIdempotencyConflict
			}
		case "SUCCEEDED":
		default:
			return nil, ErrAdminUnavailable
		}
		var response struct {
			Account *Account `json:"account"`
			Deleted bool     `json:"deleted"`
		}
		if !json.Valid([]byte(facts.ResponseText)) || json.Unmarshal([]byte(facts.ResponseText), &response) != nil {
			return nil, ErrAdminUnavailable
		}
		return response.Account, nil
	}
	return nil, ErrAdminUnavailable
}
