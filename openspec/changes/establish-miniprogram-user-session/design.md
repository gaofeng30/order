## Context

Base `2b83e93cc2a8d2bb16b606068028f34ee662b677` serves only anonymous catalog/menu reads and has migrations v1–v7, one structured MySQL config, and no WeChat client, user/session tables, or auth route. The 0818 product baseline allows startup `wx.login` without startup phone authorization; phone, employee, merchant, RBAC, PC QR and protected transaction behavior remain separate product modules.

WeChat's current official [`wx.login`](https://developers.weixin.qq.com/miniprogram/dev/api/open-api/login/wx.login.html) document states that the returned code is valid for five minutes and must be exchanged by the developer server. The current official [`code2Session`](https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/user-login/code2Session.html) contract is one HTTPS GET with `appid`, `secret`, `js_code`, and `grant_type=authorization_code`, returning openid/`session_key` and documented errors `-1`, `40029`, `40226`, and `45011`. Those platform facts define the wire boundary; opaque application Bearer sessions, 24-hour TTL and concurrent-session policy are this change's technical contract.

This is `gate_type=W3`, `ui_level_target=UI0`, initially `ui_level_actual=NOT_RUN`. The real MySQL 8 asset is writer-managed and required locally. Real WeChat account/secrets/network/code are a delivery external blocker, not a substitute for local proof.

## Goals / Non-Goals

**Goals:**

- Exchange one startup code through the fixed official endpoint, keep only openid, atomically resolve one internal ordinary user, and issue a hash-only opaque session.
- Prove provider wire/error handling, high-entropy token handling, transaction rollback, same-openid concurrency, multi-session behavior, persistence lookup, exact expiry, API errors and log secrecy.
- Preserve existing anonymous catalog/menu contracts and current production secret fail-closed behavior.

**Non-Goals:**

- Phone authorization/binding, additional phone/P4, employee/list/discount/P5, merchant binding/roles/RBAC, PC QR/P3, P1/P2, checkout/payment/order, frontend, profile/logout/refresh, global auth middleware, protected business routes, session revocation, or production SSM secret loading.
- UnionID persistence, `session_key` persistence/use, cross-device mutual exclusion, one-session-per-user, configurable TTL, configurable WeChat origin, automatic provider retry, cleanup jobs, or compatibility routes.
- Reviving the historical identity-and-access DRAFT's bundled phone, four-role, Web Admin, idempotency/audit, or transaction scope. It is historical contrast only and is not a dependency.

## Decisions

### Use one strict session-creation contract

`identity.Handler` registers only `POST /api/v1/auth/miniprogram/session`. It accepts at most 1 KiB `application/json`, uses a decoder that rejects unknown/duplicate fields and trailing values, and validates an exact nonblank code of at most 256 bytes without trimming or echoing it. Gin's existing 404/405 behavior handles missing compatibility paths and wrong methods.

Success is HTTP 201:

```json
{"access_token":"<opaque>","token_type":"Bearer","expires_at":"<UTC RFC3339>"}
```

The raw token appears only here. Request shape failures are HTTP 400 `INVALID_REQUEST`; WeChat code rejection is HTTP 401 `MINIPROGRAM_LOGIN_REJECTED`; every provider availability/protocol, crypto, database or commit failure is HTTP 503 `SESSION_UNAVAILABLE`. A single public unavailable category avoids leaking whether an account, secret, provider or row exists. Alternative 200 login responses and separate upstream/database errors were rejected because every success creates a session resource and callers do not need infrastructure detail.

### Keep the provider boundary fixed and disposable

`internal/wechat` owns `Credentials`, a `CodeExchanger` interface and the fixed `https://api.weixin.qq.com/sns/jscode2session` client. Runtime construction accepts credentials and an HTTP client with a three-second timeout/no redirects, but no endpoint. A package-local test constructor may inject an `httptest.Server` endpoint; that injection is not reachable through config or main.

The client builds one GET, performs no retry, reads at most 16 KiB, and decodes exactly one documented JSON object. Runtime owns a dedicated clone of Go's production transport with HTTP keep-alives disabled and HTTP/2 negotiation explicitly disabled. Therefore each accepted exchange starts on a fresh HTTP/1.1 connection: Go's transparent replay path for an idempotent GET on a reused HTTP/1.x connection is unreachable, and HTTP/2's internal retry loop cannot reintroduce replay. Package-local tests prove both transport flags and two sequential exchanges using two distinct connections while an HTTP/2-capable TLS server observes only HTTP/1.1. The client requires a success `errcode` and nonempty openid/`session_key`, returns only openid, and drops every other field immediately. It converts all failures to typed categories without wrapping the request URL or provider body, preventing AppSecret/code/body leakage through Go error strings. Following redirects, accepting partial/oversize JSON, returning `session_key`, a reusable runtime connection, HTTP/2, and a configurable origin were rejected as unnecessary credential-exposure or replay surfaces.

### Add minimal v8/v9 tables

v8 creates `miniprogram_users`:

- `id BIGINT UNSIGNED AUTO_INCREMENT` primary key;
- `openid VARBINARY(128) NOT NULL` with one UNIQUE key for byte-exact opaque identity;
- `created_at TIMESTAMP(6) NOT NULL` and `last_login_at TIMESTAMP(6) NOT NULL`.

v9 creates `miniprogram_sessions`:

- `token_hash BINARY(32)` primary key;
- `user_id BIGINT UNSIGNED NOT NULL` with a restrictive foreign key;
- `issued_at TIMESTAMP(6) NOT NULL`, `expires_at TIMESTAMP(6) NOT NULL`, a user/expiry index, and `expires_at > issued_at` CHECK.

No session ID is needed because the collision-resistant hash is the lookup/uniqueness key. No phone, merchant, role, UnionID, `session_key`, code, raw token, revocation, device or metadata column is admitted. Each migration stays one forward-only statement as required by the existing runner. All embedded-name/count and v1→latest tests move from 7 to 9.

### Make user upsert and session insert one transaction

`identity.Repository` begins one SQL transaction, executes an openid UNIQUE upsert using `LAST_INSERT_ID(id)` to obtain either the new or existing internal ID, updates `last_login_at`, inserts the hash-only session, and commits before returning. MySQL's unique key serializes concurrent creation of the same openid; the service does not add process-local locks. Duplicate token hash, insert failure or commit failure returns an unavailable category and leaves no new user/session or login timestamp change.

Every successful startup creates a distinct session. Existing sessions are not deleted or invalidated, so simultaneous devices remain valid until their own expiry. Mutual exclusion was rejected because it changes cross-device product behavior without a confirmed requirement. The real MySQL W3 test starts simultaneous transactions for one new openid, verifies one user/two sessions, and forces a session-hash collision to prove rollback.

### Treat token creation and validation as one internal session primitive

`identity.Service` receives the exchanger, repository, clock and random reader. Runtime uses `time.Now` and `crypto/rand.Reader`; tests inject deterministic versions. It reads exactly 32 random bytes, returns unpadded URL-safe base64, hashes the raw token with SHA-256 before the transaction, and zeroes the temporary byte slice after encoding/hash computation where practical. There is no fallback generator or collision retry.

The clock value is normalized to UTC and truncated to microseconds, and expiry is fixed at exactly 24 hours. Internal lookup hashes a presented raw token and queries the same repository for `issued_at <= now AND expires_at > now`; at the exact expiry instant it fails unauthenticated. This internal primitive is necessary to prove that the returned Bearer token maps to hash-only persistence and honors expiry, but no middleware or public consumer is registered in this change. Configurable TTL and refresh/revocation were rejected because no confirmed contract requires them.

### Extend only development/test structured config

`config.Config` gains a nested Mini Program credential value loaded from `ORDER_WECHAT_MINIPROGRAM_APP_ID` and `ORDER_WECHAT_MINIPROGRAM_APP_SECRET` in development/test. Validation accepts only nonempty bounded ASCII values and emits stable field/reason errors without values. Production checks the AppSecret alongside existing environment secrets: if present it returns `production_secret_environment_forbidden`; otherwise it keeps returning `production_secret_source_unavailable`. No base URL, TTL or compatibility switch is added.

`cmd/order-api/main.go` wires the shared DB, fixed WeChat client, identity repository/service/handler and existing router. `httpapi.NewRouter` receives the identity handler in addition to catalog/menu; no global middleware changes. The smoke test supplies only canary development credentials and asserts they never appear in logs.

### Gate local and real-platform evidence separately

The local provider suite uses controlled HTTP/TLS test servers and covers exact method/path/query, fresh-connection HTTP/1.1 runtime transport, HTTP/2 exclusion, redirect refusal, timeout, body cap, strict JSON, error mapping and canary secrecy. The local W3 script uses the existing isolated `order-mysql-w3` contract and runs migrations plus identity/catalog/menu/router integrations; it is a required PASS, not an external asset.

Real code2Session proof stays `BLOCKED_EXTERNAL/NOT_RUN` until the customer Mini Program administrator and developer provide a real AppID/AppSecret, authenticated account, network and fresh code. That future run must capture only non-sensitive request/result references; a local stub can never promote this status.

## Risks / Trade-offs

- [Risk] A code is consumed by WeChat but MySQL commit fails, so replay cannot recover. → Return the stable recoverable 503 and require the Mini Program to call `wx.login` again; never retry the consumed provider code server-side.
- [Risk] AppSecret/code can leak through URL-bearing transport errors. → Never wrap/return raw HTTP errors, refuse redirects, log only typed categories, and scan canaries across every failure path.
- [Risk] Concurrent openid creation or token collision can leave partial rows. → Unique keys plus one transaction, forced-collision rollback tests, and real MySQL concurrency evidence.
- [Risk] Multiple active sessions increase exposure after device loss. → Keep the explicitly selected 24-hour short TTL; logout/revocation and mutual exclusion remain separately scoped product changes.
- [Risk] v8/v9 are forward-only. → Before deployment, rollback removes code/change only; after migration, prior code safely ignores the additive tables, and any table removal requires a separately approved forward migration.
- [Risk] Real WeChat behavior is not exercised locally. → Preserve `BLOCKED_EXTERNAL/NOT_RUN` and never label stub results as platform PASS.

## Migration Plan

1. After explicit main-session approval, enter IMPLEMENTING and obtain provider/API/service/MySQL Red evidence before business implementation.
2. Add v8/v9 and update every exact migration count/name/upgrade assertion; run the isolated migration and identity W3 suite.
3. Implement provider, service/repository/crypto, config, handler/router/main and smoke changes in owned paths only; rerun identical focused checks for Green and Refactor.
4. Run strict, owned-path/diff, gofmt, all Go tests, focused/all race, vet, build, smoke and catalog/menu regressions. Commit only after every writer Gate passes, then hand the exact clean SHA to an independent verifier.
5. No push, PR, integration, archive, deploy, production mutation or real-platform call is authorized by this change. If later deployed, run migrations v8 then v9 before the new binary; rollback of the binary is compatible with the additive tables.

## Open Questions

No behavior-, contract-, data-, authorization- or acceptance-changing question remains for implementation. P1–P5 and all delivery external assets retain their separate boundaries.
