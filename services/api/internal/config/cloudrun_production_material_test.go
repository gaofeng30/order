package config

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type productionSTSTransport func(*http.Request) (*http.Response, error)

func (f productionSTSTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestProductionCOSRestrictedPolicy(t *testing.T) {
	material, err := ParseCOSMaterial(productionCOSBucket, "ap-shanghai")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	transport := productionSTSTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var request struct {
			Policy          string
			Name            string
			DurationSeconds int
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		policy, err := url.QueryUnescape(request.Policy)
		if err != nil {
			t.Fatal(err)
		}
		var decoded struct {
			Statement []struct {
				Resource []string
				Action   []string
				Effect   string
			}
		}
		if err := json.Unmarshal([]byte(policy), &decoded); err != nil {
			t.Fatal(err)
		}
		if len(decoded.Statement) != 1 {
			t.Fatal("policy statement count")
		}
		st := decoded.Statement[0]
		want := "qcs::cos:ap-shanghai:uid/1473835269:6f72-order-prod-d0g4xnw49c5ada5ce-1473835269/images/*"
		if len(st.Resource) != 1 || st.Resource[0] != want || strings.Contains(policy, "order-uat") || st.Effect != "allow" || len(st.Action) != 2 || st.Action[0] != "name/cos:PutObject" || st.Action[1] != "name/cos:GetObject" {
			t.Fatal("policy not restricted to production image prefix")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"Response":{"Credentials":{"TmpSecretId":"temporary-id","TmpSecretKey":"temporary-key","Token":"temporary-token"},"ExpiredTime":2000000000}}`))}, nil
	})
	provider, err := newCloudRunCOSCredentialProvider(material, "isolated-id", "isolated-key", transport, func() time.Time { return time.Unix(1900000000, 0) })
	if err != nil {
		t.Fatal(err)
	}
	if _, err = provider.GetCredential(); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("missing STS call")
	}
}

func TestProductionCloudRunSecretScope(t *testing.T) {
	t.Setenv("ORDER_ENV", "production")
	t.Setenv("ORDER_REDEMPTION_TOKEN_KEY", "isolated-key")
	source := cloudRunSecretSource{}
	if v, err := source.Get(context.Background(), "order-production-"+redemptionTokenPurpose); err != nil || v != "isolated-key" {
		t.Fatal("production material unavailable")
	}
	if _, err := source.Get(context.Background(), "order-uat-"+redemptionTokenPurpose); err == nil {
		t.Fatal("cross-environment secret accepted")
	}
}
