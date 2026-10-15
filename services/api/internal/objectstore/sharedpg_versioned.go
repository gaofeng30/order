package objectstore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"time"
)

// SharedPGVersionedService is the UAT COS/shared-PG image composition.
type SharedPGVersionedService struct {
	adapter   *COSAdapter
	client    sharedPGExecutor
	now       func() time.Time
	entropy   io.Reader
	immutable bool
}

func NewSharedPGVersionedService(adapter *COSAdapter, client sharedPGExecutor) *SharedPGVersionedService {
	return &SharedPGVersionedService{adapter: adapter, client: client, now: time.Now, entropy: rand.Reader}
}

// NewSharedPGImmutableService is the production composition for a bucket that
// has never enabled versioning. Signed uploads prohibit replacing an object.
// Existing native-version receipts remain readable; new receipts use the original key.
func NewSharedPGImmutableService(adapter *COSAdapter, client sharedPGExecutor) *SharedPGVersionedService {
	s := NewSharedPGVersionedService(adapter, client)
	s.immutable = true
	return s
}

func (s *SharedPGVersionedService) available(ctx context.Context) bool {
	return s != nil && s.adapter != nil && s.client != nil && s.now != nil && s.entropy != nil && ctx != nil && ctx.Err() == nil
}

func (s *SharedPGVersionedService) PrepareImageUpload(ctx context.Context, meta WriteMeta, spec UploadSpecification) (UploadIntent, error) {
	key, ok := objectKeyForUpload(meta, spec)
	if !ok {
		return UploadIntent{}, ErrInvalidImage
	}
	if !s.available(ctx) {
		return UploadIntent{}, ErrUnavailable
	}
	return s.adapter.prepareVersionedUpload(ctx, key, spec, uploadURLTTL, s.now().UTC())
}

func (s *SharedPGVersionedService) ConfirmImageUpload(ctx context.Context, meta WriteMeta, spec UploadSpecification, key string) (StoredImage, error) {
	expected, ok := objectKeyForUpload(meta, spec)
	if !ok || expected != key || !uploadObjectKeyMatches(spec, key) {
		return StoredImage{}, ErrInvalidImage
	}
	if !s.available(ctx) {
		return StoredImage{}, ErrUnavailable
	}
	digest, err := imageRequestDigest(spec, key)
	if err != nil {
		return StoredImage{}, err
	}
	replay, err := s.replayPin(ctx, meta)
	if err != nil {
		return StoredImage{}, err
	}
	if replay.Outcome == "SUCCEEDED" {
		// Preserve actual-image validation before the original receipt conflict.
		if replay.Response.ObjectKey == key {
			if _, err := s.readPin(ctx, *replay.Response, &spec); err != nil {
				return StoredImage{}, err
			}
		} else if _, err := s.readUpload(ctx, key, spec); err != nil {
			return StoredImage{}, err
		}
		if replay.RequestDigest != digest || replay.Response.ObjectKey != key {
			return StoredImage{}, ErrIdempotencyConflict
		}
		return s.storedPinURL(ctx, *replay.Response)
	}
	img, err := s.readUpload(ctx, key, spec)
	if err != nil {
		return StoredImage{}, err
	}
	pin := imageVersionPin{ObjectKey: key, PublishedKey: key}
	if !s.immutable {
		var nonce [32]byte
		if _, err := io.ReadFull(s.entropy, nonce[:]); err != nil {
			return StoredImage{}, ErrUnavailable
		}
		publishedKey := "images/published/" + hex.EncodeToString(nonce[:]) + "/" + key[strings.LastIndexByte(key, '/')+1:]
		version, err := s.adapter.publishVersioned(ctx, publishedKey, img.Image)
		if err != nil {
			return StoredImage{}, ErrUnavailable
		}
		pin = imageVersionPin{ObjectKey: key, PublishedKey: publishedKey, VersionID: version}
	}
	committed, err := s.commitPin(ctx, meta, key, digest, pin, s.now().UTC())
	if err != nil {
		if errors.Is(err, ErrIdempotencyConflict) {
			return StoredImage{}, err
		}
		// A response can be lost after commit. Read the exact original receipt
		// before deciding failure; never retry publication or business writes.
		committed, err = s.replayPin(ctx, meta)
		if err != nil || committed.Outcome != "SUCCEEDED" || committed.RequestDigest != digest || committed.Response.ObjectKey != key {
			return StoredImage{}, ErrUnavailable
		}
	}
	if _, err := s.readPin(ctx, *committed.Response, &spec); err != nil {
		return StoredImage{}, err
	}
	return s.storedPinURL(ctx, *committed.Response)
}

func (s *SharedPGVersionedService) readUpload(ctx context.Context, key string, spec UploadSpecification) (versionedImage, error) {
	var img versionedImage
	var err error
	if s.immutable {
		img.Image, err = s.adapter.Read(ctx, key, MaxImageBytes+1)
	} else {
		img, err = s.adapter.readVersioned(ctx, key, "", MaxImageBytes+1)
	}
	if err != nil {
		return versionedImage{}, err
	}
	if !matchesSpecification(img.Image, spec) {
		return versionedImage{}, ErrInvalidImage
	}
	return img, nil
}

func (s *SharedPGVersionedService) readPin(ctx context.Context, pin imageVersionPin, expected *UploadSpecification) (versionedImage, error) {
	if !validImageVersionPin(pin) {
		return versionedImage{}, ErrUnavailable
	}
	var img versionedImage
	var err error
	if pin.VersionID == "" {
		img.Image, err = s.adapter.Read(ctx, pin.ObjectKey, MaxImageBytes)
	} else {
		img, err = s.adapter.readVersioned(ctx, pin.PublishedKey, pin.VersionID, MaxImageBytes)
	}
	if err != nil {
		return versionedImage{}, ErrUnavailable
	}
	suffix := pin.ObjectKey[strings.LastIndexByte(pin.ObjectKey, '/')+1:]
	contentType := "image/png"
	if strings.HasSuffix(suffix, ".jpg") {
		contentType = "image/jpeg"
	}
	storedSpec := UploadSpecification{ContentType: contentType, SizeBytes: int64(len(img.Bytes)), SHA256: suffix[:64]}
	if !matchesSpecification(img.Image, storedSpec) {
		return versionedImage{}, ErrUnavailable
	}
	if expected != nil && !matchesSpecification(img.Image, *expected) {
		return versionedImage{}, ErrInvalidImage
	}
	return img, nil
}

func (s *SharedPGVersionedService) storedPinURL(ctx context.Context, pin imageVersionPin) (StoredImage, error) {
	rawURL, err := s.adapter.versionURL(ctx, pin.PublishedKey, pin.VersionID, uploadURLTTL, s.now().UTC())
	if err != nil {
		return StoredImage{}, ErrUnavailable
	}
	return StoredImage{ObjectKey: pin.ObjectKey, URL: rawURL}, nil
}

// PublicURLs resolves confirmed immutable versions once per bounded batch.
// Browsing never downloads object bytes; upload and write confirmation still do.
func (s *SharedPGVersionedService) PublicURLs(ctx context.Context, keys []string) (map[string]string, error) {
	urls := make(map[string]string, len(keys))
	unique := make([]string, 0, len(keys))
	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		if !seen[key] {
			seen[key] = true
			unique = append(unique, key)
		}
	}
	for start := 0; start < len(unique); start += 128 {
		end := start + 128
		if end > len(unique) {
			end = len(unique)
		}
		pins, err := s.imagePins(ctx, unique[start:end])
		if err != nil {
			return nil, err
		}
		for key, pin := range pins {
			stored, err := s.storedPinURL(ctx, pin)
			if err != nil {
				return nil, err
			}
			urls[key] = stored.URL
		}
	}
	return urls, nil
}
func (s *SharedPGVersionedService) PublicURL(ctx context.Context, key string) (string, error) {
	urls, err := s.PublicURLs(ctx, []string{key})
	if err != nil || urls[key] == "" {
		return "", ErrUnavailable
	}
	return urls[key], nil
}
func (s *SharedPGVersionedService) Confirmed(ctx context.Context, keys []string) (map[string]bool, error) {
	if len(keys) == 0 || len(keys) > 3 {
		return nil, ErrUnavailable
	}
	pins, err := s.imagePins(ctx, keys)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(pins))
	for key, pin := range pins {
		if _, err := s.readPin(ctx, pin, nil); err != nil {
			return nil, ErrUnavailable
		}
		result[key] = true
	}
	return result, nil
}
