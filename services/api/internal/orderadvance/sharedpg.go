package orderadvance

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgconn"
	"strconv"
	"time"
)

type sharedProductionExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}
type SharedService struct{ client sharedProductionExecutor }

func NewSharedPG(client sharedProductionExecutor) *SharedService {
	return &SharedService{client: client}
}

type productionReply struct {
	Outcome  string   `json:"outcome"`
	IDs      []string `json:"ids"`
	Advanced bool     `json:"advanced"`
	SQLState string   `json:"sqlstate"`
}

func (s *SharedService) RunProductionDue(ctx context.Context, now time.Time, limit uint16) (RunResult, error) {
	if ctx == nil || now.IsZero() || limit == 0 || limit > 100 {
		return RunResult{}, ErrInvalidInput
	}
	if s == nil || s.client == nil {
		return RunResult{}, ErrUnavailable
	}
	now = now.UTC().Truncate(time.Microsecond)
	var r productionReply
	if s.client.CallRPC(ctx, "order_production_due_v1", map[string]any{"p_now": productionTimeWire(now, true), "p_limit": limit}, &r) != nil || r.Outcome != "FOUND" {
		return RunResult{}, ErrUnavailable
	}
	ids := make([]uint64, 0, len(r.IDs))
	for _, text := range r.IDs {
		id, e := strconv.ParseUint(text, 10, 64)
		if e != nil || id == 0 || strconv.FormatUint(id, 10) != text {
			return RunResult{}, ErrUnavailable
		}
		ids = append(ids, id)
	}
	result := RunResult{Scanned: uint16(len(ids))}
	for _, id := range ids {
		advanced, e := s.advanceWithRetry(ctx, id, now)
		if e != nil {
			return result, ErrUnavailable
		}
		if advanced {
			result.Advanced++
		}
	}
	return result, nil
}
func (s *SharedService) advanceWithRetry(ctx context.Context, id uint64, now time.Time) (bool, error) {
	for attempt := 0; attempt < 2; attempt++ {
		var r productionReply
		if s.client.CallRPC(ctx, "order_production_advance_v1", map[string]any{"p_order": strconv.FormatUint(id, 10), "p_now": productionTimeWire(now, false)}, &r) != nil {
			return false, ErrUnavailable
		}
		if r.Outcome == "ADVANCED" {
			return r.Advanced, nil
		}
		if r.Outcome != "SQL_FAILURE" || !retryable(&pgconn.PgError{Code: r.SQLState}) || attempt == 1 {
			return false, ErrUnavailable
		}
	}
	return false, ErrUnavailable
}

// The worker owns the original UTC/microsecond clock, cutoff Add and evidence encoding.
func productionTimeValue(at time.Time) map[string]any {
	return map[string]any{"epoch_seconds": strconv.FormatInt(at.Unix(), 10), "nanosecond": at.Nanosecond(), "zero": at.IsZero(), "canonical": at.Format(time.RFC3339Nano)}
}
func productionTimeWire(now time.Time, due bool) map[string]any {
	p := productionTimeValue(now)
	if due {
		p["cutoff"] = productionTimeValue(now.Add(productionLeadTime))
		return p
	}
	after, _ := json.Marshal(struct {
		State       string    `json:"state"`
		PreparingAt time.Time `json:"preparing_at"`
	}{State: "PREPARING", PreparingAt: now})
	if after == nil {
		p["after_json_hex"] = nil
	} else {
		p["after_json_hex"] = hex.EncodeToString(after)
	}
	return p
}
