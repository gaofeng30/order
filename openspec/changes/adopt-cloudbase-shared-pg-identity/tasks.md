# Tasks

Status: IMPLEMENTED / SCOPED_UAT_VERIFIED；gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN；base_sha=e4657b11eadc2c87a0d6606038ff3c97d49b9b00；candidate_sha=NOT_CREATED。第二候选精确SHA独验已PASS，前两切片未main集成。

## 1. 需求与接口
- [x] 1.1 亲读端口图及native PC/账号/商户/Session/Phone/bootstrap关键合同，明确三种身份与外部材料边界。
- [x] 1.2 冻结18新增RPC、构造/ownership/Go provider及entropy阶段、clock/hash/receipt和管理RR方案。
- [x] 1.3 writer逐参数与原生错误分支复核；OpenSpec strict及四工件一致性通过。

## 2. Session 与 Phone
- [x] 2.1 实际native成功/shared缺能力Red，不以compile失败代替。
- [x] 2.2 原事务/bytes/时间边界/碰撞回滚/immutable phone与provider调用数Green。
- [x] 2.3 真实handler/session中间件、logout单token、并发phone、未知提交及必要Refactor回归。

## 3. Merchant Store
- [x] 3.1 Start/Complete/Recover/Extra真实缺能力Red。
- [x] 3.2 登录当前账号、phone savepoint、拒绝audit持久、same-code恢复与provider最多一次Green。
- [x] 3.3 Extra精确hash/锁/pricing/receipt及当前值冲突、故障回滚/并发/未知响应差分。

## 4. Admin 与 PC
- [x] 4.1 账号CRUD/current/list及PC各阶段实际Red。
- [x] 4.2 current OWNER/lastowner/集合锁/原receipt/版本及错误优先级Green。
- [x] 4.3 PC WAITING零entropy、并发一次消费、到期/消费与entropy失败barrier、批准后段/当前authVersion/单tokenlogout。
- [x] 4.4 真实HTTP与Order opaque bearer链、16并发/未知commit/故障及必要Refactor回归。

## 5. 管理 bootstrap 与权限
- [x] 5.1 真UAT管理入口在首次query前建立RepeatableRead能力探针。
- [x] 5.2 原三组created/exactunchanged/conflict、非全库empty、默认/partial漂移、并发和末步骤失败native差分。
- [x] 5.3 普通身份不可bootstrap/配置写及完整平台权限验收（普通JWT真实正负例及管理面隔离已核）。
- [x] 5.3a 本地30固定runtimeRPC、12表/5seq精确ACL/RLS/readiness及额外列权限漂移负例。

## 6. 真实UAT及候选
- [x] 6.1 普通JWT业务矩阵、当前权限/审计/receipt/16并发/故障/unknowncommit；管理合成bootstrap单独验证。
- [x] 6.2 全部临时fixture/故障/身份清理，业务定义/权限/readiness复读。
- [x] 6.3 必需八包ordinary/race、相邻回归、vet/build/smoke/fmt/diff/strict、25 owned路径/secret/index核对及真实RGR验证；root提交前刷新最终四文档指纹。
- [ ] 6.4 两轴precommit及C/T/V/R、候选exactSHA与外部clean验证包。
- [ ] 6.5 clean detached精确SHA独验；变更使旧验收失效。
- [ ] 6.6 依赖main集成及授权核验后集成；无自动push/PR/archive。

## 当前验收证据

外部证据根为 `/Users/vivix/.codex/visualizations/2026/10/08/order-uat-status/`。以下均为亲读回执与决定性记录后的分层结论，不将测试子项累计为独立业务场景。

### 本地实现与必要相邻回归

`shared-pg-identity-writer/final-writer-receipt.json` SHA256 `6cfff501dbf2eb8f1fac4c96ddd62820af8426f7fa70fee584a561f986d33b36`：identity138、merchantidentity215、bootstrap48，ordinary/race各401 PASS，0 FAIL/0 SKIP。统一真实v44安装下覆盖会话/Phone、Start/Complete/Recover/Extra、Admin/PC、固定管理bootstrap及实际opaque会话HTTP差分。

`shared-pg-identity-common/required-adjacent-final-ordinary-receipt.json`及原始JSONL已亲核：五包cloudbasepg72、sharedpg17、fulfillment153、orderquery63、migrate191，共496 PASS/0 FAIL/0 SKIP，整组exit0，日志SHA `f5a82a7494dc064c2b30b2b84848fffa92af194047c160bcadad24600f09eec4`。`required-adjacent-race-receipt.json`同五包496 PASS/0 FAIL/0 SKIP、exit0。加writer三包401，每种模式必需八包共897 PASS。先前`required-adjacent-ordinary-receipt.json`因错误migrate路径整体exit1，保留为历史命令失败、明确不计该命令PASS；最终依据为纠正后的五包整组回执。

`shared-pg-identity-common/final-static-source-receipt.json` SHA256 `68fd26bba1f0fe07a21e494d0a67a3124d3aafcfd1cb0cdb6184bcb895c7ce28`：全services vet、Linux amd64 CGO0 build、正确路径smoke、gofmt、diff检查均exit0；482源码文件前后聚合SHA `737aef936c5bb81e8cf7bd5e10c0eaaebe2eaa8e921cd8420d8f0183e50be547`一致。先前smoke错误路径exit127不计通过。

决定性Red/Green保留在writer目录：缺RPC的真实行为Red后端口Green；account不可编码/ignored字段8项native基线；MAX uint64 scope改精确div；PC锁等待跨expiry原3 FAIL（c407f3e…）到同场景3 PASS（bfc3ff9…），最终SQL-clock focused34 PASS。Approve最终3参，Poll首NULL锁后observed_at、二次冻结，slowentropy跨expiry及后续expired零熵均已实证。最终18签名manifest SHA `cab6a9cadbc03b6115321078f355c4055ad2e1ed5f72573aaf7e7cc87d209e1a`。

本次移除非必要id UPDATE授权后，原fulfillment fault fixture仅撤id已不能制造故障；必要调整为撤全部有效UPDATE并断言has_any_column_privilege=false，同原Unavailable断言已纳入153项Green，不改核销业务body。无需要的额外Refactor。

### 真实UAT普通身份与管理面

`shared-pg-identity-platform-probe/platform-receipt.json` SHA256 `a4e0547083d4eb92733b798093ef17078f05d62972caecd62d88c4b98fd48fbf`及其全部evidence_files哈希已复核一致。版本20261009000500、task-13adcab2 Succeed，安装SQL SHA `0e543c84b2cde61831233f59ca18c4a080a5c85e79a366943a7113657ccbb534`。普通UID2108167831809454081真实REST加原Order opaque bearer服务链，30固定RPC/12 RLS表/5seq；微信provider为合成，非微信真联调。

C01–C09涵盖16会话、跨用户Phone、登录绑定、Extra、四kind各16、同key跨命令、跨OWNER lastowner及PC批准/消费。跨actor采用同批真正16并发；C07独立readback仅1 enabled OWNER，8成功/8原503，旧PC/receipt401。未知提交L01–L07均实际丢已提交响应并独立读库；PC原raw token不可恢复，无新重放能力。Recover云端同code成功/异code拒绝且proof/binding各1；完整provider阶段及额外负例由本地native差分覆盖，不重复声称全部云测。

F01–F06覆盖用户/session、绑定/拒绝audit、Extra、account receipt、PC消费及bootstrap末步故障全回滚，撤fault后相同请求恢复。F04原23514为ACCOUNT_CONFLICT409，初探针误期望503不是业务Red。管理RR真实空三组首次query barrier16得到1 created/15 unchanged，15次实际named23505按原最多2次规则恢复；sleep/GUC仅探针，不进入产品SQL。

anon/其他UID RPC42501、其他UID读取0行/INSERT42501，普通服务配置INSERT及主键UPDATE42501；最终Go Check=true。清理后25业务关系除原meal_periods两行均0，ledger44、fault0；临时UID删除后查询Total0，探针结束，保留正式普通服务身份及受审定义。PC精确到期等号/锁等待/slowentropy以本地真实PG差分为据，不冒云端复测。

### 明确未完成及既有失败

clean e465同条件复现可选main TestAcceptanceUserBoundariesAreFailClosed原生READY503，日志SHA `b6bd888f3cb9607a22958414022f687f4c0a6fb0b71a5d4d50f6557c8808ac93`。归类已存在的可选失败，不计PASS、不扩大修复；四个server未启用UI1/UI3场景为skip，不能替代本切片必需门。

本切片未生成候选，exactSHA须提交后写外部candidate receipt以避免自引用；干净detached独验及依赖main集成分别待完成。真实微信、客户owner初始化、CloudRun完整main、其余业务端口、存储及95Case/25页面/UI3均未由本切片证明。原始探针错误和旧clock日志保留external，不混入最终Green。

最终收尾：OpenSpec strict实际exit0。`shared-pg-identity-common/final-owned-secret-receipt.json`亲核25 owned、73源码及回执文本逐字扫描76个当前任务敏感值，0 match、index为空；不声称已删除历史工具输出。四文档本轮收束后由root刷新spec-only指纹及扫描回执；业务source保持上述静态回执绑定。Spec precommit检查0 findings见外部`shared-pg-identity-common/final-precommit-spec-review.md`；Standards及候选步骤仍由root独立收尾，6.4不提前勾选。
