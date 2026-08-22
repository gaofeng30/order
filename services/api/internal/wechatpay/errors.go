package wechatpay

import "encoding/json"

// ErrorKind is a stable failure category that never includes provider payloads.
type ErrorKind string

const (
	ErrorInvalidConfig       ErrorKind = "INVALID_CONFIG"
	ErrorTransport           ErrorKind = "TRANSPORT"
	ErrorTimeout             ErrorKind = "TIMEOUT"
	ErrorRateLimited         ErrorKind = "RATE_LIMITED"
	ErrorProviderUnavailable ErrorKind = "PROVIDER_UNAVAILABLE"
	ErrorProviderRejected    ErrorKind = "PROVIDER_REJECTED"
	ErrorUnknownSerial       ErrorKind = "UNKNOWN_SERIAL"
	ErrorTimestamp           ErrorKind = "TIMESTAMP_INVALID"
	ErrorSignature           ErrorKind = "SIGNATURE_INVALID"
	ErrorDecryption          ErrorKind = "DECRYPTION_FAILED"
	ErrorProtocol            ErrorKind = "PROTOCOL_INVALID"
)

// Error exposes only stable, non-sensitive provider failure metadata.
type Error struct {
	kind         ErrorKind
	statusCode   int
	providerCode string
}

func (err *Error) Error() string { return "wechatpay: " + string(err.kind) }

// Kind returns the stable failure category.
func (err *Error) Kind() ErrorKind { return err.kind }

// StatusCode returns a provider HTTP status, or zero before a response exists.
func (err *Error) StatusCode() int { return err.statusCode }

// ProviderCode returns a provider code only when it was safe to parse.
func (err *Error) ProviderCode() string { return err.providerCode }

// Retryable reports whether the provider fact remains unknown and may be retried unchanged.
func (err *Error) Retryable() bool {
	switch err.kind {
	case ErrorTransport, ErrorTimeout, ErrorRateLimited, ErrorProviderUnavailable,
		ErrorUnknownSerial, ErrorTimestamp, ErrorSignature:
		return true
	default:
		return false
	}
}

func safeProviderCode(body []byte) string {
	if rejectDuplicateJSONKeys(body) != nil {
		return ""
	}
	var envelope struct {
		Code string `json:"code"`
	}
	if json.Unmarshal(body, &envelope) != nil || len(envelope.Code) == 0 || len(envelope.Code) > 64 {
		return ""
	}
	for _, character := range envelope.Code {
		if !(character >= 'A' && character <= 'Z') && !(character >= '0' && character <= '9') &&
			character != '_' && character != '-' && character != '.' {
			return ""
		}
	}
	return envelope.Code
}
