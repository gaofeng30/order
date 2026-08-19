## ADDED Requirements

### Requirement: Authenticated callers can read only their primary-phone status

The API MUST expose only `GET /api/v1/me/primary-phone` for this capability. The caller MUST be the logged-in WeChat Mini Program: Personal Center MUST call it on every `onShow`, and first checkout MUST call it as a preflight before deciding whether to enter the existing phone-binding flow. This change MUST NOT add `/api/v1/me`, an unversioned/compatibility/profile route, global auth middleware, or frontend/checkout behavior.

The request body MUST be exactly empty; any received byte MUST return HTTP 400 with exact body `{"error":{"code":"INVALID_REQUEST","message":"invalid request"}}` before authentication or phone persistence access. The route MUST NOT require or validate `Content-Type`. It MUST require exactly one valid `Authorization: Bearer <opaque-session-token>` header and reuse the integrated session `Authenticate`; a missing, duplicate, malformed, unknown, or expired session MUST return HTTP 401 with exact body `{"error":{"code":"UNAUTHENTICATED","message":"authentication required"}}`.

Every response produced by this GET handler, including 200, 400, 401 and 503, MUST include `Cache-Control: no-store`.

#### Scenario: Personal Center reads an authenticated status

- **WHEN** Personal Center `onShow` sends an empty GET with one active Bearer session
- **THEN** the route reads only that session user's current primary-phone status and returns the corresponding exact HTTP 200 representation
- **AND** the response includes `Cache-Control: no-store`

#### Scenario: First checkout performs preflight

- **WHEN** first checkout sends the same empty authenticated GET before payment preparation
- **THEN** the route returns only the current binding state used to choose whether the existing bind flow is needed
- **AND** this change creates no checkout, order, payment, phone-binding, or frontend behavior

#### Scenario: GET body is non-empty

- **WHEN** the GET contains any body byte, including whitespace or JSON, with any or no Content-Type
- **THEN** the route returns the exact HTTP 400 `INVALID_REQUEST` envelope with `Cache-Control: no-store`
- **AND** authentication, phone persistence, provider and write operations are not called

#### Scenario: Session is not acceptable

- **WHEN** Authorization is absent, duplicated or malformed, or its token is unknown or expired
- **THEN** the route returns the exact HTTP 401 `UNAUTHENTICATED` envelope with `Cache-Control: no-store`
- **AND** no phone state, identity existence detail or persistence error is exposed

### Requirement: Bound and unbound representations are exact and minimal

For an authenticated user with a v10 primary phone, the route MUST return HTTP 200 with exactly `{"primary_phone_bound":true,"masked_phone":"<integrated-mask>"}`. The mask MUST use the integrated primary-phone mask rule: retain `+`, replace every normalized digit except the last at most four with `*`, and hide at least one digit.

For an authenticated user whose v10 primary phone is SQL NULL, the route MUST return HTTP 200 with exactly `{"primary_phone_bound":false,"masked_phone":null}`. SQL NULL validity MUST be preserved through the repository/domain result; an empty string MUST NOT be treated as equivalent to NULL. The field MUST be present as JSON null, not omitted or encoded as an empty string. Neither representation MUST include internal user ID, openid, full phone, employee/visitor state, discount, whitelist, merchant/PC state, avatar, nickname, or another profile field.

#### Scenario: Current user is bound

- **WHEN** the authenticated current user's stored canonical phone is `+8613712345678`
- **THEN** the API returns exactly `{"primary_phone_bound":true,"masked_phone":"+*********5678"}` with HTTP 200
- **AND** the full phone and all other identity/profile fields remain absent

#### Scenario: Current user is unbound

- **WHEN** the authenticated current user's primary-phone pair is NULL
- **THEN** the API returns exactly `{"primary_phone_bound":false,"masked_phone":null}` with HTTP 200
- **AND** it does not infer a phone or identity state from another user

### Requirement: Status reads are isolated and have no provider or write side effect

The handler MUST derive user ID only from the integrated Bearer authenticator and MUST pass only that ID to `PhoneService.Status`. The service MUST reuse `PhoneStore.FindPhoneUser` and the integrated mask rule. The request MUST accept no user ID, openid, phone or other selector, and MUST NOT query another user's state.

The complete GET path MUST NOT call WeChat, stable-token/getPhoneNumber code, the phone provider, `BindPrimaryPhone`, any transaction or any database write. Successful bound/unbound reads, invalid bodies, authentication failures and persistence failures MUST leave all user and session data unchanged.

#### Scenario: Two users have different phone states

- **WHEN** one valid session belongs to a bound user and another valid session belongs to an unbound user
- **THEN** each session receives only its own bound or unbound representation
- **AND** neither response or observable query result reveals the other user's phone or state

#### Scenario: Status is read repeatedly

- **WHEN** either user calls the GET repeatedly
- **THEN** the same server-backed state is returned without provider calls, binding calls, transactions or writes
- **AND** phone, bound-at, openid, last-login and session rows remain unchanged

### Requirement: Database failures fail closed instead of becoming unbound

Authentication persistence failure or current-user phone persistence failure MUST return HTTP 503 with exact body `{"error":{"code":"PRIMARY_PHONE_STATUS_UNAVAILABLE","message":"primary phone status temporarily unavailable"}}` and `Cache-Control: no-store`. A missing/invalid stored user identity or invalid stored phone that cannot be safely masked MUST use the same sanitized 503. The route MUST NOT convert any such failure into HTTP 200 unbound.

The persisted phone state MUST be valid only when `(bound=false, phone empty)` or `(bound=true, phone is a maskable canonical value)`. Bound plus empty/malformed, or unbound plus non-empty, MUST fail closed. The existing POST bind preflight MUST apply the same validation before provider access: inconsistent state returns its existing sanitized HTTP 503 unavailable envelope without provider, transaction, binding call or write; valid bound/unbound POST behavior remains unchanged. If the binding transaction observes a non-NULL current value after provider access, it MUST accept only a canonical phone: same phone remains idempotent, different canonical phone remains conflict, and empty/malformed returns persistence failure/HTTP 503 without a write.

Errors, logs, responses and ordinary evidence MUST NOT contain Authorization, session token/hash, internal user ID, openid, full phone, SQL, DSN, database error text, provider data, employee/whitelist/discount or merchant state.

#### Scenario: Authentication database is unavailable

- **WHEN** the Bearer token shape is valid but the integrated active-session lookup fails
- **THEN** the route returns the exact HTTP 503 unavailable envelope with `Cache-Control: no-store`
- **AND** it performs no phone read, provider call or write

#### Scenario: Phone database read is unavailable

- **WHEN** authentication succeeds but `FindPhoneUser` fails or returns an invalid current-user record
- **THEN** the route returns the same exact HTTP 503 unavailable envelope with `Cache-Control: no-store`
- **AND** it does not report unbound, expose persistence detail or modify data

#### Scenario: Non-NULL empty stored phone is inconsistent

- **WHEN** v10 contains a non-NULL empty `primary_phone` paired with non-NULL `primary_phone_bound_at`
- **THEN** the authenticated GET returns the exact HTTP 503 status-unavailable envelope instead of HTTP 200 unbound
- **AND** POST bind returns its exact HTTP 503 binding-unavailable envelope before provider or write access
- **AND** both requests leave the user and session rows unchanged

### Requirement: Real MySQL proves the read contract while real WeChat is not required

The writer and exact-SHA verifier MUST run the dedicated isolated real MySQL 8 status integration. It MUST prove exact bound/unbound HTTP results, unknown and exact-expiry session rejection, two-user isolation, authentication-database and phone-database failure mapping, zero provider/binding/write calls, and unchanged user/session data before and after the read/error matrix. Mock or SQLite evidence MUST NOT replace this W3 Gate.

The existing POST bind, session creation, anonymous catalog/menu and smoke regressions MUST remain exact. Real WeChat MUST be recorded as `NOT_REQUIRED` for this GET because the request is forbidden from calling it; the repository result MUST NOT promote the separately recorded phone/platform delivery Gates for the complete user journey.

#### Scenario: Local W3 status acceptance runs

- **WHEN** the dedicated real MySQL 8 runner executes the status matrix on the candidate
- **THEN** every required status, isolation, failure and no-write assertion passes without a provider call
- **AND** existing POST/session/catalog/menu regressions also pass

#### Scenario: Platform readiness is reviewed

- **WHEN** the local candidate passes without real Mini Program credentials, code, account, network or phone capability
- **THEN** this read-only change may satisfy its local acceptance because real WeChat is `NOT_REQUIRED`
- **AND** the complete phone-binding/customer journey remains governed by its existing external platform Gate and is not reported PASS
