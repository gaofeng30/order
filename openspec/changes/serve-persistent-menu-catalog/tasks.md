> 状态：`APPROVED`。`approval_date=2026-08-13`；`approver=主 Agent`。主 Agent 在用户授权的自主裁决范围内批准本规划，不表述为用户亲自确认；批准依据为单能力 `persistent-menu-catalog`、`W3/UI0`、唯一冻结的目录/schema/HTTP/真实 MySQL 8.0 RGR 边界、完整 owned/非目标/依赖硬门/45 tasks 与 strict PASS。
>
> `gate_type=W3`、`ui_level_target=UI0`、`ui_level_actual=NOT_RUN`、`base_sha=5ba5340cf9098724c0eb2284fdc5b14cb97be5dc`、`candidate_sha=NOT_CREATED`。foundation exact SHA `c17ba4a5bfe7556b779fac093925df609358fe05` 当前仍仅 `APPROVED`；本 change 不得进入 `IMPLEMENTING` 或 apply，直到上游在 current main `INTEGRATED`。本轮所有任务保持未勾选，不安装 runtime、不修改业务文件。
>
> 每项完成后按 `docs/quality/change-quality-gates.md` 记录 change/gate/ui/base/candidate/phase、实际命令或操作、exit result、首个脱敏结果、artifact/environment、未验证边界和 asset/cleanup 状态。APPROVED 不评定实现 C/T/V/R；candidate 目标 `C=10、T=10、V=8、R=8` 只有在进入 apply 后真实证据齐全且硬阻断为零时才能记录。

## 1. Approval、Dependency 与唯一 Writer Gate

- [ ] 1.1 核验本 change 的 APPROVED 记录；重新完整读取 proposal、spec、design、tasks、根 `AGENTS.md`、质量门禁、canonical mvp/production/api-service-bootstrap 与 current main 的 foundation artifacts，运行 `openspec validate serve-persistent-menu-catalog --strict`，确认无 Open Question 后才能进入 `IMPLEMENTING`。
- [ ] 1.2 只读确认 `establish-mysql-persistence-foundation` 已在 current `main` 进入 `INTEGRATED`，不是 branch/APPROVED/CANDIDATE/INDEPENDENT_VERIFIED；未满足即保持阻断，不执行任何 Red 或共享路径修改。
- [ ] 1.3 依赖满足后把本 branch 吸收 latest main，记录新的完整 `base_sha`，重新审阅实际 config/database/migrate/readiness/router/main/migration embed 接口；若与本 APPROVED 规划有差异，先同步四类 artifacts、重新 strict 和 approval，旧证据全部失效。
- [ ] 1.4 确认 branch/worktree 是本 change 唯一 writer、开场 index/worktree clean、owned paths 无第二 writer；显式复核 router/router_test/main 三个共享路径已与其他 changes 串行。
- [ ] 1.5 按 integrated foundation 当前冻结的方式建立专属隔离真实 MySQL 8.0 本地资产，核对 engine/digest/loopback/随机凭据/安全闩/cleanup preflight；环境未建立或目标归属不明即 FAIL，不进入 Red，不转交客户/平台或冒充 `BLOCKED_EXTERNAL`。

## 2. Red: Migration、Schema 与安全 Fixture

- [ ] 2.1 先写 migration 文件集合测试，断言只追加连续的 `000002_create_categories.sql`、`000003_create_products.sql`，每文件 UTF-8/LF/末尾换行/单终止分号/单 `CREATE TABLE` statement，且无 seed/down/repair/force；运行并记录文件缺失的 Red。
- [ ] 2.2 先写 `information_schema` 真实 MySQL assertions，冻结两表字段、类型、null/default、PK/index、InnoDB、utf8mb4_0900_ai_ci 与 `ON UPDATE/DELETE RESTRICT`；断言无 stock/availability/employee/sales/soft-delete/store/tenant/image/timestamp 字段，运行并记录 Red。
- [ ] 2.3 先写 v1→v3 首次、current 重复零写入、version 2/3 name+checksum 与 applied_at 不漂移测试；修改 test migration bytes/history 制造 checksum mismatch，断言零重写，运行并记录 Red。
- [ ] 2.4 先写外键真实测试：存在引用商品时删除或 re-key category 必须被 RESTRICT，原数据保持；运行并记录 schema 尚不存在的 Red。
- [ ] 2.5 先写 fixture harness 安全 Red：只接受 foundation 的隔离实例安全闩，只创建本次记录的 `order_test_<128-bit-random-hex>` schema，PASS/FAIL/interrupt 都精确 cleanup；空名、prefix/归属不符或 cleanup 失败必须 FAIL 且不扩大删除。

## 3. Red: Repository Query、Visibility 与一致性

- [ ] 3.1 先写 list repository tests，冻结一个带 context、显式列名、`LEFT JOIN`、active/listed 过滤与 `c.sort_order,c.id,p.sort_order,p.id` 的单个 query；standard-library counting driver 断言任意分类数量都只执行一次且不存在 `SELECT *`/逐分类 N+1，运行并记录 Red。
- [ ] 3.2 先写 list fold tests，覆盖启用空分类 `products=[]`、无启用分类 `categories=[]`、空文本、多个分类/商品与相同 sort_order 的 id tie-break；断言 slices 非 nil、稳定顺序和 scan/rows error，运行并记录 Red。
- [ ] 3.3 先写 detail repository tests，冻结一个带 context、显式列名、products/categories join、listed+active 过滤和 `LIMIT 1` 的 query；visible 返回、`sql.ErrNoRows` 映射 `ErrProductNotFound`、其他 query/scan error 保留 unavailable，运行并记录 Red。
- [ ] 3.4 在真实 MySQL fixture 先写 visible/hidden matrix：上架+启用可见，停用分类及其商品隐藏，下架商品隐藏，empty catalog、enabled empty category、整数分、稳定排序全部按 spec；运行并记录 Red。
- [ ] 3.5 在真实 MySQL 两连接先写一致性场景：writer transaction 同时改变分类/商品可见性并控制 commit，list 的单 statement 响应必须只出现 commit 前或 commit 后完整 snapshot，不得撕裂；同时记录 query count one，运行并记录 Red。
- [ ] 3.6 先关闭受控 DB 连接再调用 list/detail repository，断言 context/error 返回且无 retry/cache/第二 pool；运行并记录 Red。

## 4. Red: HTTP Exact Contract 与 Router Regression

- [ ] 4.1 先写 httptest exact JSON：列表 envelope、详情 envelope、字段顺序/层级、十进制 string id/category_id、JSON integer `price_cents`、空文本 `""`、空数组 `[]`，并断言无 sort/active/listed/stock/availability/orderable/employee/sales/image 字段；运行并记录 Red。
- [ ] 4.2 先写匿名请求测试，证明两条 GET 不要求登录、微信会话、手机号或 RBAC，handler 不读取日期/餐段/库存/身份输入；运行并记录 route/handler 缺失的 Red。
- [ ] 4.3 先写 id parser table：1、前导零正整数有效；空、0、符号、空白、小数、十六进制、非 ASCII 数字和 uint64 overflow 在 repository 前统一 exact 404，运行并记录 Red。
- [ ] 4.4 先写 unknown/下架/停用分类商品相同 exact 404：`{"error":{"code":"PRODUCT_NOT_FOUND","message":"product not found"}}`，断言不存在性/隐藏原因不泄漏，运行并记录 Red。
- [ ] 4.5 先写 list/detail repository 任意非 not-found 错误的 exact 503：`{"error":{"code":"CATALOG_UNAVAILABLE","message":"catalog temporarily unavailable"}}`；用 SQL/DSN/connection/error canary 断言 body 与捕获日志不泄漏，运行并记录 Red。
- [ ] 4.6 先写 GET-only matrix：POST/PUT/PATCH/DELETE/HEAD/OPTIONS 对两条 catalog path 均 405 空 body且零 query；unknown path 仍 404 空 body，不得变成 PRODUCT_NOT_FOUND，运行并记录 Red。
- [ ] 4.7 重跑/扩展既有 health、request ID、sanitized access log、panic recovery、404/405 与 graceful lifecycle tests，先证明 catalog 装配缺失但 bootstrap 旧契约基线仍可观察；禁止通过改坏既有断言制造 Green。

## 5. Green: 最小 Catalog 纵向实现

- [ ] 5.1 只新增两份冻结的单 statement migration，使 2.1–2.4 Green；不改 embed/runner、go.mod/go.sum/README，不加入任何禁止字段、seed 或 down。
- [ ] 5.2 在 `services/api/internal/catalog/**` 实现 uint64/uint32 model、concrete `*sql.DB` repository、single LEFT JOIN list fold、single INNER JOIN detail 与稳定 sentinel，使 3.1–3.6 Green；不建 ORM、cache、retry、第二 pool或通用 repository abstraction。
- [ ] 5.3 实现窄 Reader seam、strict positive unsigned decimal parser、固定 DTO/error structs 与两个 GET handler，使 4.1–4.5 Green；不使用 `omitempty`，不记录/返回 raw error。
- [ ] 5.4 只在 `router.go` 直接接收并注册 catalog handler、在 `router_test.go` 保持共享契约、在 `main.go` 用 foundation 唯一 pool 装配 repository/handler，使 4.6–4.7 Green；不创建第二 router/middleware/pool，不修改 health/app/config/database/migrate。
- [ ] 5.5 新增 `services/api/scripts/catalog-integration.sh`，只校验 foundation 结构化 test 变量、实例名与隔离安全闩并运行真实 catalog tests；不安装/选择 runtime、不输出 secret/DSN，完成 2.5 Green。
- [ ] 5.6 在同一真实进程完成 foundation migrate/readiness + catalog list/detail 联合 Green：空 v1 应用 v2/v3、ready current、catalog success/404、断开 DB 后 ready/catalog 503，且随机 schema cleanup PASS。

## 6. Refactor 与 Forward Recovery Review

- [ ] 6.1 `gofmt` 并审查职责仅为 model/repository/handler；删除重复/死代码，确认 query/context/snapshot/DTO/error 仍唯一，不新增接口、配置或相邻优化。
- [ ] 6.2 从 clean random schema 重跑 2.1–4.7 的同一 focused 与真实 MySQL tests，记录 Refactor 后 Green；任何 SQL bytes/name/number、JSON、message、route 或 query shape 改动先同步四 artifacts/approval。
- [ ] 6.3 人工检查目录与 availability 边界：代码/schema/response 无 stock_type/stock_quantity/SOLD_OUT/AVAILABLE/orderable/date/meal-period/employee/image/sales，图片仍明确属于后续独立 change。
- [ ] 6.4 人工演练恢复设计：v2/v3 不 down；旧 v1-only foundation binary 因 schema-too-new 不可直接 rollback；撤回 route 只能使用仍携带相同 v1-v3 bytes 的兼容修复 binary，schema 修正只走更高 version forward fix或另行授权恢复。

## 7. Writer Verification 与 Candidate

- [ ] 7.1 运行 focused httptest/repository/migration tests 与 `test -z "$(gofmt -l services/api)"`，确认 exact JSON、GET-only、404/503、query count one、context 和 schema assertions全部 PASS。
- [ ] 7.2 运行 `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/...`、`go test -race ./services/api/...`、`go vet ./services/api/...`、`go build ./services/api/...`，记录当前 candidate diff 的真实结果。
- [ ] 7.3 运行 foundation 既有 smoke、真实 MySQL integration 与 `services/api/scripts/catalog-integration.sh`；记录 v1→v3 first/repeat/checksum/cleanup、visibility、sorting、empty/snapshot/no-N+1、DB disconnect 和 foundation+catalog joint PASS，UI actual 仍为 `NOT_RUN`。
- [ ] 7.4 运行 `openspec validate serve-persistent-menu-catalog --strict`、`openspec status --change serve-persistent-menu-catalog --json`、`git diff --check <base_sha>...HEAD`，确认四 artifacts 与实现一致、tasks 有真实证据且无行为未决。
- [ ] 7.5 运行 owned-path allowlist：只允许本 change、两份 migration、`internal/catalog/**`、router/router_test/main 与 catalog integration script；显式确认 go.mod/go.sum/README、app/config/database/migrate/health/middleware、apps、product/architecture docs、canonical/archive、skills/AGENTS 零 diff。
- [ ] 7.6 对全部 owned source/SQL/artifacts/script 执行 forbidden schema、`SELECT *`、seed/down 与敏感扫描；禁止 SQL/DSN/password/Authorization/Cookie、私钥/证书、手机号/身份、raw DB error/body，canary 必须证明 HTTP/log 不泄漏。
- [ ] 7.7 基于当前真实 evidence 才评定 `C=10,T=10,V=8,R=8` 且硬阻断为零；只暂存 owned paths并提交一个中文完整 CANDIDATE，记录 full SHA/base/命令摘要，确认 index/worktree clean。不得推送、创建/更新 PR、部署或写外部系统。

## 8. Exact-SHA Independent Verification 与 Integration Gate

- [ ] 8.1 verifier 在另一个 clean detached worktree 检出 7.7 完整 SHA，核对 exact base/candidate、foundation 已 integrated、owned/protected diff与开场 clean；用新随机 schema/按 foundation 规则重建干净真实 MySQL 8.0 环境。
- [ ] 8.2 verifier 只读重跑 7.1–7.6 与全部 2.1–4.7 场景，确认 exact JSON、schema/checksum、single snapshot/query count、DB断开、foundation+catalog joint matrix、敏感边界和 UI `NOT_RUN`；任何 FAIL 返回原 writer，新 SHA 从头重验。
- [ ] 8.3 verifier 确认每个随机 schema 已精确 cleanup、结束 worktree/index clean；清理目标不安全或 cleanup 失败立即 FAIL，不删除其他 profile/schema/宿主数据，不改用更强命令。
- [ ] 8.4 只有 exact-SHA independent PASS、foundation 依赖仍满足、C/T/V/R 与全部 Gate 有效且获得单独集成授权后才能进入 `INDEPENDENT_VERIFIED/INTEGRATED`；rebase/merge/main推进或任一 code/SQL/artifact/task/command/SHA 变化均重验。
- [ ] 8.5 本 change `INTEGRATED` 后才允许 `connect-miniprogram-menu-catalog` 开始；本任务不触发该 consumer、UI、push/PR、deploy 或 archive。
