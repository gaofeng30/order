package main

import (
	"errors"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type localObjectRoutes struct{ root string }

func newLocalObjectRoutes(root string) (*localObjectRoutes, error) {
	root = filepath.Clean(root)
	if root == "." || root == string(filepath.Separator) || !filepath.IsAbs(root) {
		return nil, errors.New("invalid local object root")
	}
	return &localObjectRoutes{root: root}, nil
}

func (routes *localObjectRoutes) RegisterRoutes(engine *gin.Engine) {
	if routes == nil || engine == nil || routes.root == "" {
		return
	}
	engine.StaticFS("/api/v1/objects", gin.Dir(routes.root, false))
}
