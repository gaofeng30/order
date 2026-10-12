package billing

import (
	"context"
	"encoding/hex"
	"encoding/json"
)

func (s *SharedService) sharedOwnershipFacts(ctx context.Context, entries []ownershipEntry, cutoff uint64) ([]ownershipFact, uint64, error) {
	v, err := s.call(ctx, "order_billing_ownership_facts_v1", map[string]any{"p_environment": s.config.Environment, "p_entries": entries, "p_evidence_cutoff": billingDecimal(cutoff)}, "evidence_cutoff", "entries")
	if err != nil {
		return nil, 0, err
	}
	fixed, err := billingUint(v["evidence_cutoff"])
	if err != nil {
		return nil, 0, err
	}
	var rows []json.RawMessage
	if json.Unmarshal(v["entries"], &rows) != nil || rows == nil || len(rows) != len(entries) {
		return nil, 0, ErrUnavailable
	}
	facts := make([]ownershipFact, 0, len(rows))
	for i, raw := range rows {
		fields, err := billingObject(raw, "entry", "payment_proof", "refund_proof", "confirmations")
		if err != nil {
			return nil, 0, err
		}
		if _, err = billingObject(fields["entry"], "ordinal", "kind", "appid_hex", "mchid_hex", "out_trade_no_hex", "out_refund_no_hex", "provider_id_hex", "payment_provider_id_hex"); err != nil {
			return nil, 0, err
		}
		var fact ownershipFact
		if !decodeOwnershipJSON(raw, &fact) || fact.Entry != entries[i] || fact.Confirmations == nil {
			return nil, 0, ErrUnavailable
		}
		for _, name := range []string{"payment_proof", "refund_proof"} {
			if _, err = billingObject(fields[name], "state", "digest_hex", "environment"); err != nil {
				return nil, 0, err
			}
		}
		for _, proof := range []ownershipProof{fact.Payment, fact.Refund} {
			if proof.State != "ABSENT" && proof.State != "VALID" && proof.State != "CORRUPT" && proof.State != "CONFLICT" {
				return nil, 0, ErrUnavailable
			}
			if proof.Environment != "" && proof.Environment != "uat" && proof.Environment != "production" {
				return nil, 0, ErrUnavailable
			}
			if proof.State == "ABSENT" {
				if proof.DigestHex != "" || proof.Environment != "" {
					return nil, 0, ErrUnavailable
				}
			} else if !validDigestHex(proof.DigestHex) {
				return nil, 0, ErrUnavailable
			}
		}
		var confirmations []json.RawMessage
		if json.Unmarshal(fields["confirmations"], &confirmations) != nil {
			return nil, 0, ErrUnavailable
		}
		var last uint64
		for j, c := range fact.Confirmations {
			if _, err = billingObject(confirmations[j], "audit_id", "environment", "source_ref_hex", "evidence_sha256", "supersedes"); err != nil {
				return nil, 0, err
			}
			id, ok := canonicalOwnershipUint(c.AuditID)
			parent, parentOK := canonicalOwnershipUint(c.Supersedes)
			if !ok || !parentOK || id <= last || id > fixed || parent >= id || (c.Environment != "uat" && c.Environment != "production") || !validOwnershipHex(c.SourceRefHex, false) || !validDigestHex(c.EvidenceSHA256) {
				return nil, 0, ErrUnavailable
			}
			last = id
		}
		facts = append(facts, fact)
	}
	return facts, fixed, nil
}

func (s *SharedService) sharedSystemFacts(ctx context.Context, entries []BillEntry, cutoff uint64) ([]ownershipFact, error) {
	result := []ownershipFact{}
	ordered := append([]BillEntry(nil), entries...)
	sortEntries(ordered)
	h := func(s string) string { return hex.EncodeToString([]byte(s)) }
	for _, entry := range ordered {
		e := ownershipEntry{Ordinal: "1", Kind: string(entry.Kind), AppIDHex: h(s.config.AppID), MerchantIDHex: h(s.config.MerchantID), OutTradeNoHex: h(entry.OutTradeNo), OutRefundNoHex: h(entry.OutRefundNo), ProviderIDHex: h(entry.ProviderID), PaymentProviderIDHex: h(entry.ProviderID)}
		if entry.Kind == EntryRefund {
			e.PaymentProviderIDHex = ""
			var raw json.RawMessage
			if s.client.CallRPC(ctx, "order_billing_ownership_facts_v1", map[string]any{"p_environment": s.config.Environment, "p_entries": []ownershipEntry{e}, "p_evidence_cutoff": billingDecimal(cutoff)}, &raw) != nil {
				return nil, ErrUnavailable
			}
			var hint struct {
				Outcome string `json:"outcome"`
			}
			if json.Unmarshal(raw, &hint) != nil {
				return nil, ErrUnavailable
			}
			if hint.Outcome == "UNAVAILABLE" {
				if _, err := billingObject(raw, "outcome"); err != nil {
					return nil, err
				}
				continue
			}
			reply, err := billingObject(raw, "outcome", "evidence_cutoff", "entries")
			if err != nil || hint.Outcome != "FOUND" {
				return nil, ErrUnavailable
			}
			fixed, err := billingUint(reply["evidence_cutoff"])
			if err != nil || fixed != cutoff {
				return nil, ErrUnavailable
			}
			var rows []json.RawMessage
			if json.Unmarshal(reply["entries"], &rows) != nil || len(rows) != 1 {
				return nil, ErrUnavailable
			}
			fields, err := billingObject(rows[0], "entry", "payment_proof", "refund_proof", "confirmations", "resolved_parent_hex", "resolved_payment_provider_id_hex")
			if err != nil {
				return nil, err
			}
			var echo ownershipEntry
			if !decodeOwnershipJSON(fields["entry"], &echo) || echo != e {
				return nil, ErrUnavailable
			}
			parent, err := billingBytes(fields["resolved_parent_hex"])
			if err != nil || parent == "" {
				return nil, ErrUnavailable
			}
			provider, err := billingBytes(fields["resolved_payment_provider_id_hex"])
			if err != nil {
				return nil, ErrUnavailable
			}
			e.OutTradeNoHex = h(parent)
			e.PaymentProviderIDHex = h(provider)
		}
		facts, fixed, err := s.sharedOwnershipFacts(ctx, []ownershipEntry{e}, cutoff)
		if err != nil || fixed != cutoff || len(facts) != 1 {
			return nil, ErrUnavailable
		}
		result = append(result, facts[0])
	}
	return result, nil
}
