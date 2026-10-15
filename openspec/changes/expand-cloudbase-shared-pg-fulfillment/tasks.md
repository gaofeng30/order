# Tasks

Status: APPROVED / IMPLEMENTING；gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN；base_sha=775e6746ded9555f9dd243682db011a869b72fbd；candidate_sha=NOT_CREATED。依赖首切片775e6746独验已完成、main集成未完成；ownership见proposal。

## 1. 需求与接口
- [x] 1.1 精读原生fulfillment、orderquery与subscription真实源码，记录重放差异、定位、consent锁序、Go cipher及商户查询只读身份依赖。
- [x] 1.2 需求文档结构Red：目标spec文件不存在，test -f exit1；建立四工件，不冒充业务Red。
- [x] 1.3 root/fulfillment writer确认最终固定RPC参数、proof和只读identity seam及文件ownership；冻结接口。
- [x] 1.4 OpenSpec strict与四工件一致性Green；按实际实现修正规格不扩大合同。

## 2. 履约与通知原子命令
- [x] 2.1 Red：同原生fixture通过Execute观察三个新增命令尚缺失；失败类型与状态明确。
- [x] 2.2 Green：MarkReady token生成/Seal与订单、consent、outbox、receipt原子提交；全部consent分支及错误映射。
- [x] 2.3 Green：TOKEN/CODE live优先、receipt fallback、上海日期冻结、已完成新key特殊Replay；REDEEM_ORDER合同回归。
- [x] 2.4 失败/竞态：当前授权、receipt损坏/跨scope、cipher/proof变动、随机/Seal失败、同key跨locator、16同异key、未知提交恢复。
- [x] 2.5 本地真实事务故障在outbox/consent/receipt步骤全回滚；同native状态/副作用差分。
- [x] 2.6 Refactor：仅必要共享逻辑，重跑受影响测试；无必要额外重构则记录理由。

## 3. 真实读取与HTTP合同
- [x] 3.1 Red：共享读取实现缺失，通过真实Application列表/详情对照观察Unavailable；其余五方法组合由Green矩阵验证，不伪造逐方法独立Red。
- [x] 3.2 Green：五端口列表/详情/历史投影、真实items与identity只读依赖；不使用stub当完成证据。
- [x] 3.3 查询负例与差分：owner隔离、授权/配置漂移、分页/LIKE、类型/金额/历史损坏、mask/token/通知选项/actions与30分钟阈值。
- [x] 3.4 现有四履约HTTP与四查询HTTP routes同原生差分，publicDTO/错误/no-store保持，真实reader贯通。
- [x] 3.5 必要Refactor后ordinary/race与相邻回归。

## 4. 安装、权限与真实UAT
- [x] 4.1 固定registry/allowlist、新version管理安装；原44不变；readiness检出新函数/表/策略/权限漂移。
- [x] 4.2 普通服务UID正例、anon/其他UID负例，新增表/sequence有效权限最小化与DDL/提升拒绝。
- [x] 4.3 真UAT Go普通JWT主链MARK_READY→用户解密token→TOKEN/CODE/ORDER核销→真实商户响应；consent/outbox/receipt独立读回。
- [x] 4.4 真UAT关键失败/16并发/提交后丢响应/受控故障回滚/当前授权竞态，逐证据标层，不冒充WeChat发送。
- [x] 4.5 清理合成fixture/故障定义/临时授权并readiness复读；保留最终业务定义/服务身份。

## 5. 候选及独验
- [x] 5.1 ordinary/race、vet/build/smoke、fmt/owned/secret/diff/OpenSpec及真实RGR回执，未运行不记PASS。
- [ ] 5.2 Standards/Spec两轴及C/T/V/R、候选exactSHA/clean外部验证包；V8仅候选形成后记实。
- [ ] 5.3 不同verifier在clean detached exactSHA复验；业务/源码/配置变化使相关旧证据失效。
- [ ] 5.4 核验依赖已集成与集成授权后方能集成；本change不自动获push/PR/merge授权。

## Evidence

证据根目录：`/Users/vivix/.codex/visualizations/2026/10/08/order-uat-status/`。以下为本地 Writer 层，不能替代本切片真实 UAT、候选或独验。

### 需求与真实 Red

需求 owner 已逐读 native fulfillment/orderquery/subscription/audit 与只读身份依赖，固定接口见 design。四工件结构 Red 为目标 spec 不存在（test -f exit1），Green 为 OpenSpec strict exit0；不将文档缺失或 compile 失败称为业务 Red。

`shared-pg-fulfillment-local-test/token-json-red-receipt.json`：四种 native 合法 token（内部NUL、HTML/引号、U+2028/U+2029、中文）native Execute 均完成，shared 均 Unavailable；4子项加父项 FAIL，日志 SHA256 `0c1a4034d6ec62ef310275d0d58ecbe67fcd369e09541d138576a2414cb21364`。`other-commands-red-receipt.json` 补 MARK_READY/CODE 的同类真实 Red，日志 SHA256 `7f97569450556b0245e4311b35825ada35f0a6d5dd8fba273d6ea394d5978b61`。三命令 Red 齐备。

`token-parser-receipt.json` 记录旧 object_keys/-> 两项22P05，新固定scalar方案4正例及7负例均符合预期；这是parser层证据，不独自证明整命令。`receipt-clock-order-red.jsonl` 记录三个旧receipt被新clock错误阻断的 native/shared 反例；后续Green保留原重放优先级。

`shared-pg-local-test/query-red.log` 记录 native list/detail 可用而 shared Unavailable；其他五方法组合由差分Green覆盖，不伪造每方法独立Red。`query-snapshot-red.log` 是旧双SELECT的结构Red，不冒称运行并发Red。修正后真实barrier并发读取证据与单SELECT结构要求分开记录。权限 Red 包含额外relation/列/sequence授权，以及未知SECDEF经anon实际泄漏但旧Check通过；修正后严格拒绝并允许撤EXECUTE的unused定义保留。

### 本地 Green 与必要 Refactor

`shared-pg-fulfillment-local-test/fulfillment-final-package-receipt.json`（SHA256 `25a3c0a1048289b423393d43f7b0dfb68aa8617508fa4c7da8bfe3e0aef84715`）记录统一 InstallSQL 下 fulfillment ordinary/race 各153 PASS、49顶层、0 FAIL/0 SKIP，覆盖三命令、最早consent所有分支、outbox/consume/receipt故障全回滚、合法token摘要、三clock微秒/旧receipt优先、16同异key、未知响应恢复、scope/proof/receipt竞态及四履约HTTP真实reader。必要Refactor仅移除被统一安装替代的临时定义/grant fixture；静态回执中 vet/build/fmt/diff 通过。20新增顶层的115项属于153子集，不相加。

最终联合权威回执为 `shared-pg-local-test/second-known-acl-final-package-receipt.json`：
- `go test -p=1 ./sharedpg ./internal/cloudbasepg ./internal/merchantidentity ./internal/orderquery ./internal/fulfillment -count=1 -json`：5包、377 PASS、0 FAIL/0 SKIP；ordinary日志SHA256 `65634a2a4b810a6ebea05abe7fdc9c046bd958df22aefd83b346fb2e4ef3acfa`。
- 同命令加 `-race`：5包、377 PASS、0 FAIL/0 SKIP；race日志SHA256 `1657e9b5b756a6ac43b443b6b8032b6050f6ba17194cb58a3a5a751cc6c63256`。

需求owner亲读回执并重算两个日志hash及pass/fail/skip计数一致。该轮包含最终已知ACL修正，替代旧374联合结果；闭合3.5。查询五端口的过滤/分页/身份完整性、历史投影、金额/二进制/UTF8/token损坏、30分钟阈值与四查询HTTP差分已覆盖。现有 identityRecord/projectIdentity 与查询投影仅作必要复用，未迁移身份写端。以上与153为重叠测试集，不累计宣称更多独立场景。

### 审查、平台与剩余门禁

已亲读 `shared-pg-fulfillment-review/root-precommit.md`，Standards root-owned14路径报告0 open actionable findings；SECDEF P2已以原泄漏反例修复，26表/16seq/354业务列+5ledger列与权威清单一致，首两条Redeem业务body保持base字节不变。该报告是precommit审查及writer日志核验，不是候选独验。

真实安装前 `shared-pg-fulfillment-platform-probe/prechange-permissions.json` 保存60条额外有效role-relation授权、10条显式列ACL、0额外可执行SECDEF。当前受控方案显式撤销权威资源宽授权后精确regrant；未知对象仅阻断不自动撤权。真实UAT安装、普通JWT业务链路、负例和清理现已核验，见下方最终平台回执；闭合4.1–4.5。

首切片 exact `775e6746ded9555f9dd243682db011a869b72fbd` 独验已完成，但尚未集成main；本切片不得提前集成。Writer Gate已由下方最终回执闭合；candidate exactSHA、干净detached独验与集成未完成，5.2–5.4保持pending。完整runtime、5个持续worker、私有存储最小权限与30天恢复、真实微信资料/发送、Mini UI3及95Case/25页面均不在本地Green中视为完成。


### 第二切片真实 UAT 业务准入

需求owner亲读 `shared-pg-fulfillment-platform-probe/platform-receipt.json` 及引用分项，重算SHA256为 `df73a04cb28cfe1bc5368699d2cf07dcabbe3fa1b5c425621a5dcbeba2d11057`。环境仅 `order-uat-d1gtzpzj50615c300`，普通服务UID沿用2108167831809454081。helper SHA256 `348a20fcb84e6719f5ed8a43733fa0b2b2232087ba81f24fb69d2bdea1198bd6`，businessSourceHash `a41171b7c1fc474950bd970f4271d45220d51a8336c3f88ad74ce8fafd1ced5a`，candidateSHA仍null。

4.1/4.2：受控版本20261008222000、task-a498be56 Succeed；安装SQL 275706字节、SHA256 `157dfb708e05de79cc737f8fa124c537d61c98aa25caaea8fde03252991f06a2`。实际12RPC、10固定UID RLS、44ledger及26表/16seq最小ACL对照通过，额外关系授权/可执行SECDEF为零。原始catalog 222092字节、SHA256 `2a3293c63b6d7811e248cc8317e97b9f0d3ccb1fb7af12841786e542ec70cf0b`，普通Check=true。RLS数量依据post-install-policies.json的实际10条policy与readiness；post-install-permissions.json旧聚合policy_count=0不作为计数依据。anon RPC/ledger为42501，其他UID RPC为42501、orders GET/outbox PATCH为0行；合法服务UID修改不可变outbox被42501拒绝。DDL/提升隔离按首切片未变身份证据及本轮严格metadata保留，不虚构本轮重跑提权探针。

4.3：四履约HTTP及四查询路由真实Go→CloudBase REST/RPC通过，用户READY解密token、商户隐私投影、其他owner404、uint64最大值、订单已完成后原MARK_READY key返回历史READY均有回执。独立main-http-readback记录3订单COMPLETED/version3、4receipt、1PENDING outbox及1consent消费。HTTP业务身份由合成transport auth注入；真实的是CloudBase普通JWT与业务RPC/读模型，不代表真实Order Bearer或WeChat登录已联调。

4.4：四命令各同/异key16并发，共8组；TOKEN/CODE异key按原合同1次状态变化、16条各key receipt，其余异key仅1成功，其余拒绝。4类提交后丢响应均原keyReplay且独立数据库无重复。4组跨locator各16请求为1fresh/7replay/8冲突，只有一目标变化。写次数以Replay和数据库version/receipt/outbox/consumed独立读回判断，不能将重放仍为true的Changed计为再次写入。真实outbox及receipt CHECK故障均Unavailable且状态/consent/outbox/receipt全回滚；撤故障后同keyfresh成功。proof后禁用账号、商户query403、旧receipt前置拒绝及cipher变动无版本变化均无业务副作用。猜测SUB_ACCOUNT/SUB枚举被数据库拒绝属探针输入错误，不是业务Red；合法SUBACCOUNT验证另有回执。

4.5：fault清理task-a4429c0a Succeed；最终11业务fixture表均0、fault约束0、ledger44，临时用户删除Success1/Failed0；最终普通Check=true，保留最终12RPC/RLS/服务身份。无Prod、收费资源或完整runtime部署。

5.1由下方最终Writer Gate闭合；5.2仅已有两轴precommit前置，不代表candidate exactSHA；5.3/5.4未完成。此准入只覆盖第二业务切片，不提升完整runtime、微信真实认证/发送/支付退款、CloudRun/static、UI3、95Case/25页面或main集成状态。


### 最终 Writer Gate 与文档冻结

已亲读 `shared-pg-fulfillment-writer/static-receipt.json`：五受影响包vet、全API host/Linux build、smoke、gofmt、diff、OpenSpec strict共7命令exit0。`owned-source-manifest.json`记录22个owned路径、unrelatedPaths为空、liveCredentialMatches=0；本次需求文档终稿后由root刷新清单/暂存核验，业务源码不变。

已亲读 `shared-pg-local-test/second-migrate-pg16-receipt.json`：完整internal/migrate ordinary/race各191 PASS、0 FAIL/0 SKIP，日志SHA256分别 `3db68cc5f44563f126adb2b9238453824f32e863e22aed5234e8b07e655cc707`、`27ba46409dc9833e14d9d9a804d14b7a8fdd2ab85dfc6d2e12f96206393a5fdb`。此前错误PATH选择PG17 dump对PG16恢复导致transaction_timeout失败，原失败证据保留；将PATH纠正到 `/opt/homebrew/opt/postgresql@16/bin` 后，同完整测试命令通过，未改业务源码、不标为基线缺陷、不跳过失败项。

结合五包ordinary/race377、真实UAT矩阵及清理、两轴precommit审查，5.1闭合。当前拟分C9/T10/R10；V8只有candidate exactSHA和clean detached验证包形成后才可记录，当前不预报V8/V9或总分。5.2尚缺候选绑定，5.3/5.4仍pending。candidate_sha在提交后外部receipt/harness绑定，四文档不为填写自身commit SHA再次修改。本轮strict/diff通过后四工件冻结；文档冻结不等于独验或集成。
