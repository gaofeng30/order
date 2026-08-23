package objectstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Adapter interface {
	Put(context.Context, string, []byte, string) error
	PublicURL(context.Context, string) (string, error)
}
type Service struct{ adapter Adapter }

func NewService(adapter Adapter) *Service { return &Service{adapter: adapter} }
func (s *Service) PutImage(ctx context.Context, _ WriteMeta, image Image) (StoredImage, error) {
	if s.adapter == nil || len(image.Bytes) == 0 {
		return StoredImage{}, ErrInvalidImage
	}
	sum := sha256.Sum256(image.Bytes)
	ext := map[string]string{"image/png": ".png", "image/jpeg": ".jpg"}[image.ContentType]
	if ext == "" {
		return StoredImage{}, ErrInvalidImage
	}
	key := "images/" + hex.EncodeToString(sum[:]) + ext
	if err := s.adapter.Put(ctx, key, image.Bytes, image.ContentType); err != nil {
		return StoredImage{}, ErrUnavailable
	}
	public, err := s.adapter.PublicURL(ctx, key)
	if err != nil || public == "" {
		return StoredImage{}, ErrUnavailable
	}
	return StoredImage{ObjectKey: key, URL: public}, nil
}
func (s *Service) PublicURL(ctx context.Context, key string) (string, error) {
	if s.adapter == nil || key == "" {
		return "", ErrUnavailable
	}
	return s.adapter.PublicURL(ctx, key)
}

// FileAdapter is the deterministic local provider. It is not the production object-store adapter.
type FileAdapter struct{ root, publicPrefix string }

func NewFileAdapter(root, publicPrefix string) (*FileAdapter, error) {
	root = filepath.Clean(root)
	if root == "." || !filepath.IsAbs(root) || strings.TrimSpace(publicPrefix) == "" {
		return nil, errors.New("invalid file object store configuration")
	}
	return &FileAdapter{root: root, publicPrefix: strings.TrimRight(publicPrefix, "/")}, nil
}
func (a *FileAdapter) Put(_ context.Context, key string, data []byte, _ string) error {
	target, err := a.target(key)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}
	if existing, readErr := os.ReadFile(target); readErr == nil {
		return nilIfEqual(existing, data)
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return readErr
	}
	temp, err := os.CreateTemp(filepath.Dir(target), ".upload-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if err = temp.Chmod(0o640); err == nil {
		_, err = temp.Write(data)
	}
	closeErr := temp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, target)
}
func (a *FileAdapter) PublicURL(_ context.Context, key string) (string, error) {
	if _, err := a.target(key); err != nil {
		return "", err
	}
	parts := strings.Split(key, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return a.publicPrefix + "/" + strings.Join(parts, "/"), nil
}
func (a *FileAdapter) target(key string) (string, error) {
	if key == "" || strings.Contains(key, "\\") || filepath.IsAbs(key) {
		return "", errors.New("invalid object key")
	}
	target := filepath.Clean(filepath.Join(a.root, key))
	prefix := a.root + string(os.PathSeparator)
	if !strings.HasPrefix(target, prefix) {
		return "", errors.New("invalid object key")
	}
	return target, nil
}
func nilIfEqual(a, b []byte) error {
	if string(a) == string(b) {
		return nil
	}
	return fmt.Errorf("object key content mismatch")
}
