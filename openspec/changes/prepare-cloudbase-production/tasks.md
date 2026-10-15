# 生产准备任务账本

- gate_type: W3
- ui_level_target: UI3
- ui_level_actual: NOT_RUN
- base_sha: 531638a8961796a7938072463e31ff15dba466ac
- candidate_sha: NOT_CREATED
- owner/writer: root
- status: IMPLEMENTING

- [x] 读取规则、保护原目录、SSH 核对 main/uat、建立独立 worktree/branch。
  - SSH ls-remote：main 8aef176b69239ab9b952bff6f452108da8ad9476；uat 531638a8961796a7938072463e31ff15dba466ac。rev-list main...uat = 0 / 243。
- [x] Prod 环境信息只读刷新。
  - queryEnv info 显式 EnvId：order-prod-d0g4xnw49c5ada5ce；2026-10-15 RequestId eea77690-9092-40d8-a62a-14515cc373f1；NORMAL、标准版、ap-shanghai、postgres-gy041jr4 SHARED、私有桶候选 6f72-order-prod-d0g4xnw49c5ada5ce-1473835269（权限尚待核验）；到期 2027-04-30 23:59:59，QPS quota 500，自动续费及超额使用原本 true，本轮未改变。
  - CloudRun list 不支持该操作显式 EnvId；当前工具绑定 UAT，其结果不得记为 Prod。改用显式 EnvId 官方管理 API 只读核验。
- [ ] 生产配置、普通身份、COS/材料、回调、formal/release 隔离 Red/Green。
- [ ] 持久异常补偿、30 分钟失败间隔、成功停止、重启/多副本验证。
- [ ] 最终 SQL 生成、117/113 基线核对、定义/ACL/RLS/身份/账本验证。
- [ ] 唯一 Runbook、架构规格与旧入口冲突收敛。
- [ ] PC /pc/ 发布包、缓存、扫码登录和主路径验证。
- [ ] COS 缩略图/存量/新上传/三图导入，体积/耗时实测。
- [ ] 只读套餐/预算/备份/正式域名 TLS/小程序备案与发布资格。
- [ ] Writer、逐文件网页提交读回、最终候选与 detached 独验。
- [ ] 具体生产执行清单及发布顺序、GO/NO-GO。

本轮用户要求必要验证覆盖旧 2026-10-10 停测批次说明。当前已完成的生产写入为普通身份/publish_key、三个已核实扩展及已清理的隔离探测；业务SQL安装、部署、切流、费用开关和真实资金仍 NOT_RUN。无候选、无完整 E2E 结论。下文按时间保留历史证据，旧“未创建身份/缺key/工具绑定UAT/Realtime前置”等记录已由后续读回更新，不作为当前阻断。

当前工具绑定Prod；正式域名已确认order.suianwuye.com。普通登录及HTTP RPC实测通过。记录级补偿SQL生成、目录/ACL门禁、RunOne、HTTP adapter、生产运行组合、并发与本地PG重启测试已通过；有限平台实测已证明数据库提交后实际执行临时云函数，且已清理。正式持久调用凭据、生产部署和完整业务恢复尚未完成。生产业务材料、PC/图片/真机、备份/资格/预算验收以及精确候选独验仍须继续。

## 2026-10-15 首轮本地证据

- 基线（未改业务源码时）：`GOPROXY=off GOTOOLCHAIN=go1.26.5 ORDER_TEST_PG_BIN=/opt/homebrew/opt/postgresql@16/bin go test ./services/api/...` exit 0。包括隔离本机 PG 测试；大量包仍无测试，不推断资金/E2E 覆盖。
- Red：`go test ./services/api/internal/cloudbasepg -run TestProductionEnvironmentBinding -count=1` exit 1，production ordinary 被拒绝 `shared PG unavailable`。
- Green/Refactor：仅增加确切 Prod EnvId allowlist，保留 anon/audience/project 校验及禁重定向。`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/internal/cloudbasepg -count=1` exit 0。覆盖 UAT/Prod 双向混 key、高权 key、未知环境拒绝；无线上身份登录。
- Red：`go test ./services/api/internal/config -run TestCloudRunProductionIsolation -count=1` exit 1，ORDER_ENV 生产被拒绝。
- Green：同命令 exit 0；显式生产 EnvId/桶、独立数值 UID（拒绝 UAT UID）、生产 COS 材料字段，拒绝 UAT 材料/native DSN/管理 key。生产运行组合、STS policy、材料来源、formal、回调校验尚未完成；不能启动生产发布。
- 官方 API 二次核对：DescribeCloudRunEnvs 显式 Prod 返回 UNAVAILABLE（RequestId 346a7293-549d-4b19-a029-4abc6dbae68e）；DescribeEnvBaseInfo 显式 Prod 返回 IsExist=true、Status=normal、PackageType=Trial（RequestId c92b438b-27d1-4b75-881b-3863c3ad9fc7）。与 CloudBase 标准版是不同资源面，尚需查询服务和资格，不能单凭任一字段判定可部署或采购必要。
- 官方资料已核对：https://cloud.tencent.com/document/product/1243/75701 ，https://cloud.tencent.com/document/product/1243/75706 ，https://docs.cloudbase.net/api-reference/webv3-pg/realtime/postgres-cdc 。CDC 需数据库 publication/connector/RLS；尚未证明可靠补偿路线，禁止当作实现完成。

## 2026-10-15 CloudRun 生产组合与 Runbook 切片

- 生产请求现在从 main 进入共用 CloudRun runtime；配置/类型命名改为 CloudRun，不保留 UAT 命名别名。生产环境向微信、支付、核销、退款和账单传递 production，不伪装 UAT。
- Red：`go test ./services/api/internal/config -run 'TestProduction(COS|CloudRun)' -count=1` 拒绝生产桶及生产材料。Green：`go test -race ./services/api/internal/config -count=1` exit 0；隔离假STS HTTP验证只允许指定生产桶images前缀GetObject/PutObject，生产材料读取拒绝UAT命名。
- 回调/通知测试先出现新接口编译缺失，再以缺少校验的实现运行得到3个真实断言失败（UAT回调、trial、不同origin）；补齐后 `go test -race ./services/api/internal/config ./services/api/cmd/order-api -count=1` exit 0。
- 配置要求 ORDER_PUBLIC_ORIGIN 正式HTTPS origin，与支付回调精确匹配；production通知formal，UAT通知trial。新增负例暴露UAT自定义域名/localhost/IP被接受，修复后同组race通过。正式域名尚未核实，未填入真实客户端release配置；没有上线材料或普通身份实际登录证明。
- `GOPROXY=off GOTOOLCHAIN=go1.26.5 ORDER_TEST_PG_BIN=/opt/homebrew/opt/postgresql@16/bin go test ./services/api/...` exit 0（部分未改包使用Go缓存；并非新的线上证明）。最后新增origin拒绝项后重跑config/API/cloudbasepg race。
- Runbook增加12步生产执行表，每步目标、操作、验证、失败处理和回滚条件；纠正117/113、移除制作worker、PC完整包先资源后HTML及唯一/pc/；补齐图片三类场景与字节/耗时记录要求、资金及设备验收边界。UAT运行/部署文档当前入口同步；历史回执保持历史。架构spec及handoff旧路线仍待收敛，Runbook尚非最终交付。
- 未完成：可靠持久异常补偿、SQL最终候选生成与ACL/RLS核验、生产真实身份/材料与域名、客户端release、图片实测、PC登录/业务矩阵、平台资格/预算/备份刷新、候选逐文件网页提交及独验。全部继续，NO-GO。

## 2026-10-15 平台、SQL与PC验证续记

- 用户再次明确：涉及加钱停下核对。未新增资源、启用扩展、改动费用开关或执行资金操作；费用门禁写入生产 Runbook。
- 显式生产 DescribeCloudRunServers：服务 0（57f9d9c7-a1e0-4a70-81d3-c8b12b7a0a06）；DescribeHTTPServiceRoute 仅两个默认域名，无自定义生产域名或 API route（4d9740f9-623b-4b6c-b747-a41fed9490dc）。
- ExecutePGSql 只读生产目录：PG17.11，public 表0、order函数0、publication空、实时connector角色不存在（5f43ced6-21ad-4bf8-aba2-4538d29c1dc3）。管理身份查询不代表普通身份可用。
- DescribeBackupPlans 被CAM拒绝 postgres:DescribeBackupPlans（23c688c4-3d3c-4b7b-bde8-04a6d21c968f）；停止该通道，不推断无备份。续接需资源持有人提供当前计划/恢复证明或最小查询权限。预算告警、COS实际策略、备案及发布资格仍未核实。
- SQL生成器增加纯本地 functions/environment/service-uid/origin 参数；旧命令不支持参数的Red exit1，新单元测试Green。未填未知生产身份/域名，未生成可供生产执行的最终候选SQL。
- 本地隔离PG真实安装目录测试扩为UAT/production两套：117函数实体、113可执行函数，定义/ACL/RLS/readiness及错环境错身份拒绝通过。`ORDER_TEST_PG_BIN=/opt/homebrew/opt/postgresql@16/bin GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/sharedpg -run TestInstalledCatalogPassesReadiness -count=1` exit0；不代表平台安装。
- 架构spec已收敛CloudRun/SharedPG/COS并增加change delta；`openspec validate prepare-cloudbase-production --strict`通过。可靠补偿尚未完成，旧每秒worker仍为阻断。
- PC既有选择测试在精确基线抽出临时目录后实际复现失败：window.addEventListener缺失；测试mock仍使用已弃用listOrders。仅更新测试mock为当前orderPage和事件API，保留选择不重复请求的断言。
- 新备餐分页Red：8项中5项实际失败，缺游标、游标超出页尾、空页带后续、页内乱序、后续页低于上页游标均被错误接受。最小修复仅强化现有分页完整性契约，拒绝部分读取用于全部备好，无UI扩展。无需额外Refactor。
- Green：`node --test apps/web-admin/tests/*.test.cjs tools/tests/*.test.cjs` 16/16通过；使用本地VM和假API，无网络、无经营数据写入。实际页面、扫码、批量操作UI、真机及最终候选独验仍NOT_RUN。

## 2026-10-15 批量备好与通知重试续记

- 上轮分类为进展：分页缺陷修复及16项本地测试、费用门禁已写入权威文件；当前不满足完成或阻断审计。
- PC备餐新增7项隔离VM行为测试，直接调用页面按钮绑定：仅当前餐段制作中且服务端允许MARK_READY参与；确认和请求期间重复点击不重复提交；网络未知重试沿用同订单幂等键；401/403停止剩余订单；过期会话取消确认/后续请求且新视图不展示旧结果。无业务页面代码修改。`node --test apps/web-admin/tests/*.test.cjs tools/tests/*.test.cjs` 23/23通过；仍非浏览器/扫码/真机证明。
- 通知暂时失败间隔Red：SharedService.RunDue通过store边界实际登记04:01而非04:30，未来cutoff测试也失败。仅将共用retryDelay改为30分钟；成功和永久失败不登记重试。`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/internal/subscription -count=1`通过。测试使用假store/provider，只证明服务写入重试时间的契约；不证明数据库持久恢复、多实例或真实微信送达。无需额外重构。
- 支付/退款补偿和每秒扫描仍未完成改造；本次通知间隔修复不得当作可靠补偿完成。正式生产域名、身份/材料、实时唤醒平台验证及必要授权仍需续接。
- 本切片后全量 `GOPROXY=off GOTOOLCHAIN=go1.26.5 ORDER_TEST_PG_BIN=/opt/homebrew/opt/postgresql@16/bin go test ./services/api/...` exit0，SharedPG本次运行5.431s，部分未变包缓存/部分包无测试；`git diff --check`及change严格校验通过。无候选提交、无部署。

## 2026-10-15 退款失败持久重试续记

- 上轮有实际实现及测试进展；本轮核对发现退款SharedPG `order_refund_release_query_v1` 主动查询失败只延迟60秒，原生PG对应路径同样1分钟。支付查询失败仍复用15秒ReconcileInterval，正常等待也登记扫描；后者尚未修复，不能仅改全局间隔宣称满足异常补偿。
- Red：在现有隔离PG真实目录/完整关联夹具中插入CREATE_UNKNOWN且持QUERY租约的退款，按普通authenticated身份完成失败请求，断言失败后30分钟、version只增加一次，实际失败。Green：仅该失败RPC延迟改为1800秒；原生PG失败路径单独使用refundFailureDelay=30分钟，不将正常PROCESSING改成异常。
- 同一测试重放旧owner/version完成请求，无额外版本变化/延后；重启隔离PG后检查next_query_at持久保存，测试通过。此为数据库持久与过期完成防护证据，不是生产多实例唤醒或外部退款验证。
- 测试重启工具首次未将服务日志重定向，导致CombinedOutput管道持有；确认仅本测试PG进程后停止，修正pg_ctl restart为显式日志文件，再运行通过。无生产连接、无迁移安装、无资金操作。
- `GOPROXY=off GOTOOLCHAIN=go1.26.5 ORDER_TEST_PG_BIN=/opt/homebrew/opt/postgresql@16/bin go test ./services/api/sharedpg -run TestTimeDerivedOrdersWithoutProductionWorker -count=1` exit0，1.820s；`git diff --check`、change严格校验通过。函数实体数量不变，但定义已变，需最终候选重新生成/核对；历史安装收据不能复用。

## 2026-10-15 支付查询失败补偿续记

- 上轮为进展：退款失败重试SQL与真实本地PG重启验证完成。当前仍有代码与平台缺口，不满足完成或阻断审计。
- 支付queryOne新增真实服务入口+假RPC/provider契约测试。Red三例均失败：截止前查询错误、过期关闭失败均只等待15秒；过期关闭成功仍调用第二次完成RPC登记重试。
- 最小修复：查询/观察写入失败改用固定30分钟failedReconcileInterval；关闭交易并完成数据库关闭登记成功即结束该异常处理，不再releaseQuery登记重试。共用原生PG路径保持一致；不修改正常未支付等待的ReconcileInterval，后者及每秒扫描仍待可靠调度改造。
- Green：`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/internal/paymentorder -count=1` exit0。测试证明RPC重试时间与停止调用行为，不证明平台持久化、资金结果、重启和多实例端到端。
- 对齐架构与delta规格：60秒队列告警按已到期任务的超期时间计算，明确30分钟退避中未到期任务不算超期；不改阈值或扩容授权。
- 正式域名仍无实际确认，已异步请求用户填写，依赖该值的客户端release/回调最终绑定保持未完成；其他本地工作继续。新增费用仍须先确认。

## 2026-10-15 13:43 CST 平台能力与当前阻断刷新

- 上轮支付失败补偿实现及本地验证为进展。此次仅官方文档与资源只读查询，未建立实时连接、未启用publication/connector、未部署或恢复资源。
- 官方实时快速开始确认浏览器/Node/小程序可用app.realtime，Node需ws；默认约270秒双连接热切，Auth事件同步token。文档没有证明离线事件重放或任务定时可靠性，因此必须在订阅确认后从持久任务表重读，并在断连重连/进程启动执行相同恢复；CDC只能作为唤醒信号，不能作为唯一账本。来源：https://docs.cloudbase.net/api-reference/webv3-pg/realtime/quick-start 、https://docs.cloudbase.net/api-reference/webv3-pg/realtime/realtime 。尚未选择未经平台实测的生产实现。
- Prod usage显式EnvId读回：账期2026-09-30 14:01:13至2026-10-31 23:59:59；COS历史8次写请求对应0.01资源点、APIInvocation历史89次对应0.89资源点，列出的其他模块本次返回0。RequestId f9cb10a8-802a-46aa-9f9f-59644027763b，账期RequestId 3ec43f52-6274-44c6-9f52-934d609ee6e6。资源点不是人民币，不据此推断套餐剩余、实时权益或未来零费用；预算告警仍未确认。
- UAT只读SQL能力检查返回原文：`instance status must be Running to execute SQL, current: Isolated`。查询只有目录SELECT，无写入。未通过换协议、普通身份或生产资源尝试恢复/绕过。
- 随即只读Env info确认：环境NORMAL，个人版，postgres-o7vmhmxa SHARED Status=1；套餐到期2026-10-31 23:59:59、EnableOverrun=false、IsAutoRenew=false、UsageStatus=normal。RequestId cfd20e41-eec1-4724-b1c6-30d4e97220d1。环境总览与数据库执行状态不同层，不能用NORMAL覆盖Isolated，也不能臆断欠费原因。官方搜索未给出该PG状态的明确原因/无费用恢复路径。
- UAT平台数据库验证通道现为BLOCKED_EXTERNAL；先由资源持有人/控制台核实PG实例隔离原因及恢复报价，涉及加钱必须用户校验后再执行。未恢复、未续费、未启用超额开关；不向生产写测试表替代UAT。其他本地适配继续，整体仍NO-GO。

## 2026-10-15 PC发布包与域名推荐

- 新增tools/pc-release-package.cjs及6项包完整性测试；Red为工具缺失，Green覆盖缺资源、旧base、越界/外链引用、CSS资源、符号链接、已有输出保护。CLI真实dirty发布模式exit1且无输出；draft生成27文件、354235字节、HTML最后、candidateSHA=null。
- `node --test apps/web-admin/tests/*.test.cjs tools/tests/*.test.cjs` 29/29通过。草稿路径 /var/folders/41/f3b8tfgd7ygbqwp8h_lbd3mr0000gn/T/order-pc-release-check-35few2ff/package/manifest.json 。仅临时草稿，不是最终候选。
- Chrome通过CUA打开127.0.0.1:8768/pc/，实际AX读回侧栏、备餐/订单等入口、未登录及“后台服务不可用/请求失败404”。本地静态服务无API代理，未连接UAT/Prod；只证明静态壳实际加载，未证明扫码或业务主路径。首次header policy初始化失败，按工具提示重试后成功；赋值绑定未生效，按实际tab id重新绑定读回。
- 用户要求自行调查域名后给推荐。公开DNS核对suianwuye.com NS及uat子域CloudBase CNAME；order.suianwuye.com A/AAAA均NXDOMAIN。TLS握手证实UAT证书仅覆盖uat，2026-10-13至2027-01-11有效。DNSPod DescribeDomain CAM拒绝，RequestId b361f37f-3343-4c84-baf3-56b1ff9a67a3；停止权限通道，未更换身份绕过。
- 推荐order.suianwuye.com（/pc/与/api/v1同源），写入Runbook待确认队列；不将NXDOMAIN当成管理权/备案证明，不将推荐当成确认。不购买、不绑定、不填release；新增费用仍先确认。

## 2026-10-15 接续与审计

- 上轮实际完成PC包工具/29项测试/本地Chrome渲染及域名调查，属于进展。
- 重新读取simplify-background-order-work决策：通知首次发送及时，正常待支付不循环查单，只有持久异常待办驱动30分钟重试。保留这些约束，不将每秒扫描降频冒充完成。
- CUA打开腾讯云控制台真实跳转登录页，登录未完成。已保留该页供本人登录；不自动恢复/续费/开超额计费。此为隔离原因查询的当前外部接续条件，无新平台写入。
- 新增readiness-review.md逐条核对全部8项、精确SHA缺口、具体变更顺序和外部续接；Runbook唯一执行入口不变。修正Runbook把10月15日读回链接到10月12日历史资源收据的证据混用。

## 2026-10-15 控制台资源耗尽证据

- 上轮完成逐项审计；本轮正常微信快捷登录进入项目所属账号，未更换账号绕过此前DNSPod/备份CAM拒绝，也未访问这些被拒绝的功能。无需再等待用户登录。
- 新版UAT套餐用量页面实读：个人版有效至2026-10-31，套餐资源点4万/4万，明确“已经超额”；资源包0点；页面“按量计费”累计640.34点，按量付费开关关闭。总消耗约4.06万点。此处是资源点，不是人民币报价或已付款证明。
- 构成：PostgreSQL约1.82万点（53.2核时），云托管6108.58点（CPU22.12核时、内存44.25GB时、外网4.34GB），环境调用约1.63万点（云开发API163.05万次、HTTP6235次），云存储8.5点。各项按页面标明小时/天更新，取整可能不完全相加；尚不能仅据累计次数将费用全部归因于扫描worker。
- 点击SQL型数据库只读管理入口返回 ResourceUnavailable：PG环境当前状态res_stopped，无法处理请求，请检查资源用尽/套餐过期/账号欠费。RequestId 5b6ae4be-1ccf-42d4-b5a7-4dec3d7ab47c。结合未到期且资源点耗尽，确认资源耗尽是当前明确存在的恢复约束；不声称排除了账户其他因素。
- 未点击购买、续费、升级、按量付费或资源恢复。遵循用户“涉及加钱停下来校验”，付费恢复等待具体报价与用户批准；整体NO-GO，生产未写入。

## 2026-10-15 UAT最小资源包报价（未下单）

- 上轮为进展：控制台确认资源耗尽；本轮只读进入加购页面，未点击“立即购买”。
- 当前报价：上海order-uat-d1gtzpzj50615c300，1万资源点×1个，人民币10.00元，有效期1年；环境自身仍于2026-10-31到期，资源包不延长基础套餐。仅限该环境，不可转移；页面提示使用后余量不支持退订、到期作废。
- 报价页：https://buy.cloud.tencent.com/CloudBase?buyType=resPack&envId=order-uat-d1gtzpzj50615c300&region=ap-shanghai 。截图/tmp/order-uat-resource-quote-20261015.jpg。该报价不保证消除其他账户限制，也不保证覆盖所有剩余验收消耗。
- 建议单次10元作为恢复和有界验证预算；不开按量付费、不升级、不自动续费。须用户明确确认后才能购买；恢复后先核实状态和现有扫描消耗，再开展限定测试。尚未取得新增费用批准。

## 2026-10-15 Prod控制台用量与运维只读补核

- 上轮取得UAT10元/1万点报价，属于进展；费用批准仍未收到，未购买。
- Prod标准版页面当前账期2026-09-30至10-31、额度33万点、消耗17.05点，其中PG16.15点（136核秒、3.15GB时）、存储0.01点、环境调用0.89点；按量累计0点。页面当前自动续费已开、按量付费已开，均未变更。基础套餐有效至2027-04-30。此前API用量与当前界面为不同时刻/更新粒度，不把旧值覆盖当前。
- 套餐能力项列出“数据回档14天”，仅是权益说明；没有访问此前CAM拒绝的备份详情，不将套餐文字作为备份计划/恢复点/成功恢复证据。
- 备案管理仅列出order-prod“可备案”，未提供已备案证明；小程序认证页面展示扫码授权/全托管认证入口，没有当前微信备案/发布资格结果。未点击认证、上传密钥或扩大授权。
- Prod日志管理页面显示“开启日志”及服务条款/日志付费模式未勾选；未开启。告警策略→平台策略列表0条、近7天告警0。这个结果仅覆盖该平台策略列表，不证明账户可观测平台或费用中心没有其他规则，也不证明告警已送达。预算告警仍未核实。
- 仅页面读回，无资源、SQL、计费、安全配置或告警写入；UAT加购确认继续待用户回复。

## 2026-10-15 费用中心预算核验

- 上轮生产套餐和平台策略只读核查为进展。本轮同一项目账号费用中心主页显示可用余额0.00元、欠费0.00元、余额预警关闭、待支付订单0；未充值、未设置自动充值或余额预警。
- 成本管理→预算管理（https://console.cloud.tencent.com/expense/cost/budget）列表显示“您还尚未创建预算”，共0条；无搜索条件。账户预算告警缺失已证实，不再只标“未核实”。这不代表所有产品通知均不存在。
- Prod按量开关已开但账户余额为0、预算0条；上线前必须确定预算上限、接收人与余额责任，再按明确授权配置并验证到人。未将现有开关视为新增支出授权。
- UAT10元资源包仍未获费用确认，未下单。真实平台恢复、可靠调度平台验证、精确候选/真机与正式域名仍未完成，整体NO-GO。

## 2026-10-15 按用户纠正恢复Prod主线

- 用户明确继续Prod准备，UAT恢复/10元加购不再是整体前置；原BLOCKED判断过度扩大依赖，撤销该推进限制，不自动购买。真实平台验证选择需准备具体范围及费用后确认，不用生产经营数据充当测试。
- 支付创建未知结果补偿Red：实际RPC next为04:00:15，要求04:30。修复SharedPG与原生PG失败路径为30分钟，支付包race通过；成功路径暂未改变，每秒扫描及正常等待清理仍未完成，不宣称可靠补偿闭环。
- 用户明确确认正式域名order.suianwuye.com（本地配置/发布清单，不含DNS/证书购买/切流）。release-red：空release及接受UAT地址两例失败；release-green：正式origin绑定并限定release只接受该origin，develop/trial拒绝Prod。本地Node全组38/38通过，无网络、无微信平台或真机操作。
- release-refactor：仅增确切Prod origin判断，无额外重构。新增miniprogram-gates.json，要求UI3主路径与相邻回归；实际UI3仍NOT_RUN，不形成候选。两项formal模板仍未核实。

## 2026-10-15 Prod普通身份创建回执

- [x] 用户了解用途后明确“那你去创建吧”，授权创建Prod专用普通身份。
- [x] 创建前Name查询0条，RequestId `77f8ecd8-c8fa-4e0a-a40e-53af8ae916aa`。
- [x] CreateUser返回UID `2110616903056039938`，RequestId `9be02697-d68a-4ecd-b59a-25a31aa52b13`。
- [x] DescribeUserList读回1条：环境 `order-prod-d0g4xnw49c5ada5ce`，Name `order-prod-api`，Type `internalUser`，状态 `ACTIVE`，UID一致；RequestId `9c05eca3-50a0-4ce0-83a3-63917987be78`。
- [x] 独立随机32字符密码保存于仓库外 `/Users/vivix/.config/order/production/order-prod-api.json`，目录0700、文件0600；材料附创建/读回回执，密码不入仓库。
- [ ] 普通身份登录、RPC权限及跨环境拒绝尚未验证。本回执不证明SQL安装或部署成功；未授业务表权限、未安装SQL、未切流、未购买。

## 2026-10-15 正常支付退出补偿与构建验证

- [x] 成功创建预支付：Go RPC测试Red为仍排队15秒，PG实测Red为正常支付进入补偿；SharedPG完成函数及原生PG成功路径改为NULL，失败仍30分钟。
- [x] 主动查单确认NOTPAY：隔离PG真实函数Red为`confirmed NOTPAY has compensation scheduled`；移除SharedPG/原生PG正常等待重新排队，保留READY应用工作及其他实例持有租约保护。Green：paymentorder与sharedpg完整包通过（PG环境变量指向本机PostgreSQL16），不连接线上或微信。
- [x] 无额外重构；每秒扫描、退款正常PROCESSING和可靠唤醒仍未完成，不能登记补偿闭环PASS。
- [x] 专用Dockerfile.production移除UAT订阅ENV，配default-deny忽略清单。本机legacy builder跨架构多阶段未通过；改用本地Go1.26.5 CGO=0 linux/amd64编译加同一runtime stage，本地镜像构建成功，非root65532、ENV仅PATH/SSL_CERT_FILE；network none缺配置启动按预期拒绝。未推镜像/未部署，且后续源码修改已使该构建失效，最终候选必须重构建。
- [x] 后端正式origin精确绑定已确认order.suianwuye.com；其他域名及8443端口Red后修复，config/cmd race通过。
- [x] Prod普通身份读回后生成本地functions SQL草稿（service UID2110616903056039938），非最终候选安装证明。
- [ ] Prod普通身份实际登录尚缺可用publish_key：DescribeApiKeyList仅查publish_key返回Total0，RequestId `02054747-fd26-4d45-9c23-e823613993fc`；未改用高权api_key、未复制UAT key、未创建新key。

## 2026-10-15 Prod匿名key与普通身份登录回执

- 用户明确批准创建publish_key并验证登录。CreateApiKey仅使用KeyType=publish_key，未创建管理员api_key；KeyId `3znAvPKBTAC-OiW38En_lw`，创建RequestId `95afe11a-e2d4-4732-98fb-6484a8db8b20`。
- key的role=anon，aud/project_id均为Prod环境；已存入同一仓库外0600材料文件。
- 2026-10-15 14:25:04北京时间，真实Prod auth/v1/signin验证通过：返回sub与UID `2110616903056039938`一致、role=authenticated、aud/project_id均为 `order-prod-d0g4xnw49c5ada5ce`、Bearer且有效期7200秒；RequestId `bd11369a-c087-4887-a394-1a39b0034074`。核对了实际响应身份，不仅HTTP状态；访问令牌未落盘或写入文档。
- 此前“缺publish_key/登录未验证”已解除；数据库RPC、ACL/RLS、业务与真机仍未验收。无SQL安装、业务授权、切流、采购或资金操作。

## 2026-10-15 恢复完整上线Goal与退款补偿

- get_goal读回新的完整Prod上线目标与status=active；必要生产配置/最小权限/审核SQL/部署/DNS-TLS/切流/main集成已明确授权，保留费用与真实资金确认边界。旧逐项生产写确认说明不再作为继续本任务的障碍。
- SSH刷新main=`8aef176b69239ab9b952bff6f452108da8ad9476`、uat=`531638a8961796a7938072463e31ff15dba466ac`；本轮未推送/提交。
- [x] 退款创建请求失败：隔离PG实际finish_create函数Red立即排队；修复SharedPG与原生PG为失败后30分钟；重复旧claim完成不递增版本/延后重试。
- [x] 正常PROCESSING创建和查询：Red分别证明创建立即排队、查询排1分钟后任务。改为next_query_at=NULL；重复观察释放自身租约且停止重试。创建直接返回SUCCESS时仍保留尚未应用事实的后续工作，不将其误判为已落账。
- [x] due列表和claim不把无next_query_at且无租约的PROCESSING视为异常；保留过期CREATE/QUERY租约恢复，隔离PG覆盖正常排除和崩溃租约再选出。
- [x] Green：`ORDER_TEST_PG_BIN=/opt/homebrew/opt/postgresql@16/bin GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/refund ./services/api/sharedpg -count=1`，sharedpg通过12.197s；refund包无测试文件，不能算独立native PG行为证明。测试数据库退出清理；未调用真实支付或退款。
- [x] 创建返回身份/金额不匹配的永久人工队列不再自动排队；真实隔离PG Red `manual mismatch requeued automatically` 后修复为NULL，Green通过。可靠唤醒、断连补读、多实例、通知首次及时发送及平台验收仍未完成。

## 2026-10-15 Prod CDC有界平台探测（已清理）

- 本机临时目录`/tmp/order-prod-realtime-probe`安装官方npm `@cloudbase/js-sdk@3.10.2` 与ws，未加入生产构建；使用新Prod普通账号和Prod anon key。
- 只读确认Prod PG17.11、public表0/publication0/connector0，RequestId `9eb977e3-3ab4-473c-bebe-b33578126352`。
- 在新的完整Goal授权范围创建专用`public.order_readiness_cdc_probe`和仅含该表的`cloudbase_realtime` publication；SELECT仅授authenticated，RLS限定Prod服务UID，无anon权限、无业务数据。创建RequestId `19abd871-04ff-4a9d-94ba-074904f734d5`。
- 实测官方SDK登录成功；订阅使用wait=true/15秒确认，35秒硬截止。首次CHANNEL_ERROR缺底层原因，后续仅增加诊断，确认cause为`Unexpected server response: 503`，属于WebSocket握手失败，未收到SUBSCRIBED/事件。不把它归因于RLS、套餐不足或整个PG不可用；同时管理SQL可读，connector仍0。
- 三次探测均已终止，未插入测试行/调用provider。确认publication只包含本任务表后清理publication+表，RequestId `118640eb-5ff7-4156-b034-91ef58e9e301`；读回public表0/publication0/connector0，RequestId `f1087e9a-480a-4dc6-a13f-d96472226050`。未手工创建connector或删除复制槽。
- 无费用开关、采购或资源升级；官方CDC本次目标环境握手未通过。继续控制台/官方能力排查和其他本地工作，不做无新证据的重复连接，不将此单一路线失败当整体Goal阻断。

## 2026-10-15 调用方式自查与UAT对照（不提交工单）

- 用户要求先核查自身调用及官方用法，未批准工单；工单草稿保持未发送，不能作为已核实的平台故障结论。
- 固定UAT基线531638a的cloudbasepg使用普通身份HTTP `/v1/rdb/rest/rpc/`；三个worker仍为time.Second。源码及回执没有Realtime成功证据；不能以UAT业务API曾通过推断Realtime在UAT可用。本次只读UAT SQL返回Isolated，不充值、不反复重试，也不将恢复UAT设为Prod前置。
- 对照官方quick-start及已安装SDK3.10.2源码cloudbase.ts：实际Prod握手host正确，path=/v1/realtime/websocket，vsn=1.0.0，apikey实为与刚登录完全相等的access token，aud=Prod、role=authenticated、sub等于本次服务UID。去掉CDC/table/select/wait配置，仅订阅空频道仍握手503；故本次不能归因于表RLS或CDC参数。根因尚未查明，不推断套餐/平台整体不可用。
- 同一身份访问Prod `/v1/rdb/rest/` 返回200和Swagger2.0（仅根路径），RequestId df344b1a-2643-407f-9e3d-05d89227c46f；这仅验证HTTP接口响应。
- 隔离真实RPC探测：创建只读SECURITY INVOKER函数order_readiness_identity_probe，仅authenticated有EXECUTE，PUBLIC/anon撤权，固定search_path；创建RequestId a1e87ffd-096a-4a5f-a23b-2f7aa5c92092。同款HTTP POST RPC实际返回database=postgres-gy041jr4、role=authenticated、uid匹配Prod，RequestId aa5ca6a5-671e-410a-9498-be9e473fe4c5。anon被401/DATABASE_42501拒绝，RequestId f56af3af-d057-46ba-954c-ec566b446961。未读写经营数据、未装业务函数。
- 探测函数已删除并发出schema reload，RequestId e36ecaa3-1732-42c9-ad90-4ad6b2c4f617；管理读回public函数/表/publication全部0，RequestId 297a94f3-000b-42d2-8542-02a8a6e9d144。临时Node进程已终止，无生产定时器遗留。
- 结论：Prod普通登录和HTTP RPC接入已实测；可靠补偿的Realtime只是本轮探索依赖，既非用户固定技术路线，也非UAT已有成功组件。不得把该依赖失败当成整个数据库/上线目标阻断。继续在既定CloudRun+SharedPG范围核对最小可靠唤醒实现。
- 本地退款/支付/通知与真实隔离PG回归通过；跨环境access token拒绝新增Red/Green，cloudbasepg/config/order-api race通过。以上不是线上业务、资金或真机PASS。

## 2026-10-15 官方SQL调用云函数能力补查（进行中）

- 此前仅查pg_net/http不够完整。官方 `/database/postgresql/tencentdb-scf` 明确有tencentdb_scf：限制本环境函数路径、加密配置key、事务提交后唤醒异步worker、回滚不消费。Prod pg_available_extensions实际有tencentdb_scf1.1/pg_cron1.6，RequestId 88cdf8b1-9cff-41d4-b1b0-6b90f8664138；该能力不依赖Realtime。
- 现有完整Goal授权下安装pgcrypto1.3和tencentdb_scf1.1，仅检查schema/ACL/持久性，RequestId c0bd4de6-4bc1-439a-8898-324845d17ecd；读回1db8a659-0e49-472b-ab31-9e4b86607b78。各函数仅安装者EXECUTE，配置表仅安装者权限；queue/response为普通持久表（relpersistence=p），RequestId 69ca1f96-9a38-4ea3-9b9e-df3bab5d70a8。未配置key/env、未调用云函数，config和queue均0。
- 安装pg_cron1.6成功，RequestId f084bd1a-8451-4c82-a6eb-8392df241545，初始jobs0（b7a30452-8460-4bc2-9ba7-2c6a05d091de）。安装成功不是后台调度通过。shared_preload_libraries读取被42501拒绝，不绕过、不修改实例参数或重启共享实例。
- 已创建独立order_readiness_probe schema/cron_receipt及order-readiness-once任务，5秒执行一次，任务仅写自有回执并自行unschedule；RequestId d728c8ef-0267-4d82-9f38-07eea7a5d1e0，jobid1。须读回执行、自停及清理；不含网络调用、经营表或资金。
- 还须验证事务回滚、提交后唤醒、执行失败/停机恢复、仅异常挂调度/成功停、最小身份权限。未创建付费资源、未开启日志或新的计费开关。云函数试验尚未部署；先核对现有额度和可控费用边界。


### 已完成的数据库侧唤醒实测与清理

- pg_cron测试任务2026-10-15 14:55:15北京时间实际执行，run status=succeeded，回执仅1条，执行后jobs=0；RequestId 96251896-23bd-4e84-a698-a0ebd230ff29。回滚注册的jobid2既无job也无run/回执，5c755bda-1925-413c-a060-6ff1f926e89a / 4ff7cf2e-2667-47dd-9e5c-4f255443c8fc。不是每秒业务扫描，也不是内存timer恢复。
- CloudBase MCP当前绑定已切至Prod，后续非显式EnvId工具必须复核目标。Prod云函数列表0，RequestId 1d495b96-11cd-4f4e-843d-7227cafa8d62。
- tencentdb_scf隔离探测使用限定Prod UID、固定函数路径、固定search_path且撤PUBLIC/anon EXECUTE的临时SECURITY DEFINER RPC。只向已确认不存在的order-readiness-nonexistent-20261015发起请求，没有创建或执行云函数、没有经营/资金调用。临时调用凭据为Prod普通登录短时token，经HTTP RPC参数进入扩展加密配置；无管理员key，未记录token。
- 回滚分支取得队列id1后回滚子事务：HTTP RequestId f0b60467-9fea-487d-8d54-92517499662f，管理读回queue/response/config均0（81e32c62-ad41-44c8-8ef4-1b1e25c6e837）。提交分支id2：9afcb38c-275e-40ce-8171-d16302f61211；后台真实消费队列，返回404/FUNCTIONS_NOT_FOUND、timed_out=false、error=null、queue=0，e33b204c-3899-4533-a855-9f0e78430832。404是本次不存在目标的预期结果，证明提交后后台实际出站，不代表真实业务函数通过。
- 已删除临时RPC/凭据配置/响应及测试schema/table（eee81160-d0bc-4e67-9f69-dcaf0eca731e）。最终读回jobs/queue/config/responses/probe_functions均0、probe_schema=null（bdd42307-aca0-4d38-8b16-aecc5657ceea）。保留已成功安装的pgcrypto1.3、pg_cron1.6、tencentdb_scf1.1，不重复安装；业务SQL仍未安装，扩展函数不得混入业务117/113统计。
- 这组新证据支持继续采用现有SharedPG数据库调度+事务队列验证可靠补偿，无需依赖Realtime或提交工单。尚未证明：函数实际执行、传输失败重试、崩溃/多实例全链路及异常到期30分钟/成功撤销的最终实现。下一步按这些边界完成实现与验收；费用和资金边界仍有效。

- 后续全量本地回归：`GOPROXY=off GOTOOLCHAIN=go1.26.5 ORDER_TEST_PG_BIN=/opt/homebrew/opt/postgresql@16/bin go test ./services/api/... -count=1`通过，包含sharedpg真实本地PG8.938s、config、paymentorder、subscription、billing等现有测试。无测试文件的包仅编译，不记业务覆盖；无任何真实支付/退款调用。

## 2026-10-15 持久唤醒执行链代码切片

- 本轮续接核对worktree/current源码，上一轮为有效进展（真实pg_cron/tencentdb_scf证据改变实施路线），不复做生产探测。
- 新owned paths：deploy/functions/order-compensation-wake/index.js、services/api/internal/compensation/、tools/tests/compensation-wake.test.cjs。唯一writer仍root。
- 转发器Red：模块不存在；Green固定Prod HTTPS endpoint、一次POST、禁止重定向、有限超时。第二轮Red：非法event触发请求/抛出含凭据诊断；Green严格只接收64位小写hex token，无URL/kind等额外参数，传输异常返回accepted=false、不循环重试、不输出原始错误。Node 3测试通过。
- 后端入口Red：Handler不存在；Green要求唯一Bearer capability、拒绝非法格式、执行有20秒deadline、只在执行成功后204，未知token401、存储/执行错误503且无响应诊断。新增Red GET错误通过后修复方法/查询串/请求体/重复认证头拒绝，race通过。
- Executor Red→Green：必须先取得Store原子授权才能选择payment/refund/subscription runner；未知类型拒绝；执行结果须持久Finish确认，Finish失败不得返回成功。runner allowlist在组装时复制；底层错误不向HTTP暴露。compensation包race通过1.539s。
- 上述只证明转发与执行编排，不证明数据库CAS、重放保护或恢复。入口尚未加入runtime registrars，云函数未部署，Store真实SQL adapter与事务调度/触发器尚未完成；严禁将unit fake当真实平台完整补偿。
- 下一步以真实隔离PG测试补齐调度表、token一次消费、成功撤销/失败30分钟、事务回滚及两个并发连接；接入现有业务表变化与初次通知，再替换每秒worker。数据库状态是唯一恢复依据，不能改成内存timer。

### 真实隔离PG状态机与记录级隔离

- 新增services/api/sharedpg/compensation.sql及CompensationSQL生成函数、compensation_integration_test.go。尚未合入正式InstallSQL/CLI/运行时，也未安装Prod业务调度SQL；后续必须完成catalog/ACL校验及最终迁移登记。
- 首个测试Red：生成函数不存在；Green：真实本地PG事务验证空闲、正常等待、回滚均无cron/请求，首次通知发出持久唤醒，凭证重复消费被拒，成功撤销cron。测试cron/SCF适配器只记录事务调用，不能替代平台结果。
- 失败完成函数Red不存在后补齐；Green失败完成时间起至少30分钟不再次发送。新增两个独立psql进程并发消费同一token，精确一CLAIMED/一REJECTED；本地PG immediate stop/start后已消费但未完成状态和任务仍保留，过期恢复产生新token，旧token拒绝。
- 发现整类任务共享重试闸门会阻塞新通知首次发送；真实PG新测试Red=`old failure delayed new initial notification`。修正为(kind,entity_id)记录级私有状态、行触发器和独立任务名；旧异常继续等待，新通知立即单独唤醒。重跑同一真实PG测试Green1.724s。
- Executor同步改为只接收数据库Claim返回的具体uint64记录ID，runner不得扫同类全部到期待办。新目标ID测试先Red，后实现Target及记录级runner接口，race通过。后续必须提供三个业务模块RunOne并在SQL条件中保持租约和版本CAS。
- 当前仍无正式SQL安装、云函数部署、worker替换或发布。组装旧每秒worker仍在，所以不能以这些新增测试登记生产补偿完成。

### 记录级业务执行与 HTTP RPC 接入

- 支付、退款新增 RunOne，复用原租约/版本 CAS 流程；测试只允许指定 ID，禁止退回整类 due 扫描。通知新增 claimOne 与 RunOne；定向 SQL 从 subscription.sql 的原 claim 定义生成，只收窄 SELECT 条件，保留原验证和租约语义。真实 PG 测试先因函数不存在失败，生成后通过，证明领取 ID 5 不误领较早的 ID 4。
- 退款 READY 在首次 provider 领取前崩溃的恢复测试先失败，refresh 增加该持久状态的立即唤醒后通过。正常 PROCESSING 且 next_query/lease 均空仍不登记补偿。
- 新增 compensation.SharedPGStore：Claim 只接受已确认 outcome、允许类型和规范正 uint64 ID；Finish 必须返回 FINISHED，底层诊断不外泄。Red 为 adapter 未定义，Green/race 通过。
- 三个新 RPC 的真实 HTTP 客户端边界测试先失败（客户端白名单漏项，网络调用次数 0），加入 order_compensation_claim_v1 / order_compensation_finish_v1 / order_subscription_claim_one_v1 后通过。不是生产网关或平台故障。
- 全量本地 `GOPROXY=off GOTOOLCHAIN=go1.26.5 ORDER_TEST_PG_BIN=/opt/homebrew/opt/postgresql@16/bin go test ./services/api/... -count=1` 通过，sharedpg 9.815s；无测试文件包仅编译。转发器 Node 3/3 通过，cloudbasepg/compensation race 通过，git diff --check 通过。测试未连接真实支付、退款或经营数据。
- 仍待完成正式 InstallSQL / catalog / ACL 接入及运行时替换，现有 worker 尚未删除，未发布候选或生产业务版本。本轮没有生产写入。
- 官方能力续查：`/cloud-function/security-rules` 明确仅支持四种函数级规则，不支持按 UID 表达式；细粒度须函数内部校验。`/v1/functions/{name}` 支持 Bearer 身份，受安全规则与认证双重控制。不能用已过期探测 token 作为持久配置；正式最小凭据及费用边界仍需核对。tencentdb_scf 官方文档说明响应有 TTL 清理，不能把响应缓存当业务恢复账本。

### 已安装平台扩展与首次迁移，以及传输失败计时

- 新增 production 隔离安装用例：先安装 pgcrypto 1.3，再运行现有首次安装脚本。Red 为 shared_install_public_not_empty，确认不是数据库不可用，而是脚本与已安装平台依赖冲突。
- 最小修复仅允许当前安装身份持有、public schema 中版本精确 1.3 的 pgcrypto 扩展本体及 pg_depend 登记属于它的函数；不按函数名前缀放行、不允许其他扩展或其他 owner。反例 pgcrypto_unreviewed 仍拒绝，拒绝后 schema_migrations 不存在，无部分迁移；移除反例后完整安装及原 readiness 通过。
- Prod 只读核对 RequestId bfa63232-88fc-4c4b-bf4f-4a3b8b9c2964：pgcrypto 1.3/public、pg_cron 1.6/pg_catalog、tencentdb_scf 1.1/tencentdb_scf 均由当前安装身份 cloudbase_postgres_postgres_gy041jr4 持有。未执行安装、重装或业务写入。
- 新增传输失败计时用例先 Red：请求发起时就开始30分钟闸门，不能保证从传输失败之后再等30分钟。修复为记录扩展 request_id；发现响应已完成而业务 Finish 尚未确认时，撤销 capability，从观察到失败时起持久等待30分钟。正常 Finish 清除该 request_id 或已移除唤醒，旧响应不会影响新工作。不依赖平台错误正文格式、不暴露响应材料。
- `TestDurableCompensationCapabilities|TestInstalledCatalogPassesReadiness` 真实隔离 PG 回归通过5.689s，diff检查通过。扩展响应在本地为事务桩，仅证明算法；真实平台传输故障与运行时接入仍待验收。tencentdb_scf_result 的真实定义只读核对为官方 C 扩展函数，RequestId 7a611d5a-6772-4728-9b4d-34c356d28c82。

### 目录门禁、生产运行接入与真实云函数执行

- 新增补偿目录只读 RPC 和 CheckCompensation：核对7个函数实体的定义摘要/签名/owner/搜索路径/ACL、私有表列/约束/索引、三个行触发器及平台扩展版本。普通身份不获得私有schema或cron/tencentdb_scf管理权限。授予cron schema USAGE、改变私有表RLS、加入未审查业务触发器的反例逐个Red后修复并Green；其他授权/约束/默认值/搜索路径漂移也拒绝。
- Prod只读有效权限：cron schedule/unschedule有默认EXECUTE，但anon/authenticated均无cron schema USAGE；tencentdb_scf配置/发送既无schema USAGE也无EXECUTE。RequestId 8d173474-6efb-4ea8-87da-0f64b84b1160。门禁保持schema不可访问，不误把函数单项EXECUTE当有效可调用权限。
- 完整业务schema与补偿安装在隔离PG合并验证通过。实际本地传输桩不能通过平台扩展门禁；另以明确的扩展目录fixture验证组合门禁正例，该fixture不是平台实测。未安装补偿时生产ready先Red（错误放行），接入新门禁后拒绝。
- 生产CloudRun组合接入受限wake入口、只运行目标记录RunOne，删除生产分支三类每秒扫描，仅保留账单调度。UAT行为未变。新增runtime路由测试先Red缺少组装，后3类精确ID/其他runner不触发/Finish确认Green，相关race通过。Native开发入口仍保留原行为；main将production明确导向CloudRun。
- CLI增加独立 --compensation --service-uid <uid> --environment production；不接受UAT、混用--functions或无用origin。基线--functions仍117/113，新增调度7实体（4个public可调用、3个私有不可调用）；完整集合当前为124实体/117个普通身份可执行，最终仍须按候选读回，不拿旧收据替代。
- 安装原子性Red：后段故意失败留下部分schema。生成器改为单条DO中的事务性EXECUTE，适配官方ExecutePGSql单语句约束；相同后段失败测试Green，不拆成跨调用BEGIN/COMMIT。SharedPG与CLI回归通过6.976s/0.825s。此前接入后的全量API回归通过，sharedpg10.259s；后续目录强化的sharedpg回归7.095s。无候选提交或业务部署。
- 官方资料核对：CloudBase CreateFunction支持AutoCreateClsTopic/AutoDeployClsTopicIndex/TraceEnable显式FALSE；Nodejs20.19为推荐事件函数运行时。现有SCF用量0（786d1538-2af1-46bb-90cf-a2737a81f821），函数列表0（28968367-82a1-40fd-8a75-cb356fad3f0f）；本次使用已有套餐作有限探测，不购买、充值或启用新费用项。
- 临时函数order-readiness-wake-20261015创建04b3e082-a150-42a9-92e9-108abaee2212；读回Active、Nodejs20.19、128MB、3秒、无trigger、无CLS topic/logset、无环境秘密（da9381db-1938-4046-b2da-8d0e0da3294a）。函数仅返回固定回执，不连接业务/COS/支付。平台默认执行角色TCB_QcsRole，未新增角色或策略。
- 普通身份真实函数调用200且accepted=true，4c63ef87-bded-44d4-aa7e-4310b95a9c79；anon403/EXCEED_AUTHORITY，6429d880-b462-4de6-8d9d-f76b02c5781b。未放开匿名规则，未创建管理员API key。Prod用户OPA为空（2550ba0c-f304-453a-b308-4df6354bf80e）。
- 临时UID限定RPC只配置短时普通token并排队固定函数，创建2769c06e-02a5-4c24-a38f-1a4f4b77eb52；提交75794c00-4cd9-4ff3-8014-f580e225a46f，request_id=3。后台执行实际返回固定probe和accepted=true/status200、timed_out=false，2026-10-15 15:56:27+08；读回db5a99e8-0900-4d0e-bc29-42e0dc029978，SCF执行回执eceea946-c86d-11f1-ba00-5254002608d6，queue0。这证明DB提交→持久队列→网关→真实函数，仍不是完整业务补偿或资金验收。
- 已删除临时RPC、两个加密配置项与id3响应（4067b483-3aa6-4014-9016-5e51abf86d5b），删除临时函数bffb11f5-3020-4d73-85f5-43a0eadaf108。最终DB configs/queue/responses/jobs均0、probe不存在（035f2447-36df-42db-b6f5-e7ab1a73571f），函数列表0（47b215b1-56e8-4619-950f-2a6b1d7f9ade）。保留已安装扩展；无临时token文件或后台探测进程。
- 待续：正式持久调用凭据（不能使用已清理的短时token）、平台全链路故障/重启/多实例业务验收、最终候选和部署。未提交工单，Realtime不是这条已实测通道的依赖。

### 持久调用凭据的官方边界（继续核对，未写策略）

- 官方 CreateApiKey 文档明确 api_key 为整个环境的数据流管理员权限，不支持子资源粒度；publish_key 是长期匿名凭据。不能把普通短时JWT当持久凭据，也不能把管理员key伪称为最小调用身份。参考 https://cloud.tencent.com/document/product/876/129835 。本轮未创建api_key。
- PG环境使用OPA用户策略，当前authz.user.rego为空。官方 https://docs.cloudbase.net/envconfig/authz-opa/intro 说明input含身份、HTTP方法/路径/headers、EnvId/资源类型；策略只追加allow/deny，但启用setPolicy会禁用旧网关鉴权。需要核对旧默认边界及其他资源影响，再验证仅目标函数的最小放通，不能以全环境公开规则解除403。本轮未写OPA或放开匿名。
- 云函数临时探测已完成清理，后续不要重复已通过的普通登录/普通调用/DB出站试验；新的验证应针对正式持久身份和权限边界。HTTP200必须伴随实际函数回执，已通过的此层证据仍不等于业务E2E。

## 2026-10-15 用户调整执行顺序：先部署，再完整回归
- 用户明确要求复用 UAT 上线流程，减少上线前复杂测试。后续完整业务/真机/容量/恢复验证放在 Prod 部署后；不以旧 UI3 前置和全矩阵阻止首次部署，不标记未执行项通过。保留构建、环境与凭据隔离、SQL/ACL差异核对、成本边界及必要回滚检查。
- 新增普通认证续签：登录后先将网关已认证 bearer 写入扩展加密配置，再允许业务 RPC；只传空参数，不传秘密正文。独立续签循环只维护登录，不承载业务补偿。私有 expiry 元数据防止多实例旧令牌覆盖。无新增管理 key，无匿名函数放行，无 OPA 切换。
- 真实 Prod 最小探测：认证请求头存在；登录A后登录B，A仍可用；匿名401。创建541cb427-5334-485a-930c-9ef310536793；首次e44491f9-619b-4e3f-a106-23a0d13a189d；第二次c5753668-8387-4129-9ed7-74420f8ae3c1；旧令牌e904310f-04cb-406a-87a4-4277e0486a0e；匿名8612126b-a961-492e-8d60-cd3f6a0f7632；删除512c2ab0-7cf1-402e-95c7-5454d34b8610。只验证续签前提，不是长期运行或业务完成证明。
- Red：生产客户端先运行了业务RPC、未登记调用身份；SQL缺少续签RPC。实现后 sharedpg/cloudbasepg已有回归通过（7.190s/0.811s），新增续签SQL最小检查继续收尾。当前完整函数集合125实体/118可执行，覆盖此前124/117统计。
- CloudRun envStatus再读回Prod normal、isExist=true、Trial；官方工具明确可使用现有 manageCloudRun deploy，无须重新initEnv或先买资源。实际套餐消耗及首次部署结果仍需记录。
