package objectstore

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	cos "github.com/tencentyun/cos-go-sdk-v5"
)

// versionedImage binds the bytes and native version returned by one COS GET.
type versionedImage struct {
	Image
	VersionID string
}

func (adapter *COSAdapter) readVersioned(ctx context.Context, key, versionID string, maxBytes int64) (versionedImage, error) {
	if adapter == nil || adapter.client == nil || ctx == nil || ctx.Err() != nil || maxBytes <= 0 || maxBytes == math.MaxInt64 ||
		(versionID == "" && !ValidImageObjectKey(key)) ||
		(versionID != "" && (!validVersionedImageObjectKey(key) || !validNativeVersionID(versionID))) {
		return versionedImage{}, ErrUnavailable
	}
	ctx, responseLimit := versionedCOSContext(ctx)
	var response *cos.Response
	var err error
	if versionID == "" {
		response, err = adapter.client.Object.Get(ctx, key, nil)
	} else {
		// The SDK appends its variadic ID directly to the query string. Escape
		// the opaque version before passing it, so it remains one query value.
		response, err = adapter.client.Object.Get(ctx, key, nil, url.QueryEscape(versionID))
	}
	if err != nil || responseLimit.exceeded.Load() || response == nil || response.Response == nil || response.Body == nil || response.StatusCode != http.StatusOK {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return versionedImage{}, ErrUnavailable
	}
	defer response.Body.Close()
	actualVersion := response.Header.Get("x-cos-version-id")
	if !validNativeVersionID(actualVersion) || (versionID != "" && actualVersion != versionID) {
		return versionedImage{}, ErrUnavailable
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return versionedImage{}, ErrUnavailable
	}
	if int64(len(data)) > maxBytes {
		return versionedImage{}, ErrImageTooLarge
	}
	contentType := strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0])
	return versionedImage{Image: Image{ContentType: contentType, Bytes: data}, VersionID: actualVersion}, nil
}

func (adapter *COSAdapter) publishVersioned(ctx context.Context, key string, img Image) (string, error) {
	if adapter == nil || adapter.client == nil || ctx == nil || ctx.Err() != nil || !validPublishedImageObjectKey(key) {
		return "", ErrUnavailable
	}
	sum := sha256.Sum256(img.Bytes)
	specification := UploadSpecification{ContentType: img.ContentType, SizeBytes: int64(len(img.Bytes)), SHA256: hex.EncodeToString(sum[:])}
	contentKey, valid := objectKeyForSpecification(specification)
	if !valid || !strings.HasSuffix(key, "/"+strings.TrimPrefix(contentKey, "images/")) || !matchesSpecification(img, specification) {
		return "", ErrUnavailable
	}
	checksum := md5.Sum(img.Bytes)
	ctx, responseLimit := versionedCOSContext(ctx)
	response, err := adapter.client.Object.Put(ctx, key, bytes.NewReader(img.Bytes), &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentType: img.ContentType, ContentLength: int64(len(img.Bytes)), ContentMD5: base64.StdEncoding.EncodeToString(checksum[:]),
		},
	})
	if err != nil || responseLimit.exceeded.Load() || response == nil || response.Response == nil || response.StatusCode != http.StatusOK {
		return "", ErrUnavailable
	}
	versionID := response.Header.Get("x-cos-version-id")
	if !validNativeVersionID(versionID) {
		return "", ErrUnavailable
	}
	return versionID, nil
}

func (adapter *COSAdapter) versionURL(ctx context.Context, key, versionID string, ttl time.Duration, at time.Time) (string, error) {
	if adapter == nil || adapter.client == nil || ctx == nil || ctx.Err() != nil || !validVersionedImageObjectKey(key) ||
		!validNativeVersionID(versionID) || !validVersionedSigningTime(ttl, at) {
		return "", ErrUnavailable
	}
	query := url.Values{"versionId": []string{versionID}}
	signedURL, err := adapter.client.Object.GetPresignedURL2(ctx, http.MethodGet, key, ttl, &cos.PresignedURLOptions{
		Query: &query, AuthTime: versionedAuthTime(ttl, at),
	})
	if err != nil || signedURL == nil || signedURL.Scheme != "https" || signedURL.Host == "" {
		return "", ErrUnavailable
	}
	return signedURL.String(), nil
}

func (adapter *COSAdapter) prepareVersionedUpload(ctx context.Context, key string, specification UploadSpecification, ttl time.Duration, at time.Time) (UploadIntent, error) {
	if adapter == nil || adapter.client == nil || ctx == nil || ctx.Err() != nil || !ValidImageObjectKey(key) || !validVersionedSigningTime(ttl, at) {
		return UploadIntent{}, ErrUnavailable
	}
	contentKey, valid := objectKeyForSpecification(specification)
	if !valid || (key != contentKey && !uploadObjectKeyMatches(specification, key)) {
		return UploadIntent{}, ErrUnavailable
	}
	headers := http.Header{
		"Content-Type":           []string{specification.ContentType},
		"Content-Length":         []string{strconv.FormatInt(specification.SizeBytes, 10)},
		"x-cos-forbid-overwrite": []string{"true"},
	}
	signedURL, err := adapter.client.Object.GetPresignedURL2(ctx, http.MethodPut, key, ttl, &cos.PresignedURLOptions{
		Header: &headers, AuthTime: versionedAuthTime(ttl, at),
	})
	if err != nil || signedURL == nil || signedURL.Scheme != "https" || signedURL.Host == "" {
		return UploadIntent{}, ErrUnavailable
	}
	return UploadIntent{
		ObjectKey: key, Method: http.MethodPut, URL: signedURL.String(),
		Headers:   map[string]string{"Content-Type": specification.ContentType, "x-cos-forbid-overwrite": "true"},
		ExpiresAt: at.UTC().Add(ttl),
	}, nil
}

func validVersionedSigningTime(ttl time.Duration, at time.Time) bool {
	return ttl > 0 && ttl <= 15*time.Minute && !at.IsZero()
}

func versionedAuthTime(ttl time.Duration, at time.Time) *cos.AuthTime {
	at = at.UTC()
	return &cos.AuthTime{SignStartTime: at, KeyStartTime: at, SignEndTime: at.Add(ttl), KeyEndTime: at.Add(ttl)}
}

func validVersionedImageObjectKey(key string) bool {
	return ValidImageObjectKey(key) || validPublishedImageObjectKey(key)
}

func validPublishedImageObjectKey(key string) bool {
	const prefix = "images/published/"
	if !strings.HasPrefix(key, prefix) {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(key, prefix), "/")
	return len(parts) == 2 && lowerHexDigest(parts[0]) && ValidImageObjectKey("images/"+parts[1])
}

// COS exposes native IDs as opaque strings. Do not impose the sample IDs'
// alphabet or size on this internal locator; reject only absent/null or
// malformed values that cannot be a native response header value.
func validNativeVersionID(versionID string) bool {
	if versionID == "" || versionID == "null" || !utf8.ValidString(versionID) {
		return false
	}
	for _, character := range versionID {
		if unicode.IsSpace(character) || unicode.IsControl(character) {
			return false
		}
	}
	return true
}

const maxVersionedCOSResponseBytes int64 = 64 * 1024

type versionedCOSContextKey struct{}

type versionedCOSResponseLimit struct {
	exceeded atomic.Bool
}

func versionedCOSContext(ctx context.Context) (context.Context, *versionedCOSResponseLimit) {
	limit := &versionedCOSResponseLimit{}
	return context.WithValue(ctx, versionedCOSContextKey{}, limit), limit
}

// boundVersionedCOSResponse is invoked by the shared credential transport.
// COS's SDK drains PUT responses and reads error bodies before returning.
// Only versioned calls opt into this bound; successful image GETs are bounded
// by readVersioned instead.
func boundVersionedCOSResponse(request *http.Request, response *http.Response, err error) (*http.Response, error) {
	if request == nil {
		return response, err
	}
	limit, marked := request.Context().Value(versionedCOSContextKey{}).(*versionedCOSResponseLimit)
	if !marked || response == nil || response.Body == nil {
		return response, err
	}
	if err != nil {
		_ = response.Body.Close()
		return nil, ErrUnavailable
	}
	if request.Method == http.MethodGet && response.StatusCode == http.StatusOK {
		return response, nil
	}
	if response.ContentLength > maxVersionedCOSResponseBytes {
		limit.exceeded.Store(true)
		_ = response.Body.Close()
		return nil, ErrUnavailable
	}
	response.Body = &versionedCOSBoundedBody{ReadCloser: response.Body, remaining: maxVersionedCOSResponseBytes, limit: limit}
	return response, nil
}

type versionedCOSBoundedBody struct {
	io.ReadCloser
	remaining int64
	limit     *versionedCOSResponseLimit
}

func (body *versionedCOSBoundedBody) Read(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if body.limit.exceeded.Load() {
		return 0, ErrUnavailable
	}
	if int64(len(data)) > body.remaining+1 {
		data = data[:body.remaining+1]
	}
	count, err := body.ReadCloser.Read(data)
	if err != nil && err != io.EOF {
		body.limit.exceeded.Store(true)
		return count, err
	}
	if int64(count) > body.remaining {
		body.limit.exceeded.Store(true)
		return 0, ErrUnavailable
	}
	body.remaining -= int64(count)
	return count, err
}
