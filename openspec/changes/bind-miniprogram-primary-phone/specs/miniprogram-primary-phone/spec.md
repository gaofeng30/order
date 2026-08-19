## ADDED Requirements

### Requirement: One authenticated route binds the primary phone

The API MUST expose only `POST /api/v1/me/bind-phone` for primary-phone binding. The caller MUST be an already logged-in WeChat Mini Program user, and it MUST call this route immediately after the user actively grants `getPhoneNumber` from the first-checkout interception or the Personal Center entry. Startup and anonymous browsing MUST NOT invoke phone authorization, and no unversioned compatibility or profile route SHALL be added.

The request MUST contain exactly one valid `Authorization: Bearer <opaque-session-token>` header and at most 1 KiB of `application/json` with exactly one non-empty `code` string of at most 256 bytes. The route MUST reuse the integrated session `Authenticate` operation without global middleware. Unknown/duplicate fields, trailing JSON, an oversize/blank code or wrong media type MUST return HTTP 400 `INVALID_REQUEST` without provider/phone-database work; a missing, malformed or expired session MUST return HTTP 401 `UNAUTHENTICATED`; an authentication storage failure MUST return HTTP 503 `PHONE_BINDING_UNAVAILABLE`.

Successful first binding, same-phone idempotency, a valid retry that finds the user already bound before provider access, and same-code concurrency recovery MUST return exact HTTP 200 fields `primary_phone_bound: true` and `masked_phone`. The masked value MUST preserve `+`, replace every normalized digit except the last at most four with `*`, and hide at least one digit. No success or error response SHALL expose a user ID, openid, full phone, code, access token, AppSecret or provider body.

#### Scenario: First checkout requests primary-phone binding

- **WHEN** an authenticated unbound user actively grants getPhoneNumber at the first-checkout interception and submits the fresh code
- **THEN** the route binds the verified primary phone and returns HTTP 200 with only `primary_phone_bound: true` and a masked phone
- **AND** the caller can return to checkout, while this change itself creates no checkout, order or payment behavior

#### Scenario: Personal Center retries after binding

- **WHEN** an authenticated request with valid shape reaches the route after that user already has a primary phone
- **THEN** the API returns the same HTTP 200 representation of the existing binding
- **AND** it does not call stable token or getPhoneNumber and does not replace the phone

#### Scenario: Route authentication or JSON is invalid

- **WHEN** the Bearer session is absent/invalid/expired or the request JSON/media type is not exact
- **THEN** the API returns the corresponding stable 401 or 400 envelope
- **AND** it does not consume a phone code or write phone data

#### Scenario: Anonymous routes are exercised after phone wiring

- **WHEN** health, catalog, menu or session creation is called without a Bearer token
- **THEN** its integrated anonymous contract remains unchanged
- **AND** no global auth middleware intercepts it

### Requirement: Stable access token uses one fixed cached official flow

The runtime MUST obtain phone API credentials only by `POST https://api.weixin.qq.com/cgi-bin/stable_token` with a strict JSON object containing `grant_type: client_credential`, configured AppID/AppSecret and `force_refresh: false`. The official origin MUST NOT be runtime-configurable. A package-local test constructor MAY inject controlled endpoints, but config/main MUST NOT expose them.

The process MUST merge concurrent cache misses so only one stable-token wire request is in flight and all waiters observe the same sanitized result. A valid result MUST be cached only in memory according to `expires_in` and stop being reusable five minutes before expiry; a result with no more than five minutes remaining MAY serve the current merged callers but MUST NOT remain reusable. No access token SHALL enter MySQL, logs, errors or evidence.

Each refresh MUST perform one wire attempt with a three-second request timeout, redirect refusal, 16 KiB response cap and strict single-object JSON. Stable-token transport/protocol/provider failure MUST return unavailable without automatic retry. Redis/DB cache, background refresh and `force_refresh=true` MUST NOT be added.

#### Scenario: Concurrent callers miss the token cache

- **WHEN** multiple unbound authenticated requests concurrently require a token and no reusable token exists
- **THEN** exactly one fixed official stable-token request is observed
- **AND** all waiters receive the same in-memory token result or the same sanitized unavailable result

#### Scenario: Cached token approaches expiry

- **WHEN** the cached token reaches its five-minute early-refresh boundary
- **THEN** a subsequent fresh phone-code request obtains one ordinary-mode stable-token result before calling getPhoneNumber
- **AND** no forced refresh, persistent cache or stale-token replay is used

#### Scenario: Stable-token response is unavailable or malformed

- **WHEN** the token request times out, redirects, exceeds the body cap, returns a failure status/code or invalid JSON/fields
- **THEN** phone binding returns HTTP 503 `PHONE_BINDING_UNAVAILABLE`
- **AND** AppSecret, access token, provider body and URL query material remain absent from errors/logs/evidence

### Requirement: Phone exchange verifies code, openid, watermark and E.164

For each accepted unbound request, the backend MUST call `POST https://api.weixin.qq.com/wxa/business/getuserphonenumber?access_token=...` exactly once with strict JSON containing the submitted getPhoneNumber `code` and the current session user's v8 openid. The openid MUST come from server persistence, never the client. The runtime MUST use the fixed official endpoint, three-second timeout, redirect refusal, 16 KiB cap, strict single-object JSON and a non-reusing HTTP/1.1 transport with keep-alive and HTTP/2 disabled.

A success MUST include phone info whose watermark AppID exactly equals the configured AppID. The backend MUST construct the stored phone only as `+` followed by `countryCode + purePhoneNumber`; the components MUST be ASCII digits, country code MUST not start with zero, and the result MUST contain no more than 15 digits and fit `VARBINARY(16)`. `phoneNumber` MUST be a valid non-empty provider string but MUST NOT be the normalization source.

Current official provider errors `40013` or `40029` MUST return HTTP 422 `PHONE_CODE_REJECTED`. Transport/protocol/quota/system/general invalid-token/unknown errors and watermark/phone format mismatch MUST return HTTP 503 `PHONE_BINDING_UNAVAILABLE`. No phone-code, transport, protocol or invalid-token failure SHALL automatically retry stable token or getPhoneNumber; an invalid cached token MAY be evicted only for a later fresh user-click code.

#### Scenario: Official phone exchange succeeds

- **WHEN** the fixed provider accepts one fresh code bound to the authenticated openid and returns matching watermark/phone fields
- **THEN** exactly one getPhoneNumber request is observed with code and server-resolved openid
- **AND** only the normalized E.164 value crosses into the binding transaction

#### Scenario: Phone code is invalid or bound to a different identity

- **WHEN** the provider returns documented code/AppID rejection for the submitted code/openid relationship
- **THEN** the API returns HTTP 422 `PHONE_CODE_REJECTED` without a phone write
- **AND** neither provider message nor any identity/credential value is exposed

#### Scenario: Cached token is rejected

- **WHEN** getPhoneNumber rejects the access token as invalid
- **THEN** the current request returns HTTP 503 without replaying the phone code or refreshing and retrying it
- **AND** a later request can refresh only after a new user click supplies a fresh code

#### Scenario: Watermark or phone format is invalid

- **WHEN** provider success has another AppID, missing/invalid phone fields, a zero-leading country code or more than 15 combined digits
- **THEN** the API fails closed with HTTP 503
- **AND** no partial or alternative-format phone is persisted

### Requirement: Primary phone binds once with deterministic concurrency

The provider call MUST occur without an open database transaction. After a verified E.164 result, one MySQL transaction MUST lock the current user row, re-read binding state and either write the first phone/time, return same-phone idempotency, reject a different-phone race, or reject a phone already owned by another user. A successful/idempotent retry MUST retain the original `primary_phone_bound_at`.

If the locked user is unbound, the transaction MUST write the verified phone and first binding time atomically. If it is already bound to that phone, the request MUST return the same HTTP 200 without a write. If it is already bound to a different phone, the request MUST return HTTP 409 `PRIMARY_PHONE_ALREADY_BOUND`; if another user owns the phone, it MUST return HTTP 409 `PHONE_IN_USE`. Statement/commit failure MUST roll back and return HTTP 503.

When two concurrent requests use the same one-time code and one provider call is rejected after another request has bound the user, the rejected branch MUST re-read only the current binding state: an observed binding returns the current HTTP 200 result, while no binding returns HTTP 422. It MUST NOT retry the provider or infer a phone from the rejected response.

#### Scenario: First verified phone is committed

- **WHEN** the locked user remains unbound and no other user owns the normalized phone
- **THEN** phone and bound-at commit together once
- **AND** the response is returned only after commit

#### Scenario: Two different phones race for one user

- **WHEN** two provider-success requests reach the binding transaction for different phones of the same initially unbound user
- **THEN** exactly one phone commits and the other returns `PRIMARY_PHONE_ALREADY_BOUND`
- **AND** the winner's phone/time are not overwritten

#### Scenario: Two users bind the same phone

- **WHEN** two authenticated users concurrently bind the same normalized phone
- **THEN** the unique constraint allows exactly one owner and the other request returns `PHONE_IN_USE`
- **AND** neither transaction leaves a partial pair of phone/time fields

#### Scenario: Same code is consumed concurrently

- **WHEN** one concurrent request binds successfully and another receives code rejection for the same one-time code
- **THEN** the rejected branch returns current HTTP 200 only if its authenticated user is now observed bound
- **AND** otherwise it returns 422 without another provider call

#### Scenario: Persistence fails after locking

- **WHEN** phone update, uniqueness handling or commit fails
- **THEN** the transaction rolls back and returns HTTP 503
- **AND** no new phone, timestamp change or cross-user ownership leak remains

### Requirement: Migration v10 adds only the paired primary-phone fields

Migration `000010_add_miniprogram_primary_phone.sql` MUST be the next continuous, forward-only migration and contain one `ALTER TABLE miniprogram_users`. It MUST add nullable `primary_phone VARBINARY(16)`, nullable `primary_phone_bound_at TIMESTAMP(6)`, a unique key on phone and a CHECK requiring the two fields to be both NULL or both non-NULL. It MUST NOT modify v8/v9 or add an additional/merchant/name/role/whitelist/discount field.

Existing users MUST migrate with both fields NULL. MySQL byte equality on canonical E.164 MUST enforce cross-user uniqueness while permitting multiple NULL rows. The embedded catalog, menu, migrate and identity migration suites MUST all require exact v1 through v10 ordering and pass first-run/repeat against real MySQL 8.

#### Scenario: Existing database upgrades to v10

- **WHEN** migrations v1 through v9 are present and v10 runs
- **THEN** all existing users remain unbound with both new fields NULL
- **AND** the exact unique key, CHECK and column types are present

#### Scenario: Invalid partial pair is written

- **WHEN** SQL attempts to store only phone or only bound-at
- **THEN** MySQL rejects the row through the v10 CHECK
- **AND** existing v8/v9 bytes and session behavior remain unchanged

#### Scenario: Migration is repeated

- **WHEN** the migration runner executes again after v10 is recorded
- **THEN** it applies zero statements and remains ready at version 10
- **AND** catalog, menu and identity integration behavior remains intact

### Requirement: Sensitive data and real-platform evidence remain bounded

Logs, returned errors, tests, task evidence and ordinary traces MUST NOT contain a full phone, phone code, access token, AppSecret, Authorization value, openid, session token/hash or provider body/query. Runtime failures MUST expose only stable route/status/reason fields and existing non-sensitive request IDs. Tests MAY use synthetic canaries only when failure output itself cannot print the canary.

Local provider stubs MUST prove only wire/cache/concurrency/error/secrecy. Real platform acceptance MUST remain `BLOCKED_EXTERNAL/NOT_RUN` until an authenticated non-personal Mini Program subject, enabled phone capability and usable quota, real credentials, official network and a fresh user-click code are available under separate authority. Local MySQL/stub PASS MUST NOT promote this Gate.

#### Scenario: Sensitive canaries cross every failure path

- **WHEN** auth, token, phone-provider, protocol, database and panic tests use distinct synthetic canaries
- **THEN** captured logs, responses and evidence contain none of them
- **AND** only stable sanitized categories remain observable

#### Scenario: Local candidate has no real platform assets

- **WHEN** all local provider, W3, regression, race, vet, build and smoke checks pass without real customer/platform assets
- **THEN** the repository candidate may reach local `ACCEPT`
- **AND** real getPhoneNumber remains `BLOCKED_EXTERNAL/NOT_RUN`
