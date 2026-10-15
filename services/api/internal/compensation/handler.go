// Package compensation handles database-issued durable wake capabilities.
package compensation

import (
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var ErrUnauthorized = errors.New("compensation capability unavailable")

type Handler struct {
	execute func(context.Context, string) error
}

// NewHandler requires an executor which atomically consumes the capability in
// persistent storage before doing work. Syntax validation alone is not auth.
func NewHandler(execute func(context.Context, string) error) *Handler {
	return &Handler{execute: execute}
}

func (h *Handler) RegisterRoutes(engine *gin.Engine) {
	engine.POST("/internal/compensation/wake", gin.WrapH(h))
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery || r.ContentLength != 0 || len(r.TransferEncoding) != 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	_, err := hex.DecodeString(token)
	if len(r.Header.Values("Authorization")) != 1 || r.Header.Get("Authorization") != "Bearer "+token || len(token) != 64 || token != strings.ToLower(token) || err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if h == nil || h.execute == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	if err := h.execute(ctx, token); err != nil {
		if errors.Is(err, ErrUnauthorized) {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
