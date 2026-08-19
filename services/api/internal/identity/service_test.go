package identity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/wechat"
)

var testNow = time.Date(2026, time.August, 20, 1, 2, 3, 456789999, time.FixedZone("test", 8*60*60))

func TestSessionIssuePersistsOnlyHashAndFixedExpiry(t *testing.T) {
	t.Parallel()
	randomBytes := make([]byte, 32)
	for index := range randomBytes {
		randomBytes[index] = byte(index + 1)
	}
	store := &recordingSessionStore{}
	service := newService(
		staticExchanger{openid: "opaque-provider-user"},
		store,
		func() time.Time { return testNow },
		bytes.NewReader(randomBytes),
	)

	issued, err := service.Issue(context.Background(), "one-time-code")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(issued.AccessToken)
	if err != nil || !bytes.Equal(decoded, randomBytes) {
		t.Fatalf("access token is not the encoded 32-byte entropy")
	}
	if len(store.created) != 1 {
		t.Fatalf("created sessions = %d", len(store.created))
	}
	wantHash := sha256.Sum256([]byte(issued.AccessToken))
	created := store.created[0]
	if created.OpenID != "opaque-provider-user" || created.TokenHash != wantHash {
		t.Fatal("stored identity/hash mismatch")
	}
	wantIssuedAt := testNow.UTC().Truncate(time.Microsecond)
	if !created.IssuedAt.Equal(wantIssuedAt) || !created.ExpiresAt.Equal(wantIssuedAt.Add(24*time.Hour)) {
		t.Fatalf("stored interval = %s..%s", created.IssuedAt, created.ExpiresAt)
	}
	if !issued.ExpiresAt.Equal(created.ExpiresAt) {
		t.Fatal("response expiry does not match persistence")
	}
}

func TestSessionAllowsMultipleActiveTokensForOneUser(t *testing.T) {
	t.Parallel()
	entropy := append(bytes.Repeat([]byte{0x11}, 32), bytes.Repeat([]byte{0x22}, 32)...)
	store := &recordingSessionStore{}
	service := newService(staticExchanger{openid: "same-user"}, store, func() time.Time { return testNow }, bytes.NewReader(entropy))

	first, firstErr := service.Issue(context.Background(), "first-code")
	second, secondErr := service.Issue(context.Background(), "second-code")
	if firstErr != nil || secondErr != nil {
		t.Fatalf("Issue() errors = %v, %v", firstErr, secondErr)
	}
	if first.AccessToken == second.AccessToken || len(store.created) != 2 || store.created[0].TokenHash == store.created[1].TokenHash {
		t.Fatal("multiple sessions were not preserved distinctly")
	}
	if store.created[0].OpenID != store.created[1].OpenID {
		t.Fatal("same provider user changed between sessions")
	}
}

func TestSessionAuthenticateUsesHashAndExactExpiryBoundary(t *testing.T) {
	t.Parallel()
	token := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x33}, 32))
	hash := sha256.Sum256([]byte(token))
	expiresAt := testNow.UTC().Truncate(time.Microsecond).Add(24 * time.Hour)
	store := &recordingSessionStore{activeHash: hash, activeUserID: 42, expiresAt: expiresAt}
	clock := testNow
	service := newService(staticExchanger{}, store, func() time.Time { return clock }, bytes.NewReader(nil))

	clock = expiresAt.Add(-time.Microsecond)
	userID, err := service.Authenticate(context.Background(), token)
	if err != nil || userID != 42 {
		t.Fatalf("Authenticate(before expiry) = (%d, %v)", userID, err)
	}
	clock = expiresAt
	userID, err = service.Authenticate(context.Background(), token)
	if userID != 0 || !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("Authenticate(at expiry) = (%d, %v)", userID, err)
	}
	clock = expiresAt.Add(time.Microsecond)
	userID, err = service.Authenticate(context.Background(), token)
	if userID != 0 || !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("Authenticate(after expiry) = (%d, %v)", userID, err)
	}
}

func TestTokenRejectsMalformedPresentedValueBeforeStore(t *testing.T) {
	t.Parallel()
	store := &recordingSessionStore{}
	service := newService(staticExchanger{}, store, func() time.Time { return testNow }, bytes.NewReader(nil))
	for _, token := range []string{"", "not-base64", base64.RawURLEncoding.EncodeToString([]byte("short")), base64.URLEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))} {
		if userID, err := service.Authenticate(context.Background(), token); userID != 0 || !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("malformed Authenticate result = (%d, %v)", userID, err)
		}
	}
	if store.lookupCalls != 0 {
		t.Fatalf("store lookups = %d", store.lookupCalls)
	}
}

func TestTokenEntropyFailureDoesNotPersist(t *testing.T) {
	t.Parallel()
	store := &recordingSessionStore{}
	service := newService(staticExchanger{openid: "opaque-user"}, store, func() time.Time { return testNow }, failingReader{})
	issued, err := service.Issue(context.Background(), "one-time-code")
	if issued != (IssuedSession{}) || !errors.Is(err, ErrUnavailable) {
		t.Fatal("entropy failure Issue result mismatch")
	}
	if len(store.created) != 0 {
		t.Fatal("entropy failure reached persistence")
	}
}

func TestTokenCollisionFailsWithoutFallback(t *testing.T) {
	t.Parallel()
	reader := &countingReader{source: bytes.NewReader(bytes.Repeat([]byte{0x44}, 64))}
	store := &recordingSessionStore{createErr: errors.New("token collision")}
	service := newService(staticExchanger{openid: "opaque-user"}, store, func() time.Time { return testNow }, reader)
	issued, err := service.Issue(context.Background(), "one-time-code")
	if issued != (IssuedSession{}) || !errors.Is(err, ErrUnavailable) {
		t.Fatal("collision Issue result mismatch")
	}
	if reader.bytesRead != 32 || len(store.created) != 1 {
		t.Fatalf("entropy bytes = %d, create calls = %d", reader.bytesRead, len(store.created))
	}
}

func TestSessionMapsProviderErrorsWithoutPersistence(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		err  error
		want error
	}{
		{name: "rejected", err: wechat.ErrLoginRejected, want: ErrLoginRejected},
		{name: "unavailable", err: wechat.ErrUnavailable, want: ErrUnavailable},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store := &recordingSessionStore{}
			service := newService(staticExchanger{err: test.err}, store, func() time.Time { return testNow }, bytes.NewReader(nil))
			if _, err := service.Issue(context.Background(), "one-time-code"); !errors.Is(err, test.want) {
				t.Fatalf("Issue() error = %v", err)
			}
			if len(store.created) != 0 {
				t.Fatal("provider failure reached persistence")
			}
		})
	}
}

type staticExchanger struct {
	openid string
	err    error
}

func (exchanger staticExchanger) Exchange(context.Context, string) (string, error) {
	return exchanger.openid, exchanger.err
}

type recordingSessionStore struct {
	created      []CreateSessionParams
	createErr    error
	activeHash   [sha256.Size]byte
	activeUserID uint64
	expiresAt    time.Time
	lookupCalls  int
}

func (store *recordingSessionStore) CreateSession(_ context.Context, params CreateSessionParams) error {
	store.created = append(store.created, params)
	return store.createErr
}

func (store *recordingSessionStore) FindActiveUser(_ context.Context, hash [sha256.Size]byte, at time.Time) (uint64, error) {
	store.lookupCalls++
	if hash == store.activeHash && at.Before(store.expiresAt) {
		return store.activeUserID, nil
	}
	return 0, ErrSessionNotFound
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

type countingReader struct {
	source    io.Reader
	bytesRead int
}

func (reader *countingReader) Read(target []byte) (int, error) {
	count, err := reader.source.Read(target)
	reader.bytesRead += count
	return count, err
}
