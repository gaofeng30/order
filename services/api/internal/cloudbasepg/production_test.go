package cloudbasepg

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProductionEnvironmentBinding(t *testing.T) {
	const prod = "order-prod-d0g4xnw49c5ada5ce"
	key := func(env string, role string) string {
		return "e30." + base64.RawURLEncoding.EncodeToString([]byte(`{"aud":"`+env+`","role":"`+role+`","project_id":"`+env+`"}`)) + ".signature"
	}
	for _, tc := range []struct {
		name, env, audience, role string
		allowed                   bool
	}{
		{"production ordinary", prod, prod, "anon", true},
		{"uat ordinary", uatEnvironment, uatEnvironment, "anon", true},
		{"uat key in production", prod, uatEnvironment, "anon", false},
		{"production key in uat", uatEnvironment, prod, "anon", false},
		{"management key", prod, prod, "service_role", false},
		{"unknown environment", "order-prod-other", "order-prod-other", "anon", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, err := New(tc.env, Credentials{Username: "isolated-service", Password: "isolated-password", PublishableKey: key(tc.audience, tc.role), ServiceUID: "123"})
			if (err == nil) != tc.allowed {
				t.Fatalf("allowed=%v, error=%v", tc.allowed, err)
			}
			if tc.allowed && client.origin != "https://"+tc.env+".api.tcloudbasegateway.com" {
				t.Fatal("wrong origin")
			}
		})
	}
}

func TestSignInRejectsCrossEnvironmentAccessToken(t *testing.T) {
	token := func(v map[string]any) string {
		b, _ := json.Marshal(v)
		return "e30." + base64.RawURLEncoding.EncodeToString(b) + ".signature"
	}
	for _, tc := range []struct {
		name, audience, project string
		allowed                 bool
	}{
		{"same environment", productionEnvironment, productionEnvironment, true},
		{"uat audience", uatEnvironment, productionEnvironment, false},
		{"uat project", productionEnvironment, uatEnvironment, false},
		{"missing audience", "", productionEnvironment, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			access := token(map[string]any{"sub": "123", "role": "authenticated", "exp": time.Now().Add(time.Hour).Unix(), "aud": tc.audience, "project_id": tc.project})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/auth/v1/signin" {
					t.Errorf("unexpected path %s", r.URL.Path)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"token_type": "Bearer", "access_token": access, "expires_in": 3600, "sub": "123"})
			}))
			defer server.Close()
			c := newClient(server.URL, Credentials{Username: "test", Password: "test", ServiceUID: "123", PublishableKey: token(map[string]any{"role": "anon", "aud": productionEnvironment, "project_id": productionEnvironment})}, server.Client())
			_, err := c.signIn(context.Background())
			if (err == nil) != tc.allowed {
				t.Fatalf("allowed=%v err=%v", tc.allowed, err)
			}
		})
	}
}
