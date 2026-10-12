# Order CloudBase 生产发布 Runbook

更新：2026-10-12，北京时间。**当前不可直接发布，生产操作均为 NOT_RUN。** 本文把已实测的UAT通道、现有Prod资产与尚缺的生产实现分开。执行人必须先关闭第1节硬前置，再按顺序操作；不得把UAT配置换一个环境名直接上线。

唯一后续实施方向（不是当前已生效的生产实现）：已存在的独立Prod CloudBase环境 + CloudRun Go API及同进程worker + Shared PG + 私有COS + 同源HTTP网关/静态托管 + 已备案自定义域名。后续人类明确希望尽量托管CloudBase，晚于8月CVM/MySQL技术选择；[决策原文与源码审计](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/production/report.md)保留来源。旧systemd/MySQL示例不能作为执行命令。业务、资金、隔离与恢复合同保持，不以技术迁移降低验收。

本轮只有只读Prod核查和交付准备；实际安装、配置、部署、切流和发布必须在后续明确生产授权窗口执行。费用开关、采购和真实资金试验分别按已获授权边界执行。

## 1. 发布前硬前置

当前production-architecture-baseline规格尚未同步后续托管方向。本W0只整理证据与执行顺序，不以Runbook覆盖现行spec：生产适配change必须先通过MODIFIED架构规格及依赖对齐，替换CVM/MySQL、迁移锁、身份/材料、观测和恢复产品名，保留相应业务安全不变量，再进入实现和发布。该工程手续按既有人类托管方向执行，不要求甲方重新选择数据库；完成前禁止把下文当作绕过规格的执行授权。旧CVM入口也不因其仍写在spec中而恢复为可部署实现。


| 条件 | 当前证据 / 下一步 | 责任 |
|---|---|---|
| 架构规格同步 | 生产适配change先更新production-architecture-baseline及对应依赖；本W0不替代该规范变更，未完成不得实现/发布新路线 | Owner/后端 |
| 生产运行入口 | 当前main仅ORDER_ENV=uat走SharedPG；client、COS STS及多个provider固定UAT。须完成生产独立资源绑定、环境语义、材料注入与负向隔离验证，生成exact候选并独验 | 后端 |
| 完整UAT | [UAT验收](../../docs/quality/order-uat-acceptance-20261012.md)仍未全PASS；体验版、通知、真实身份及原生步骤待闭环 | Owner/实际参与者 |
| 3000人容量 | 没有通过证明；菜单小样本p95约981ms，高于500ms读目标。按业务高峰、HTTP/RPC、worker、数据库、发布并存测量，不把配额当吞吐 | SRE |
| Shared PG备份与恢复 | 本地PG16恢复成功不能代替线上；必须查明当前实例计划/恢复点/保留期/恢复目标与权限，并在隔离目标演练。未证明RPO≤5分钟、RTO≤2小时 | SRE/数据库执行者 |
| 平台及正式域名 | 分别读回小程序备案、域名ICP备案、认证、类目资质、隐私、手机号余量、支付关联、模板、审核；正式域名/DNS/TLS未有完整实证 | 甲方管理员/开发方 |
| 正式客户端 | release endpoint和两模板目前为空；后端必须formal，真实深链可打开对应订单 | 前后端/平台管理员 |
| 正式材料与预算 | 现有Prod自动续费、超额使用已为true；未在本轮改变。确认现有预算/余额/告警责任，不推断无限支出授权 | 客户资源持有人/SRE |

这些条件未通过时停止发布部分，继续独立本地验证和材料准备。禁止在Runbook里用猜测的UID、域名、密钥或SHA填空冒充可执行配置。

## 2. 资产、身份与版本登记

已只读核实的Prod：order-prod-d0g4xnw49c5ada5ce，ap-shanghai，标准版baas_pf_standard，Shared PG postgres-gy041jr4；独立于UAT order-uat-d1gtzpzj50615c300。Prod预付到期2027-04-30，API返回QPS配额500，配额不是性能承诺。完整读回见[资源收据](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/platform-live-readback.json)。

执行前使用auth status确认账号和目标；queryEnv info显式传Prod EnvId，保留脱敏结果。所有写操作先核对目标Prod，禁止切换错环境。记录以下内容：

- API源码完整SHA、候选独验、构建binary SHA256、镜像digest、BuildId/RunId/版本与流量。
- SQL关系链版本、函数安装摘要、定义/ACL/RLS指纹、执行账本和恢复点。
- PC各文件manifest、对象摘要/缓存版本；Mini exact SHA、上传版本、体验/审核/正式状态。
- Prod独立DB服务UID、桶与最小权限身份、域名/证书、告警接收人、预算责任；秘密只登记私有安全位置及版本引用，不存值。

只有env和PG存在已证实；其他条目现场核验。旧UAT收据不转记Prod。

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
- 函数链：services/api/sharedpg.InstallSQL接收真实Prod serviceUID、production及两个正式回调URL，生成经审查产物；核对113实体/109可执行函数与4旧入口撤权是否仍为本候选定义，不把历史数字硬写为将来必过值。
- 管理通道必须以原子执行结果、定义摘要和真实账本读回确认成功。PushPGUserMigrations不是唯一执行前提；平台Repair仅可登记已经成功的精确SQL，不能修饰失败。
- 普通服务身份验证RLS、SECURITY INVOKER和允许RPC；匿名/非允许身份不能直接访问业务表或管理函数。确认退款、核销与支付回调不跨环境。
- 初始化沿SharedPG bootstrap五参数逻辑，首次创建真实OWNER和门店；native order-bootstrap不能直接冒充SharedPG执行命令。先查created/unchanged及冲突，未知结果只读回、不重复初始化。
- 不复制UAT人员、订单、低价菜、许可或账本进Prod。已存在真实数据时以增量方案和恢复点执行。

安装receipt必须包含目标EnvId、exact候选、SQL摘要、提交/完成时间、最终账本、定义和权限指纹；不含手机号、token或密钥。未取得明确结果不继续启动业务。

## 5. API、worker与静态发布

已实测可参考通道是UAT CloudRun构建/部署/读回，见[UAT运行手册](order-uat-cloudrun-runtime.md)。Prod执行顺序：

1. 用已独验源码、锁定Go1.26.5与目标架构构建；镜像非root、端口和health配置与候选一致。生成可回退构建物与配置版本，不把含UAT材料的现存包上传Prod。
2. 使用CloudRun现有部署通道提交一次；记录BuildId/RunId。结果不明先查部署任务和记录，不重复创建版本。
3. 查询服务详情时保持revealEnvParams=false，确认版本、digest、资源、环境键、normal与预期流量；读取live和ready。ready必须验证Prod真实通道、函数、身份与配置；HTTP200不是业务验收。
4. 五类worker同进程：支付、退款、订单推进、订阅和账单。保留至少一个常驻实例；并发副本/扩缩配置须来自负载和租约验证，不能机械复制UAT Min1/Max1、0.25CPU/0.5GiB。
5. PC产物按manifest上传，逐文件摘要读回。业务文件版本、网关缓存和浏览器实际版本对应；记录保存/换页读回/恢复。
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

本轮只读实测确认Prod底层PG17.11实例running、REST active；DescribeBackupPlans被CAM拒绝，操作postgres:DescribeBackupPlans，requestId 23617329-099e-4514-95b0-bc49b61c2bec。没有绕过拒绝或修改权限，不能据此写无备份。续接由资源持有人提供现有计划/记录的只读结果，或明确授权最小查询权限；见[实际阻断收据](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/prod-backup-readback.json)。

执行人先只读实际备份计划、成功记录、恢复点、保留期、权限、可恢复目标和费用。在同产品隔离目标恢复，不覆盖当前生产；核对schema/函数/RLS/ACL/序列、订单/支付/退款/核销/审计及外部资金状态。记录恢复点、开始/结束和丢失窗口，再判断目标。不存在可证明的恢复路径则保持NO-GO，报告失败的合同和需要的外部条件，不降低目标。

COS单独验证版本/保留、误删恢复和私有访问，不能用上传下载成功替代恢复。恢复后重新核对回调/worker和财务，避免重放已发生的退款。生产数据恢复切换需后续明确授权。

## 10. 回滚与故障处理

发布前登记已验证的前一镜像digest、配置版本和静态manifest，证明兼容当前SQL/函数。失败时由唯一发布者查询任务结果，停止继续切流；按CloudRun版本流量管理回到兼容版本，并读回版本/流量/ready、inbox/outbox、订单/退款数量与金额。未经实证兼容的旧镜像不可回切。

数据库不自动down、不清dirty、不删账本；使用经审查的forward fix或已验证恢复。遇支付/退款响应不明只查询供应商和本地投影，再收敛同一业务对象。通知终态不人工重放；健康但业务失败分别查身份、路由、函数和worker，不重复安装全库。

任何身份验证或平台安全拒绝由本人处理；必要材料/资源缺失只停对应链路。记录首个决定性错误、已经通过的条件和下一次重试所需新证据，禁止无变化忙重试。

## 11. 清理与交付完成

按授权清理临时演练身份、测试对象、未用构建物与临时权限；保留真实订单/资金审计。读回清理结果，核对密钥轮换、证书/套餐到期、余额告警和责任人。

最终交付包包括：平台资格/正式域名、API/PC/Mini各自精确版本、资源/密钥安全引用索引、SQL/初始化账本、完整真实业务验收、容量与恢复报告、告警到人、回滚回执、经营配置和客户交接。只有实际完成正式发布且全部必需项通过，才登记“已上线可交付”。本轮所有Prod执行项保持NOT_RUN。
