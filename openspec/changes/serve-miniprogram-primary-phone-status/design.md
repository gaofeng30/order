## Context

Exact base `a5728022c8e497947267f1b8db5ff50983c03be9` already contains v10 primary-phone storage, `Repository.FindPhoneUser`, the shared masking rule, hash-only `Service.Authenticate`, and route-specific `POST /api/v1/me/bind-phone`. Fresh verification permanently invalidated candidate `e318a76f4175f14ef3b2894de3d40536b9a4d76b`: `FindPhoneUser` discarded `sql.NullString.Valid`, while v10 permits a non-NULL empty value paired with non-NULL bound-at, so GET falsely returned unbound and POST treated the record as bindable. The repair retains validity in the domain mapping without changing v10 or SQL.

0818 PRD section 4.1 forbids startup phone authorization and requires binding before first order submission; section 5.9 requires Personal Center to show masked phone or a binding entry. The read contract therefore has exactly two callers: Personal Center on every `onShow`, and first checkout as a preflight before entering the existing binding flow. It is not a general user profile.

This change is `gate_type=W3` and `ui_level_target=UI0`; DRAFT `ui_level_actual=NOT_RUN`. The W3 classification comes from authenticated persisted-state isolation and the requirement to prove no writes and database-failure semantics on real MySQL, not from any migration or mutation in this change.

## Goals / Non-Goals

**Goals:**

- Serve one exact, non-cacheable bound/unbound representation for the authenticated Mini Program user.
- Reuse the integrated Bearer parser/authenticator, phone read, and mask rule without adding provider, transaction, or write behavior.
- Prove strict cross-user isolation, unknown/expired session behavior, auth/phone database failure mapping, and no side effects on real MySQL.
- Keep valid-state bind/session/catalog/menu contracts unchanged, while failing closed before provider/write access for an inconsistent stored binding.

**Non-Goals:**

- Full `/api/v1/me` profile, avatar, nickname, internal user ID, openid, full phone, employee/visitor, discount, merchant or PC state.
- Any valid-state binding behavior change, getPhoneNumber, stable-token cache, migration, repository SQL, router/main wiring, global middleware, frontend, checkout, order, payment, P1-P5 product feature, or external platform state.

## Decisions

### Register one GET on the existing route-specific phone handler

`PhoneHandler.RegisterRoutes` will add only `GET /api/v1/me/primary-phone` beside the existing POST route. The existing handler service boundary will gain `Status(context.Context, userID)` and the concrete `PhoneService` will implement it, so existing main/router construction stays byte-identical. A broader `/api/v1/me`, a new handler wired through router/main, and global middleware were rejected because they expand the public/product boundary or shared-writer surface without serving this outcome.

The caller contract is explicit: Personal Center calls on every `onShow`; first checkout calls before deciding whether to show the binding interception. This change does not edit either caller.

### Freeze one exact, non-cacheable HTTP representation

At handler entry, set `Cache-Control: no-store` so every handled 200/400/401/503 response is non-cacheable. A GET body must be exactly empty; any received byte, including whitespace or JSON, returns exact HTTP 400 `{"error":{"code":"INVALID_REQUEST","message":"invalid request"}}` before authentication or phone reads. `Content-Type` is ignored and never required.

After body validation, reuse `exactBearer` and require exactly one `Authorization: Bearer <opaque-session-token>` value. Missing, duplicated, malformed, unknown, or expired sessions return exact HTTP 401 `{"error":{"code":"UNAUTHENTICATED","message":"authentication required"}}`. Authentication storage failure or current-user phone-read failure returns exact HTTP 503 `{"error":{"code":"PRIMARY_PHONE_STATUS_UNAVAILABLE","message":"primary phone status temporarily unavailable"}}`; the response does not distinguish row/storage/provider details.

Success is HTTP 200 with exactly two fields in stable struct order:

```json
{"primary_phone_bound":true,"masked_phone":"+*********1234"}
```

or:

```json
{"primary_phone_bound":false,"masked_phone":null}
```

The bound value is produced only by the integrated mask function. A pointer/nullable response field represents unbound explicitly as JSON `null`; `omitempty`, empty string, full phone, IDs, openid, roles and profile fields are forbidden.

### Reuse the current-user read and perform no side effects

`PhoneUser` carries both `PrimaryPhone` and `PrimaryPhoneBound`. `Repository.FindPhoneUser` keeps the existing SELECT and `WHERE id=?` query byte-identical, but maps `sql.NullString.Valid` into `PrimaryPhoneBound`. NULL plus empty is the only unbound state. Bound plus empty/malformed, or unbound plus non-empty, is inconsistent and returns sanitized unavailable; only a valid bound phone reaches the existing mask helper.

`PhoneService.Bind` applies the same state validation to its initial read before provider access. A valid bound phone retains the existing idempotent masked response; valid unbound continues into the existing provider/write flow. Any inconsistent initial state returns unavailable without provider or binding calls. The same validation fails closed on the rejected-code recovery read, without changing valid-state POST behavior.

`Repository.BindPrimaryPhone` also validates any `current.Valid` value read under its existing lock. A canonical same phone remains idempotent and a canonical different phone remains `ErrPrimaryPhoneAlreadyBound`; an empty or malformed non-NULL current value becomes persistence failure/HTTP 503. This closes the provider-call race where state becomes inconsistent between service preflight and transaction lock. It reuses the package's minimal phone validator and changes no transaction or SQL text.

Status never calls `PhoneProvider.Exchange`, `BindPrimaryPhone`, begins a transaction, or writes any row. Reusing `FindPhoneUser` was selected over new repository SQL because its `WHERE id=?` current-user lookup and existing phone representation already provide the exact needed data. Querying by client input, phone, openid, or arbitrary user ID is forbidden; the route accepts no such field.

### Prove auth isolation and no writes with real MySQL

The dedicated phone integration runner uses isolated MySQL 8 and the actual session authenticator, repository, phone service and HTTP handler. In addition to the bound/unbound/auth/failure matrix, it seeds the v10-permitted inconsistent pair `primary_phone=''` with non-NULL bound-at and proves GET 503 plus POST 503, zero provider/binding calls and byte/value-equivalent user/session state. This is required because mock data cannot prove the NULL-validity mapping.

Unit/router tests additionally freeze empty-body precedence, no `Content-Type` requirement, exact single Authorization handling, no-store on all handled outcomes, mask reuse, exact JSON null/string shapes, absent broad/compatibility routes, and unchanged POST behavior. Smoke proves the compiled binary exposes the GET with route-specific 401 while health/catalog/menu/session remain unaffected.

### Keep local and platform evidence separate

Required external assets are none. A writer-managed isolated MySQL 8 runtime is a required local W3 environment and must not be replaced by mocks. Real WeChat is `NOT_REQUIRED` because the GET must make zero provider calls. Existing phone/platform Gates continue to govern the complete bind and customer journey; no local status result can upgrade them.

## Risks / Trade-offs

- [Risk] A proxy or client caches a successful phone-state response. → Set `Cache-Control: no-store` before every handler exit and test every status class.
- [Risk] A generic profile response expands personal-data exposure. → Freeze the two-field response and dedicated path; reject IDs, openid, full phone and unrelated identity/merchant fields.
- [Risk] A future refactor accidentally calls binding/provider code from GET. → Keep Status as a read-only service operation and require zero provider/write counters plus real-MySQL before/after equality.
- [Risk] Two users receive each other's state. → Derive user ID only from Authenticate, accept no identity selector, query only that ID, and execute a two-session/two-user integration matrix.
- [Risk] Database outage or a non-NULL invalid value is confused with unbound. → Preserve SQL NULL validity and permit HTTP 200 unbound only for NULL plus empty; all inconsistent combinations use the fixed 503.

## Migration Plan

1. After explicit DRAFT approval, obtain focused handler/service/router and real-MySQL Red evidence while production code still lacks the GET/Status operation.
2. Add the minimum service status read and handler route/representation; retain NULL validity in the repository result mapping without editing migrations, provider, repository SQL, router or main.
3. Run the unchanged focused checks, isolated MySQL W3, existing POST/session/catalog/menu regressions, full Go/static/smoke, strict and owned-path Gates for Green/Refactor.
4. Form a single owned-path candidate only after all writer Gates pass, then require another clean detached exact-SHA verifier. Integration, archive, push and deployment remain separately authorized.

Rollback removes only the GET/status code and its tests/script assertions. There is no schema, data, provider or frontend rollback.

## Open Questions

None. The route, caller timing, response/error/cache contract, authorization boundary and acceptance evidence are frozen; no P1-P5 decision is touched.
