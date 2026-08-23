package config

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

type redemptionSecretSource struct {
	value string
	err   error
	name  string
}

func (source *redemptionSecretSource) Get(_ context.Context, name string) (string, error) {
	source.name = name
	return source.value, source.err
}

func TestLoadRedemptionTokenMaterialAcceptsOnlyCanonicalVersionedKey(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	source := &redemptionSecretSource{value: `{"version":"1","key_base64":"` + base64.StdEncoding.EncodeToString(key) + `"}`}
	material, err := LoadRedemptionTokenMaterial(context.Background(), source)
	if err != nil || source.name != redemptionTokenSecretName || material.Version() != 1 || string(material.Key()) != string(key) {
		t.Fatalf("material = version=%d key=%d name=%q err=%v", material.Version(), len(material.Key()), source.name, err)
	}
	first := material.Key()
	first[0] = 'x'
	if string(material.Key()) != string(key) {
		t.Fatal("Key returned mutable retained material")
	}
}

func TestLoadRedemptionTokenMaterialFailsClosedWithoutSecretLeak(t *testing.T) {
	canary := "redemption-secret-canary"
	values := []string{
		`{"version":"0","key_base64":"` + base64.StdEncoding.EncodeToString(make([]byte, 32)) + `"}`,
		`{"version":"1","key_base64":"short-` + canary + `"}`,
		`{"version":"1","key_base64":"` + base64.StdEncoding.EncodeToString(make([]byte, 32)) + `","extra":"` + canary + `"}`,
		`{"version":"1","key_base64":"` + base64.StdEncoding.EncodeToString(make([]byte, 32)) + `"} {"trailing":"` + canary + `"}`,
	}
	for _, value := range values {
		if _, err := LoadRedemptionTokenMaterial(context.Background(), &redemptionSecretSource{value: value}); err == nil || strings.Contains(err.Error(), canary) {
			t.Fatalf("invalid material error = %v", err)
		}
	}
	providerErr := errors.New("provider " + canary)
	if _, err := LoadRedemptionTokenMaterial(context.Background(), &redemptionSecretSource{err: providerErr}); err == nil || strings.Contains(err.Error(), canary) {
		t.Fatalf("provider error = %v", err)
	}
}
