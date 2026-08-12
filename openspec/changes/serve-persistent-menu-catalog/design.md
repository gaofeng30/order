## Context

Base `5ba5340cf9098724c0eb2284fdc5b14cb97be5dc` 只有 Go/Gin bootstrap：`main.go` 构造无数据库 router，`/health/live` 与 `/health/ready` 都是进程级 200，未知 path 404、已知 path 错误方法 405。canonical `mvp-product-baseline` 要求匿名用户可浏览分类和商品，并把库存唯一键冻结为 `营业日期 × 餐段 × 商品`；因此本 change 只交付目录读取，不得把目录误作库存或可售事实。

上游 `establish-mysql-persistence-foundation` 在 exact SHA `c17ba4a5bfe7556b779fac093925df609358fe05` 仅为 `APPROVED`：其设计冻结结构化 MySQL pool、embedded forward-only migrations、`order-migrate`、DB/schema readiness 和真实 MySQL 8.0 Gate，但尚未进入 local `main`、没有实现或集成。本 change 已完成规划批准，但 apply 必须等待上游 `INTEGRATED` 后吸收最新 main 并按真实接口重验。

治理状态为 `APPROVED`：`approval_date=2026-08-13`，`approver=主 Agent`。主 Agent 在用户授权的自主裁决范围内，依据单能力 `persistent-menu-catalog`、`W3/UI0`、唯一冻结的目录/schema/HTTP/真实 MySQL 8.0 RGR 边界、完整 ownership/依赖/45 tasks 与 strict PASS 作出规划裁决；该记录不表述为用户亲自确认。本次只记录 approval，不进入 `IMPLEMENTING` 或执行任何 apply task。

本 change 的调用方是后续匿名微信小程序菜单客户端：进入/刷新菜单时调用列表，进入商品详情时调用详情。它不是结算或下单接口，也不构成小程序 UI、图片、员工价、库存或售罄交付。

## Goals / Non-Goals

**Goals:**

- 用两个最小 MySQL 表持久化分类与商品目录，并遵循 foundation 的 migration/checksum/forward-only 契约。
- 用一次 snapshot query 读取启用分类及其上架商品，用一次 query 读取同可见性规则下的商品详情。
- 冻结匿名 GET、精确 JSON、字符串 ID、整数分、稳定排序、空集合、404/503 与非敏感错误边界。
- 用真实 MySQL 8.0 验证 schema、visibility、排序、一致性、无 N+1、故障和 cleanup；保持 bootstrap/foundation 回归。
- 保持 ownership 最小，仅串行修改 router、router tests 与 `main.go` 三个共享装配点。

**Non-Goals:**

- 不做商户 CRUD、seed、后台 UI、小程序接入或通用 catalog SDK。
- 不做图片/COS；schema 与 JSON 均没有图片字段，本阶段不承诺图片展示。
- 不做员工价、身份、登录、手机号、RBAC 或任何后台字段。
- 不做库存、售罄、availability/orderable、日期、餐段、预约、购物车、报价、订单、支付或退款。
- 不做多门店/tenant、软删除、销量、口味选项、ORM、通用 repository abstraction、down/force/repair。
- 不修改 dependency、README、foundation config/database/migrate/health/middleware、app、前端、产品/架构文档、canonical/archive 或治理文件。

## Decisions

### D1. 依赖先集成，再以最新 main 作为唯一实现起点

当前 change 只完成 APPROVED 规划 artifacts，尚未进入 `IMPLEMENTING`。上游 foundation exact SHA 只用于只读确认已批准的预期接口，不作为 implemented/integrated 依赖，也不 cherry-pick 到本分支。apply 的第一硬门是 foundation 已在当前 `main` 进入 `INTEGRATED`；解除后 writer 必须把本 branch 更新到最新 main，重新读取实际 `config/database/migrate/readiness/router/main` 和 migration embed 形态，并同步 proposal/spec/design/tasks 后重新 strict/approval。

理由：catalog 必须复用唯一 pool、runner、checksum 和 readiness；在旧 base 上猜实现会复制 foundation 或让共享 router/main 产生不可验证冲突。没有依赖的 changes 仍可并行，只有本 change 与 foundation 在共享装配点串行。`connect-miniprogram-menu-catalog` 反向等待本 change `INTEGRATED`。

### D2. 两张表只保存目录事实，字段与 SQL shape 一次冻结

`000002_create_categories.sql` 的唯一 statement 创建：

```text
categories(
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  sort_order INT UNSIGNED NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  INDEX(is_active, sort_order, id)
)
```

`000003_create_products.sql` 的唯一 statement 创建：

```text
products(
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  category_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(100) NOT NULL,
  description VARCHAR(1000) NOT NULL DEFAULT '',
  specification VARCHAR(255) NOT NULL DEFAULT '',
  price_cents INT UNSIGNED NOT NULL,
  sort_order INT UNSIGNED NOT NULL DEFAULT 0,
  is_listed BOOLEAN NOT NULL DEFAULT TRUE,
  INDEX(category_id, is_listed, sort_order, id),
  FOREIGN KEY(category_id) REFERENCES categories(id)
    ON UPDATE RESTRICT ON DELETE RESTRICT
)
```

两表固定 InnoDB、`utf8mb4`、`utf8mb4_0900_ai_ci`。`INT UNSIGNED price_cents` 同时保证非负、整数和 JSON 安全整数范围；不使用 `DECIMAL/FLOAT` 或元单位。`description/specification` 用非空字符串避免 JSON `null`/omitempty 分支。外键 `RESTRICT` 防止目录读取出现孤儿商品。索引直接服务两个冻结的可见性与排序条件，不增加全文、唯一名称或未来查询索引。

不在目录表加入 `stock_type/stock_quantity/sold_out/orderable`，因为库存事实必须由后续 change 按日期、餐段和商品建模；不加入 `employee_price`、图片、销量、timestamps、soft delete 或 tenant，因为这些都不是本读取切片的必需事实。fixture 只在 test runtime 直接写随机 schema，不存在生产 seed/down 文件。

### D3. 列表用一个 LEFT JOIN consistent read，详情用一个 INNER JOIN

`internal/catalog` 的 concrete repository 持有 foundation 的唯一 `*sql.DB`。列表固定执行一个 `QueryContext`：

```sql
SELECT
  c.id, c.name,
  p.id, p.category_id, p.name, p.description, p.specification, p.price_cents
FROM categories AS c
LEFT JOIN products AS p
  ON p.category_id = c.id AND p.is_listed = TRUE
WHERE c.is_active = TRUE
ORDER BY c.sort_order ASC, c.id ASC, p.sort_order ASC, p.id ASC
```

MySQL InnoDB 的单个非锁定 SELECT 以一个 statement snapshot 同时决定分类与商品，避免“两次查询之间分类被停用/商品被上下架”的撕裂；`LEFT JOIN` 保留启用空分类。一个 query 与顺序 fold 组装结果，不按分类追加查询，因此无 N+1，也不需要长事务、锁或 `FOR UPDATE`。

详情固定执行一个 `QueryRowContext`：

```sql
SELECT
  p.id, p.category_id, p.name, p.description, p.specification, p.price_cents
FROM products AS p
INNER JOIN categories AS c ON c.id = p.category_id
WHERE p.id = ? AND p.is_listed = TRUE AND c.is_active = TRUE
LIMIT 1
```

两个 query 都使用 request context、显式列名，不读取排序/开关以外的后台或 availability 字段。`sql.ErrNoRows` 只映射内部 `ErrProductNotFound`；query、scan、rows iteration/close 或数据不变量错误保持内部 error，HTTP 统一映射 503。repository 不构造第二 pool、不重试、不缓存。

相比“先查分类再批量查商品”的两 query transaction，单 join 更短且天然包含空分类；相比逐分类查询，它直接消除 N+1。相比把 availability join 进目录，它避免把尚未存在的日期/餐段库存模型伪造为目录字段。

### D4. Domain 值使用无符号数，HTTP DTO 显式编码字符串 ID

repository model 使用 `uint64` 保存 category/product id 与 category_id，使用 `uint32` 保存 `price_cents`，文本字段始终是 string。空 catalog 初始化为非 nil `[]Category{}`，每个 category 的 products 初始化为非 nil `[]Product{}`。

HTTP success DTO 与字段声明顺序固定为：

```text
catalogResponse{Categories []categoryResponse `json:"categories"`}
categoryResponse{ID string, Name string, Products []productResponse}
productEnvelope{Product productResponse `json:"product"`}
productResponse{ID string, CategoryID string, Name string,
                Description string, Specification string, PriceCents uint32}
```

`strconv.FormatUint(..., 10)` 把所有内部 BIGINT id 编为无符号、无前导零的十进制 JSON string；价格直接编码 JSON integer。DTO 不使用 `omitempty`，因此空文本是 `""`、空集合是 `[]`，字段不会缺失或变 `null`。

path id 在 HTTP 解码后先逐 byte 验证 ASCII `0-9`，再用 `strconv.ParseUint(..., 10, 64)` 且要求结果 `>0`；前导零有效，其他表示和 overflow 在访问 repository 前统一 404。这样不会把 `+1`、空白、浮点、Unicode 数字或 0 交给 SQL，也不额外创造 400 契约。

### D5. Catalog 自己注册两条 route，root router 保持唯一

`internal/catalog` 只包含 model、repository、handler 及 tests。handler 依赖一个只含 `List(context.Context)` 与 `GetProduct(context.Context,uint64)` 的窄 Reader seam，生产传入 concrete repository，httptest 传入 deterministic fake；这只服务两条已冻结读取行为，不是通用 repository 框架。

选定装配为：

```text
main.go
  └─ catalog.NewRepository(existingDB)
      └─ catalog.NewHandler(reader)
          └─ httpapi.NewRouter(logger, existingReadiness, catalogHandler)
              └─ catalogHandler.RegisterRoutes(existingGinEngine)
```

`RegisterRoutes` 只注册两个 GET；不创建 router/group middleware、鉴权或第二 engine。`httpapi.NewRouter` 采用直接的 `*catalog.Handler` 参数，不引入 generic plugin/registrar。router 仍统一设置 `HandleMethodNotAllowed`、middleware、health、NoRoute 与 NoMethod，因此 catalog 非 GET 继续 405 空 body、未知 path 继续 404 空 body。apply 时若 integrated foundation 的实际 readiness 签名与 APPROVED 设计不同，先更新 artifacts/approval，不能并行保留两个 constructor。

handler 只写冻结的 response structs：404 为 `PRODUCT_NOT_FOUND/product not found`，所有非 not-found repository error 为 503 `CATALOG_UNAVAILABLE/catalog temporarily unavailable`。handler 不把 raw error 传入 body 或 logger；现有 sanitized access log 只记录 request id、method、path、status 与 duration。

### D6. 真实 MySQL fixture 与 HTTP tests 分工，不用 mock 代替 W3

httptest 先冻结 exact JSON bytes/field types/empty arrays、GET-only、非法/未知/隐藏 404、fake repository error 503、无 repository call 的非法 id、现有 unknown 404/NoMethod 405 和 health/middleware regressions。standard-library counting SQL driver 或等价的本包 test seam 只用于证明 list 对任意分类数执行一次 query、显式 scan/fold 与 error mapping；不新增 go module dependency。

`services/api/scripts/catalog-integration.sh` 只校验 foundation 已冻结的结构化 `ORDER_TEST_MYSQL_*`、`ORDER_TEST_MYSQL_INSTANCE=order-mysql-w3` 与 `ORDER_TEST_MYSQL_ISOLATED=YES` 安全闩，再运行 catalog real integration tests；它不安装 runtime、不选择替代 DB、不打印 secret/DSN。Go test 只创建本次记录的 `order_test_<128-bit-random-hex>` schema，先从 clean v1 应用 v2/v3，再直接写受控 fixture，验证：

- migration 首次/重复/checksum、schema exact、RESTRICT、无 forbidden column/seed/down；
- 上架可见、分类停用/商品下架隐藏、稳定 `sort_order,id`、非负整数分；
- empty catalog、enabled empty category、single-snapshot list 与 query count one；
- visible detail、invalid/unknown/hidden 404，关闭 DB 后列表/详情 503；
- foundation live/ready/migrate 与 catalog routes 在同一真实进程的联合 smoke。

所有 PASS/FAIL/interrupt 路径都只尝试 drop 本次已创建、已记录且 prefix 精确匹配的 schema。目标为空、归属不明、prefix 不符或 cleanup 失败立即 FAIL，不扩大删除或换强命令。APPROVED 当前 runtime `NOT_ESTABLISHED`、W3 `NOT_RUN`；apply writer 在 foundation 集成后负责恢复其冻结的专属本地资产，这不是客户/平台 `BLOCKED_EXTERNAL`。

### D7. RGR、writer/verifier 与 C/T/V/R 只认当前 exact SHA 证据

Red 顺序固定为：migration/schema assertions → repository query/visibility/snapshot tests → httptest HTTP contracts → real MySQL foundation+catalog matrix；失败必须由目标实现缺失产生。Green 只追加两份 SQL、`internal/catalog/**` 与三个共享装配点的最小实现，再让同一检查通过。Refactor 只做职责/重复清理，并重跑同一 focused、real MySQL、全 API regression、race/vet/build/smoke。

writer candidate 还必须通过 strict、diff check、owned allowlist、protected zero-diff、migration forbidden schema/seed/down 和 sensitive scan。目标 `C=10,T=10,V=8,R=8` 只在真实 evidence 完整后记录；UI actual 始终 `NOT_RUN`。verifier 只在另一 clean detached worktree 对已提交完整 SHA 从空随机 schema 重跑全部 Gate，结尾确认 worktree 和 fixture clean。

### D8. Schema forward-only，失败回流不自动 down

迁移成功后不得删除、重命名或修改 000002/000003，也不提供 down。因为 foundation readiness 会拒绝 too-new schema，应用 v2/v3 后不能把未携带同一 embedded 集合的旧 foundation binary 当作可用 rollback。若 catalog binary 必须撤回，只能构建一个仍携带原 v1-v3 migration bytes、但移除 catalog route 装配的兼容修复 binary，或以前向更高 version 修正 schema；任何真实数据库修复/恢复都需要独立 review、恢复点和写授权。

测试 recovery 只清理本次随机 schema。implementation/test/spec/tasks/base/dependency/SQL bytes/name/number、router/DTO/error body、验收命令、rebase/merge 或 candidate SHA 任一变化都使旧 writer/verifier evidence 失效并从头重验。

## Risks / Trade-offs

- [上游仅 APPROVED，实际接口可能变化] → apply 硬门等其 INTEGRATED，吸收 latest main 后重新审计并同步四 artifacts/approval；不在旧 base 预实现。
- [单 join 重复 category columns] → 目录规模下换取单 statement snapshot、空分类和无 N+1；只有真实容量 Gate 失败才另立优化 change。
- [目录不返回售罄/图片/员工价，尚不是完整菜单体验] → 明确这是 W3/UI0 服务端目录切片；availability、图片和 client connector 分别独立集成后才宣称完整 UI。
- [unsigned auto-increment id 在 JS 中不安全] → API 永远返回十进制 string；只在 Go/MySQL 内使用 uint64。
- [应用 v2/v3 后旧 foundation binary schema-too-new] → 不做 down；失败通过携带相同 migrations 的兼容 binary 或 forward fix 恢复，并在部署 change 中重新验证顺序。
- [DB error 统一 503 降低客户端诊断细度] → 客户端只需要稳定可重试边界；内部底层错误不得泄漏，后续观测 change 可增加脱敏 reason，不改变公共 body。

## Migration Plan

1. DRAFT proposal/spec/design/tasks 已提交；本次只追加 APPROVED 治理记录，strict 与规划范围检查通过，所有 tasks 未勾，不安装 runtime、不修改业务文件。
2. 只有 foundation 已在 current main `INTEGRATED` 后，writer 才能申请进入 apply lane；开始前更新到 latest main，重读实际接口，若 artifacts 发生变化则重新 approval/strict。
3. 按 foundation 契约建立隔离真实 MySQL 8.0；先写 migration/repository/HTTP/real integration Red，再追加 000002/000003、catalog 包和最小 router/main 装配取得 Green。
4. Refactor 后从 clean random schema 重跑真实 matrix、全 Go Gate、foundation+catalog smoke、strict、owned/protected/sensitive，提交 exact candidate。
5. verifier 在另一 clean detached worktree 对 exact SHA 全量重跑；失败回原 writer产生新 SHA，任何变更从头验证。
6. 获得单独集成授权且依赖/verification 均有效后才能集成；`connect-miniprogram-menu-catalog` 此后才能开始。生产 rollout/deploy 不在本 change。

Rollback 不执行 down。测试只删除本次随机 schema；已应用真实 schema 如需撤回 route，使用仍携带相同 v1-v3 bytes 的兼容修复 binary，schema 变化只走更高编号 forward fix 或另行授权的恢复。

## Open Questions

无。产品可见性、字段、排序、空值、ID、价格、错误文案、依赖、ownership、真实 W3 与 rollback 边界均已冻结；图片、availability、员工价和小程序接入已明确拆为后续 changes。
