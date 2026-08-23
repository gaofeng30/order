package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLocalObjectRoutesServeOnlyFilesBelowDedicatedRoot(t *testing.T) {
	root := t.TempDir()
	imageDir := filepath.Join(root, "images")
	if err := os.Mkdir(imageDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imageDir, "sample.png"), []byte("image-bytes"), 0o640); err != nil {
		t.Fatal(err)
	}
	routes, err := newLocalObjectRoutes(root)
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	routes.RegisterRoutes(router)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/objects/images/sample.png", nil))
	if response.Code != http.StatusOK || response.Body.String() != "image-bytes" {
		t.Fatalf("file response = %d %q", response.Code, response.Body.String())
	}

	directory := httptest.NewRecorder()
	router.ServeHTTP(directory, httptest.NewRequest(http.MethodGet, "/api/v1/objects/images/", nil))
	if directory.Code == http.StatusOK {
		t.Fatalf("directory listing unexpectedly available: %s", directory.Body.String())
	}
}

func TestLocalObjectRoutesRejectBroadOrRelativeRoots(t *testing.T) {
	for _, root := range []string{".", "/"} {
		if _, err := newLocalObjectRoutes(root); err == nil {
			t.Fatalf("root %q accepted", root)
		}
	}
}
