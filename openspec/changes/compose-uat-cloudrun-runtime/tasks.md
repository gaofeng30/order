> 2026-10-10 当前执行约束：用户已明确停止全部测试与回归并删除测试用例；旧测试及 Gate 记录仅作为历史证据，未运行不能记为通过。本轮直接完成 UAT SQL 安装、服务构建部署及小程序预览交付；不等待旧测试门禁。实际数据库已安装 113 个函数并登记 38 条平台执行记录，既有业务数据与账本保持不变。

# UAT 完整运行组合任务

Status: APPROVED_REQUIREMENTS / IMPLEMENTING。gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN。

## 1 范围与配置

- [x] 1.1 冻结 parent b991bd0、12 owned paths 与 disjoint writer/dependencies；Astra v2 合同为权威。结构 Red：正确 change 名称不存在，openspec validate compose-uat-cloudrun-runtime --strict 实际失败，external cloudrun-runtime-root/structure-red.log。
- [x] 1.2 配置 Red：新测试因 LoadUATRuntime 尚不存在而实际编译失败；合成无 native 输入在旧 loader 仍失败。external cloudrun-runtime-root/config-red.log，exit 1。
- [x] 1.3 配置 Green：固定身份、原 Mini 字段、HTTP默认/覆盖、普通 PG 与具名 COS输入；缺失、空值、错身份、禁止材料及 canary 脱敏场景通过。原真实25项经新 loader、普通PG客户端及原支付/核销构造校验通过，网络调用0；未等同完整组合或部署。
- [x] 1.4 Refactor：无需本切片之外的整理；完整 config ordinary/race 各166项（46顶层）、0失败/跳过，包含原 local/prod 相邻测试。同一 loader 测试及原测试保留。

## 2 完整组合

- [ ] 2.1 依赖采用及 full composer Red；缺存储定义保留真实依赖失败，不制作 stub。
  - 已取得行为 Red：原 run 在 ORDER_ENV=uat 进入 native 配置，uat-dispatch-red.log exit 1。既有完整组合编译反例包含尚未采用的 NewUATCOSCredentialProvider / NewSharedPGVersionedService，未重复同条件失败。
  - [x] 2.1a 最终依赖接线：Finance `61df2807938839b27b85248a553ba1f904239c4e` 完整86路径及 Storage `2811cabfb21005330c38abb205ce7f4810974f8b` 独占4文档通过精确 git blob 采用，原R9 owned19保持；`r9-progress/exact-dependency-adoption.json` 逐项hash守护。Storage独验已通过，Finance独验仍待完成。最终采用后的组合 ordinary23项通过，0失败/跳过，见 `root-exact-adoption-composer-ordinary-receipt.json`；不重跑已过Storage25及财务Writer49。
  - [x] 2.1b 新签名 ABI Red→Green：`composer-callsite-red.log` 为真实编译失败；native main/六个原 acceptance 构造及四值 Check 最小适配后，同一 dispatch selector 的 `composer-callsite-green.log` exit0。ABI Red与既有行为 Red分开记录。
- [ ] 2.2 完整 Shared ports、原 HTTP handler/provider/鉴权与五 worker 接入；重点核对订阅 resolver、原子 enqueue、后台 command 与固定图片版本。
  - [x] 2.2a payment 固定 UAT Environment；refund/billing 五字段 Config 同源接线。历史 `source-contract-red.json` → `source-contract-green.json` 仅为源码合同检查；此后实际生产依赖采用与本地组合 Green 另见当前回执。
  - [x] 2.2b 原 ordinary HTTP/provider/Shared session 组合：原登录201/Bearer、仅token hash持久化、注销204/no-store、Mini token不能越过PC权限边界；检查普通PG signin/RPC Bearer及原token hash。五worker数量、四due原limit/lease owner和五循环取消已验证。首轮 `composer-first-run.jsonl` 及最终受影响 ordinary/race 回执分开记录；不等于全部业务HTTP/云链路。
  - [x] 2.2c 最后只读review补强：注销及PC认证RPC各恰好1次，防止固定204/401短路仍通过；必要单项ordinary/race见 `composer-bearer-port-count-ordinary.jsonl` / `composer-bearer-port-count-race.jsonl`，生产源未改，其余已过影响面继承。
- [ ] 2.3 ready成功/漂移/超时和错误脱敏；五循环、取消、HTTP drain 与 lease 交接保留原合同。
  - [x] 2.3a 使用 Finance 实际隔离PG ordinary 113实体/109执行权生成夹具，sha256 `b3e8f5f6162a70e68af475e0190661b323b0f8441822f03b5a07e0c1bfc2e1f2`。原HTTP成功200、精确单项定义/ACL/UID漂移安全503与2秒超时在ordinary通过；UseNumber保留sequence int64范围，避免负例同时改变其他字段。夹具只在test读取、镜像排除，不能替代真实云四值ready。
  - [x] 2.3b 新原Shared四due循环与五worker取消/有界等待通过；小时对账保留原节奏未提前运行，实际业务due/lease交接/云SIGTERM待3.2。
  - [x] 2.3c 插桩时限失败定位与记录完成：原完整HTTP race 两次成功路径时限失败保留为 `FAIL_TIMING`，不是DATA RACE或整套PASS。阶段证据 `readiness-phase-ordinary.log` 为RPC前97.55ms/总106.49ms；`readiness-phase-race.log` 为RPC前2262.47ms且deadline已过期。生产2秒不变，真实Check成功/三漂移的有界10秒测试context race另验正确性，不代替HTTP时限。
  - [x] 2.3d `composer-race-impact-corrected.jsonl`：真实Check成功/三漂移与原登录/worker取消各受影响race通过，共8项（2顶层），0失败/跳过；10秒仅测试context，无生产变更。原HTTP2秒ordinary及race时限失败分别保留，不能把此结果记为完整HTTP race通过。

## 3 镜像与真实运行

- [ ] 3.1 专用镜像/ignore结构 Red→Green；公开11+服务14有效25项、CA/crypto、无秘密进入层或上下文。
  - [x] 3.1a 实际 BuildKit 上下文 Red→Green 已通过：[Red](/Users/vivix/.codex/visualizations/2026/10/06/01a11042-3601-7850-a38f-974660ec7462/cloudrun-build-preflight/red-v3/context-receipt.json)、[Green](/Users/vivix/.codex/visualizations/2026/10/06/01a11042-3601-7850-a38f-974660ec7462/cloudrun-build-preflight/green-v3/context-receipt.json)。334 文件、121 嵌入资产、秘密/无关源排除；ignore 未变直接复用，最终采用后重验变化内容和实际业务构建。
  - [x] 3.1b 固定 runtime 已通过：[receipt](/Users/vivix/.codex/visualizations/2026/10/06/01a11042-3601-7850-a38f-974660ec7462/cloudrun-base-probe/receipt.json)。Linux amd64、UID/GID 65532、150 CA、隔离 stdlib probe exit 0，原业务/网络调用 0；builder 当时仅 metadata，不等同最终业务镜像。
  - [x] 3.1c 实际25项原 loader 局部核对已通过：[receipt](/Users/vivix/.codex/visualizations/2026/10/08/order-uat-status/cloudrun-runtime-root/actual-config-private-probe/actual-original-uat-config-loader-receipt-v1.json)。UAT loader、普通PG构造、原支付与核销材料构造通过且网络调用0；此历史回执未测 subscription parser，后续独立检查见3.1e。完整 provider composer、有效25项最终容器及业务镜像仍待完成，3.1 整项不勾。
  - [x] 3.1d 2026-10-10 首次真实 builder：原固定 amd64 builder 在本机模拟下执行依赖下载时 Go runtime 崩溃 exit2；改为固定同版本官方多架构 index、`$BUILDPLATFORM` 原生构建后，Go1.26.5 linux/arm64、依赖下载及 CGO0/linux/amd64 stdlib 交叉编译 exit0，ELF machine62。[原失败](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/cloudrun-builder-preparation/receipt.json)、[native receipt](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/cloudrun-builder-preparation/native/receipt.json)。官方 index 包含原 amd64 child；Go版本、依赖版本、固定amd64产物及 runtime pin 保持。容器内 go.sum 增加一个既有传递模块 go.mod checksum，源码锁文件未改。业务镜像未构建、应用未启动，不重跑不变 runtime 基础 probe。
  - [x] 3.1e 冻结镜像11项真实公开订阅元组经原 parser 通过：[receipt](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/subscription-real-tuple-probe/receipt.json)。external probe 逐字采用原 wechat_subscription_provider.go，11项各消费一次、全部返回字段和 trial/语言/版本一致；实际构建与执行 exit0、OS禁网、network0，原源/Dockerfile/go.mod/go.sum未改。未重跑历史loader/支付/config及合成测试；完整composer、最终容器25项、真实openid/DB/微信发送均NOT_RUN，不代替部署门。
- [ ] 3.2 真材原 loader及完整启动校验，API 接受、digest/配置读回、CloudRun实际 ready、due worker、版本交接和SIGTERM。
  - [x] 3.2a root独占真实25项材料完整构造一次通过：`r9-progress/actual-tuple-private-receipt.json`，原provider/crypto、live与五worker闭包成功；STS仅本地TLS模拟，外部network/PG/业务/openid/付款0，workers未启动。binary SHA256 `047e494e94c26b04a17fd1cf03241fe44f429b3435ed23e137d2cf52e01d3e50`，14服务项4603B≤5120及镜像11公开项；binding见 `actual-tuple-binary-binding.json`。该次构造后仅测试断言/UseNumber及Finance native账单保存复核变化；后者不触及配置/provider构造源，依该范围继承构造证据不重跑。不是容器25项、云API接受或实际部署。
- [ ] 3.3 日志、费用、告警、备份/隔离恢复/回滚真实回执；外部资产缺失记 BLOCKED_EXTERNAL 并记录续接条件。

## 4 生命周期

- [ ] 4.1 受影响 ordinary/race/static/vet/Linux build/smoke/fmt/strict和秘密扫描。
  - [x] 4.1a 新组合及原runtime/worker相邻 ordinary聚合 `composer-ordinary-corrected.jsonl` 共23项（11顶层），0失败/跳过；新增metadata测试先取得 `composer-ordinary-final.jsonl` 的none正例失败，再精确保留int64得到Green，负例只改目标字段。原166配置/25loader/11parser/62基础probe直接复用不重跑。此前原runtime相邻race通过的9顶层见 `composer-race.jsonl`；其中新完整HTTP ready时限失败另记2.3c。
  - [x] 4.1b vet、Linux CGO0/amd64业务编译（ELF machine62）、fmt、diff内容及strict通过；`composer-vet-final.log`、`composer-linux-build.log`、`composer-strict.log`。业务二进制只编译未运行，不等于固定镜像构建或云发布。秘密/最终上下文检查和全生命周期由最终精确依赖采用后闭合。
- [ ] 4.2 双轴源审、owned-only暂存、Writer Gate；本地候选完整SHA与clean detached包。
- [ ] 4.3 exact SHA独立验证，失败由root修复产生新SHA。
- [ ] 4.4 依赖精确采用、本组合必要验证及独验满足后执行已授权本地候选 UAT 部署；main 集成/push/Prod 不执行。UAT 核心鉴权/RLS、事务、资金隔离、支付退款、核销、恢复及真实核心 UI3 保留；95 场景/25 页面及无部署依赖全量为 DEFERRED_PRE_PROD，未跑不记 PASS。每次重验先记录变化、旧证据失效范围及本轮风险，条件未变证据直接引用。
