## Context

当前 Go API 通过 `catalog` 包和 MySQL v1–v3 提供匿名 `/api/v1/catalog*`；查询只认识分类启用与商品上架，产品表没有餐段，数据库没有按日期售罄事实。0818 基线与本 change 的补充裁决要求把“可浏览”和“可购买”分开：合法但已截单的菜单仍能浏览，日期级售罄商品仍能看到，而 checkout 强校验另属后续 change。

本 change 依赖已经位于 main `babd1ef662811e3df6a75aa28995268352531438` 的 `adopt-0818-prd-baseline`（调度状态 `INTEGRATED`、尚未 archive）。它独占 migration、router 与 `cmd/order-api/main.go` 共享路径；在 writer 完成前不得启动其他 Order writer。

## Goals / Non-Goals

**Goals:**

- 用唯一 `/api/v1/menu` 返回日期、离散时间、餐段、截单、普通价、售罄与最终 `orderable` 事实。
- 用最小 v4–v7 schema 表达商品餐段、午晚餐当前时间配置与按日售罄，并以真实 MySQL 8 证明迁移和读取结果。
- 保持旧 catalog 的请求与响应合同完全不变。

**Non-Goals:**

- 不做营业状态及 P1，不做任何管理端写 API/鉴权，不解决 P2–P5。
- 不做员工折扣、购物车、checkout/prepay/payment/order、数量库存、预占、扣减、前端或兼容路由。
- 不做餐段配置管理写 API/鉴权、特殊停餐日历或多门店模型；只读取当前 lunch/dinner 配置。

## Decisions

### Add a separate `menu` read package and leave `catalog` behavior intact

新增 `services/api/internal/menu`，内部包含配置校验/选择规则、窄 `Reader`、MySQL repository 和 HTTP handler。`httpapi.NewRouter` 显式接收 catalog 与 menu handler，`cmd/order-api/main.go` 让二者共享同一 `*sql.DB`；只注册 `/api/v1/menu`。没有把条件参数塞进 `/api/v1/catalog`，因为这会改变已发布路径语义；也没有注册无版本 `/menu`。

handler 使用可注入时钟，并在任何 DB 读取前拒绝缺失/重复参数、非 ASCII 格式和非今天/明天日期。格式合法的 time 是否为离散点只能在 repository 读取完整两行 `meal_periods` 后判断：先验证恰有一条 lunch/dinner，cutoff/start/end 均在同一营业日 `00:00..23:59` 且秒为 0，并验证 interval/顺序/整除/闭区间范围不重叠，再由 start/end/interval 生成离散点并计算所选餐段与 cutoff。配置无效统一 `MENU_UNAVAILABLE` 503，不回退 migration 初始值；time 格式合法但不属于当前点集则 `INVALID_MENU_SELECTION` 400 且不读取商品。

初始数据使用客户已确认 lunch `11:30/11:30–13:30/30`、dinner `17:00/17:00–19:00/30`，但它是可由后续已授权管理能力更新的业务数据，不是代码常量或不可变公共契约。只要新数据满足相同 shape，路径、字段和错误语义不变。

成功 JSON 固定为：

```json
{
  "selection": {"date":"2026-08-20","time":"12:00","timezone":"Asia/Shanghai"},
  "meal": {"code":"lunch","cutoff_at":"2026-08-20T11:30:00+08:00","orderable":false},
  "categories": [{
    "id":"1","name":"Meals","products":[{
      "id":"9","category_id":"1","name":"Rice","description":"","specification":"Large",
      "price_cents":1250,"sold_out":true,"orderable":false
    }]
  }]
}
```

示例使用 migration 初始 lunch 数据；合法 `meal_periods` 调整后，`meal.code`、`cutoff_at` 与 time 有效性必须动态来自当前数据，JSON shape 不变。

`INVALID_MENU_SELECTION` 统一覆盖缺失、重复、格式、日期与离散点错误：明显格式/date 错误在任何 repository 读取前返回，格式合法但非当前离散点的错误在配置查询后、商品查询前返回。repository 错误或配置非法统一映射 `MENU_UNAVAILABLE`。不回显 query、配置内容或底层错误。

### Persist meal scope current schedule and exact-date sold-out facts

v4 给 `products` 增加 `meal_period ENUM('all','lunch','dinner') NOT NULL DEFAULT 'all'`，保证现有行升级后仍被午晚餐读取。v5 创建 `meal_periods`，主键 code 只允许 lunch/dinner，保存 cutoff/start/end/interval；行内 CHECK 尽可能拒绝负数、`24:00:00` 及更大、非零秒、interval 越界和基本时间逆序，读取层仍重复验证，不能信任 schema 已挡住所有历史/旁路数据。v6 `initialize_meal_periods` 用单个 INSERT 写入客户确认的两行初始值；命名不含 runner 禁止的 `seed`。v7 创建 `product_sold_out_dates(service_date, product_id)`，主键为 `(service_date, product_id)`，外键对 products 使用 `ON UPDATE RESTRICT ON DELETE RESTRICT`。

跨行“恰有两餐段”和区间不重叠无法由单行 CHECK 完整表达，由 menu 配置校验在读取时强制；缺失、重复或非法配置使整个 menu 503。售罄表不存在数量、原因、操作者、自动恢复或软删除字段；D+1 恢复来自请求日期不匹配 D 的事实，而不是清理任务。

repository 使用固定两次有界查询：第一条一次读取全部 `meal_periods`（合法状态只能为两行）；配置校验和 time 归属通过后，第二条用一个显式 SELECT 读取 active category INNER JOIN listed product，LEFT JOIN 请求日期的 sold-out 记录，meal 条件只接受 `all` 或所选 lunch/dinner，并按既有 category/product sort 排序。不存在按 category/product 的 N+1。INNER JOIN 使没有符合商品的分类不返回；handler 初始化非 null 数组。商品 `orderable = meal.orderable && !sold_out`，同时保留 `sold_out` 让调用方区分截单与售罄。

没有把配置写 API 放入本 change：读取当前业务数据不需要同时决定管理员权限、写入审计或 P1 营业状态。没有把售罄落在 products：那会让日期 D 污染 D+1。

### Prove W3 with the existing owned MySQL runner

先新增会因能力缺失而失败的 config/selection/handler/repository/router/migration 契约测试，再做最小实现并重跑相同 focused tests。真实 `order-mysql-w3` 集成必须从 v3 升至 v7、重复运行零写、检查 schema/history/初始行/CHECK/主外键，验证非默认合法配置实际改变离散点与 cutoff、负数/跨日/非零秒和其他非法配置 fail closed、D/D+1 隔离，并清理随机 owned schema。现有 catalog 精确 JSON、全包 race/vet/build/smoke 作为回归。

本 change 没有业务写入口，因此并发写、事务中断后的业务恢复与 API 幂等写为 `N/A`；migration runner 的锁、dirty、checksum、crash recovery 和 repeat 契约仍由现有真实 MySQL suite 重跑，不以 mock 代替。

## Risks / Trade-offs

- [Risk] 配置表可被错误数据破坏，导致同一 time 落入两餐段或无法形成离散点。→ 每次菜单读取先完整验证两行配置；任一异常整体稳定 503，不返回部分结果或 fallback。
- [Risk] 新列使现有 catalog schema 精确断言失败。→ 只更新 migration/schema 测试期望，不改 catalog SQL/handler/JSON，并运行完整 catalog 回归。
- [Risk] 实现前没有 `order-mysql-w3`。→ writer 已按 canonical foundation 在 Red 前建立并核验 runtime，且 `services/api/scripts/menu-integration.sh` 已在 exact-digest MySQL 8.0.46 取得真实 PASS；profile 保留供 verifier fresh rebuild，不冒充外部资产。
- [Risk] 两次请求跨上海午夜会得到不同“今天”。→ 每次请求只读取一次注入时钟并从该快照派生日期、cutoff 和 orderable。
- [Risk] proposal/spec/design/tasks、base、迁移 bytes、配置 shape、响应字段、验收命令或 candidate SHA 任一变化。→ 旧 writer/verifier 证据全部失效，新 SHA 从 Red/适用 Gate 重新证明；测试或运行所依据的配置数据变化使对应运行证据失效，但合法管理员数据调整不要求修改 API 代码契约。

## Migration Plan

1. 明确批准后进入 IMPLEMENTING，先按 canonical foundation 建立/核验 writer-owned `order-mysql-w3`；未通过 preflight 不执行 Red。
2. 测试先行取得 Red，不修改 v1–v3；随后应用 v4–v7、menu package、router/main wiring，取得 focused Green。
3. Refactor 后重跑同一检查，并在真实隔离 MySQL 8 运行迁移/查询 Gate，再跑 catalog、全 API race/vet/build/smoke 与 owned-path audit。
4. 只提交 owned paths 形成 exact candidate；独立 verifier 在 clean detached worktree 从头重跑全部 Gate。
5. 只有依赖仍在当前 main、exact candidate 独立 PASS 且获得单独授权时才能集成。回退应用代码即可恢复旧路由；forward-only v4–v7 为兼容性新增且不得执行破坏性 down migration。push、deploy、archive 均不在授权内。

## Open Questions

无会改变本 change 行为或验收的未决问题。P1–P5 与后续 checkout 强校验均明确不在本范围。
