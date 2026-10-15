# Order CloudBase 生产发布 Runbook

更新：2026-10-15，北京时间。**当前 NO-GO。Prod普通身份、登录及同款HTTP RPC已验证；平台扩展已安装并完成有限探测，业务SQL安装、部署和切流仍为 NOT_RUN。** 按用户最新要求，先沿用 UAT 发布通道完成 Prod 部署，再完成完整业务回归；部署成功不代表业务验收通过。部署前只保留目标/凭据隔离、可构建、SQL差异与权限核对、成本边界及必要回滚检查。

唯一后续实施方向（不是当前已生效的生产实现）：已存在的独立Prod CloudBase环境 + CloudRun Go API及同进程worker + Shared PG + 私有COS + 同源HTTP网关/静态托管 + 已备案自定义域名。后续人类明确希望尽量托管CloudBase，晚于8月CVM/MySQL技术选择；[决策原文与源码审计](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/production/report.md)保留来源。旧systemd/MySQL示例不能作为执行命令。业务、资金、隔离与恢复合同保持，不以技术迁移降低验收。

当前完整Goal已授权本项目必要生产配置、最小权限、经核对SQL安装、部署、DNS/TLS、切流和main集成；每项仍须先满足本Runbook前置、核对具体差异并读回。新增费用、采购、启用/扩大计费及真实资金操作另行确认，未获确认不执行。普通身份及匿名publish_key已创建并验证真实登录；业务SQL、部署与切流仍未执行。

## 1. 发布前硬前置

本次 `prepare-cloudbase-production` 已按用户确认方向更新 production-architecture-baseline 与 MODIFIED delta，并通过 OpenSpec strict。规格对齐不代表实现、独验或生产发布通过；可靠补偿、平台配置和真实验收仍是硬前置。


| 条件 | 当前证据 / 下一步 | 责任 |
|---|---|---|
| 架构规格同步 | 已本地同步CloudRun/SharedPG/私有COS、身份材料及30分钟异常补偿契约；OpenSpec strict通过，尚无最终候选独验 | Owner/后端 |
| 生产运行入口 | 本地适配已接入production CloudRun组合、EnvId/UID/桶隔离、STS和材料、同源回调及formal校验；普通身份已创建并真实登录通过；仍需数据库权限、补偿恢复、客户端和最终候选独验 | 后端 |
| 最终候选业务验收 | 复用[验收矩阵](../../docs/quality/order-uat-acceptance-20261012.md)，对最终Prod候选完成隔离、目标平台及真机验证；UAT恢复/充值不是前置。通知、真实身份及原生步骤待闭环 | Owner/实际参与者 |
| 3000人容量 | 没有通过证明；菜单小样本p95约981ms，高于500ms读目标。按业务高峰、HTTP/RPC、worker、数据库、发布并存测量，不把配额当吞吐 | SRE |
| Shared PG备份与恢复 | 本地PG16恢复成功不能代替线上；必须查明当前实例计划/恢复点/保留期/恢复目标与权限，并在隔离目标演练。未证明RPO≤5分钟、RTO≤2小时 | SRE/数据库执行者 |
| 平台及正式域名 | 分别读回小程序备案、域名ICP备案、认证、类目资质、隐私、手机号余量、支付关联、模板、审核；正式域名/DNS/TLS未有完整实证 | 甲方管理员/开发方 |
| 正式客户端 | release endpoint已绑定order.suianwuye.com；两模板仍未核实；后端必须formal，真实深链可打开对应订单 | 前后端/平台管理员 |
| 正式材料与预算 | 现有Prod自动续费、超额使用已为true；未在本轮改变。确认现有预算/余额/告警责任，不推断无限支出授权 | 客户资源持有人/SRE |

上述容量、真机、完整业务与恢复演练移到部署后验收，不再全部作为首次部署前置；实际缺失的运行材料、权限、安全和费用边界仍须解决。禁止在Runbook里用猜测的UID、域名、密钥或SHA填空冒充可执行配置。

## 2. 资产、身份与版本登记

2026-10-15重新只读核实的Prod（queryEnv RequestId `eea77690-9092-40d8-a62a-14515cc373f1`）：order-prod-d0g4xnw49c5ada5ce，ap-shanghai，标准版baas_pf_standard，Shared PG postgres-gy041jr4；独立于UAT order-uat-d1gtzpzj50615c300。Prod预付到期2027-04-30，API返回QPS配额500，配额不是性能承诺。本次读回与RequestId见[任务账本](../../openspec/changes/prepare-cloudbase-production/tasks.md)；10月12日资源收据仅为历史，不作为本次状态证明。

执行前使用auth status确认账号和目标；queryEnv info显式传Prod EnvId，保留脱敏结果。所有写操作先核对目标Prod，禁止切换错环境。记录以下内容：

- API源码完整SHA、候选独验、构建binary SHA256、镜像digest、BuildId/RunId/版本与流量。
- SQL关系链版本、函数安装摘要、定义/ACL/RLS指纹、执行账本和恢复点。
- PC各文件manifest、对象摘要/缓存版本；Mini exact SHA、上传版本、体验/审核/正式状态。
- Prod独立DB服务UID、桶与最小权限身份、域名/证书、告警接收人、预算责任；秘密只登记私有安全位置及版本引用，不存值。

CloudRun单独读回：DescribeEnvBaseInfo的 IsExist=true、Status=normal、PackageType=Trial（RequestId `c92b438b-27d1-4b75-881b-3863c3ad9fc7`），而DescribeCloudRunEnvs返回UNAVAILABLE（`346a7293-549d-4b19-a029-4abc6dbae68e`）；这不能与CloudBase标准版混为同一个套餐结论，需继续核对服务、当前可用性及已有权益，不据此直接采购。CloudBase工具当前已绑定Prod；每次操作仍核对EnvId，历史UAT绑定结果不记入Prod。

2026-10-15显式Prod读回：CloudRun服务0（`57f9d9c7-a1e0-4a70-81d3-c8b12b7a0a06`）；public业务表0、order函数0、PG17.11（`5f43ced6-21ad-4bf8-aba2-4538d29c1dc3`）；网关仅两个默认域名，无正式自定义域名/API路由（`4d9740f9-623b-4b6c-b747-a41fed9490dc`）。这是首次生产安装准备，不是已有生产版本升级。其他条目仍待现场核验；旧UAT收据不转记Prod。

## 3. 生产运行入口与配置准备

后端实现需在独立change完成并通过配置隔离Red/Green、真实隔离SQL、假provider HTTP/worker、秘密扫描和exact候选独验。具体入口见[源码差异审计](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/production/report.md)：

1. main及SharedPG组合明确production分支，向支付、退款、核销、通知、账单传production，不能伪装uat。
2. cloudbasepg client保留允许环境、publishable JWT audience/project/anon role、普通ServiceUID认证、禁重定向及RPC allowlist；错环境与高权材料必须拒绝。
3. COS STS策略独立绑定Prod私有桶images前缀，保持受限临时身份；禁止把UAT桶或临时对象混入Prod。
4. 材料来源唯一且受保护：DB普通身份、AppSecret、商户支付私钥/APIv3/验证公钥和核销密钥。当前旧CVM Role/SSM和UAT内联方式不直接成为已验证Prod方案；选定并实测轮换后写入最终配置清单。
5. 生产模板的11项订阅配置完整，字段、页面、formal及环境对应；支付/退款回调为正式同源/api/v1路径。后端不得启动时自动迁移/初始化。

此实现包未完成前，不给当前二进制增加Prod流量；本Runbook不是绕过配置guard的许可。

## 4. 数据库安装、迁移与初始化

以下管理动作只在生产授权后执行。先只读schema、现有数据、关系/函数清单、版本账本、序列、权限和恢复点，确定空库安装还是明确增量。

- 关系链：services/api/migrations/postgresql的实际v1–44；order-shared-pg-install只生成关系安装SQL，不能当作完整函数安装器。非匹配数据库不能靠清dirty或删账本强行安装。
- 函数链：services/api/sharedpg.InstallSQL接收真实Prod serviceUID、production及两个正式回调URL，生成经审查产物；本次 UAT 基线为117个函数实体/113个可执行函数、4个旧入口撤权；最终候选若改变函数集合，按实际生成结果与 readiness 契约逐项核对。历史113/109收据不能证明本版本安装。
- 补偿链：使用 `order-shared-pg-install --compensation --service-uid <已核验UID> --environment production` 单独生成；单条DO原子安装私有状态、触发器及8个函数实体（5个public可调用、3个私有不可调用）。与基线合计当前125实体/118个普通身份可执行，最终以精确候选为准。已安装时不得重跑首次创建脚本；未知结果先读目录及账本。
- 管理通道必须以原子执行结果、定义摘要和真实账本读回确认成功。PushPGUserMigrations不是唯一执行前提；平台Repair仅可登记已经成功的精确SQL，不能修饰失败。
- 普通服务身份验证RLS、SECURITY INVOKER和允许RPC；匿名/非允许身份不能直接访问业务表或管理函数。确认退款、核销与支付回调不跨环境。
- 初始化沿SharedPG bootstrap五参数逻辑，首次创建真实OWNER和门店；native order-bootstrap不能直接冒充SharedPG执行命令。先查created/unchanged及冲突，未知结果只读回、不重复初始化。
- 不复制UAT人员、订单、低价菜、许可或账本进Prod。已存在真实数据时以增量方案和恢复点执行。

安装receipt必须包含目标EnvId、exact候选、SQL摘要、提交/完成时间、最终账本、定义和权限指纹；不含手机号、token或密钥。未取得明确结果不继续启动业务。

平台扩展已成功安装：pgcrypto 1.3、pg_cron 1.6、tencentdb_scf 1.1；不要重复安装。安装及临时探测清理见[平台回执](../../openspec/changes/prepare-cloudbase-production/platform-prerequisites-receipt.json)。它们不是业务函数，不并入117/113基线。2026-10-15只读核对三者owner均为当前安装身份（RequestId bfa63232-88fc-4c4b-bf4f-4a3b8b9c2964）。首次关系安装已在本地适配已有pgcrypto，仍拒绝不属于该扩展的public对象；生产业务安装尚未执行。

可靠唤醒当前候选实现为事务内记录级状态 + pg_cron + tencentdb_scf + 固定云函数转发至API。Realtime不是UAT已验证路径，也不是该实现前置。SQL安装生成、目录/权限检查、定向RunOne、HTTP adapter及生产运行组合已通过本地隔离测试；有限平台探测已证明数据库提交后实际执行云函数。普通登录续签现已接入扩展加密配置：RPC仅接受网关认证请求头、不接受token参数；多实例按到期时间防止旧令牌覆盖。API维护登录续签，任务仍由数据库持久调度；整个API不可用时不能完成业务，恢复后重新登录并继续数据库中的待办。最终候选及真实部署后回归仍未完成。临时探测的普通登录token已清理，不能当作正式持久凭据。

## 5. API、worker与静态发布

已实测可参考通道是UAT CloudRun构建/部署/读回，见[UAT运行手册](order-uat-cloudrun-runtime.md)。Prod执行顺序：

1. 用已独验源码及 `deploy/cloudrun/Dockerfile.production`（专用default-deny dockerignore）构建，禁止使用写死trial模板的Dockerfile.uat。生产镜像不嵌入订阅模板或秘密，由受保护的运行配置注入；锁定Go1.26.5与目标架构构建；镜像非root、端口和health配置与候选一致。生成可回退构建物与配置版本，不把含UAT材料的现存包上传Prod。
2. 使用CloudRun现有部署通道提交一次；记录BuildId/RunId。结果不明先查部署任务和记录，不重复创建版本。
3. 查询服务详情时保持revealEnvParams=false，确认版本、digest、资源、环境键、normal与预期流量；读取live和ready。ready必须验证Prod真实通道、函数、身份与配置；HTTP200不是业务验收。
4. 当前源码只有支付、退款、订阅及账单后台任务。自动制作扫描已移除，制作中由服务端时间派生，取消仅在距取餐严格大于30分钟时允许。本地production组合已替换支付/退款/订阅每秒扫描，改为数据库持久唤醒后仅执行指定记录；UAT保留历史worker。新实现尚未部署，最终候选及生产异常恢复验收仍是发布前置。可靠补偿只登记真实异常，失败后30分钟重试、成功停止；必须证明持久登记后崩溃、重启及多实例恢复，不能以进程timer或固定30分钟全局扫描替代。保留至少一个常驻实例；并发副本/扩缩配置须来自负载和租约验证，不能机械复制UAT Min1/Max1、0.25CPU/0.5GiB。
5. PC唯一入口为正式origin下 `/pc/`，网关 `/pc` 映射静态托管根；不增加旧地址兼容。将完整 `apps/web-admin/` 内容作为一个版本，资源先上传、`index.html`最后上传，按manifest逐文件摘要读回。业务文件版本、网关缓存和浏览器实际版本对应；记录保存/换页读回/恢复。
6. 正式小程序仅在Prod入口就绪后配置release endpoint/模板、按exact候选上传。上传接受、设体验、审核通过、实际发布分别记账。

新旧实例共存或重启必须实测租约交接、停止领取、inbox/outbox不丢失和外部请求结果不明后的收敛。worker串行每批100不等于100笔/秒。

## 6. 域名、证书与微信平台

先查已有自定义域名与DNS，复用合法资源；生产PC/API使用已备案自定义域名与有效TLS。HTTP网关API路径透传，PC同源/api/v1，静态路由与缓存规则按真实配置读回。微信request/upload/download只登记实际使用域名，不能关闭合法域名校验。

正式回调分别为正式origin下/api/v1/payments/wechat/notify与/api/v1/refunds/wechat/notify；签名、金额、订单号、AppID、商户及production校验保留。已确认AppID和商户身份不重新索取，也不假定平台普遍要求第二个AppID。

平台管理员按[逐项官方核查表](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/platform/report.md)读回：小程序备案、域名/网站备案分别记录；认证与有效期；自营餐饮类目和经营资质；现网/提审隐私指引；手机号已购额度与余量；支付关联/JSAPI权限；订阅模板字段；体验成员/版本及审核发布。

手机号默认1000次体验额度由开发/体验/正式共用，标准费用0.03元/成功调用；以后台已有资源为准，差额先报告，未经授权不采购。见[微信官方手机号说明](https://developers.weixin.qq.com/miniprogram/dev/framework/open-ability/getPhoneNumber.html)。真实手机号同意、支付确认、原生扫码和平台主体核验均由本人完成。后台安全拒绝时只停该操作，不换入口绕过。

订阅发送miniprogram_state用formal，前端环境名用release；两者不可混填。依次验证provider接受、本地SENT、设备到达、点击正确订单；页面存在不代表订单归属正确，不为通链路放宽鉴权。[微信发送API](https://developers.weixin.qq.com/miniprogram/dev/server/API/mp-message-management/subscribe-message/api_sendmessage.html)

## 7. 经营数据与发布验收

复用已确认店名“绥安食品”、既有地址和唯一取餐点；午11:30截单/11:30–13:30取餐，晚17:00截单/17:00–19:00取餐，30分钟粒度。甲方提供正式商品、图片、售价、可售日期、员工折扣名单及STAFF；通过正常UI保存、离页和后端读回。不要把UAT合成数据上生产。

正式验收按完整UAT映射覆盖：顾客菜单/购物车/报价/折扣/价格漂移/隐私手机号；支付取消与成功；预约、worker、备好；真实相机和重复/跨日核销；顾客与OWNER全额退款及未知结果收敛；通知；PC角色/重登；财务明细、账单来源/一致性；异常恢复。真实资金演练金额/次数另按已获生产授权执行，未知结果先查询，不重复扣款或退款。

同形态隔离负载需报告数据基线、到达模型、资源、HTTP/RPC延迟、5xx、锁等待、worker积压以及资金不变量。先采用既有200并发/300单每5分钟基线，再按用户最高峰输入判断2000–3000人是否覆盖。达到读p95≤500ms、写p95≤800ms、5xx<0.1%并无重复资金/订单投影，才填写容量通过；资源健康和配额不足以通过。

## 8. 监控、告警与值守

发布前落实CLS日志脱敏和关联、资源指标、API错误/延迟、inbox/outbox最老年龄与DEAD、worker处理/租约、账单差异、余额/费用、证书和备份失败。不要记录姓名/手机号/openid/备注/token/取餐码/密钥或支付原文。

沿用既有语义门槛：每分钟探测连续3次失败、5xx五分钟>1%、读p95>500ms/写p95>800ms持续5分钟、待处理>60秒、任一DEAD/对账不一致/备份失败、证书不足30天。阈值映射至实际CloudRun/SharedPG指标，不能沿用不存在的CVM监控对象。触发一条隔离告警并实际确认到人及恢复通知，登记接收人和值守替补；没有送达回执不算完成。

## 9. 备份与恢复

现有目标：进程/DB故障RTO≤5分钟、计算节点丢失≤60分钟、逻辑损坏RPO≤5分钟/RTO≤2小时，区域级灾难不承诺。它们未在当前Shared PG实证达到。官方存在PG备份/恢复管控接口，不代表当前共享实例套餐适用；不能拿MySQL/NoSQL或独享PG能力套用，也不能因无原生DSN断言无恢复。

2026-10-15只读SQL确认Prod PG17.11可查询；备份计划重新查询仍被CAM拒绝，操作postgres:DescribeBackupPlans，RequestId `23c688c4-3d3c-4b7b-bde8-04a6d21c968f`。没有绕过拒绝或修改权限，不能据此写无备份。续接由资源持有人提供现有计划/记录的只读结果，或明确授权最小查询权限；见[实际阻断收据](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/prod-backup-readback.json)。

执行人先只读实际备份计划、成功记录、恢复点、保留期、权限、可恢复目标和费用。在同产品隔离目标恢复，不覆盖当前生产；核对schema/函数/RLS/ACL/序列、订单/支付/退款/核销/审计及外部资金状态。记录恢复点、开始/结束和丢失窗口，再判断目标。不存在可证明的恢复路径则保持NO-GO，报告失败的合同和需要的外部条件，不降低目标。

COS单独验证版本/保留、误删恢复和私有访问，不能用上传下载成功替代恢复。恢复后重新核对回调/worker和财务，避免重放已发生的退款。生产数据恢复切换需后续明确授权。

## 10. 回滚与故障处理

发布前登记已验证的前一镜像digest、配置版本和静态manifest，证明兼容当前SQL/函数。失败时由唯一发布者查询任务结果，停止继续切流；按CloudRun版本流量管理回到兼容版本，并读回版本/流量/ready、inbox/outbox、订单/退款数量与金额。未经实证兼容的旧镜像不可回切。

数据库不自动down、不清dirty、不删账本；使用经审查的forward fix或已验证恢复。遇支付/退款响应不明只查询供应商和本地投影，再收敛同一业务对象。通知终态不人工重放；健康但业务失败分别查身份、路由、函数和worker，不重复安装全库。

任何身份验证或平台安全拒绝由本人处理；必要材料/资源缺失只停对应链路。记录首个决定性错误、已经通过的条件和下一次重试所需新证据，禁止无变化忙重试。

## 11. 清理与交付完成

按授权清理临时演练身份、测试对象、未用构建物与临时权限；保留真实订单/资金审计。读回清理结果，核对密钥轮换、证书/套餐到期、余额告警和责任人。

最终交付包包括：平台资格/正式域名、API/PC/Mini各自精确版本、资源/密钥安全引用索引、SQL/初始化账本、完整真实业务验收、容量与恢复报告、告警到人、回滚回执、经营配置和客户交接。只有实际完成正式发布且全部必需项通过，才登记“已上线可交付”。本轮所有Prod执行项保持NOT_RUN。


## 12. 本次发布执行清单（按顺序，不是执行回执）

用户于 2026-10-15 明确要求：涉及加钱必须停下核对。新增付费资源、套餐升级、超额计费开关，以及可能新增账单的恢复演练/图片处理/运行资源，先列明项目、计费方式、预估金额与停止条件，获确认后执行；已有开关状态不视为本次费用授权。

本节为唯一生产操作顺序。未满足验证栏即停止后续依赖步骤，继续无依赖的本地工作。所有生产写入、切流、费用及真实资金操作先呈交具体资源、差异和回退内容，取得对应授权；表中计划不得勾记为已执行。当前候选 `NOT_CREATED`，所有生产写步骤 `NOT_RUN`，结论 **NO-GO**。本地进度见 [生产适配任务账本](../../openspec/changes/prepare-cloudbase-production/tasks.md)。

| 序号 / 目标 | 操作 | 验证及交付证据 | 失败处理 / 回滚条件 |
|---|---|---|---|
| 1 固定源码 | SSH读取main/uat；本次基线uat `531638a8961796a7938072463e31ff15dba466ac`，main `8aef176b69239ab9b952bff6f452108da8ad9476`；main无独有提交，UAT领先243。独立适配分支完成后按授权逐文件网页提交；每次读回hash/diff | 文件清单全量核验；干净候选完整SHA；独立detached验证；集成main后视为新SHA重新验证并构建 | 基线变化先比较，禁止覆盖main；缺独验或客户端receipt不得集成；未获授权不写生产分支 |
| 2 只读确认资产 | 显式Prod EnvId检查套餐、配额、服务/版本、账期用量、预算告警、备份计划及恢复点、域名/DNS/TLS、小程序备案/认证/类目/隐私/支付关联和发布资格 | 保存时间、接口、RequestId及脱敏值；网站ICP备案与小程序备案分别有实证；告警接收人/预算责任落实 | 安全拒绝停止该通道，记录权限及资源持有人续接条件；不能以网站ICP备案替代小程序备案，不能自动开费用 |
| 3 冻结材料清单 | 保护配置中准备独立PG普通用户名/密码/UID及对应anon publishable key、Prod COS受限身份、已确认AppID/AppSecret、支付与核销JSON、正式origin、11项订阅配置 | 配置负向测试拒绝UAT EnvId/桶/UID/key、管理key和原生DSN；formal；正式回调精确同源；材料只记录安全版本引用 | 缺值拒绝启动；不可复制UAT身份或凭据；材料泄漏先停发并由持有人轮换，禁止把撤销版本作为回滚材料 |
| 4 冻结数据库差异 | 从精确候选生成关系安装/增量与函数SQL；只读对比当前schema、关系账本、平台迁移历史、函数定义/ACL/RLS；确定仅需执行的差异 | 候选SHA、SQL SHA256、117/113基线与最终集合差异、26表/16序列及普通身份契约；已成功迁移排除 | 不重复成功迁移；结果不明先读回；禁止清dirty/删账本/自动down |
| 5 恢复准备 | 核对在线实际备份和可恢复目标，在批准的隔离目标演练，COS版本/误删恢复独立演练 | RPO/RTO、函数/ACL/RLS/序列、订单/资金/核销/审计一致性及时间证据；本地PG恢复不算线上证明 | 无可验证恢复路线保持NO-GO；不覆盖生产，不扩大为付费资源采购 |
| 6 安装明确SQL差异〔生产写〕 | 授权后由唯一writer对指定Prod执行已审核SQL；关系与函数分别记账，设置锁/执行超时 | 原子执行结果、普通身份ready、定义摘要、ACL/RLS及平台/业务账本读回；Repair只登记确实已执行的SQL | 未知结果先查，不重复；不兼容则停止应用发布，走审查后的forward fix或已验证恢复 |
| 7 部署API〔生产写〕 | 由集成后精确SHA构建独立生产镜像/包，注入保护材料；部署一次，读回BuildId/RunId/digest/版本/配置键与流量 | live/ready分别记录；普通身份访问成功；跨环境/越权拒绝；重启、多实例租约、补偿成功停止验证 | 任务未决先查；无变化不重发。业务失败停止切流，只有已证明兼容当前SQL的版本可回退 |
| 8 配置图片〔可能费用/写入〕 | 核实Prod私有桶、版本控制、images非当前版本30天保留、数据万象派生能力与现有权益；固定versionId签名派生GET使用 `imageMogr2/thumbnail/640x640>/format/webp/quality/80` | 存量图、新上传、三图导入分别验证：固定版本、匿名拒绝、过期拒绝、图片内容/裁剪正确；记录原图/缩略图字节、Content-Type、冷/热请求耗时、端侧加载时间 | 403/样式不可用先查授权/版本/签名；不得公有化桶或静默退回原图掩盖性能问题；新增费用先确认 |
| 9 发布PC〔生产写〕 | 正式域名 `/pc/`；上传完整静态包资源后最后上传HTML。manifest含相对路径/字节数/SHA256；HTML不长缓存；稳定文件名JS/CSS须重新验证缓存，不能长期immutable | 所有资源摘要与候选一致；Content-Type/Cache-Control/ETag；干净浏览器及已有缓存浏览器实际页面；无旧路径兼容；扫码登录、过期/拒绝/退出、权限检查 | 任一资源缺失不上传HTML；混版本停止对外开放；回退同一完整manifest，资源先HTML后，清理或重新验证缓存 |
| 10 发布小程序〔生产写〕 | Prod入口通过后填写release origin和已核实两模板；合法域名配置与真实上传/下载域名一致；绑定精确候选，上传、体验、提审、发布分别操作记账 | 顾客/商户/PC完整矩阵；release与formal对应；备案/审核/发布资格确认；原生手机号/支付/扫码/订阅由授权本人完成 | release空值或回调未就绪停止；上传成功不等于审核/发布；不关闭合法域名验证绕过 |
| 11 受控资金和通知验收〔资金另授权〕 | 确认具体金额/次数/退款对象后执行付款、支付恢复、退款；通知首次发送及失败补偿；重复/并发与30分钟边界使用隔离夹具优先 | 同一交易/退款唯一事实，观察→物化→页面→外部资金结果；通知provider接受、本地SENT、设备到达、点击正确归属订单分别记录 | 未知资金结果只查询同一业务对象，不重复付款/退款；永久错误停止补偿；无设备到达不得写通知E2E通过 |
| 12 切流与交接〔生产写〕 | 所有必需门禁通过后按批准流量计划切换；值守、告警到人及恢复通知、业务指标和版本清单交接 | 最终GO表、精确API/SQL/PC/Mini版本、容量/恢复/资金/业务验收及回滚证据 | 任一必需项FAIL/BLOCKED_EXTERNAL/NOT_RUN为NO-GO；业务故障按第10节，数据库不自动回滚 |

### 12.1 图片实测记录格式

每个“存量图片/新上传/三图批量导入”场景至少记录一次原图及派生图的同网络对照，并重复5次报告中位数与最慢值；小样本不称为p95或容量结论。字段：候选SHA、环境、脱敏对象引用、版本ID摘要、原图字节/派生字节、响应类型、缓存命中情况、请求耗时、端侧可见耗时、失败率、观察时间。临时签名URL不写入Runbook/普通日志。存量图片需确认存在可绑定的已确认版本，不能把未验证原图批量改写为已确认。

三图导入须覆盖完整预览→确认→提交→离页重读，验证三图顺序、裁剪、重复提交幂等、部分错误与用户可见结果；禁止为测试重导UAT示例或把样本经营数据写入生产。未获生产测试写入授权时先完成本地链路，生产步骤保持NOT_RUN。

### 12.2 本次生产配置新增项

`ORDER_ENV=production`、已核实Prod `CLOUDBASE_ENV_ID` / `ORDER_COS_BUCKET` / `ORDER_TENCENT_REGION`；独立 `ORDER_SHARED_PG_*` 普通身份；`ORDER_PRODUCTION_COS_SECRET_ID/KEY`；`ORDER_PUBLIC_ORIGIN` 为无尾斜杠的正式HTTPS origin。其余AppID/AppSecret、支付/核销JSON和11项订阅键沿第3/6节契约，材料读取使用CloudRun保护EnvParams，不再依赖CVM Role/SSM。生产组合尚在本地实现，不能把本节当作部署成功。不得携带 `ORDER_UAT_COS_*`、原生DB/管理API key或通用云环境凭据。配置总字节数按当前平台限制核对，秘密不得进入构建物。


### 12.3 精确候选 SQL 生成（本地，不执行数据库写入）

先在最终干净候选worktree核对 `git rev-parse HEAD`。仅当正式UID/origin已读回后设置下列变量；不填测试UID或示例域名。文件输出到仓库外受限目录，记录SHA256及候选SHA。关系SQL和函数SQL不是同一安装工件；生产本次读回为空库仍须在执行当天重读，排除并发安装。

```sh
# 在精确候选worktree执行；变量必须由已核实的生产清单提供。
: "${ORDER_RELEASE_SQL_DIR:?必须为已创建的仓库外受限目录}"
: "${ORDER_RELEASE_SERVICE_UID:?必须为生产普通服务身份UID}"
: "${ORDER_RELEASE_ORIGIN:?必须为已核实的正式HTTPS origin，无尾斜杠}"
umask 077
GOPROXY=off GOTOOLCHAIN=go1.26.5 go run ./services/api/cmd/order-shared-pg-install > "$ORDER_RELEASE_SQL_DIR/relations.sql"
GOPROXY=off GOTOOLCHAIN=go1.26.5 go run ./services/api/cmd/order-shared-pg-install --functions --service-uid "$ORDER_RELEASE_SERVICE_UID" --environment production --origin "$ORDER_RELEASE_ORIGIN" > "$ORDER_RELEASE_SQL_DIR/functions.sql"
shasum -a 256 "$ORDER_RELEASE_SQL_DIR/relations.sql" "$ORDER_RELEASE_SQL_DIR/functions.sql"
```

每条命令必须成功，任何失败产物不可执行；空文件或缺绑定标记检查失败即停止。生成器无数据库连接能力，生成成功不代表安装。安装前审阅事务/锁、函数集合和授权；安装后用普通身份执行readiness并核对实际定义/权限/账本。

### 12.4 最新平台验证限制（2026-10-15）

UAT `order-uat-d1gtzpzj50615c300` 的 `ExecutePGSql` 只读目录查询返回 `instance status must be Running to execute SQL, current: Isolated`。环境总览仍NORMAL、个人版到期2026-10-31、超额开关false；不能据此确认数据库可用或判断隔离原因。暂停依赖此通道的UAT验证，先由资源持有人核实实例状态与恢复条件；若涉及费用，列明报价并经用户确认后操作。禁止为了继续验收直接在生产安装测试表或复制UAT数据。恢复后重新读回实例及普通身份能力，再续接精确候选验证。

随后正常登录新版控制台补充证据：UAT套餐资源点4万/4万，页面明确“已经超额”，累计消耗约4.06万点；资源包0点、按量付费开关关闭。SQL型数据库入口返回res_stopped（RequestId `5b6ae4be-1ccf-42d4-b5a7-4dec3d7ab47c`）。资源耗尽已确认存在，不能继续称为只有未知隔离原因；仍须核实具体恢复报价及账户其他限制。页面640.34点为资源点而非人民币。尚未购买、续费或切换计费；按用户要求停在付费恢复前。

Prod本账期用量接口读回COS 0.01资源点、APIInvocation 0.89资源点，其他列出模块返回0；这是历史计量结果，不是人民币金额、剩余额度或实时服务免费证明。实时连接、CloudRun常驻实例、图片派生和恢复演练的新增费用仍需分别核对。当前未获开启对应费用的授权。

### 12.5 PC静态包本地生成

在干净的最终候选worktree执行 `node tools/pc-release-package.cjs /绝对路径/新的发布目录`。命令拒绝dirty仓库、已有输出目录、旧base、缺失引用及符号链接；生成pc文件树和manifest.json，按清单顺序先上传所有非HTML文件、读回其摘要，最后上传HTML。当前稳定文件名统一要求Cache-Control: no-cache（允许存储但每次重新验证），不能据此声称云端已生效。manifest列出Content-Type、字节数、SHA256和目标/pc/路径。上传工具须显式设置并读回响应头，随后按第9步验证浏览器缓存/登录/页面。

未提交阶段仅可使用 `--draft` 本地检查，产物标记DRAFT_NOT_DEPLOYABLE且candidateSHA为空，不能上传或作为精确候选收据。脚本不上传、不连接后台、不自动修改云端缓存。

### 12.6 正式域名（已确认，本地已绑定，线上未切流）

2026-10-15只读核验后推荐 `order.suianwuye.com`：PC入口 `https://order.suianwuye.com/pc/`，API同源 `/api/v1`。现有UAT `uat.suianwuye.com` 已解析到CloudBase；根域NS为mallard.dnspod.net和caramel.dnspod.net。推荐子域的A/AAAA查询返回NXDOMAIN，目前未发现公开解析占用；这不证明管理权或备案状态。

UAT TLS实读仅覆盖uat.suianwuye.com，有效期2026-10-13至2027-01-11，不能复用于推荐域名。DNSPod DescribeDomain被CAM拒绝（RequestId b361f37f-3343-4c84-baf3-56b1ff9a67a3），停止该权限通道；域名持有人、注册到期、ICP备案、生产证书和绑定资格仍待有权限主体核验。推荐复用已有根域，不购买新域名；实际DNS套餐/证书/网关费用未证明为零，任何加钱先核对。用户随后已确认采用order.suianwuye.com，本地release现已绑定并拒绝UAT origin；此确认不含DNS写入、证书购买或切流。


### 12.7 UAT恢复费用待确认（2026-10-15）

控制台当前最小资源包为1万点×1个，报价10.00元、有效期1年，仅限上海UAT环境，不可转移；使用后余量不支持退订、到期作废。该包不延长UAT基础套餐2026-10-31的到期时间。建议仅在用户确认后单次购买该包，保持按量付费关闭；未下单，不视为恢复成功或全部验收预算已足够。购买后先只读确认实例恢复、资源点余额及持续消耗，再执行已授权的限定验证；再次不足立即停止并核对，不自动追加购买。


### 12.8 Prod用量与运维状态补核（2026-10-15）

当前控制台标准版额度33万点、消耗17.05点（PG16.15、存储0.01、环境调用0.89），按量累计0点；自动续费和按量付费均原已开启，本次未改动。页面“数据回档14天”仅为权益，不证明已有恢复点或恢复成功。日志管理尚显示“开启日志”及付费模式确认；未开启。CloudBase平台告警策略列表0条、近7天告警0；账户费用预算已确认0条（见12.9），外部可观测平台策略尚未核实，不得将0告警视作健康或已配置。备案管理只显示Prod可备案；小程序认证页无微信备案及发布资格证明。相关门禁继续未通过。


### 12.9 账户预算与余额（2026-10-15只读）

同一项目账户费用中心：可用余额0.00元、欠费0.00元、余额预警关闭；预算管理无筛选列表0条并显示尚未创建预算。生产按量付费原已开启，但该状态不授权新增支出。上线前须确认预算上限、接收人和余额责任，再经明确授权配置并验证通知；本次未充值、未创建预算、未切换开关。UAT10元资源包仍待用户确认，未下单。


### 12.10 Prod主线纠正

用户明确Prod上线不以UAT恢复为整体前置。12.7的10元报价仅作历史备选，未获批准且不执行。可靠补偿继续本地实现和隔离验证；目标平台必要验证另列独立对象、权限、费用和清理清单供授权。正式域名已确认order.suianwuye.com，本地release已配置，DNS/TLS及平台验收仍未完成。

## 2026-10-15 Prod普通身份创建回执

- [x] 用户了解用途后明确“那你去创建吧”，授权创建Prod专用普通身份。
- [x] 创建前Name查询0条，RequestId `77f8ecd8-c8fa-4e0a-a40e-53af8ae916aa`。
- [x] CreateUser返回UID `2110616903056039938`，RequestId `9be02697-d68a-4ecd-b59a-25a31aa52b13`。
- [x] DescribeUserList读回1条：环境 `order-prod-d0g4xnw49c5ada5ce`，Name `order-prod-api`，Type `internalUser`，状态 `ACTIVE`，UID一致；RequestId `9c05eca3-50a0-4ce0-83a3-63917987be78`。
- [x] 独立随机32字符密码保存于仓库外 `/Users/vivix/.config/order/production/order-prod-api.json`，目录0700、文件0600；材料附创建/读回回执，密码不入仓库。
- [ ] 普通身份登录、RPC权限及跨环境拒绝尚未验证。本回执不证明SQL安装或部署成功；未授业务表权限、未安装SQL、未切流、未购买。

## 2026-10-15 Prod匿名key与普通身份登录回执

- 用户明确批准创建publish_key并验证登录。CreateApiKey仅使用KeyType=publish_key，未创建管理员api_key；KeyId `3znAvPKBTAC-OiW38En_lw`，创建RequestId `95afe11a-e2d4-4732-98fb-6484a8db8b20`。
- key的role=anon，aud/project_id均为Prod环境；已存入同一仓库外0600材料文件。
- 2026-10-15 14:25:04北京时间，真实Prod auth/v1/signin验证通过：返回sub与UID `2110616903056039938`一致、role=authenticated、aud/project_id均为 `order-prod-d0g4xnw49c5ada5ce`、Bearer且有效期7200秒；RequestId `bd11369a-c087-4887-a394-1a39b0034074`。核对了实际响应身份，不仅HTTP状态；访问令牌未落盘或写入文档。
- 此前“缺publish_key/登录未验证”已解除；数据库RPC、ACL/RLS、业务与真机仍未验收。无SQL安装、业务授权、切流、采购或资金操作。

### 本地退款补偿进展（2026-10-15）

创建请求失败改为30分钟后补偿；正常PROCESSING创建、查询及重复观察不再安排下一次查询；due/claim不将NULL补偿时间视为普通待办，同时保留过期租约恢复。隔离PostgreSQL回归通过，未调用真实微信退款。创建结果不匹配已改为人工待处理且不自动排队；可靠跨实例唤醒和通知首次发送仍未完成；不可仅据此部署为完整可靠补偿版本。
