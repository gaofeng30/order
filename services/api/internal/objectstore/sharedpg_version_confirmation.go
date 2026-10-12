package objectstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// imageVersionPin is internal evidence, never a public upload DTO or URL.
type imageVersionPin struct {
	ObjectKey    string `json:"object_key"`
	PublishedKey string `json:"published_key"`
	VersionID    string `json:"version_id"`
}

type imagePinReceiptFacts struct {
	Outcome       string           `json:"outcome"`
	RequestDigest string           `json:"request_digest"`
	Response      *imageVersionPin `json:"response"`
}

func imageOperationHash(value string) string {
	sum := sha256.Sum256([]byte("operation:" + value))
	return hex.EncodeToString(sum[:])
}
func imageRequestHash(value string) any {
	if value == "" {
		return nil
	}
	sum := sha256.Sum256([]byte("request:" + value))
	return hex.EncodeToString(sum[:])
}
func imageRequestDigest(spec UploadSpecification, key string) (string, error) {
	raw, err := json.Marshal(imageConfirmationRequest{spec.ContentType, spec.SizeBytes, spec.SHA256, key})
	if err != nil {
		return "", ErrUnavailable
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func validImageVersionPin(pin imageVersionPin) bool {
	if !ValidImageObjectKey(pin.ObjectKey) || !validPublishedImageObjectKey(pin.PublishedKey) || !validNativeVersionID(pin.VersionID) {
		return false
	}
	return pin.ObjectKey[strings.LastIndexByte(pin.ObjectKey, '/')+1:] == pin.PublishedKey[strings.LastIndexByte(pin.PublishedKey, '/')+1:]
}

func (s *SharedPGVersionedService) replayPin(ctx context.Context, meta WriteMeta) (imagePinReceiptFacts, error) {
	var f imagePinReceiptFacts
	if s.client.CallRPC(ctx, "order_image_confirmation_replay_v1", map[string]any{"p_actor": strconv.FormatUint(meta.ActorUserID, 10), "p_key_hash": imageOperationHash(meta.IdempotencyKey)}, &f) != nil {
		return f, ErrUnavailable
	}
	if f.Outcome == "ABSENT" && f.RequestDigest == "" && f.Response == nil {
		return f, nil
	}
	if f.Outcome != "SUCCEEDED" || !lowerHexDigest(f.RequestDigest) || f.Response == nil || !validImageVersionPin(*f.Response) {
		return imagePinReceiptFacts{}, ErrUnavailable
	}
	return f, nil
}

func (s *SharedPGVersionedService) commitPin(ctx context.Context, meta WriteMeta, key, digest string, pin imageVersionPin, at time.Time) (imagePinReceiptFacts, error) {
	var f imagePinReceiptFacts
	if !validImageVersionPin(pin) || pin.ObjectKey != key || !lowerHexDigest(digest) || at.IsZero() {
		return f, ErrUnavailable
	}
	err := s.client.CallRPC(ctx, "order_image_confirmation_commit_v1", map[string]any{
		"p_actor": strconv.FormatUint(meta.ActorUserID, 10), "p_object_key": key, "p_key_hash": imageOperationHash(meta.IdempotencyKey),
		"p_request_hash": imageRequestHash(meta.RequestID), "p_request_digest": digest, "p_published_key": pin.PublishedKey, "p_version_id": pin.VersionID,
		"p_at": at.UTC().Format(time.RFC3339Nano),
	}, &f)
	if err != nil {
		return f, ErrUnavailable
	}
	if f.Outcome == "CONFLICT" {
		return f, ErrIdempotencyConflict
	}
	if f.Outcome != "SUCCEEDED" || f.RequestDigest != digest || f.Response == nil || f.Response.ObjectKey != key || !validImageVersionPin(*f.Response) {
		return imagePinReceiptFacts{}, ErrUnavailable
	}
	return f, nil
}

func (s *SharedPGVersionedService) imagePins(ctx context.Context, keys []string) (map[string]imageVersionPin, error) {
	if !s.available(ctx) || len(keys) == 0 || len(keys) > 3 {
		return nil, ErrUnavailable
	}
	requested := make(map[string]bool, len(keys))
	for _, key := range keys {
		if !ValidImageObjectKey(key) {
			return nil, ErrUnavailable
		}
		requested[key] = true
	}
	var f struct {
		Outcome  string            `json:"outcome"`
		Versions []imageVersionPin `json:"versions"`
	}
	if s.client.CallRPC(ctx, "order_image_versions_v1", map[string]any{"p_object_keys": keys}, &f) != nil || f.Outcome != "SUCCEEDED" || f.Versions == nil || len(f.Versions) > len(requested) {
		return nil, ErrUnavailable
	}
	result := make(map[string]imageVersionPin, len(f.Versions))
	for _, pin := range f.Versions {
		_, duplicate := result[pin.ObjectKey]
		if duplicate || !requested[pin.ObjectKey] || !validImageVersionPin(pin) {
			return nil, ErrUnavailable
		}
		result[pin.ObjectKey] = pin
	}
	return result, nil
}
