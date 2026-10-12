package merchantidentity

import (
	"context"
	"database/sql"
	"encoding/hex"
	"strconv"
	"time"
)

type sharedPGIdentityExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}

// SharedPGIdentityReader supports the existing current-identity read boundary.
// Login and phone mutations are separate ports and are not implemented here.
type SharedPGIdentityReader struct{ client sharedPGIdentityExecutor }

func NewSharedPGIdentityReader(client sharedPGIdentityExecutor) *SharedPGIdentityReader {
	return &SharedPGIdentityReader{client: client}
}

func (reader *SharedPGIdentityReader) Identity(ctx context.Context, userID uint64) (Identity, error) {
	if ctx == nil || reader == nil || reader.client == nil || userID == 0 {
		return Identity{}, ErrUnavailable
	}
	var facts sharedPGIdentityFacts
	if reader.client.CallRPC(ctx, "order_identity_projection_v1", map[string]any{"p_user": strconv.FormatUint(userID, 10)}, &facts) != nil {
		return Identity{}, ErrUnavailable
	}
	record, err := facts.record()
	if err != nil {
		return Identity{}, ErrUnavailable
	}
	projection, err := projectIdentity(record)
	if err != nil || !validIdentity(projection, false) {
		return Identity{}, ErrUnavailable
	}
	return projection, nil
}

type sharedPGIdentityFacts struct {
	PhoneHex                 *string    `json:"phone_hex"`
	PhoneBoundAt             *time.Time `json:"phone_bound_at"`
	ExtraPhoneHex            *string    `json:"extra_phone_hex"`
	ExtraName                *string    `json:"extra_name"`
	ExtraNameKeyHex          *string    `json:"extra_name_key_hex"`
	ExtraPhoneSetAt          *time.Time `json:"extra_phone_set_at"`
	RecordVersion            string     `json:"record_version"`
	AccountID                *string    `json:"account_id"`
	AccountRole              *string    `json:"account_role"`
	AuthVersion              *string    `json:"auth_version"`
	RatePercent              string     `json:"rate_percent"`
	WhitelistVersion         string     `json:"whitelist_version"`
	PrimaryEnabled           *bool      `json:"primary_enabled"`
	ExtraEnabled             *bool      `json:"extra_enabled"`
	WhitelistExtraNameKeyHex *string    `json:"whitelist_extra_name_key_hex"`
}

func (f sharedPGIdentityFacts) record() (identityRecord, error) {
	var r identityRecord
	var err error
	bytesOf := func(value *string) ([]byte, error) {
		if value == nil {
			return nil, nil
		}
		return hex.DecodeString(*value)
	}
	phone, err := bytesOf(f.PhoneHex)
	if err != nil {
		return r, err
	}
	extra, err := bytesOf(f.ExtraPhoneHex)
	if err != nil {
		return r, err
	}
	r.phone = sql.NullString{String: string(phone), Valid: f.PhoneHex != nil}
	r.extraPhone = sql.NullString{String: string(extra), Valid: f.ExtraPhoneHex != nil}
	if f.ExtraName != nil {
		r.extraName = sql.NullString{String: *f.ExtraName, Valid: true}
	}
	if f.AccountRole != nil {
		r.role = sql.NullString{String: *f.AccountRole, Valid: true}
	}
	if f.PhoneBoundAt != nil {
		r.phoneBoundAt = sql.NullTime{Time: *f.PhoneBoundAt, Valid: true}
	}
	if f.ExtraPhoneSetAt != nil {
		r.extraPhoneSetAt = sql.NullTime{Time: *f.ExtraPhoneSetAt, Valid: true}
	}
	if f.PrimaryEnabled != nil {
		r.primaryEnabled = sql.NullBool{Bool: *f.PrimaryEnabled, Valid: true}
	}
	if f.ExtraEnabled != nil {
		r.extraEnabled = sql.NullBool{Bool: *f.ExtraEnabled, Valid: true}
	}
	r.extraNameKey, err = bytesOf(f.ExtraNameKeyHex)
	if err != nil {
		return r, err
	}
	r.whitelistExtraNameKey, err = bytesOf(f.WhitelistExtraNameKeyHex)
	if err != nil {
		return r, err
	}
	r.recordVersion, err = strconv.ParseUint(f.RecordVersion, 10, 64)
	if err != nil {
		return r, err
	}
	r.whitelistVersion, err = strconv.ParseUint(f.WhitelistVersion, 10, 64)
	if err != nil {
		return r, err
	}
	rate, err := strconv.ParseUint(f.RatePercent, 10, 8)
	if err != nil {
		return r, err
	}
	r.ratePercent = uint8(rate)
	if f.AccountID != nil {
		v, err := strconv.ParseUint(*f.AccountID, 10, 64)
		if err != nil {
			return r, err
		}
		r.accountID = sql.Null[uint64]{V: v, Valid: true}
	}
	if f.AuthVersion != nil {
		v, err := strconv.ParseUint(*f.AuthVersion, 10, 64)
		if err != nil {
			return r, err
		}
		r.authVersion = sql.Null[uint64]{V: v, Valid: true}
	}
	return r, nil
}
