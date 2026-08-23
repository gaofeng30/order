package objectstore

import (
	"bytes"
	"context"
	"github.com/gin-gonic/gin"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeObjectApp struct{ called bool }

func (f *fakeObjectApp) PutImage(context.Context, WriteMeta, Image) (StoredImage, error) {
	f.called = true
	return StoredImage{}, nil
}

func TestFileAdapterIsContentAddressedAndStable(t *testing.T) {
	adapter, err := NewFileAdapter(t.TempDir(), "http://local.test/objects")
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(adapter)
	first, err := service.PutImage(context.Background(), WriteMeta{}, Image{Name: "a.png", ContentType: "image/png", Bytes: []byte("same")})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.PutImage(context.Background(), WriteMeta{}, Image{Name: "b.png", ContentType: "image/png", Bytes: []byte("same")})
	if err != nil {
		t.Fatal(err)
	}
	if first != second || first.ObjectKey == "" {
		t.Fatalf("first=%#v second=%#v", first, second)
	}
}
func TestUploadRejectsNonImageBeforeAdapter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	f := &fakeObjectApp{}
	e := gin.New()
	g := e.Group("/api/v1")
	g.Use(func(c *gin.Context) { c.Set("actor_user_id", uint64(1)); c.Next() })
	NewHandler(f).RegisterRoutes(g)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	p, _ := mw.CreateFormFile("file", "x.txt")
	_, _ = p.Write([]byte("not image"))
	_ = mw.Close()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/upload", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.Header.Set("Idempotency-Key", "upload-1")
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	if w.Code != http.StatusUnprocessableEntity || f.called {
		t.Fatalf("status=%d called=%v body=%s", w.Code, f.called, w.Body.String())
	}
}
