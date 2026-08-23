package fulfillment

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"io"
	"strings"
	"unicode/utf8"
)

type TokenCipher interface {
	Seal(context.Context, string) (uint16, []byte, error)
	Open(context.Context, uint16, []byte) (string, error)
}

type AESGCMTokenCipher struct {
	keys    map[uint16][]byte
	current uint16
	random  io.Reader
}

func NewAESGCMTokenCipher(keys map[uint16][]byte, current uint16, random io.Reader) (*AESGCMTokenCipher, error) {
	if len(keys) == 0 || current == 0 || random == nil {
		return nil, ErrTokenInvalid
	}
	copyKeys := make(map[uint16][]byte, len(keys))
	for version, key := range keys {
		if version == 0 || len(key) != 32 {
			return nil, ErrTokenInvalid
		}
		copyKeys[version] = append([]byte(nil), key...)
	}
	if _, ok := copyKeys[current]; !ok {
		return nil, ErrTokenInvalid
	}
	return &AESGCMTokenCipher{keys: copyKeys, current: current, random: random}, nil
}

func (value *AESGCMTokenCipher) Seal(ctx context.Context, token string) (uint16, []byte, error) {
	if err := ctx.Err(); err != nil || value == nil || !validPlainToken(token) {
		return 0, nil, ErrTokenInvalid
	}
	aead, err := value.aead(value.current)
	if err != nil {
		return 0, nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(value.random, nonce); err != nil {
		return 0, nil, ErrUnavailable
	}
	envelope := append([]byte(nil), nonce...)
	envelope = aead.Seal(envelope, nonce, []byte(token), nil)
	if len(envelope) > 192 {
		return 0, nil, ErrTokenInvalid
	}
	return value.current, envelope, nil
}

func (value *AESGCMTokenCipher) Open(ctx context.Context, version uint16, envelope []byte) (string, error) {
	if err := ctx.Err(); err != nil || value == nil {
		return "", ErrTokenInvalid
	}
	aead, err := value.aead(version)
	if err != nil || len(envelope) <= aead.NonceSize() || len(envelope) > 192 {
		return "", ErrTokenInvalid
	}
	nonce := envelope[:aead.NonceSize()]
	plaintext, err := aead.Open(nil, nonce, envelope[aead.NonceSize():], nil)
	if err != nil || !validPlainToken(string(plaintext)) {
		return "", ErrTokenInvalid
	}
	return string(plaintext), nil
}

func (value *AESGCMTokenCipher) aead(version uint16) (cipher.AEAD, error) {
	key, ok := value.keys[version]
	if !ok {
		return nil, ErrTokenInvalid
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	return aead, nil
}

func validPlainToken(token string) bool {
	return token != "" && len(token) <= 128 && utf8.ValidString(token) &&
		strings.TrimSpace(token) == token && !strings.ContainsAny(token, " \t\r\n")
}
