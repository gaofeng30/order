# Order CloudBase 生产发布 Runbook

更新：2026-10-15，北京时间。**当前 NO-GO。main 版本 `149423eac8b179cf453e224534c64f84fe458086` 已构建；Prod 业务 SQL 已安装且普通身份定义/ACL/RLS 校验通过；补偿中继 Active；PC 27 个文件已上传并读回摘要一致。API 001 已正常承载 100% 流量，正式 TLS、API live/ready 和 `/pc/` 27 文件摘要验证通过；业务/真机验收尚未完成。** 按用户最新要求，先沿用 UAT 发布通道完成 Prod 部署，再完成完整业务回归；部署成功不代表业务验收通过。部署前只保留目标/凭据隔离、可构建、SQL差异与权限核对、成本边界及必要回滚检查。

当前已部署的生产方向：已存在的独立Prod CloudBase环境 + CloudRun Go API及同进程worker + Shared PG + 私有COS + 同源HTTP网关/静态托管 + 已备案自定义域名。后续人类明确希望尽量托管CloudBase，晚于8月CVM/MySQL技术选择；[决策原文与源码审计](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/production/report.md)保留来源。旧systemd/MySQL示例不能作为执行命令。保留原始 PRD 与用户明确新增的业务、资金、隔离及核心数据恢复要求；旧技术文档额外引入的产品、数值和采购门禁不再自动继承。

当前完整Goal已授权本项目必要生产配置、最小权限、经核对SQL安装、部署、DNS/TLS、切流和main集成；每项仍须先满足本Runbook前置、核对具体差异并读回。新增费用、采购、启用/扩大计费及真实资金操作另行确认，未获确认不执行。普通身份及匿名publish_key已创建并验证真实登录；业务 SQL 与 API 001 部署、正式域名路由已执行，业务验收仍未完成。

## 最新执行回执（优先于下文历史状态）

详见 [2026-10-15 部署回执](../../openspec/changes/prepare-cloudbase-production/deployment-receipt-20261015.json)。此前条目中的 NOT_CREATED、业务 SQL NOT_RUN、所有生产写步骤 NOT_RUN 和未集成等状态已被本节证据取代；下文历史权限/费用记录仍保留其原始时间和适用范围。

- 已合并 PR #4，main 精确 SHA 为 `149423eac8b179cf453e224534c64f84fe458086`，与提交前候选内容一致；81/81 适配文件已网页读回及 SSH 字节核验。
- 44 条迁移、26 表、16 序列、125 函数实体/118 普通身份可执行已安装；匿名可执行为 0。普通身份返回的原始 catalog 已用该 SHA 的源码验证定义、权限、RLS、身份及扩展。不得重跑首次安装 SQL。
- `order-compensation-wake` 已部署为 Active；API 001 已运行，真实异常唤醒、重启及多实例验收仍待完成。
- PC 26 个资源先上传、HTML 最后上传；正式 `https://order.suianwuye.com/pc/` 27/27 文件摘要一致，浏览器显示主账号扫码登录。扫码授权和商户业务验收未完成。
- 在本人授权的主账号控制台创建专用程序身份 `order-prod-cos-runtime`；策略仅允许 Prod 桶 `images/*` 的 GetObject/PutObject，无控制台登录。实际 STS 获取成功，请求 `1945b415-e7a9-494e-8ab5-5f51c673758a`。旧 `TCB_QcsRole` 的 CAM 拒绝仅适用于该旧调用主体。
- 本人完成钥匙串访问后，正式 AppSecret/支付材料已存至本机 0600 文件并注入 CloudRun；不含于构建镜像。两项订阅模板已从正式 AppID 当前接口核实，后端 formal 已配置；小程序 release 模板已本地补齐，尚未发布或真机验收。
- 免费证书 `baqx2Bd6` 已签发、绑定，截止 `2027-01-13 17:59:59`（控制台时间）；DNS TXT 与 CNAME 已保存。正式域名绑定请求 `3bb2fea9-ae42-4049-aeff-8cd25a578278`，读回 SUCCESS。证书未启用自动续期。
- API BuildId `2608725142`、RunId `multi_tenant_1xHJWNLLXCKypY`、版本 `001`，状态 normal、流量 100%；正式 `/health/live` 与 `/health/ready` 返回预期 JSON。此证据不等于业务 E2E。
- 使用已有标准版预付资源，未购买、升级或变更费用开关；真实资金操作未执行。公众平台网页查询被浏览器站点安全策略拒绝，未绕过，备案/发布资格仍待当前证据；request合法域名已按文末真实登录回执验证，图片上传下载尚待验证。

续接顺序：Prod OWNER 和正式门店已初始化（closed，0 商品/订单，禁止重复） → 完成小程序配置发布与平台资格核验 → 实际业务/图片/异常恢复/备份及真机回归 → 更新 GO/NO-GO。已成功 SQL、身份、DNS 和部署不重复执行。

## 本次消融范围（本地变更，尚未发布）

用户已确认以原始 PRD 为准，无明确要求的自加功能删除。依据 [PRD §6.8、§9.2–9.5](../../docs/product/online-ordering-system-prd-0818.md)，生产图片收敛为服务端唯一 key、禁止覆盖、校验内容后持久确认；不启用 COS 版本控制，不保留 30 天历史版本，不承诺图片误删恢复。现有线上 `149423eac8b179cf453e224534c64f84fe458086` 仍依赖版本号；不能仅改桶开关，须完成代码/SQL适配、精确候选验证与发布后再验收图片。

移除旧技术方案附加的固定 RPO/RTO、指定 CLS/CAT、30 天日志和 3000 人/固定并发与延迟发布门。保留 PRD 核心数据备份与恢复演练方案、5–10 分钟 100–300 单/每日 500–1000 单规模，以及本次用户明确要求的缩略图、三图导入、可靠异常补偿、预算与正式资格核实。删除额外门禁不等于其余未验收项通过。

## 1. 发布前硬前置

本次 `prepare-cloudbase-production` 已按用户确认方向更新 production-architecture-baseline 与 MODIFIED delta，并通过 OpenSpec strict。规格对齐不代表实现、独验或生产发布通过；可靠补偿、平台配置和真实验收仍是硬前置。


| 条件 | 当前证据 / 下一步 | 责任 |
|---|---|---|
| 架构规格同步 | 已本地同步CloudRun/SharedPG/私有COS、身份材料及30分钟异常补偿契约；OpenSpec strict通过，尚无最终候选独验 | Owner/后端 |
| 生产运行入口 | API 001、数据库普通身份及定义/ACL/RLS已验证；图片消融版本未发布，补偿恢复、正式客户端与完整业务验收待完成 | 后端 |
| 最终候选业务验收 | 复用[验收矩阵](../../docs/quality/order-uat-acceptance-20261012.md)，对最终Prod候选完成隔离、目标平台及真机验证；UAT恢复/充值不是前置。通知、真实身份及原生步骤待闭环 | Owner/实际参与者 |
| PRD已知规模 | 5–10分钟100–300单、每日500–1000单；按同形态隔离场景记录实际资源、耗时及资金不变量，不把配额或小样本当容量通过 | 后端 |
| 核心数据备份与恢复 | 保留PRD要求的备份与恢复演练方案；核实已有导出/备份能力，在隔离目标验证可用性，记录实际恢复窗口，不附加固定RPO/RTO或付费平台能力 | 后端/资源持有人 |
| 平台及正式域名 | 分别读回小程序备案、域名ICP备案、认证、类目资质、隐私、手机号余量、支付关联、模板、审核；正式域名/DNS/TLS已验证；小程序平台资格仍未核验 | 甲方管理员/开发方 |
| 正式客户端 | release endpoint已绑定order.suianwuye.com；两模板已实时核实，release 配置待发布；后端必须formal，真实深链可打开对应订单 | 前后端/平台管理员 |
| 正式材料与预算 | 现有Prod自动续费、超额使用已为true；未在本轮改变。确认现有预算/余额/告警责任，不推断无限支出授权 | 客户资源持有人/SRE |

上述容量、真机、完整业务与恢复演练移到部署后验收，不再全部作为首次部署前置；实际缺失的运行材料、权限、安全和费用边界仍须解决。禁止在Runbook里用猜测的UID、域名、密钥或SHA填空冒充可执行配置。

## 2. 资产、身份与版本登记

2026-10-15重新只读核实的Prod（queryEnv RequestId `eea77690-9092-40d8-a62a-14515cc373f1`）：order-prod-d0g4xnw49c5ada5ce，ap-shanghai，标准版baas_pf_standard，Shared PG postgres-gy041jr4；独立于UAT order-uat-d1gtzpzj50615c300。Prod预付到期2027-04-30，API返回QPS配额500，配额不是性能承诺。本次读回与RequestId见[任务账本](../../openspec/changes/prepare-cloudbase-production/tasks.md)；10月12日资源收据仅为历史，不作为本次状态证明。

执行前使用auth status确认账号和目标；queryEnv info显式传Prod EnvId，保留脱敏结果。所有写操作先核对目标Prod，禁止切换错环境。记录以下内容：

- API源码完整SHA、候选独验、构建binary SHA256、镜像digest、BuildId/RunId/版本与流量。
- SQL关系链版本、函数安装摘要、定义/ACL/RLS指纹、执行账本和恢复点。
- PC各文件manifest、对象摘要/缓存版本；Mini exact SHA、上传版本、体验/审核/正式状态。
- Prod独立DB服务UID、桶与最小权限身份、域名/证书、告警接收人、预算责任；秘密只登记私有安全位置及版本引用，不存值。

首次部署前历史读回（已由最新 API 001 回执更新，不作为当前阻断）：CloudRun单独读回：DescribeEnvBaseInfo的 IsExist=true、Status=normal、PackageType=Trial（RequestId `c92b438b-27d1-4b75-881b-3863c3ad9fc7`），而DescribeCloudRunEnvs返回UNAVAILABLE（`346a7293-549d-4b19-a029-4abc6dbae68e`）；这不能与CloudBase标准版混为同一个套餐结论，需继续核对服务、当前可用性及已有权益，不据此直接采购。CloudBase工具当前已绑定Prod；每次操作仍核对EnvId，历史UAT绑定结果不记入Prod。

2026-10-15首次安装前历史读回（不可据此重跑安装）：CloudRun服务0（`57f9d9c7-a1e0-4a70-81d3-c8b12b7a0a06`）；public业务表0、order函数0、PG17.11（`5f43ced6-21ad-4bf8-aba2-4538d29c1dc3`）；网关仅两个默认域名，无正式自定义域名/API路由（`4d9740f9-623b-4b6c-b747-a41fed9490dc`）。这是首次生产安装准备，不是已有生产版本升级。其他条目仍待现场核验；旧UAT收据不转记Prod。

## 3. 生产运行入口与配置准备

后续后端改动需在独立change完成并通过配置隔离Red/Green、隔离SQL、假provider HTTP/worker、秘密扫描和exact候选独验；已有同条件证据复用。具体入口见[源码差异审计](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/production/report.md)：

1. main及SharedPG组合明确production分支，向支付、退款、核销、通知、账单传production，不能伪装uat。
2. cloudbasepg client保留允许环境、publishable JWT audience/project/anon role、普通ServiceUID认证、禁重定向及RPC allowlist；错环境与高权材料必须拒绝。
3. COS STS策略独立绑定Prod私有桶images前缀，保持受限临时身份；禁止把UAT桶或临时对象混入Prod。
4. 材料来源唯一且受保护：DB普通身份、AppSecret、商户支付私钥/APIv3/验证公钥和核销密钥。当前旧CVM Role/SSM和UAT内联方式不直接成为已验证Prod方案；选定并实测轮换后写入最终配置清单。
5. 生产模板的11项订阅配置完整，字段、页面、formal及环境对应；支付/退款回调为正式同源/api/v1路径。后端不得启动时自动迁移/初始化。

当前实现包已部署；后续版本仍须保留配置 guard，并核对新旧版本与当前 SQL 的兼容性。

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

平台扩展已成功安装：pgcrypto 1.3、pg_cron 1.6、tencentdb_scf 1.1；不要重复安装。安装及临时探测清理见[平台回执](../../openspec/changes/prepare-cloudbase-production/platform-prerequisites-receipt.json)。它们不是业务函数，不并入117/113基线。2026-10-15只读核对三者owner均为当前安装身份（RequestId bfa63232-88fc-4c4b-bf4f-4a3b8b9c2964）。首次关系安装已在本地适配已有pgcrypto，仍拒绝不属于该扩展的public对象；生产业务安装已执行，见最新回执；不得重复安装。

API 001 已部署的可靠唤醒实现为事务内记录级状态 + pg_cron + tencentdb_scf + 固定云函数转发至API。生产不依赖 Realtime。SQL安装生成、目录/权限检查、定向RunOne、HTTP adapter及生产运行组合已通过本地隔离测试；有限平台探测已证明数据库提交后实际执行云函数。普通登录续签现已接入扩展加密配置：RPC仅接受网关认证请求头、不接受token参数；多实例按到期时间防止旧令牌覆盖。API维护登录续签，任务仍由数据库持久调度；整个API不可用时不能完成业务，恢复后重新登录并继续数据库中的待办。生产真实异常、重启与多实例业务恢复回归仍未完成。临时探测的普通登录token已清理，不能当作正式持久凭据。

## 5. API、worker与静态发布

已实测可参考通道是UAT CloudRun构建/部署/读回，见[UAT运行手册](order-uat-cloudrun-runtime.md)。Prod执行顺序：

1. 用已独验源码及 `deploy/cloudrun/Dockerfile.production`（专用default-deny dockerignore）构建，禁止使用写死trial模板的Dockerfile.uat。生产镜像不嵌入订阅模板或秘密，由受保护的运行配置注入；锁定Go1.26.5与目标架构构建；镜像非root、端口和health配置与候选一致。生成可回退构建物与配置版本，不把含UAT材料的现存包上传Prod。
2. 使用CloudRun现有部署通道提交一次；记录BuildId/RunId。结果不明先查部署任务和记录，不重复创建版本。
3. 查询服务详情时保持revealEnvParams=false，确认版本、digest、资源、环境键、normal与预期流量；读取live和ready。ready必须验证Prod真实通道、函数、身份与配置；HTTP200不是业务验收。
4. 当前源码只有支付、退款、订阅及账单后台任务。自动制作扫描已移除，制作中由服务端时间派生，取消仅在距取餐严格大于30分钟时允许。production组合已替换支付/退款/订阅每秒扫描，改为数据库持久唤醒后仅执行指定记录；UAT保留历史worker。该实现在 API 001 已部署，生产异常恢复验收仍待完成。可靠补偿只登记真实异常，失败后30分钟重试、成功停止；必须证明持久登记后崩溃、重启及多实例恢复，不能以进程timer或固定30分钟全局扫描替代。保留至少一个常驻实例；并发副本/扩缩配置须来自负载和租约验证，不能机械复制UAT Min1/Max1、0.25CPU/0.5GiB。
5. PC唯一入口为正式origin下 `/pc/`，网关 `/pc` 映射静态托管根；不增加旧地址兼容。将完整 `apps/web-admin/` 内容作为一个版本，资源先上传、`index.html`最后上传，按manifest逐文件摘要读回。业务文件版本、网关缓存和浏览器实际版本对应；记录保存/换页读回/恢复。
6. 正式小程序仅在Prod入口就绪后配置release endpoint/模板、按exact候选上传。上传接受、设体验、审核通过、实际发布分别记账。

新旧实例共存或重启必须实测租约交接、停止领取、inbox/outbox不丢失和外部请求结果不明后的收敛。任务领取数或健康检查不能代替业务吞吐与恢复证据。

## 6. 域名、证书与微信平台

先查已有自定义域名与DNS，复用合法资源；生产PC/API使用已备案自定义域名与有效TLS。HTTP网关API路径透传，PC同源/api/v1，静态路由与缓存规则按真实配置读回。微信request/upload/download只登记实际使用域名，不能关闭合法域名校验。

正式回调分别为正式origin下/api/v1/payments/wechat/notify与/api/v1/refunds/wechat/notify；签名、金额、订单号、AppID、商户及production校验保留。已确认AppID和商户身份不重新索取，也不假定平台普遍要求第二个AppID。

平台管理员按[逐项官方核查表](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/platform/report.md)读回：小程序备案、域名/网站备案分别记录；认证与有效期；自营餐饮类目和经营资质；现网/提审隐私指引；手机号已购额度与余量；支付关联/JSAPI权限；订阅模板字段；体验成员/版本及审核发布。

手机号默认1000次体验额度由开发/体验/正式共用，标准费用0.03元/成功调用；以后台已有资源为准，差额先报告，未经授权不采购。见[微信官方手机号说明](https://developers.weixin.qq.com/miniprogram/dev/framework/open-ability/getPhoneNumber.html)。真实手机号同意、支付确认、原生扫码和平台主体核验均由本人完成。后台安全拒绝时只停该操作，不换入口绕过。

订阅发送miniprogram_state用formal，前端环境名用release；两者不可混填。依次验证provider接受、本地SENT、设备到达、点击正确订单；页面存在不代表订单归属正确，不为通链路放宽鉴权。[微信发送API](https://developers.weixin.qq.com/miniprogram/dev/server/API/mp-message-management/subscribe-message/api_sendmessage.html)

## 7. 经营数据与发布验收

2026-10-15 用户确认沿用已确认 UAT 主账号身份到 Prod；本次只复用姓名/手机号身份，不复用 UAT 权限凭据或示例经营数据。bootstrap 请求 `1803a6d7-c237-4b90-8b20-a459053be5ac` 返回 created；读回 1 个 enabled/unbound OWNER、正式门店 closed、0 商品/订单。后续经真实手机授权完成绑定，不直接写 bound_user_id。

复用已确认店名“绥安食品”、既有地址和唯一取餐点；午11:30截单/11:30–13:30取餐，晚17:00截单/17:00–19:00取餐，30分钟粒度。甲方提供正式商品、图片、售价、可售日期、员工折扣名单及STAFF；通过正常UI保存、离页和后端读回。不要把UAT合成数据上生产。

正式验收按完整UAT映射覆盖：顾客菜单/购物车/报价/折扣/价格漂移/隐私手机号；支付取消与成功；预约、worker、备好；真实相机和重复/跨日核销；顾客与OWNER全额退款及未知结果收敛；通知；PC角色/重登；财务明细、账单来源/一致性；异常恢复。真实资金演练金额/次数另按已获生产授权执行，未知结果先查询，不重复扣款或退款。

按 PRD 已知规模（5–10 分钟集中 100–300 单、每日 500–1000 单）做必要的同形态隔离验证，记录数据基线、到达模型、资源、实际耗时、错误及资金不变量。保留重复/并发及关键链路不受统计任务阻塞的检查；不附加 3000 人、固定并发数或延迟阈值作为上线门。未测试的规模标记未实测，不把资源配额当容量证明。

## 8. 日志、审计与费用核实

使用已有运行日志和业务审计定位故障，关键操作可查询操作者、对象、时间、结果和幂等关系。不得记录姓名/手机号/openid/备注/token/取餐码/密钥或支付原文；按最小必要保留脱敏关联和错误类别。日志脱敏和审计不能消融，指定 CLS、CAT、保留天数及完整阈值告警产品不作为新增发布门。

按用户要求只读刷新预算告警、资源用量、余额、套餐和证书状态；报告已有告警是否配置与接收责任，不把零告警当健康。新增收费日志、监控或费用开关先确认。已有证书续期步骤保留；本次不承诺全天人工值守或未建立的自动告警。

## 9. 核心数据备份与恢复

PRD 要求核心数据有备份与恢复演练方案。先核实已有 Shared PG 导出/备份能力、记录及恢复目标，复用可满足核心数据完整性的已有路径；不继承旧 CDB 的固定保留期、季度频率或 RPO/RTO，也不据此要求升级独享实例。

历史接口证据：2026-10-15 `postgres:DescribeBackupPlans` 被 CAM 拒绝，RequestId `23c688c4-3d3c-4b7b-bde8-04a6d21c968f`。后续主账号在 Prod SQL 数据库备份页确认“当前共享集群暂不提供相关能力，可以升级至独享集群启用能力”。这只证明该平台备份入口不可用，不能直接证明逻辑导出/隔离恢复不可行；尚未升级或采购。见[历史只读回执](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/prod-backup-readback.json)。

操作：选择经实测可用的已有数据导出或备份路径，保存受限工件和来源时间，在隔离目标恢复；核对 schema、函数、ACL/RLS、序列及订单、支付、退款、核销、审计。验证：记录导出一致性、实际恢复耗时、数据窗口和外部资金核对结果；本地合成数据恢复不能替代真实来源工件可用性验证。失败处理：保留失败证据，检查同授权范围内的官方替代通道；无法恢复核心数据时保持该项未完成，不用改状态掩盖失败。费用、架构变更或覆盖生产的恢复须另获对应授权。

COS 图片仅验收私有访问、禁止覆盖、确认内容与实际显示，不再要求历史版本/误删恢复。数据库恢复后须核对回调、补偿及外部资金状态，避免重复退款。消融版仍未部署，线上版本号依赖尚未解除。

## 10. 回滚与故障处理

发布前登记已验证的前一镜像digest、配置版本和静态manifest，证明兼容当前SQL/函数。失败时由唯一发布者查询任务结果，停止继续切流；按CloudRun版本流量管理回到兼容版本，并读回版本/流量/ready、inbox/outbox、订单/退款数量与金额。未经实证兼容的旧镜像不可回切。

数据库不自动down、不清dirty、不删账本；使用经审查的forward fix或已验证恢复。遇支付/退款响应不明只查询供应商和本地投影，再收敛同一业务对象。通知终态不人工重放；健康但业务失败分别查身份、路由、函数和worker，不重复安装全库。

任何身份验证或平台安全拒绝由本人处理；必要材料/资源缺失只停对应链路。记录首个决定性错误、已经通过的条件和下一次重试所需新证据，禁止无变化忙重试。

## 11. 清理与交付完成

按授权清理临时演练身份、测试对象、未用构建物与临时权限；保留真实订单/资金审计。读回清理结果，核对密钥轮换、证书/套餐到期、余额告警和责任人。

最终交付包包括：平台资格/正式域名、API/PC/Mini各自精确版本、资源/密钥安全引用索引、SQL/初始化账本、完整真实业务验收、PRD规模验证与核心数据恢复记录、已有运维状态、回滚回执、经营配置和客户交接。只有实际完成正式发布且原始PRD及用户明确新增的必需项通过，才登记“已上线可交付”。各执行项以本页最新回执为准；未完成的业务与真机验收保持 NOT_RUN。


## 12. 本次发布执行清单（按顺序，不是执行回执）

用户于 2026-10-15 明确要求：涉及加钱必须停下核对。新增付费资源、套餐升级、超额计费开关，以及可能新增账单的恢复演练/图片处理/运行资源，先列明项目、计费方式、预估金额与停止条件，获确认后执行；已有开关状态不视为本次费用授权。

本节为唯一生产操作顺序。未满足验证栏即停止后续依赖步骤，继续无依赖的本地工作。范围内生产写入与切流已获授权，执行前核对具体资源、差异和回退内容；新增费用和真实资金操作仍需另行确认。表中计划不得当作回执；当前已部署 SHA 为 `149423eac8b179cf453e224534c64f84fe458086`，整体验收结论仍为 **NO-GO**。本地进度见 [生产适配任务账本](../../openspec/changes/prepare-cloudbase-production/tasks.md)。

| 序号 / 目标 | 操作 | 验证及交付证据 | 失败处理 / 回滚条件 |
|---|---|---|---|
| 1 固定源码 | SSH读取main/uat；本次基线uat `531638a8961796a7938072463e31ff15dba466ac`，main `8aef176b69239ab9b952bff6f452108da8ad9476`；main无独有提交，UAT领先243。独立适配分支完成后按授权逐文件网页提交；每次读回hash/diff | 文件清单全量核验；干净候选完整SHA；独立detached验证；集成main后视为新SHA重新验证并构建 | 基线变化先比较，禁止覆盖main；缺独验或客户端receipt不得集成；未获授权不写生产分支 |
| 2 只读确认资产 | 显式Prod EnvId检查套餐、配额、服务/版本、账期用量、预算告警、备份计划及恢复点、域名/DNS/TLS、小程序备案/认证/类目/隐私/支付关联和发布资格 | 保存时间、接口、RequestId及脱敏值；网站ICP备案与小程序备案分别有实证；已有预算告警状态与费用责任记录 | 安全拒绝停止该通道，记录权限及资源持有人续接条件；不能以网站ICP备案替代小程序备案，不能自动开费用 |
| 3 冻结材料清单 | 保护配置中准备独立PG普通用户名/密码/UID及对应anon publishable key、Prod COS受限身份、已确认AppID/AppSecret、支付与核销JSON、正式origin、11项订阅配置 | 配置负向测试拒绝UAT EnvId/桶/UID/key、管理key和原生DSN；formal；正式回调精确同源；材料只记录安全版本引用 | 缺值拒绝启动；不可复制UAT身份或凭据；材料泄漏先停发并由持有人轮换，禁止把撤销版本作为回滚材料 |
| 4 冻结数据库差异 | 从精确候选生成关系安装/增量与函数SQL；只读对比当前schema、关系账本、平台迁移历史、函数定义/ACL/RLS；确定仅需执行的差异 | 候选SHA、SQL SHA256、117/113基线与最终集合差异、26表/16序列及普通身份契约；已成功迁移排除 | 不重复成功迁移；结果不明先读回；禁止清dirty/删账本/自动down |
| 5 恢复准备 | 核对核心数据导出/备份与可恢复目标，在授权隔离目标演练；不要求COS历史恢复 | 实际数据窗口和耗时、函数/ACL/RLS/序列、订单/资金/核销/审计一致性；本地PG恢复不算线上证明 | 无可验证恢复路线保持NO-GO；不覆盖生产，不扩大为付费资源采购 |
| 6 安装明确SQL差异〔生产写〕 | 授权后由唯一writer对指定Prod执行已审核SQL；关系与函数分别记账，设置锁/执行超时 | 原子执行结果、普通身份ready、定义摘要、ACL/RLS及平台/业务账本读回；Repair只登记确实已执行的SQL | 未知结果先查，不重复；不兼容则停止应用发布，走审查后的forward fix或已验证恢复 |
| 7 部署API〔生产写〕 | 由集成后精确SHA构建独立生产镜像/包，注入保护材料；部署一次，读回BuildId/RunId/digest/版本/配置键与流量 | live/ready分别记录；普通身份访问成功；跨环境/越权拒绝；重启、多实例租约、补偿成功停止验证 | 任务未决先查；无变化不重发。业务失败停止切流，只有已证明兼容当前SQL的版本可回退 |
| 8 配置图片〔可能费用/写入〕 | 先发布唯一key/禁止覆盖/持久确认的代码和SQL适配；保持Prod私有桶未启用版本控制，核实缩略图能力与已有权益，按最终候选签名读取派生图 | 存量图、新上传、三图导入分别验证：已确认唯一对象、覆盖拒绝、匿名拒绝、过期拒绝、图片内容/裁剪正确；记录原图/缩略图字节、Content-Type、冷/热请求耗时、端侧加载时间 | 403/样式不可用先查授权/对象确认/签名；不得公有化桶或静默退回原图掩盖性能问题；新增费用先确认 |
| 9 发布PC〔生产写〕 | 正式域名 `/pc/`；上传完整静态包资源后最后上传HTML。manifest含相对路径/字节数/SHA256；HTML不长缓存；稳定文件名JS/CSS须重新验证缓存，不能长期immutable | 所有资源摘要与候选一致；Content-Type/Cache-Control/ETag；干净浏览器及已有缓存浏览器实际页面；无旧路径兼容；扫码登录、过期/拒绝/退出、权限检查 | 任一资源缺失不上传HTML；混版本停止对外开放；回退同一完整manifest，资源先HTML后，清理或重新验证缓存 |
| 10 发布小程序〔生产写〕 | Prod入口通过后填写release origin和已核实两模板；合法域名配置与真实上传/下载域名一致；绑定精确候选，上传、体验、提审、发布分别操作记账 | 顾客/商户/PC完整矩阵；release与formal对应；备案/审核/发布资格确认；原生手机号/支付/扫码/订阅由授权本人完成 | release空值或回调未就绪停止；上传成功不等于审核/发布；不关闭合法域名验证绕过 |
| 11 受控资金和通知验收〔资金另授权〕 | 确认具体金额/次数/退款对象后执行付款、支付恢复、退款；通知首次发送及失败补偿；重复/并发与30分钟边界使用隔离夹具优先 | 同一交易/退款唯一事实，观察→物化→页面→外部资金结果；通知provider接受、本地SENT、设备到达、点击正确归属订单分别记录 | 未知资金结果只查询同一业务对象，不重复付款/退款；永久错误停止补偿；无设备到达不得写通知E2E通过 |
| 12 切流与交接〔生产写〕 | 所有必需门禁通过后按批准流量计划切换；已有运维/预算/证书状态、业务结果和版本清单交接 | 最终GO表、精确API/SQL/PC/Mini版本、容量/恢复/资金/业务验收及回滚证据 | 任一必需项FAIL/BLOCKED_EXTERNAL/NOT_RUN为NO-GO；业务故障按第10节，数据库不自动回滚 |

### 12.1 图片实测记录格式

每个“存量图片/新上传/三图批量导入”场景至少记录一次原图及派生图的同网络对照，并重复5次报告中位数与最慢值；小样本不称为p95或容量结论。字段：候选SHA、环境、脱敏对象引用、对象内容摘要、原图字节/派生字节、响应类型、缓存命中情况、请求耗时、端侧可见耗时、失败率、观察时间。临时签名URL不写入Runbook/普通日志。存量图片须校验实际内容及持久确认记录，不能把未验证对象批量改写为已确认；不得为迁移复制UAT样本。

三图导入须覆盖完整预览→确认→提交→离页重读，验证三图顺序、裁剪、重复提交幂等、部分错误与用户可见结果；禁止为测试重导UAT示例或把样本经营数据写入生产。未获生产测试写入授权时先完成本地链路，生产步骤保持NOT_RUN。

### 12.2 本次生产配置新增项

`ORDER_ENV=production`、已核实Prod `CLOUDBASE_ENV_ID` / `ORDER_COS_BUCKET` / `ORDER_TENCENT_REGION`；独立 `ORDER_SHARED_PG_*` 普通身份；`ORDER_PRODUCTION_COS_SECRET_ID/KEY`；`ORDER_PUBLIC_ORIGIN` 为无尾斜杠的正式HTTPS origin。其余AppID/AppSecret、支付/核销JSON和11项订阅键沿第3/6节契约，材料读取使用CloudRun保护EnvParams，不再依赖CVM Role/SSM。生产组合已在 API 001 运行；本节为配置契约，实际发布状态见最新回执。不得携带 `ORDER_UAT_COS_*`、原生DB/管理API key或通用云环境凭据。配置总字节数按当前平台限制核对，秘密不得进入构建物。


### 12.3 精确候选 SQL 生成（本地，不执行数据库写入）

先在最终干净候选worktree核对 `git rev-parse HEAD`。仅当正式UID/origin已读回后设置下列变量；不填测试UID或示例域名。文件输出到仓库外受限目录，记录SHA256及候选SHA。关系SQL和函数SQL不是同一安装工件；生产已完成首次安装，后续必须读回现有目录和账本，仅执行已核对差异，不能执行全量关系安装产物。

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

### 12.4 历史平台验证限制（2026-10-15，非Prod前置）

UAT `order-uat-d1gtzpzj50615c300` 的 `ExecutePGSql` 只读目录查询返回 `instance status must be Running to execute SQL, current: Isolated`。环境总览仍NORMAL、个人版到期2026-10-31、超额开关false；不能据此确认数据库可用或判断隔离原因。暂停依赖此通道的UAT验证，先由资源持有人核实实例状态与恢复条件；若涉及费用，列明报价并经用户确认后操作。禁止为了继续验收直接在生产安装测试表或复制UAT数据。恢复后重新读回实例及普通身份能力，再续接精确候选验证。

随后正常登录新版控制台补充证据：UAT套餐资源点4万/4万，页面明确“已经超额”，累计消耗约4.06万点；资源包0点、按量付费开关关闭。SQL型数据库入口返回res_stopped（RequestId `5b6ae4be-1ccf-42d4-b5a7-4dec3d7ab47c`）。资源耗尽已确认存在，不能继续称为只有未知隔离原因；仍须核实具体恢复报价及账户其他限制。页面640.34点为资源点而非人民币。尚未购买、续费或切换计费；按用户要求停在付费恢复前。

Prod本账期用量接口读回COS 0.01资源点、APIInvocation 0.89资源点，其他列出模块返回0；这是历史计量结果，不是人民币金额、剩余额度或实时服务免费证明。实时连接、CloudRun常驻实例、图片派生和恢复演练的新增费用仍需分别核对。当前未获开启对应费用的授权。

### 12.5 PC静态包本地生成

在干净的最终候选worktree执行 `node tools/pc-release-package.cjs /绝对路径/新的发布目录`。命令拒绝dirty仓库、已有输出目录、旧base、缺失引用及符号链接；生成pc文件树和manifest.json，按清单顺序先上传所有非HTML文件、读回其摘要，最后上传HTML。当前稳定文件名统一要求Cache-Control: no-cache（允许存储但每次重新验证），不能据此声称云端已生效。manifest列出Content-Type、字节数、SHA256和目标/pc/路径。上传工具须显式设置并读回响应头，随后按第9步验证浏览器缓存/登录/页面。

未提交阶段仅可使用 `--draft` 本地检查，产物标记DRAFT_NOT_DEPLOYABLE且candidateSHA为空，不能上传或作为精确候选收据。脚本不上传、不连接后台、不自动修改云端缓存。

### 12.6 历史域名准备回执（部署前；当前已绑定切流）

2026-10-15只读核验后推荐 `order.suianwuye.com`：PC入口 `https://order.suianwuye.com/pc/`，API同源 `/api/v1`。现有UAT `uat.suianwuye.com` 已解析到CloudBase；根域NS为mallard.dnspod.net和caramel.dnspod.net。推荐子域的A/AAAA查询返回NXDOMAIN，目前未发现公开解析占用；这不证明管理权或备案状态。

UAT TLS实读仅覆盖uat.suianwuye.com，有效期2026-10-13至2027-01-11，不能复用于推荐域名。DNSPod DescribeDomain被CAM拒绝（RequestId b361f37f-3343-4c84-baf3-56b1ff9a67a3），停止该权限通道；域名持有人、注册到期、ICP备案、生产证书和绑定资格仍待有权限主体核验。推荐复用已有根域，不购买新域名；实际DNS套餐/证书/网关费用未证明为零，任何加钱先核对。用户随后已确认采用order.suianwuye.com，本地release现已绑定并拒绝UAT origin；此确认不含DNS写入、证书购买或切流。


### 12.7 UAT恢复费用待确认（2026-10-15）

控制台当前最小资源包为1万点×1个，报价10.00元、有效期1年，仅限上海UAT环境，不可转移；使用后余量不支持退订、到期作废。该包不延长UAT基础套餐2026-10-31的到期时间。建议仅在用户确认后单次购买该包，保持按量付费关闭；未下单，不视为恢复成功或全部验收预算已足够。购买后先只读确认实例恢复、资源点余额及持续消耗，再执行已授权的限定验证；再次不足立即停止并核对，不自动追加购买。


### 12.8 历史Prod用量与运维只读快照（2026-10-15）

当前控制台标准版额度33万点、消耗17.05点（PG16.15、存储0.01、环境调用0.89），按量累计0点；自动续费和按量付费均原已开启，本次未改动。页面“数据回档14天”仅为权益，不证明已有恢复点或恢复成功。日志管理尚显示“开启日志”及付费模式确认；未开启。CloudBase平台告警策略列表0条、近7天告警0；账户费用预算已确认0条（见12.9），外部可观测平台策略尚未核实，不得将0告警视作健康或已配置。备案管理只显示Prod可备案；小程序认证页无微信备案及发布资格证明。相关门禁继续未通过。


### 12.9 账户预算与余额（2026-10-15只读）

同一项目账户费用中心：可用余额0.00元、欠费0.00元、余额预警关闭；预算管理无筛选列表0条并显示尚未创建预算。生产按量付费原已开启，但该状态不授权新增支出。该历史快照记录费用风险；后续按用户要求只读刷新预算与余额，新增费用或告警配置须有对应授权；本次未充值、未创建预算、未切换开关。UAT10元资源包仍待用户确认，未下单。


### 12.10 历史Prod主线纠正（部署前）

用户明确Prod上线不以UAT恢复为整体前置。12.7的10元报价仅作历史备选，未获批准且不执行。可靠补偿继续本地实现和隔离验证；目标平台必要验证另列独立对象、权限、费用和清理清单供授权。正式域名已确认order.suianwuye.com，本地release已配置，DNS/TLS及平台验收仍未完成。

## 历史：2026-10-15 Prod普通身份创建回执

- [x] 用户了解用途后明确“那你去创建吧”，授权创建Prod专用普通身份。
- [x] 创建前Name查询0条，RequestId `77f8ecd8-c8fa-4e0a-a40e-53af8ae916aa`。
- [x] CreateUser返回UID `2110616903056039938`，RequestId `9be02697-d68a-4ecd-b59a-25a31aa52b13`。
- [x] DescribeUserList读回1条：环境 `order-prod-d0g4xnw49c5ada5ce`，Name `order-prod-api`，Type `internalUser`，状态 `ACTIVE`，UID一致；RequestId `9c05eca3-50a0-4ce0-83a3-63917987be78`。
- [x] 独立随机32字符密码保存于仓库外 `/Users/vivix/.config/order/production/order-prod-api.json`，目录0700、文件0600；材料附创建/读回回执，密码不入仓库。
- [ ] 普通身份登录、RPC权限及跨环境拒绝尚未验证。本回执不证明SQL安装或部署成功；未授业务表权限、未安装SQL、未切流、未购买。

## 历史：2026-10-15 Prod匿名key与普通身份登录回执

- 用户明确批准创建publish_key并验证登录。CreateApiKey仅使用KeyType=publish_key，未创建管理员api_key；KeyId `3znAvPKBTAC-OiW38En_lw`，创建RequestId `95afe11a-e2d4-4732-98fb-6484a8db8b20`。
- key的role=anon，aud/project_id均为Prod环境；已存入同一仓库外0600材料文件。
- 2026-10-15 14:25:04北京时间，真实Prod auth/v1/signin验证通过：返回sub与UID `2110616903056039938`一致、role=authenticated、aud/project_id均为 `order-prod-d0g4xnw49c5ada5ce`、Bearer且有效期7200秒；RequestId `bd11369a-c087-4887-a394-1a39b0034074`。核对了实际响应身份，不仅HTTP状态；访问令牌未落盘或写入文档。
- 此前“缺publish_key/登录未验证”已解除；数据库RPC、ACL/RLS、业务与真机仍未验收。无SQL安装、业务授权、切流、采购或资金操作。

### 历史本地退款补偿进展（2026-10-15，部署前）

创建请求失败改为30分钟后补偿；正常PROCESSING创建、查询及重复观察不再安排下一次查询；due/claim不将NULL补偿时间视为普通待办，同时保留过期租约恢复。隔离PostgreSQL回归通过，未调用真实微信退款。创建结果不匹配已改为人工待处理且不自动排队；可靠跨实例唤醒和通知首次发送仍未完成；不可仅据此部署为完整可靠补偿版本。

### 12.11 正式证书续期与交付

目标：避免证书到期中断正式 PC、API、支付退款回调。当前证书 baqx2Bd6 到期 2027-01-13，免费证书不等于自动续期服务。

操作：到期前至少 30 天核对官方免费申请资格，由持有者按现有主账号权限申请新证书并完成 DNS 验证；签发后将 CloudBase 自定义域名绑定至新证书，保留原证书到新绑定验证结束。不新购付费服务。

验证：从公网验证主机名、信任链、新到期日及 /pc/、live/ready，记录新证书 ID 和绑定请求。告警接收人与续期执行责任尚待登记，不把本段步骤当作已建立自动告警。

失败处理：DNS/签发失败保留仍有效旧证书；新绑定导致 TLS 错误且旧证书未过期时回切旧证书并再次验证。若免费额度或平台能力要求付费，停止并确认费用。官方说明：https://cloud.tencent.com/document/product/400/61353 。

### 12.12 历史平台验收缺口（2026-10-15，域名修复与消融前）

以下保留当时事实和请求，不是当前执行指令。request域名阻断已由下一节解除；版本控制/30天历史保留请求已由本次消融撤销，不再等待该项付费授权。缩略图实测和图片上传下载仍待验收，新增费用仍须确认。

- 微信模拟器受控使用 release origin，真实 wx.login 后请求被 `request:fail url not in domain list` 阻止，未抵达会话接口。原生环境仍是 develop，该结果不是正式版或真机验收。已请管理员在公众平台补 request/uploadFile/downloadFile 合法域名；浏览器站点安全策略拒绝 mp.weixin.qq.com，未绕过。
- Prod COS 原生 SDK 只读确认 ACL 仅 owner FULL_CONTROL；版本状态为空（未启用），生命周期返回 NoSuchLifecycleConfiguration。版本控制/30天历史保留及缩略图验收可能增加计费用量，等待用户确认不超过1元的本次验收额度后执行。
- 正式接口未登录读取顾客订单、商户订单和身份均返回401；公开 storefront 返回确认过的门店及closed。此为真实平台鉴权和初始化检查，不等于业务验收。

### 2026-10-15 服务器域名配置后追加核验

用户确认公众平台域名已配置。当前项目重新编译仍读旧白名单；仅关闭并重开生产测试项目后，保持 `urlCheck=true`，正式 origin 的 `wx.request /health/ready` 返回 200，真实 `wx.login → Prod API` 会话 ready。身份读取为未绑定手机号的 VISITOR；顾客订单读取 200 空列表，商户订单读取 403，首页门店设置和订单状态均 ready，显示绥安食品、休息中。

证据：`prod-domain-login-verification.json`、`prod-domain-login-home.jpg`（本次 production-deployment 证据目录）。这是 develop 模拟器通过受控 release endpoint 覆盖调用真实 Prod 的验证；不是正式包、真机、OWNER 绑定、支付退款通知或图片上传下载全链路通过。此前 request 域名阻断已解除；uploadFile/downloadFile 及真实图片链路仍未验证。未开启版本控制、图片处理或新增付费功能。
