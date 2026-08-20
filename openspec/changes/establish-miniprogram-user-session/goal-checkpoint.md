# Goal checkpoint

## Frozen runner base

| field | value |
| --- | --- |
| goal | 小程序启动时以一次性 `wx.login` code 原子建立内部普通用户和 hash-only 不透明 Bearer session |
| module | `establish-miniprogram-user-session` |
| lifecycle | `CANDIDATE` |
| repo_sha | `2b83e93cc2a8d2bb16b606068028f34ee662b677` |
| skill_blob | `0f41f64ad87fc9fd410cb916b4d1562aee92e42f` |
| skill_sha256 | `2781bdda1544106e30e7483c4b500d611df85c79753cc1bd3b717d91d1edaac8` |
| runner_version | `unversioned` |
| owner | same Writer agent; branch `codex/establish-miniprogram-user-session`; worktree `/Users/vivix/.codex/worktrees/order-establish-miniprogram-user-session.Writer` |
| dependency | `serve-reservation-menu-availability` is `INTEGRATED` in exact base main; no unmet change dependency |
| required_local_external_assets | `none`; writer-managed isolated MySQL 8 W3 is a required local Gate and is PASS |
| writer_runtime | Colima profile `order-mysql-w3`; MySQL `8.0.46` container `order-session-w3`, loopback-only random port, tmpfs data |
| delivery_external_blocker | real Mini Program credentials/account/network and a fresh login value are `BLOCKED_EXTERNAL/NOT_RUN` |
| previous_candidate_sha | `dbc334a5edfde8f28bf599f86e2d49bdac3327fe` (`INVALIDATED`: `PROVIDER_GET_TRANSPARENT_REPLAY`) |
| candidate_sha | `external-post-commit`; bind replacement full SHA from Git after the immutable repair commit |
| score | `C9/T10/V8/R9=36`; every dimension at least 8; writer V capped at 8 |
| error_fingerprint | `PROVIDER_GET_TRANSPARENT_REPLAY`; fresh verifier high-risk P1; repeat_count `1` |
| hard_blockers | `0` inside replacement writer scope |
| next | commit only the six owned repair files, bind the exact clean replacement SHA, then hand it to a different detached verifier |

## Frozen boundary

- `gate_type=W3`; `ui_level_target=UI0`; `ui_level_actual=UI0`.
- Only `POST /api/v1/auth/miniprogram/session` is public. It accepts strict JSON, creates one 24-hour opaque session, and returns the raw token only in the one success response.
- v8 stores only internal ID, unique opaque provider identity and timestamps. v9 stores only SHA-256 token hash, user FK and timestamps. No phone, merchant, role, UnionID, provider session material, original login value or raw token is persisted.
- The fixed official provider origin is not configurable in main. Local stub injection is package-local and proves only wire/error behavior.
- Same-provider-identity concurrency yields one user and multiple sessions. Upsert/session insert is one MySQL transaction; collision, statement and commit failure roll back user/session/last-login changes.
- Phone, employee, merchant/RBAC, PC QR, P1/P2, protected routes/middleware, order/payment, frontend and production SSM loader remain excluded. Catalog/menu remain anonymous and exact.
- The frozen runner blob/hash above matches the exact base; no drift is recorded.

## Gate ledger

| gate | state | decisive condition |
| --- | --- | --- |
| Approval/lifecycle | `PASS` | approved scope remains unchanged; failed candidate returned `CANDIDATE → IMPLEMENTING` |
| Provider local wire/error | `REPAIR GREEN/REFACTOR PASS` | dedicated non-reusing HTTP/1.1 transport; HTTP/2-capable TLS regression proves two requests use two distinct connections |
| Service/API/config/main | `PASS` | exact service and handler/router commands pass; config package and binary smoke pass |
| MySQL W3 | `PASS` | real MySQL 8.0.46 race suite proves v1→v9/repeat, exact schema, concurrency, rollback, lookup and expiry |
| Anonymous regressions | `PASS` | mysql/catalog/menu integration scripts pass with exact anonymous catalog/menu behavior |
| Writer repository Gate | `REPLACEMENT PASS` | focused RGR, four MySQL scripts, gofmt, all test/race, vet, build, smoke, strict, Harness, diff and owned-path pass |
| Independent exact SHA | `REPLACEMENT NOT_RUN` | fresh verifier rejected old SHA; no independent result exists for the post-commit replacement |
| Real WeChat delivery | `BLOCKED_EXTERNAL/NOT_RUN` | real customer/platform assets and separately authorized non-sensitive acceptance are absent |

## Sanitized RGR and writer verdict

- Provider Red: focused compile failed because the provider constructor was absent. Minimal fixed-origin implementation made the unchanged command Green; Refactor and final replay pass.
- Session Red: focused compile failed because the service was absent. Minimal entropy/hash/TTL implementation made the unchanged command Green; Refactor and final replay pass.
- HTTP Red: focused compile failed because the issuer/handler and five-dependency router were absent. Strict handler/router made the same command Green; invalid JSON, including duplicate keys, reaches neither provider nor repository.
- Schema Red: real W3 failed because the embedded chain ended at v7. Additive one-statement v8/v9 made the same command pass twice with exact information-schema checks.
- Repository Red: W3 compile failed because the repository was absent. The minimal MySQL transaction made the unchanged race-enabled command pass for concurrency, multi-session, rollback, hash lookup and expiry.
- Config Red: focused compile failed because structured Mini Program credentials were absent. Minimal development/test config and production fail-closed checks made the same command pass twice.
- Main/smoke Red: binary build failed because identity was not wired into the extended router. Minimal fixed-client/repository/service/handler wiring made smoke pass twice without any provider request.
- Old writer PASS remains invalidated. Replacement writer score is `C9/T10/V8/R9=36` with hard blockers `0`; V stays capped at 8 until a different clean detached verifier checks the exact post-commit SHA.

## External boundary

Real official provider acceptance remains `BLOCKED_EXTERNAL/NOT_RUN`. Recovery requires customer administrator/developer-controlled real credentials, authenticated account, reachable official network, a fresh login value and separate authority to run a non-sensitive acceptance. Local HTTP stubs can never satisfy that Gate.
