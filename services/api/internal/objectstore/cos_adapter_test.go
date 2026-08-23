package objectstore

import (
	"context"
	"errors"
	"hash/crc64"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

type cosRoundTripFunc func(*http.Request) (*http.Response, error)

func (function cosRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestCOSAdapterPutUsesOfficialCOSRequest(t *testing.T) {
	const key = "images/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.png"
	var calls int
	transport := cosRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.Method != http.MethodPut || request.URL.Host != "order-images-1250000000.cos.ap-guangzhou.myqcloud.com" || request.URL.Path != "/"+key {
			t.Fatalf("method=%s url=%s", request.Method, request.URL.String())
		}
		if request.Header.Get("Content-Type") != "image/png" || request.Header.Get("Authorization") == "" || request.Header.Get("x-cos-security-token") != "test-token" {
			t.Fatalf("headers=%v", request.Header)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil || string(body) != "image-bytes" {
			t.Fatalf("body=%q err=%v", body, err)
		}
		responseHeader := make(http.Header)
		responseHeader.Set("x-cos-hash-crc64ecma", strconv.FormatUint(crc64.Checksum([]byte("image-bytes"), crc64.MakeTable(crc64.ECMA)), 10))
		return &http.Response{StatusCode: http.StatusOK, Header: responseHeader, Body: io.NopCloser(strings.NewReader("")), Request: request}, nil
	})
	adapter, err := newCOSAdapter(COSConfig{
		Bucket: "order-images-1250000000", Region: "ap-guangzhou", PublicOrigin: "https://images.example.com",
	}, common.NewTokenCredential("test-secret-id", "test-secret-key", "test-token"), transport)
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.Put(context.Background(), key, []byte("image-bytes"), "image/png"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	publicURL, err := adapter.PublicURL(context.Background(), key)
	if err != nil || publicURL != "https://images.example.com/"+key {
		t.Fatalf("url=%q err=%v", publicURL, err)
	}
}

func TestCOSAdapterFailsClosedForInvalidKeyAndProviderError(t *testing.T) {
	providerSecret := "provider-secret-must-not-leak"
	calls := 0
	transport := cosRoundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New(providerSecret)
	})
	adapter, err := newCOSAdapter(COSConfig{
		Bucket: "order-images-1250000000", Region: "ap-guangzhou", PublicOrigin: "https://images.example.com",
	}, common.NewTokenCredential("test-secret-id", "test-secret-key", "test-token"), transport)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"", "/images/a.png", "../images/a.png", "images/../a.png", `images\\a.png`, "images/a b.png"} {
		if err := adapter.Put(context.Background(), key, []byte("x"), "image/png"); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("key=%q err=%v", key, err)
		}
		if _, err := adapter.PublicURL(context.Background(), key); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("key=%q public err=%v", key, err)
		}
	}
	if calls != 0 {
		t.Fatalf("invalid key reached provider calls=%d", calls)
	}
	validKey := "images/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.jpg"
	err = adapter.Put(context.Background(), validKey, []byte("x"), "image/jpeg")
	if !errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), providerSecret) {
		t.Fatalf("err=%v", err)
	}
	if calls != 1 {
		t.Fatalf("provider calls=%d", calls)
	}
}

func TestCOSAdapterRejectsInvalidConstructorInputs(t *testing.T) {
	validCredential := common.NewTokenCredential("id", "key", "token")
	tests := []struct {
		name       string
		config     COSConfig
		credential common.CredentialIface
	}{
		{name: "nil credential", config: COSConfig{Bucket: "order-images-1250000000", Region: "ap-guangzhou", PublicOrigin: "https://images.example.com"}},
		{name: "wrong endpoint bucket", config: COSConfig{Bucket: "order-images", Region: "ap-guangzhou", PublicOrigin: "https://images.example.com"}, credential: validCredential},
		{name: "invalid region", config: COSConfig{Bucket: "order-images-1250000000", Region: "guangzhou", PublicOrigin: "https://images.example.com"}, credential: validCredential},
		{name: "insecure public origin", config: COSConfig{Bucket: "order-images-1250000000", Region: "ap-guangzhou", PublicOrigin: "http://images.example.com"}, credential: validCredential},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := newCOSAdapter(test.config, test.credential, http.DefaultTransport); !errors.Is(err, ErrUnavailable) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}
