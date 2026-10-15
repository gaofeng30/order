package compensation

import (
	"context"
	"strconv"
)

type rpcClient interface {
	CallRPC(context.Context, string, map[string]any, any) error
}
type SharedPGStore struct{ client rpcClient }

func NewSharedPGStore(client rpcClient) *SharedPGStore { return &SharedPGStore{client: client} }

func (s *SharedPGStore) Claim(ctx context.Context, token string) (Target, bool, error) {
	var reply struct {
		Outcome string `json:"outcome"`
		Kind    string `json:"kind"`
		ID      string `json:"id"`
	}
	if s == nil || s.client == nil || s.client.CallRPC(ctx, "order_compensation_claim_v1", map[string]any{"p_token": token}, &reply) != nil {
		return Target{}, false, errUnavailable
	}
	if reply.Outcome == "REJECTED" {
		return Target{}, false, nil
	}
	id, err := strconv.ParseUint(reply.ID, 10, 64)
	if reply.Outcome != "CLAIMED" || err != nil || id == 0 || strconv.FormatUint(id, 10) != reply.ID {
		return Target{}, false, errUnavailable
	}
	switch reply.Kind {
	case "payment", "refund", "subscription":
		return Target{Kind: reply.Kind, ID: id}, true, nil
	}
	return Target{}, false, errUnavailable
}

func (s *SharedPGStore) Finish(ctx context.Context, token string, success bool) error {
	var reply struct {
		Outcome string `json:"outcome"`
	}
	if s == nil || s.client == nil || s.client.CallRPC(ctx, "order_compensation_finish_v1", map[string]any{"p_token": token, "p_success": success}, &reply) != nil || reply.Outcome != "FINISHED" {
		return errUnavailable
	}
	return nil
}
