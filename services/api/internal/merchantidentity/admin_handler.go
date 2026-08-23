package merchantidentity

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
	ErrAdminInvalidInput        = errors.New("merchant admin invalid input")
	ErrAdminNotFound            = errors.New("merchant admin not found")
	ErrAdminConflict            = errors.New("merchant admin conflict")
	ErrAdminIdempotencyConflict = errors.New("merchant admin idempotency conflict")
	ErrLastOwner                = errors.New("merchant last owner")
	ErrPCLoginExpired           = errors.New("pc login expired")
	ErrPCSessionExpired         = errors.New("pc session expired")
	ErrAdminUnavailable         = errors.New("merchant admin unavailable")
)

type AdminWriteMeta struct {
	ActorUserID               uint64
	IdempotencyKey, RequestID string
}
type Account struct {
	ID                uint64
	Name, MaskedPhone string
	Role              Role
	Enabled, Bound    bool
}
type AccountCommandKind string

const (
	CreateAccount     AccountCommandKind = "CREATE_ACCOUNT"
	UpdateAccount     AccountCommandKind = "UPDATE_ACCOUNT"
	DeleteAccount     AccountCommandKind = "DELETE_ACCOUNT"
	SetAccountEnabled AccountCommandKind = "SET_ACCOUNT_ENABLED"
)

type AccountCommand struct {
	Kind        AccountCommandKind
	AccountID   uint64
	Name, Phone string
	Role        Role
	Enabled     *bool
}
type PCLogin struct {
	LoginID, PollSecret, QRPayload string
	ExpiresAt                      time.Time
}
type PCSession struct {
	State, Token string
	ExpiresAt    time.Time
}

// AdminApplication owns the account locks/last-owner invariant and intrinsic QR dedupe.
type AdminApplication interface {
	CurrentAccount(context.Context, uint64) (Account, error)
	ListAccounts(context.Context, uint64, string) ([]Account, error)
	ExecuteAccount(context.Context, AdminWriteMeta, AccountCommand) (*Account, error)
	BeginPCLogin(context.Context) (PCLogin, error)
	ApprovePCLogin(context.Context, uint64, string, string, string) error
	PollPCLogin(context.Context, string, string) (PCSession, error)
	AuthenticatePC(context.Context, string) (uint64, error)
}
type AdminHandler struct{ app AdminApplication }

func NewAdminHandler(app AdminApplication) *AdminHandler { return &AdminHandler{app: app} }
func (h *AdminHandler) RegisterAdminRoutes(group *gin.RouterGroup) {
	group.GET("/me", h.me)
	group.GET("/merchant-accounts", h.accounts)
	group.POST("/merchant-accounts", h.createAccount)
	group.PUT("/merchant-accounts/:id", h.updateAccount)
	group.DELETE("/merchant-accounts/:id", h.deleteAccount)
}

// RegisterPCAuthRoutes mounts intrinsic-dedupe begin/poll below /api/v1 without
// PC-session middleware; protecting these routes would make initial login impossible.
func (h *AdminHandler) RegisterPCAuthRoutes(api *gin.RouterGroup) {
	api.POST("/admin/auth/qrcode", h.begin)
	api.POST("/admin/auth/poll", h.poll)
}

// RegisterApprovalRoute mounts below the Mini Program authenticated /api/v1/me group.
func (h *AdminHandler) RegisterApprovalRoute(me *gin.RouterGroup) {
	me.POST("/admin-login/approve", h.approve)
}

type accountDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PhoneMasked string `json:"phone_masked"`
	Role        Role   `json:"role"`
	Enabled     bool   `json:"enabled"`
	Bound       bool   `json:"bound"`
}

func accountView(a Account) accountDTO {
	return accountDTO{strconv.FormatUint(a.ID, 10), a.Name, a.MaskedPhone, a.Role, a.Enabled, a.Bound}
}
func (h *AdminHandler) me(c *gin.Context) {
	actor := c.GetUint64("actor_user_id")
	if actor == 0 {
		adminError(c, ErrPCSessionExpired)
		return
	}
	a, err := h.app.CurrentAccount(c.Request.Context(), actor)
	if err != nil {
		adminError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"account": accountView(a)})
}
func (h *AdminHandler) accounts(c *gin.Context) {
	actor := c.GetUint64("actor_user_id")
	if actor == 0 {
		adminError(c, ErrPCSessionExpired)
		return
	}
	items, err := h.app.ListAccounts(c.Request.Context(), actor, strings.TrimSpace(c.Query("q")))
	if err != nil {
		adminError(c, err)
		return
	}
	out := make([]accountDTO, 0, len(items))
	for _, item := range items {
		out = append(out, accountView(item))
	}
	c.JSON(http.StatusOK, gin.H{"accounts": out})
}

type accountWrite struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Role    Role   `json:"role"`
	Enabled *bool  `json:"enabled"`
}

var adminPhone = regexp.MustCompile(`^1[3-9][0-9]{9}$`)

func (h *AdminHandler) createAccount(c *gin.Context) {
	var in accountWrite
	if !adminDecode(c, &in) || !adminPhone.MatchString(in.Phone) || strings.TrimSpace(in.Name) == "" || (in.Role != RoleOwner && in.Role != RoleSubaccount) || in.Enabled != nil {
		adminError(c, ErrAdminInvalidInput)
		return
	}
	h.accountCommand(c, AccountCommand{Kind: CreateAccount, Name: strings.TrimSpace(in.Name), Phone: in.Phone, Role: in.Role}, http.StatusCreated)
}
func (h *AdminHandler) updateAccount(c *gin.Context) {
	id, ok := adminUint(c.Param("id"))
	var in accountWrite
	if !ok || !adminDecode(c, &in) {
		adminError(c, ErrAdminInvalidInput)
		return
	}
	if in.Name != "" || in.Phone != "" || in.Role != "" {
		if (in.Phone != "" && !adminPhone.MatchString(in.Phone)) || strings.TrimSpace(in.Name) == "" || (in.Role != RoleOwner && in.Role != RoleSubaccount) || in.Enabled != nil {
			adminError(c, ErrAdminInvalidInput)
			return
		}
		h.accountCommand(c, AccountCommand{Kind: UpdateAccount, AccountID: id, Name: strings.TrimSpace(in.Name), Phone: in.Phone, Role: in.Role}, http.StatusOK)
		return
	}
	if in.Enabled == nil {
		adminError(c, ErrAdminInvalidInput)
		return
	}
	h.accountCommand(c, AccountCommand{Kind: SetAccountEnabled, AccountID: id, Enabled: in.Enabled}, http.StatusOK)
}
func (h *AdminHandler) deleteAccount(c *gin.Context) {
	id, ok := adminUint(c.Param("id"))
	if !ok {
		adminError(c, ErrAdminNotFound)
		return
	}
	h.accountCommand(c, AccountCommand{Kind: DeleteAccount, AccountID: id}, http.StatusOK)
}
func (h *AdminHandler) accountCommand(c *gin.Context, cmd AccountCommand, status int) {
	meta, ok := accountMeta(c)
	if !ok {
		adminError(c, ErrAdminInvalidInput)
		return
	}
	a, err := h.app.ExecuteAccount(c.Request.Context(), meta, cmd)
	if err != nil {
		adminError(c, err)
		return
	}
	if a == nil {
		c.JSON(status, gin.H{"deleted": true})
		return
	}
	c.JSON(status, accountView(*a))
}

func (h *AdminHandler) begin(c *gin.Context) {
	if !emptyJSONObject(c) {
		adminError(c, ErrAdminInvalidInput)
		return
	}
	login, err := h.app.BeginPCLogin(c.Request.Context())
	if err != nil {
		adminError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, gin.H{"login_id": login.LoginID, "poll_secret": login.PollSecret, "qr_payload": login.QRPayload, "expires_at": login.ExpiresAt})
}

type approveWrite struct {
	LoginID        string `json:"login_id"`
	ApprovalSecret string `json:"approval_secret"`
	Code           string `json:"code"`
}

func (h *AdminHandler) approve(c *gin.Context) {
	var in approveWrite
	if !adminDecode(c, &in) || in.LoginID == "" || in.ApprovalSecret == "" || in.Code == "" {
		adminError(c, ErrAdminInvalidInput)
		return
	}
	userID := c.GetUint64("user_id")
	if userID == 0 {
		adminError(c, ErrPCSessionExpired)
		return
	}
	if err := h.app.ApprovePCLogin(c.Request.Context(), userID, in.LoginID, in.ApprovalSecret, in.Code); err != nil {
		adminError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"approved": true})
}

type pollWrite struct {
	LoginID    string `json:"login_id"`
	PollSecret string `json:"poll_secret"`
}

func (h *AdminHandler) poll(c *gin.Context) {
	var in pollWrite
	if !adminDecode(c, &in) || in.LoginID == "" || in.PollSecret == "" {
		adminError(c, ErrAdminInvalidInput)
		return
	}
	session, err := h.app.PollPCLogin(c.Request.Context(), in.LoginID, in.PollSecret)
	if err != nil {
		adminError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	if session.State != "APPROVED" {
		c.JSON(http.StatusAccepted, gin.H{"state": "WAITING"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"state": "APPROVED", "session": gin.H{"token": session.Token, "expires_at": session.ExpiresAt}})
}

func emptyJSONObject(c *gin.Context) bool {
	var in map[string]any
	return adminDecode(c, &in) && len(in) == 0
}
func adminDecode(c *gin.Context, out any) bool {
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
func adminUint(v string) (uint64, bool) {
	id, err := strconv.ParseUint(v, 10, 64)
	return id, err == nil && id > 0
}
func accountMeta(c *gin.Context) (AdminWriteMeta, bool) {
	actor := c.GetUint64("actor_user_id")
	keys := c.Request.Header.Values("Idempotency-Key")
	if actor == 0 || len(keys) != 1 || strings.TrimSpace(keys[0]) == "" || strings.ContainsAny(keys[0], " \t\r\n") {
		return AdminWriteMeta{}, false
	}
	return AdminWriteMeta{actor, keys[0], c.GetString("request_id")}, true
}
func adminError(c *gin.Context, err error) {
	status, code, message := http.StatusServiceUnavailable, "MERCHANT_AUTH_UNAVAILABLE", "merchant auth temporarily unavailable"
	switch {
	case errors.Is(err, ErrAdminInvalidInput):
		status, code, message = http.StatusBadRequest, "INVALID_REQUEST", "invalid request"
	case errors.Is(err, ErrAdminNotFound):
		status, code, message = http.StatusNotFound, "NOT_FOUND", "resource not found"
	case errors.Is(err, ErrForbidden):
		status, code, message = http.StatusForbidden, "FORBIDDEN", "forbidden"
	case errors.Is(err, ErrAdminConflict):
		status, code, message = http.StatusConflict, "ACCOUNT_CONFLICT", "account conflict"
	case errors.Is(err, ErrAdminIdempotencyConflict):
		status, code, message = http.StatusConflict, "IDEMPOTENCY_CONFLICT", "idempotency conflict"
	case errors.Is(err, ErrLastOwner):
		status, code, message = http.StatusConflict, "LAST_OWNER", "at least one enabled owner is required"
	case errors.Is(err, ErrPCLoginExpired):
		status, code, message = http.StatusGone, "PC_LOGIN_EXPIRED", "pc login expired"
	case errors.Is(err, ErrPCSessionExpired):
		status, code, message = http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required"
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
