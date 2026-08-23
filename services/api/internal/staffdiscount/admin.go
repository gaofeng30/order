package staffdiscount

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	ErrInvalidInput        = errors.New("staff discount invalid input")
	ErrNotFound            = errors.New("staff discount not found")
	ErrConflict            = errors.New("staff discount conflict")
	ErrIdempotencyConflict = errors.New("staff discount idempotency conflict")
	ErrUnavailable         = errors.New("staff discount unavailable")
)

type WriteMeta struct {
	ActorUserID               uint64
	IdempotencyKey, RequestID string
}
type Staff struct {
	ID                uint64
	Name, MaskedPhone string
	Enabled, Bound    bool
	SpendCents        uint64
	OrderCount        uint32
	CreatedAt         time.Time
}
type CommandKind string

const (
	CreateStaff     CommandKind = "CREATE_STAFF"
	UpdateStaff     CommandKind = "UPDATE_STAFF"
	DeleteStaff     CommandKind = "DELETE_STAFF"
	SetStaffEnabled CommandKind = "SET_STAFF_ENABLED"
	SetDiscountRate CommandKind = "SET_DISCOUNT_RATE"
)

type Command struct {
	Kind        CommandKind
	StaffID     uint64
	Name, Phone string
	Enabled     *bool
	RatePercent uint8
}
type Result struct {
	Staff       *Staff
	RatePercent uint8
}

// Application keeps phone normalization, duplicate protection, receipts and MySQL locks internal.
type Application interface {
	List(context.Context, string) ([]Staff, error)
	DiscountRate(context.Context) (uint8, error)
	Execute(context.Context, WriteMeta, Command) (Result, error)
}
type Handler struct{ app Application }

func NewHandler(app Application) *Handler { return &Handler{app: app} }
func (h *Handler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/staff-whitelist", h.list)
	group.POST("/staff-whitelist", h.create)
	group.PUT("/staff-whitelist/:id", h.update)
	group.DELETE("/staff-whitelist/:id", h.delete)
	group.GET("/discount-rate", h.rate)
	group.PUT("/discount-rate", h.setRate)
}

type staffDTO struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	PhoneMasked string    `json:"phone_masked"`
	Enabled     bool      `json:"enabled"`
	Bound       bool      `json:"bound"`
	SpendCents  uint64    `json:"spend_cents"`
	OrderCount  uint32    `json:"order_count"`
	CreatedAt   time.Time `json:"created_at"`
}

func view(s Staff) staffDTO {
	return staffDTO{strconv.FormatUint(s.ID, 10), s.Name, s.MaskedPhone, s.Enabled, s.Bound, s.SpendCents, s.OrderCount, s.CreatedAt}
}
func (h *Handler) list(c *gin.Context) {
	items, err := h.app.List(c.Request.Context(), strings.TrimSpace(c.Query("q")))
	if err != nil {
		writeError(c, err)
		return
	}
	out := make([]staffDTO, 0, len(items))
	for _, item := range items {
		out = append(out, view(item))
	}
	c.JSON(http.StatusOK, gin.H{"staff": out})
}
func (h *Handler) rate(c *gin.Context) {
	rate, err := h.app.DiscountRate(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"rate_percent": rate})
}

type staffWrite struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Enabled *bool  `json:"enabled"`
}
type rateWrite struct {
	Rate uint8 `json:"rate_percent"`
}

var phonePattern = regexp.MustCompile(`^1[3-9][0-9]{9}$`)

func (h *Handler) create(c *gin.Context) {
	var in staffWrite
	if !decode(c, &in) || !phonePattern.MatchString(in.Phone) || strings.TrimSpace(in.Name) == "" || in.Enabled != nil {
		writeError(c, ErrInvalidInput)
		return
	}
	h.execute(c, Command{Kind: CreateStaff, Name: strings.TrimSpace(in.Name), Phone: in.Phone}, http.StatusCreated)
}
func (h *Handler) update(c *gin.Context) {
	id, ok := idOf(c.Param("id"))
	var in staffWrite
	if !ok || !decode(c, &in) {
		writeError(c, ErrInvalidInput)
		return
	}
	if in.Name != "" || in.Phone != "" {
		if (in.Phone != "" && !phonePattern.MatchString(in.Phone)) || strings.TrimSpace(in.Name) == "" || in.Enabled != nil {
			writeError(c, ErrInvalidInput)
			return
		}
		h.execute(c, Command{Kind: UpdateStaff, StaffID: id, Name: strings.TrimSpace(in.Name), Phone: in.Phone}, http.StatusOK)
		return
	}
	if in.Enabled == nil {
		writeError(c, ErrInvalidInput)
		return
	}
	h.execute(c, Command{Kind: SetStaffEnabled, StaffID: id, Enabled: in.Enabled}, http.StatusOK)
}
func (h *Handler) delete(c *gin.Context) {
	id, ok := idOf(c.Param("id"))
	if !ok {
		writeError(c, ErrNotFound)
		return
	}
	h.execute(c, Command{Kind: DeleteStaff, StaffID: id}, http.StatusOK)
}
func (h *Handler) setRate(c *gin.Context) {
	var in rateWrite
	if !decode(c, &in) || in.Rate < 1 || in.Rate > 100 {
		writeError(c, ErrInvalidInput)
		return
	}
	h.execute(c, Command{Kind: SetDiscountRate, RatePercent: in.Rate}, http.StatusOK)
}
func (h *Handler) execute(c *gin.Context, cmd Command, status int) {
	meta, ok := metaOf(c)
	if !ok {
		writeError(c, ErrInvalidInput)
		return
	}
	result, err := h.app.Execute(c.Request.Context(), meta, cmd)
	if err != nil {
		writeError(c, err)
		return
	}
	if result.Staff != nil {
		c.JSON(status, view(*result.Staff))
		return
	}
	c.JSON(status, gin.H{"rate_percent": result.RatePercent})
}
func decode(c *gin.Context, out any) bool {
	if !strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
		return false
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 16385))
	if err != nil || len(body) == 0 || len(body) > 16384 {
		return false
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return false
	}
	var x any
	return errors.Is(d.Decode(&x), io.EOF)
}
func idOf(v string) (uint64, bool) {
	id, err := strconv.ParseUint(v, 10, 64)
	return id, err == nil && id > 0
}
func metaOf(c *gin.Context) (WriteMeta, bool) {
	actor := c.GetUint64("actor_user_id")
	keys := c.Request.Header.Values("Idempotency-Key")
	if actor == 0 || len(keys) != 1 || strings.TrimSpace(keys[0]) == "" || strings.ContainsAny(keys[0], " \t\r\n") {
		return WriteMeta{}, false
	}
	return WriteMeta{actor, keys[0], c.GetString("request_id")}, true
}
func writeError(c *gin.Context, err error) {
	status, code, message := http.StatusServiceUnavailable, "STAFF_DISCOUNT_UNAVAILABLE", "staff discount temporarily unavailable"
	switch {
	case errors.Is(err, ErrInvalidInput):
		status, code, message = http.StatusBadRequest, "INVALID_REQUEST", "invalid request"
	case errors.Is(err, ErrNotFound):
		status, code, message = http.StatusNotFound, "NOT_FOUND", "resource not found"
	case errors.Is(err, ErrConflict):
		status, code, message = http.StatusConflict, "STAFF_CONFLICT", "staff conflict"
	case errors.Is(err, ErrIdempotencyConflict):
		status, code, message = http.StatusConflict, "IDEMPOTENCY_CONFLICT", "idempotency conflict"
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
