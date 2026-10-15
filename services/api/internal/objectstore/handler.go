package objectstore

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"strings"

	"github.com/gaofeng30/order/services/api/internal/httpdto"
	"github.com/gin-gonic/gin"
)

var (
	ErrInvalidImage  = errors.New("invalid image")
	ErrImageTooLarge = errors.New("image too large")
	ErrUnavailable   = errors.New("object store unavailable")
)

const maxUploadRequestBytes = 512

type WriteMeta struct {
	ActorUserID               uint64
	IdempotencyKey, RequestID string
}
type Image struct {
	Name, ContentType string
	Bytes             []byte
}
type StoredImage struct{ ObjectKey, URL string }

type Application interface {
	PrepareImageUpload(context.Context, WriteMeta, UploadSpecification) (UploadIntent, error)
	ConfirmImageUpload(context.Context, WriteMeta, UploadSpecification, string) (StoredImage, error)
}
type Handler struct{ app Application }

func NewHandler(app Application) *Handler { return &Handler{app: app} }
func (h *Handler) RegisterRoutes(api *gin.RouterGroup) {
	api.POST("/upload/intent", h.prepareUpload)
	api.POST("/upload", h.confirmUpload)
}

type uploadSpecificationRequest struct {
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	SHA256      string `json:"sha256"`
}

type uploadConfirmationRequest struct {
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	SHA256      string `json:"sha256"`
	ObjectKey   string `json:"object_key"`
}

func (h *Handler) prepareUpload(ctx *gin.Context) {
	meta, ok := meta(ctx)
	if !ok {
		writeError(ctx, ErrInvalidImage)
		return
	}
	var request uploadSpecificationRequest
	if !decodeUploadJSON(ctx, &request) {
		writeError(ctx, ErrInvalidImage)
		return
	}
	specification := UploadSpecification{ContentType: request.ContentType, SizeBytes: request.SizeBytes, SHA256: request.SHA256}
	if _, valid := objectKeyForSpecification(specification); !valid {
		writeError(ctx, ErrInvalidImage)
		return
	}
	intent, err := h.app.PrepareImageUpload(ctx.Request.Context(), meta, specification)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusCreated, gin.H{"upload": gin.H{
		"object_key": intent.ObjectKey,
		"method":     intent.Method,
		"url":        intent.URL,
		"headers":    intent.Headers,
		"expires_at": intent.ExpiresAt.UTC().Format("2006-01-02T15:04:05.000000Z"),
	}})
}

func (h *Handler) confirmUpload(ctx *gin.Context) {
	meta, ok := meta(ctx)
	if !ok {
		writeError(ctx, ErrInvalidImage)
		return
	}
	var request uploadConfirmationRequest
	if !decodeUploadJSON(ctx, &request) {
		writeError(ctx, ErrInvalidImage)
		return
	}
	specification := UploadSpecification{ContentType: request.ContentType, SizeBytes: request.SizeBytes, SHA256: request.SHA256}
	if !uploadObjectKeyMatches(specification, request.ObjectKey) {
		writeError(ctx, ErrInvalidImage)
		return
	}
	stored, err := h.app.ConfirmImageUpload(ctx.Request.Context(), meta, specification, request.ObjectKey)
	if err != nil {
		writeError(ctx, err)
		return
	}
	if stored.ObjectKey != request.ObjectKey || stored.URL == "" {
		writeError(ctx, ErrUnavailable)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusCreated, gin.H{"image": gin.H{"object_key": stored.ObjectKey, "url": stored.URL}})
}

func decodeUploadJSON(ctx *gin.Context, target any) bool {
	mediaType, _, err := mime.ParseMediaType(ctx.GetHeader("Content-Type"))
	return err == nil && mediaType == "application/json" && httpdto.DecodeStrict(ctx.Request.Body, maxUploadRequestBytes, target) == nil
}

func meta(c *gin.Context) (WriteMeta, bool) {
	actor := c.GetUint64("actor_user_id")
	keys := c.Request.Header.Values("Idempotency-Key")
	if actor == 0 || len(keys) != 1 || strings.TrimSpace(keys[0]) == "" || strings.ContainsAny(keys[0], " \t\r\n") {
		return WriteMeta{}, false
	}
	return WriteMeta{actor, keys[0], c.GetString("request_id")}, true
}
func writeError(c *gin.Context, err error) {
	status, code, message := http.StatusServiceUnavailable, "OBJECT_STORE_UNAVAILABLE", "object store temporarily unavailable"
	if errors.Is(err, ErrInvalidImage) {
		status, code, message = http.StatusUnprocessableEntity, "INVALID_IMAGE", "invalid image"
	} else if errors.Is(err, ErrImageTooLarge) {
		status, code, message = http.StatusRequestEntityTooLarge, "IMAGE_TOO_LARGE", "image too large"
	} else if errors.Is(err, ErrIdempotencyConflict) {
		status, code, message = http.StatusConflict, "IDEMPOTENCY_CONFLICT", "idempotency key conflicts with an earlier upload"
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
