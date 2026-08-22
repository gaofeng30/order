# implement-storefront-settings-backend-v11

## DRAFT 状态与目标

- 状态：`IMPLEMENTING`；DRAFT 已在首个产品测试前建立，本次 delegated instruction 已明确授权实现；Matt tracker 尚待用户确认且不影响本地实现授权。
- tracker：`BLOCKED_LOCAL_GOVERNANCE`。仓库缺少 `docs/agents/issue-tracker.md`，本 change 不创建或更新 GitHub Issue，也不把 tracker 状态伪装为 `READY`。
- 唯一目标：建立 v11 单门店公开设置持久化与可独立挂载的匿名 provider seam。
- 最小成功标准：v11 schema、repository/model 与独立 GET provider 满足冻结契约；真实隔离 MySQL 8.0、focused/full/race/vet/build/smoke、双轴 Review 和 exact-SHA detached 验证全部通过。

## Gate 声明

- `gate_type`: `W3`（新增持久化 schema，取最高风险）。
- `ui_level_target`: `UI0`。
- `ui_level_actual`: `UI0`（纯后端；不宣称浏览器、小程序、真机或生产运行）。
- owner: `implement-storefront-settings-backend-v11` independent Writer Session。
- worktree: `/Users/vivix/.codex/worktrees/44ff/order`。
- branch: `codex/implement-storefront-settings-backend-v11`。
- `base_sha`: `70522ed01b08aaf5dad77f51f1bfbc431bb84530`。
- base branch: `codex/order-delivery-integration`，开工时已核验精确指向 `base_sha`。
- read-only reference: frozen WIP exact SHA `69431230ffb96875037501d8530ea6fe8d118496`；其代码、PASS、Red 和 Review 均不作为本 change 证据。
- `candidate_sha`: `not-yet-created`。
- dependencies: 仅固定代码基线与 canonical PRD §5.1、§6.9、§13.3、§14、§16.5；无其他实现 change 依赖。
- downstream dependency: root router/main wiring 由主控后续独立 integration change 串行完成，本 change 不修改共享 composition。

## Owned / read-only / non-goals

唯一 owned paths：

- `.scratch/implement-storefront-settings-backend-v11/**`
- `services/api/migrations/000011_create_storefront_settings.sql`
- `services/api/internal/storefront/**`
- `services/api/migrations/embed_test.go`，仅 v11 embed/chain 断言
- `services/api/internal/catalog/migrations_test.go`，仅 exact migration list 末尾追加 v11

只读共享契约：

- `AGENTS.md`
- `CONTEXT.md`
- `docs/quality/change-quality-gates.md`
- `docs/product/online-ordering-system-prd-0818.md` 的指定章节
- `services/api/internal/httpapi/router.go`
- `services/api/cmd/order-api/main.go`
- `apps/**`、package files、其他 migrations、`openspec/**` 与其他 change scratch

明确非目标：root router/main 接线、小程序、PC/Admin 写 API、COS、订单、支付、RBAC、seed/cache/mock fallback、兼容字段、push/PR/deploy/staging/main integration、微信/腾讯云/生产访问或写入。

## 冻结公共契约

- 表 `storefront_settings` 只有 `id=1` 合法；不 seed。
- 字段为 `store_name`、`store_address`、`pickup_point`、`announcement`、`business_status`，以及 `launch_png_url`、`center_x`、`center_y`、`width_ratio`、`aspect_ratio`。
- `business_status` 仅 `open|closed|cutoff`。
- 三个必填文本为合法 UTF-8、trimmed、非空；`announcement` 为合法 UTF-8 且最多 1000 rune。
- launch 五字段必须全 NULL 或全非 NULL。URL 必须为 HTTPS PNG、无 userinfo/fragment、端口为空或 `1..65535`；中心坐标在 `[0,1]`，宽比在 `(0,1]`，宽高比 `>0`。
- 缺行、DB/scan 错误、非法持久值、部分 launch 组全部 fail closed。
- `GET /api/v1/storefront/settings` 成功 exact DTO 外层仅 `settings`；无图层时 `launch_layer:null`，有图层时仅 `png_url,center_x,center_y,width_ratio,aspect_ratio`。
- 所有失败 exact `503`：`{"error":{"code":"STOREFRONT_UNAVAILABLE","message":"storefront temporarily unavailable"}}`。
- provider 匿名且只读；无认证副作用、缓存、seed、mock fallback 或兼容字段。

## 已确认 TDD seams

1. HTTP provider seam：调用方把 `Handler` 独立挂载到 Gin engine，以匿名 GET 观察 exact 200 DTO 或 exact 503；不接 root router/main。
2. Repository/model seam：调用方通过 `Repository.Get(context.Context)` 读取 id=1；观察合法设置，或对缺行、DB/scan、非法文本/枚举/launch 值的统一 fail-closed error。
3. Migration seam：embedded v1-v11 exact chain 在真实隔离 MySQL 8.0 首次应用、重复应用、schema/check/singleton/all-or-none 不变量可执行验证。

## Red → Green → Refactor

- Red：先增加一个当前 tracer test，再运行 `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/storefront -run '<当前 tracer>' -count=1`；必须因 v11/provider/model/repository 目标行为在新基线缺失而真实 FAIL，并记录首个决定性错误。
- Green：每次只加入使当前 tracer 通过的最小实现；同一命令 PASS 后再进入下一 vertical slice。
- Migration Red/Green：先让现有 embed/chain 测试精确期待 v11 并真实失败，再增加唯一 v11 migration，使同一测试通过。
- Refactor：仅在所有 slices Green 后收敛命名/重复，重跑全部 focused tests、真实 MySQL、race 与完整 Writer Gate。

## Writer / verifier / integration 可执行命令

- focused：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/storefront ./services/api/migrations ./services/api/internal/catalog -count=1`。
- focused race：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/internal/storefront ./services/api/migrations ./services/api/internal/catalog -count=1`。
- 真实 MySQL：启动仅监听 loopback 的临时 `mysql:8.0.46-oraclelinux9` container，注入仓库规定的七个 `ORDER_TEST_MYSQL_*` 变量，运行 `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/internal/storefront -run '^TestStorefrontMySQL8Integration$' -count=1 -timeout=3m`，随后清理 container；不得跳过或用 sqlmock 冒充。
- full：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/... -count=1`。
- full race：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/... -count=1`。
- static/build：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go vet ./services/api/...`；`GOPROXY=off GOTOOLCHAIN=go1.26.5 go build ./services/api/...`；`test -z "$(gofmt -l services/api/internal/storefront services/api/migrations/embed_test.go services/api/internal/catalog/migrations_test.go)"`。
- smoke：`GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/smoke.sh`。
- diff/ownership：`git diff --check 70522ed01b08aaf5dad77f51f1bfbc431bb84530...HEAD`；`git diff --name-only 70522ed01b08aaf5dad77f51f1bfbc431bb84530...HEAD` 只能命中上述 owned paths。
- Review：功能提交后固定 `git diff 70522ed01b08aaf5dad77f51f1bfbc431bb84530...HEAD` 与 `git log 70522ed01b08aaf5dad77f51f1bfbc431bb84530..HEAD --oneline`，按 `$code-review` 并行运行 Standards/Spec 两轴；任一 finding 由 Writer 修复、重新跑完整 Writer Gate、形成新 commit，并从头重做两轴 Review。
- Verifier：在 fresh clean detached worktree 对 exact candidate SHA 只读重跑上述 focused/real-MySQL/full/race/vet/build/smoke/diff/owned/clean Gate；不得修改业务文件。
- Integration：本 change 不执行。只有依赖满足、exact-SHA independent PASS 且主控获得单独集成授权后，才能由主控处理 shared router/main wiring 与 staging/main。
- OpenSpec：`N/A`；`openspec/**` 是只读历史且不在 owned paths，不伪造 validation PASS。

## 外部资产与恢复

| 资产 | owner | 当前状态 | 恢复条件 |
| --- | --- | --- | --- |
| 临时隔离 MySQL 8.0.46 | Writer/Verifier | `AVAILABLE_LOCAL`；Writer W3 latest rerun: `PASS`，临时 container 与凭据已清理 | Candidate 或实现/验证命令变化后启动独占临时 container，经 TCP readiness 后从头重跑全部真实数据库场景并清理 |
| Matt issue tracker | workflow owner / 用户 | `BLOCKED_LOCAL_GOVERNANCE` | 用户确认并配置 `docs/agents/issue-tracker.md` 后另行建立 linkage；不得追溯伪造本 change READY |
| 正式 storefront 数据、COS PNG、生产 DB/流量 | 客户/平台/UAT owner | `BLOCKED_EXTERNAL` 且非本 change Candidate Gate | 经单独授权提供脱敏资产、环境和验收标识后，由后续集成/UAT change 验证 |

## 未验证边界与恢复

- 本 change 只证明后端 provider seam 可独立挂载；root router/main 未接线，因此不宣称真实服务路径已暴露。
- 不 seed，因而候选运行时默认缺行并按契约 503；正式配置写入属于后续受权流程。
- 不验证客户端消费、COS 上传、真实 PNG、认证系统、生产数据库、部署或业务流量。
- rollback：未集成前删除本 branch/worktree 即可；集成后 migration 不提供 down，恢复方式由 integration/runbook 在未写入正式数据前回退部署，禁止本 change 自行改生产 schema。
