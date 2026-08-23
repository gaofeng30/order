package objectstore

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	ErrInvalidImage  = errors.New("invalid image")
	ErrImageTooLarge = errors.New("image too large")
	ErrUnavailable   = errors.New("object store unavailable")
)

type WriteMeta struct {
	ActorUserID               uint64
	IdempotencyKey, RequestID string
}
type Image struct {
	Name, ContentType string
	Bytes             []byte
}
type StoredImage struct{ ObjectKey, URL string }

// Application is the only HTTP-facing seam; production and deterministic fake adapters stay behind it.
type Application interface {
	PutImage(context.Context, WriteMeta, Image) (StoredImage, error)
}
type Handler struct{ app Application }

func NewHandler(app Application) *Handler              { return &Handler{app: app} }
func (h *Handler) RegisterRoutes(api *gin.RouterGroup) { api.POST("/upload", h.upload) }
func (h *Handler) upload(c *gin.Context) {
	meta, ok := meta(c)
	if !ok {
		writeError(c, ErrInvalidImage)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 5*1024*1024+4096)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		writeError(c, ErrInvalidImage)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 5*1024*1024+1))
	if err != nil {
		writeError(c, ErrInvalidImage)
		return
	}
	if len(data) > 5*1024*1024 {
		writeError(c, ErrImageTooLarge)
		return
	}
	kind := http.DetectContentType(data)
	if kind != "image/png" && kind != "image/jpeg" {
		writeError(c, ErrInvalidImage)
		return
	}
	if _, _, err := image.DecodeConfig(bytes.NewReader(data)); err != nil {
		writeError(c, ErrInvalidImage)
		return
	}
	stored, err := h.app.PutImage(c.Request.Context(), meta, Image{Name: header.Filename, ContentType: kind, Bytes: data})
	if err != nil {
		writeError(c, err)
		return
	}
	if stored.ObjectKey == "" || stored.URL == "" {
		writeError(c, ErrUnavailable)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"image": gin.H{"object_key": stored.ObjectKey, "url": stored.URL}})
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
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
