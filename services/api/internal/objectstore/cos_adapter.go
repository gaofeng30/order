package objectstore

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cos "github.com/tencentyun/cos-go-sdk-v5"
)

const cosRequestTimeout = 15 * time.Second

type COSConfig struct {
	Bucket string
	Region string
}

// COSAdapter uses the selected cloud credential for native COS operations.
// Each signing operation uses an atomic temporary credential snapshot.
type COSAdapter struct {
	client *cos.Client
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
	client := cos.NewClient(&cos.BaseURL{BucketURL: bucketURL}, &http.Client{
		Timeout: cosRequestTimeout,
		Transport: &cosCredentialTransport{
			credential: credential,
			transport:  transport,
		},
	})
	client.Conf.RetryOpt.Count = 0
	return &COSAdapter{client: client}, nil
}

type cosCredentialTransport struct {
	credential common.CredentialIface
	transport  http.RoundTripper
}

func (transport *cosCredentialTransport) GetCredential() (string, string, string, error) {
	id, key, token := transport.credential.GetCredential()
	if id == "" || key == "" || token == "" {
		return "", "", "", ErrUnavailable
	}
	return id, key, token, nil
}

func (transport *cosCredentialTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	id, key, token, err := transport.GetCredential()
	if err != nil {
		return nil, ErrUnavailable
	}
	signer := &cos.AuthorizationTransport{SecretID: id, SecretKey: key, SessionToken: token, Transport: transport.transport}
	response, err := signer.RoundTrip(request)
	return boundVersionedCOSResponse(request, response, err)
}

func (adapter *COSAdapter) Put(ctx context.Context, key string, data []byte, contentType string) error {
	if adapter == nil || adapter.client == nil || ctx == nil || !ValidImageObjectKey(key) || len(data) == 0 ||
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

func (adapter *COSAdapter) PrepareUpload(ctx context.Context, key string, specification UploadSpecification, ttl time.Duration) (UploadIntent, error) {
	if adapter == nil || adapter.client == nil || ctx == nil || !ValidImageObjectKey(key) || ttl <= 0 || ttl > 15*time.Minute {
		return UploadIntent{}, ErrUnavailable
	}
	headers := http.Header{
		"Content-Type":           []string{specification.ContentType},
		"Content-Length":         []string{strconv.FormatInt(specification.SizeBytes, 10)},
		"x-cos-forbid-overwrite": []string{"true"},
	}
	signedURL, err := adapter.client.Object.GetPresignedURL2(ctx, http.MethodPut, key, ttl, &cos.PresignedURLOptions{Header: &headers})
	if err != nil || signedURL == nil || signedURL.Scheme != "https" || signedURL.Host == "" {
		return UploadIntent{}, ErrUnavailable
	}
	return UploadIntent{
		ObjectKey: key,
		Method:    http.MethodPut,
		URL:       signedURL.String(),
		Headers:   map[string]string{"Content-Type": specification.ContentType, "x-cos-forbid-overwrite": "true"},
		ExpiresAt: time.Now().UTC().Add(ttl),
	}, nil
}

func (adapter *COSAdapter) Read(ctx context.Context, key string, limit int64) (Image, error) {
	if adapter == nil || adapter.client == nil || ctx == nil || !ValidImageObjectKey(key) || limit <= 0 {
		return Image{}, ErrUnavailable
	}
	response, err := adapter.client.Object.Get(ctx, key, nil)
	if err != nil || response == nil || response.Response == nil || response.Body == nil || response.StatusCode != http.StatusOK {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return Image{}, ErrUnavailable
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return Image{}, ErrUnavailable
	}
	if int64(len(data)) > limit {
		return Image{}, ErrImageTooLarge
	}
	contentType := strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0])
	return Image{ContentType: contentType, Bytes: data}, nil
}

func (adapter *COSAdapter) PublicURL(ctx context.Context, key string) (string, error) {
	if adapter == nil || adapter.client == nil || ctx == nil || !ValidImageObjectKey(key) {
		return "", ErrUnavailable
	}
	signedURL, err := adapter.client.Object.GetPresignedURL2(ctx, http.MethodGet, key, uploadURLTTL, nil)
	if err != nil || signedURL == nil || signedURL.Scheme != "https" || signedURL.Host == "" {
		return "", ErrUnavailable
	}
	return signedURL.String(), nil
}

func ValidImageObjectKey(key string) bool {
	const prefix = "images/"
	if !strings.HasPrefix(key, prefix) {
		return false
	}
	remainder := strings.TrimPrefix(key, prefix)
	parts := strings.Split(remainder, "/")
	if len(parts) == 2 {
		if !lowerHexDigest(parts[0]) {
			return false
		}
		remainder = parts[1]
	} else if len(parts) != 1 {
		return false
	}
	if len(remainder) != 64+4 {
		return false
	}
	digest := remainder[:64]
	extension := remainder[64:]
	if extension != ".png" && extension != ".jpg" {
		return false
	}
	return lowerHexDigest(digest)
}

func lowerHexDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for index := 0; index < len(value); index++ {
		if (value[index] < '0' || value[index] > '9') && (value[index] < 'a' || value[index] > 'f') {
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
