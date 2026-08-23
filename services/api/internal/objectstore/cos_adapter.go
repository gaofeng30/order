package objectstore

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cos "github.com/tencentyun/cos-go-sdk-v5"
)

const cosRequestTimeout = 15 * time.Second

type COSConfig struct {
	Bucket       string
	Region       string
	PublicOrigin string
}

// COSAdapter is the production object-store adapter. Its credential is
// supplied by the CVM role composition seam and refreshed by the official
// Tencent credential implementation.
type COSAdapter struct {
	client       *cos.Client
	publicOrigin string
}

func NewCOSAdapter(configuration COSConfig, credential common.CredentialIface) (*COSAdapter, error) {
	return newCOSAdapter(configuration, credential, nil)
}

func newCOSAdapter(configuration COSConfig, credential common.CredentialIface, transport http.RoundTripper) (*COSAdapter, error) {
	if credential == nil || !validCOSBucket(configuration.Bucket) || !validCOSRegion(configuration.Region) {
		return nil, ErrUnavailable
	}
	bucketURL, err := cos.NewBucketURL(configuration.Bucket, configuration.Region, true)
	if err != nil || bucketURL == nil {
		return nil, ErrUnavailable
	}
	publicOrigin, err := url.ParseRequestURI(configuration.PublicOrigin)
	if err != nil || publicOrigin.Scheme != "https" || publicOrigin.Hostname() == "" || publicOrigin.Port() != "" ||
		publicOrigin.User != nil || (publicOrigin.Path != "" && publicOrigin.Path != "/") || publicOrigin.RawQuery != "" || publicOrigin.Fragment != "" {
		return nil, ErrUnavailable
	}
	client := cos.NewClient(&cos.BaseURL{BucketURL: bucketURL}, &http.Client{
		Timeout: cosRequestTimeout,
		Transport: &cos.CredentialTransport{
			Credential: credential,
			Transport:  transport,
		},
	})
	client.Conf.RetryOpt.Count = 0
	return &COSAdapter{client: client, publicOrigin: strings.TrimRight(configuration.PublicOrigin, "/")}, nil
}

func (adapter *COSAdapter) Put(ctx context.Context, key string, data []byte, contentType string) error {
	if adapter == nil || adapter.client == nil || ctx == nil || !validImageObjectKey(key) || len(data) == 0 ||
		(contentType != "image/png" && contentType != "image/jpeg") {
		return ErrUnavailable
	}
	_, err := adapter.client.Object.Put(ctx, key, bytes.NewReader(data), &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{ContentType: contentType},
	})
	if err != nil {
		return ErrUnavailable
	}
	return nil
}

func (adapter *COSAdapter) PublicURL(ctx context.Context, key string) (string, error) {
	if adapter == nil || ctx == nil || !validImageObjectKey(key) {
		return "", ErrUnavailable
	}
	select {
	case <-ctx.Done():
		return "", ErrUnavailable
	default:
	}
	return adapter.publicOrigin + "/" + key, nil
}

func validImageObjectKey(key string) bool {
	const prefix = "images/"
	if !strings.HasPrefix(key, prefix) || len(key) != len(prefix)+64+4 {
		return false
	}
	digest := key[len(prefix) : len(prefix)+64]
	extension := key[len(prefix)+64:]
	if extension != ".png" && extension != ".jpg" {
		return false
	}
	for index := 0; index < len(digest); index++ {
		if (digest[index] < '0' || digest[index] > '9') && (digest[index] < 'a' || digest[index] > 'f') {
			return false
		}
	}
	return true
}

func validCOSBucket(value string) bool {
	if len(value) < 7 || len(value) > 63 || value[0] == '-' || value[len(value)-1] == '-' {
		return false
	}
	separator := strings.LastIndexByte(value, '-')
	if separator < 1 || separator == len(value)-1 {
		return false
	}
	appID := value[separator+1:]
	if len(appID) < 5 || len(appID) > 20 || appID[0] == '0' {
		return false
	}
	for index := 0; index < len(value); index++ {
		character := value[index]
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
			return false
		}
	}
	for index := 0; index < len(appID); index++ {
		if appID[index] < '0' || appID[index] > '9' {
			return false
		}
	}
	return true
}

func validCOSRegion(value string) bool {
	if len(value) < len("ap-a") || len(value) > 64 || !strings.HasPrefix(value, "ap-") || value[len(value)-1] == '-' {
		return false
	}
	for index := 0; index < len(value); index++ {
		character := value[index]
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
			return false
		}
	}
	return true
}
