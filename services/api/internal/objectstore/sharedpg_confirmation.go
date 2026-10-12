package objectstore

import (
	"context"
	"encoding/hex"

	"github.com/gaofeng30/order/services/api/internal/audit"
)

type sharedPGExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}
type SharedPGConfirmationReader struct{ client sharedPGExecutor }

func NewSharedPGConfirmationReader(client sharedPGExecutor) *SharedPGConfirmationReader {
	return &SharedPGConfirmationReader{client}
}
func (r *SharedPGConfirmationReader) Confirmed(ctx context.Context, keys []string) (map[string]bool, error) {
	if r == nil || r.client == nil || ctx == nil || len(keys) == 0 || len(keys) > 3 {
		return nil, ErrUnavailable
	}
	hashes := make([]string, 0, len(keys))
	keyByHash := make(map[string]string, len(keys))
	for _, key := range keys {
		if !ValidImageObjectKey(key) {
			return nil, ErrUnavailable
		}
		hash := audit.TargetKeyHash(key)
		raw := hex.EncodeToString(hash[:])
		hashes = append(hashes, raw)
		keyByHash[raw] = key
	}
	var facts struct {
		Outcome string   `json:"outcome"`
		Hashes  []string `json:"target_key_hashes"`
	}
	if r.client.CallRPC(ctx, "order_image_confirmed_v1", map[string]any{"p_target_key_hashes": hashes}, &facts) != nil || facts.Outcome != "SUCCEEDED" || facts.Hashes == nil || len(facts.Hashes) > 3 {
		return nil, ErrUnavailable
	}
	result := make(map[string]bool, len(keys))
	for _, hash := range facts.Hashes {
		decoded, err := hex.DecodeString(hash)
		key, ok := keyByHash[hash]
		if err != nil || len(decoded) != 32 || !ok || hex.EncodeToString(decoded) != hash {
			return nil, ErrUnavailable
		}
		result[key] = true
	}
	return result, nil
}
