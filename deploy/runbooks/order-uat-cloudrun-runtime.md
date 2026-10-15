# Order UAT CloudRun 运行 Runbook

> 2026-10-12 Owner接管：当前完整验收见[全局验收](../../docs/quality/order-uat-acceptance-20261012.md)，后续生产唯一执行入口见[CloudBase生产Runbook](order-production-release.md)。本轮已授权必要局部验证；下文历史停止套件不再作为当前停测指令。Prod适配/容量/恢复仍未通过。

更新日期：2026-10-12（北京时间）。当前APIa833/004、Minib3、PC87三文件及原finance7fe/api700切片分别绑定证据；10/10首次001构建和早期演练为历史。本文件维护运行、构建、配置、SQL和路由；业务/PC营业日修复状态见 [UAT部署Runbook](./order-uat-deployment.md)，Prod边界与剩余见 [生产Handoff](../../docs/quality/order-production-delivery-handoff.md)。

## 1. 当前唯一运行组合

- 环境 `order-uat-d1gtzpzj50615c300`，地域 `ap-shanghai`，AppID `wx15e136b256d4dc5a`。
- API 经共享 PostgreSQL **官方 HTTPS RPC** 访问，使用普通专用身份、完整函数/ACL/RLS与16个序列权限；没有原生 PG 私网、管理 API Key、SSM 或通用腾讯云凭据的运行时入口。
- 启动只检查数据库，不迁移/bootstrap。ready 核对最终定义/ACL、实际身份和原迁移账本，保持 2 秒预算；不加跳过检查、禁用 worker 或权限降级开关。
- 以当前UAT源码为准：制作扫描已移除，制作状态由服务端时间派生；支付、退款、订阅仍有1秒扫描，可靠异常补偿未完成；对账北京时间10:05处理昨日，失败30分钟重试，但停机/多副本恢复仍需验收。旧四循环/每小时说明仅对应历史版本，ready不是任务推进证明。
- 图片桶 `6f72-order-uat-d1gtzpzj50615c300-1473835269` 保持私有和版本控制，图片确认固定版本；`images/` 非当前版本按已确认 30 天策略保留。具名 COS 身份仅用于受限临时凭据及所需读写；既有核心/隔离恢复回执不代表全部生产恢复目标。

用户本轮明确停止自动测试/回归、删除测试用例，交付分支已删除 485 个文件。旧 95 场景、UI3、TDD/Gate、ordinary/race 仅作历史，**不恢复、不重跑、不把未运行记 PASS**。生产构建、SQL 安装/读回、部署/ready、代码上传和用户实际操作分别记录。

## 2. 当前实际发布快照

| 项目 | 绑定结果与边界 |
| --- | --- |
| 后端 | `a833507a809decd718729501d2bab96cef2d0f35`；`order-api` / `004`，normal、流量100%、HasTraffic，10/12 01:35 live/ready200；二进制SHA256 `c40119e9206a47ef65e41247f3e53d0c42aa3e06ce27b851d310ec27904c5120` |
| 最新构建 / 发布回执 | BuildId `2608516656`、RunId `multi_tenant_1xFxN39xmNNpKS`；`uat-notification-msgid-deployment-a833507/deployment-receipt.json`；14项环境配置、PUBLIC及资源保持，不能沿用下面001摘要 |
| Mini | `b3f1f6da24935a9a46e4615b87374a56027b70af`；136源码文件精确匹配，既有根project.config.json另计；`uat-20261012-b3f1f6d` 于02:02:46开发上传接受，默认首页预览完整二维码已查看；金额/输入保留/商户列表及详情实际读回通过 |
| b3 页面检查 | 10分×2=20分预估、修改返回输入保留、商户退款列表/133查看正常；未新增quote/prepay/order/资金操作；制作中0单，备好入口现场未出现，复用精确候选UI2证据 |
| PC | 初始3709917包加finance7fe；Admin87的app.css/app.js/settings.js三文件已发布，公告保存/换页读回/清空恢复PASS；api700营业日修复只维护在[唯一登记点](./order-uat-deployment.md#pc-营业日修复唯一状态登记点) |
| 业务 | N1订单131退款10分；N2订单132备注/变价/自然worker/真手机相机核销COMPLETED；N3订单133预约仅查看、用户取消/refund112成功1分/顾客REFUNDED无QR，case15 PASS；case11复用197原生取消主路径PASS |
| 财务 | N3前四笔快照收43/退12/net31分，收款CSV4行358B已核对；133后今日PC收1/退1/net0分，本次未重导出大范围；audit967系统核平已观察但单次HTTP下载证明缺 |
| 通知 | READY0002已收到但点击不存在；outbox15两次HTTP200/errcode0/msgid被旧解析拒绝，第三次USER_NOT_SUBSCRIBED永久失败；a833兼容已部署，新本地SENT/设备点击Green未观察，13/14/15不重放、不由15倒推old14 |

Mini/PC静态切片不触发API、SQL或bootstrap重装。APPID_ERROR已恢复原publicID。完整18场景未PASS；剩余通知点击/新发送Green、真实子账号/未绑手机号、同132重复UI、最后PC正常OWNER重登（退出及保护页已PASS）、体验版选择。case11回调未观察不升级额外门禁。

- [当前总账](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-delivery-status.json)、[剩余场景清单](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/admin-remaining-20261011.md)
- [APIa833 /004实际发布](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-notification-msgid-deployment-a833507/deployment-receipt.json)
- [Admin87三文件发布/公告恢复](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/admin-uiux-87dcc0b-publication/publication-receipt.json)、[N3退款/通知实证](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/notification-N3-133-server-readback.json)、[顾客实际演练/原生取消197](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/customer-execution-receipt.json)、[PC实际配置/财务](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-case-execution-receipt.json)
- [Mini b3 实际交付](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/customer-b3-combined-UAT-delivery-receipt.json)、[新版默认首页预览](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/uat-preview-b3f1f6d-combined.jpg)、[组合候选独立验证](/Users/vivix/.codex/visualizations/2026/10/11/01a12bbb-ec58-79a3-bc80-7ad3d2d599d9/uiux-delivery/combined-b3-independent-receipt.json)；[67efa 真手机相机核销132](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-N2-camera-actual-phone-green.json)保留复用
- [四笔财务/实际收款CSV核对](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-finance-four-orders-receipt.json)、[4行收款CSV](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-finance-20261010-12-four-orders.csv)（358 B，SHA256 `35c7e1dd76062de516c72c1c3198282c521a32eef14038cba83e51dbe9a8e80b`）
- [账单967系统核平](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-bill967-system-readback-20261011.json)、[下载证据边界](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/admin-bill967-provenance-note.md)、[财务标签单文件发布](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/finance-label-publication-20261011.json)

<details>
<summary>10/10 首次001发布与早期业务（历史，构建摘要只属于当时版本）</summary>


| 项目 | 绑定结果 |
| --- | --- |
| 后端源码 | `57ed75f7f8df30c4f7693a1e80dc8b8148e3f88b` |
| API 服务/版本 | `order-api` / `001`；normal、FlowRatio 100、HasTraffic true |
| 构建 | BuildId `2608451769`；RunId `multi_tenant_1xFTFY2GmpMuyV` |
| 镜像 | `ccr.ccs.tencentyun.com/tcb-100052186355-paxl/ca-ifypfabp_order-api:order-api-001-20261010172027` |
| 构建镜像 digest | `sha256:b017869c0944b4c41e3c91c5e814ef0f074b80f7351f9e13f84c58151fa8ff38` |
| Linux amd64 二进制 SHA256 | `5af36a04880b8e1dd28fd81432165b5cd46b1eb49f19369b76951720c104b84a` |
| 服务规格 | 0.25 CPU / 0.5 GiB，Min1/Max1，8080，PUBLIC |
| EnvParams | 14 项、4603 B；摘要 `b2f3802be13e55d9736a1499ca04531a2bb11668ea2dd4d7e9d21575871980cf` |
| HTTP | live 200 / 318 ms；ready 200 / 1642 ms（当次） |
| PC | 源码 `370991794227d2b5e9f1c6eb7b163814b5cb9d13`；24 文件；现有托管 ID `1829864` |
| PC 实际业务 | 内置浏览器已认证 OWNER；工作台、订单 129、菜单配置及首笔财务已实际读回；一次导入新增 7 菜/3 类 |
| 当时实际 Mini 上传 | 源码 `bf949f75ff8e7d6e3966d07d0528513057bf8532`；19:12:04 上传 `uat-20261010-bf949f7`，平台 success=true，135 文件精确一致 |
| Mini 编译/预览与回调边界 | 唯一运行项目编译后首页 OWNER/session/storefront ready；19:13:58 默认 launch 预览完整图已生成并查看；前序 27d0289 已真实报价/预支付 201 |
| 支付返回收敛 | bf 原生支付返回后实际自动进入 result=ready、“订单已创建”、同一订单 130；具体 wx success/fail 分支未知，不分别记 PASS |
| 预约付款/退款 | 用户支付的 2 分预支付 192 为 `PAID / APPLIED`；订单 129 曾预约 10/11 11:30，现 `REFUNDED`；退款 110 为 `SUCCESS / APPLIED`，MATCH CALLBACK 成功到账 19:46:57 |
| 财务/导出 | PC 收款 2 笔 3 分、退款 1 笔 2 分、净额 1 分；历史待处理页 0；最终收款 CSV 210 B、2 行 0.02/0.01 元；退款 tab 导出仍为收款 CSV，退款 CSV 未观测，账单未核平 |
| 10/10 付款/履约 | 用户支付 1 分，预支付 193/报价 191 为 `PAID / APPLIED`；订单 130 19:44:52 备好/取餐码签发、19:47:22 手工核销 `COMPLETED`，DB 与实际 UI 一致 |
| 重复码/通知边界 | 第二次输入 0001 仍显示成功页，已核销提示未通过；19:52:48 DB 完成/核销时间仍为 19:47:22；129/130 consent/outbox 均空，没有消息送达证明 |

各切片不必同 SHA。SQL 回执的 `service_deployment=NEXT`、初次 API 回执的旧 Mini SHA 为历史状态，后续专项回执覆盖相应进度，不触发重做。

受控回执根目录：`/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a`。

- [SQL113 安装/登记](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/sql113-closure/closure-receipt.json)
- [精确 API 构建](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/actual-cloudrun-deployment-build/receipt.json)、[实际发布/ready](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/actual-cloudrun-deployment-build/deployment-receipt.json)
- [PC 发布](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/actual-web-admin-package/deployment-receipt.json)、[当时 bf 实际编译/上传/预览](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/customer-source-update-bf949f7-receipt.json)
- [商户入口/付款物化读回](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/merchant-execution-receipt.json)、[订阅配置对齐](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/subscription-alignment-receipt.json)
- [PC 菜单](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-products-readback.txt)、[首笔财务](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-finance-paid-readback.txt)、[待处理为空](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-pending-empty-readback.txt)、[收款明细导出](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-finance-export.csv)
- [最终两笔收款实际导出](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-finance-final-export.csv)（210 B，SHA256 `a6e4b908e0e8591ca5822df76d7d07d4d5a18f916a5b52d6ee957ecc32a380d9`）；退款 CSV 未观测。
- [退款后财务](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-finance-refunded-readback.txt)、[顾客结果/菜单/核销](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/customer-execution-receipt.json)、[恢复设置](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-restored-settings.txt)


</details>

## 3. SQL 和初始化：已闭环，不重做

最终 SQL 已实际执行一次，SHA256 `859c8c232c9ac5dd44759bba2d2186080bfaa09bafbd70dc87512aea32e74f59`，RequestId `c0c60931-36fa-43c8-9045-19c2ebbd7f88`。事务守护与实际读回确认 113 个实体/109 个有效执行函数，4 个旧实体保留并撤销 authenticated EXECUTE；非函数目录、业务数据、序列、原生 v44、原平台账本保持。普通专用身份当次 ready 1842 ms，未改 2 秒预算。

成功 SQL 的 27 条语句按官方分句规则核对顺序/内容/摘要后经 `RepairPGUserMigrations` 登记：version `20261010091221`，name `order_uat_payment_runtime_binding`，RequestId `0b44bf64-19d5-4649-bcf5-0cef71215342`；最终读回 `97e42829-9669-4e59-88ca-4fe710c5bae8`。平台 38 条、旧 37 条保持。Repair 仅记录已执行成功事实，不重登、不执行 down、不篡改旧历史。

确有新 SQL 时先核对目标现状和恢复点，单 writer 经官方管理 SQL 通道执行受控事务，限定锁等待/执行时间；成功读回完整定义、ACL/RLS、序列、账本后才登记实际内容。结果不明先读回。旧 migration Push InternalError 无新证据不重试；不能据此报告数据库不可用。

首个 OWNER/门店/100% 默认折扣已建；UAT 示例分类、1/2 分商品、2026-10-11 至 17 日七个日期和开门状态已实际安装。原两餐段/30 分钟间隔保持，没有自动订单、资金或假 PC 会话。**常规发布不重做 bootstrap/seeding，不覆盖用户后续配置。** 正式经营资料已有归档确认，按记录进入 Prod，不重索取、不以示例替代。

## 4. 生产构建与官方部署

编译权威为 [`Dockerfile.uat`](../cloudrun/Dockerfile.uat) 及专用 ignore。成功链路是：精确干净源码 → Go1.26.5/CGO0/linux/amd64、`-trimpath`、`-ldflags="-s -w"` 生产构建 → 外部 final-stage 包 → 现有环境内官方 `manageCloudRun targetPath` 构建/部署。源码、Go.sum、嵌入资产、编译器、二进制摘要绑定回执；本轮没有自动测试。

外部 Dockerfile 逐字保留权威 final stage，仅把 `COPY --from=build /out/order-api /app/order-api` 改为 `COPY order-api /app/order-api`。runtime digest、11 项公开 ENV、非 root 65532、8080 和入口不变。基础镜像：

`gcr.io/distroless/static-debian13:nonroot@sha256:2293b36c7c9082bf4115aab724b4d2cddec82c8eba39bf27ac0517e159acf150`

包只含绑定二进制/Dockerfile；平台可加入无材料 `cloudbaserc.json`。秘密不进入 build 参数、源码、镜像 ENV 或包；新发布不用未绑定的探针产物。

手动 TCR 发现曾被 CAM 拒绝（RequestId `3666469b-f6e2-4b4d-b323-696fdc9a1416`），该拒绝仅适用于已放弃路径。官方代码包路线已实际成功，不等待额外 TCR 权限，不换身份、增权或采购绕过。

发布时读回实际 BuildId/RunId、构建/部署结果、版本、digest、配置摘要和流量。现有额度 Min1/Max1、0.25 CPU/0.5 GiB；原超额按量/自动续费关闭设置不擅改。结果不明先查服务，不重复创建。

## 5. 配置与秘密保护

镜像内 11 项公开订阅字段组成完整模板版本，当前后端已配置 trial。CloudRun 版本经受保护 EnvParams 仅注入以下 14 项（包含敏感值）：

```text
ORDER_ENV
CLOUDBASE_ENV_ID
ORDER_TENCENT_REGION
ORDER_COS_BUCKET
ORDER_SHARED_PG_USERNAME
ORDER_SHARED_PG_PASSWORD
ORDER_SHARED_PG_PUBLISHABLE_KEY
ORDER_SHARED_PG_SERVICE_UID
ORDER_UAT_COS_SECRET_ID
ORDER_UAT_COS_SECRET_KEY
ORDER_WECHAT_MINIPROGRAM_APP_ID
ORDER_WECHAT_MINIPROGRAM_APP_SECRET
ORDER_WECHATPAY_API_V3
ORDER_REDEMPTION_TOKEN_KEY
```

不注入原生 `ORDER_DB_*`/DSN、ambient `PG*`、SSM、管理 API Key、通用云凭据或凭据文件入口。服务不覆盖公开模板；读回核对 14 项及有效 25 项，只记录数量/摘要。实际 EnvParams 4603 B 已接受；改变后仍核对平台 5 KB 限制，不以升规格、裁剪支付公钥解限。

支付 JSON 保留严格字段集合：merchant_id、merchant_certificate_serial、merchant_private_key_pem、wechatpay_public_keys（serial/public_key_pem）、api_v3_key、notify_url。保留 RSA2048/指定 PEM、1–8 个唯一公钥、32 字节 APIv3 key、未知字段拒绝。回调为本 origin `/api/v1/payments/wechat/notify`，退款派生 `/api/v1/refunds/wechat/notify`；保留签名、金额、编号、AppID、商户/环境校验。

核销 JSON 保留 version/key_base64：version 是 1–65535 字符串，标准 Base64 解码为 32 字节，JSON 最大 256 字节。轮换发布新配置重载，撤销材料不纳入回退。

工具错误可能把配置拼进诊断 URL。禁止输出原始 EnvParams、完整错误对象或含 percent-encoded 配置的诊断链接；仅安全错误码、脱敏摘要、RequestId，不访问/提交这类自动链接。

## 6. HTTPS、静态与 Mini

统一 origin：`https://order-uat-d1gtzpzj50615c300-1473835269.ap-shanghai.app.tcloudbase.com`。实际 `/` → `CBR/order-api`、透传 true；`/pc` → `STATIC_STORE/staticstore`、透传 false，最长前缀优先。平台鉴权关闭，应用 Bearer/角色/支付鉴签保持。

PC当前唯一入口 `https://uat.suianwuye.com/pc/`，将完整 `apps/web-admin/` 内容映射静态托管根，HTML基址 `/pc/`，同源 `/api/v1`。先资源、后HTML，无旧地址兼容；旧web-admin目录记录仅是历史收据。

Mini AppID `wx15e136b256d4dc5a`，合法域名校验开启；develop/trial 同一精确 UAT origin，release endpoint 为空。`subscriptionTemplateConfig` 的 develop/trial READY 与 REFUND_RESULT 已对齐已部署真实模板，早期已包含在 cc45be2 上传中，当前b3精确源码保持该模板合同；release 两类 ID 仍为空。后端与客户端配置对齐不等于真实授权或设备送达。

OWNER 已真实绑定，正常首页 → 商户端 → 商户订单页正文“登录电脑后台”及右上角“切换身份”已实际通过。电脑登录沿原 Mini Bearer + 原生 getPhoneNumber code 到 `/api/v1/me/admin-login/approve`，OWNER/手机号/过期检查不变。此前电脑原生工具返回 `Computer Use server error -10005: timeoutReached` 后已停止同条件重试；后续内置浏览器已实际认证 OWNER，工作台、订单和管理页面可见。最后一次PC正常退出及保护页刷新已PASS，当前PC为登录页，仅正常OWNER扫码重新批准待完成。QR、代码上传、绑定、PC会话和认证操作分别登记，不重复绑定。合法域名用户已配置；新 origin/upload/download 需求按实际登记，不关闭校验绕过。

当前菜单保留一次导入的3分类/7商品及原1/2分示例，不重导入。员工80%正常服务端报价42→34分、停用回42分、空名单/100%恢复均有实际结果；8分优惠金额格式已修正并随b3上传，隔离UI2证据为-¥0.08；现场员工价不重跑。售罄按服务端营业日和顾客所选取餐日期隔离；PC跨午夜缓存旧日的修复及真实Green仅维护在[UAT部署状态](./order-uat-deployment.md#pc-营业日修复唯一状态登记点)，不得用浏览器日期替代权威日。

当前 b3 默认首页预览及开发上传已接受，金额与输入保留、商户列表和详情已实际检查；136 源文件与精确候选匹配。旧67efa真手机直接相机核销132证据保留复用，当前正式体验版尚未选择。4行CSV对应N3前43/12/31分有效快照；133之后今日PC收1/退1/net0分。未重导出大范围，消息边界见当前表。

### 10/10–11 菜单、营业恢复和旧预览（历史）

以下仅保留当时金额、配置与手工核销证据；相机、员工、当前财务及版本的后续结果以第2节为准。

真实 PC 一次导入新增 7 菜/3 类，DB 单次快照确认分类 12–14、商品 7–13；E11 下架、F12 仅 10 月 10 日售罄、G13 所属分类 14 停用。原分类 11/商品 5、6 保留，全库 4 类/9 菜。顾客实际 10/11 午餐 A/B/D/F、晚餐 A/C/D/F，E/G/分类 14 隐藏；10/10 F 售罄不可购、临时时段 19:50/20:20 曾可见，10/11 F 实际加购 1 份 0.50 元、cart canCheckout 后清空。员工折扣未据商品描述登记通过。当时财务实际收款 2 笔 3 分、退款 1 笔 2 分、净额 1 分。首笔付款时导出的 CSV 为 136 B、1 行、0.02 元，SHA256 `f8a9eb0cb1c339f101296dbe0d3c291778a86e2170404120e4e8fbdb3b0c8119`，是历史首笔明细导出。最终真实收款导出 root-admin-finance-final-export.csv 为 210 B、2 行 0.02/0.01 元，SHA256 `a6e4b908e0e8591ca5822df76d7d07d4d5a18f916a5b52d6ee957ecc32a380d9`，与财务页面一致；退款 tab 导出仍返回收款 CSV，退款 CSV 未观测，微信账单核平仍未证实。

原设置投影保存在商户 receipt 的 `cases_public_database_readback.settings_restore_write_projection`，另有[原设置页面](/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/uat-e2e-rehearsal/root-admin-original-settings.txt)。真实 PC 临时开放 10/10、晚餐截单 19:50/取餐 19:50–20:20用于实际演练；随后通过 PC 保存并换页读回恢复设置。19:50:56 DB 全公开投影确认午餐截单 11:30/取餐 11:30–13:30、晚餐截单 17:00/取餐 17:00–19:00、步长 30、原取餐点/空公告/store open、10/11–17 open，10/10 closed。10/10 原无日期行，现有 closed 行并有审计/版本变化，恢复的是不可预约业务安排，不是字节一致恢复或删除该行。

实际顾客链路在本地 27d0289 已报价 201、预支付 201并显示微信支付二维码；全部真实付款由用户完成。2 分预支付 192 已 `PAID / APPLIED` 并生成预约订单 129；用户随后本人确认原路全额退款，19:48:39 只读确认订单 `REFUNDED`、退款 110 `SUCCESS / APPLIED`，MATCH CALLBACK 成功到账 19:46:57、物化 19:47:00。旧 ACTIVE_QUERY PROCESSING/DEFERRED 保留为历史。SDK fail 后错误页的 bf949f7 修正已交付；本轮真实原生支付返回后自动进入 result=ready、“订单已创建”、同一订单 130，旧 debug modal pending 被实际结果替代，确切 wx success/fail 分支未知。禁止重复资金操作，agent 不执行真实付款。

10/10 1 分预支付 193/报价 191 已 `PAID / APPLIED`，MATCH ACTIVE_QUERY 付款观察已应用。订单 130 取餐 19:50，真实 merchant UI 备好、顾客凭证和手工 0001 核销完成；19:48:39 DB 确认 paid 19:40:34 → preparing 19:40:37 → ready/code-issued 19:44:52 → redeemed/completed 19:47:22，取餐码密文清除、hash 保留。再次输入 0001 仍显示成功页，已核销提示未通过；19:52:48 单行只读确认 completed/redeemed 时间仍为 19:47:22，没有重写完成时间，不第三次尝试。相机扫码未执行；两订单 consent/outbox 均空，不能登记通知接受或设备送达。

当时默认预览不指定 page-path，沿 `app.json` 首项进入 `pages/launch/launch`；完整 JPEG 为 46050 B、SHA256 `75661e3d29773f5bd03d1e135f218521cfffdb13765d1b3fff6943b8a863fbc8`，已生成并实际查看。CLI wrapper 的巨大 Buffer 输出截断导致未解析成功 JSON，仅登记完整文件/视图证据；19:12:04 的上传 success=true 单独复用实际回执，不重复上传。

## 7. 故障、恢复与尚未闭环

live/ready/PC资源 200 是对应运行事实；手机首页用户反馈恢复、扫码页显示、provider 接受不推广为全部业务通过。没有新影响面不重复构建、SQL 或回归。

`APPID_ERROR`是已恢复的历史故障；保留原public AppID `wx15e136b256d4dc5a`。131/133退款、132备注/变价/worker与真手机扫码、员工报价、case11取消、case15预约仅查看、公告及售罄恢复证据直接复用，不重做。

PC营业日修复、售罄禁用及恢复读回已PASS，证据只维护在[唯一登记点](./order-uat-deployment.md#pc-营业日修复唯一状态登记点)。仅续接通知点击/新发送Green、真实子账号/未绑手机号、同132重复UI、最后PC正常OWNER重登（退出及保护页已PASS）、正式体验版选择。READY0002点击不存在为真实RED；old14前两次响应未知，15的窗口日志不能倒推其原因。13/14/15终态保留，不重放，不为诊断要求新付款；旧过期预支付不复用。

10/10 audit967系统核平已读回，单次HTTP下载及提前运行来源不据此确认；10/11账单不手工重跑。生产容量/监控/恢复/worker交接等仍按生产影响面完成，UAT自然worker一次推进不代替生产运维证明。完整E2E未PASS；续接见[UAT演练](./order-uat-deployment.md#2-当前入口与最短下一步)，Prod剩余由[Handoff](../../docs/quality/order-production-delivery-handoff.md)维护，不恢复删除测试。

恢复先核对兼容版本、实际数据和外部资金事实；无验证旧版不盲切流量，隔离恢复后再授权切换，不覆盖新业务、不重放资金。重复失败先回到用户目标评估已授权官方替代通道，保留安全边界，不重复安装、转换或 Push。

生产已确认采用 CloudBase/CloudRun + Shared PG + 私有 COS，唯一执行入口是[生产Runbook](order-production-release.md)。旧CVM/MySQL/SSM为待清理历史规格，不作为发布路线；生产适配、最终候选、独验与真实生产验收尚未完成。
