package audit

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	ErrInvalidInput = errors.New("audit invalid input")
	ErrForbidden    = errors.New("audit forbidden")
	ErrUnavailable  = errors.New("audit unavailable")
)

type PageQuery struct {
	AfterID uint64
	Limit   uint16
}
type Filter struct {
	Target, Action, From, To string
	Page                     PageQuery
}
type Entry struct {
	ID                                                  uint64
	Action, TargetType, TargetID, ResultCode, RequestID string
	ActorAccountID                                      uint64
	CreatedAt                                           time.Time
}

// Searcher exposes redacted audit evidence only; command receipts remain internal to business Modules.
type Searcher interface {
	Search(context.Context, uint64, Filter) ([]Entry, uint64, error)
}
type Handler struct{ searcher Searcher }

func NewHandler(searcher Searcher) *Handler              { return &Handler{searcher: searcher} }
func (h *Handler) RegisterRoutes(group *gin.RouterGroup) { group.GET("/audits", h.search) }
func (h *Handler) search(c *gin.Context) {
	actor := c.GetUint64("actor_user_id")
	if actor == 0 {
		writeError(c, ErrForbidden)
		return
	}
	p, ok := page(c)
	if !ok {
		writeError(c, ErrInvalidInput)
		return
	}
	filter := Filter{Target: strings.TrimSpace(c.Query("target")), Action: strings.TrimSpace(c.Query("action")), From: c.Query("from"), To: c.Query("to"), Page: p}
	if filter.From != "" && filter.To != "" && filter.From > filter.To {
		writeError(c, ErrInvalidInput)
		return
	}
	items, next, err := h.searcher.Search(c.Request.Context(), actor, filter)
	if err != nil {
		writeError(c, err)
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		out = append(out, gin.H{"id": strconv.FormatUint(item.ID, 10), "action": item.Action, "target_type": item.TargetType, "target_id": item.TargetID, "result_code": item.ResultCode, "request_id": item.RequestID, "actor_account_id": strconv.FormatUint(item.ActorAccountID, 10), "created_at": item.CreatedAt})
	}
	var cursor any
	if next > 0 {
		cursor = strconv.FormatUint(next, 10)
	}
	c.JSON(http.StatusOK, gin.H{"audits": out, "next_after_id": cursor})
}
func page(c *gin.Context) (PageQuery, bool) {
	q := PageQuery{Limit: 50}
	if raw := c.Query("after_id"); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return PageQuery{}, false
		}
		q.AfterID = id
	}
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.ParseUint(raw, 10, 16)
		if err != nil || n < 1 || n > 100 {
			return PageQuery{}, false
		}
		q.Limit = uint16(n)
	}
	return q, true
}
func writeError(c *gin.Context, err error) {
	status, code, message := http.StatusServiceUnavailable, "AUDIT_UNAVAILABLE", "audit temporarily unavailable"
	if errors.Is(err, ErrInvalidInput) {
		status, code, message = http.StatusBadRequest, "INVALID_REQUEST", "invalid request"
	} else if errors.Is(err, ErrForbidden) {
		status, code, message = http.StatusForbidden, "FORBIDDEN", "forbidden"
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
