package compensation

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type rpcStub struct {
	reply string
	err   error
	name  string
	args  map[string]any
}

func (s *rpcStub) CallRPC(_ context.Context, name string, args map[string]any, out any) error {
	s.name, s.args = name, args
	if s.err != nil {
		return s.err
	}
	return json.Unmarshal([]byte(s.reply), out)
}
func TestSharedPGClaimValidatesCapabilityTarget(t *testing.T) {
	for _, tc := range []struct {
		reply          string
		accepted, fail bool
	}{
		{`{"outcome":"CLAIMED","kind":"payment","id":"42"}`, true, false},
		{`{"outcome":"REJECTED"}`, false, false},
		{`{"outcome":"CLAIMED","kind":"other","id":"42"}`, false, true},
		{`{"outcome":"CLAIMED","kind":"payment","id":"0"}`, false, true},
		{`{"outcome":"CLAIMED","kind":"payment","id":"042"}`, false, true},
		{`{"outcome":"CLAIMED","kind":"payment","id":"18446744073709551616"}`, false, true},
		{`{"outcome":"UNKNOWN"}`, false, true},
	} {
		rpc := &rpcStub{reply: tc.reply}
		store := NewSharedPGStore(rpc)
		target, accepted, err := store.Claim(context.Background(), "token")
		if accepted != tc.accepted || (err != nil) != tc.fail {
			t.Fatalf("reply=%s accepted=%v err=%v", tc.reply, accepted, err)
		}
		if accepted && (target.Kind != "payment" || target.ID != 42) {
			t.Fatal(target)
		}
		if rpc.name != "order_compensation_claim_v1" || rpc.args["p_token"] != "token" || len(rpc.args) != 1 {
			t.Fatal("wrong claim request")
		}
	}
}
func TestSharedPGFinishRequiresConfirmedReceipt(t *testing.T) {
	for _, reply := range []string{`{"outcome":"FINISHED"}`, `{"outcome":"REJECTED"}`, `{}`} {
		rpc := &rpcStub{reply: reply}
		err := NewSharedPGStore(rpc).Finish(context.Background(), "token", false)
		if (err == nil) != (reply == `{"outcome":"FINISHED"}`) {
			t.Fatalf("reply=%s err=%v", reply, err)
		}
		if rpc.name != "order_compensation_finish_v1" || rpc.args["p_token"] != "token" || rpc.args["p_success"] != false || len(rpc.args) != 2 {
			t.Fatal("wrong finish request")
		}
	}
	rpc := &rpcStub{err: errors.New("private provider diagnostic")}
	if _, _, err := NewSharedPGStore(rpc).Claim(context.Background(), "token"); err != errUnavailable {
		t.Fatal("raw diagnostic escaped")
	}
}
