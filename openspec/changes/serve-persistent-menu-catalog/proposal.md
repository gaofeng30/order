## Why

当前仓库只有进程级 HTTP bootstrap，已批准但尚未集成的 MySQL foundation 也只规划连接、迁移与 readiness；匿名用户仍无法从真实持久化数据读取菜单。需要在该 foundation 集成后增加一个最小服务端纵向切片，让后续小程序接入只消费稳定目录契约，而不把库存、售罄、员工价或图片混入目录事实。

## What Changes

- 追加 `categories` 与 `products` 两个 MySQL 8.0 表的 forward-only migration，且只包含冻结的目录字段、稳定排序字段、展示开关和 `RESTRICT` 外键。
- 新增基于 `database/sql` 的 catalog repository：列表使用一次带 context 的一致性 join 查询返回启用分类及其已上架商品，详情只返回启用分类下的已上架商品；显式列名、无 `SELECT *`、无 N+1。
- 新增匿名只读 `GET /api/v1/catalog` 与 `GET /api/v1/catalog/products/:id`，冻结精确 JSON、十进制字符串 ID、整数分、404 与 503 错误契约。
- 目录不接收营业日期或餐段，不返回或判断库存、售罄、可购买性、员工价、图片或销量；这些属于后续独立 change。
- 保持现有未知路径 404、已知路径非 GET 405、health、middleware 与进程生命周期契约不漂移。

## Capabilities

### New Capabilities

- `persistent-menu-catalog`: 定义 MySQL 目录 schema、匿名列表/详情查询、稳定排序、可见性、精确 HTTP JSON/错误契约及真实 MySQL 8.0 W3 验收。

### Modified Capabilities

无。此 change 使用 `mysql-persistence-foundation` 预留的后续业务 migration 扩展点，并在现有 router 装配新业务路由；不改变 foundation 的连接、migration runner、readiness 或既有 health requirements。

## Impact

### 状态、Outcome 与 Acceptance Verdict

- 状态：`APPROVED`。`approval_date=2026-08-13`；`approver=主 Agent`。主 Agent 在用户授权的自主裁决范围内批准本规划，依据为单能力 `persistent-menu-catalog`、`W3/UI0`、目录与库存/售罄/员工价/图片边界、schema/HTTP/ID/排序/可见性/一致性读取/真实 MySQL 8.0 RGR 均唯一冻结，owned/非目标/依赖硬门/45 tasks 完整且 strict PASS；该记录不得表述为用户亲自确认。
- 本次只记录 approval，不执行 apply task、不进入 `IMPLEMENTING`，不修改 Go、SQL、依赖、README、runtime 或外部系统，所有 tasks 保持未勾选。
- 唯一 outcome：匿名调用方可从真实 MySQL 读取稳定、最小、与库存事实解耦的菜单目录及可见商品详情。
- 单一 acceptance verdict：未来 candidate 只有在 schema migration、repository 一致性、HTTP 精确契约、真实 MySQL 8.0 故障矩阵、全仓相关回归和 exact-SHA 独立验证全部通过后才可接受；任一部分不可独立发布或回滚。
- `base_sha=5ba5340cf9098724c0eb2284fdc5b14cb97be5dc`；`candidate_sha=NOT_CREATED`。
- `gate_type=W3`：apply 将追加持久化 schema 并改变公共读取结果；`ui_level_target=UI0`、`ui_level_actual=NOT_RUN`，本 change 无 UI，且不宣称交付 PRD 的完整菜单页面。
- APPROVED 仍不评定实现 C/T/V/R；candidate 目标为 `C=10、T=10、V=8、R=8`，只有进入 apply 后真实证据齐全且硬阻断为零时才能记录。

### 调用方与使用时机

- 调用方：后续 `connect-miniprogram-menu-catalog` 中的匿名微信小程序菜单客户端。
- 业务场景：进入或刷新菜单时读取分类与商品列表；从菜单进入商品详情时按商品 ID 读取详情。
- 调用时机：页面展示目录或详情前；不是结算、下单、库存校验或可购买性判断时机。小程序接入必须等待本 change `INTEGRATED`。

### Owner、Branch 与 Worktree

- owner：`Persistent Menu Catalog Planning Writer`；同一路径不得存在第二 writer。
- branch：`codex/serve-persistent-menu-catalog`。
- worktree：`/Users/vivix/.codex/worktrees/77b3/order`。

### Owned Paths

- `openspec/changes/serve-persistent-menu-catalog/**`
- `services/api/migrations/000002_create_categories.sql`
- `services/api/migrations/000003_create_products.sql`
- `services/api/internal/catalog/**`
- `services/api/internal/httpapi/router.go`（共享串行装配点）
- `services/api/internal/httpapi/router_test.go`（共享串行契约测试）
- `services/api/cmd/order-api/main.go`（共享串行装配点）
- `services/api/scripts/catalog-integration.sh`（仅用于冻结的真实 MySQL 8.0 catalog Gate）

禁止扩大 ownership。尤其不拥有 `go.mod`、`go.sum`、`README.md`、`services/api/internal/app/**`、`services/api/internal/config/**`、`services/api/internal/database/**`、`services/api/internal/migrate/**`、`services/api/internal/httpapi/health.go`、`services/api/internal/httpapi/middleware.go`、`apps/**`、产品/架构文档、canonical/archived specs、skills 或 `AGENTS.md`。

### 只读共享契约与依赖硬门

- 只读遵循根 `AGENTS.md`、`docs/quality/change-quality-gates.md`、canonical `mvp-product-baseline`、`production-architecture-baseline`、`api-service-bootstrap` 以及现有 HTTP 404/405/middleware 契约。
- 上游 `establish-mysql-persistence-foundation` 当前仅在 exact SHA `c17ba4a5bfe7556b779fac093925df609358fe05` 记录 `APPROVED`，不在本地 `main`，不得表述为 `INTEGRATED` 或实现 PASS。
- 本 change 可继续规划和被批准，但 apply 硬阻断于上游先进入当前 `main` 的 `INTEGRATED`。解除后 writer 必须吸收最新 `main`，重新读取实际 config/database/migrate/readiness/router 接口，同步四类 artifacts，并从头重跑 strict、writer Gate 与 exact-SHA independent verification；当前 base 上不得实施。
- 后续 `connect-miniprogram-menu-catalog` 必须等待本 change `INTEGRATED`；它不是本 change 的 consumer/UI 验收替代品。

### 必要资产与最小成功标准

- APPROVED 规划阶段无外部资产；当前 `ui_level_actual=NOT_RUN`。真实 MySQL 8.0 本地验收资产当前为 `NOT_ESTABLISHED`、catalog W3 为 `NOT_RUN`，本轮禁止安装或启动 runtime，也不得把 unit/mock 写成 W3 PASS。
- 未来 apply writer 负责在上游集成后按 foundation 已冻结的隔离 MySQL 8.0、随机 `order_test_` schema、安全闩和清理边界建立本地资产；这不是客户或平台 `BLOCKED_EXTERNAL`。若资产未建立、目标不安全或 cleanup 失败，writer Gate 为 FAIL。
- 最小成功标准：两份单 statement migration 的首次/重复/checksum/无 seed/down 与随机 schema cleanup 通过；列表/详情在真实 MySQL 上满足可见性、空目录/空分类、稳定排序、一致性读取、无 N+1、整数分和 DB 断开 503；httptest 满足 exact JSON、GET-only、非法/未知/隐藏 404 且既有 404/405 不漂移；Go test/race/vet/build/smoke、foundation+catalog integration、strict、owned/protected/sensitive 检查和 exact-SHA verifier 全部通过。UI actual 保持 `NOT_RUN`。

### Non-Goals

- 商户分类/商品 CRUD、seed、后台 UI 或任何写 API。
- 图片/COS、图片字段或本阶段图片承诺。
- 员工价、身份、RBAC、登录、手机号或小程序接入。
- 库存、售罄、`AVAILABLE`/`SOLD_OUT`/`orderable`、营业日期、餐段、预约或可购买性判断。
- 购物车、报价/算价、订单、支付、退款、销量或看板。
- 软删除、多门店、tenant、口味选项、迁移 down/repair/force 或通用 repository/ORM 抽象。
