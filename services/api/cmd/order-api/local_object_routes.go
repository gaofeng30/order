package main

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gaofeng30/order/services/api/internal/objectstore"
	"github.com/gin-gonic/gin"
)

type localObjectRoutes struct {
	root    string
	adapter *objectstore.FileAdapter
}

func newLocalObjectRoutes(root string) (*localObjectRoutes, error) {
	root = filepath.Clean(root)
	if root == "." || root == string(filepath.Separator) || !filepath.IsAbs(root) {
		return nil, errors.New("invalid local object root")
	}
	adapter, err := objectstore.NewFileAdapter(root, "/api/v1/objects")
	if err != nil {
		return nil, err
	}
	return &localObjectRoutes{root: root, adapter: adapter}, nil
}

func (routes *localObjectRoutes) RegisterRoutes(engine *gin.Engine) {
	if routes == nil || engine == nil || routes.root == "" {
		return
	}
	engine.PUT("/api/v1/objects/*key", routes.put)
	engine.StaticFS("/api/v1/objects", gin.Dir(routes.root, false))
}

func (routes *localObjectRoutes) put(ctx *gin.Context) {
	key := strings.TrimPrefix(ctx.Param("key"), "/")
	contentType := ctx.GetHeader("Content-Type")
	if (contentType != "image/png" && contentType != "image/jpeg") || ctx.GetHeader("x-cos-forbid-overwrite") != "true" {
		ctx.Status(http.StatusUnprocessableEntity)
		return
	}
	data, err := io.ReadAll(io.LimitReader(ctx.Request.Body, objectstore.MaxImageBytes+1))
	if err != nil || len(data) == 0 {
		ctx.Status(http.StatusUnprocessableEntity)
		return
	}
	if len(data) > objectstore.MaxImageBytes {
		ctx.Status(http.StatusRequestEntityTooLarge)
		return
	}
	if !objectstore.UploadObjectKeyMatchesContent(key, contentType, data) || http.DetectContentType(data) != contentType {
		ctx.Status(http.StatusUnprocessableEntity)
		return
	}
	if err := routes.adapter.Put(ctx.Request.Context(), key, data, contentType); err != nil {
		ctx.Status(http.StatusServiceUnavailable)
		return
	}
	ctx.Status(http.StatusNoContent)
}
