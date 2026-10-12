package objectstore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	MaxImageBytes = 5 * 1024 * 1024
	uploadURLTTL  = 10 * time.Minute
)

var (
	ErrIdempotencyConflict = errors.New("object upload idempotency conflict")
)

type Adapter interface {
	Put(context.Context, string, []byte, string) error
	PrepareUpload(context.Context, string, UploadSpecification, time.Duration) (UploadIntent, error)
	Read(context.Context, string, int64) (Image, error)
	PublicURL(context.Context, string) (string, error)
}
type ConfirmationStore interface {
	Record(context.Context, WriteMeta, UploadSpecification, string) error
	Confirmed(context.Context, []string) (map[string]bool, error)
}

type Service struct {
	adapter       Adapter
	confirmations ConfirmationStore
}

func NewService(adapter Adapter, stores ...ConfirmationStore) *Service {
	service := &Service{adapter: adapter}
	if len(stores) == 1 {
		service.confirmations = stores[0]
	}
	return service
}

type UploadSpecification struct {
	ContentType string
	SizeBytes   int64
	SHA256      string
}

type UploadIntent struct {
	ObjectKey string
	Method    string
	URL       string
	Headers   map[string]string
	ExpiresAt time.Time
}

func (s *Service) PrepareImageUpload(ctx context.Context, meta WriteMeta, specification UploadSpecification) (UploadIntent, error) {
	key, ok := objectKeyForUpload(meta, specification)
	if s == nil || s.adapter == nil || !ok {
		return UploadIntent{}, ErrInvalidImage
	}
	intent, err := s.adapter.PrepareUpload(ctx, key, specification, uploadURLTTL)
	if err != nil || intent.ObjectKey != key || intent.Method != http.MethodPut || intent.URL == "" || intent.ExpiresAt.IsZero() {
		return UploadIntent{}, ErrUnavailable
	}
	return intent, nil
}

func (s *Service) ConfirmImageUpload(ctx context.Context, meta WriteMeta, specification UploadSpecification, objectKey string) (StoredImage, error) {
	expectedKey, valid := objectKeyForUpload(meta, specification)
	if s == nil || s.adapter == nil || !valid || objectKey != expectedKey {
		return StoredImage{}, ErrInvalidImage
	}
	actual, err := s.adapter.Read(ctx, objectKey, MaxImageBytes+1)
	if err != nil {
		if errors.Is(err, ErrImageTooLarge) {
			return StoredImage{}, ErrImageTooLarge
		}
		return StoredImage{}, ErrUnavailable
	}
	if !matchesSpecification(actual, specification) {
		return StoredImage{}, ErrInvalidImage
	}
	if s.confirmations != nil {
		if err := s.confirmations.Record(ctx, meta, specification, objectKey); err != nil {
			return StoredImage{}, err
		}
	}
	readURL, err := s.adapter.PublicURL(ctx, objectKey)
	if err != nil || readURL == "" {
		return StoredImage{}, ErrUnavailable
	}
	return StoredImage{ObjectKey: objectKey, URL: readURL}, nil
}

func (s *Service) PutImage(ctx context.Context, _ WriteMeta, image Image) (StoredImage, error) {
	if s == nil || s.adapter == nil || len(image.Bytes) == 0 {
		return StoredImage{}, ErrInvalidImage
	}
	sum := sha256.Sum256(image.Bytes)
	specification := UploadSpecification{ContentType: image.ContentType, SizeBytes: int64(len(image.Bytes)), SHA256: hex.EncodeToString(sum[:])}
	key, ok := objectKeyForSpecification(specification)
	if !ok || !matchesSpecification(image, specification) {
		return StoredImage{}, ErrInvalidImage
	}
	if err := s.adapter.Put(ctx, key, image.Bytes, image.ContentType); err != nil {
		return StoredImage{}, ErrUnavailable
	}
	public, err := s.adapter.PublicURL(ctx, key)
	if err != nil || public == "" {
		return StoredImage{}, ErrUnavailable
	}
	return StoredImage{ObjectKey: key, URL: public}, nil
}

func objectKeyForSpecification(specification UploadSpecification) (string, bool) {
	if specification.SizeBytes <= 0 || specification.SizeBytes > MaxImageBytes || len(specification.SHA256) != sha256.Size*2 {
		return "", false
	}
	for index := range specification.SHA256 {
		character := specification.SHA256[index]
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return "", false
		}
	}
	extension := map[string]string{"image/png": ".png", "image/jpeg": ".jpg"}[specification.ContentType]
	if extension == "" {
		return "", false
	}
	return "images/" + specification.SHA256 + extension, true
}

func objectKeyForUpload(meta WriteMeta, specification UploadSpecification) (string, bool) {
	contentKey, ok := objectKeyForSpecification(specification)
	if !ok || meta.ActorUserID == 0 || strings.TrimSpace(meta.IdempotencyKey) == "" {
		return "", false
	}
	writeSum := sha256.Sum256([]byte(strconv.FormatUint(meta.ActorUserID, 10) + ":" + meta.IdempotencyKey))
	return "images/" + hex.EncodeToString(writeSum[:]) + "/" + strings.TrimPrefix(contentKey, "images/"), true
}

func uploadObjectKeyMatches(specification UploadSpecification, objectKey string) bool {
	contentKey, ok := objectKeyForSpecification(specification)
	if !ok {
		return false
	}
	prefix := "images/"
	if !strings.HasPrefix(objectKey, prefix) {
		return false
	}
	remainder := strings.TrimPrefix(objectKey, prefix)
	separator := strings.IndexByte(remainder, '/')
	if separator != sha256.Size*2 || remainder[separator+1:] != strings.TrimPrefix(contentKey, prefix) {
		return false
	}
	for index := 0; index < separator; index++ {
		character := remainder[index]
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func UploadObjectKeyMatchesContent(objectKey, contentType string, data []byte) bool {
	if len(data) == 0 || len(data) > MaxImageBytes {
		return false
	}
	sum := sha256.Sum256(data)
	specification := UploadSpecification{ContentType: contentType, SizeBytes: int64(len(data)), SHA256: hex.EncodeToString(sum[:])}
	return uploadObjectKeyMatches(specification, objectKey)
}

func matchesSpecification(actual Image, specification UploadSpecification) bool {
	if int64(len(actual.Bytes)) != specification.SizeBytes || int64(len(actual.Bytes)) > MaxImageBytes || actual.ContentType != specification.ContentType {
		return false
	}
	if detected := http.DetectContentType(actual.Bytes); detected != specification.ContentType {
		return false
	}
	if _, _, err := image.DecodeConfig(bytes.NewReader(actual.Bytes)); err != nil {
		return false
	}
	sum := sha256.Sum256(actual.Bytes)
	return hex.EncodeToString(sum[:]) == specification.SHA256
}
func (s *Service) PublicURL(ctx context.Context, key string) (string, error) {
	if s == nil || s.adapter == nil || !ValidImageObjectKey(key) {
		return "", ErrUnavailable
	}
	return s.adapter.PublicURL(ctx, key)
}

func (s *Service) Confirmed(ctx context.Context, keys []string) (map[string]bool, error) {
	if s == nil || s.adapter == nil || len(keys) == 0 {
		return nil, ErrUnavailable
	}
	if s.confirmations == nil {
		confirmed := make(map[string]bool, len(keys))
		for _, key := range keys {
			if !ValidImageObjectKey(key) {
				return nil, ErrUnavailable
			}
			confirmed[key] = true
		}
		return confirmed, nil
	}
	return s.confirmations.Confirmed(ctx, keys)
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
func (a *FileAdapter) PrepareUpload(_ context.Context, key string, specification UploadSpecification, ttl time.Duration) (UploadIntent, error) {
	if _, err := a.target(key); err != nil || ttl <= 0 {
		return UploadIntent{}, ErrUnavailable
	}
	return UploadIntent{
		ObjectKey: key,
		Method:    http.MethodPut,
		URL:       a.objectURL(key),
		Headers:   map[string]string{"Content-Type": specification.ContentType, "x-cos-forbid-overwrite": "true"},
		ExpiresAt: time.Now().UTC().Add(ttl),
	}, nil
}
func (a *FileAdapter) Read(_ context.Context, key string, limit int64) (Image, error) {
	target, err := a.target(key)
	if err != nil || limit <= 0 {
		return Image{}, ErrUnavailable
	}
	file, err := os.Open(target)
	if err != nil {
		return Image{}, ErrUnavailable
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return Image{}, ErrUnavailable
	}
	if int64(len(data)) > limit {
		return Image{}, ErrImageTooLarge
	}
	return Image{ContentType: http.DetectContentType(data), Bytes: data}, nil
}
func (a *FileAdapter) PublicURL(_ context.Context, key string) (string, error) {
	if _, err := a.target(key); err != nil {
		return "", err
	}
	return a.objectURL(key), nil
}
func (a *FileAdapter) objectURL(key string) string {
	parts := strings.Split(key, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return a.publicPrefix + "/" + strings.Join(parts, "/")
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
