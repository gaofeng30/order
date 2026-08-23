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

	mysqlDriver "github.com/go-sql-driver/mysql"
	"golang.org/x/text/unicode/norm"
)

const extraPhoneAction = "user.extra_phone.set"

var errExtraReceiptExists = errors.New("extra phone receipt exists")

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

func (repository *Repository) SetExtraPhone(ctx context.Context, meta WriteMeta, command ExtraPhoneCommand) (ExtraPhoneResult, error) {
	phone, nameKey, ok := canonicalExtraIdentity(command.Phone, command.Name)
	name := canonicalExtraDisplayName(command.Name)
	if repository == nil || repository.db == nil || ctx == nil || !ok || !validIdentityWriteMeta(meta) {
		return ExtraPhoneResult{}, ErrInvalidInput
	}
	command = ExtraPhoneCommand{Phone: phone, Name: name}
	for attempt := 0; attempt < 2; attempt++ {
		result, err := repository.setExtraPhoneOnce(ctx, meta, command, nameKey)
		if errors.Is(err, errExtraReceiptExists) {
			return repository.replayExtraPhone(ctx, meta, command)
		}
		if retryableTransaction(err) && attempt == 0 {
			continue
		}
		if err == nil || errors.Is(err, ErrPrimaryPhoneRequired) || errors.Is(err, ErrIdempotencyConflict) {
			return result, err
		}
		return ExtraPhoneResult{}, ErrUnavailable
	}
	return ExtraPhoneResult{}, ErrUnavailable
}

func (repository *Repository) setExtraPhoneOnce(ctx context.Context, meta WriteMeta, command ExtraPhoneCommand, nameKey []byte) (ExtraPhoneResult, error) {
	transaction, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return ExtraPhoneResult{}, err
	}
	defer transaction.Rollback()

	var primaryPhone sql.NullString
	var primaryBoundAt sql.NullTime
	var recordVersion uint64
	if err := transaction.QueryRowContext(ctx, `SELECT CONVERT(primary_phone USING ascii),primary_phone_bound_at,record_version FROM miniprogram_users WHERE id=? FOR UPDATE`, meta.ActorUserID).Scan(&primaryPhone, &primaryBoundAt, &recordVersion); err != nil {
		return ExtraPhoneResult{}, err
	}
	if !validPhoneState(primaryPhone, primaryBoundAt) || recordVersion == 0 {
		return ExtraPhoneResult{}, ErrUnavailable
	}
	if !primaryPhone.Valid {
		return ExtraPhoneResult{}, ErrPrimaryPhoneRequired
	}

	var ratePercent uint8
	var whitelistVersion uint64
	if err := transaction.QueryRowContext(ctx, `SELECT rate_percent,whitelist_version FROM discount_settings WHERE id=1 FOR UPDATE`).Scan(&ratePercent, &whitelistVersion); err != nil || ratePercent < 1 || ratePercent > 100 || whitelistVersion == 0 {
		return ExtraPhoneResult{}, ErrUnavailable
	}

	type whitelistFact struct {
		nameKey []byte
		enabled bool
	}
	facts := make(map[string]whitelistFact, 2)
	rows, err := transaction.QueryContext(ctx, `SELECT CONVERT(phone USING ascii),name_key,enabled FROM staff_whitelist WHERE phone=? OR phone=? ORDER BY id FOR UPDATE`, primaryPhone.String, command.Phone)
	if err != nil {
		return ExtraPhoneResult{}, err
	}
	for rows.Next() {
		var candidatePhone string
		var candidate whitelistFact
		if err := rows.Scan(&candidatePhone, &candidate.nameKey, &candidate.enabled); err != nil || !canonicalPhone(candidatePhone) || len(candidate.nameKey) == 0 || len(candidate.nameKey) > 400 {
			rows.Close()
			return ExtraPhoneResult{}, ErrUnavailable
		}
		if _, duplicate := facts[candidatePhone]; duplicate {
			rows.Close()
			return ExtraPhoneResult{}, ErrUnavailable
		}
		facts[candidatePhone] = candidate
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return ExtraPhoneResult{}, err
	}
	if err := rows.Close(); err != nil {
		return ExtraPhoneResult{}, err
	}

	pricing := PricingProjection{Kind: PricingVisitor, RatePercent: 100}
	if fact, found := facts[primaryPhone.String]; found && fact.enabled {
		pricing = PricingProjection{Kind: PricingStaff, RatePercent: ratePercent}
	} else if fact, found := facts[command.Phone]; found && fact.enabled && bytes.Equal(fact.nameKey, nameKey) {
		pricing = PricingProjection{Kind: PricingStaff, RatePercent: ratePercent}
	}
	updated, err := transaction.ExecContext(ctx, `UPDATE miniprogram_users SET extra_phone=?,extra_name=?,extra_name_key=?,extra_phone_set_at=UTC_TIMESTAMP(6),record_version=record_version+1 WHERE id=? AND record_version=?`, command.Phone, command.Name, nameKey, meta.ActorUserID, recordVersion)
	if err != nil {
		return ExtraPhoneResult{}, err
	}
	rowsAffected, err := updated.RowsAffected()
	if err != nil || rowsAffected != 1 {
		if err == nil {
			err = ErrUnavailable
		}
		return ExtraPhoneResult{}, err
	}

	result := ExtraPhoneResult{
		ExtraPhone: ExtraPhoneProjection{MaskedPhone: maskIdentityPhone(command.Phone), Name: command.Name},
		Pricing:    pricing,
	}
	if err := appendExtraPhoneReceipt(ctx, transaction, meta, command, pricing); err != nil {
		return ExtraPhoneResult{}, err
	}
	if err := repository.commit(transaction); err != nil {
		return ExtraPhoneResult{}, err
	}
	return result, nil
}

func (repository *Repository) replayExtraPhone(ctx context.Context, meta WriteMeta, command ExtraPhoneCommand) (ExtraPhoneResult, error) {
	scopeHash := identityUserScopeHash(meta.ActorUserID)
	operationHash := sha256.Sum256([]byte(meta.IdempotencyKey))
	var evidenceRaw, responseRaw []byte
	if err := repository.db.QueryRowContext(ctx, `SELECT before_state_json,response_json FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='USER' AND actor_scope_hash=? AND action=? AND operation_key_hash=? LIMIT 1`, scopeHash[:], extraPhoneAction, operationHash[:]).Scan(&evidenceRaw, &responseRaw); err != nil {
		return ExtraPhoneResult{}, ErrUnavailable
	}
	wantDigest := digestExtraPhoneCommand(command)
	gotDigest, err := decodeExtraPhoneEvidence(evidenceRaw)
	if err != nil || subtle.ConstantTimeCompare(gotDigest[:], wantDigest[:]) != 1 {
		return ExtraPhoneResult{}, ErrIdempotencyConflict
	}
	var receipt extraPhoneReceipt
	if !decodeExactJSON(responseRaw, &receipt) || !validPricing(receipt.Pricing.Kind, receipt.Pricing.RatePercent) {
		return ExtraPhoneResult{}, ErrUnavailable
	}
	var currentPhone, currentName sql.NullString
	if err := repository.db.QueryRowContext(ctx, `SELECT CONVERT(extra_phone USING ascii),extra_name FROM miniprogram_users WHERE id=?`, meta.ActorUserID).Scan(&currentPhone, &currentName); err != nil {
		return ExtraPhoneResult{}, ErrUnavailable
	}
	if !currentPhone.Valid || !currentName.Valid || currentPhone.String != command.Phone || currentName.String != command.Name {
		return ExtraPhoneResult{}, ErrIdempotencyConflict
	}
	return ExtraPhoneResult{
		ExtraPhone: ExtraPhoneProjection{MaskedPhone: maskIdentityPhone(command.Phone), Name: command.Name},
		Pricing:    PricingProjection{Kind: receipt.Pricing.Kind, RatePercent: receipt.Pricing.RatePercent},
	}, nil
}

func appendExtraPhoneReceipt(ctx context.Context, transaction *sql.Tx, meta WriteMeta, command ExtraPhoneCommand, pricing PricingProjection) error {
	digest := digestExtraPhoneCommand(command)
	evidenceRaw, err := json.Marshal(extraPhoneEvidence{RequestDigest: hex.EncodeToString(digest[:])})
	if err != nil {
		return err
	}
	responseRaw, err := json.Marshal(extraPhoneReceipt{Pricing: pricingReceipt{Kind: pricing.Kind, RatePercent: pricing.RatePercent}})
	if err != nil {
		return err
	}
	scopeHash := identityUserScopeHash(meta.ActorUserID)
	operationHash := sha256.Sum256([]byte(meta.IdempotencyKey))
	requestIDHash := sha256.Sum256([]byte(meta.RequestID))
	_, err = transaction.ExecContext(ctx, `INSERT INTO action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,action,target_type,target_id,operation_key_hash,request_id_hash,result,reason_code,before_state_json,response_json,occurred_at) VALUES('COMMAND_RECEIPT','USER',?,?,?,'miniprogram_user',?,?,?,'SUCCEEDED','EXTRA_PHONE_SET',?,?,UTC_TIMESTAMP(6))`, scopeHash[:], meta.ActorUserID, extraPhoneAction, meta.ActorUserID, operationHash[:], requestIDHash[:], evidenceRaw, responseRaw)
	var mysqlError *mysqlDriver.MySQLError
	if errors.As(err, &mysqlError) && mysqlError.Number == 1062 {
		return errExtraReceiptExists
	}
	return err
}

func canonicalExtraIdentity(phone, name string) (string, []byte, bool) {
	phone = strings.TrimSpace(phone)
	if len(phone) == 11 && phone[0] == '1' {
		phone = "+86" + phone
	}
	name = canonicalExtraDisplayName(name)
	if !canonicalPhone(phone) || name == "" || utf8.RuneCountInString(name) > 100 {
		return "", nil, false
	}
	key := strings.Map(func(character rune) rune {
		if unicode.IsSpace(character) {
			return -1
		}
		return character
	}, name)
	if key == "" || len([]byte(key)) > 400 {
		return "", nil, false
	}
	return phone, []byte(key), true
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

func digestExtraPhoneCommand(command ExtraPhoneCommand) [sha256.Size]byte {
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
