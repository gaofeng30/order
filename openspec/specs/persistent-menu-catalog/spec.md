# persistent-menu-catalog Specification

## Purpose

MySQL foundation 已集成并归档，但仓库仍没有持久化菜单业务表或匿名目录 API；匿名用户无法从真实持久化数据读取菜单。现在增加一个最小服务端纵向切片，让后续小程序接入只消费稳定目录契约，而不把库存、售罄、员工价或图片混入目录事实。
## Requirements
### Requirement: Catalog schema contains only persistent directory facts

系统 MUST 只通过两个新的 forward-only 文件追加目录 schema：`000002_create_categories.sql` 与 `000003_create_products.sql`。每个文件 MUST 符合 foundation 的连续编号、单文件单 statement、embedded checksum 与不可漂移规则；不得包含 seed、down、repair、force 或第二个 statement。

`categories` MUST 且只能有：`id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT` 主键、`name VARCHAR(100) NOT NULL`、`sort_order INT UNSIGNED NOT NULL DEFAULT 0`、`is_active BOOLEAN NOT NULL DEFAULT TRUE`。`products` MUST 且只能有：`id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT` 主键、`category_id BIGINT UNSIGNED NOT NULL`、`name VARCHAR(100) NOT NULL`、`description VARCHAR(1000) NOT NULL DEFAULT ''`、`specification VARCHAR(255) NOT NULL DEFAULT ''`、`price_cents INT UNSIGNED NOT NULL`、`sort_order INT UNSIGNED NOT NULL DEFAULT 0`、`is_listed BOOLEAN NOT NULL DEFAULT TRUE`。两表 MUST 使用 InnoDB、`utf8mb4` 与 `utf8mb4_0900_ai_ci`；`products.category_id` MUST 以 `ON UPDATE RESTRICT ON DELETE RESTRICT` 引用 `categories.id`。

除支持 `is_active, sort_order, id` 与 `category_id, is_listed, sort_order, id` 读取顺序的索引、主键和外键外，schema MUST NOT 包含库存、售罄、员工价、销量、软删除、多门店、tenant、图片、时间戳或其他字段。`price_cents` 的持久化与 API 值 MUST 为非负整数分，不存在浮点价格或元单位字段。

#### Scenario: Fresh foundation schema is migrated to catalog current

- **WHEN** `order-migrate` 在只含 clean version 1 的真实 MySQL 8.0 随机隔离 schema 上首次运行
- **THEN** 只按顺序应用 version 2 与 version 3，并写入对应 name/checksum 的 clean history
- **AND** 最终两张表、字段、类型、默认值、引擎、字符集、索引和 `RESTRICT` 外键与本 requirement 完全一致

#### Scenario: Catalog migrations are repeated

- **WHEN** 同一 embedded migration 集合在 current schema 上再次运行
- **THEN** runner 成功且 `applied_count=0`
- **AND** 表、数据、history、checksum 与 `applied_at` 均不被重写

#### Scenario: Applied catalog SQL drifts

- **WHEN** version 2 或 version 3 的文件 bytes、name 或数据库 history checksum 与已应用记录不同
- **THEN** foundation runner/readiness 按既有契约报告 `schema_checksum_mismatch`
- **AND** 不覆盖 checksum、不重跑 SQL、不修改目录数据

#### Scenario: Category with products is deleted or re-keyed

- **WHEN** fixture 尝试删除或修改仍被商品引用的 category id
- **THEN** MySQL 以 `RESTRICT` 拒绝该写入
- **AND** category 与 products 保持不变

#### Scenario: Schema scope is inspected

- **WHEN** verifier 检查两份 migration 和 `information_schema`
- **THEN** 只存在本 requirement 冻结的字段、索引与约束，且每文件只有一个 `CREATE TABLE` statement
- **AND** 不存在 stock、availability、employee price、sales、soft delete、store/tenant、image、seed 或 down 表面

### Requirement: Catalog reads are anonymous and carry no availability semantics

系统 MUST 暴露且只暴露匿名 `GET /api/v1/catalog` 与 `GET /api/v1/catalog/products/:id`；调用前不得要求登录、微信会话、手机号、员工身份或 RBAC。两个 handler MUST 不读取或传递营业日期、餐段、取餐时段、库存或身份输入。

目录响应 MUST NOT 包含 `SOLD_OUT`、`AVAILABLE`、`orderable` 或其他可售状态，也不得包含库存、员工价、销量、图片或后台字段。目录只表示持久化的分类/商品展示事实；后续 `serve-menu-availability` 才能依据 `营业日期 × 餐段 × 商品` 计算可售结果。

#### Scenario: Anonymous client opens the menu

- **WHEN** 未登录且没有手机号的调用方发送 `GET /api/v1/catalog`
- **THEN** 服务直接读取公开目录并返回目录成功或稳定的数据库错误
- **AND** 不触发身份、库存、日期、餐段或可购买性检查

#### Scenario: Client opens a product detail

- **WHEN** 匿名调用方从菜单进入商品详情并发送有效 product id 的 GET 请求
- **THEN** 服务只按目录可见性读取商品详情
- **AND** 响应不承诺商品在任一日期或餐段可购买

#### Scenario: Catalog surface is inspected for excluded semantics

- **WHEN** verifier 检查 request binding、repository、response structs 与 exact JSON
- **THEN** 请求面没有日期/餐段/身份输入，响应面没有库存、availability、orderable、员工价、销量、图片或后台字段
- **AND** 不存在 `stock_type`、`stock_quantity` 或目录层售罄判断

### Requirement: Catalog list is visible, stable, complete, and one-snapshot

`GET /api/v1/catalog` MUST 返回所有 `is_active=TRUE` 的分类，包括没有可见商品的启用分类；停用分类 MUST 完全隐藏。每个分类只包含 `is_listed=TRUE` 的商品；下架商品 MUST 隐藏。分类 MUST 按 `categories.sort_order ASC, categories.id ASC` 排序，分类内商品 MUST 按 `products.sort_order ASC, products.id ASC` 排序。

repository MUST 使用 request context、一个带 `LEFT JOIN` 的 MySQL 非锁定一致性 `SELECT`、显式列名和上述完整 `ORDER BY` 读取列表。单个 SQL statement 的 snapshot MUST 同时决定分类与商品可见性，避免两次读取间撕裂；不得使用 `SELECT *`、逐分类查询或其他 N+1 路径。

#### Scenario: Active categories contain listed products

- **WHEN** 数据同时含启用/停用分类与上架/下架商品
- **THEN** 列表只返回启用分类，并只在其中返回该分类的上架商品
- **AND** 停用分类及其所有商品、下架商品都不出现在 JSON 中

#### Scenario: Sort orders collide

- **WHEN** 多个分类或同一分类内多个商品拥有相同 `sort_order`
- **THEN** 分类和商品分别以无符号 id 升序打破并列
- **AND** 对同一 snapshot 的重复读取产生相同顺序

#### Scenario: Catalog is empty

- **WHEN** schema 中没有启用分类
- **THEN** 接口返回 HTTP 200 与 `{"categories":[]}`
- **AND** `categories` 不是 `null`、缺失字段或错误

#### Scenario: Active category has no listed product

- **WHEN** 一个启用分类没有商品或只有下架商品
- **THEN** 列表仍返回该分类并将其 `products` 固定为 `[]`
- **AND** `products` 不是 `null` 或缺失字段

#### Scenario: Catalog changes during a list read

- **WHEN** 另一个连接在列表 statement 执行期间更新分类或商品可见性
- **THEN** 当前响应只反映该单个一致性 statement 的一个 snapshot，不出现来自两个时点的分类/商品组合
- **AND** repository 对任意分类数量都只执行一个 list query

### Requirement: Product detail applies the same visibility and strict identifier rules

详情 `:id` MUST 是 HTTP path 解码后的非空 ASCII 十进制数字串，解析值严格大于 0 且不超过 `BIGINT UNSIGNED` 最大值；前导零允许但 `+`、`-`、空白、浮点、十六进制、非 ASCII 数字、0 与溢出 MUST 统一视为不存在。repository MUST 使用 request context、显式列名和单个 join query，且只有商品 `is_listed=TRUE` 且所属分类 `is_active=TRUE` 时才返回详情。

非法 id、未知 id、下架商品及停用分类下商品 MUST 产生完全相同的 404 契约，不得向调用方区分存在性或隐藏原因。

#### Scenario: Visible product is requested

- **WHEN** id 指向启用分类下的上架商品
- **THEN** 详情返回该商品的公开目录字段
- **AND** 查询不读取库存、员工价、销量、图片或后台字段

#### Scenario: Product is unknown or hidden

- **WHEN** id 不存在、商品已下架或所属分类已停用
- **THEN** 三种情况均返回相同的 HTTP 404 与 `PRODUCT_NOT_FOUND` body
- **AND** body 与日志不暴露命中行、隐藏原因或 SQL 结果

#### Scenario: Product id is invalid

- **WHEN** id 为 0、带符号、含空白/小数/非十进制字符、非 ASCII 数字或超出 uint64
- **THEN** handler 不查询数据库并返回与未知商品相同的 404 契约
- **AND** 前导零的有效正整数按其数值查询并以规范十进制字符串返回

### Requirement: Success JSON is exact, typed, and minimal

列表成功 body MUST 精确符合 `{"categories":[{"id":"<decimal>","name":"<string>","products":[{"id":"<decimal>","category_id":"<decimal>","name":"<string>","description":"<string>","specification":"<string>","price_cents":<non-negative-integer>}]}]}`。详情成功 body MUST 精确符合 `{"product":{"id":"<decimal>","category_id":"<decimal>","name":"<string>","description":"<string>","specification":"<string>","price_cents":<non-negative-integer>}}`。

所有内部 `BIGINT UNSIGNED` id MUST 在 JSON 中编码为不带符号或前导零的十进制字符串；`price_cents` MUST 编码为 JSON number 整数。所有列出的字段 MUST 始终存在：`name`、`description` 与 `specification` 即使为空也为 `""`；`categories` 与 `products` 即使为空也为 `[]`。响应 MUST NOT 额外暴露 `sort_order`、`is_active`、`is_listed` 或任何非冻结字段，并 MUST 使用 `application/json`。

#### Scenario: Empty text fields are returned

- **WHEN** 可见商品的 description 与 specification 为数据库空字符串
- **THEN** 列表与详情都显式返回 `"description":""` 和 `"specification":""`
- **AND** 不返回 `null`、省略字段或改变字段类型

#### Scenario: Unsigned ids and integer price are encoded

- **WHEN** fixture 使用合法的 `BIGINT UNSIGNED` id 与非负 `price_cents`
- **THEN** id/category_id 均为规范十进制 JSON string，price_cents 为非负 JSON integer number
- **AND** 不出现浮点、科学计数价格、元单位或 JSON numeric id

#### Scenario: Internal fields are inspected

- **WHEN** httptest 对列表与详情执行 exact JSON 比较
- **THEN** body 只含冻结字段、层级、类型和数组语义
- **AND** sort_order、is_active、is_listed、employee price、stock、sales 和 image 字段均不存在

### Requirement: Catalog errors are stable and non-sensitive

catalog 404 body MUST 精确为 `{"error":{"code":"PRODUCT_NOT_FOUND","message":"product not found"}}`。数据库不可达、query/scan/rows/invariant 错误 MUST 统一返回 HTTP 503，body MUST 精确为 `{"error":{"code":"CATALOG_UNAVAILABLE","message":"catalog temporarily unavailable"}}`。错误响应与日志 MUST NOT 包含 SQL、DSN、连接字段、数据库错误正文、表/列名、凭据或内部堆栈。

现有 router 行为 MUST 保持：任一 catalog path 的非 GET 方法返回 405 空 body；未知 path 返回 404 空 body；既有 health route、request ID、访问日志、recovery 和生命周期行为不改变。catalog 的业务 404/503 MUST 使用 `application/json`。

#### Scenario: Database is disconnected during list or detail

- **WHEN** repository 的真实 MySQL 连接在 catalog 请求前被断开或 query 失败
- **THEN** 列表与详情都返回相同的 503 `CATALOG_UNAVAILABLE` exact JSON
- **AND** 响应和捕获日志不包含底层错误、SQL、DSN 或连接信息

#### Scenario: Known catalog path uses a non-GET method

- **WHEN** 调用方对列表或详情 path 使用 POST、PUT、PATCH、DELETE、HEAD 或 OPTIONS
- **THEN** router 沿用既有 NoMethod 契约返回 405 与空 body
- **AND** 不执行 repository query，也不返回 catalog success/error JSON

#### Scenario: Unknown path is requested

- **WHEN** 调用方请求未注册 path
- **THEN** router 沿用既有 NoRoute 契约返回 404 与空 body
- **AND** 不把 unknown route 改写成 `PRODUCT_NOT_FOUND`

#### Scenario: Existing health and middleware contracts regress

- **WHEN** verifier 重跑 foundation/bootstrap 的 health、404/405、request ID、sanitized access log、panic recovery 与 shutdown tests
- **THEN** 全部既有契约继续通过
- **AND** catalog 装配不创建第二 router、第二 pool 或新的 middleware 顺序

### Requirement: Catalog candidate is dependency-gated and proven on real MySQL 8.0

本 change 可在 foundation 仅 `APPROVED` 时完成 DRAFT 与 approval，但 MUST NOT 进入 apply，直到 `establish-mysql-persistence-foundation` 在当前 `main` 为 `INTEGRATED`。解除依赖后 writer MUST 吸收最新 main、按实际 foundation API 同步四类 artifacts，并使旧 base 上的任何 strict、test 或验证结论失效。

W3 candidate MUST 使用 foundation 冻结的隔离真实 MySQL 8.0 资产和一次性随机 `order_test_` schema，fixture 只由 test runtime 写入且在所有退出路径精确清理。真实矩阵 MUST 覆盖 v1→v3 首次/重复/checksum、schema 精确性与禁用字段、可见/隐藏、稳定排序、整数分、空目录、空分类、一致性单查询、DB 断开 503 以及 foundation readiness+catalog 联合 smoke；mock/fake/httptest 只能补充内部 Red 和 HTTP 映射，不能替代 W3 PASS。

#### Scenario: Planning completes before foundation integration

- **WHEN** 当前 base 仍为 `5ba5340cf9098724c0eb2284fdc5b14cb97be5dc`，foundation 只在 exact SHA `c17ba4a5bfe7556b779fac093925df609358fe05` 记录 `APPROVED`
- **THEN** change 只能保持 DRAFT/APPROVED 规划状态，所有 apply tasks 未执行
- **AND** 不修改业务代码、migration、依赖或 runtime，不把 foundation 写成 implemented/integrated

#### Scenario: Apply dependency becomes satisfied

- **WHEN** foundation 已在当前 main `INTEGRATED` 且 writer 准备开始 catalog Red
- **THEN** writer 先吸收 latest main、重新核对实际 migration/router/pool/readiness 共享契约并重跑 strict
- **AND** 任一差异先同步 proposal/design/spec/tasks 和 approval，不能沿用旧 base 结论

#### Scenario: Real catalog matrix passes and cleans up

- **WHEN** writer 与 exact-SHA verifier 分别在 clean 隔离 schema 运行完整 foundation+catalog W3 matrix
- **THEN** 两者都取得全部冻结场景 PASS，且每次只删除本次创建并验证归属的随机 schema
- **AND** cleanup 失败、目标不安全、MySQL 非真实 8.0 或缺任一场景均使 Gate FAIL

#### Scenario: Candidate is independently verified

- **WHEN** writer 形成已提交 candidate 完整 SHA
- **THEN** verifier 在另一 clean detached worktree 重跑 httptest、真实 MySQL、test/race/vet/build/smoke、strict、owned/protected/sensitive 与结束 clean 检查
- **AND** 任何代码、SQL bytes/name/number、artifact、base、依赖、命令、rebase/merge 或 candidate SHA 变化都使旧验证失效
