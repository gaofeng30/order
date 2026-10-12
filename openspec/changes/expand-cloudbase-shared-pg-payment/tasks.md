# 支付切片任务

Status: APPROVED_ATOMIC / WRITER_VERIFIED / CANDIDATE_NOT_CREATED；W3/UI0，V3限定普通UAT合成provider支付切片PASS；当前V4格式修正受影响回归/源码核验PASS，真实UI/微信支付NOT_RUN。父第五exact 5f5af0cd05eafbddfb660631adf6e12daf450a87已INDEPENDENT_VERIFIED，未集成main。

## 1 需求与证据基线
- [x] 1.1 冻结六Application/15 named ABI、24 owned paths及root/leaf私有协议；原生63PASS/0FAIL/SKIP，overlay registry15缺失+parent16FAIL。证据shared-pg-payment-root/native-and-registry-red-receipt.json SHA e3d932894a92dd1ced3f9bba291efd74640a20b8b72c97612fc46bfee0d81a52；native权威12文件hash匹配。只代表baseline/registry Red，不代表共享业务通过。正式四工件首次openspec validate expand-cloudbase-shared-pg-payment --strict已PASS；未计实现/平台测试。
- [x] 1.2 原公共六方法真实native成功/shared失败Red及逐功能Green、纯codec边界；实现前不计registry Red为业务Red。

## 2 原子支付行为
- [x] 2.1 receipt/raw读取：三action、scope/摘要、strict256/1024、target/当前权限优先、损坏/未知字段、完整uint64。
- [x] 2.2 Prepare完整Finalize/source锁/窗口→SQL96bit熵→canonical request/claim；原provider已知提交后一次调用、finish成功/unknown终态容忍、原独立receipt；named23505与原max2重读。
- [x] 2.3 claim/query/close/observe：锁后leaseNow、cutoff/receivedAt原时点、全纳秒Normalize/hash、受信Go同源canonical时间及year1000000非PAID/PAID范围差分、NUL与invalidUTF8、PAID-close竞态、owner/version/lease晚响应。
- [x] 2.4 materialize/manual：最早观测、当前OWNER顺序、冻结聚合/原金额digest、defer持久分支、9999序号及order/items/prepay/observation原子性。
- [x] 2.5 六原handler/完整durable差分；16同/异key/quote/回调/手工自动竞争、丢ack/进程恢复、provider次数、所有晚步故障、取消/锁等待/无deadline活性及跨第五源锁验证。

## 3 统一接入
- [x] 3.1 统一70函数install/repeat/readiness：exact body/signature/Invoker/UTC/searchpath/role、原44ledger/checksum；实际表列/序列清单与额外relation/SECDEF/role权限漂移拒绝。
- [x] 3.2 SQL_FAILURE闭合三成员、四named unique/三retry状态、未知失败及全业务子事务回滚；原重试次数/provider0证据；QUOTE_UNAVAILABLE_RETRY单字段仅原quote临时不可用来源，Prepare snapshot不retry、Materialize坏snapshot持久defer、lateCAS/权限/unknown不retry。
- [x] 3.3 15精确新增名仍8MiB，55旧容量分组不变；原最大合法canonical字段/receipt/due100/pending100/uint64max与错误/401/EOF/timeout回归。

## 4 实际普通UAT切片
- [x] 4.1 先闭合父第五exact独验及自身本地review；管理面受控部署70，实际普通服务UID全量readiness，原迁移/history按官方生命周期核验。
- [x] 4.2 普通JWT→实际Go六方法/原handler+合成provider，关键16并发/未知ack/故障回滚/当前授权/lease/source锁矩阵；真实外部支付另列NOT_RUN。
- [x] 4.3 anon/其他UID全部新RPC拒绝、RLS零行、DDL/ledger/额外grant负例与恢复；runtime无宽管理key。
- [x] 4.4 精确合成fixture/临时身份/故障与probe清理，baseline逐表count/hash、44ledger、70body/权限/readiness恢复与source绑定。

## 5 生命周期
- [x] 5.1 必需影响面ordinary及隔离serialized race、vet/build/smoke/fmt/文档diff/strict/source无漂移、秘密扫描；V3完整+V4纯格式受影响验证分层见下。最终owned-only staged diff属于5.2，必须实际重过，不继承此前未覆盖新SQL的普通diff。
- [ ] 5.2 最终两轴review、owned-only暂存核验、本地candidate及外部exactSHA/clean verification包；不在同提交自引用SHA。
- [ ] 5.3 干净detached exact candidate独立两轴及受影响回归，无writer修改；失败新SHA重验。
- [ ] 5.4 满足父依赖且独验通过后另行集成main；当前不push/merge/archive，不称完整runtime/95/25通过。

## 最终证据与范围

全部路径以external shared-pg-payment-root为准；Red/中间失败保留在对应leaf/root目录，未重命名为PASS。

|层|决定性结果/回执|
|---|---|
|历史源码V3|V3 531文件，source-freeze-v3.json SHA f306414c9155991ad9539f2be94319e802012a5bd57a5c8489c8b6f1ec6721a0，ordinary/race/static零漂移|
|历史V3 ordinary|17包1944 PASS/0FAIL/SKIP、exit0；final-v3-all17-ordinary-receipt.json SHA 9e3ca0c5451ce209860c0089e1f09521c92a1129bfd57fa632fcedde59dd4e5e|
|历史V3 serialized race|17包1944 PASS/0FAIL/SKIP、exit0；final-v3-all17-race-receipt.json SHA ae5398150fe37d4e55dd4bf4548aa94cd28dcad767c9d177802aca2e9113a1fa|
|原时间/消费|39PASS；paid-time-and-infinite-consume-final-green-receipt.json SHA 255bee496a1d75048cb52fa6a1311e711caa7fdbea08613e44d5ca62ebafa061；PG17五定点oracle含精度/wrap/±Infinity消费零写|
|历史V3平台源码/管理|final-70-source-44-ledger-29-history-readiness-receipt.json SHA 0a937011270b0d3d75e1aea9652c1c1a0187a62d213ec9e582fc1713b8f32183；70精确body/config/Invoker、44ledger不变、29history、全部15正向RPC、ordinaryCheck true|
|V3业务清理|final-platform-business-cleanup-25-table-receipt.json SHA 248c59250c580b285f3fa838b74dc21f196adf59a6fd2c8722e10d56467f80ea；25表count/hash恢复原样、meal2不变、runner exit0、fault/trigger0、临时UID删除|
|历史V3静态|vet/Linux amd64 build/gofmt/diff/smoke exit0；final-v3-static-checks-receipt.json SHA ed20f708512e5c81cf4cef20f48da5bb959aa248b739ba34bd33bf03c4e78418；本次最终四文档strict另实跑|
|秘密扫描|24 owned文件27当前实际私值0命中，final-owned-24-private-value-scan.json SHA 30bbb4087a5a0f3776253ecfb0f12e6e73e4f8791905ca3da422d9200e8c2f54；130 external文件0命中，final-root-evidence-private-value-scan.json SHA 9a34679b78de7bb9e6288437213260cf37d9fabeae350a931e750b44cadc2843；最终stage前按修改范围复核|

V3平台业务（V4未重跑）：16同key预支付唯一prepay/providercreate1与稳定重放；16 callback仅观测不提前订单，8manual+8自动唯一order/item/counter；provider Create/Query/Close持有时独立JOIN可见可锁，晚NOTPAY/CLOSED不覆盖PAID。五commit后丢ack跨新实例恢复且application外调0；当前OWNER撤销阻断旧manual receipt；坏snapshot持久DEFER/PENDING_MANUAL；六晚故障六表snapshot原样，只有40001原尝试2，未知23505/CAS为1且新Create0。anon/otherUID各15RPC拒绝、otherUID24RLS零行、DDL/ledger拒绝、extra immutable列权限使Checkfalse且恢复true。最大合法canonical/wx字段及100due/100pending/maxuint平台完整wire通过。

V3管理：Push Blob限制无TaskId后回读未变，官方versioned Execute成功并完整Check后才Repair登记；旧24Query逐字不变。主安装7LF与四小probe格式差异分别经具体词法核验。未执行V2 full草稿标SUPERSEDED；最终完整bundle SHA fb66810fe9ec16ace25a2bea0d149e9897a168cb081ce3e560fa7a726bc9a0fd及实际versioned16定义与当时V3一致，非当前V4安装源。

RGR/Refactor：receipt/prepare/observe/materialize有真实native成功/shared失败→Green；配置四string/hex与JSON alias/null/未知字段优先级、1ns lease、fulluint64、固定+08时区、item锁、PAID整数wrap/Infinity消费均有决定性差分。复用原typed facts，无新配置/通用helper；仅必要空循环清理/格式化。早期compile/fixture空header/FakeNotification Query计数/JS解析舍入/UTC显示与真实产品Red分列。旧V2 race中断标INCOMPLETE/SUPERSEDED，不计V3通过。

5.2/5.3/5.4仍未完成。当前为提交前文档快照；最终candidate SHA在外部lifecycle receipt绑定，独验后只更新外部回执，避免同提交自引用。限定合成provider UAT通过不等于真实微信支付、CloudRun、存储恢复、95cases/25pages或UI3；全部change仍未集成main。

## 当前V4补充回执（不覆盖V3历史）

|层|实际结果|
|---|---|
|V4 源|531源，4SQL仅尾空白，527不变；source-freeze-v4.json SHA 3d7807a66a86f6e6eafe30bff2db3fb801c98e82ddae86a2aa9595edf505e5b8|
|V4 ordinary|sharedpg/migrate/paymentorder：471 PASS、3包、0FAIL/SKIP、exit0、前后0漂移；final-v4-affected3-ordinary-receipt.json SHA 0edf8dd8cb4752411eabaaeec574d80e06b2c7c82a259d05cebe535b9d710973|
|V4 race|相同3包471 PASS、0FAIL/SKIP、exit0、前后0漂移；final-v4-affected3-race-receipt.json SHA 48133c66fb5e223d86e2276280583734bb258c6427073cf1f89adb08d938039b|
|V4 平台|14变化body+完整70源/ordinaryCheck，44ledger/旧29Query/25表baseline不变；当前30history；v4-final-70-body-44-ledger-30-history-25-table-receipt.json SHA f4c847bef8f9d551aad93c1e6336f2d204410ba37a6cd39943434175b4677adb|
|V4 静态|vet/Linux allcmd build/smoke/gofmt各exit0、531源0漂移；最终文档strict另跑、owned-only staged diff归5.2；final-v4-static-checks-receipt.json SHA 12ccd2898ca0b4500e42a1e564ecfb428205f96003b6227b5987295e1d3f00f9|

真正staged Red：candidate-precommit-whitespace-red-and-fix-receipt.json记录exit2/15尾空白，候选未创建，root仅解除自身24文件暂存后修复；不是通过旧diff忽略错误。词法批准sixth-v4-trailing-whitespace-lexical-review.json SHA ceeab4d4539eb4b44cee9f12093ec369d3954cfea4938866d3eb62ca5c2fc9f0。初次V4工具fixture socket路径过长（v4-fixture-diagnostics-attempt1/diagnostic-receipt.json，tests_started=false）单列，不计产品Red或测试失败；短路径重建后两模式完整成功。

V4迁移版本20261009091500执行SHA322a6e0a63a9534b1903ee3341ea2e1292b2335f54981f843c7b4a3448266e46、登记SHA4ecac6cf2f1a232f90f1aa6c87fac8192841d1d75e8ac4db45d21befc895bed7；仅末尾LF差异，外部sixth-v4-migration-query-format-review.json SHA a4b432b267c0e68319cb2776b948d038a2dc526721b5ee3824147f15f8766233。

V4 Writer冻结后candidate/SHA/独验仅由外部lifecycle回执更新，不在同commit自引用SHA。5.2、5.3、5.4仍未完成；最终exact detached独验必须全17包ordinary/race，不以本次3包缩减。真实微信/完整CloudRun/95/25/UI3没有新增通过声明。
