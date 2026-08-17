## Why

当前 README、小程序说明、Demo 指南和技术文档仍把“两个前端全量使用 mock、菜单客户端尚未接入 API、仓库尚未实现 schema/migration”写成现状，但代码已经具备 MySQL 显式迁移、匿名菜单 API，且小程序首页、菜单和详情已调用该 API。新人会据此误判系统边界、启动方式和剩余工作；同时新完成的轻量 Harness 尚无面向开发者的最短入口说明。

## What Changes

- 把项目根 README、微信小程序 README、Demo 指南和技术文档统一为同一份 as-built 事实：菜单读取已接 API，其余订单、支付、商户管理和 Web Admin 仍是 mock/内存态。
- 在根 README 增加最短 Harness 使用入口，说明它是本地调度索引而非生命周期或验证证明。
- 在文档索引中提供当前实现与本地运行入口，避免新人从历史或客户材料推断代码状态。
- 删除或改写被代码取代的“前端尚未接入”“全量 mock”“本阶段不实现 schema/migration”等现状表述，不改业务规则、公共 API、数据或运行行为。

## Capabilities

### New Capabilities

- `repository-knowledge-handoff`: 当前实现、运行入口、Harness 使用方式和未实现边界在面向开发者的文档中保持一致、可验证。

### Modified Capabilities

无。

## Impact

- Owner：branch `codex/sync-repository-knowledge`，worktree `/Users/vivix/.codex/worktrees/order-knowledge-sync.Writer`。
- Owned paths：`README.md`、`apps/wechat-miniprogram/README.md`、`docs/README.md`、`docs/product/online-ordering-system-demo-guide.md`、`docs/product/online-ordering-system-technical.md`、`openspec/changes/sync-repository-knowledge/**`。
- Read-only evidence：`tools/harness`、`apps/wechat-miniprogram/app.js`、`apps/wechat-miniprogram/utils/catalogApi.js`、`apps/wechat-miniprogram/utils/catalogStore.js`、`services/api/**` 和既有 OpenSpec/质量门禁。
- Dependency：`add-lightweight-harness-loop` exact candidate `79c4f1d8ebf300bf2f9f4c226fcd2c2aa2643963`；本 change 可以形成候选，但该依赖进入当前 `main` 前不得集成。
- Non-goals：不修改代码、API、schema、产品规则、客户/合同材料、历史归档、AGENTS/Skills、部署配置或外部系统；不宣称真实 MySQL、微信开发者工具、真机、支付、UAT 或生产已经验证。
- Gate：`gate_type=W0`；`ui_level_target=UI0`；`ui_level_actual=UI0`；external assets none。
- 最小成功标准：五份面向开发者的文档对当前代码边界一致；旧现状语句消失；本地 Markdown 链接有效；OpenSpec strict、文档内容检查、`git diff --check` 和 owned-path audit 通过。
