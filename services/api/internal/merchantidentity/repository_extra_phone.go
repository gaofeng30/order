package merchantidentity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/text/unicode/norm"
)

var errExtraReceiptExists = errors.New("contact profile receipt exists")

type extraPhoneEvidence struct {
	RequestDigest string `json:"request_digest"`
}
type extraPhoneReceipt struct {
	Pricing pricingReceipt `json:"pricing_identity"`
}
type pricingReceipt struct {
	Kind        PricingKind `json:"kind"`
	RatePercent uint8       `json:"rate_percent"`
}

func profileAction(field string) string {
	if field == profileName {
		return "user.default_contact_name.set"
	}
	return "user.extra_phone.set"
}

func canonicalProfileName(value string) (string, []byte, bool) {
	name := canonicalExtraDisplayName(value)
	key := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, name)
	return name, []byte(key), name != "" && utf8.RuneCountInString(name) <= 100 && len(key) > 0 && len(key) <= 400
}

func canonicalProfileCommand(command ContactProfileCommand) (ContactProfileCommand, []byte, bool) {
	if command.Field == profileName {
		name, key, ok := canonicalProfileName(command.Value)
		command.Value = name
		return command, key, ok && len(name) <= 64
	}
	if command.Field == profilePhone {
		command.Value = strings.TrimSpace(command.Value)
		if len(command.Value) == 11 && command.Value[0] == '1' {
			command.Value = "+86" + command.Value
		}
		return command, nil, canonicalPhone(command.Value)
	}
	return command, nil, false
}

func profileResult(name, phone sql.NullString, pricing PricingProjection) ContactProfileResult {
	result := ContactProfileResult{DefaultContactName: name.String, Pricing: pricing}
	if phone.Valid {
		result.ExtraPhone = &ExtraPhoneProjection{Phone: phone.String}
	}
	return result
}

func (repository *Repository) SetContactProfile(ctx context.Context, meta WriteMeta, command ContactProfileCommand) (ContactProfileResult, error) {
	command, key, ok := canonicalProfileCommand(command)
	if repository == nil || repository.db == nil || ctx == nil || !ok || !validIdentityWriteMeta(meta) {
		return ContactProfileResult{}, ErrInvalidInput
	}
	for attempt := 0; attempt < 2; attempt++ {
		result, err := repository.setContactProfileOnce(ctx, meta, command, key)
		if errors.Is(err, errExtraReceiptExists) {
			return repository.replayContactProfile(ctx, meta, command)
		}
		if retryableTransaction(err) && attempt == 0 {
			continue
		}
		if err == nil || errors.Is(err, ErrPrimaryPhoneRequired) || errors.Is(err, ErrIdempotencyConflict) {
			return result, err
		}
		return ContactProfileResult{}, ErrUnavailable
	}
	return ContactProfileResult{}, ErrUnavailable
}

func (repository *Repository) setContactProfileOnce(ctx context.Context, meta WriteMeta, command ContactProfileCommand, key []byte) (ContactProfileResult, error) {
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return ContactProfileResult{}, err
	}
	defer tx.Rollback()
	var primary, extra, name sql.NullString
	var boundAt, extraAt sql.NullTime
	var nameKey []byte
	var version uint64
	err = tx.QueryRowContext(ctx, `SELECT primary_phone,primary_phone_bound_at,extra_phone,extra_phone_set_at,extra_name,extra_name_key,record_version FROM miniprogram_users WHERE id=$1 FOR UPDATE`, meta.ActorUserID).Scan(&primary, &boundAt, &extra, &extraAt, &name, &nameKey, &version)
	if err != nil {
		return ContactProfileResult{}, err
	}
	if !validPhoneState(primary, boundAt) || version == 0 || extra.Valid != extraAt.Valid || name.Valid != (nameKey != nil) || (extra.Valid && (!primary.Valid || !canonicalPhone(extra.String))) {
		return ContactProfileResult{}, ErrUnavailable
	}
	if name.Valid {
		_, expected, ok := canonicalProfileName(name.String)
		if !ok || !bytes.Equal(expected, nameKey) {
			return ContactProfileResult{}, ErrUnavailable
		}
	}
	if command.Field == profilePhone {
		if !primary.Valid {
			return ContactProfileResult{}, ErrPrimaryPhoneRequired
		}
		extra = sql.NullString{String: command.Value, Valid: true}
	} else {
		name = sql.NullString{String: command.Value, Valid: true}
		nameKey = key
	}
	var rate uint8
	var whitelistVersion uint64
	if err = tx.QueryRowContext(ctx, `SELECT rate_percent,whitelist_version FROM discount_settings WHERE id=1 FOR UPDATE`).Scan(&rate, &whitelistVersion); err != nil || rate < 1 || rate > 100 || whitelistVersion == 0 {
		return ContactProfileResult{}, ErrUnavailable
	}
	pricing := PricingProjection{Kind: PricingVisitor, RatePercent: 100}
	rows, err := tx.QueryContext(ctx, `SELECT name_key,enabled FROM staff_whitelist WHERE phone=$1 OR phone=$2 ORDER BY id FOR UPDATE`, []byte(primary.String), []byte(extra.String))
	if err != nil {
		return ContactProfileResult{}, err
	}
	for rows.Next() {
		var candidateKey []byte
		var enabled bool
		if err = rows.Scan(&candidateKey, &enabled); err != nil || len(candidateKey) == 0 || len(candidateKey) > 400 {
			rows.Close()
			return ContactProfileResult{}, ErrUnavailable
		}
		if primary.Valid && enabled && len(nameKey) > 0 && bytes.Equal(candidateKey, nameKey) {
			pricing = PricingProjection{Kind: PricingStaff, RatePercent: rate}
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return ContactProfileResult{}, err
	}
	if err = rows.Close(); err != nil {
		return ContactProfileResult{}, err
	}
	var updated sql.Result
	if command.Field == profileName {
		updated, err = tx.ExecContext(ctx, `UPDATE miniprogram_users SET extra_name=$1,extra_name_key=$2,record_version=record_version+1 WHERE id=$3 AND record_version=$4`, command.Value, key, meta.ActorUserID, version)
	} else {
		updated, err = tx.ExecContext(ctx, `UPDATE miniprogram_users SET extra_phone=$1,extra_phone_set_at=clock_timestamp(),record_version=record_version+1 WHERE id=$2 AND record_version=$3`, []byte(command.Value), meta.ActorUserID, version)
	}
	if err != nil {
		return ContactProfileResult{}, err
	}
	count, err := updated.RowsAffected()
	if err != nil || count != 1 {
		return ContactProfileResult{}, ErrUnavailable
	}
	if err = appendContactProfileReceipt(ctx, tx, meta, command, pricing); err != nil {
		return ContactProfileResult{}, err
	}
	if err = repository.commit(tx); err != nil {
		return ContactProfileResult{}, err
	}
	return profileResult(name, extra, pricing), nil
}

func (repository *Repository) replayContactProfile(ctx context.Context, meta WriteMeta, command ContactProfileCommand) (ContactProfileResult, error) {
	scope := identityUserScopeHash(meta.ActorUserID)
	operation := sha256.Sum256([]byte(meta.IdempotencyKey))
	var evidence, response []byte
	if err := repository.db.QueryRowContext(ctx, `SELECT before_state_json,response_json FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='USER' AND actor_scope_hash=$1 AND action=$2 AND operation_key_hash=$3 LIMIT 1`, scope[:], profileAction(command.Field), operation[:]).Scan(&evidence, &response); err != nil {
		return ContactProfileResult{}, ErrUnavailable
	}
	want := digestContactProfileCommand(command)
	got, err := decodeExtraPhoneEvidence(evidence)
	if err != nil || subtle.ConstantTimeCompare(got[:], want[:]) != 1 {
		return ContactProfileResult{}, ErrIdempotencyConflict
	}
	var receipt extraPhoneReceipt
	if !decodeExactJSON(response, &receipt) || !validPricing(receipt.Pricing.Kind, receipt.Pricing.RatePercent) {
		return ContactProfileResult{}, ErrUnavailable
	}
	var phone, name sql.NullString
	if err := repository.db.QueryRowContext(ctx, `SELECT extra_phone,extra_name FROM miniprogram_users WHERE id=$1`, meta.ActorUserID).Scan(&phone, &name); err != nil {
		return ContactProfileResult{}, ErrUnavailable
	}
	current := phone
	if command.Field == profileName {
		current = name
	}
	if !current.Valid || current.String != command.Value {
		return ContactProfileResult{}, ErrIdempotencyConflict
	}
	return profileResult(name, phone, PricingProjection{Kind: receipt.Pricing.Kind, RatePercent: receipt.Pricing.RatePercent}), nil
}

func appendContactProfileReceipt(ctx context.Context, tx *sql.Tx, meta WriteMeta, command ContactProfileCommand, pricing PricingProjection) error {
	digest := digestContactProfileCommand(command)
	evidence, err := json.Marshal(extraPhoneEvidence{RequestDigest: hex.EncodeToString(digest[:])})
	if err != nil {
		return err
	}
	response, err := json.Marshal(extraPhoneReceipt{Pricing: pricingReceipt{Kind: pricing.Kind, RatePercent: pricing.RatePercent}})
	if err != nil {
		return err
	}
	scope := identityUserScopeHash(meta.ActorUserID)
	operation := sha256.Sum256([]byte(meta.IdempotencyKey))
	request := sha256.Sum256([]byte(meta.RequestID))
	_, err = tx.ExecContext(ctx, `INSERT INTO action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,action,target_type,target_id,operation_key_hash,request_id_hash,result,reason_code,before_state_json,response_json,occurred_at) VALUES('COMMAND_RECEIPT','USER',$1,$2,$3,'miniprogram_user',$4,$5,$6,'SUCCEEDED','CONTACT_PROFILE_SET',$7,$8,clock_timestamp())`, scope[:], meta.ActorUserID, profileAction(command.Field), meta.ActorUserID, operation[:], request[:], evidence, response)
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" && pgError.ConstraintName == "uq_action_audits_command_receipt" {
		return errExtraReceiptExists
	}
	return err
}

func canonicalExtraDisplayName(name string) string {
	if !utf8.ValidString(name) {
		return ""
	}
	name = strings.TrimSpace(norm.NFKC.String(name))
	for _, character := range name {
		if unicode.IsControl(character) {
			return ""
		}
	}
	return name
}

func maskIdentityPhone(phone string) string {
	local := strings.TrimPrefix(phone, "+86")
	if len(local) < 7 {
		return "***"
	}
	return local[:3] + "****" + local[len(local)-4:]
}

func validIdentityWriteMeta(meta WriteMeta) bool {
	return meta.ActorUserID > 0 && meta.IdempotencyKey != "" && len(meta.IdempotencyKey) <= 128 && strings.TrimSpace(meta.IdempotencyKey) == meta.IdempotencyKey && meta.RequestID != "" && len(meta.RequestID) <= 64 && strings.TrimSpace(meta.RequestID) == meta.RequestID
}

func digestContactProfileCommand(command ContactProfileCommand) [sha256.Size]byte {
	encoded, _ := json.Marshal(command)
	return sha256.Sum256(encoded)
}

func identityUserScopeHash(userID uint64) [sha256.Size]byte {
	var material [13]byte
	copy(material[:5], "USER\x00")
	binary.BigEndian.PutUint64(material[5:], userID)
	return sha256.Sum256(material[:])
}

func decodeExtraPhoneEvidence(raw []byte) ([sha256.Size]byte, error) {
	var evidence extraPhoneEvidence
	if !decodeExactJSON(raw, &evidence) {
		return [sha256.Size]byte{}, ErrUnavailable
	}
	decoded, err := hex.DecodeString(evidence.RequestDigest)
	if err != nil || len(decoded) != sha256.Size || hex.EncodeToString(decoded) != evidence.RequestDigest {
		return [sha256.Size]byte{}, ErrUnavailable
	}
	var digest [sha256.Size]byte
	copy(digest[:], decoded)
	return digest, nil
}

func decodeExactJSON(raw []byte, target any) bool {
	if len(raw) == 0 || len(raw) > 4096 || !json.Valid(raw) {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target) == nil && decoder.Decode(&struct{}{}) == io.EOF
}

func validPricing(kind PricingKind, rate uint8) bool {
	return (kind == PricingVisitor && rate == 100) || (kind == PricingStaff && rate >= 1 && rate <= 100)
}
