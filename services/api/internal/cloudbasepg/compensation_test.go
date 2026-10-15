package cloudbasepg

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCompensationRPCsReachDatabase(t *testing.T) {
	for _, name := range []string{"order_compensation_claim_v1", "order_compensation_finish_v1", "order_subscription_claim_one_v1", "order_compensation_readiness_v1"} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "POST" || r.URL.Path != "/v1/rdb/rest/rpc/"+name {
					t.Error("wrong RPC route")
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"outcome":"REJECTED"}`))
			}))
			defer server.Close()
			client := newClient(server.URL, Credentials{}, server.Client())
			client.cached = accessToken{value: "test-token", usableUntil: time.Now().Add(time.Hour)}
			var out map[string]any
			if err := client.CallRPC(context.Background(), name, map[string]any{}, &out); err != nil || calls != 1 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
		})
	}
}
