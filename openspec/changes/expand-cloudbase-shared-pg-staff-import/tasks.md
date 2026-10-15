# Tasks

Status: IMPLEMENTED / WRITER_V4_GREEN / PRECOMMIT_SNAPSHOT_FROZEN / CANDIDATE_NOT_CREATED；W3/UI0（UI actual NOT_RUN）。最终source-v4完整ordinary/serialized race各1679PASS、16包、0FAIL/SKIP；普通UAT证据分为V2全矩阵与V4容量恢复subset，均保留准确源码层。提交前最终两轴、候选、独验与main集成仍待。第四exact base a6e597976e4ddee4ed2101ac21e9d1d4a3147983已INDEPENDENT_VERIFIED，前序changes未集成main。

## 1. 开工与证据基线
- [x] 1.1 核父exact aggregate independent回执，创建独立branch/WT并拷入四工件；核owned paths/index干净与native源码基线。
- [x] 1.2 冻结六named ABI及闭合材料/结果shape、真实SQL表列sequence清单；核定18个实际owned路径。

## 2. Staff纵向TDD
- [x] 2.1 List/Rate原生成功对共享缺RPC的实际Red→rawfacts Green；LIKE、mask、bound/STAFF非REFUNDED聚合与uint扫描边界。
- [x] 2.2 五kind逐一Red→Green；currentOWNER/settings/receipt/目标/输入顺序、空phone复用、nilEnabled、ignored/NUL/非法UTF8及完整摘要差分；五kind原StaffID审计target逐列比对，CREATE/SET_RATE覆盖0、非零、uint64最大值，不替换成新建ID。
- [x] 2.3 16同/异key、跨actor/currentauth、版本/无效目标、末尾receipt故障回滚、丢ack同key恢复及Quote源锁竞争；真实原handler/PC会话。

- [x] 2.4 公共Client容量合同修复：明确55逐项分类的24项精确HTTP200白名单/31项仍8MiB；三Import实际Red同caseGreen，24/31逐名策略与错误JSON/取消/超时/鉴权回归，无新业务限额。

- [x] 2.5 Staff command response codec同一容量边界：合法XLSX大Name→原公开SetEnabled小body，native/shared实际Red→Green；独立核已提交后503及同key恢复，保留before256/digest/严格codec。Standards实际原公开链路fresh/replay普通/race回执已通过，见V4证据。

## 3. 两类Import纵向TDD
- [x] 3.1 必要纯plan提取前后native回归；真实XLSX两类Preview缺RPC Red→Green，模板/文件/行数/列/重复/价格/phone/Unicode原行错误完整。
- [x] 3.2 Preview保留parse/plan→entropy→clock→OWNER→batch→receipt顺序；旧key不early replay，晚约束/duplicate rollback、原token恢复、丢ack。
- [x] 3.3 Commit两类缺RPC Red→Green，COMMITTED/当前授权/锁后expiry/skip/坏plan优先；真实rowlock跨expiry Red→Green与slow-material跨expiry晚CHECK/原错误映射/全回滚差分；不得以observed_at伪造最终committed_at。
- [x] 3.4 全批行/分类sort/version/batch/receipt原子，staff清plan/product留plan；16同token/不同key/跨actor、并发配置、receipt/batch故障全回滚与恢复，原HTTP完整链路。

## 4. 统一registry与真实UAT
- [x] 4.1 统一55固定RPC installer+repeat+Check、最小RLS/列ACL/sequence和额外权限漂移Red→Green；原44及旧业务body不变。
- [x] 4.2 受控UAT管理安装/历史字节保持/最终函数source对照；若需delta先旧→delta及rollback真实测试。
- [x] 4.3 普通服务JWT真实Go/原HTTP执行六入口；Staff五kind和两类导入矩阵、16竞争/独立DB读回、丢ack/最后故障/currentauth撤销，不重复全部本地编码矩阵。
- [x] 4.4 anon/其他UID/DDL/账本写拒绝、列/额外relation权限漂移failclosed；清理明确synthetic身份/数据/fault并保留baseline、ledger44和最终普通Check true。

- [x] 4.5 source-v4普通JWT真实Go/原HTTP大facts→1行合法XLSX Preview成功、独立batch/receipt读回及synthetic清理/55Check；v2大Staff证据仅按未变行为承接，不冒称新Client平台完成。SQL未变不重装/重登记历史。

## 5. Writer Gate与候选
- [x] 5.1 最终source-v4完整必需受影响ordinary/race、直接相邻Quote/catalog/身份/迁移回归、vet/build/smoke/fmt/diff/strict，日志与source manifest实证；既有失败/NOT_RUN/SKIP分列，不降低门禁。
- [ ] 5.2 Spec/Standards提交前review修复、C/T/V/R按证据评分、prepared clean包；最终owned/index/实际credential secret scan实证并核暂存后candidate提交，external receipt绑定exact SHA（不得自引用修改提交）。
- [ ] 5.3 另一个clean detached WT对exact SHA只读独验、两轴review和必需运行回执；失败回writer修新SHA重验。
- [ ] 5.4 依赖满足且独验后才按授权集成；未main集成不archive。完整runtime、真实WeChat/COS/95/25/UI3继续单独门禁。

## 决定性证据（2026-10-09，候选前快照）

外部根目录：`/Users/vivix/.codex/visualizations/2026/10/08/order-uat-status/`。本节各层以最终回执为准，早期writer回执中的ROOT_PENDING是当时边界，不覆盖后续root平台结果。

- 1.1：独立WT/base与初始空index已核；父lifecycle为INDEPENDENT_VERIFIED，exact a6e5979的1296 ordinary+1296 race、0FAIL/SKIP及两轴PASS已完成。仍未main集成。
- 2.1–2.3：`shared-pg-staff-admin-writer/final-writer-receipt.json`记录真实缺RPC/版本/long-name/codec Red→Green、五kind原audit target/digest、16并发、五receipt rollback/丢ack、原Gin字节差分；最终worker ordinary/race各120PASS。5000内部空格不设新cap，List时间恢复time.Local、fresh命令零时刻不变。
- 3.1–3.4：`shared-pg-staff-import-writer/final-writer-receipt.json`记录plan提取前后native各6PASS、缺RPC Red→Green、XLSX原handler/真实PC授权、两kind完整原子/并发/故障/丢ack；最终Import各154PASS，四包含直接相邻各539PASS。到期12PASS日志SHA `37d8be5cdc2cebd1ecf094889a0202e26c2c13ff446fb32fa62eaeee06c8e763`；末尾真实DB clock跨期仍InvalidInput全回滚。bytea/NULLscanner10PASS日志SHA `b6b28db790dafc0c84585bdd91033b906652bd29f405b86be2337a661dd44bf0`，保留原晚bytea文本语义。
- 2.3/3.4直接相邻源锁：`shared-pg-staff-import-root/cross-module-source-lock-final.jsonl` SHA `dfcb630f68fd2d3aea82d6673cd127680a3a7401ebd8d28815794c6ac4a1153b`，13PASS；native/shared各4Staff+2Import实际阻塞于Quote源锁，释放后旧80%/STAFF/160报价与immutableRead保持。
- 限定取消差异：`shared-pg-staff-admin-writer/native-concurrency-and-cancel-boundary.jsonl` SHA `d4a1da6a24a93d03d985ac873851194425fc0591ff37686c3ce2a5b4039a898f`，18PASS。三阶段真实HTTP取消均无可交付response、事务锁释放且staff/两version/audit无写；target内部native NotFound/shared Unavailable差异按design限定保留，不声称所有云端取消均回滚或错误类别完全等价。
- 4.1–4.4：`shared-pg-staff-import-root/final-platform-receipt.json` SHA `9c00200e5cf439e14dd07b42570a250a47edc6e70a09be8c09431008fb2b9171`。目标UAT PG17.11普通JWT→真实Go/原HTTP，54labels/189responses覆盖六新RPC；55完整源码、21RLS、10 USAGE，最终ordinary Check=true。Staff/Preview/Commit各16同/异key、3丢ack、6晚故障完整snapshot不变；owner撤销经真实PC auth得到401。anon六/其他UID六RPC42501，其他UID21表0行，DDL/ledger两项42501。25业务表回到baseline仅meal原2行，ledger44，临时身份删除读回0，runner退出0。
- 容量P2已闭合：旧8MiB真实Client+原publicHTTP Red保留，固定List成功路径修复后本地组合Green；`actual-cloud-large-list-receipt.json`真实云HTTP200，601行含600长姓名、9,709,994B、所有姓名逐字相同。管理seed符合原合法数据形状；public创建合法性由独立原native handler实验另证，不伪称云上600次POST。此证明覆盖该实测集合，不宣称平台无限响应容量。
- 4.2管理生命周期：`versioned-management-lifecycle-receipt.json`与平台总回执。批量Push Blob拒绝/无TaskId证据保留；官方Execute受审版本化事务→55全核验→Repair登记完成，history24，旧23相对fresh Describe baseline逐字不变。new executed SHA `8df687ebe37e4fe62c36b4d85f6bbbccb5bfdbf0b9daae41da02eb5fd0b69ae5` 与stored SHA `5031b53f7d5f87f1762e1bd1ac971aaaeaf27f92c2de628f54210953d0ada4d2`不同，精确3LF外围空白核验为RECORDED_FORMAT_VARIANT_VERIFIED，绝不称原文字节相同或通用规范化算法。

### 最终V4源码与验证

- `source-freeze-v4.json` SHA `19efda8e86a58be0ecba5af7c85b206540360c37784a79711add752d807dd25c`覆盖517文件，亲核逐文件0drift。容量修复依次为Client24/31精确HTTP200策略及Staff response codec移除人工cap；before256/strict/EOF/digest保持。三Import实际Red→真实Client/原handler Green，24/31逐名策略与错误/取消/鉴权回归完成，SQL55/ABI/ACL/history未变。
- ordinary：`final-v4-all16-ordinary-repaired-harness-receipt.json` SHA `e4e65364fa95bd5399dd5c83df93538278e23fa36a4033880820baaeda192cb7`；16包1679PASS/0FAIL/SKIP/exit0，日志SHA `dc49730eb4d5f4fdacca81fad4cd74ff90122976e8ab25b56b4bc16070cb37eb`。race：`final-v4-all16-race-serialized-receipt.json` SHA `43fa930801bbd8c834b5bd83172df75f9c7591a670dc7fe3f2e61c2d64357485`；16包1679PASS/0FAIL/SKIP/exit0，日志SHA `9e97164fc66b0d4da525846d623da409b9955431e9aabbabcfaa51b8d66e112a`。均包级串行，race detector开启。
- `final-v4-static-command-receipt.json` SHA `2b395a239336b83fe2aa2631a6a597d96af696c1eb5b72172b023d57e3849d9b`：vet/Linux allcmd构建/smoke exit0；最终四文档strict/diff及Go fmt核验后冻结。
- 2.5独立 `standards-staff-decoder-v4-fix-review` SHA `c0443a4613229bce9998a27a293d0a73c6619084c666f6a2a5f273ecc84802fc`：真实原9553B XLSX/9437456B worksheet导入9437186B name后16B SET_ENABLED，native/shared fresh/replay/recovery均200，普通/race专项各1PASS。旧V3提交后503及重复503不重复写保留，不误称无副作用。
- 最终候选前secret scan/owned/index与两轴review属于5.2的提交前条件，由root在冻结四文档后刷新。preliminary扫描仅记录其实际范围；不把未生成candidate或独验包填成已完成。

### V4平台容量与恢复subset

`capacity-v4-platform-receipt.json` SHA `4306b3230fcca22f42c5d6ce921af08802dbd3b4741b1d56621aa62f41a7b6ff`：普通JWT/真实Go/hash-backed PC/原HTTP；旧V3已经提交的原key由V4恢复200，25快照不变；新key200、replay200同体且不重复写，9MB List200。18001合法商品目录10440698B facts200→原Preview201/new1/error0，facts内容SHA `942385596be3014a542abb7edf20440bf949c32530d0907ed106461840f2f212`；独立batch PREVIEWED/row1/receipt1、无商品commit。25表count/hash恢复baseline仅meal2，55普通Check=true，runners0。

云Staff fixture按已独立公开XLSX证明的合法持久形状seed，不冒称云端执行了该大XLSX导入。旧54labels/189responses只属V2，不虚称V4重跑；V4只重测容量/恢复subset，其他增长型入口未逐一极值云测。SQL与权限/历史未变，无需重安装/登记。

### 保留失败与诊断边界

- V2 parallel race失败日志SHA `c561d8b95b2bf0511fc4cae55b9e9ad7dd8f0cbcc49f2ff5aa369bb9f426805c`保留；cluster-global authenticated角色成员对抗测试干扰并行其他库readiness。串行全包重跑仍启race，不删失败或跳过测试，不宣称同条件baseline产品既有失败。
- V4新临时PG56502普通失败日志SHA `bf0a5a56d76b91ffbb7d0578e687d41d5b5c92915499ce7eedaec2d73d716d6a`保留。临时host SCRAM与fixture无密码runtime role不匹配造成SQL前28P01；slow entropy全局rand.Reader与pgx SCRAM nonce交互造成递归等待。仅该隔离loopback HBA恢复既有fixture trust，517产品源/UAT认证不改。相同V4 focused各PASS后，上述完整ordinary1679已重跑通过；不以focused替代完整Gate。
- 容量P2原生成功/shared失败、harness-only误设及其修正均保留外部原日志。最终结果仅绑定明确source-v4，不将V2/V3测试数字混成一次运行。

## Writer冻结与未完成门

四工件现为候选提交前冻结快照。5.2–5.4不勾：root完成最终两轴、secret/owned/index核验后形成candidate；完整SHA、clean detached独验与后续状态仅写外部receipt，避免提交SHA自引用。若源码/规格实际修复则须新候选重验。全部changes未main集成，不archive。完整WeChat、真实支付、COS、CloudRun业务部署、95cases/25pages/UI3均未在本切片验收。
