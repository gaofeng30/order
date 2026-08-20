# Goal checkpoint

## Frozen runner base

| field | value |
| --- | --- |
| goal | 按预约日期与离散时间返回普通价菜单、餐段截单和日期级售罄可购买事实，并保留 catalog |
| module | `serve-reservation-menu-availability` |
| lifecycle | `CANDIDATE` |
| repo_sha | `babd1ef662811e3df6a75aa28995268352531438` |
| skill_blob | `0f41f64ad87fc9fd410cb916b4d1562aee92e42f` |
| skill_sha256 | `2781bdda1544106e30e7483c4b500d611df85c79753cc1bd3b717d91d1edaac8` |
| runner_version | `unversioned` |
| owner | branch `codex/serve-reservation-menu-availability`, worktree `/Users/vivix/.codex/worktrees/order-serve-reservation-menu-availability.Writer` |
| dependency | `adopt-0818-prd-baseline@babd1ef662811e3df6a75aa28995268352531438` is `INTEGRATED` on current main and not archived |
| required_external_assets | `none` |
| writer_runtime | `order-mysql-w3=ESTABLISHED`; canonical foundation real MySQL W3 preflight=`PASS`; retained for writer tests and later exact-SHA verifier rebuild |
| blocker | none; hard blockers `0` |
| candidate_sha | external post-commit evidence; exact full SHA must be reported from the clean committed worktree |
| integrated_sha | none |
| archive_sha | none |
| error_fingerprint | repaired once each: `RUNTIME_ENV_TLS_MODE_FALSE`, `GATE_SHELL_SPECIAL_PATH_VARIABLE`; both same-command retries passed |
| repeat_count | `0` |
| next | commit only owned paths, bind the exact full candidate SHA externally, and hand off for fresh independent verification |

## Boundary

- `gate_type=W3`; `ui_level_target=UI0`; `ui_level_actual=UI0` from API/static plus real database evidence; no UI exists and no frontend behavior is claimed.
- Primary outcome and the single ACCEPT/REJECT verdict are frozen in `proposal.md`.
- Owned paths and read-only contracts are frozen in `proposal.md`; migration/router/main paths have one writer and no parallel Order writer is allowed.
- Public menu contract：only anonymous `GET /api/v1/menu?date=YYYY-MM-DD&time=HH:MM`；today/tomorrow in `Asia/Shanghai`；离散点、餐段与 cutoff 必须由当前 `meal_periods` 两行配置计算。初始数据为 lunch `11:30/11:30–13:30/30`、dinner `17:00/17:00–19:00/30`，但不是代码常量或 fallback；合法数据调整不改变公共路径、字段或错误语义。
- Browseability and orderability are separate: a cut-off meal still returns `200` menu with `meal.orderable=false`; date D sold-out products remain visible with `sold_out=true,orderable=false` and D+1 is unaffected.
- 配置 TIME 必须为同一营业日 `00:00..23:59` 且秒=0；负数、跨日、秒级值、缺失、重复、重叠或其他非法配置统一稳定 503 且不 fallback。明显格式/date 错误 DB 前 400，格式合法 time 是否为离散点由当前配置判定。
- P1, P2–P4, P5, all merchant config/sold-out writes and auth, special closure dates, checkout/payment/order, quantity inventory, frontend, compatibility routes and external writes are excluded.
- This module uses the frozen runner above. Runner identity or acceptance-contract changes must be recorded and invalidate prior evidence; they cannot silently change the rules.

## Current verdict

- 2026-08-20 主会话在复核 DB-driven `meal_periods`、同营业日分钟点不变量和 fail-closed 契约后明确批准本 change；lifecycle 已按授权记录 `DRAFT → APPROVED → IMPLEMENTING`，没有扩展 push、PR、deploy、integration 或 archive 权限。
- IMPLEMENTING 入口证据：HEAD 与 main 均为 `babd1ef662811e3df6a75aa28995268352531438`，该 SHA 包含依赖的最终候选 bytes；`openspec validate serve-reservation-menu-availability --strict` exit `0`（`Change 'serve-reservation-menu-availability' is valid`），`git diff --check` exit `0`。index/HEAD 未含业务改动，当前 working changes 仅为本 change 的 owned OpenSpec artifacts。
- Writer runtime PASS：Colima 0.10.3 stable bottle digest、`order-mysql-w3` aarch64/VZ/Docker/2 CPU/4 GiB/10 GiB/no mounts/no network address 均核验；Official Registry live enumeration 仍选择 `8.0.46-oraclelinux9`、manifest `sha256:7dcddc01...8ae2b`、arm64 `sha256:213bbfaf...2458e`。exact-digest 容器为 healthy、127.0.0.1 随机端口、0600 随机凭据、noexec/nosuid 1 GiB tmpfs、zero mounts、MySQL 8.0.46、随机 schema 残留 0。
- Foundation integration first failed only because runtime env used unsupported TLS value `false`; it was changed to existing contract value `disabled`, and the identical `GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/mysql-integration.sh` passed (`internal/migrate`, race, exit 0). This proves the pre-Red runtime, not menu behavior.
- Config/selection, migrations, repository/date sold-out, handler/router and catalog preservation each retain observable Red followed by same-command Green/Refactor evidence in `tasks.md`.
- Writer Gate PASS：strict、untracked-aware owned paths、read-only catalog/v1–v3 bytes、whitespace/gofmt、唯一 route/script mode、全 API test/race/vet/build、smoke、foundation/catalog/menu integration 全部 exit 0；MySQL 容器仍 healthy、loopback、zero mounts，随机 schema 残留 0。
- `C9/T10/V8/R9=36`; hard blockers `0`. V is capped at 8 because exact-SHA independent verification remains `NOT_RUN`.
- Candidate commit、independent verification、integration 和 archive 尚未发生；checkpoint is prepared as `CANDIDATE` and exact SHA is external post-commit evidence.
- No external system, push, PR, deployment, integration or archive action has occurred. The only database action is the approved writer-owned disposable W3 runtime and isolated test schemas, which cleaned to zero residue; the profile is intentionally retained for verifier rebuild per foundation contract.

## Observations

None.
