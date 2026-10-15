package audit

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"
)

type sharedAuditExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}

// SharedPGSearcher intentionally exposes Search only, not receipt storage.
type SharedPGSearcher struct{ client sharedAuditExecutor }

func NewSharedPGSearcher(client sharedAuditExecutor) *SharedPGSearcher {
	return &SharedPGSearcher{client: client}
}

var _ Searcher = (*SharedPGSearcher)(nil)

func (s *SharedPGSearcher) Search(ctx context.Context, user uint64, f Filter) ([]Entry, uint64, error) {
	var r struct {
		Outcome string              `json:"outcome"`
		Rows    [][]json.RawMessage `json:"rows"`
	}
	// Before the first owner query, cancellation is its native Forbidden error.
	if ctx.Err() != nil {
		return nil, 0, ErrForbidden
	}
	hx := func(v string) string { return hex.EncodeToString([]byte(v)) }
	if s.client == nil || s.client.CallRPC(ctx, "order_audit_search_v1", map[string]any{"p_owner": strconv.FormatUint(user, 10), "p_action_hex": hx(f.Action), "p_target_hex": hx(f.Target), "p_from_hex": hx(f.From), "p_to_hex": hx(f.To), "p_after": strconv.FormatUint(f.Page.AfterID, 10), "p_limit": f.Page.Limit}, &r) != nil {
		return nil, 0, ErrUnavailable
	}
	switch r.Outcome {
	case "FORBIDDEN":
		return nil, 0, ErrForbidden
	case "FOUND":
	default:
		return nil, 0, ErrUnavailable
	}
	out := []Entry{}
	for _, row := range r.Rows {
		var e Entry
		var result, reason, requestHash string
		if sharedAuditRow(row).Scan(&e.ID, &e.Action, &e.TargetType, &e.TargetID, &result, &reason, &requestHash, &e.ActorAccountID, &e.CreatedAt) != nil {
			return nil, 0, ErrUnavailable
		}
		e.ResultCode = result + ":" + reason
		e.RequestID = maskHash(requestHash)
		out = append(out, e)
	}
	next := uint64(0)
	if len(out) == int(f.Page.Limit) {
		next = out[len(out)-1].ID
	}
	return out, next, nil
}

type sharedAuditRow []json.RawMessage

func (r sharedAuditRow) Scan(dest ...any) error {
	if len(r) != len(dest) {
		return ErrUnavailable
	}
	for i, d := range dest {
		raw := r[i]
		switch v := d.(type) {
		case *string:
			if string(raw) == "null" || json.Unmarshal(raw, v) != nil {
				return ErrUnavailable
			}
		case *uint64:
			var text string
			if json.Unmarshal(raw, &text) != nil {
				return ErrUnavailable
			}
			n, err := strconv.ParseUint(text, 10, 64)
			if err != nil {
				return ErrUnavailable
			}
			*v = n
		case *time.Time:
			var at struct {
				Seconds    string `json:"seconds"`
				Nanosecond int    `json:"nanosecond"`
			}
			if json.Unmarshal(raw, &at) != nil {
				return ErrUnavailable
			}
			seconds, err := strconv.ParseInt(at.Seconds, 10, 64)
			if err != nil || at.Nanosecond < 0 || at.Nanosecond >= 1000000000 {
				return ErrUnavailable
			}
			*v = time.Unix(seconds, int64(at.Nanosecond)).In(time.Local)
		default:
			return ErrUnavailable
		}
	}
	return nil
}
