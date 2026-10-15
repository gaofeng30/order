package billing

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strconv"

	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

type ownershipEntry struct {
	Ordinal              string `json:"ordinal"`
	Kind                 string `json:"kind"`
	AppIDHex             string `json:"appid_hex"`
	MerchantIDHex        string `json:"mchid_hex"`
	OutTradeNoHex        string `json:"out_trade_no_hex"`
	OutRefundNoHex       string `json:"out_refund_no_hex"`
	ProviderIDHex        string `json:"provider_id_hex"`
	PaymentProviderIDHex string `json:"payment_provider_id_hex"`
}
type ownershipProof struct {
	State       string `json:"state"`
	DigestHex   string `json:"digest_hex"`
	Environment string `json:"environment"`
}
type ownershipConfirmation struct {
	AuditID        string `json:"audit_id"`
	Environment    string `json:"environment"`
	SourceRefHex   string `json:"source_ref_hex"`
	EvidenceSHA256 string `json:"evidence_sha256"`
	Supersedes     string `json:"supersedes"`
}
type ownershipFact struct {
	Entry         ownershipEntry          `json:"entry"`
	Payment       ownershipProof          `json:"payment_proof"`
	Refund        ownershipProof          `json:"refund_proof"`
	Confirmations []ownershipConfirmation `json:"confirmations"`
}

// Exact original provider request order is needed to verify the immutable Go
// digest. PostgreSQL JSONB textual order is not the original signed input.
type ownershipPaymentRequest struct {
	AppID       string `json:"appid"`
	MerchantID  string `json:"mchid"`
	Description string `json:"description"`
	OutTradeNo  string `json:"out_trade_no"`
	TimeExpire  string `json:"time_expire"`
	NotifyURL   string `json:"notify_url"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	PayerOpenID string `json:"payer_openid"`
	QuoteID     string `json:"quote_id"`
	QuoteDigest string `json:"quote_digest"`
}
type ownershipRefundRequest struct {
	OutTradeNo    string `json:"out_trade_no"`
	TransactionID string `json:"transaction_id,omitempty"`
	OutRefundNo   string `json:"out_refund_no"`
	Reason        string `json:"reason"`
	NotifyURL     string `json:"notify_url"`
	AmountCents   uint64 `json:"amount_cents"`
	TotalCents    uint64 `json:"total_cents"`
	Currency      string `json:"currency"`
}

func decodeOwnershipJSON(raw []byte, out any) bool {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	return d.Decode(out) == nil && d.Decode(new(any)) == io.EOF
}
func ownershipDigestValid(raw, digest []byte, out any) bool {
	if !decodeOwnershipJSON(raw, out) || len(digest) != sha256.Size {
		return false
	}
	canonical, err := json.Marshal(out)
	if err != nil {
		return false
	}
	sum := sha256.Sum256(canonical)
	return bytes.Equal(sum[:], digest)
}
func ownershipDecode(s string) []byte { b, _ := hex.DecodeString(s); return b }
func (s *Service) nativePaymentProof(ctx context.Context, e ownershipEntry) (ownershipProof, error) {
	return s.nativePaymentProofWith(ctx, s.db.QueryRowContext, e)
}
func (s *Service) nativePaymentProofWith(ctx context.Context, query func(context.Context, string, ...any) *sql.Row, e ownershipEntry) (ownershipProof, error) {
	proof := ownershipProof{State: "ABSENT"}
	var raw, digest, appid, mchid, trade []byte
	var amount int64
	var currency string
	err := query(ctx, `SELECT provider_create_request_json,provider_create_request_digest,expected_appid,expected_mchid,out_trade_no,expected_amount_cents,currency FROM prepayments WHERE out_trade_no=$1`, ownershipDecode(e.OutTradeNoHex)).Scan(&raw, &digest, &appid, &mchid, &trade, &amount, &currency)
	if errors.Is(err, sql.ErrNoRows) {
		return proof, nil
	}
	if err != nil {
		return proof, ErrUnavailable
	}
	proof.State = "CORRUPT"
	proof.DigestHex = hex.EncodeToString(digest)
	var request ownershipPaymentRequest
	if !ownershipDigestValid(raw, digest, &request) || request.OutTradeNo != string(trade) || request.AppID != string(appid) || request.MerchantID != string(mchid) || request.AmountCents != amount || request.Currency != currency {
		return proof, nil
	}
	if !bytes.Equal(appid, ownershipDecode(e.AppIDHex)) || !bytes.Equal(mchid, ownershipDecode(e.MerchantIDHex)) {
		proof.State = "CONFLICT"
		return proof, nil
	}
	proof.State = "VALID"
	if request.AppID == s.config.AppID && request.MerchantID == s.config.MerchantID && request.NotifyURL == s.config.PaymentNotifyURL {
		proof.Environment = s.config.Environment
	}
	return proof, nil
}
func (s *Service) nativeRefundProof(ctx context.Context, e ownershipEntry) (ownershipProof, error) {
	return s.nativeRefundProofWith(ctx, s.db.QueryRowContext, e)
}
func (s *Service) nativeRefundProofWith(ctx context.Context, query func(context.Context, string, ...any) *sql.Row, e ownershipEntry) (ownershipProof, error) {
	proof := ownershipProof{State: "ABSENT"}
	if e.Kind != "REFUND" {
		return proof, nil
	}
	var raw, digest, number, parent, provider []byte
	var amount uint64
	var currency string
	err := query(ctx, `SELECT r.provider_request_json,r.provider_request_digest,r.out_refund_no,p.out_trade_no,coalesce(r.provider_refund_id,''::bytea),r.amount_cents,r.currency FROM refunds r JOIN prepayments p ON p.id=r.prepayment_id WHERE r.out_refund_no=$1`, ownershipDecode(e.OutRefundNoHex)).Scan(&raw, &digest, &number, &parent, &provider, &amount, &currency)
	if errors.Is(err, sql.ErrNoRows) {
		return proof, nil
	}
	if err != nil {
		return proof, ErrUnavailable
	}
	proof.State = "CORRUPT"
	proof.DigestHex = hex.EncodeToString(digest)
	var request ownershipRefundRequest
	if !ownershipDigestValid(raw, digest, &request) || request.OutRefundNo != string(number) || request.OutTradeNo != string(parent) || request.AmountCents != amount || request.TotalCents != amount || request.Currency != currency {
		return proof, nil
	}
	if !bytes.Equal(parent, ownershipDecode(e.OutTradeNoHex)) || (len(provider) > 0 && !bytes.Equal(provider, ownershipDecode(e.ProviderIDHex))) || (request.TransactionID != "" && request.TransactionID != string(ownershipDecode(e.PaymentProviderIDHex))) {
		proof.State = "CONFLICT"
		return proof, nil
	}
	proof.State = "VALID"
	if request.NotifyURL == s.config.RefundNotifyURL {
		proof.Environment = s.config.Environment
	}
	return proof, nil
}

type ownershipQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (s *Service) nativeOwnershipFacts(ctx context.Context, entries []ownershipEntry, cutoff uint64) ([]ownershipFact, uint64, error) {
	return s.nativeOwnershipFactsWith(ctx, s.db, entries, cutoff)
}
func (s *Service) nativeOwnershipFactsWith(ctx context.Context, q ownershipQuerier, entries []ownershipEntry, cutoff uint64) ([]ownershipFact, uint64, error) {
	if len(entries) > 100 {
		return nil, 0, ErrUnavailable
	}
	if cutoff == 0 {
		if err := q.QueryRowContext(ctx, `SELECT coalesce(max(id),0) FROM action_audits WHERE entry_kind='SYSTEM_EVIDENCE' AND actor_kind='SYSTEM' AND action='billing.ownership_confirmed'`).Scan(&cutoff); err != nil {
			return nil, 0, ErrUnavailable
		}
	}
	identities := make([]ownershipIdentity, 0, len(entries))
	positions := make(map[ownershipIdentity]int, len(entries))
	facts := make([]ownershipFact, len(entries))
	for i, entry := range entries {
		identity := ownershipIdentityOf(entry)
		identities = append(identities, identity)
		positions[identity] = i
		fact := ownershipFact{Entry: entry, Confirmations: []ownershipConfirmation{}}
		var err error
		fact.Payment, err = s.nativePaymentProofWith(ctx, q.QueryRowContext, entry)
		if err != nil {
			return nil, 0, err
		}
		fact.Refund, err = s.nativeRefundProofWith(ctx, q.QueryRowContext, entry)
		if err != nil {
			return nil, 0, err
		}
		facts[i] = fact
	}
	// One bounded exact-identity query per batch. Unrelated confirmation payloads
	// are neither ownership evidence nor a reason to reject these entries.
	material, err := json.Marshal(identities)
	if err != nil {
		return nil, 0, ErrUnavailable
	}
	rows, err := q.QueryContext(ctx, `SELECT id,after_state_json FROM action_audits WHERE entry_kind='SYSTEM_EVIDENCE' AND actor_kind='SYSTEM' AND action='billing.ownership_confirmed' AND id<=$1 AND after_state_json->'entry' IN (SELECT value FROM jsonb_array_elements($2::jsonb)) ORDER BY id`, cutoff, material)
	if err != nil {
		return nil, 0, ErrUnavailable
	}
	defer rows.Close()
	for rows.Next() {
		var id uint64
		var raw []byte
		if rows.Scan(&id, &raw) != nil {
			return nil, 0, ErrUnavailable
		}
		var envelope struct {
			Entry ownershipIdentity `json:"entry"`
		}
		if json.Unmarshal(raw, &envelope) != nil {
			return nil, 0, ErrUnavailable
		}
		position, ok := positions[envelope.Entry]
		if !ok {
			return nil, 0, ErrUnavailable
		}
		confirmation, match, err := decodeOwnershipConfirmation(raw, id, facts[position].Entry)
		if err != nil || !match {
			return nil, 0, ErrUnavailable
		}
		facts[position].Confirmations = append(facts[position].Confirmations, confirmation)
	}
	if rows.Err() != nil {
		return nil, 0, ErrUnavailable
	}
	return facts, cutoff, nil
}
func canonicalOwnershipUint(value string) (uint64, bool) {
	n, e := strconv.ParseUint(value, 10, 64)
	return n, e == nil && strconv.FormatUint(n, 10) == value
}

func (s *Service) nativeSystemFacts(ctx context.Context, entries []BillEntry, cutoff uint64) ([]ownershipFact, error) {
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return nil, ErrUnavailable
	}
	defer tx.Rollback()
	return s.nativeSystemFactsWith(ctx, tx, entries, cutoff)
}
func (s *Service) nativeSystemFactsWith(ctx context.Context, q ownershipQuerier, entries []BillEntry, cutoff uint64) ([]ownershipFact, error) {
	result := []ownershipFact{}
	ordered := append([]BillEntry(nil), entries...)
	sortEntries(ordered)
	for _, entry := range ordered {
		e := ownershipEntry{Ordinal: "1", Kind: string(entry.Kind), AppIDHex: hex.EncodeToString([]byte(s.config.AppID)), MerchantIDHex: hex.EncodeToString([]byte(s.config.MerchantID)), OutTradeNoHex: hex.EncodeToString([]byte(entry.OutTradeNo)), OutRefundNoHex: hex.EncodeToString([]byte(entry.OutRefundNo)), ProviderIDHex: hex.EncodeToString([]byte(entry.ProviderID)), PaymentProviderIDHex: hex.EncodeToString([]byte(entry.ProviderID))}
		if entry.Kind == EntryRefund {
			var parent, paymentID, appid, mchid []byte
			err := q.QueryRowContext(ctx, `SELECT p.out_trade_no,convert_to(coalesce(r.provider_request_json->>'transaction_id',''),'UTF8'),p.expected_appid,p.expected_mchid FROM refunds r JOIN prepayments p ON p.id=r.prepayment_id WHERE r.out_refund_no=$1 AND r.materialization_state='APPLIED' AND r.provider_refund_id=$2`, []byte(entry.OutRefundNo), []byte(entry.ProviderID)).Scan(&parent, &paymentID, &appid, &mchid)
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			if err != nil {
				return nil, ErrUnavailable
			}
			if string(appid) != s.config.AppID || string(mchid) != s.config.MerchantID {
				continue
			}
			if env, kind, ok := wechatpay.ParseMerchantNumber(string(parent)); ok && (env != s.config.Environment || kind != "payment") {
				continue
			}
			if env, kind, ok := wechatpay.ParseMerchantNumber(entry.OutRefundNo); ok && (env != s.config.Environment || kind != "refund") {
				continue
			}
			e.OutTradeNoHex = hex.EncodeToString(parent)
			e.PaymentProviderIDHex = hex.EncodeToString(paymentID)
		}
		facts, fixed, err := s.nativeOwnershipFactsWith(ctx, q, []ownershipEntry{e}, cutoff)
		if err != nil || fixed != cutoff || len(facts) != 1 {
			return nil, ErrUnavailable
		}
		result = append(result, facts[0])
	}
	return result, nil
}
func sameOwnershipFacts(a, b []ownershipFact) bool {
	aa, e := json.Marshal(a)
	bb, f := json.Marshal(b)
	return e == nil && f == nil && bytes.Equal(aa, bb)
}
func systemFactMatches(e BillEntry, f ownershipFact) bool {
	if string(e.Kind) != f.Entry.Kind {
		return false
	}
	if e.Kind == EntryPayment {
		return hex.EncodeToString([]byte(e.OutTradeNo)) == f.Entry.OutTradeNoHex
	}
	return hex.EncodeToString([]byte(e.OutRefundNo)) == f.Entry.OutRefundNoHex
}
func systemFactOwned(c Config, f ownershipFact) bool {
	if f.Entry.AppIDHex != hex.EncodeToString([]byte(c.AppID)) || f.Entry.MerchantIDHex != hex.EncodeToString([]byte(c.MerchantID)) || f.Payment.State != "VALID" || f.Payment.Environment != c.Environment {
		return false
	}
	if f.Entry.Kind == "REFUND" && (f.Refund.State != "VALID" || f.Refund.Environment != c.Environment) {
		return false
	}
	for _, v := range []struct{ number, kind string }{{string(ownershipDecode(f.Entry.OutTradeNoHex)), "payment"}, {string(ownershipDecode(f.Entry.OutRefundNoHex)), "refund"}} {
		if env, kind, ok := wechatpay.ParseMerchantNumber(v.number); ok && (env != c.Environment || kind != v.kind) {
			return false
		}
	}
	return true
}
