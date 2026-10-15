# Order UAT 部署与交接 Runbook

> 2026-10-12 Owner接管：当前完整验收见[全局验收](../../docs/quality/order-uat-acceptance-20261012.md)，后续生产唯一执行入口见[CloudBase生产Runbook](order-production-release.md)。本轮已授权必要局部验证；下文历史停止套件不再作为当前停测指令。Prod适配/容量/恢复仍未通过。

更新日期：2026-10-12（北京时间）。当前状态以最新总账与专项真实回执为准；10/10 首次发布、旧 bf/001 和两笔财务仅为历史证据。当前唯一 UAT 路线为 **CloudRun Go API + CloudBase 共享 PostgreSQL 官方 HTTPS RPC + 私有 COS + 同源 HTTP 网关/静态托管**。原生 PG 私网连接、IDE 环境转换和 `PushPGUserMigrations` 不是此路线前提；已成功 SQL、bootstrap、菜单和业务验收不重做。

本文是执行入口；实际配置见 [CloudRun 运行 Runbook](./order-uat-cloudrun-runtime.md)，Prod 剩余清单见 [生产交付 Handoff](../../docs/quality/order-production-delivery-handoff.md)。UAT 部署完成不等于真机业务全链路、生产发布或独立验证完成。

## 1. 当前已交付状态

UAT的131/133顾客退款、133预约仅查看、N2价格/两级备注/自然worker、真手机相机核销、员工报价、原生支付取消和财务已有实际结果。完整验收尚有下表明确余项，不能登记全量 PASS；Prod 架构与责任边界保持原批准规范。

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

### PC 营业日修复：唯一状态登记点

<!-- ROOT_UPDATE_PC_SOLDOUT_PUBLICATION -->
PC切片 `700031c13e105bd32a0bc88ffbb511693a254271` 仅发布 `web-admin/data/api.js`，Hosting成功1/1；远端对象30233B，ETag `86b166237ad894aff4501cbf8e8a6194` 与精确源码MD5一致，源码SHA256 `c656b128c3c009afa19656b8aba66a4212cc153adca9c022b9a0ef7adf5f6afb`。正常PC重载后标记F售罄，PG确认原10/10及新10/12记录；顾客正常选择10/12午餐11:30，F为 `sold_out=true / orderable=false / 已售罄`，正常实际点击一次购物车前后均0，没有确认、quote或prepay。该定向售罄主链真实Green已完成，不登记正式INDEPENDENT_VERIFIED或整套PASS。

root随后用正常PC恢复10/12售卖，PG只剩原10/10记录，PC显示可购；顾客恢复后正常重选今天10/12午11:30，已实际读回 `sold_out=false / orderable=true / 选择`，购物车仍为空；禁用与恢复两张实际截图及安全 before/after/restore 读回均保存在顾客回执。旧候选 `554b854` 仅是会话guard不足的阻断历史，未发布。原始脚本浏览器导航一次 `ERR_BLOCKED_BY_CLIENT` 后没有重试，HTTP-bodyhash **NOT_OBSERVED**；远端对象ETag/字节与正常PC实际行为分别为对应证据，不写HTTP200内容精确。此次仅PC单文件发布，API7e7与Mini67efa没有重建或重上传。

### 当前决定性证据

受控工件根目录：`/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a`；外部回执不自动进入 Git，敏感材料不进入文档。

- [当前总账](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-delivery-status.json)、[剩余场景清单](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/admin-remaining-20261011.md)
- [APIa833 /004实际发布](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-notification-msgid-deployment-a833507/deployment-receipt.json)
- [Admin87三文件发布/公告恢复](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/admin-uiux-87dcc0b-publication/publication-receipt.json)、[N3退款/通知实证](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/notification-N3-133-server-readback.json)、[顾客实际演练/原生取消197](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/customer-execution-receipt.json)、[PC实际配置/财务](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-case-execution-receipt.json)
- [Mini b3 实际交付](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/customer-b3-combined-UAT-delivery-receipt.json)、[新版默认首页预览](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/uat-preview-b3f1f6d-combined.jpg)、[组合候选独立验证](/Users/vivix/.codex/visualizations/2026/10/11/01a12bbb-ec58-79a3-bc80-7ad3d2d599d9/uiux-delivery/combined-b3-independent-receipt.json)；[67efa 真手机相机核销132](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-N2-camera-actual-phone-green.json)保留复用
- [四笔财务/实际收款CSV核对](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-finance-four-orders-receipt.json)、[4行收款CSV](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-finance-20261010-12-four-orders.csv)（358 B，SHA256 `35c7e1dd76062de516c72c1c3198282c521a32eef14038cba83e51dbe9a8e80b`）
- [账单967系统核平](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-bill967-system-readback-20261011.json)、[下载证据边界](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/admin-bill967-provenance-note.md)、[财务标签单文件发布](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/finance-label-publication-20261011.json)

<details>
<summary>10/10–11 首次发布与早期演练（历史，已由上方当前状态覆盖）</summary>


| 项目 | 已证实结果 | 边界 |
| --- | --- | --- |
| 环境与身份 | `order-uat-d1gtzpzj50615c300` / `ap-shanghai` / AppID `wx15e136b256d4dc5a` | 仅本 UAT，不复制到 Prod |
| 数据库 | 实际事务安装 113 个函数实体、109 个可执行函数；4 个旧实体保留但撤销执行权；原生迁移账本 v44 保持 | 原始数据、序列、账本和非函数对象读回保持，无需重新迁移 |
| 平台登记 | 已执行 SQL 的 27 条语句按顺序/内容核对后经 `RepairPGUserMigrations` 登记 `20261010091221`，平台记录共 38 条 | Repair 仅登记已成功执行的 SQL，不代替执行、不改失败历史 |
| API | 后端源码 `57ed75f7f8df30c4f7693a1e80dc8b8148e3f88b`；`order-api` 版本 `001`，BuildId `2608451769`，normal，流量 100% | Min1/Max1、0.25 CPU/0.5 GiB；不是容量验收 |
| 就绪 | 统一 HTTPS live/ready 实际均 200；ready 当次 1642 ms，原预算仍 2 秒 | 不等于支付、退款、后台登录或 worker 业务结果 |
| PC | 来源 `370991794227d2b5e9f1c6eb7b163814b5cb9d13`，24 个静态文件已部署；内置浏览器已实际认证 OWNER，菜单、订单和财务操作已记录 | 复用实际登录，不重复绑定或扫码；完整 E2E 未记 PASS |
| Mini 实际上传 | 源码 `bf949f75ff8e7d6e3966d07d0528513057bf8532`，19:12:04 上传 `uat-20261010-bf949f7` 平台 success=true；135 文件与源码一致 | 正常编译后首页 OWNER/session/storefront ready；上传不等于正式体验版已选择 |
| Mini 实际预览 / 支付返回 | 19:13:58 默认 `pages/launch/launch` 预览完整图已生成并实际查看；前序 `27d02898620c74df247b5122031e7419b4424030` 已真实报价 201、预支付 201；bf 原生支付返回后实际自动进入 result=ready、“订单已创建”、同一订单 130 | 支付返回结果收敛已观察，具体 wx success/fail 分支未知，不分别记 PASS；预览 wrapper 因 Buffer 截断未解析，不能登记预览 success JSON=true |
| 真机修正 | 已替换 7 处 `Object.hasOwn` 兼容依赖；用户随后反馈“看起来没问题了” | 原故障根因仍未确认，不宣称全部真机业务已验收 |
| OWNER / 商户入口 | OWNER 已真实绑定；正常首页 → 商户端 → 商户订单页的“登录电脑后台”和“切换身份”已实际点击通过，PC 已真实登录 | 不重新绑定或重走已通过入口 |
| 预约付款 / 退款 | 预支付 192、金额 2 分为 `PAID / APPLIED`；订单 129 曾实际显示 10/11 11:30 `RESERVED`；用户本人确认退款后，19:48:39 读回订单 `REFUNDED`、退款 110 `SUCCESS / APPLIED` | 退款到账 19:46:57，MATCH CALLBACK 已应用；不重复付款或退款，不把旧 PROCESSING 观察当当前失败 |
| 初始化 | 已创建门店、100% 默认折扣和已批准身份的首个 OWNER；开门并准备 1/2 分示例商品、10 月 11–17 日预约日期 | 均为 UAT 示例；没有自动创建订单、资金或假 PC 会话 |
| 多场景菜单 | PC 一次导入新增 3 类/7 菜；DB 确认 E11 下架、F12 仅 10/10 售罄、G13 分类 14 停用；顾客实际 10/11 午餐 A/B/D/F、晚餐 A/C/D/F，E/G/分类 14 隐藏 | 10/10 F 售罄不可购，19:50/20:20 时段曾可见；10/11 F 实际加购 1 份 0.50 元、cart canCheckout 后清空；全库 4 类/9 菜，不重导入，不宣称员工折扣已验收 |
| 财务 / 导出 | PC 汇总收款 2 笔 3 分、退款 1 笔 2 分、净额 1 分；历史待处理页 0；最终实际收款 CSV 210 B、2 行 0.02/0.01 元，与页面一致 | 退款 tab 的导出仍返回收款 CSV，退款 CSV 未观测；不是微信账单核平证明；首笔 136 B/1 行文件保留为历史 |
| 10/10 1 分履约 | 预支付 193/报价 191 为 `PAID / APPLIED`；订单 130 19:40:37 制作中、19:44:52 备好/签发取餐码、19:47:22 手工核销 `COMPLETED`，实际 UI 与 DB 一致 | 仅手工核销；再次输入 0001 仍显示成功页，已核销提示未通过。19:52:48 单行确认完成/核销时间未变，不第三次重试 |
| 10/11 新确认切片 | A 选 2 份、少饭及菜品备注后，点击确认出现 `ETIMEDOUT / APPID_ERROR`，该切片停止 | 整单备注/新报价未完成；保留原 public AppID `wx15e136b256d4dc5a` 及既有通过结果，不改身份配置 |
| 通知 | 19:48:39 订单 129/130 的 consent/outbox 均为空；develop/trial 模板已配置 | 没有真实授权、provider 接受或设备送达证据；不能把订单/退款成功推广为通知通过 |
| 营业配置恢复 | 真实 PC 保存并换页读回原餐段；19:50:56 DB 确认午餐 11:30–13:30/截单 11:30、晚餐 17:00–19:00/截单 17:00、步长 30、原取餐点/空公告、10/11–17 开放，10/10 关闭 | 10/10 原无日期行，现为 closed 行且有审计/版本变化；业务不可预约已恢复，不宣称数据字节一致 |
| 自动测试 | 交付分支删除 485 个测试文件；本轮按用户指令停止测试及回归 | 旧 95 场景、UI3、TDD/Gate 为历史，不恢复、不重跑、不伪记 PASS |

后端、PC、Mini 按各自完整 SHA 追踪。当时 Mini 改动未重建后端；总账中的旧 Mini SHA 或专项回执的历史 `NEXT` 字段不能成为重部署已完成 API 的理由。

### 首次发布与早期演练证据（历史）

受控工件根目录：`/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a`。外部回执不自动进入 Git，不包含凭据。

- [SQL 安装/登记闭环](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/sql113-closure/closure-receipt.json)
- [API 实际部署/ready](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/actual-cloudrun-deployment-build/deployment-receipt.json)
- [PC 静态部署](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/actual-web-admin-package/deployment-receipt.json)
- [首个 OWNER 初始化](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/r9-progress/actual-owner-bootstrap-receipt.json)、[示例菜单/开门](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/r9-progress/actual-uat-example-menu-receipt.json)
- [真机兼容修正/用户反馈](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/mini-delivery/device-compatibility-fix-receipt.json)
- [当时 bf 实际编译/代码上传/正常首页预览](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/customer-source-update-bf949f7-receipt.json)、[商户实际入口/付款物化读回](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/merchant-execution-receipt.json)
- [UAT 客户端订阅模板对齐](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/subscription-alignment-receipt.json)、[顾客实际演练](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/customer-execution-receipt.json)
- [PC 菜单实际读回](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-products-readback.txt)、[财务页面](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-finance-paid-readback.txt)、[待处理为空](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-pending-empty-readback.txt)、[实际收款明细导出](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-finance-export.csv)
- [原营业设置](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-original-settings.txt)、[临时营业设置](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-temporary-settings.txt)、[10/10 制作中订单（历史）](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-tonight-preparing-readback.txt)
- [最终两笔收款实际导出](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-finance-final-export.csv)（210 B，SHA256 `a6e4b908e0e8591ca5822df76d7d07d4d5a18f916a5b52d6ee957ecc32a380d9`）；退款 CSV 未观测。
- [退款后实际财务](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-finance-refunded-readback.txt)、[已恢复营业设置](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-restored-settings.txt)；最终交易、通知、重复码时间及恢复投影见商户 receipt。
- [停止测试/删除清单](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/test-deletion/deletion-receipt.json)


</details>

## 2. 当前入口与最短下一步

统一 origin：`https://order-uat-d1gtzpzj50615c300-1473835269.ap-shanghai.app.tcloudbase.com`。

- [PC 管理台](https://uat.suianwuye.com/pc/)
- [当前 b3 默认首页预览](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/uat-preview-b3f1f6d-combined.jpg)；开发上传已接受，正式体验版选择仍待完成。此码只用于打开小程序。

1. 复用已绑定 OWNER 与现有菜单；PC 已完成正常退出和受保护页刷新，最后正常重登待手机扫码批准。APPID_ERROR已恢复，不重新配置身份、绑定或导入数据。
2. 复用PC售罄禁用/恢复、公告保存/恢复、case11取消、case15预约仅查看和全部已通过业务证据；不重做退款、员工、配置或顾客恢复读回。
3. 仅诊断READY0002点击不存在及已部署a833后的真实发送/设备点击缺口；不重置或重放13/14/15，不为验证要求新付款。旧过期预支付不复用，结果不明保留边界。
4. 有真实参与者时完成商户子账号与未绑手机号边界；续接同132重复核销UI、最后PC正常OWNER重新批准（退出与保护页已通过）、正式体验版选择/读回。缺参与者仅暂停对应项，不伪造身份或完成结果。
5. 复用10/10 audit967系统核平；后续正式账单按原worker时序读取，不手动重跑。员工空名单/100%、今日F售卖、空公告及H14 OFF均已恢复，只处理新产生的实际配置差异。

取餐前30分钟制作中、制作中才可备好、待取餐才可核销的契约不变。N2以真实调度和正常扫码完成，未修改时钟、订单状态或绕过核销权限。用户停止自动套件/Gate的要求继续生效。

## 3. 后续 UAT 发布操作顺序

只有相关源码、配置或契约发生变化时才发布对应切片；静态/Mini 改动不触发 SQL/API 重装。

1. **冻结范围和发布物。** 记录各切片源码 SHA、构建/文件摘要、变更配置版本、目标环境。凭据仅从受保护通道装载，不进入代码包、镜像、日志或清单。
2. **处理确实新增的 SQL。** 已成功安装/登记的 113/109、原生 v44、平台 38 条复用回执。新 SQL 先核对 schema、定义、ACL/RLS、序列及账本，以事务、有界锁执行；结果不明先读回。仅真实执行成功后登记精确内容，不重复旧 migration Push InternalError 请求。
3. **初始化只做一次。** 常规发布不执行 bootstrap、示例 seeding 或覆盖经营配置；新增经营数据走已授权真实管理入口。
4. **构建/部署 API。** 按 [运行 Runbook](./order-uat-cloudrun-runtime.md) 生产构建、官方代码包发布并读回 BuildId/RunId、镜像 digest、配置摘要、流量及 ready。不自动变更 CAM、购买套餐或新增资源。
5. **发布静态/Mini。** PC 保持文件树和同源 `/api/v1`；Mini 绑定确认 AppID、合法域名、同一 UAT origin，编译、预览、上传并绑定源码 SHA。上传、预览、本人绑定和手机结果分别登记。
6. **按影响面记录事实。** 本轮不运行旧测试/回归或补 Gate。必要构建、平台接受、状态读回和用户操作分别记录；未发生的业务结果保持未确认。

### 当前实际网关

| 前缀 | 目标 | 路径透传 |
| --- | --- | --- |
| `/pc` | `STATIC_STORE/staticstore` | false；入口 `/pc/`，完整 apps/web-admin 内容发布到静态根；先资源后HTML |
| `/` | `CBR/order-api` | true；保留 `/api/v1/...` 和 `/health/...` |

按最长前缀匹配。平台网关鉴权关闭，应用 Bearer、角色和支付签名继续校验。旧 `/` 静态、`/api` 与 `/health` CBR 示例不是当前路由，不能照旧覆盖。

## 4. 故障与回退

- API ready 失败：先读当前服务、配置和安全日志，区分 RPC 身份、定义/ACL/账本、材料/外部身份和网络；不反复重装 SQL，不因单一路径失败认定数据库整体不可用。
- Mini 请求失败：确认真机为最新预览、有效 AppID、实际 endpoint/合法域名和安全错误信息；模拟器 200 不替代手机结果，已修正兼容调用不重复改动。
- PC 登录失败：沿手机号绑定 → QR challenge → 手机确认 → PC 会话定位，检查过期与角色；不放宽鉴权或直接写认证状态。
- 版本回退：确认上一镜像/配置真实存在、兼容当前 schema 且材料有效。当前记录不证明存在已验证旧线上版，不盲切旧标签。
- 数据恢复：停止相关写入/worker，先隔离核验数据、权限、账本和外部资金事实再按授权切换；不重放资金，不重新启用已撤销材料。
- 同条件重复失败无新证据不重试；在原授权/环境/成本内评估官方替代通道，平台安全拒绝不绕过。

## 5. Prod 复用边界

批准基线仍是 [生产架构规格](../../openspec/specs/production-architecture-baseline/spec.md)：**单 CVM + systemd/Nginx + 同地域 TencentDB MySQL 8.0 + CVM CAM Role 读取 SSM**。本轮没有 Prod 部署、生产资源写入、main 集成或远端发布证明。

| 项目 | 当前 UAT | 批准 Prod |
| --- | --- | --- |
| 运行入口 | `ORDER_ENV=uat`；CloudRun `:8080` | `ORDER_ENV=production`；API `127.0.0.1:8080`；Nginx 443，80 跳 HTTPS |
| 数据库 | 共享 PG HTTPS RPC；普通身份；113/109 闭包 | MySQL 8.0；本环境角色、匹配迁移器及账本 |
| 材料 | CloudRun 受控变量；镜像公开模板 | CVM Role 读取四份 SSM `SSM_Current`，不向 env 写秘密 |
| COS | UAT 私有版本桶和受限身份 | 独立生产私有桶、CVM Role 临时身份、恢复策略 |
| Mini | develop/trial 同一 UAT origin；release 空 | 正式 release origin/模板、formal 通知、提审发布 |
| 公网入口 | CloudBase 测试 HTTPS origin | 客户生产域名、证书、备案与平台登记 |

**架构规范与源码尚未收敛。** 当前非 UAT main/data 实现走原生 PostgreSQL 及 PostgreSQL 迁移，shared PG 分支仅允许 UAT 且绑定当前资源；批准生产规范仍为 MySQL。因此不能宣称生产 MySQL 源码已就绪，不能仅改 `ORDER_ENV` 或复制本 UAT SQL 上线。先确认并收敛规范/实现冲突，不擅自改变批准生产架构。

生产 SSM 名称仍为 `order-production-db-password`、`order-production-wechat-miniprogram-app-secret`、`order-production-wechatpay-api-v3`、`order-production-redemption-token-key`。生产使用 Role 加四份 SSM，禁止 inline AppSecret；非敏感 env 为 root-owned、权限不宽于 0640，角色只访问本环境材料与 COS。

旧原生 PG 私网/TLS、迁移执行机与连接预算属于已替代 UAT 路线参考，不阻塞当前 RPC；也不授权将 Prod 改为 PG/CloudRun。生产经营资料已有归档确认，沿确认记录装载，不重新索取、不复制 UAT 示例。生产资源/安全、业务链路、正式模板、监控恢复和平台发布的剩余工作只维护在 [生产 Handoff](../../docs/quality/order-production-delivery-handoff.md)，避免重复清单漂移。

## PC 后台静态发布与入口验收

- 唯一用户入口为 `https://uat.suianwuye.com/pc/`。HTML 的资源基址为 `/pc/`，不依赖用户输入末尾斜杠。不维护旧路径兼容或跳转逻辑。
- 网关 `/pc` 映射静态托管根目录。将同一候选的整个 `apps/web-admin/` 目录内容上传至托管根，先上传资源、最后上传 `index.html`。不可只上传一个业务 JS 后宣称后台整体最新。
- 发布后从用户入口打开，检查 CSS、JS、图片、扫码登录页和浏览器错误；逐文件下载比对候选内容。HTTP 200 或单文件一致不能代替页面验收。登录后业务验收另行记录。
