package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthRoutes(t *testing.T) {
	router := NewRouter(discardLogger())

	for _, path := range []string{"/health/live", "/health/ready"} {
		t.Run(path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, path, nil)

			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", recorder.Code)
			}
			if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
				t.Fatalf("Content-Type = %q, want application/json", contentType)
			}
			if body := strings.TrimSpace(recorder.Body.String()); body != `{"status":"ok"}` {
				t.Fatalf("body = %q, want health JSON", body)
			}
		})
	}
}

func TestHealthRoutesRejectWrongMethodAndUnknownPath(t *testing.T) {
	router := NewRouter(discardLogger())

	for _, path := range []string{"/health/live", "/health/ready"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, path, nil)

		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusMethodNotAllowed {
			t.Fatalf("POST %s status = %d, want 405", path, recorder.Code)
		}
		if strings.TrimSpace(recorder.Body.String()) == `{"status":"ok"}` {
			t.Fatalf("POST %s returned health success body", path)
		}
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unknown path status = %d, want 404", recorder.Code)
	}
}

func TestRequestIDAndAccessLogAreServerControlledAndSanitized(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	router := newRouter(logger, func(engine *gin.Engine) {
		engine.POST("/inspect", func(context *gin.Context) {
			context.Status(http.StatusNoContent)
		})
	})
	request := httptest.NewRequest(http.MethodPost, "/inspect?token=query-secret", strings.NewReader("body-secret"))
	request.Header.Set("X-Request-ID", "client-request-id")
	request.Header.Set("Authorization", "Bearer auth-secret")
	request.Header.Set("Cookie", "session=cookie-secret")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	requestID := recorder.Header().Get("X-Request-ID")
	if requestID == "" || requestID == "client-request-id" {
		t.Fatalf("X-Request-ID = %q, want nonempty server-generated value", requestID)
	}
	entry := findLogEntry(t, output.Bytes(), "request completed")
	if entry["request_id"] != requestID {
		t.Fatalf("logged request_id = %v, want %s", entry["request_id"], requestID)
	}
	if entry["method"] != http.MethodPost || entry["path"] != "/inspect" || entry["status"] != float64(http.StatusNoContent) {
		t.Fatalf("access log fields = %#v", entry)
	}
	if _, ok := entry["duration_ms"]; !ok {
		t.Fatalf("access log missing duration_ms: %#v", entry)
	}
	logged := output.String()
	for _, secret := range []string{"query-secret", "body-secret", "auth-secret", "cookie-secret", "client-request-id"} {
		if strings.Contains(logged, secret) {
			t.Fatalf("access log leaked %q: %s", secret, logged)
		}
	}
}

func TestRecoveryReturns500WithoutExposingPanic(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	router := newRouter(logger, func(engine *gin.Engine) {
		engine.GET("/panic", func(context *gin.Context) {
			panic("panic-secret")
		})
	})
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "panic-secret") || strings.Contains(recorder.Body.String(), "stack") {
		t.Fatalf("response exposed panic details: %q", recorder.Body.String())
	}
	recovery := findLogEntry(t, output.Bytes(), "request panic recovered")
	access := findLogEntry(t, output.Bytes(), "request completed")
	if recovery["request_id"] == "" || recovery["request_id"] != access["request_id"] {
		t.Fatalf("panic and access logs are not correlated: recovery=%#v access=%#v", recovery, access)
	}
	if strings.Contains(output.String(), "panic-secret") {
		t.Fatalf("panic log exposed panic value: %s", output.String())
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
}

func findLogEntry(t *testing.T, data []byte, message string) map[string]any {
	t.Helper()
	for line := range bytes.SplitSeq(data, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Fatalf("decode log %q: %v", line, err)
		}
		if entry["msg"] == message {
			return entry
		}
	}
	t.Fatalf("log message %q not found in %s", message, data)
	return nil
}
