# Order 生产交付与当前上线缺口

> 2026-10-12 Owner接管：当前结论见[全局UAT验收](order-uat-acceptance-20261012.md)，后续生产以[CloudBase生产Runbook](../../deploy/runbooks/order-production-release.md)为唯一执行入口。以下运行收据保留原范围；完整UAT与生产均未PASS。本轮必要局部验证已获授权，历史停测不再约束当前验证。

> 更新日期：2026-10-12（北京时间）；当前结论以uat-delivery-status.json及最新N1/N2/N3、财务、员工、通知专项回执为准，首次001/bf/两笔财务均保留为历史。
> 本文替代 2026-08-24 的待办快照；旧候选、旧测试总账、旧工期与“尚无 UAT URL”等结论不再作为当前执行依据。
> 产品基线：[PRD](../product/online-ordering-system-prd-0818.md)。生产批准基线：[production-architecture-baseline](../../openspec/specs/production-architecture-baseline/spec.md)。

只记录非敏感配置、版本、状态和证据索引。密码、AppSecret、API v3 key、私钥、证书正文、Cookie、Token、真实手机号和员工名单不得进入文档或 Git。

## 1. 当前交付结论

UAT已走通131/133顾客退款、133预约仅查看、132价格/两级备注/自然worker/真手机相机核销，以及员工折扣、原生支付取消和财务。**完整18场景仍未PASS，Prod没有实际发布完成证据。** 当前通知点击/新发送Green、真实身份参与者、同132重复UI、最后PC正常OWNER重登（退出及保护页已PASS）、体验版选择保留，不重跑已通过工作。

| 项目 | 当前已证实结果 | 边界 |
| --- | --- | --- |
| 环境与身份 | `order-uat-d1gtzpzj50615c300` / `ap-shanghai` / public AppID `wx15e136b256d4dc5a` | `APPID_ERROR` 已恢复，保留原已确认身份；不作为当前阻断 |
| 数据库 / 平台登记 | 113 个实体、109 个可执行函数，4 个旧入口撤权；原生账本 v44、平台记录 38 条保持 | 已成功 SQL 和 bootstrap 不重做；后续源码修复不重装数据库 |
| API | `a833507a809decd718729501d2bab96cef2d0f35`；CloudRun `order-api` / `004`，normal、流量100%、HasTraffic，10/12 01:35 live/ready200 | opaque msgid兼容已实际部署；新通知SENT/设备点击Green未观察，就绪不代替业务结果 |
| PC | 初始3709917的24文件包、finance.js来源7fe4de8已发布；Admin87的app.css/app.js/settings.js三文件已发布，公告保存→换页读回→清空恢复PASS | 不虚构统一PC发布SHA；原finance切片保持，api700营业日修复见唯一登记点 |
| Mini / 根配置 | `b3f1f6da24935a9a46e4615b87374a56027b70af`；136 源文件精确匹配，保留既有 `project.config.json`，发布项目共 137 文件；原 AppID、合法域名校验开启 | 组合候选独立验证通过；实际 UAT 项目与上传物分别核对 |
| Mini 上传 / 预览 | 开发版本 `uat-20261012-b3f1f6d` 于 02:02:46 上传接受；02:02:36 默认首页预览已生成，完整二维码已查看 | 当前版本尚未选择正式体验版；67efa 真手机相机核销证据保留，不写成 b3 真手机全量验收 |
| b3 交互检查 | 实际 A 单价10分×2=20分预估；修改菜单后返回确认，姓名/整单备注和已绑定手机号保留；商户退款列表及133查看入口正常 | 未产生新 quote/prepay/order 或资金操作；制作中实际0单，备好入口仅复用精确候选 UI2 证据 |
| N1 顾客退款 | 131 / `20261012-0001` 用户付 10 分，正常顾客确认取消；refund111 为 10 分 `SUCCESS / APPLIED` | 顾客退款链已通过；退款成功不等于退款通知送达 |
| N2 价格 / 备注 / worker | 132 / `20261011-0002` 用户付 30 分；A 单价从 10 分改为 15 分后数量 2，少饭、菜品备注、整单备注真实冻结；23:20 自然推进制作中，23:33 正常备好 | 复用同单价格、备注、worker 与凭证证据，不再付款或手工推进 |
| N2 真手机相机核销 | 用户反馈「是直接开相机，已核销了」；132 于 10/12 00:17:33.179504 `COMPLETED`，redeemed_at 相同 | 入口直接开相机、真实 QR 识别和同单完成主链 PASS；只观察到一次成功核销审计，重复扫描次数/新版幂等页面未明确 |
| 员工折扣 | 80% 下 D 原价 21 分→17 分，数量 2，服务端 quote 原42/减8/应34分；停用后正常刷新回原42分，空名单及100%已恢复 | 未支付员工单；金额格式修正已随 b3 上传，8 分显示「-¥0.08」由精确候选隔离 UI2 验证；现场员工价不重跑 |
| 财务 / 收款 CSV | N3前四笔快照收43分、退12分、净31分，实际收款CSV4行/358B已核对；133之后今日PC收1分/退1分/net0 | 四行CSV保留为历史有效证据，本次未重导出大范围；退款汇总来自PC，不伪称退款CSV或微信账单下载证明 |
| 10/10 正式账单系统核平 | 既有 audit967 `SUCCEEDED / BILL_MATCHED`，matched=3，provider_only/system_only/ownership_unknown 均0 | 系统核平已观察；单次 HTTP 账单下载回执未见，早于10点的来源未独立解释；10/11 账单不手工重跑 |
| 通知 | READY0002用户已收到，但点击提示不存在；133 outbox15前两次HTTP200/errcode0含msgid被旧严格解析拒绝，第三次USER_NOT_SUBSCRIBED永久失败 | a833兼容已部署；新本地SENT/设备点击Green未观察，不重放13/14/15，不由15倒推old14原因；CLS按窗口关联、tag无outboxid |
| N3 预约 / 顾客退款 | 用户付1分→133 / `20261012-0002`；预约商户卡仅查看、无备好（case15 PASS）；用户确认取消→refund112 `SUCCESS / APPLIED`，顾客REFUNDED无取餐码；H14 OFF | 预约与退款主链通过，通知发送/点击不能据此登记 |
| 原生支付取消 | case11复用用户本人关闭未付0.34元二维码、197无业务订单/物化及后续正常UI与新支付证据，主路径PASS | wx取消回调未观察单列，不升级额外门禁或重跑要求 |
| 自动测试 / 完整验收 | 按用户要求停止套件/回归/Gate；已通过场景复用，完整18场景未记PASS | 剩余通知点击/新发送Green、真实子账号/未绑手机号、同132重复UI、最后PC正常OWNER重登（退出及保护页已PASS）、体验版选择 |

PC营业日缓存修复的候选、独立复核、实际单文件发布与真实售罄Green统一维护于[UAT部署Runbook唯一登记点](../../deploy/runbooks/order-uat-deployment.md#pc-营业日修复唯一状态登记点)。本页不复制临时pending版本，避免后续状态漂移。

实际版本分别绑定：APIa833/004（BuildId2608516656、RunId multi_tenant_1xFxN39xmNNpKS）；Minib3/136源码加既有根配置、开发上传及实际页面结果，旧67efa真手机核销证据保留；PC初始3709917包加finance7fe、api700及Admin87三文件。源码review、发布、平台状态与业务Green分别报告，不虚构统一SHA或整包重部署。

### 当前证据

- [当前总账](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-delivery-status.json)、[剩余场景清单](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/admin-remaining-20261011.md)
- [APIa833 /004实际发布](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-notification-msgid-deployment-a833507/deployment-receipt.json)
- [Admin87三文件发布/公告恢复](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/admin-uiux-87dcc0b-publication/publication-receipt.json)、[N3退款/通知实证](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/notification-N3-133-server-readback.json)、[顾客实际演练/原生取消197](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/customer-execution-receipt.json)、[PC实际配置/财务](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-case-execution-receipt.json)
- [Mini b3 实际交付](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/customer-b3-combined-UAT-delivery-receipt.json)、[新版默认首页预览](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/uat-preview-b3f1f6d-combined.jpg)、[组合候选独立验证](/Users/vivix/.codex/visualizations/2026/10/11/01a12bbb-ec58-79a3-bc80-7ad3d2d599d9/uiux-delivery/combined-b3-independent-receipt.json)；[67efa 真手机相机核销132](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-N2-camera-actual-phone-green.json)保留复用
- [四笔财务/实际收款CSV核对](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-finance-four-orders-receipt.json)、[4行收款CSV](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-finance-20261010-12-four-orders.csv)（358 B，SHA256 `35c7e1dd76062de516c72c1c3198282c521a32eef14038cba83e51dbe9a8e80b`）
- [账单967系统核平](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-bill967-system-readback-20261011.json)、[下载证据边界](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/admin-bill967-provenance-note.md)、[财务标签单文件发布](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/finance-label-publication-20261011.json)

<details>
<summary>10/10–11 首次交付与早期演练快照（历史，不能作为当前未完成项）</summary>


UAT 已实际完成部署、OWNER/PC 登录、菜单配置/过滤、用户真实付款、支付返回结果收敛、备好/手工核销、预约单退款、财务及营业配置恢复。1 分订单 130 为 `COMPLETED`，2 分订单 129 为 `REFUNDED`，退款 `SUCCESS / APPLIED`；当时 PC 总收 3 分、退款 2 分、净额 1 分。重复核销仍显示成功页，已核销提示未通过；相机扫码、通知与账单核平没有完成证据，完整 E2E 未记 PASS。**Prod 尚无实际发布完成证据，不能把 UAT 已完成业务片段等同于正式营业可用。**

本轮按用户要求停止测试、回归和 Gate；交付分支已删除 485 个测试用例、纯测试辅助和 runner 文件。不自动恢复，不以历史 PASS 代替本轮结果；Prod 上线后的 smoke 由用户重新安排。当前 UAT 演练记录实际页面、订单和服务端结果，文档只登记已发生的操作。

| 层次 | 已证实结果 | 证据边界 |
| --- | --- | --- |
| SQL 安装 | 113 项安装闭合；109 项可调用、4 个旧入口撤权；原生 v44 保留；平台历史 37→38 | 直接安装、事务/权限/读回已完成；不再等待失败的 PushPGUserMigrations |
| UAT 后端 | CloudRun order-api/001，构建 2608451769；状态 normal、流量 100%；实际 live/ready 200 | 不代表真实支付、退款或业务结果 |
| PC 实际业务 | 24 文件已部署；内置浏览器真实认证 OWNER，菜单、订单、退款及财务实际操作已记录 | 不重复登录/绑定；通知等剩余场景分别记录 |
| 小程序实际上传/预览 | `bf949f75ff8e7d6e3966d07d0528513057bf8532` / `uat-20261010-bf949f7` 已编译、上传 success=true；默认正常首页完整预览图已生成/查看 | 135 文件精确匹配；上传不代表正式体验版已选择，预览 wrapper 截断不登记成功 JSON |
| 顾客链路/支付返回 | 27d0289 实际报价/预支付 201；bf 原生支付返回后实际自动进入 result=ready、“订单已创建”、同一订单 130，旧调试弹窗 pending 被实际结果替代 | 结果收敛已观察，确切 wx success/fail 分支未知，不分别记 PASS；顾客操作身份是已绑定 OWNER，不是独立访客 |
| 手机启动 | 7 处 Object.hasOwn 兼容改动已交付，用户随后报告“看起来没问题了” | 初次故障根因未最终确认；仅记录用户报告的启动恢复 |
| OWNER / PC 原生确认 | 已绑定、正常商户入口及 PC 实际 OWNER 登录均完成 | 电脑原生工具历史 `-10005: timeoutReached` 不再作为当前登录阻断，不重复绑定/扫码或伪造认证 |
| 预约付款/退款 | 预支付 192、2 分为 `PAID / APPLIED`；订单 129 曾预约 10/11 11:30，现 `REFUNDED`；退款 110 为 `SUCCESS / APPLIED`，MATCH CALLBACK 到账 19:46:57 | 用户本人付款与确认退款；历史 PROCESSING/DEFERRED 不当当前失败，不重复资金操作 |
| UAT 数据 | 示例店已开业，2 个示例商品（1/2 分），7 个开放服务日 | 非正式经营数据，不复制到 Prod |
| 多场景菜单 | PC 新增 3 分类/7 商品；实际 10/11 午餐 A/B/D/F、晚餐 A/C/D/F，E/G/分类 14 隐藏；10/10 F 售罄不可购，临时 19:50/20:20 可见 | 10/11 F 实际加购 1 份 0.50 元、cart canCheckout 后清空；员工折扣未验证；保留原 1/2 分示例，全库 4 类/9 菜，不重新导入 |
| 财务 / 导出 | PC 收款 2 笔 3 分、退款 1 笔 2 分、净额 1 分；历史待处理页 0；最终收款导出 210 B、2 行 0.02/0.01 元 | 退款 tab 导出仍返回收款 CSV，退款 CSV 未观测；未证明微信账单核平；首笔导出保留为历史 |
| 10/10 1 分履约 | 预支付 193 `PAID / APPLIED`；订单 130 paid 19:40:34 → preparing 19:40:37 → ready/code-issued 19:44:52 → 手工 redeemed/completed 19:47:22 | 实际 UI/DB 一致；重复码仍显示成功页，提示 case 未通过，19:52:48 时间未改；相机扫码未执行 |
| 10/11 新确认切片 | A 选 2 份、少饭及菜品备注后，点击确认出现 `ETIMEDOUT / APPID_ERROR`，该切片停止 | 整单备注/新报价未完成；保留原 public AppID `wx15e136b256d4dc5a` 及既有通过结果，不改身份配置 |
| 通知 | 19:48:39 订单 129/130 consent/outbox 均空 | 没有实际授权、provider 接受或设备送达证明；真实模板配置不代替送达 |
| 营业配置恢复 | PC 保存/换页读回，19:50:56 DB 确认原午晚餐/step30/取餐点/空公告/storeopen、10/11–17open、10/10closed | 原 10/10 无日期行，现 closed 行并有审计/版本变化；业务安排已恢复，不宣称字节一致，不复制 UAT 数据到 Prod |
| 私有 COS | 已有真实签名上传/读取、权限及清理回执 | 复用已通过证据；不等于 Prod 桶、备份恢复或页面全链路已完成 |
| 历史退出/布局缺口 | PC 已实现服务端 logout；个人中心折叠表单、输入/按钮布局已修 | 两者实际用户结果未观测；Mini 仍只有“切换身份”，服务端注销存在但客户端未接入口，不能称为真正注销 |

实际版本分别记录，不能虚构成一次同 SHA 发布：

- 后端：`57ed75f7f8df30c4f7693a1e80dc8b8148e3f88b`。
- PC 静态站：`370991794227d2b5e9f1c6eb7b163814b5cb9d13`。
- 当时实际小程序上传：`bf949f75ff8e7d6e3966d07d0528513057bf8532` / `uat-20261010-bf949f7`；19:12:04 平台上传成功，19:13:58 默认 launch 预览完整图已生成/查看。
- bf 实际原生支付返回已收敛 result=ready、“订单已创建”、同一订单 130；具体 wx success/fail 分支未知。报价/预支付 201 的早期证据绑定 `27d02898620c74df247b5122031e7419b4424030`，不重复资金操作。
- 当时交付分支：`codex/order-uat-cloudrun-runtime`；本地提交与部署回执不等于远端 main 已集成。

UAT origin：`https://order-uat-d1gtzpzj50615c300-1473835269.ap-shanghai.app.tcloudbase.com`。
PC 入口：上述 origin 的 `/pc/web-admin/index.html`。
操作续接使用 [UAT 部署 runbook](../../deploy/runbooks/order-uat-deployment.md) 和 [CloudRun runtime runbook](../../deploy/runbooks/order-uat-cloudrun-runtime.md)。


</details>

## 2. 生产路线已收敛，代码与恢复前置尚未完成

8月工程规范曾要求CVM/Nginx/systemd/MySQL8；后续人类明确要求尽量托管CloudBase，且本轮只读确认独立Prod标准版与SharedPG已存在。Owner据此选定CloudRun＋SharedPG＋私有COS的唯一后续方向；生产适配change必须先同步现行架构spec，本W0不覆盖仍有效的规范。该方向不重新采购旧技术资源，不降低资金/隔离/恢复合同。原文与实际资产见新Runbook链接。

当前SharedPG组合仍只允许ORDER_ENV=uat，client、COS策略及多个provider固定UAT。非UAT路径已是原生PostgreSQL，旧MySQL env与CVM恢复手册不适用于当前发布。必须补生产资源绑定、材料来源、安装/初始化入口与独立验证，不能只改环境名或复制UAT凭据。

本轮未改生产源码/配置，未生产部署。新Runbook的硬前置明确标出生产适配、完整UAT、容量、平台资格、SharedPG恢复及真实发布授权；当前不可直接执行发布。

## 3. 距离正式上线的具体工作

| 工作 | 当前状态与最小完成结果 |
| --- | --- |
| 生产方案、代码与资源 | 按第 2 节路线补生产实现；确定与 UAT 隔离的生产身份/数据/资源，补对应 runtime 和完整非敏感配置，安装实际 schema，记录最终版本及构建物。生产资源当前没有完成回执，不能推断为不存在。 |
| 正式域名与 HTTPS | 绑定自定义域名、DNS、证书及适用备案；配置实际所用微信 request/upload/download 域名、PC origin、支付/退款回调。CloudBase 默认域名仅开发测试，不能作为生产浏览器 Admin 入口。 |
| 正式客户端和通知配置 | 小程序 [release endpoint](../../apps/wechat-miniprogram/utils/runtimeEndpointConfig.js) 及 [release 两类订阅 ID](../../apps/wechat-miniprogram/utils/subscriptionTemplateConfig.js) 仍为空；develop/trial 两类真实 ID 已对齐并随 cc45be2 上传。UAT 后端 11 项通知配置为 trial；READY已由用户确认收到但点击不存在，退款通知本地SENT/设备结果未闭环。Prod 必须与正式前端模板/字段对齐，并使用 formal。 |
| 生产凭据与权限 | 按最终方案配置本环境的身份、最小权限、私有 COS 和材料存储。生产材料来源须按新CloudRun入口验证；当前非UAT CVM/SSM路径尚不能作为该路线实现。数据库身份、AppSecret、完整支付材料和核销密钥保持隔离，不能把UAT内联配置复制成production。 |
| 微信平台状态与发布 | 当前 AppID wx15e136b256d4dc5a、商户号 1750169200 已提供，不再索取。认证、备案、实际类目/资质审核、隐私指引、真实手机号/扫码能力、模板审核、商户关联、提审及发布状态须读取当前后台；现阶段记“待核实”，不记“未申请”。 |
| 实际营业链路 | UAT132相机核销/价格/备注/自然worker，131与133退款、133预约仅查看、员工价、case11取消、财务和配置恢复已证实，10/10系统账单核平已读回。只剩上方通知、真实身份、同132重复UI、最后PC正常OWNER重登（退出及保护页已PASS）、体验版选择；生产须对应环境真实结果，完整E2E未PASS；当前按影响面执行必要隔离验证，不恢复无关旧套件。 |
| 正式经营数据 | 复用第 4 节已确认门店与时段；录入实际分类、商品、价格、餐段、图片、折扣/员工名单、操作人员及公告。需要当前经营输入的项目才待补，不重新确认已知规则。 |
| 运维与交付闭合 | 实际生产部署/ready、私有资源权限、日志和告警送达、备份/恢复与回滚可用性、最终版本/配置索引、微信正式发布结果。历史恢复目标不是实际达到的 RPO/RTO；目前没有生产完成回执。 |

平台依据：[CloudBase HTTP 服务](https://cloud.tencent.cn/document/product/876/130728)、[默认域名限制](https://docs.cloudbase.net/service/alias)、[微信支付关联管理](https://pay.wechatpay.cn/doc/v3/merchant/4013289251)。商户端提交关联和小程序端确认均需有效；主体不一致的材料要求按实际官方流程核实，不能泛化旧文档“所有主体必须一致”，也不将第二个 AppID 当作通用平台硬要求。

## 4. 已确认资料直接复用

- 门店“绥安食品”、地址“党政办公中心后院老食堂”：已归档 [店名地址确认](../../openspec/changes/archive/2026-08-22-set-store-identity/proposal.md)。
- 唯一取餐点“党政办公中心后院老食堂北门”：已归档 [取餐点确认](../../openspec/changes/archive/2026-08-22-set-single-pickup-point/proposal.md)。
- 午餐 11:30 截单、11:30–13:30 取餐；晚餐 17:00 截单、17:00–19:00 取餐；30 分钟粒度：[PRD](../product/online-ordering-system-prd-0818.md) 的已确认运营参数。
- 客户个体工商户主体、持有食品/餐饮相关资质、保留管理员身份及收款责任：[客户确认清单](../product/微信点单小程序客户资料与业务规则确认清单.md)。该文档旧“暂无账号/商户”不再作为当前状态；已持有资质不等于当前微信审核已通过。

## 5. 续接顺序与证据

UAT先按[最短下一步](../../deploy/runbooks/order-uat-deployment.md#2-当前入口与最短下一步)诊断通知并完成真正余项，不要求新付款，不重放13/14/15；顾客退款、员工、配置及恢复读回不重做。以下为生产续接顺序。

1. 按已收敛CloudBase路线补生产实现、恢复及容量前置；使用新生产Runbook，已有UAT安装/部署不重做。
2. 并行准备正式域名/平台配置、经营数据、生产资源/材料；不因旧 migration Push 故障暂停这些工作。
3. 补 release endpoint 与真实模板 ID，完成生产部署和必要实际状态读回；真实 OWNER/支付/退款/核销/通知结果仅在实际完成后登记。
4. 完成可用运维和正式发布登记；未发布、未运行或待核实项目保持原状态，不以删除测试或构建成功冒充完成。

非敏感回执根目录：`/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a`。

| 证据 | 相对回执根目录的路径 |
| --- | --- |
| 当前状态（含未完成项） | uat-delivery-status.json |
| SQL 直接安装及历史闭合 | sql113-closure/closure-receipt.json |
| 当前后端a833/004实际部署与健康 | uat-notification-msgid-deployment-a833507/deployment-receipt.json；首次001和003回执保留为历史 |
| PC 实际静态部署 / 公告恢复 | actual-web-admin-package/deployment-receipt.json、uat-e2e-rehearsal/admin-uiux-87dcc0b-publication/publication-receipt.json；finance7fe/api700分别绑定原回执 |
| OWNER / 示例菜单 | r9-progress/actual-owner-bootstrap-receipt.json、r9-progress/actual-uat-example-menu-receipt.json |
| 手机启动修正及用户反馈 | mini-delivery/device-compatibility-fix-receipt.json、mini-delivery/compatibility-delivery-receipt.json |
| 当前 Mini b3 交付、预览及继承相机证据 | uat-e2e-rehearsal/customer-b3-combined-UAT-delivery-receipt.json、uat-e2e-rehearsal/uat-preview-b3f1f6d-combined.jpg；root-N2-camera-actual-phone-green.json 为67efa真手机证据，未新增付款核销 |
| 商户入口/真实付款物化 | uat-e2e-rehearsal/merchant-execution-receipt.json |
| 顾客实际演练 | uat-e2e-rehearsal/customer-execution-receipt.json |
| UAT 订阅模板对齐 | uat-e2e-rehearsal/subscription-alignment-receipt.json |
| N3退款 / 通知真实边界 | uat-e2e-rehearsal/notification-N3-133-server-readback.json、uat-e2e-rehearsal/customer-execution-receipt.json；133/refund112成功，15终态保留，无新SENT/设备点击Green |
| 实际 PC / 菜单配置 | uat-e2e-rehearsal/root-admin-products-readback.txt、uat-e2e-rehearsal/merchant-execution-receipt.json |
| 首笔财务 / 待处理 / 实际导出 | uat-e2e-rehearsal/root-admin-finance-paid-readback.txt、uat-e2e-rehearsal/root-admin-pending-empty-readback.txt、uat-e2e-rehearsal/root-admin-finance-export.csv |
| N3前四笔财务与收款CSV | uat-e2e-rehearsal/root-admin-finance-four-orders-receipt.json、uat-e2e-rehearsal/root-admin-finance-20261010-12-four-orders.csv（358B，4行、43分）；210B旧两笔CSV保留为历史，退款汇总来自PC |
| 退款后财务 / 最终交易快照 | uat-e2e-rehearsal/root-admin-finance-refunded-readback.txt、uat-e2e-rehearsal/merchant-execution-receipt.json |
| 原设置 / 临时 / 已恢复设置 | uat-e2e-rehearsal/root-admin-original-settings.txt、uat-e2e-rehearsal/root-admin-temporary-settings.txt、uat-e2e-rehearsal/root-admin-restored-settings.txt |
| 10/10 制作/备好/手工核销/重复码 | uat-e2e-rehearsal/root-admin-tonight-preparing-readback.txt、uat-e2e-rehearsal/customer-execution-receipt.json、uat-e2e-rehearsal/merchant-execution-receipt.json |
| 真实 COS 既有证据 | actual-image-core/closure-receipt.json、storage-independent-preparation/storage-independent-complete-receipt.json |
| 用户停测与删除 | test-deletion/deletion-receipt.json、根 AGENTS.md 本轮用户覆盖 |

旧本地验收矩阵仅保留历史事实，不作为新测试执行命令。PC 退出修复 53ad8bc 已有本次正常退出及保护页刷新证据，最后正常重登待完成；个人中心布局修复 22e6bbc 的代码证据保留，实际体验未观测。Mini 身份切换保留会话，如正式交付要求真正注销，需要接入已有服务端注销接口，不在本轮文档更新中扩展实现。
