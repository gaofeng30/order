package httpdto

import (
	"errors"
	"net/http/httptest"
	"testing"
)

func TestIdempotencyKeyAcceptsExactlyOneOpaqueValue(t *testing.T) {
	request := httptest.NewRequest("POST", "/", nil)
	request.Header.Set("Idempotency-Key", "quote-0123456789abcdef")
	got, err := IdempotencyKey(request)
	if err != nil || got != "quote-0123456789abcdef" {
		t.Fatalf("IdempotencyKey() = %q, %v", got, err)
	}
}

func TestIdempotencyKeyRejectsMissingDuplicateOrAmbiguousValues(t *testing.T) {
	for _, test := range []struct {
		name   string
		values []string
	}{
		{name: "missing"},
		{name: "empty", values: []string{""}},
		{name: "surrounding whitespace", values: []string{" key"}},
		{name: "coalesced values", values: []string{"one,two"}},
		{name: "duplicate header", values: []string{"one", "two"}},
		{name: "control", values: []string{"one\x7f"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/", nil)
			for _, value := range test.values {
				request.Header.Add("Idempotency-Key", value)
			}
			if _, err := IdempotencyKey(request); !errors.Is(err, ErrInvalidIdempotencyKey) {
				t.Fatalf("IdempotencyKey() error = %v", err)
			}
		})
	}
}

func TestBearerTokenAcceptsOneExactAuthorizationValue(t *testing.T) {
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set("Authorization", "Bearer opaque-token")
	got, err := BearerToken(request)
	if err != nil || got != "opaque-token" {
		t.Fatalf("BearerToken() = %q, %v", got, err)
	}
}

func TestBearerTokenRejectsMissingDuplicateOrAmbiguousValues(t *testing.T) {
	for _, test := range []struct {
		name   string
		values []string
	}{
		{name: "missing"},
		{name: "empty", values: []string{"Bearer "}},
		{name: "wrong scheme", values: []string{"bearer token"}},
		{name: "spaces", values: []string{"Bearer token extra"}},
		{name: "comma", values: []string{"Bearer one,two"}},
		{name: "duplicate", values: []string{"Bearer one", "Bearer two"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/", nil)
			for _, value := range test.values {
				request.Header.Add("Authorization", value)
			}
			if _, err := BearerToken(request); !errors.Is(err, ErrInvalidBearerToken) {
				t.Fatalf("BearerToken() error = %v", err)
			}
		})
	}
}
