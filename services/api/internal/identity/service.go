package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"time"

	"github.com/gaofeng30/order/services/api/internal/wechat"
)

const (
	tokenEntropyBytes = 32
	sessionTTL        = 24 * time.Hour
)

var (
	// ErrLoginRejected is safe for the public rejected-code response.
	ErrLoginRejected = errors.New("miniprogram login rejected")
	// ErrUnavailable hides provider, entropy, and persistence details.
	ErrUnavailable = errors.New("session unavailable")
	// ErrUnauthenticated identifies an absent, malformed, or expired session.
	ErrUnauthenticated = errors.New("session unauthenticated")
	// ErrSessionNotFound is the repository's non-sensitive absent-session result.
	ErrSessionNotFound = errors.New("session not found")
)

// CodeExchanger returns only the opaque Mini Program user identifier.
type CodeExchanger interface {
	Exchange(context.Context, string) (string, error)
}

// SessionStore is the atomic persistence boundary used by Service.
type SessionStore interface {
	CreateSession(context.Context, CreateSessionParams) error
	FindActiveUser(context.Context, [sha256.Size]byte, time.Time) (uint64, error)
}

// CreateSessionParams contains no raw token or provider credential.
type CreateSessionParams struct {
	OpenID    string
	TokenHash [sha256.Size]byte
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// IssuedSession is the one-time public success result.
type IssuedSession struct {
	AccessToken string
	ExpiresAt   time.Time
}

// Service exchanges provider codes and creates application sessions.
type Service struct {
	exchanger CodeExchanger
	store     SessionStore
	now       func() time.Time
	random    io.Reader
}

// NewService constructs the runtime service with the production clock and entropy source.
func NewService(exchanger CodeExchanger, store SessionStore) *Service {
	return newService(exchanger, store, time.Now, rand.Reader)
}

func newService(exchanger CodeExchanger, store SessionStore, now func() time.Time, random io.Reader) *Service {
	return &Service{exchanger: exchanger, store: store, now: now, random: random}
}

// Issue creates one new session after one provider exchange.
func (service *Service) Issue(ctx context.Context, code string) (IssuedSession, error) {
	openid, err := service.exchanger.Exchange(ctx, code)
	if err != nil {
		if errors.Is(err, wechat.ErrLoginRejected) {
			return IssuedSession{}, ErrLoginRejected
		}
		return IssuedSession{}, ErrUnavailable
	}
	if openid == "" {
		return IssuedSession{}, ErrUnavailable
	}

	entropy := make([]byte, tokenEntropyBytes)
	if _, err := io.ReadFull(service.random, entropy); err != nil {
		clear(entropy)
		return IssuedSession{}, ErrUnavailable
	}
	token := base64.RawURLEncoding.EncodeToString(entropy)
	clear(entropy)
	hash := sha256.Sum256([]byte(token))
	issuedAt := service.now().UTC().Truncate(time.Microsecond)
	expiresAt := issuedAt.Add(sessionTTL)
	if err := service.store.CreateSession(ctx, CreateSessionParams{
		OpenID: openid, TokenHash: hash, IssuedAt: issuedAt, ExpiresAt: expiresAt,
	}); err != nil {
		return IssuedSession{}, ErrUnavailable
	}
	return IssuedSession{AccessToken: token, ExpiresAt: expiresAt}, nil
}

// Authenticate resolves only an active hash-backed session for internal use.
func (service *Service) Authenticate(ctx context.Context, token string) (uint64, error) {
	entropy, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(entropy) != tokenEntropyBytes || base64.RawURLEncoding.EncodeToString(entropy) != token {
		clear(entropy)
		return 0, ErrUnauthenticated
	}
	clear(entropy)
	hash := sha256.Sum256([]byte(token))
	at := service.now().UTC().Truncate(time.Microsecond)
	userID, err := service.store.FindActiveUser(ctx, hash, at)
	if errors.Is(err, ErrSessionNotFound) {
		return 0, ErrUnauthenticated
	}
	if err != nil || userID == 0 {
		return 0, ErrUnavailable
	}
	return userID, nil
}
