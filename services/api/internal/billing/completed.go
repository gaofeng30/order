package billing

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"strconv"
	"time"
)

type completedBill struct {
	Outcome   string `json:"outcome"`
	Completed bool   `json:"completed"`
	Matched   string `json:"matched"`
	Digest    string `json:"digest"`
}

func (b completedBill) result(date time.Time) (ReconcileResult, bool, error) {
	if b.Outcome != "SUCCEEDED" {
		return ReconcileResult{}, false, ErrUnavailable
	}
	if !b.Completed {
		return ReconcileResult{}, false, nil
	}
	matched, err := strconv.ParseUint(b.Matched, 10, 64)
	digest, e := hex.DecodeString(b.Digest)
	if err != nil || e != nil || len(digest) != 32 {
		return ReconcileResult{}, false, ErrUnavailable
	}
	r := ReconcileResult{BillDate: date, Matched: matched}
	copy(r.Digest[:], digest)
	return r, true, nil
}
func (s *SharedService) completed(ctx context.Context, date time.Time) (ReconcileResult, bool, error) {
	var b completedBill
	if s.client.CallRPC(ctx, "order_billing_completed_v1", map[string]any{"p_date": date.Format("2006-01-02"), "p_environment": s.config.Environment, "p_appid": s.config.AppID, "p_merchant": s.config.MerchantID}, &b) != nil {
		return ReconcileResult{}, false, ErrUnavailable
	}
	return b.result(date)
}
func (s *Service) completed(ctx context.Context, date time.Time) (ReconcileResult, bool, error) {
	var b completedBill
	b.Outcome = "SUCCEEDED"
	err := s.db.QueryRowContext(ctx, `SELECT result='SUCCEEDED' AND reason_code='BILL_MATCHED',after_state_json->>'matched',COALESCE(after_state_json->>'bill_digest',after_state_json->>'digest') FROM action_audits WHERE entry_kind='SYSTEM_EVIDENCE' AND actor_kind='PROVIDER' AND action='billing.reconcile' AND after_state_json->>'date'=$1 AND (($2 IN ('test','development') AND after_state_json->>'environment' IS NULL) OR (after_state_json->>'environment'=$2 AND after_state_json->>'appid'=$3 AND after_state_json->>'merchant_id'=$4)) ORDER BY id DESC LIMIT 1`, date.Format("2006-01-02"), s.config.Environment, s.config.AppID, s.config.MerchantID).Scan(&b.Completed, &b.Matched, &b.Digest)
	if errors.Is(err, sql.ErrNoRows) {
		return ReconcileResult{}, false, nil
	}
	if err != nil {
		return ReconcileResult{}, false, ErrUnavailable
	}
	return b.result(date)
}
