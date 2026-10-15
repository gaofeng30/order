package staffdiscount

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"strconv"
	"time"
	"unicode/utf8"
)

type sharedStaffExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}

type SharedPGApplication struct {
	client     sharedStaffExecutor
	receiptNow func() time.Time
}

func NewSharedPGApplication(client sharedStaffExecutor) *SharedPGApplication {
	return &SharedPGApplication{client: client, receiptNow: time.Now}
}
func (a *SharedPGApplication) List(ctx context.Context, query string) ([]Staff, error) {
	if a == nil || a.client == nil || ctx == nil {
		return nil, ErrUnavailable
	}
	var f struct {
		Outcome string `json:"outcome"`
		Staff   []struct {
			ID        string    `json:"id"`
			Phone     string    `json:"phone"`
			Name      string    `json:"name"`
			Enabled   bool      `json:"enabled"`
			CreatedAt time.Time `json:"created_at"`
			Bound     bool      `json:"bound"`
			Spend     string    `json:"spend_cents"`
			Count     string    `json:"order_count"`
		} `json:"staff"`
	}
	if a.client.CallRPC(ctx, "order_staff_admin_list_v1", map[string]any{"p_query_hex": hex.EncodeToString([]byte(query))}, &f) != nil || f.Outcome != "SUCCEEDED" || f.Staff == nil {
		return nil, ErrUnavailable
	}
	out := make([]Staff, 0, len(f.Staff))
	for _, s := range f.Staff {
		id, e1 := strconv.ParseUint(s.ID, 10, 64)
		spend, e2 := strconv.ParseUint(s.Spend, 10, 64)
		count, e3 := strconv.ParseUint(s.Count, 10, 32)
		if e1 != nil || e2 != nil || e3 != nil {
			return nil, ErrUnavailable
		}
		out = append(out, Staff{ID: id, Name: s.Name, MaskedPhone: maskPhone(s.Phone), Enabled: s.Enabled, CreatedAt: s.CreatedAt.In(time.Local), Bound: s.Bound, SpendCents: spend, OrderCount: uint32(count)})
	}
	return out, nil
}
func (a *SharedPGApplication) DiscountRate(ctx context.Context) (uint8, error) {
	if a == nil || a.client == nil || ctx == nil {
		return 0, ErrUnavailable
	}
	var f struct {
		Outcome string `json:"outcome"`
		Rate    uint8  `json:"rate_percent"`
	}
	if a.client.CallRPC(ctx, "order_staff_admin_rate_v1", map[string]any{}, &f) != nil || f.Outcome != "SUCCEEDED" || f.Rate < 1 || f.Rate > 100 {
		return 0, ErrUnavailable
	}
	return f.Rate, nil
}
func sharedStaffHash(prefix, value string) string {
	h := sha256.Sum256([]byte(prefix + ":" + value))
	return hex.EncodeToString(h[:])
}

type sharedStaffResponse struct {
	Outcome  string  `json:"outcome"`
	Before   *string `json:"before_text"`
	Response *string `json:"response_text"`
}

func sharedStaffDecode(raw string, out any) bool {
	if !json.Valid([]byte(raw)) {
		return false
	}
	d := json.NewDecoder(bytes.NewReader([]byte(raw)))
	d.DisallowUnknownFields()
	return d.Decode(out) == nil && d.Decode(new(any)) == io.EOF
}
func sharedStaffResult(f sharedStaffResponse, digest [32]byte, replay bool) (Result, error) {
	if f.Response == nil {
		return Result{}, ErrUnavailable
	}
	if replay {
		if f.Before == nil {
			return Result{}, ErrUnavailable
		}
		var proof struct {
			Digest string `json:"request_digest"`
		}
		if len(*f.Before) > 256 || !sharedStaffDecode(*f.Before, &proof) {
			return Result{}, ErrUnavailable
		}
		stored, e := hex.DecodeString(proof.Digest)
		if e != nil || len(stored) != 32 || subtle.ConstantTimeCompare(stored, digest[:]) != 1 {
			return Result{}, ErrIdempotencyConflict
		}
	}
	var r Result
	// Commands return the original persisted name, including legal imports.
	if !sharedStaffDecode(*f.Response, &r) {
		return Result{}, ErrUnavailable
	}
	return r, nil
}
func (a *SharedPGApplication) Execute(ctx context.Context, meta WriteMeta, cmd Command) (Result, error) {
	if a == nil || a.client == nil || ctx == nil {
		return Result{}, ErrUnavailable
	}
	// Native PostgreSQL rejects invalid UTF-8 in this consumed text field.
	// JSON must not replace those bytes and turn it into an unknown command.
	if !utf8.ValidString(string(cmd.Kind)) {
		return Result{}, ErrUnavailable
	}
	raw, err := json.Marshal(cmd)
	if err != nil {
		return Result{}, ErrUnavailable
	}
	digest := sha256.Sum256(raw)
	p := map[string]any{"p_user": strconv.FormatUint(meta.ActorUserID, 10), "p_kind": string(cmd.Kind), "p_staff": strconv.FormatUint(cmd.StaffID, 10), "p_name_hex": nil, "p_name_key_hex": nil, "p_phone_hex": nil, "p_enabled": nil, "p_rate": nil, "p_input_status": "VALID", "p_operation_hash": sharedStaffHash("operation", meta.IdempotencyKey), "p_request_id_hash": nil, "p_request_digest": hex.EncodeToString(digest[:]), "p_receipt_at": nil}
	if meta.RequestID != "" {
		p["p_request_id_hash"] = sharedStaffHash("request", meta.RequestID)
	}
	if cmd.Kind == CreateStaff || cmd.Kind == UpdateStaff {
		phone := cmd.Phone
		if cmd.Kind == UpdateStaff && phone == "" {
			phone = "+1"
		}
		name, key, phone, ok := staffInput(cmd.Name, phone)
		p["p_name_hex"], p["p_name_key_hex"] = hex.EncodeToString([]byte(name)), hex.EncodeToString(key)
		if cmd.Kind != UpdateStaff || cmd.Phone != "" {
			p["p_phone_hex"] = hex.EncodeToString([]byte(phone))
		}
		if !ok {
			p["p_input_status"] = "INVALID_INPUT"
		}
	}
	if cmd.Kind == SetStaffEnabled {
		p["p_enabled"] = cmd.Enabled
	}
	if cmd.Kind == SetDiscountRate {
		p["p_rate"] = int(cmd.RatePercent)
	}
	var f sharedStaffResponse
	if a.client.CallRPC(ctx, "order_staff_admin_command_v1", p, &f) != nil {
		return Result{}, ErrUnavailable
	}
	if f.Outcome == "NEED_RECEIPT" {
		if a.receiptNow == nil {
			return Result{}, ErrUnavailable
		}
		p["p_receipt_at"] = a.receiptNow().UTC().Truncate(time.Microsecond)
		if a.client.CallRPC(ctx, "order_staff_admin_command_v1", p, &f) != nil {
			return Result{}, ErrUnavailable
		}
	}
	switch f.Outcome {
	case "REPLAY":
		return sharedStaffResult(f, digest, true)
	case "SUCCEEDED":
		return sharedStaffResult(f, digest, false)
	case "INVALID_INPUT":
		return Result{}, ErrInvalidInput
	case "NOT_FOUND":
		return Result{}, ErrNotFound
	case "CONFLICT":
		return Result{}, ErrConflict
	default:
		return Result{}, ErrUnavailable
	}
}

var _ Application = (*SharedPGApplication)(nil)
