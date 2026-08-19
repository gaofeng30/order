## ADDED Requirements

### Requirement: Mini Program startup creates one versioned server session

The API MUST expose only `POST /api/v1/auth/miniprogram/session` for this capability. The caller is the WeChat Mini Program startup flow, and it MUST call this route immediately after one successful `wx.login` to exchange that one-time code for a server session; no unversioned compatibility route SHALL exist.

The request MUST use `application/json`, fit within 1 KiB, and contain exactly one non-empty `code` string of at most 256 bytes. Unknown fields, duplicate fields, trailing JSON values, a missing or blank code, a body that exceeds the limit, or another content type MUST receive HTTP 400 with `{"error":{"code":"INVALID_REQUEST","message":"invalid request"}}` without calling WeChat or MySQL.

Every successful exchange MUST create a new session and return HTTP 201 with exactly `access_token`, `token_type` equal to `Bearer`, and UTC RFC3339 `expires_at`. The raw access token MUST be returned only in this success response; user ID, openid, `session_key`, UnionID, AppSecret, submitted code, and token hash MUST not be returned.

#### Scenario: Mini Program exchanges a fresh startup code

- **WHEN** the Mini Program submits one valid fresh `wx.login` code to `POST /api/v1/auth/miniprogram/session`
- **THEN** the API returns HTTP 201 with one opaque `access_token`, `token_type: Bearer`, and `expires_at`
- **AND** the response contains no user, openid, `session_key`, UnionID, secret, code, or token hash field

#### Scenario: Request JSON is not exact

- **WHEN** a request has the wrong media type, exceeds 1 KiB, contains unknown/duplicate fields or trailing JSON, or has a missing, blank, or overlength code
- **THEN** the API returns the stable HTTP 400 `INVALID_REQUEST` envelope
- **AND** neither the WeChat provider nor MySQL is called

#### Scenario: A compatibility or wrong-method route is attempted

- **WHEN** a caller uses an unversioned session path or a method other than POST
- **THEN** the existing router returns 404 or 405 respectively
- **AND** no alternate authentication route is registered

### Requirement: WeChat code exchange follows the official fixed wire contract

The backend MUST call `GET https://api.weixin.qq.com/sns/jscode2session` exactly once per accepted request with URL-encoded `appid`, `secret`, submitted `js_code`, and `grant_type=authorization_code`. Production wiring MUST use that fixed HTTPS endpoint, a three-second whole-request timeout, no redirect following, and a response-body limit of 16 KiB; runtime configuration MUST NOT accept an alternate base URL.

The client MUST accept only one complete JSON object matching the documented field types. A success MUST have `errcode` absent or zero and non-empty openid and `session_key`; the client MUST return only the opaque openid to the identity service and MUST immediately discard `session_key`, optional UnionID, and provider message. HTTP failure, timeout, redirect, oversize/malformed/trailing JSON, missing success fields, or an unknown/non-transient provider error MUST fail closed.

Official provider errors `40029` (invalid code) and `40226` (blocked code) MUST map to HTTP 401 `{"error":{"code":"MINIPROGRAM_LOGIN_REJECTED","message":"miniprogram login rejected"}}`. Provider `-1`, `45011`, transport/protocol failures, unknown nonzero errors, database failures, token-generation failures, and commit failures MUST map to HTTP 503 `{"error":{"code":"SESSION_UNAVAILABLE","message":"session temporarily unavailable"}}`. Provider responses and failures MUST NOT be retried because the submitted code is one-time.

#### Scenario: Provider wire contract succeeds

- **WHEN** a controlled provider returns one valid documented success object
- **THEN** the outgoing request is exactly one GET to `/sns/jscode2session` with the four required query values
- **AND** only openid crosses into the identity service while `session_key` and UnionID are discarded

#### Scenario: WeChat rejects the login code

- **WHEN** WeChat returns `40029` or `40226`
- **THEN** the API returns the stable HTTP 401 `MINIPROGRAM_LOGIN_REJECTED` envelope
- **AND** no user or session row is written

#### Scenario: WeChat is unavailable or malformed

- **WHEN** WeChat times out, redirects, exceeds the body limit, returns non-success HTTP, malformed/trailing JSON, missing success fields, `-1`, `45011`, or another nonzero error
- **THEN** the API returns the stable HTTP 503 `SESSION_UNAVAILABLE` envelope without retry
- **AND** no provider body, query credential, submitted code, openid, or `session_key` is exposed

### Requirement: User find-or-create and session persistence are one MySQL transaction

Migration v8 MUST create `miniprogram_users` containing only an auto-increment internal ID, one unique opaque openid, `created_at`, and `last_login_at`. Migration v9 MUST create `miniprogram_sessions` containing only a SHA-256 token hash primary key, a user foreign key, `issued_at`, and `expires_at`; it MUST NOT contain raw token, code, `session_key`, UnionID, phone, merchant, role, or authorization fields.

For each provider success, the repository MUST run one transaction that atomically inserts or resolves the user by unique openid, updates `last_login_at`, and inserts the session hash. The raw token MUST be returned only after commit. A failure or token-hash collision at any step MUST roll back both a new user and all login/session changes. Concurrent logins for the same openid MUST resolve one internal user while each successful request creates its own session; multiple still-valid sessions for one user MUST be allowed.

#### Scenario: First login creates user and session atomically

- **WHEN** a valid openid has no existing user and session insertion commits
- **THEN** exactly one user and one hash-only session exist with the same internal user reference
- **AND** the committed timestamps match the service result

#### Scenario: Same openid logs in concurrently

- **WHEN** two transactions concurrently issue sessions for the same new openid
- **THEN** exactly one `miniprogram_users` row exists
- **AND** both successful requests have distinct valid session hashes bound to that same user

#### Scenario: Session insertion fails after user upsert

- **WHEN** session insertion or commit fails after the user insert/update has begun
- **THEN** the complete transaction rolls back
- **AND** no orphan new user, session row, or advanced `last_login_at` remains

### Requirement: Bearer tokens are high-entropy, hash-only, and expire at one fixed boundary

The service MUST generate each token from 32 bytes read from `crypto/rand` and encode it as unpadded URL-safe base64. It MUST persist only `SHA-256(raw_token)` and MUST never persist or log the raw token. A deterministic reader MAY be injected only by package-local tests; runtime wiring MUST use `crypto/rand`.

The service MUST use an injected clock and set `issued_at` to the UTC clock value truncated to MySQL microsecond precision and `expires_at` to exactly 24 hours later. A session is active only while `issued_at <= now < expires_at`; the exact `expires_at` instant is expired. Internal hash lookup MUST prove persistence and this boundary, but this change MUST NOT register auth middleware or any protected/profile/logout/refresh endpoint.

#### Scenario: Raw token is issued and only its hash persists

- **WHEN** token generation and the MySQL transaction succeed
- **THEN** the response token decodes to 32 random bytes and its SHA-256 matches the single stored hash
- **AND** neither the raw token nor its hash appears in logs or any other response field

#### Scenario: Expiry boundary is evaluated

- **WHEN** the injected clock is before `expires_at`
- **THEN** internal lookup by raw token resolves the session's internal user
- **AND** lookup at or after the exact `expires_at` instant returns unauthenticated

#### Scenario: Token entropy source fails or collides

- **WHEN** `crypto/rand` cannot provide 32 bytes or the generated hash collides with an existing session
- **THEN** the request fails with `SESSION_UNAVAILABLE`
- **AND** the user/session transaction leaves no partial change and no fallback token is generated

### Requirement: Runtime configuration and observability fail closed without secrets

Development and test MUST require structured `ORDER_WECHAT_MINIPROGRAM_APP_ID` and `ORDER_WECHAT_MINIPROGRAM_APP_SECRET` values in addition to the existing structured database fields. Empty or malformed values MUST fail startup with a stable non-sensitive reason. Production MUST continue to fail with `production_secret_source_unavailable`; supplying AppSecret through an environment variable in production MUST fail with `production_secret_environment_forbidden`, and this change MUST NOT add the production SSM loader.

Access/error logs and returned errors MUST include only the existing request ID, method, templated path, status, duration, and stable reason/category. They MUST NOT contain an Authorization value, raw token/hash, submitted code, openid, `session_key`, AppSecret, provider raw body/query, or database credential.

#### Scenario: Development configuration is complete

- **WHEN** development or test starts with valid structured database, AppID, and AppSecret fields
- **THEN** configuration loads and main wires the fixed WeChat client and session handler
- **AND** no arbitrary provider URL is configurable

#### Scenario: Production env secret is attempted

- **WHEN** production starts with a Mini Program AppSecret environment value
- **THEN** startup fails with `production_secret_environment_forbidden`
- **AND** neither the secret nor the variable value is logged

#### Scenario: Sensitive canaries cross failure paths

- **WHEN** request, provider, crypto, database, and panic failure tests use distinct secret/code/openid/token/session-key canaries
- **THEN** captured logs, public errors, and non-success responses contain none of those canaries
- **AND** only stable status and error categories remain observable

### Requirement: Existing anonymous catalog and menu contracts remain unchanged

`/api/v1/catalog`, `/api/v1/catalog/products/:id`, and `/api/v1/menu` MUST remain anonymous and byte-for-byte contract compatible with the base behavior. This change MUST NOT install global auth middleware or require a Bearer token for health, catalog, or menu reads.

#### Scenario: Anonymous regressions are rerun after session wiring

- **WHEN** existing catalog/menu contract and MySQL integration tests run after the new route is registered
- **THEN** their success and error responses remain unchanged and require no Authorization header
- **AND** the only new public route is `POST /api/v1/auth/miniprogram/session`

### Requirement: Real WeChat proof remains an explicit delivery external blocker

Local provider tests MUST use a controlled HTTP stub and MUST prove only wire, parsing, timeout, error, and secrecy contracts. Real WeChat acceptance requires a real Mini Program AppID/AppSecret, authenticated account, reachable official network, and a fresh real `wx.login` code; until those assets exist, the real-platform result MUST remain `BLOCKED_EXTERNAL/NOT_RUN` and MUST NOT be represented as PASS.

#### Scenario: Local candidate is evaluated without real WeChat assets

- **WHEN** all local W3, provider-stub, API, regression, race, vet, build, and smoke checks pass but real platform assets are unavailable
- **THEN** the local candidate may satisfy its repository `ACCEPT` verdict
- **AND** delivery evidence still records real WeChat as `BLOCKED_EXTERNAL/NOT_RUN`
