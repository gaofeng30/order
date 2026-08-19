## ADDED Requirements

### Requirement: Reservation menu uses one versioned anonymous read contract

服务 MUST 只在 `GET /api/v1/menu?date=YYYY-MM-DD&time=HH:MM` 提供匿名预约菜单读取。`date` 与 `time` MUST 各出现一次；PRD 中的 `/menu` MUST 只视为业务路径，不得注册 `/menu`、其他版本或无参数兼容路由。错误响应 MUST 使用稳定、无敏感信息的 JSON；错误、访问日志和追踪 MUST 不记录原始 query。

成功响应 MUST 使用以下字段：`selection.date`、`selection.time`、`selection.timezone`，`meal.code`、`meal.cutoff_at`、`meal.orderable`，以及非 null 的 `categories` 数组。`selection.timezone` MUST 固定为 `Asia/Shanghai`；`meal.code` MUST 只为 `lunch` 或 `dinner`；`meal.cutoff_at` MUST 为带 `+08:00` 偏移的 RFC3339 截单时刻。

#### Scenario: Anonymous caller reads the versioned menu

- **WHEN** 匿名调用方用合法且各出现一次的 `date` 与 `time` 请求 `GET /api/v1/menu`
- **THEN** 服务返回 `200 application/json` 与完整 selection、meal 和 categories 字段
- **AND** 请求不需要用户、员工或商户身份

#### Scenario: Business-only path or wrong method is requested

- **WHEN** 调用方请求 `GET /menu`、`GET /api/v1/menu/anything` 或对 `/api/v1/menu` 使用非 GET 方法
- **THEN** 未注册路径保持现有空 body `404`，错误方法保持现有空 body `405`
- **AND** 服务不得执行菜单 repository 读取或注册第二条兼容路由

#### Scenario: Menu repository or configuration is unavailable

- **WHEN** 菜单读取遇到数据库/repository 错误，或餐段配置缺失、重复、重叠、非法
- **THEN** 服务返回 `503` 与 `{"error":{"code":"MENU_UNAVAILABLE","message":"menu temporarily unavailable"}}`
- **AND** 响应和访问日志不得泄露 SQL、DSN、密码、原始 query、非法配置内容或底层错误正文，也不得回退代码默认值

### Requirement: Selection uses today tomorrow and current meal configuration

服务 MUST 以请求处理时的 `Asia/Shanghai` 日期定义“今天”和“明天”，且只接受这两个 `YYYY-MM-DD` 日期。服务 MUST 从 `meal_periods` 当前持久化数据读取且只读取一条 `lunch` 与一条 `dinner`：各自行包含 `cutoff_time`、`pickup_start_time`、`pickup_end_time`、`interval_minutes`。离散时间点 MUST 从 start 开始按 interval 递增并包含 end；end 与 start 的分钟差 MUST 可被 interval 整除。

合法配置 MUST 同时满足：餐段代码恰为且各有一个 lunch/dinner；cutoff/start/end 均为同一营业日内 `00:00..23:59` 的 `HH:MM` 分钟点且秒为 `0`；interval 为 `1..1440`；cutoff 不晚于 pickup start；start 不晚于 end；区间和 interval 对齐；午晚餐的闭区间取餐范围不重叠。缺失、重复、未知代码、负数或 `24:00:00` 及更大的 TIME、非零秒、零/越界 interval、时间逆序、不对齐或范围重叠 MUST 统一返回 `MENU_UNAVAILABLE` 503，且 MUST 不使用代码常量或 migration 初始值兜底。

格式、参数重复和日期范围 MUST 在数据库读取前判定；格式合法的 `HH:MM` 是否属于当前离散点 MUST 在读取并验证配置后判定。对今天的合法时间点，当前时刻严格早于对应餐段 cutoff 时 `meal.orderable` MUST 为 `true`，到达或晚于 cutoff 时 MUST 为 `false`；明天的合法时间点对应 cutoff 尚在未来，MUST 为 `true`。截单只改变购买事实，不得把合法菜单读取变成错误。未来合法配置数据变化 MUST 直接影响时间判定与响应 cutoff，而不得改变路径、字段或错误公共契约。

#### Scenario: Initial configuration resolves today and tomorrow in Shanghai time

- **WHEN** 固定时钟为 `2026-08-20T10:00:00+08:00`，调用方分别请求 `2026-08-20&time=12:00` 与 `2026-08-21&time=18:00`
- **THEN** 两次请求分别返回 `meal.code=lunch` 与 `meal.code=dinner`
- **AND** cutoff 分别为请求日期的 `11:30:00+08:00` 与 `17:00:00+08:00`，两餐均 `orderable=true`

#### Scenario: Non-default legal configuration is read from storage

- **WHEN** 真实数据库把 lunch 合法更新为 cutoff `10:45`、pickup `11:00`–`12:00`、interval `20`，调用方请求 `11:40`
- **THEN** 服务按当前数据接受该离散点并返回 `meal.code=lunch`、对应日期 `10:45:00+08:00` cutoff
- **AND** 代码不得仍使用初始 `11:30` cutoff、30 分钟粒度或旧时间范围

#### Scenario: A legal meal is already cut off

- **WHEN** 固定时钟为 `2026-08-20T11:30:00+08:00`，调用方请求今天午餐任一离散时间点
- **THEN** 服务仍返回 `200` 和午餐菜单
- **AND** `meal.orderable=false` 且 `meal.cutoff_at=2026-08-20T11:30:00+08:00`

#### Scenario: Selection format or date is obviously invalid

- **WHEN** `date` 或 `time` 缺失、重复、不是严格 ASCII `YYYY-MM-DD`/`HH:MM` 格式，或日期不是今天/明天
- **THEN** 服务返回 `400` 与 `{"error":{"code":"INVALID_MENU_SELECTION","message":"invalid menu selection"}}`
- **AND** 不读取 repository、不返回最近日期/时间、不自动改选餐段

#### Scenario: Well-formed time is not a configured point

- **WHEN** date 与 `HH:MM` 格式合法，但该时间不属于当前 lunch/dinner 离散点
- **THEN** 服务在读取并验证当前配置后返回相同 `INVALID_MENU_SELECTION` 400
- **AND** 不回退 migration 初始时间、不选择最近时间或读取商品

#### Scenario: Meal configuration is invalid

- **WHEN** 当前配置缺失餐段、返回重复餐段、含负数/跨日/非零秒 TIME、区间重叠、时间逆序、interval 非法或不能整除区间
- **THEN** 任一原本格式和日期合法的菜单请求返回稳定 `MENU_UNAVAILABLE` 503
- **AND** 不返回部分菜单、不使用默认配置，也不把服务端配置错误伪装成调用方 400

### Requirement: Menu filters by category listing and selected meal

菜单 MUST 只读取启用分类下的上架商品，并只保留 `meal_period=all` 或与当前配置解析出的 `meal.code` 一致的商品。没有任何符合商品的分类 MUST 不返回；分类 MUST 按 `categories.sort_order, categories.id` 排序，商品 MUST 按 `products.sort_order, products.id` 排序。下架商品与停用分类 MUST 不因售罄记录或旧 catalog 可见性而返回。配置与菜单数据读取 MUST 使用固定有界查询且不得按分类或商品 N+1 查询；可以使用一个固定 SQL，或先读取完整两行餐段配置再用一个 SQL 读取所选菜单。

每个商品 MUST 且只能返回字符串 `id`、字符串 `category_id`、`name`、`description`、`specification`、整数分 `price_cents`、布尔 `sold_out` 与布尔 `orderable`。此匿名菜单 MUST 只返回普通价，不得返回员工身份、折扣率、员工价、会员、优惠券、库存数量或预占字段。

#### Scenario: Lunch request filters and orders the browseable menu

- **WHEN** 启用分类中同时存在 `all`、`lunch`、`dinner` 三种上架商品，并请求合法午餐时间点
- **THEN** 响应只包含 `all` 与 `lunch` 商品，且分类与商品按服务端顺序稳定排列
- **AND** 只剩晚餐商品或没有上架商品的分类不返回

#### Scenario: Hidden category and unlisted product are absent

- **WHEN** 商品已下架，或其分类已停用
- **THEN** 商品不得出现在任何日期与餐段的 `/api/v1/menu` 响应
- **AND** 售罄记录不得绕过长期可见性过滤

#### Scenario: No products match the selection

- **WHEN** 合法日期和时间没有任何符合可见性与餐段条件的商品
- **THEN** 服务返回 `200`、完整 selection/meal 与 `"categories":[]`
- **AND** categories 不得为 null，服务不得把空菜单映射为 `404`

### Requirement: Date sold out is browseable but not orderable

日期级售罄 MUST 只由 `(service_date, product_id)` 唯一记录表达。请求日期存在售罄记录时，符合长期可见性与餐段的商品 MUST 继续返回并带 `sold_out=true`、`orderable=false`；不存在该日期记录时 `sold_out=false`，且商品 `orderable` MUST 等于 `meal.orderable`。商品最终 `orderable` MUST 严格等于 `meal.orderable AND NOT sold_out`。

日期 D 的记录 MUST 不影响 D+1；“次日自然恢复” MUST 通过按请求日期精确读取实现，不得依赖定时清零、数量库存、库存预占、订单计数或自动扣减。读接口 MUST 不创建、删除或更新售罄记录。

#### Scenario: Product is sold out for the selected date

- **WHEN** 上架午餐商品存在日期 D 的售罄记录，调用方请求 D 的午餐
- **THEN** 商品仍返回，`sold_out=true` 且 `orderable=false`
- **AND** 响应不包含库存数量、自动售罄原因或售罄写操作

#### Scenario: Sold out does not leak into tomorrow

- **WHEN** 商品只有日期 D 的售罄记录，调用方在同一测试中读取 D 与 D+1 的同一餐段
- **THEN** D 返回 `sold_out=true`，D+1 返回 `sold_out=false`
- **AND** 若 D+1 餐段未截单，D+1 商品 `orderable=true`

#### Scenario: Meal is cut off but product is not sold out

- **WHEN** 日期级售罄记录不存在但今天目标餐段已经截单
- **THEN** 商品返回 `sold_out=false`、`orderable=false`
- **AND** 调用方可区分餐段截单与商品售罄两种不可购买原因

### Requirement: Availability and meal configuration schema is forward only and recoverable

迁移 MUST 在不可变 `000001`–`000003` 后连续新增：

- `000004_add_products_meal_period.sql`：为 `products` 增加非空 `ENUM('all','lunch','dinner')` 且默认 `all` 的 `meal_period`；
- `000005_create_meal_periods.sql`：创建 `meal_periods`，仅含主键 `code ENUM('lunch','dinner')`、`cutoff_time TIME NOT NULL`、`pickup_start_time TIME NOT NULL`、`pickup_end_time TIME NOT NULL`、`interval_minutes SMALLINT UNSIGNED NOT NULL`；CHECK MUST 在 MySQL 可表达范围内约束三类 TIME 均为 `00:00:00..23:59:00` 且 `SECOND(...)=0`、interval 为 `1..1440`、cutoff 不晚于 start、start 不晚于 end；应用读取仍 MUST 重复验证完整不变量并 fail closed；
- `000006_initialize_meal_periods.sql`：用一个 INSERT 初始化 lunch `11:30/11:30–13:30/30` 与 dinner `17:00/17:00–19:00/30` 两行，文件名不得包含 `seed`；
- `000007_create_product_sold_out_dates.sql`：创建仅含 `service_date DATE NOT NULL`、`product_id BIGINT UNSIGNED NOT NULL` 的日期级售罄表，以 `(service_date, product_id)` 为主键，并以 `RESTRICT` 外键指向 `products(id)`。

现有商品升级后 MUST 归属 `all`。餐段初始数据是当前客户配置，不是代码 fallback；服务启动和读取 MUST 以数据库当前值为准。schema MUST 不增加库存、数量、预占、订单、营业状态、特殊停餐或管理鉴权字段。

迁移 MUST 继续满足当前 forward-only 单语句、校验和、dirty、连续版本、命名锁和幂等 runner 契约。真实隔离 MySQL 8 测试 MUST 证明 v1–v3 可连续升级至 v7、初始两行精确、合法非默认配置被 API 读取、schema CHECK 拒绝负数/跨日/非零秒 TIME 与基本非法值、读取层对可观察非法配置稳定 503、重复运行零写入、schema/history 不漂移、日期键唯一、外键生效，并在测试拥有的随机 schema 内清理。并发管理写、售罄写 API 和业务事务恢复因本 change 没有写路径而 MUST 记录为 `N/A`，不得伪造 PASS。

#### Scenario: Existing catalog database upgrades to availability schema

- **WHEN** 真实隔离 MySQL 8 schema 已应用不可变 v1–v3，再运行完整迁移集
- **THEN** runner 连续应用 v4–v7，现有商品 `meal_period=all`，两个新表、初始餐段行与主键/外键形状精确匹配
- **AND** 原有 catalog 数据和 JSON 仍可读取

#### Scenario: Migration is repeated

- **WHEN** 在同一真实 schema 对完整迁移集再次运行
- **THEN** `AppliedCount=0`、版本仍为 `7`，schema_migrations 历史及业务 schema 不变化
- **AND** 不修改既有 migration checksum 或 dirty 状态

#### Scenario: Duplicate or orphan sold out fact is attempted

- **WHEN** 测试在同一日期为同一商品写入重复记录，或为不存在商品写入记录
- **THEN** MySQL 分别由主键与外键拒绝
- **AND** 已有合法售罄事实不被覆盖或扩充成数量库存

### Requirement: Existing catalog behavior is preserved exactly

本 change MUST 不修改 `/api/v1/catalog` 与 `/api/v1/catalog/products/:id` 的路径、匿名 GET 方法、JSON 字段、排序、启用分类/上架商品过滤、空数组、`404/405/503` 和稳定错误语义。新增 `meal_period` 与日期级售罄 MUST 不影响 catalog 的可见集合或把新 availability 字段泄漏到旧响应。

#### Scenario: Catalog regression runs after migration

- **WHEN** v7 schema 中存在不同餐段、非默认 meal_periods 配置与日期级售罄记录的商品并请求既有 catalog 列表和详情
- **THEN** 响应与 v3 合同保持完全相同，只按既有分类启用和商品上架规则返回普通价字段
- **AND** 响应不包含 `meal_period`、`sold_out`、`orderable` 或任何新 menu 字段

#### Scenario: Catalog unavailable behavior remains stable

- **WHEN** 数据库不可用时请求既有 catalog 列表或详情
- **THEN** 继续返回既有 `CATALOG_UNAVAILABLE` 稳定响应
- **AND** 新 menu 错误码不得替换旧 catalog 错误码
