package cloudbasepg

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Exercise the real client boundary: result reads must reach the registered RPC,
// not fail locally after a successful import commit.
func TestImportResultRPCReachesDatabase(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != "/v1/rdb/rest/rpc/order_import_result_v1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var params map[string]string
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil || params["p_user"] != "7" || params["p_batch"] != "93" {
			t.Errorf("unexpected parameters: %v %v", params, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"outcome":"SUCCEEDED","response_text":"{\"BatchID\":93,\"Rows\":[]}"}`))
	}))
	defer server.Close()
	c := newClient(server.URL, Credentials{}, server.Client())
	c.cached = accessToken{value: "test-token", usableUntil: time.Now().Add(time.Hour)}
	var out struct {
		Outcome  string `json:"outcome"`
		Response string `json:"response_text"`
	}
	err := c.CallRPC(context.Background(), "order_import_result_v1", map[string]any{"p_user": "7", "p_batch": "93"}, &out)
	if err != nil || calls != 1 || out.Outcome != "SUCCEEDED" {
		t.Fatalf("import result must reach database: calls=%d outcome=%q err=%v", calls, out.Outcome, err)
	}
}
