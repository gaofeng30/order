## Context

Exact base `a5728022c8e497947267f1b8db5ff50983c03be9` already contains v10 primary-phone storage, `Repository.FindPhoneUser`, the shared masking rule, hash-only `Service.Authenticate`, and route-specific `POST /api/v1/me/bind-phone`. The current handler is already registered by the complete router, so another method can be registered in `PhoneHandler.RegisterRoutes` without editing `httpapi/router.go` or `cmd/order-api/main.go`.

0818 PRD section 4.1 forbids startup phone authorization and requires binding before first order submission; section 5.9 requires Personal Center to show masked phone or a binding entry. The read contract therefore has exactly two callers: Personal Center on every `onShow`, and first checkout as a preflight before entering the existing binding flow. It is not a general user profile.

This change is `gate_type=W3` and `ui_level_target=UI0`; DRAFT `ui_level_actual=NOT_RUN`. The W3 classification comes from authenticated persisted-state isolation and the requirement to prove no writes and database-failure semantics on real MySQL, not from any migration or mutation in this change.

## Goals / Non-Goals

**Goals:**

- Serve one exact, non-cacheable bound/unbound representation for the authenticated Mini Program user.
- Reuse the integrated Bearer parser/authenticator, phone read, and mask rule without adding provider, transaction, or write behavior.
- Prove strict cross-user isolation, unknown/expired session behavior, auth/phone database failure mapping, and no side effects on real MySQL.
- Keep the integrated bind/session/catalog/menu contracts unchanged.

**Non-Goals:**

- Full `/api/v1/me` profile, avatar, nickname, internal user ID, openid, full phone, employee/visitor, discount, merchant or PC state.
- Any change to binding, getPhoneNumber, stable-token cache, migrations, repository SQL, router/main wiring, global middleware, frontend, checkout, order, payment, P1-P5, or external platform state.

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

`PhoneService.Status` calls only `PhoneStore.FindPhoneUser(ctx, authenticatedUserID)`. It validates the same nonzero user/provider-identity invariant used by Bind, returns unbound when v10 `primary_phone` is NULL/empty, and calls the existing mask helper only for a bound value. Any store or invariant failure becomes the sanitized unavailable result.

Status never calls `PhoneProvider.Exchange`, `BindPrimaryPhone`, begins a transaction, or writes any row. Reusing `FindPhoneUser` was selected over new repository SQL because its `WHERE id=?` current-user lookup and existing phone representation already provide the exact needed data. Querying by client input, phone, openid, or arbitrary user ID is forbidden; the route accepts no such field.

### Prove auth isolation and no writes with real MySQL

The dedicated phone integration runner will add a status suite using isolated MySQL 8 and the actual session authenticator, phone service and HTTP handler. It must create two internal users/sessions with different states, then prove exact bound/unbound responses, session unknown/expiry rejection, two-user isolation, auth-database and phone-database unavailable 503s, and byte/value-equivalent user/session state before and after all successful/error reads. A provider spy and store write spy must remain at zero.

Unit/router tests additionally freeze empty-body precedence, no `Content-Type` requirement, exact single Authorization handling, no-store on all handled outcomes, mask reuse, exact JSON null/string shapes, absent broad/compatibility routes, and unchanged POST behavior. Smoke proves the compiled binary exposes the GET with route-specific 401 while health/catalog/menu/session remain unaffected.

### Keep local and platform evidence separate

Required external assets are none. A writer-managed isolated MySQL 8 runtime is a required local W3 environment and must not be replaced by mocks. Real WeChat is `NOT_REQUIRED` because the GET must make zero provider calls. Existing phone/platform Gates continue to govern the complete bind and customer journey; no local status result can upgrade them.

## Risks / Trade-offs

- [Risk] A proxy or client caches a successful phone-state response. → Set `Cache-Control: no-store` before every handler exit and test every status class.
- [Risk] A generic profile response expands personal-data exposure. → Freeze the two-field response and dedicated path; reject IDs, openid, full phone and unrelated identity/merchant fields.
- [Risk] A future refactor accidentally calls binding/provider code from GET. → Keep Status as a read-only service operation and require zero provider/write counters plus real-MySQL before/after equality.
- [Risk] Two users receive each other's state. → Derive user ID only from Authenticate, accept no identity selector, query only that ID, and execute a two-session/two-user integration matrix.
- [Risk] Database outage is confused with unbound. → Only a successful current-user read can return `primary_phone_bound:false`; any auth/read error is the fixed 503.

## Migration Plan

1. After explicit DRAFT approval, obtain focused handler/service/router and real-MySQL Red evidence while production code still lacks the GET/Status operation.
2. Add the minimum service status read and handler route/representation; do not edit migrations, provider, repository SQL, router or main.
3. Run the unchanged focused checks, isolated MySQL W3, existing POST/session/catalog/menu regressions, full Go/static/smoke, strict and owned-path Gates for Green/Refactor.
4. Form a single owned-path candidate only after all writer Gates pass, then require another clean detached exact-SHA verifier. Integration, archive, push and deployment remain separately authorized.

Rollback removes only the GET/status code and its tests/script assertions. There is no schema, data, provider or frontend rollback.

## Open Questions

None. The route, caller timing, response/error/cache contract, authorization boundary and acceptance evidence are frozen; no P1-P5 decision is touched.
