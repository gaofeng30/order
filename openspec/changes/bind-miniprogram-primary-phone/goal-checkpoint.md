# Goal checkpoint

## Frozen runner base

| field | value |
| --- | --- |
| goal | 已登录小程序用户以一次性 getPhoneNumber code 绑定唯一 E.164 主手机号 |
| module | `bind-miniprogram-primary-phone` |
| lifecycle | `CANDIDATE` prepared (`DRAFT → APPROVED → IMPLEMENTING`; exact SHA is bound externally after commit) |
| repo_sha | `73cf1e04d6661ad08cbd2b03c2bb8324d4769f4d` |
| skill_blob | `0f41f64ad87fc9fd410cb916b4d1562aee92e42f` |
| skill_sha256 | `2781bdda1544106e30e7483c4b500d611df85c79753cc1bd3b717d91d1edaac8` |
| runner_version | `unversioned` |
| owner | same Writer agent; branch `codex/bind-miniprogram-primary-phone`; worktree `/Users/vivix/.codex/worktrees/order-bind-miniprogram-primary-phone.Writer` |
| dependency | integrated `establish-miniprogram-user-session` at exact base; no unmet change dependency |
| blocker | none inside implementation; real platform delivery assets are external and do not block local implementation |
| required_local_asset | writer-owned `order-phone-w3`; MySQL `8.0.46-oraclelinux9`; healthy, loopback-only dynamic port; W3 PASS |
| candidate_sha | external post-commit evidence; the immutable commit cannot contain its own object ID |
| integrated_sha | none |
| archive_sha | none |
| score | `C9/T10/V8/R9=36`; every dimension at least 8; writer V capped at 8 |
| error_fingerprint | none active; timeout fixture and minimum-E.164 boundary findings are repaired |
| repeat_count | `0` |
| next | commit only owned paths, bind exact clean SHA in Harness, then hand off to a different detached verifier |

## Boundary

- One outcome: route-specific authenticated primary-phone binding through only `POST /api/v1/me/bind-phone`.
- `gate_type=W3`; `ui_level_target=UI0`; `ui_level_actual=UI0` because no frontend/UI path exists in scope.
- Required local asset during implementation: writer-managed isolated real MySQL 8; mock is insufficient.
- Delivery external Gate: authenticated non-personal Mini Program subject, phone capability/quota, real credentials, official network and fresh user-click code are `BLOCKED_EXTERNAL/NOT_RUN`.
- Owned paths, read-only contracts, dependency, non-goals and single ACCEPT/REJECT verdict are frozen in `proposal.md`.
- Migration, `internal/wechat/**`, router and main have one writer and MUST NOT have a parallel writer until this change leaves the lane.
- This module uses the frozen runner identity above; it cannot silently switch to another runner.

## Official contract review

- Stable token: fixed official POST `/cgi-bin/stable_token`, ordinary `force_refresh=false`, response `access_token/expires_in`, with platform-recommended early renewal behavior.
- Phone exchange: fixed official POST `/wxa/business/getuserphonenumber?access_token=...`, body requires code and currently supports optional openid for code/openid binding validation; response includes phone info and watermark AppID.
- Phone code: five minutes, single consumption; phone code is distinct from `wx.login` code.
- Capability: current client documentation limits it to authenticated non-personal subjects and records experience/paid quota, so real qualification/quota remains an external Gate.
- No official fact invalidates the selected design. If official primary docs remove/change openid support before implementation, return to DRAFT.

## Current verdict

- Main session approved the frozen contract; the same Writer completed real Red → Green → Refactor for provider, route/service, v10/MySQL repository and main/smoke slices.
- Fixed official provider wire, process-local concurrent token refresh merge/cache, five-minute early refresh, invalid-token eviction without current-code replay, openid/watermark validation and E.164 normalization pass controlled tests. Local stubs do not prove real WeChat.
- The single v10 ALTER and real MySQL 8 race suite prove v1→v10/repeat, exact nullable binary phone/time pair, unique/CHECK, existing NULL rows, first bind, same-phone/time preservation, same-user and cross-user concurrency, statement/commit rollback and same-code recovery. No transaction spans provider I/O.
- The only public addition is route-specific `POST /api/v1/me/bind-phone`; exact success/error envelopes, already-bound provider bypass, absence of global middleware/profile/compatibility routes and unchanged anonymous health/catalog/menu/session behavior pass.
- Final writer Gate passes: all five MySQL integration scripts, gofmt, full Go test/race, vet, build, smoke, strict, owned/read-only/sensitive/route/migration audits and frozen runner identity. `C9/T10/V8/R9=36`, hard blockers `0`.
- No push, integration, archive, deploy, real provider call or production/external write was performed. Exact candidate SHA is external post-commit evidence; independent verification remains `NOT_RUN`.

## Observations

- A test-only timeout fixture initially waited for a server request-context cancellation that was not observable without server I/O; a local release channel fixed cleanup without weakening timeout/one-attempt assertions.
- Final review found the one-digit E.164 lower bound was rejected by masking; a focused Red preceded the one-byte-boundary fix and unchanged-command Green/Refactor.
