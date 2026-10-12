package httpdto

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"
)

var (
	ErrInvalidBearerToken    = errors.New("invalid_bearer_token")
	ErrInvalidIdempotencyKey = errors.New("invalid_idempotency_key")
)

// BearerToken returns the one exact bearer credential accepted by authenticated routes.
func BearerToken(request *http.Request) (string, error) {
	if request == nil {
		return "", ErrInvalidBearerToken
	}
	values := request.Header.Values("Authorization")
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") {
		return "", ErrInvalidBearerToken
	}
	token := strings.TrimPrefix(values[0], "Bearer ")
	if token == "" || !utf8.ValidString(token) || strings.ContainsAny(token, " \t\r\n,") {
		return "", ErrInvalidBearerToken
	}
	return token, nil
}

// IdempotencyKey returns the one unambiguous client operation key required by business writes.
func IdempotencyKey(request *http.Request) (string, error) {
	if request == nil {
		return "", ErrInvalidIdempotencyKey
	}
	values := request.Header.Values("Idempotency-Key")
	if len(values) != 1 {
		return "", ErrInvalidIdempotencyKey
	}
	value := values[0]
	if value == "" || !utf8.ValidString(value) || strings.TrimSpace(value) != value || strings.Contains(value, ",") {
		return "", ErrInvalidIdempotencyKey
	}
	for index := 0; index < len(value); index++ {
		if value[index] < 0x20 || value[index] == 0x7f {
			return "", ErrInvalidIdempotencyKey
		}
	}
	return value, nil
}
