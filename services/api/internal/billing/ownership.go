package billing

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
	"sort"
	"strconv"
	"unicode/utf8"
)

type ownershipClass string

const (
	ownershipThisEnv     ownershipClass = "THIS_ENV"
	ownershipProvenOther ownershipClass = "PROVEN_OTHER"
	ownershipUnknown     ownershipClass = "UNKNOWN"
)

// classifyNewNumberOwnership evaluates only strict new-number evidence. The
// complete ownership decision must also combine immutable legacy proofs.
func classifyNewNumberOwnership(environment, expectedAppID, expectedMerchantID string, entry wechatpay.BillEntry) (ownershipClass, error) {
	if (environment != "uat" && environment != "production") || expectedAppID != "wx15e136b256d4dc5a" || expectedMerchantID == "" {
		return ownershipUnknown, ErrInvalidInput
	}
	if entry.MerchantID != expectedMerchantID || (entry.Kind != wechatpay.BillEntryPayment && entry.Kind != wechatpay.BillEntryRefund) {
		return ownershipUnknown, ErrBillUnavailable
	}
	if entry.AppID != expectedAppID {
		return ownershipUnknown, nil
	}
	number, kind := entry.OutTradeNo, "payment"
	if entry.Kind == wechatpay.BillEntryRefund {
		number, kind = entry.OutRefundNo, "refund"
	}
	numberEnvironment, numberKind, ok := wechatpay.ParseMerchantNumber(number)
	if !ok || numberKind != kind {
		return ownershipUnknown, nil
	}
	if entry.Kind == wechatpay.BillEntryRefund {
		parentEnvironment, parentKind, parentOK := wechatpay.ParseMerchantNumber(entry.OutTradeNo)
		if parentOK && (parentKind != "payment" || parentEnvironment != numberEnvironment) {
			return ownershipUnknown, nil
		}
	}
	if numberEnvironment == environment {
		return ownershipThisEnv, nil
	}
	return ownershipProvenOther, nil
}

type ownershipIdentity struct {
	Kind                 string `json:"kind"`
	AppIDHex             string `json:"appid_hex"`
	MerchantIDHex        string `json:"mchid_hex"`
	OutTradeNoHex        string `json:"out_trade_no_hex"`
	OutRefundNoHex       string `json:"out_refund_no_hex"`
	ProviderIDHex        string `json:"provider_id_hex"`
	PaymentProviderIDHex string `json:"payment_provider_id_hex"`
}

func ownershipIdentityOf(e ownershipEntry) ownershipIdentity {
	return ownershipIdentity{e.Kind, e.AppIDHex, e.MerchantIDHex, e.OutTradeNoHex, e.OutRefundNoHex, e.ProviderIDHex, e.PaymentProviderIDHex}
}
func validOwnershipHex(value string, empty bool) bool {
	b, err := hex.DecodeString(value)
	return err == nil && hex.EncodeToString(b) == value && utf8.Valid(b) && !bytes.ContainsRune(b, 0) && (empty || len(b) > 0)
}
func decodeOwnershipConfirmation(raw []byte, id uint64, entry ownershipEntry) (ownershipConfirmation, bool, error) {
	var value struct {
		RuleVersion    string            `json:"rule_version"`
		Entry          ownershipIdentity `json:"entry"`
		Environment    string            `json:"environment"`
		SourceRefHex   string            `json:"source_ref_hex"`
		EvidenceSHA256 string            `json:"evidence_sha256"`
		Supersedes     string            `json:"supersedes"`
	}
	if !decodeOwnershipJSON(raw, &value) {
		return ownershipConfirmation{}, false, ErrUnavailable
	}
	parent, ok := canonicalOwnershipUint(value.Supersedes)
	if value.RuleVersion != "1" || (value.Environment != "uat" && value.Environment != "production") || !validOwnershipHex(value.SourceRefHex, false) || len(value.EvidenceSHA256) != 64 || !validDigestHex(value.EvidenceSHA256) || !ok || parent >= id {
		return ownershipConfirmation{}, false, ErrUnavailable
	}
	if value.Entry != ownershipIdentityOf(entry) {
		return ownershipConfirmation{}, false, nil
	}
	return ownershipConfirmation{strconv.FormatUint(id, 10), value.Environment, value.SourceRefHex, value.EvidenceSHA256, value.Supersedes}, true, nil
}
func validDigestHex(value string) bool {
	b, e := hex.DecodeString(value)
	return e == nil && len(b) == 32 && hex.EncodeToString(b) == value
}
func detailOwnershipEntry(e wechatpay.BillEntry) ownershipEntry {
	h := func(s string) string { return hex.EncodeToString([]byte(s)) }
	return ownershipEntry{strconv.FormatUint(e.Ordinal, 10), string(e.Kind), h(e.AppID), h(e.MerchantID), h(e.OutTradeNo), h(e.OutRefundNo), h(e.ProviderID), h(e.PaymentProviderID)}
}
func confirmedOwnership(f ownershipFact) (string, bool) {
	leaves := map[string]string{}
	seen := map[string]bool{}
	for _, c := range f.Confirmations {
		id, ok := canonicalOwnershipUint(c.AuditID)
		parent, parentOK := canonicalOwnershipUint(c.Supersedes)
		if !ok || id == 0 || !parentOK || parent >= id || seen[c.AuditID] {
			return "", false
		}
		if parent != 0 {
			if _, exists := leaves[c.Supersedes]; !exists {
				return "", false
			}
			delete(leaves, c.Supersedes)
		}
		leaves[c.AuditID] = c.Environment
		seen[c.AuditID] = true
	}
	environment := ""
	for _, candidate := range leaves {
		if environment != "" && environment != candidate {
			return "", false
		}
		environment = candidate
	}
	return environment, true
}
func classifyOwnership(config Config, detail wechatpay.BillEntry, fact ownershipFact) (ownershipClass, string) {
	if fact.Payment.State == "CORRUPT" || fact.Payment.State == "CONFLICT" || fact.Refund.State == "CORRUPT" || fact.Refund.State == "CONFLICT" {
		return ownershipUnknown, "INVALID_IMMUTABLE_PROOF"
	}
	current, err := classifyNewNumberOwnership(config.Environment, config.AppID, config.MerchantID, detail)
	if err != nil {
		return ownershipUnknown, "INVALID_IDENTITY"
	}
	environments := map[string]bool{}
	number := detail.OutTradeNo
	kind := "payment"
	if detail.Kind == wechatpay.BillEntryRefund {
		number = detail.OutRefundNo
		kind = "refund"
	}
	env, parsedKind, ok := wechatpay.ParseMerchantNumber(number)
	if ok && parsedKind == kind && detail.AppID == config.AppID {
		if current == ownershipUnknown {
			return ownershipUnknown, "PARENT_CONFLICT"
		}
		environments[env] = true
	}
	if fact.Payment.State == "VALID" && fact.Payment.Environment != "" {
		if detail.Kind == wechatpay.BillEntryPayment {
			environments[fact.Payment.Environment] = true
		} else if fact.Refund.State == "VALID" && fact.Refund.Environment != "" {
			if fact.Refund.Environment != fact.Payment.Environment {
				return ownershipUnknown, "PARENT_CONFLICT"
			}
			environments[fact.Payment.Environment] = true
		} else if ok && env != fact.Payment.Environment {
			return ownershipUnknown, "PARENT_CONFLICT"
		}
	}
	confirmed, valid := confirmedOwnership(fact)
	if !valid {
		return ownershipUnknown, "CONFIRMATION_CONFLICT"
	}
	if confirmed != "" {
		environments[confirmed] = true
	}
	if len(environments) != 1 {
		return ownershipUnknown, "UNPROVEN_OR_CONFLICTING"
	}
	for environment := range environments {
		if environment == config.Environment {
			return ownershipThisEnv, "PROVEN"
		}
		return ownershipProvenOther, "PROVEN"
	}
	return ownershipUnknown, "UNPROVEN"
}

type classifiedBillDetail struct {
	Detail wechatpay.BillEntry `json:"detail"`
	Class  ownershipClass      `json:"class"`
	Basis  string              `json:"basis"`
	Proof  ownershipFact       `json:"proof"`
}
type ownershipSummary struct {
	Class       ownershipClass `json:"class"`
	Environment string         `json:"environment"`
	Kind        string         `json:"kind"`
	Currency    string         `json:"currency"`
	State       string         `json:"state"`
	Count       uint64         `json:"count"`
	AmountCents uint64         `json:"amount_cents"`
}
type ownershipEvidence struct {
	RuleVersion    string                 `json:"rule_version"`
	Environment    string                 `json:"environment"`
	MerchantID     string                 `json:"merchant_id"`
	AppID          string                 `json:"appid"`
	Date           string                 `json:"date"`
	BillDigest     string                 `json:"bill_digest"`
	EvidenceCutoff string                 `json:"evidence_cutoff"`
	EvidenceDigest string                 `json:"evidence_digest"`
	Details        []classifiedBillDetail `json:"details"`
	Summary        []ownershipSummary     `json:"summary"`
	Matched        uint64                 `json:"matched"`
	ProviderOnly   []BillEntry            `json:"provider_only"`
	SystemOnly     []BillEntry            `json:"system_only"`
	SystemFacts    []ownershipFact        `json:"system_facts"`
}

func validateCompleteDetails(config Config, details []wechatpay.BillEntry) bool {
	if details == nil {
		return false
	}
	ordinals := map[uint64]bool{}
	keys := map[string]bool{}
	for _, d := range details {
		if d.Ordinal == 0 || d.Ordinal > uint64(len(details)) || ordinals[d.Ordinal] || d.AppID == "" || d.MerchantID != config.MerchantID || d.OutTradeNo == "" || d.ProviderID == "" || d.PaymentProviderID == "" || d.Currency != "CNY" || d.AmountCents == 0 || d.OccurredAt.IsZero() {
			return false
		}
		key := string(d.Kind) + "\x00" + d.OutTradeNo
		if d.Kind == wechatpay.BillEntryRefund {
			key = string(d.Kind) + "\x00" + d.OutRefundNo
			if d.OutRefundNo == "" || (d.State != "SUCCESS" && d.State != "PROCESSING" && d.State != "FAIL" && d.State != "CHANGE") {
				return false
			}
		} else if d.Kind != wechatpay.BillEntryPayment || d.OutRefundNo != "" || d.State != "SUCCESS" {
			return false
		}
		if keys[key] {
			return false
		}
		keys[key] = true
		ordinals[d.Ordinal] = true
	}
	return true
}
func reconcileOwnership(ctx context.Context, config Config, bill TransactionBill, system []BillEntry, read func(context.Context, []ownershipEntry, uint64) ([]ownershipFact, uint64, error), readSystem func(context.Context, []BillEntry, uint64) ([]ownershipFact, error)) (ReconcileResult, ownershipEvidence, bool, error) {
	evidence := ownershipEvidence{RuleVersion: "1", Environment: config.Environment, MerchantID: config.MerchantID, AppID: config.AppID, Date: bill.Date.Format("2006-01-02"), BillDigest: hex.EncodeToString(bill.Digest[:]), Details: []classifiedBillDetail{}, Summary: []ownershipSummary{}}
	if !validateCompleteDetails(config, bill.Details) {
		return ReconcileResult{}, evidence, false, ErrBillUnavailable
	}
	details := append([]wechatpay.BillEntry(nil), bill.Details...)
	sort.Slice(details, func(i, j int) bool { return details[i].Ordinal < details[j].Ordinal })
	facts := make([]ownershipFact, 0, len(details))
	var cutoff uint64
	for start := 0; start < len(details) || start == 0; start += 100 {
		end := start + 100
		if end > len(details) {
			end = len(details)
		}
		entries := make([]ownershipEntry, 0, end-start)
		for _, d := range details[start:end] {
			entries = append(entries, detailOwnershipEntry(d))
		}
		batch, fixed, err := read(ctx, entries, cutoff)
		if err != nil {
			return ReconcileResult{}, evidence, false, err
		}
		if start > 0 && fixed != cutoff {
			return ReconcileResult{}, evidence, false, ErrUnavailable
		}
		cutoff = fixed
		if len(batch) != len(entries) {
			return ReconcileResult{}, evidence, false, ErrUnavailable
		}
		for i, f := range batch {
			if f.Entry != entries[i] {
				return ReconcileResult{}, evidence, false, ErrUnavailable
			}
		}
		facts = append(facts, batch...)
		if end == len(details) {
			break
		}
	}
	evidence.EvidenceCutoff = strconv.FormatUint(cutoff, 10)
	projected := []BillEntry{}
	unknown := false
	summaries := map[string]ownershipSummary{}
	for i, d := range details {
		class, basis := classifyOwnership(config, d, facts[i])
		evidence.Details = append(evidence.Details, classifiedBillDetail{d, class, basis, facts[i]})
		env := ""
		if class == ownershipThisEnv {
			env = config.Environment
		} else if class == ownershipProvenOther {
			if config.Environment == "uat" {
				env = "production"
			} else {
				env = "uat"
			}
		}
		key := string(class) + "/" + env + "/" + string(d.Kind) + "/" + d.Currency + "/" + d.State
		summary := summaries[key]
		summary.Class = class
		summary.Environment = env
		summary.Kind = string(d.Kind)
		summary.Currency = d.Currency
		summary.State = d.State
		summary.Count++
		if ^uint64(0)-summary.AmountCents < d.AmountCents {
			return ReconcileResult{}, evidence, false, ErrBillUnavailable
		}
		summary.AmountCents += d.AmountCents
		summaries[key] = summary
		if class == ownershipUnknown {
			unknown = true
		}
		if class == ownershipThisEnv && d.State == "SUCCESS" {
			e := BillEntry{Kind: EntryKind(d.Kind), OutTradeNo: d.OutTradeNo, OutRefundNo: d.OutRefundNo, ProviderID: d.ProviderID, AmountCents: d.AmountCents, Currency: d.Currency, State: d.State, OccurredAt: d.OccurredAt}
			if e.Kind == EntryRefund {
				e.OutTradeNo = ""
			}
			projected = append(projected, e)
		}
	}
	keys := make([]string, 0, len(summaries))
	for k := range summaries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		evidence.Summary = append(evidence.Summary, summaries[k])
	}
	comparison := CompareBill(projected, system)
	comparison.BillDate = bill.Date
	comparison.Digest = bill.Digest
	evidence.Matched = comparison.Matched
	evidence.ProviderOnly = append([]BillEntry{}, comparison.ProviderOnly...)
	evidence.SystemOnly = append([]BillEntry{}, comparison.SystemOnly...)
	systemFacts, err := readSystem(ctx, evidence.SystemOnly, cutoff)
	if err != nil {
		return ReconcileResult{}, evidence, false, err
	}
	evidence.SystemFacts = systemFacts
	raw, err := json.Marshal(struct {
		BillFacts   []ownershipFact `json:"bill_facts"`
		SystemFacts []ownershipFact `json:"system_facts"`
	}{facts, evidence.SystemFacts})
	if err != nil {
		return ReconcileResult{}, evidence, false, ErrUnavailable
	}
	sum := sha256.Sum256(raw)
	evidence.EvidenceDigest = hex.EncodeToString(sum[:])
	return comparison, evidence, unknown, nil
}
