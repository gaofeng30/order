package httpdto

import (
	"errors"
	"strings"
	"testing"
)

type decodeFixture struct {
	Name string `json:"name"`
	ID   ID     `json:"id"`
}

func TestDecodeStrictAcceptsOneCanonicalDocument(t *testing.T) {
	var got decodeFixture
	if err := DecodeStrict(strings.NewReader(`{"name":"meal","id":"42"}`), 128, &got); err != nil {
		t.Fatalf("DecodeStrict() error = %v", err)
	}
	if got.Name != "meal" || got.ID != 42 {
		t.Fatalf("decoded = %#v", got)
	}
}

func TestDecodeStrictRejectsAmbiguousOrUnsafeJSON(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want error
	}{
		{name: "empty", body: "", want: ErrInvalidJSON},
		{name: "unknown field", body: `{"name":"x","id":"1","extra":true}`, want: ErrInvalidJSON},
		{name: "duplicate top level", body: `{"name":"x","name":"y","id":"1"}`, want: ErrInvalidJSON},
		{name: "duplicate nested", body: `{"name":{"x":1,"x":2},"id":"1"}`, want: ErrInvalidJSON},
		{name: "trailing document", body: `{"name":"x","id":"1"}{}`, want: ErrInvalidJSON},
		{name: "numeric id", body: `{"name":"x","id":1}`, want: ErrInvalidJSON},
		{name: "zero id", body: `{"name":"x","id":"0"}`, want: ErrInvalidJSON},
		{name: "non canonical id", body: `{"name":"x","id":"01"}`, want: ErrInvalidJSON},
	} {
		t.Run(test.name, func(t *testing.T) {
			var target decodeFixture
			if err := DecodeStrict(strings.NewReader(test.body), 256, &target); !errors.Is(err, test.want) {
				t.Fatalf("DecodeStrict() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestDecodeStrictRejectsInvalidUTF8AndOversize(t *testing.T) {
	var target decodeFixture
	if err := DecodeStrict(strings.NewReader(string([]byte{'{', '"', 0xff, '"', ':', '1', '}'})), 64, &target); !errors.Is(err, ErrInvalidJSON) {
		t.Fatalf("invalid UTF-8 error = %v", err)
	}
	if err := DecodeStrict(strings.NewReader(`{"name":"too large","id":"1"}`), 8, &target); !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("oversize error = %v", err)
	}
}
