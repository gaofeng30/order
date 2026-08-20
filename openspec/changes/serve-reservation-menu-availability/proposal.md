## Why

现有匿名 `/api/v1/catalog` 只能读取长期上下架后的普通价目录，不能按预约日期与离散取餐时间表达餐段、截单和按日售罄事实。0818 基线已经冻结今天/明天、午餐/晚餐、固定餐段截单和日期级售罄，因此需要一个不改变旧 catalog 的预约菜单读取契约。

## What Changes

- 新增唯一匿名公共读取路径 `GET /api/v1/menu?date=YYYY-MM-DD&time=HH:MM`；PRD 的 `/menu` 只表示业务路径，不注册第二条无版本兼容路由。
- 对合法的今天/明天与当前 `meal_periods` 配置生成的午餐/晚餐离散时间点返回普通价菜单，响应固化所选日期、时间、餐段、`Asia/Shanghai`、餐段截单事实和整餐段 `orderable`。
- 读取与下单资格分离：合法但已截单的餐段仍返回菜单，整餐段 `orderable=false`；真正的 checkout/prepay 强校验不属于本 change。
- 只返回启用分类中的上架商品，并按所选时间所属餐段过滤；日期 D 的售罄商品仍返回，但带 `sold_out=true`、`orderable=false`，且同一记录不影响 D+1。
- 增加最小持久化结构：商品的 `all/lunch/dinner` 餐段归属；只含午餐/晚餐的 `meal_periods` 当前配置；以及以 `service_date × product_id` 唯一表示的日期级售罄记录。初始餐段数据使用客户确认值，但读取不得依赖代码默认值；不增加数量、预占、扣减或管理写 API。
- 保留 `/api/v1/catalog` 与 `/api/v1/catalog/products/:id` 的既有路径、JSON、过滤、错误和匿名 GET 行为。

## Capabilities

### New Capabilities

- `reservation-menu-availability`: 定义预约菜单查询参数、餐段与截单事实、普通价商品过滤、日期级售罄、错误语义和旧 catalog 回归边界。

### Modified Capabilities

无。

## Impact

- Primary outcome：公共后端可按合法预约日期与离散时间点返回可浏览的普通价菜单及服务端可购买事实，同时不改变既有 catalog。
- 唯一接受裁决：仅当新路径、查询/响应、日期餐段与截单、下架/分类停用过滤、D/D+1 售罄隔离、稳定错误、真实隔离 MySQL、迁移连续性/幂等、handler/router 契约和 catalog 回归全部通过，且 exact candidate 获独立只读验证时为 `ACCEPT`；任一条件不满足即 `REJECT`。
- Gate：`gate_type=W3`（新增持久化 schema 与日期级可购买数据结果）；`ui_level_target=UI0`；`ui_level_actual=UI0`。本 change 无 UI，UI0 只表示 API/静态与数据库证据，不证明任何前端或真实微信行为。
- Owner：branch `codex/serve-reservation-menu-availability`，worktree `/Users/vivix/.codex/worktrees/order-serve-reservation-menu-availability.Writer`；本 change 的 migration/router/main 共享路径由该 writer 独占，禁止并行 Order writer。
- Base SHA：`babd1ef662811e3df6a75aa28995268352531438`。
- Owned paths：
  - `openspec/changes/serve-reservation-menu-availability/**`
  - `services/api/internal/menu/**`
  - `services/api/migrations/000004_add_products_meal_period.sql`
  - `services/api/migrations/000005_create_meal_periods.sql`
  - `services/api/migrations/000006_initialize_meal_periods.sql`
  - `services/api/migrations/000007_create_product_sold_out_dates.sql`
  - `services/api/migrations/embed_test.go`
  - `services/api/internal/catalog/migrations_test.go`
  - `services/api/internal/catalog/mysql_integration_test.go`
  - `services/api/internal/migrate/mysql_integration_test.go`
  - `services/api/internal/httpapi/router.go`
  - `services/api/internal/httpapi/router_test.go`
  - `services/api/cmd/order-api/main.go`
  - `services/api/scripts/menu-integration.sh`
- Read-only shared contracts：`docs/product/online-ordering-system-prd-0818.md`、`docs/product/online-ordering-system-prd-0818-review.md`、`openspec/specs/mvp-product-baseline/spec.md`、`openspec/changes/adopt-0818-prd-baseline/**`、`services/api/internal/catalog/{model.go,repository.go,handler.go,repository_test.go,handler_test.go}`、既有 `/api/v1/catalog*` 和不可变迁移 `000001`–`000003`。
- Dependency：`adopt-0818-prd-baseline@babd1ef662811e3df6a75aa28995268352531438` 已进入当前 main，状态按调度事实为 `INTEGRATED`，但 change 目录尚未 archive；本 change 不修改或 archive 该依赖。
- Required external assets：none。真实隔离 MySQL 8 是 writer-managed W3 验证 runtime；批准后的 writer 已按 canonical `mysql-persistence-foundation` 在 Red 前建立 `order-mysql-w3`，foundation 与本 change 真实 W3 均 PASS，容器/profile 保留供 exact-SHA verifier fresh rebuild；这不属于 `BLOCKED_EXTERNAL`。
- Non-goals：P1 营业状态人工切换、餐段/商品售罄的管理端写 API 与鉴权；P2–P4；P5 员工身份/折扣；特殊停餐日期；购物车、checkout、prepay、payment、order；数量库存、预占、自动扣减；前端；第二条 `/menu`；其他未来兼容；商品种子数据；push、PR、deploy、integration 或 archive。未来管理员更新合法 `meal_periods` 数据不得要求改变本公共路径、字段或错误契约。
