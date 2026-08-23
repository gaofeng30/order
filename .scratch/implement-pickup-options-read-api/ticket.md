# implement-pickup-options-read-api

## 状态与唯一结果

- lifecycle: `PRE_CANDIDATE_WIP`；实现尚未形成 candidate，当前历史运行证据已因 replay/Gate/治理工件变化失效。
- governance: `GOVERNANCE_PENDING`；`docs/agents/issue-tracker.md` 缺失，本 change 不伪造 tracker/ticket linkage，也不创建外部 issue。
- 唯一结果：新增匿名只读 `GET /api/v1/menu/pickup-options`，从现有 `Reader.MealPeriods` 一次性读取完整配置，并用一次注入 clock 投影上海今天、明天的完整午/晚餐取餐选项。
- 最小成功标准：冻结的 provider/consumer/error contract、8 个 mutation、fresh MySQL 8.0.46、focused/full/race/vet/build/smoke、现有 UI1、双轴 review 与 clean detached exact-SHA verifier 全部通过。

## Gate 与身份

- `gate_type`: `W2`（新增公共 HTTP contract）。
- `ui_level_target`: `UI1`。
- `ui_level_actual`: `NOT_RUN`（current）。历史 receipt `PO-08-ui1` 曾观察到 Chrome for Testing `151.0.7922.34`、`TOTAL 3 SUCCESS`，但在后续 replay/Gate/receipt/spec/tasks 变化后统一标记为 `INVALIDATED_NOT_CURRENT`；只有最终冻结后重新运行才可恢复为 current UI1。
- owner: `implement-pickup-options-read-api` independent Writer Session。
- worktree: `/Users/vivix/.codex/worktrees/869e/order`。
- branch: `codex/implement-pickup-options-read-api`。
- `base_sha`: `f3c4efa4cd665652d93d5da76f92d18c4bdc59ac`。
- base branch: `codex/order-delivery-integration`。
- `candidate_status`: `NOT_CREATED`；提交完整候选后才增加 immutable `candidate_sha`。
- historical reference: `codex/serve-reservation-menu-availability` 的 `2b83e93` 已在 base ancestry 中；该历史 worktree/branch 只读且不复用其旧证据。

## 依赖、owned、read-only、非目标

依赖仅为 exact base 已集成的 `menu.Handler`、`Reader.MealPeriods`、配置校验与 migration v1-v13。没有相邻 change、客户端或外部系统依赖。

唯一 owned paths：

- `.scratch/implement-pickup-options-read-api/**`
- `services/api/internal/menu/handler.go`，仅注册新 route 与最小构造接线
- `services/api/internal/menu/pickup_options.go`
- `services/api/internal/menu/pickup_options_test.go`
- `services/api/internal/menu/mysql_integration_test.go`，仅本 change MySQL 场景
- `services/api/internal/httpapi/pickup_options_test.go`

read-only：`services/api/internal/menu/repository.go`、`services/api/internal/catalog/**`、`services/api/internal/storefront/**`、`services/api/internal/httpapi/router.go`、`services/api/cmd/order-api/main.go`、`services/api/migrations/**`、`services/api/scripts/menu-integration.sh`、`apps/wechat-miniprogram/**`、`tools/miniprogram-ui/**`、`go.mod`、`go.sum`、`services/api/internal/staffidentity/**` 及其他 change scratch。

明确非目标：storefront business status、特殊日期、商品/售罄、身份、报价、支付、订单、最终 checkout 许可、小程序 picker 接线、新 repo/port/migration、`000014`、缓存/seed/fallback、push/PR/deploy/integration/生产或外部写入。

## 已确认 seams 与 Module 设计

1. 公共 HTTP seam：匿名调用 exact GET path，观察 exact 200 DTO + `Cache-Control: no-store`，或统一 exact 503；现有 `/api/v1/menu` contract 保持不变。
2. 现有 `Reader.MealPeriods` seam：每个请求恰好一次完整读取；不调用 `Reader.List`，不增加任何 port/adapter。
3. 注入 clock seam：每请求恰好调用一次，再转换到 `Asia/Shanghai`；测试使用固定时钟。
4. fresh MySQL seam：真实 v1-v13 schema 上通过现有 Repository/Handler 的 HTTP interface 观察结果，并只为“前后配置未变化”执行只读快照查询。

`pickup_options.go` 是深 Module implementation：在 package 内复用现有完整配置 validator，把稳定排序、闭区间枚举、两日 cutoff/orderable 投影与 fail-closed 结果集中在一个小 interface 后；不新增浅转发层。

## Red -> Green -> Refactor

- Red 1：exact route 在 base 返回 404。
- Red 2：独立 replay 只从成功 DTO 删除一个内部已配置取餐时点；专属 `TestPickupOptionsEnumeratesEveryConfiguredPickupTime` 单断言观察 `12:00` 缺失并 exit 1。该 replay 保留 route、上海日期、闭区间终点、配置 interval 与 cutoff，不与另外五条 Red 重叠。
- Red 3：上海 today/tomorrow、exact cutoff、闭区间 end、非默认 interval 各由命名断言真实失败。
- Green：只增加 `pickup_options.go` 与 `handler.go` 最小 route/implementation，使同一 contract tests 通过。
- Refactor：不改变 interface，重跑相同 contract、race、determinism、mutation、MySQL 与回归 Gates。

## Writer / verifier / integration 命令

- focused：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/menu ./services/api/internal/httpapi -count=1`
- focused race：同范围加 `-race`。
- determinism：同范围 `-count=20`。
- mutation：`.scratch/implement-pickup-options-read-api/verify-mutation-gate.sh`；必须先证明 infrastructure shield，再杀死 8 个命名 mutant。
- fresh MySQL 8.0.46 loopback：`.scratch/implement-pickup-options-read-api/verify-mysql.sh` 只负责临时 loopback 资产与清理，内部精确执行 `GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/menu-integration.sh`。
- full：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/... -count=1`；同范围 `-race -count=1`。
- static/build：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go vet ./services/api/...`；`GOPROXY=off GOTOOLCHAIN=go1.26.5 go build ./services/api/...`。
- controlled smoke：`GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/smoke.sh`，以及 public HTTP contract test。
- format/diff/owned/sensitive：`gofmt`、`git diff --check`、exact owned-path allowlist、敏感 token/DSN/个人数据扫描。
- UI1：`.scratch/implement-pickup-options-read-api/verify-ui1.sh` 先核对 exact staging 与当前 `package.json` / `package-lock.json` hash，临时链接其已锁定 `node_modules` runtime asset，执行原始 `npm --prefix tools/miniprogram-ui run ui1` 后移除 symlink；不改 tracked `tools/**` / `apps/**`。该结果只证明现有邻接小程序菜单主场景/503 恢复无回归，不证明未来 pickup-options consumer 已接入。
- Review：提交后固定 `git diff f3c4efa4cd665652d93d5da76f92d18c4bdc59ac...HEAD` 与对应 log，按 `$code-review` 并行 Standards/Spec 两轴；finding 必须回 Writer 修复、形成新 SHA、重跑全 Gate 和两轴 review。
- Verifier：在另一 fresh clean detached worktree 对 exact candidate SHA 只读重跑全部声明 Gates；任何实现/spec/tasks/SHA 变化使旧 receipt 失效。
- Integration：本 Session 不执行。只有主控获得单独授权后才可处理。
- OpenSpec：`N/A`；`openspec/**` 是只读历史，不伪造 validate PASS。

### Post-freeze Writer Gate

1. 先把 ticket/spec/tasks、RGR replay 与全部 Writer Gate 脚本固定并 stage，对这个 staged snapshot 运行 Standards/Spec pre-review；两轴必须零 finding。
2. 执行首轮 exact command `.scratch/implement-pickup-options-read-api/verify-writer.sh`。该脚本检查 committed base diff、staged、unstaged 与 untracked 的完整 changed-path，拒绝未 stage/untracked source，同时执行 `git diff --cached --check`、unstaged diff check 与全部声明的重型 Gate；前后完整 staged index 的 `source_tree_sha256` 必须一致。
3. 只有首轮命令实际 exit 0 并保存 exact receipt 后，才可勾选 `PO-08` 并把 receipt 写入 tasks。此写入使首轮 pre-review 与 Writer Gate 同时失效；必须 stage 最终治理树，再对新的最终 staged snapshot 重跑 Standards/Spec 双轴 pre-review并取得零 finding。
4. 双轴零 finding 后，对完全相同的最终 staged tree 再运行 Writer Gate。最后一次 exact exit-0 terminal receipt 不反写治理文件，随后才允许 commit candidate。
5. 任一实现、ticket/spec/tasks、replay/Gate、staged tree 或 receipt 变化都会使此前相应的 pre-review/Writer Gate 失效并从受影响顺序重来。当前状态：`NOT_RUN_AFTER_FINAL_FREEZE`；历史 Writer/UI1/MySQL 结果均为 `INVALIDATED_NOT_CURRENT`。

## 外部资产与边界

| asset | owner | current status | recovery |
| --- | --- | --- | --- |
| loopback MySQL 8.0.46 | Writer/Verifier | `AVAILABLE_LOCAL_CURRENT_GATE_NOT_RUN`; historical receipt `PO-07` is `INVALIDATED_NOT_CURRENT` | 最终冻结后用现有脚本 fresh 启动、执行、清理 |
| locked Chromium UI1 runner | quality owner | `AVAILABLE_LOCAL_CURRENT_GATE_NOT_RUN`; historical receipt `PO-08-ui1` is `INVALIDATED_NOT_CURRENT` | 最终冻结后运行 `verify-ui1.sh`；exact historical observation was Chrome for Testing 151.0.7922.34 / TOTAL 3 SUCCESS |
| future mini-program pickup picker | future consumer owner | `NOT_IMPLEMENTED_OUT_OF_SCOPE` | 独立 client change 接线并建立自身 RGR/UI evidence |
| UI2/UI3、生产、真实菜单 UAT | 客户/平台/UAT owner | `BLOCKED_EXTERNAL_NOT_REQUIRED_FOR_CANDIDATE` | 另行授权并提供账号、平台、版本、环境与受控 UAT |

未覆盖边界固定为：小程序 picker 未接线；UI2/UI3 未运行；无生产/部署/真实菜单 UAT；`orderable` 不是最终下单许可，选定时刻后的现有 `/menu` 与 checkout 仍负责最终事实。
