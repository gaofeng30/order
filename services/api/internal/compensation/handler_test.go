package compensation

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWakeDoesNotExecuteWithoutExactCapability(t *testing.T) {
	calls := 0
	handler := NewHandler(func(context.Context, string) error { calls++; return nil })
	for _, token := range []string{"", "Bearer x", "Bearer " + strings.Repeat("A", 64), "Bearer " + strings.Repeat("a", 63), "Basic " + strings.Repeat("a", 64)} {
		r := httptest.NewRequest(http.MethodPost, "/internal/compensation/wake", nil)
		r.Header.Set("Authorization", token)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d", w.Code)
		}
	}
	if calls != 0 {
		t.Fatalf("unauthorized executions=%d", calls)
	}
}

func TestWakeReportsOnlyDurableExecutionOutcome(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{{"completed", nil, 204}, {"unknown token", ErrUnauthorized, 401}, {"database or provider failure", errors.New("sensitive diagnostic"), 503}} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			token := strings.Repeat("ab", 32)
			handler := NewHandler(func(ctx context.Context, got string) error {
				calls++
				if got != token {
					t.Fatal("changed token")
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("unbounded request")
				}
				return tc.err
			})
			r := httptest.NewRequest(http.MethodPost, "/internal/compensation/wake", nil)
			r.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != tc.status || calls != 1 {
				t.Fatalf("status=%d calls=%d", w.Code, calls)
			}
			if w.Body.Len() != 0 {
				t.Fatalf("unexpected diagnostic: %s", w.Body.String())
			}
		})
	}
}

func TestWakeRejectsExtraInstructionsAndDuplicateAuthorization(t *testing.T) {
	calls := 0
	handler := NewHandler(func(context.Context, string) error { calls++; return nil })
	for _, tc := range []struct {
		method, path, body string
		duplicate          bool
		status             int
	}{
		{"GET", "/internal/compensation/wake", "", false, 405},
		{"POST", "/internal/compensation/wake?kind=refund", "", false, 400},
		{"POST", "/internal/compensation/wake", `{"url":"https://example.invalid"}`, false, 400},
		{"POST", "/internal/compensation/wake", "", true, 401},
	} {
		r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		r.Header.Set("Authorization", "Bearer "+strings.Repeat("a", 64))
		if tc.duplicate {
			r.Header.Add("Authorization", "Bearer "+strings.Repeat("b", 64))
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("%s %s status=%d want=%d", tc.method, tc.path, w.Code, tc.status)
		}
	}
	if calls != 0 {
		t.Fatalf("invalid executions=%d", calls)
	}
}
