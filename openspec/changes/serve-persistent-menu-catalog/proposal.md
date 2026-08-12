## Why

MySQL foundation 已集成并归档，但仓库仍没有持久化菜单业务表或匿名目录 API；匿名用户无法从真实持久化数据读取菜单。现在增加一个最小服务端纵向切片，让后续小程序接入只消费稳定目录契约，而不把库存、售罄、员工价或图片混入目录事实。

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

- 状态：`IMPLEMENTING`。`approval_date=2026-08-13`；`approver=主 Agent`。主 Agent 在用户授权的自主裁决范围内批准本规划，并在 foundation 归档后正式批准 moving-main recovery 与 apply；该记录不得表述为用户亲自确认。批准依据仍为单能力 `persistent-menu-catalog`、`W3/UI0`、目录与库存/售罄/员工价/图片边界、schema/HTTP/ID/排序/可见性/一致性读取/真实 MySQL 8.0 RGR 均唯一冻结，owned/非目标/依赖/45 tasks 完整且 strict PASS。
- moving-main recovery 已把旧 planning branch 从 `base_sha=5ba5340cf9098724c0eb2284fdc5b14cb97be5dc` 无冲突、无 merge commit 地线性重放到 `base_sha=cbfb803bb74f34b4b39fd0feff2c753613a06de2`；旧 DRAFT/APPROVED exact SHA 因重放失效，批准的产品、API 与 schema 语义不变。
- 唯一 outcome：匿名调用方可从真实 MySQL 读取稳定、最小、与库存事实解耦的菜单目录及可见商品详情。
- 单一 acceptance verdict：未来 candidate 只有在 schema migration、repository 一致性、HTTP 精确契约、真实 MySQL 8.0 故障矩阵、全仓相关回归和 exact-SHA 独立验证全部通过后才可接受；任一部分不可独立发布或回滚。
- `base_sha=cbfb803bb74f34b4b39fd0feff2c753613a06de2`；`candidate_sha=NOT_CREATED`。
- `gate_type=W3`：apply 将追加持久化 schema 并改变公共读取结果；`ui_level_target=UI0`、`ui_level_actual=NOT_RUN`，本 change 无 UI，且不宣称交付 PRD 的完整菜单页面。
- IMPLEMENTING 尚不评定 candidate C/T/V/R；目标为 `C=10、T=10、V=8、R=8`，只有真实证据齐全且硬阻断为零时才能记录。

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
- `README.md`（只允许同步目录 API、v1-v3 migration 当前事实、匿名 curl、真实 MySQL 验证命令与非目标）

README ownership 是 moving-main 后唯一新增路径：catalog 集成后原文“无业务表/无业务接口”“migration 只创建 schema_migrations”会直接失真，主 Agent 已批准由同一能力最小修正，不另拆文档 change。禁止继续扩大 ownership；尤其不拥有 `go.mod`、`go.sum`、`services/api/internal/app/**`、`services/api/internal/config/**`、`services/api/internal/database/**`、`services/api/internal/migrate/**`、`services/api/internal/httpapi/health.go`、`services/api/internal/httpapi/middleware.go`、`apps/**`、产品/架构/云文档、canonical/archived specs、skills 或 `AGENTS.md`。

### 只读共享契约与依赖硬门

- 只读遵循根 `AGENTS.md`、`docs/quality/change-quality-gates.md`、canonical `mvp-product-baseline`、`production-architecture-baseline`、`api-service-bootstrap` 以及现有 HTTP 404/405/middleware 契约。
- 上游 `establish-mysql-persistence-foundation` candidate/integrated exact 为 `14fc3c3b10eda28a1c61cc0ac552ca46d1cb14e1`，已随 archive/main exact `cbfb803bb74f34b4b39fd0feff2c753613a06de2` 进入 `ARCHIVED`；canonical MySQL/health 契约与真实 W3 Gate 已集成。
- moving-main recovery 后已重新读取实际 config/database/migrate/readiness/router/main/README 接口；它们与本 change 的 pool、migration、health 和装配假设一致，不需要改变 approved persistent-menu-catalog 行为。后续 current main、依赖、公共契约或 owned paths 再变化即停止并重新裁决。
- 后续 `connect-miniprogram-menu-catalog` 必须等待本 change `INTEGRATED`；它不是本 change 的 consumer/UI 验收替代品。

### 必要资产与最小成功标准

- 当前 `ui_level_actual=NOT_RUN`。foundation 留下的专属 `order-mysql-w3` 真实 MySQL 8.0 本地资产可供复用，但 catalog preflight 与 W3 仍为 `NOT_RUN`；writer 必须先只读核验 exact identity、双 label、loopback、tmpfs、zero mounts 和随机 schema 残留 0，再进入 Red。这不是客户或平台 `BLOCKED_EXTERNAL`。
- apply writer 只按 foundation 已冻结的隔离 MySQL 8.0、随机 `order_test_` schema、安全闩和清理边界执行；若资产未建立、目标不安全或 cleanup 失败，writer Gate 为 FAIL。
- 最小成功标准：两份单 statement migration 的首次/重复/checksum/无 seed/down 与随机 schema cleanup 通过；列表/详情在真实 MySQL 上满足可见性、空目录/空分类、稳定排序、一致性读取、无 N+1、整数分和 DB 断开 503；httptest 满足 exact JSON、GET-only、非法/未知/隐藏 404 且既有 404/405 不漂移；README 只同步当前 API/migration/curl/验证与非目标且保留 production SSM fail-fast；Go test/race/vet/build/smoke、foundation+catalog integration、strict、owned/protected/sensitive 检查和 exact-SHA verifier 全部通过。UI actual 保持 `NOT_RUN`。

### Non-Goals

- 商户分类/商品 CRUD、seed、后台 UI 或任何写 API。
- 图片/COS、图片字段或本阶段图片承诺。
- 员工价、身份、RBAC、登录、手机号或小程序接入。
- 库存、售罄、`AVAILABLE`/`SOLD_OUT`/`orderable`、营业日期、餐段、预约或可购买性判断。
- 购物车、报价/算价、订单、支付、退款、销量或看板。
- 软删除、多门店、tenant、口味选项、迁移 down/repair/force 或通用 repository/ORM 抽象。
