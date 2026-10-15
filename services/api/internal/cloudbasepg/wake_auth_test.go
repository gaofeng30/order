package cloudbasepg

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestProductionLoginRegistersWakeCredentialBeforeBusinessRPC(t *testing.T) {
	var registered atomic.Bool
	token := "e30.eyJzdWIiOiIxMjMiLCJyb2xlIjoiYXV0aGVudGljYXRlZCIsImF1ZCI6Im9yZGVyLXByb2QtZDBnNHhudzQ5YzVhZGE1Y2UiLCJleHAiOjQxMDI0NDQ4MDB9.sig"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/v1/signin":
			json.NewEncoder(w).Encode(map[string]any{"token_type": "Bearer", "sub": "123", "access_token": token, "expires_in": 3600})
		case "/v1/rdb/rest/rpc/order_compensation_authenticate_v1":
			if r.Header.Get("Authorization") != "Bearer "+token {
				t.Error("credential changed")
			}
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if len(body) != 0 {
				t.Error("credential leaked into RPC parameters")
			}
			registered.Store(true)
			w.Write([]byte(`{"outcome":"REGISTERED"}`))
		default:
			if !registered.Load() {
				t.Error("business RPC ran before wake authentication")
			}
			w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()
	c := newClient(server.URL, Credentials{Username: "test", Password: "test", ServiceUID: "123", PublishableKey: "e30.eyJhdWQiOiJvcmRlci1wcm9kLWQwZzR4bnc0OWM1YWRhNWNlIiwicm9sZSI6ImFub24ifQ.sig"}, server.Client())
	// Production identity is authoritative even with an isolated HTTP transport.
	var out map[string]any
	if err := c.CallRPC(context.Background(), "order_payment_read_v1", nil, &out); err != nil {
		t.Fatal(err)
	}
	if !registered.Load() {
		t.Fatal("production token was not registered")
	}
}
