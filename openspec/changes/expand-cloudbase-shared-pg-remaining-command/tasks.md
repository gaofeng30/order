# 剩余命令切片任务

Status: APPROVED_ATOMIC / IMPLEMENTED / LOCAL_VERIFIED / UAT_SYNTHETIC_VERIFIED / CANDIDATE_PENDING / INDEPENDENT_NOT_RUN。gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN。

## 1 需求和基线
- [x] 1.1 核对已独验父SHA、批准20 ABI/24 owned paths/构造器与原消费点纠正；父验证回执见proposal。仅源码/需求核实，不计本切片业务Red。四工件首次 `openspec validate expand-cloudbase-shared-pg-remaining-command --strict` 实际exit0；24 owned paths无重复，消费点纠正与20 ABI同批准源。
- [x] 1.2 原native公共方法基线及各Shared完整业务Red，记录失败来源；registry缺函数不替代业务Red。

## 2 三模块业务 RGR
- [x] 2.1 退款五方法：locator跨attempt冻结、权限/receipt/状态优先、NULL订单、原金额/摘要、Create已知提交后外调、未知状态持久化、current-owner与回放。
- [x] 2.2 观测/恢复：原payment→order→refund锁序、dedupe原Go字节、所有状态非零SuccessTime在INSERT消费、pgx整数µs/wrap/范围/scan、claimCreate缺失NotFound和query缺失Unavailable；finishCreate历史Local/DST同RPC材料阶段、原late codec错误与callback胜出、fresh/replay字节及丢ack；lease/query/create/callback交错。
- [x] 2.3 固定可选通知：nil与非nil构造、validIntent优先、独立晚clock、最早consent不得跳过、已有outbox精确匹配、insert/consume/退款整体rollback。
- [x] 2.4 订阅RecordConsent/RunDue：原校验与clock/entropy顺序、named unique有限重读、claim单条SKIP LOCKED、livelease三CAS、临时失败max(cutoff,completedAt)+1min、provider外调/计数。
- [x] 2.5 RunProductionDue：原排序/30min边界/µs/非法version、逐订单原子推进和SYSTEM_EVIDENCE、原两次重试与部分完成计数。
- [x] 2.6 原公共handler/实际构造consumer与完整持久facts差分；16同/异键/跨actor、支付/核销/推进/退款竞争、丢ack新实例恢复、晚audit/order/outbox/consume故障与取消；Refactor仅必要重复代码，不改native规则，重跑同场景。

## 3 统一能力与权限
- [x] 3.1 90固定RPC+实际列/26RLS/16USAGE清点，普通UID/Invoker/UTC/searchpath/role/额外relation/column/sequence/SECDEF权限负例；原44ledger/checksum不变。
- [x] 3.2 封闭wire/nullable/fulluint64/原宽JSON decoder/strict材料消费优先级、错误SQLSTATE与named unique不扩张；全部业务子事务rollback。
- [x] 3.3 20新增响应按原健康合同保留8MiB，最大合法字段/100due/pending、错误/401/单JSON/timeout真实Client验证；无新业务cap。

## 4 普通 UAT 切片
- [x] 4.1 父独验与本地review满足后root受控版本化安装；原history/执行登记字节分别核验、实际普通身份完整Check。
- [x] 4.2 普通JWT→Go原公共接口/handler+合成provider；20入口正向、关键16并发/未知ack/原子故障/当前权限/lease/通知矩阵，外调时无DB事务持有。
- [x] 4.3 anon/其他UID RPC拒绝/RLS0行、DDL/ledger写入拒绝、原readiness ledger只读权限、权限漂移拒绝与恢复；runtime无admincredential。
- [x] 4.4 fixture/临时身份/fault/probe精确清理，业务count/hash回到原baseline、ledger/RPC/readiness/source绑定。真实微信退款/订阅送达单列NOT_RUN。

## 5 生命周期
- [x] 5.1 必需影响面ordinary与serialized race、vet/Linux build/smoke/fmt/diff/strict、源码无漂移及实际秘密扫描。失败/fixture诊断/未跑/skip分列，不降Gate。
- [ ] 5.2 Writer Gate与Spec/Standards review、owned-only staged检查、本地candidate和external exactSHA包；不在提交内自引用SHA。
- [ ] 5.3 clean detached exact candidate完整独验；修复必须writer新SHA并重验。
- [ ] 5.4 依赖满足且独验通过后另行集成main；当前不push/merge/archive，不称完整UAT上线。

## 验证命令与证据记录

整切片待执行/收束的计划命令（已完成子包结果另列）：在services/api执行 `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -p 1 ./internal/refund ./internal/subscription ./internal/orderadvance ./internal/fulfillment ./internal/paymentorder ./internal/cloudbasepg ./internal/migrate ./sharedpg -count=1`，同影响面加`-race`。按实际新依赖扩充必要相邻包，不以该初始清单遗漏consumer；本地PG DSN由writer私有注入，不能用其他writer端口。`go vet ./...`、`CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/...`、`bash scripts/smoke.sh`；repo根`openspec validate expand-cloudbase-shared-pg-remaining-command --strict`。用原handler完整调用链验证，不新造HTTP成功替身。

每项完成记录exact源/命令/退出码、native与shared结果、环境、sanitized回执SHA和未覆盖层。首次四工件结构验证只能记录文档PASS，不使上述业务/平台任务完成。候选与独验后只更新外部lifecycle证据，已冻结提交不为写自身SHA而修改。

## 最终实现与普通UAT证据

证据根为本机external `order-uat-status`；各SHA以回执文件内容为准。以下均为本change实际完成的验证层，候选精确SHA和后续独验只记external lifecycle，不自引用提交。

- 1.2、2.1–2.6：refund leaf `shared-pg-remaining-command-refund/leaf-final-v3-handoff.json`记录ordinary/race各173 PASS；subscription/production leaf `shared-pg-remaining-command-subscription-production/leaf-handoff-all-closed-materials-final.json`记录各298 PASS，均0 FAIL/SKIP。包含原native基线、公开方法业务Red/Green、完整Time四字段、fulluint64/nullable、原错误优先、原receipt/dedupe字节、16并发、lease/unknown ack及late rollback。Refactor只固定RPC适配和批准材料编解码，native算法/11权威源不改；不需要额外重构。
- 完整影响面：`shared-pg-remaining-command-root/final-v3-runtime-receipt.json` SHA a6522c77c70015cc2529e7c13a50c5a5f1ee02dff79314607764017358203985；20包ordinary/race各2437 PASS、0 FAIL/SKIP。vet、Linux全部cmd构建、smoke、fmt、diff和strict均exit0；两套PG16.13集群停止并删除。该运行器的candidate_sha字段为执行当时父HEAD dde945，并非新候选；决定性绑定是545个Go/SQL文件`source-freeze-v3.json` SHA e667e03fdc36782f688480c263df24b4343424e17387258a25701f3553d5622c，实际source/owned20 drift均0。
- 3.1–3.3：完整90 registry、26RLS、16sequence USAGE，既有44迁移账本count/hash不变。固定Time结构、named SQLSTATE、单JSON/401/timeout/8MiB健康合同由完整Client与各模块测试覆盖。最大合法字段与100条由native/shared本地差分覆盖；另以原HTTP/公开方法生成100条合法订单退款，在普通UAT真实Go Client上得到due/pending各100条，见`capacity100-real-client-receipt.json`。云端容量实验不声称最大字段逐一实际验证。
- 4.1：`registered-migration90-v3-receipt.json`记录执行、实际函数、登记历史及普通身份Check；版本20261009121500。旧30条CloudBase历史已逐条读回原字节不变；首次全量历史Query超过1MiB请求上限，被平台拒绝、未登记，随后采用同源21函数+完整access的402823字节精确delta，执行与存储Query分别绑定SHA，未猜测结果或重复不可逆步骤。故障安装/清理、权限探针安装/清理均另有版本记录。
- 4.2：`platform-business-events.json`及`new20-business-runtime-coverage.json`覆盖20/20新增入口。普通JWT→原Go公开构造器/HTTP handler→真实PG；provider为合成。包括16同键/跨actor恢复、Create未知结果、finishCreate第二次RPC提交后丢返回、provider Create/Query期间NOWAIT无锁、退款成功单一observation/audit/outbox/consent、订阅临时/永久失败和成功/丢返回恢复、生产推进丢返回和16并发恢复、Query失败release、原取消handler及late receipt/consent/audit/notification整体rollback。实际provider付款/退款/消息未调用。
- 4.3：其他真实普通UID对20个入口HTTP403/SQLSTATE42501，anon HTTP401/42501；新增关系RLS为0行，服务UID有合法行；DDL及ledger写入拒绝。既有ledger SELECT44是readiness授权，不称拒绝读取。临时DELETE权限漂移使Check拒绝，撤销后Check恢复；未执行DELETE业务。runtime子进程凭据经stdin，仅普通服务身份，无管理凭据。
- 4.4：`final-platform-business-cleanup-25-table-receipt.json`记录两轮probe退出0、25业务表count/hash回到基线（meal_periods仅原2行）、native ledger44/hash不变、payment observations/故障约束/probe表0、临时认证用户实际删除读回、普通90 Check ready=true。首轮cleanup保留于`first-platform-business-cleanup-25-table-receipt.json`。fixture/harness诊断独立保留；错误ledger只读查询SQLSTATE42883无写入，随后使用原sha256表达式核验。
- review：`shared-pg-remaining-command-independent/final-v3-precommit-review.json` SHA c4f4b598382010d94687ad730226ca8dacd0a29e0379996b82fc0719c4db2204是源码两轴0 mustfix，尚非候选独验；四工件实际完成记录后重新review和冻结。最终秘密扫描`private-values-scan-final.json`实际304文件/64项私密值、0命中；5.1已完成。Writer Gate/owned-only staged和候选见external结果，5.2在全部子门完成前不勾。

## 未运行与后续责任

真实微信支付/退款、订阅授权/送达、PC与小程序页面、CloudRun五worker宿主、95cases/25pages/UI3均NOT_RUN，不是本切片的合成provider PASS。AppSecret和商户材料已恢复且官方只读验证通过；实际订阅模板0，老板姓名/手机号尚未确认。新确认的共用商户支付环境隔离另立change，依赖本切片及read-report独验，不能用当前迁移等价测试替代其业务验收。5.3/5.4未完成；不推送、集成main、归档或宣称完整UAT上线。
