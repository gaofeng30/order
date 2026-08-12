> 状态：`CANDIDATE`。`approval_date=2026-08-13`；`approver=主 Agent`。主 Agent 在用户授权的自主裁决范围内批准本规划与 moving-main 后 apply，不表述为用户亲自确认；批准依据仍为单能力 `persistent-menu-catalog`、`W3/UI0`、唯一冻结的目录/schema/HTTP/真实 MySQL 8.0 RGR 边界、完整 owned/非目标/依赖/45 tasks 与 strict PASS。writer 已完成 RGR/全部 Gate，等待 exact-SHA independent verification。
>
> `gate_type=W3`、`ui_level_target=UI0`、`ui_level_actual=NOT_RUN`、`base_sha=cbfb803bb74f34b4b39fd0feff2c753613a06de2`、`candidate_sha=SELF`（本地 commit生成并在 handoff 绑定完整 SHA）。foundation candidate/integrated exact `14fc3c3b10eda28a1c61cc0ac552ca46d1cb14e1` 已随 archive/main exact `cbfb803bb74f34b4b39fd0feff2c753613a06de2` 进入 `ARCHIVED`；moving-main 已无冲突、无 merge commit完成，旧 planning exact SHA 失效。README 只允许最小同步当前事实；2026-08-13 全量 Gate 的 `PROTECTED_MIGRATION_EMBED_TEST_FOUNDATION_ONLY` #1 与 `PROTECTED_FOUNDATION_MYSQL_INTEGRATION_ASSUMES_V1_ONLY` #1 已保留，主 Agent在用户授权的自主裁决范围内只批准新增对应两个测试 ownership，分别精确更新完整 v1-v3 链与显式分层 v1原语/current v1-v3 场景，不表述为用户亲自确认；persistent-menu-catalog spec 行为不变。
>
> 当前 writer evidence：`C=10、T=10、V=8、R=8`，总分 36、单项均不低于 8、硬阻断 0；V=8 的未验证边界是 exact-SHA independent verification 尚未运行。每项证据按 `docs/quality/change-quality-gates.md` 记录脱敏决定性结果，未把 UI、verifier、integration 或 archive 冒充 PASS。

## 1. Approval、Dependency 与唯一 Writer Gate

- [x] 1.1 核验本 change 的 APPROVED 记录；重新完整读取 proposal、spec、design、tasks、根 `AGENTS.md`、质量门禁、canonical mvp/production/api-service-bootstrap 与 current main 的 foundation artifacts，运行 `openspec validate serve-persistent-menu-catalog --strict`，确认无 Open Question 后才能进入 `IMPLEMENTING`。
  - Evidence: `base=cbfb803bb74f34b4b39fd0feff2c753613a06de2; phase=writer; command=openspec instructions apply + validate --strict + full context read; exit=PASS; summary=45 tasks、W3/UI0、无 Open Question，主 Agent 已批准 IMPLEMENTING; unverified=实现尚未开始`。
- [x] 1.2 只读确认 `establish-mysql-persistence-foundation` candidate/integrated exact `14fc3c3b10eda28a1c61cc0ac552ca46d1cb14e1` 已随 current archive/main exact `cbfb803bb74f34b4b39fd0feff2c753613a06de2` 进入 `ARCHIVED`，canonical 与真实 W3 Gate 完整。
  - Evidence: `phase=writer; action=main/archive/canonical exact audit; exit=PASS; summary=main=cbfb803、foundation implementation=14fc3c3，archive/canonical MySQL+health 存在且 strict 前提完整; unverified=catalog W3`。
- [x] 1.3 核验两个 planning commits 已无冲突、无 merge commit线性重放到 new `base_sha=cbfb803bb74f34b4b39fd0feff2c753613a06de2`，重新审阅实际 config/database/migrate/readiness/router/main/migration embed/README；确认与 approved 行为一致，旧 exact SHA 与旧证据失效。
  - Evidence: `phase=writer; action=git rebase --onto exact main + API audit; exit=PASS; summary=planning commits线性重放，无冲突/merge；唯一 README ownership 冲突经主 Agent 重批，spec 字节不变; unverified=Red/Green`。
- [x] 1.4 确认 branch/worktree 是本 change 唯一 writer、开场 index/worktree clean、owned paths 无第二 writer；显式复核 router/router_test/main 三个共享路径已与其他 changes 串行。
  - Evidence: `phase=writer; action=branch/worktree/status/ownership audit; exit=PASS; summary=codex/serve-persistent-menu-catalog 独立 clean worktree，主 Agent确认 README 与共享装配点无并行 writer; unverified=候选结束 clean`。
- [x] 1.5 按 integrated foundation 当前冻结的方式建立专属隔离真实 MySQL 8.0 本地资产，核对 engine/digest/loopback/随机凭据/安全闩/cleanup preflight；环境未建立或目标归属不明即 FAIL，不进入 Red，不转交客户/平台或冒充 `BLOCKED_EXTERNAL`。
  - Evidence: `phase=writer; action=colima/docker exact preflight + internal version/schema count; exit=PASS; summary=Colima 0.10.3、exact name+双 label唯一 healthy MySQL 8.0.46 arm64/v8，127.0.0.1随机端口、noexec/nosuid tmpfs、zero mounts、guest无/Users mount、order_test_残留0; artifact=order-mysql-w3; unverified=catalog W3`。

## 2. Red: Migration、Schema 与安全 Fixture

- [x] 2.1 先写 migration 文件集合测试，断言只追加连续的 `000002_create_categories.sql`、`000003_create_products.sql`，每文件 UTF-8/LF/末尾换行/单终止分号/单 `CREATE TABLE` statement，且无 seed/down/repair/force；运行并记录文件缺失的 Red。
  - Red: `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/catalog -run '^TestCatalogMigrationSet$' -count=1` exit 1，首个目标失败 `migration count = 1, want 3`。
- [x] 2.2 先写 `information_schema` 真实 MySQL assertions，冻结两表字段、类型、null/default、PK/index、InnoDB、utf8mb4_0900_ai_ci 与 `ON UPDATE/DELETE RESTRICT`；断言无 stock/availability/employee/sales/soft-delete/store/tenant/image/timestamp 字段，运行并记录 Red。
  - Red: 专属 MySQL 8.0 运行 `TestCatalogSchemaIntegration` exit 1，v2/v3 缺失使 exact schema assertions 无法成立；cleanup 后残留 0。
- [x] 2.3 先写 v1→v3 首次、current 重复零写入、version 2/3 name+checksum 与 applied_at 不漂移测试；修改 test migration bytes/history 制造 checksum mismatch，断言零重写，运行并记录 Red。
  - Red: 同一真实测试已冻结 first/repeat/history/checksum-drift 零重写矩阵，首个失败仍为 migration count 1/3，未伪造后续 PASS。
- [x] 2.4 先写外键真实测试：存在引用商品时删除或 re-key category 必须被 RESTRICT，原数据保持；运行并记录 schema 尚不存在的 Red。
  - Red: 同一真实测试已冻结 delete/re-key RESTRICT 与原行保持；v2/v3 未存在时 exit 1，随机 schema 精确清理。
- [x] 2.5 先写 fixture harness 安全 Red：只接受 foundation 的隔离实例安全闩，只创建本次记录的 `order_test_<128-bit-random-hex>` schema，PASS/FAIL/interrupt 都精确 cleanup；空名、prefix/归属不符或 cleanup 失败必须 FAIL 且不扩大删除。
  - Red: 初次发现 `CATALOG_CLEANUP_CLOSED_SERVER_DB` 后只枚举并删除唯一匹配残留，修复 defer 次序；复跑 `red_exit=1 remaining_order_test_schemas=0`。候选脚本尚不存在时 exit 127，未选择替代环境。

## 3. Red: Repository Query、Visibility 与一致性

- [x] 3.1 先写 list repository tests，冻结一个带 context、显式列名、`LEFT JOIN`、active/listed 过滤与 `c.sort_order,c.id,p.sort_order,p.id` 的单个 query；standard-library counting driver 断言任意分类数量都只执行一次且不存在 `SELECT *`/逐分类 N+1，运行并记录 Red。
  - Red: focused repository test exit 1，首个目标失败为 `NewRepository`/model 未定义；counting driver 已冻结单 query、显式列与禁用 `SELECT *`。
- [x] 3.2 先写 list fold tests，覆盖启用空分类 `products=[]`、无启用分类 `categories=[]`、空文本、多个分类/商品与相同 sort_order 的 id tie-break；断言 slices 非 nil、稳定顺序和 scan/rows error，运行并记录 Red。
  - Red: 同一 focused test exit 1；fold、non-nil empty slices、稳定顺序及 query/scan/rows/invariant error assertions 已先于实现落盘。
- [x] 3.3 先写 detail repository tests，冻结一个带 context、显式列名、products/categories join、listed+active 过滤和 `LIMIT 1` 的 query；visible 返回、`sql.ErrNoRows` 映射 `ErrProductNotFound`、其他 query/scan error 保留 unavailable，运行并记录 Red。
  - Red: 同一 focused test exit 1，`ErrProductNotFound` 与 repository 尚未定义；detail query shape/error mapping assertions 已存在。
- [x] 3.4 在真实 MySQL fixture 先写 visible/hidden matrix：上架+启用可见，停用分类及其商品隐藏，下架商品隐藏，empty catalog、enabled empty category、整数分、稳定排序全部按 spec；运行并记录 Red。
  - Red: 专属 MySQL 命令运行 real behavior test exit 1，catalog production package 尚不存在；fixture/visible-hidden/empty/sort/整数分 assertions 已落盘，残留 0。
- [x] 3.5 在真实 MySQL 两连接先写一致性场景：writer transaction 同时改变分类/商品可见性并控制 commit，list 的单 statement 响应必须只出现 commit 前或 commit 后完整 snapshot，不得撕裂；同时记录 query count one，运行并记录 Red。
  - Red: `TestCatalogSingleStatementSnapshotIntegration` 与 counting-driver test 均已先写；real command exit 1 于 catalog implementation 缺失，随机 schema 残留 0。
- [x] 3.6 先关闭受控 DB 连接再调用 list/detail repository，断言 context/error 返回且无 retry/cache/第二 pool；运行并记录 Red。
  - Red: unit 与 real behavior tests 已冻结 canceled/closed DB list/detail 和恢复矩阵；focused build exit 1 于 repository 未定义。

## 4. Red: HTTP Exact Contract 与 Router Regression

- [x] 4.1 先写 httptest exact JSON：列表 envelope、详情 envelope、字段顺序/层级、十进制 string id/category_id、JSON integer `price_cents`、空文本 `""`、空数组 `[]`，并断言无 sort/active/listed/stock/availability/orderable/employee/sales/image 字段；运行并记录 Red。
  - Red: `go test ./services/api/internal/catalog -run '^TestHandler'` exit 1，首个目标失败为 Category/Product/Handler 未定义；exact bytes/type/forbidden-field assertions 已落盘。
- [x] 4.2 先写匿名请求测试，证明两条 GET 不要求登录、微信会话、手机号或 RBAC，handler 不读取日期/餐段/库存/身份输入；运行并记录 route/handler 缺失的 Red。
  - Red: 同一 httptest 直接无身份 header 调用，两条 route/handler 尚缺失导致 build exit 1；测试未加入身份/date/餐段输入。
- [x] 4.3 先写 id parser table：1、前导零正整数有效；空、0、符号、空白、小数、十六进制、非 ASCII 数字和 uint64 overflow 在 repository 前统一 exact 404，运行并记录 Red。
  - Red: parser table 与 zero-reader-call assertions 已先写，focused build exit 1 于 `parseProductID`/Handler 缺失。
- [x] 4.4 先写 unknown/下架/停用分类商品相同 exact 404：`{"error":{"code":"PRODUCT_NOT_FOUND","message":"product not found"}}`，断言不存在性/隐藏原因不泄漏，运行并记录 Red。
  - Red: fake not-found 与真实 hidden matrix均已先写，focused/real commands exit 1 于实现缺失；404 body 单一冻结。
- [x] 4.5 先写 list/detail repository 任意非 not-found 错误的 exact 503：`{"error":{"code":"CATALOG_UNAVAILABLE","message":"catalog temporarily unavailable"}}`；用 SQL/DSN/connection/error canary 断言 body 与捕获日志不泄漏，运行并记录 Red。
  - Red: list/detail error canary、closed DB 与 exact 503 assertions 已先写，focused build exit 1 于 Handler 缺失。
- [x] 4.6 先写 GET-only matrix：POST/PUT/PATCH/DELETE/HEAD/OPTIONS 对两条 catalog path 均 405 空 body且零 query；unknown path 仍 404 空 body，不得变成 PRODUCT_NOT_FOUND，运行并记录 Red。
  - Red: 12 组 method/path、零 query与 root unknown tests 已先写，focused build exit 1 于 route/Handler 缺失。
- [x] 4.7 重跑/扩展既有 health、request ID、sanitized access log、panic recovery、404/405 与 graceful lifecycle tests，先证明 catalog 装配缺失但 bootstrap 旧契约基线仍可观察；禁止通过改坏既有断言制造 Green。
  - Red: base `cbfb803` 已有 foundation bootstrap PASS；扩展 root router tests 后 focused command exit 1 `catalog: no non-test Go files`，既有 health/middleware/lifecycle assertions未放宽。

## 5. Green: 最小 Catalog 纵向实现

- [x] 5.1 只新增两份冻结的单 statement migration，使 2.1–2.4 Green；不改 embed/runner、go.mod/go.sum，不加入任何禁止字段、seed 或 down。
  - Green: migration focused test与真实 `TestCatalogSchemaIntegration` PASS；v1→v3、repeat/history/checksum、exact schema/index/RESTRICT全部通过，只有 000002/000003 两个单 statement文件。首次全量 test/race 保留 `PROTECTED_MIGRATION_EMBED_TEST_FOUNDATION_ONLY` #1 Red；主 Agent重批唯一测试路径后，exact v1-v3 embed package test/race均 PASS，`embed.go`/runner零修改。
- [x] 5.2 在 `services/api/internal/catalog/**` 实现 uint64/uint32 model、concrete `*sql.DB` repository、single LEFT JOIN list fold、single INNER JOIN detail 与稳定 sentinel，使 3.1–3.6 Green；不建 ORM、cache、retry、第二 pool或通用 repository abstraction。
  - Green: catalog unit全包 PASS；counting driver query count 1，real visibility/sort/empty/snapshot/closed DB/recovery matrix PASS，无 ORM/cache/retry/第二 pool。
- [x] 5.3 实现窄 Reader seam、strict positive unsigned decimal parser、固定 DTO/error structs 与两个 GET handler，使 4.1–4.5 Green；不使用 `omitempty`，不记录/返回 raw error。
  - Green: handler exact JSON、匿名、ID table、404/503与 forbidden/leak assertions PASS；IDs 为十进制 string、价格为 JSON integer、空集合非 null。
- [x] 5.4 只在 `router.go` 直接接收并注册 catalog handler、在 `router_test.go` 保持共享契约、在 `main.go` 用 foundation 唯一 pool 装配 repository/handler，使 4.6–4.7 Green；不创建第二 router/middleware/pool，不修改 health/app/config/database/migrate。
  - Green: root httpapi全包 PASS；catalog GET-only/空 body 404/405 与既有 health/request ID/access log/recovery均通过，main 仅复用现有 `db` 装配。
- [x] 5.5 新增 `services/api/scripts/catalog-integration.sh`，只校验 foundation 结构化 test 变量、实例名与隔离安全闩并运行真实 catalog tests；不安装/选择 runtime、不输出 secret/DSN，完成 2.5 Green。
  - Green: 脚本在 exact专属环境 exit 0，两个 real packages PASS；只检查 7 个结构化变量、instance/isolated闩，结束残留 0且输出无 secret/DSN。
- [x] 5.6 在同一真实进程完成 foundation migrate/readiness + catalog list/detail 联合 Green：空 v1 应用 v2/v3、ready current、catalog success/404、断开 DB 后 ready/catalog 503，且随机 schema cleanup PASS；最小更新 README 的目录 API、v1-v3 migration、匿名 curl、真实 MySQL 验证与非目标，保留 production SSM fail-fast。
  - Green: `TestFoundationAndCatalogIntegration` + schema/behavior integration PASS，覆盖 ready/catalog 200、hidden 404、关闭 pool 后 ready/catalog 503与 residue 0；README 只同步获批事实且 production SSM fail-fast原文保留。

## 6. Refactor 与 Forward Recovery Review

- [x] 6.1 `gofmt` 并审查职责仅为 model/repository/handler；删除重复/死代码，确认 query/context/snapshot/DTO/error 仍唯一，不新增接口、配置或相邻优化。
  - Refactor: `test -z "$(gofmt -l services/api)"` PASS；人工审查仅有 model/repository/handler，single list/detail query、Reader/DTO/error各一套，无死代码/新配置/通用抽象。
- [x] 6.2 从 clean random schema 重跑 2.1–4.7 的同一 focused 与真实 MySQL tests，记录 Refactor 后 Green；任何 SQL bytes/name/number、JSON、message、route 或 query shape 改动先同步四 artifacts/approval。
  - Refactor Green: catalog/httpapi/migrations focused packages PASS；catalog integration race 两包 PASS，schema/checksum/visibility/sort/snapshot/HTTP/断连矩阵不变，`refactor_remaining_order_test_schemas=0`。
- [x] 6.3 人工检查目录与 availability 边界：代码/schema/response 无 stock_type/stock_quantity/SOLD_OUT/AVAILABLE/orderable/date/meal-period/employee/image/sales，图片仍明确属于后续独立 change。
  - Review: 对 production catalog source + 000002/000003 精确扫描 `production_forbidden_fields=0`；测试只以 forbidden assertions出现这些词，README/四 artifacts明确图片与 availability 为后续 change。
- [x] 6.4 人工演练恢复设计：v2/v3 不 down；旧 v1-only foundation binary 因 schema-too-new 不可直接 rollback；撤回 route 只能使用仍携带相同 v1-v3 bytes 的兼容修复 binary，schema 修正只走更高 version forward fix或另行授权恢复。
  - Review: migration seed/down文件计数 0；current `migrate.Check` 对旧集合识别 `schema_too_new`。恢复只允许携带原 v1-v3 bytes 的兼容 binary或更高版本 forward fix，未执行真实恢复写入。

## 7. Writer Verification 与 Candidate

- [x] 7.1 运行 focused httptest/repository/migration tests 与 `test -z "$(gofmt -l services/api)"`，确认 exact JSON、GET-only、404/503、query count one、context 和 schema assertions全部 PASS。
  - Writer: gofmt zero diff；catalog/httpapi/migrations focused exact contract packages exit 0，exact JSON/GET-only/404/503/query count/context/schema chain均 PASS。
- [x] 7.2 运行 `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/...`、`go test -race ./services/api/...`、`go vet ./services/api/...`、`go build ./services/api/...`，记录当前 candidate diff 的真实结果。
  - Writer: full test/race/vet/build 全部 exit 0；order-api/order-migrate 双 cmd 安全临时构建 PASS且临时文件已精确清理。首次 stale embed test Red 已按批准修复后同 package test/race PASS。
- [x] 7.3 运行 foundation 既有 smoke、真实 MySQL integration 与 `services/api/scripts/catalog-integration.sh`；记录 v1→v3 first/repeat/checksum/cleanup、visibility、sorting、empty/snapshot/no-N+1、DB disconnect 和 foundation+catalog joint PASS，UI actual 仍为 `NOT_RUN`。
  - Gate Red: `mysql-integration.sh` exit 1，首错 fresh current set `AppliedCount=3` 与旧硬编码 1 冲突，后续 synthetic v2/v3 与正式 checksum 冲突；指纹 `PROTECTED_FOUNDATION_MYSQL_INTEGRATION_ASSUMES_V1_ONLY` #1，`remaining_order_test_schemas=0`。主 Agent批准后只允许修改该 test file。
  - Writer Green: smoke、`mysql-integration.sh`、`catalog-integration.sh` 均 exit 0；current fresh v1-v3=3/repeat=0、原语/故障、checksum、visibility/sort/empty/snapshot/query-count/断连恢复与 joint routes PASS，真实 order-api 进程覆盖迁移前 catalog 503、外部 CLI 后 ready/空目录/列表/详情/隐藏404；每轮 schema residue 0。JS syntax 与 42 JSON static PASS；本 change 无 UI，`ui_level_actual=NOT_RUN`。
- [x] 7.4 运行 `openspec validate serve-persistent-menu-catalog --strict`、`openspec status --change serve-persistent-menu-catalog --json`、`git diff --check <base_sha>...HEAD`，确认四 artifacts 与实现一致、tasks 有真实证据且无行为未决。
  - Writer: strict PASS、status 四 artifacts done、45 tasks结构保持；pre-commit cached diff-check PASS，commit 后 exact base...candidate diff-check由 handoff绑定，spec 与 `.openspec.yaml` 相对获批版本字节不变。
- [x] 7.5 运行 owned-path allowlist：只允许本 change、两份 migration、精确更新的 `services/api/migrations/embed_test.go`、`internal/catalog/**`、精确更新的 `services/api/internal/migrate/mysql_integration_test.go`、router/router_test/main、catalog integration script 与根 README；显式确认 `services/api/migrations/embed.go`、go.mod/go.sum、app/config/database、除获批单测外的migrate、health/middleware、apps、product/architecture/cloud docs、canonical/archive、skills/AGENTS 零 diff，并确认 README 只含批准的最小事实更新。
  - Writer: 21 changed files全部命中批准 allowlist；protected zero diff、behavior spec bytes unchanged PASS。README 只含 catalog API/v1-v3/curl/验证/非目标，production SSM fail-fast原文保留。
- [x] 7.6 对全部 owned source/SQL/artifacts/script 执行 forbidden schema、`SELECT *`、seed/down 与敏感扫描；禁止 SQL/DSN/password/Authorization/Cookie、私钥/证书、手机号/身份、raw DB error/body，canary 必须证明 HTTP/log 不泄漏。
  - Writer: production forbidden/sensitive scan、migration seed/down count 0、script syntax/mode PASS；HTTP/log canary tests PASS，唯一 JSON tags为批准字段/error envelopes，无 raw repository error记录或返回。
- [x] 7.7 基于当前真实 evidence 才评定 `C=10,T=10,V=8,R=8` 且硬阻断为零；只暂存 owned paths并提交一个中文完整 CANDIDATE，记录 full SHA/base/命令摘要，确认 index/worktree clean。不得推送、创建/更新 PR、部署或写外部系统。
  - Candidate: `C=10,T=10,V=8,R=8`、hard blockers 0、`candidate_sha=SELF`；本条随中文本地 commit完成，完整 SHA与 post-commit clean在 writer handoff 绑定，避免 SHA 自嵌导致再变化。未 push/PR/deploy/外部写。

## 8. Exact-SHA Independent Verification 与 Integration Gate

- [ ] 8.1 verifier 在另一个 clean detached worktree 检出 7.7 完整 SHA，核对 exact base/candidate、foundation 已 integrated、owned/protected diff与开场 clean；用新随机 schema/按 foundation 规则重建干净真实 MySQL 8.0 环境。
- [ ] 8.2 verifier 只读重跑 7.1–7.6 与全部 2.1–4.7 场景，确认 exact JSON、schema/checksum、single snapshot/query count、DB断开、foundation+catalog joint matrix、敏感边界和 UI `NOT_RUN`；任何 FAIL 返回原 writer，新 SHA 从头重验。
- [ ] 8.3 verifier 确认每个随机 schema 已精确 cleanup、结束 worktree/index clean；清理目标不安全或 cleanup 失败立即 FAIL，不删除其他 profile/schema/宿主数据，不改用更强命令。
- [ ] 8.4 只有 exact-SHA independent PASS、foundation 依赖仍满足、C/T/V/R 与全部 Gate 有效且获得单独集成授权后才能进入 `INDEPENDENT_VERIFIED/INTEGRATED`；rebase/merge/main推进或任一 code/SQL/artifact/task/command/SHA 变化均重验。
- [ ] 8.5 本 change `INTEGRATED` 后才允许 `connect-miniprogram-menu-catalog` 开始；本任务不触发该 consumer、UI、push/PR、deploy 或 archive。
