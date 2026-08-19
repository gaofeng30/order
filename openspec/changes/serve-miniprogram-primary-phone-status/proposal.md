## Why

The integrated backend can create a Mini Program session and bind a primary phone, but an authenticated caller cannot read whether the current user is already bound without attempting the mutating bind flow. Personal Center `onShow` and the first-checkout preflight need one narrow read contract so they can render or gate from server truth without exposing a full profile or consuming a WeChat phone code.

## What Changes

- Add only `GET /api/v1/me/primary-phone`, protected by the integrated route-specific Bearer session lookup; do not add `/api/v1/me`, a profile route, or global auth middleware.
- Return exact bound/unbound HTTP 200 representations with `Cache-Control: no-store`: bound returns `{"primary_phone_bound":true,"masked_phone":"+..."}` using the integrated mask rule, while unbound returns `{"primary_phone_bound":false,"masked_phone":null}`.
- Reject any non-empty GET body as HTTP 400 without requiring `Content-Type`; require exactly one valid Authorization header; map missing/malformed/unknown/expired sessions to 401 and authentication/phone persistence failures to one sanitized 503.
- Reuse v10, `identity.Repository.FindPhoneUser`, the existing mask rule, and `identity.Service.Authenticate`. Preserve `sql.NullString.Valid` in the read model so NULL is the only unbound state; a non-NULL empty or malformed phone fails closed instead of becoming a false unbound result. The repository SQL remains byte-identical.
- The GET performs no WeChat/provider call, transaction, or database write and isolates every result to the authenticated user. Preserve valid-state POST bind behavior; its existing preflight must also fail closed with no provider/write call when the stored NULL-validity/phone value is inconsistent. Preserve session creation, catalog, and menu contracts exactly.

## Capabilities

### New Capabilities

- `miniprogram-primary-phone-status`: Defines the authenticated, read-only primary-phone status route, exact representations, isolation, failure semantics, cache control, and no-side-effect boundary.

### Modified Capabilities

None.

## Impact

- Primary outcome: Personal Center calls the route on every `onShow`, and first checkout calls it as a preflight before deciding whether to enter the existing binding flow; both receive only the current user's bound boolean and optional masked phone.
- Owner: same Writer agent; branch `codex/serve-miniprogram-primary-phone-status`; worktree `/Users/vivix/.codex/worktrees/serve-miniprogram-primary-phone-status/order`.
- Base SHA: `a5728022c8e497947267f1b8db5ff50983c03be9`.
- Owned paths:
  - `openspec/changes/serve-miniprogram-primary-phone-status/**`
  - `services/api/internal/identity/phone_handler*.go`
  - `services/api/internal/identity/phone_repository.go`
  - `services/api/internal/identity/phone_service*.go`
  - `services/api/internal/identity/mysql_integration_test.go`
  - `services/api/internal/httpapi/router_test.go`
  - `services/api/scripts/miniprogram-phone-integration.sh`
  - `services/api/scripts/smoke.sh`
- Read-only shared contracts: root governance and quality/self-evolution rules; 0818 PRD sections 4.1, 5.9 and 15.6.6; adopted product-baseline delta; integrated session and primary-phone OpenSpec artifacts; migrations v8-v10; the SQL text and query boundary inside `identity.Repository.FindPhoneUser`; `identity.Service.Authenticate`; valid-state POST mask/bind behavior; `internal/wechat/**`; `httpapi/router.go`; `cmd/order-api/main.go`; and current session/catalog/menu public contracts.
- Dependencies: exact base main already integrates `establish-miniprogram-user-session@73cf1e04d6661ad08cbd2b03c2bb8324d4769f4d` and `bind-miniprogram-primary-phone@a5728022c8e497947267f1b8db5ff50983c03be9`; no unmet change dependency exists.
- Parallel boundary: this change is the sole writer for its owned paths, including repository result mapping and post-read state validation only. It does not own or require a writer lock on migrations, `internal/wechat/**`, `httpapi/router.go`, or `cmd/order-api/main.go`.
- Gate: `gate_type=W3` because authentication, persisted phone state, database failure, no-write, and cross-user isolation are acceptance-critical; `ui_level_target=UI0`, with DRAFT `ui_level_actual=NOT_RUN` because no frontend/UI path is owned.
- Required external assets: none. Implementation must use a writer-managed isolated real MySQL 8 runtime as a local W3 Gate. Real WeChat is `NOT_REQUIRED` for this read-only change because the GET must never call it; the complete bind/user journey remains subject to the already recorded phone/platform delivery Gates and this change cannot promote them to PASS.
- Non-goals: changing valid-state `POST /api/v1/me/bind-phone`, provider/token cache or phone-code behavior; changing repository SQL; migrations; additional phone/P4; employee/whitelist/discount/P5; merchant/PC/P3; profile/avatar/nickname; checkout/order/payment; frontend; global middleware; compatibility routes; P1/P2 product scope; push, PR, integration, archive, deploy, production/external writes, or real-platform calls.
- Single acceptance verdict: the local candidate is `ACCEPT` only when the exact GET contract, no-store header, exact Authorization/body handling, NULL-only unbound semantics, invalid-state GET/POST fail-closed behavior, bound masking, unknown/expired sessions, auth/phone database failures, two-user isolation, no provider/no writes, real-MySQL W3 evidence, existing valid POST/session/catalog/menu regression, strict validation, owned-path audit and all writer Gates pass. Any miss is `REJECT`; local `ACCEPT` does not prove the complete WeChat journey or platform readiness.
