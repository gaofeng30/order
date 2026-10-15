package objectstore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"image"
	"image/png"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

type immutableRPC struct {
	mu      sync.Mutex
	receipt imagePinReceiptFacts
	commits int
	lose    bool
}

func (r *immutableRPC) CallRPC(_ context.Context, name string, p map[string]any, out any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var value any
	switch name {
	case "order_image_confirmation_replay_v1":
		value = r.receipt
		if r.receipt.Outcome == "" {
			value = imagePinReceiptFacts{Outcome: "ABSENT"}
		}
	case "order_image_confirmation_commit_v1":
		if r.receipt.Outcome == "" {
			r.commits++
			r.receipt = imagePinReceiptFacts{Outcome: "SUCCEEDED", RequestDigest: p["p_request_digest"].(string), Response: &imageVersionPin{ObjectKey: p["p_object_key"].(string), PublishedKey: p["p_published_key"].(string), VersionID: p["p_version_id"].(string)}}
		}
		value = r.receipt
		if r.lose {
			r.lose = false
			return ErrUnavailable
		}
	default:
		return ErrUnavailable
	}
	b, _ := json.Marshal(value)
	return json.Unmarshal(b, out)
}
func TestImmutableConfirmationNoNativeVersionOrCopy(t *testing.T) {
	img, spec := ablationImage()
	meta := WriteMeta{ActorUserID: 1, IdempotencyKey: "confirm"}
	key, _ := objectKeyForUpload(meta, spec)
	rpc := &immutableRPC{lose: true}
	a := ablationAdapter(t, ablationTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != "GET" || r.URL.Query().Has("versionId") {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		return ablationResponse(r, 200, img.Bytes, ""), nil
	}))
	for i := 0; i < 2; i++ {
		s := NewSharedPGImmutableService(a, rpc)
		got, e := s.ConfirmImageUpload(context.Background(), meta, spec, key)
		if e != nil || got.ObjectKey != key || strings.Contains(got.URL, "versionId=") {
			t.Fatalf("confirm: %+v %v", got, e)
		}
	}
	img.Bytes[0] ^= 1
	if _, err := NewSharedPGImmutableService(a, rpc).ConfirmImageUpload(context.Background(), meta, spec, key); err == nil {
		t.Fatal("tampered confirmed object accepted")
	}
	img.Bytes[0] ^= 1
	if rpc.commits != 1 {
		t.Fatalf("commits=%d", rpc.commits)
	}
	if rpc.receipt.Response.PublishedKey != key || rpc.receipt.Response.VersionID != "" {
		t.Fatal("not immutable direct pin")
	}
}
func TestImmutablePinDoesNotAcceptArbitraryMissingVersion(t *testing.T) {
	_, spec := ablationImage()
	key, _ := objectKeyForUpload(WriteMeta{ActorUserID: 1, IdempotencyKey: "x"}, spec)
	if !validImageVersionPin(imageVersionPin{key, key, ""}) {
		t.Fatal("direct immutable pin rejected")
	}
	if validImageVersionPin(imageVersionPin{key, "images/published/" + strings.Repeat("a", 64) + "/" + spec.SHA256 + ".png", ""}) {
		t.Fatal("missing historical version accepted")
	}
}

type ablationTransport func(*http.Request) (*http.Response, error)

func (f ablationTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func ablationAdapter(t *testing.T, tr http.RoundTripper) *COSAdapter {
	t.Helper()
	a, e := newCOSAdapter(COSConfig{"lab-123456", "ap-shanghai"}, common.NewTokenCredential("dummy", "dummy", "dummy"), tr)
	if e != nil {
		t.Fatal(e)
	}
	return a
}
func ablationImage() (Image, UploadSpecification) {
	var b bytes.Buffer
	_ = png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	s := sha256.Sum256(b.Bytes())
	return Image{ContentType: "image/png", Bytes: b.Bytes()}, UploadSpecification{"image/png", int64(b.Len()), fmt.Sprintf("%x", s)}
}
func ablationResponse(r *http.Request, code int, data []byte, version string) *http.Response {
	h := http.Header{"Content-Type": []string{"image/png"}}
	if version != "" {
		h.Set("x-cos-version-id", version)
	}
	return &http.Response{StatusCode: code, Header: h, Body: io.NopCloser(bytes.NewReader(data)), ContentLength: int64(len(data)), Request: r}
}

func TestImmutableUploadIntentRequiresOverwriteGuard(t *testing.T) {
	_, spec := ablationImage()
	a := ablationAdapter(t, ablationTransport(func(*http.Request) (*http.Response, error) { t.Fatal("unexpected network"); return nil, nil }))
	s := NewSharedPGImmutableService(a, &immutableRPC{})
	intent, err := s.PrepareImageUpload(context.Background(), WriteMeta{ActorUserID: 1, IdempotencyKey: "upload"}, spec)
	if err != nil {
		t.Fatal(err)
	}
	if intent.Headers["x-cos-forbid-overwrite"] != "true" || !strings.Contains(intent.URL, "x-cos-forbid-overwrite") {
		t.Fatal("overwrite prohibition missing from signed upload")
	}
}
