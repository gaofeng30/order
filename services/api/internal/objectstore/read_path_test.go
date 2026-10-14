package objectstore

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gaofeng30/order/services/api/internal/catalog"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

type readRPC struct {
	calls int
	pins  []imageVersionPin
	fail  bool
}

func (r *readRPC) CallRPC(_ context.Context, name string, params map[string]any, out any) error {
	r.calls++
	if r.fail {
		return ErrUnavailable
	}
	pins := []imageVersionPin{}
	for _, key := range params["p_object_keys"].([]string) {
		for _, pin := range r.pins {
			if pin.ObjectKey == key {
				pins = append(pins, pin)
			}
		}
	}
	b, _ := json.Marshal(map[string]any{"outcome": "SUCCEEDED", "versions": pins})
	return json.Unmarshal(b, out)
}

type noObjectRead struct{ calls int }

func (r *noObjectRead) RoundTrip(req *http.Request) (*http.Response, error) {
	r.calls++
	return nil, fmt.Errorf("unexpected object read: %s", req.Method)
}
func readService(t *testing.T) (*SharedPGVersionedService, *readRPC, *noObjectRead) {
	t.Helper()
	transport := &noObjectRead{}
	rpc := &readRPC{}
	adapter, err := newCOSAdapter(COSConfig{"lab-123456", "ap-shanghai"}, common.NewTokenCredential("lab-id", "lab-key", "lab-token"), transport)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 16; i++ {
		suffix := fmt.Sprintf("%064x.png", i+1)
		rpc.pins = append(rpc.pins, imageVersionPin{"images/" + suffix, "images/published/" + strings.Repeat("a", 64) + "/" + suffix, "fixed-version"})
	}
	s := NewSharedPGVersionedService(adapter, rpc)
	s.now = func() time.Time { return time.Unix(1791900000, 0) }
	return s, rpc, transport
}
func TestPublicURLDoesNotDownloadConfirmedImage(t *testing.T) {
	s, rpc, transport := readService(t)
	url, err := s.PublicURL(context.Background(), rpc.pins[0].ObjectKey)
	if err != nil || !strings.Contains(url, "versionId=fixed-version") || transport.calls != 0 {
		t.Fatalf("browse must only resolve confirmed pin and sign: url=%q err=%v object calls=%d", url, err, transport.calls)
	}
	if _, err = s.Confirmed(context.Background(), []string{rpc.pins[0].ObjectKey}); err == nil || transport.calls == 0 {
		t.Fatal("write confirmation must still validate actual bytes")
	}
}
func TestPublicURLsBatchAndMissingPins(t *testing.T) {
	s, rpc, transport := readService(t)
	keys := []string{}
	for _, p := range rpc.pins {
		keys = append(keys, p.ObjectKey)
	}
	keys = append(keys, keys[0], "images/"+strings.Repeat("f", 64)+".png")
	urls, err := s.PublicURLs(context.Background(), keys)
	if err != nil || len(urls) != 16 || rpc.calls != 1 || transport.calls != 0 {
		t.Fatalf("batch budget: urls=%d rpc=%d object=%d err=%v", len(urls), rpc.calls, transport.calls, err)
	}
	rpc.fail = true
	if _, err = s.PublicURLs(context.Background(), keys); err == nil {
		t.Fatal("pin store failure must not produce guessed URLs")
	}
}

type browseCatalog struct{ products []catalog.Product }

func (c browseCatalog) Browse(context.Context) ([]catalog.Category, error) {
	return []catalog.Category{{ID: 1, Name: "menu", Products: c.products}}, nil
}
func (c browseCatalog) Detail(context.Context, uint64, string) (catalog.Product, catalog.CurrentFacts, error) {
	return catalog.Product{}, catalog.CurrentFacts{}, fmt.Errorf("unused")
}
func TestCatalogHTTPReadBudget(t *testing.T) {
	s, rpc, transport := readService(t)
	products := []catalog.Product{}
	for i := 0; i < 8; i++ {
		products = append(products, catalog.Product{ID: uint64(i + 1), CategoryID: 1, Name: "dish", MealPeriod: "all", Listed: true, OriginalUnitPriceCents: 100, ImageObjectKeys: []string{rpc.pins[i*2].ObjectKey, rpc.pins[i*2+1].ObjectKey}})
	}
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	catalog.NewHandler(browseCatalog{products}, catalog.WithPublicURLs(s)).RegisterRoutes(engine)
	for _, fail := range []bool{false, true} {
		rpc.fail = fail
		rpc.calls = 0
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/catalog", nil))
		if rec.Code != 200 || rpc.calls != 1 || transport.calls != 0 {
			t.Fatalf("HTTP=%d pins=%d objects=%d body=%s", rec.Code, rpc.calls, transport.calls, rec.Body.String())
		}
		if !fail && strings.Count(rec.Body.String(), "versionId=fixed-version") != 16 {
			t.Fatal("missing pinned images")
		}
		if fail && strings.Contains(rec.Body.String(), "versionId=") {
			t.Fatal("failed pin lookup exposed a URL")
		}
	}
}

func TestDisplayURLUsesSignedBoundedImageProcessing(t *testing.T) {
	s, rpc, transport := readService(t)
	raw, err := s.PublicURL(context.Background(), rpc.pins[0].ObjectKey)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	q := parsed.Query()
	rule := "imageMogr2/thumbnail/640x640>/format/webp/quality/80"
	if !q.Has(rule) || q.Get("versionId") != "fixed-version" {
		t.Fatal("display URL must resize the exact confirmed version")
	}
	signedParams, _ := url.QueryUnescape(q.Get("q-url-param-list"))
	if !strings.Contains(signedParams, strings.ToLower(rule)) {
		t.Fatalf("image processing must be signed: %s", q.Get("q-url-param-list"))
	}
	if transport.calls != 0 {
		t.Fatal("display must not download images on the API server")
	}
}
