# Order 一期生产交付 Handoff

> 状态日期：2026-08-24
> 接手对象：下一位研发负责人、甲方平台管理员、甲方云管理员
> 产品唯一基线：[`online-ordering-system-prd-0818.md`](../product/online-ordering-system-prd-0818.md) §1–§14
> 技术基线：[`online-ordering-system-technical.md`](../product/online-ordering-system-technical.md)
> 本地验收矩阵：[`order-mvp-prd-acceptance-matrix.md`](./order-mvp-prd-acceptance-matrix.md)

本文只记录非敏感状态、资源名称占位符、执行步骤和完成证据。不得在本文、Git、聊天、普通邮件、工单评论或日志中粘贴密码、AppSecret、APIv3 key、私钥、完整证书、Cookie、Token、真实手机号或人员名单。

## 1. 交付裁决

当前结论为 **NO-GO：不能直接交付甲方上线或宣称真实用户全链路已验收**。

现有实现已经达到完整本地开发版候选水平，但不是已上线产品：

- 本地 exact candidate：`b20c0fe51a4bb264703e889c6334ef7f8474a42b`。
- 本地 95-case 总账：95 `PASS`、0 `MISSING`、0 `NOT_RUN`、0 `FAILED`；仍有 22 项 L4 `BLOCKED_EXTERNAL`。
- 微信开发者工具只完成了 WXML/WXSS 官方编译；没有真实手机号、支付资金、退款资金、订阅送达、相机扫码、PC 扫码和真实 COS 验收。
- 远端 `main` 在交接时仍为 `64373ad50a53225b995105b5d4fbe83498b1d16b`；本地候选未推送、未合并、未部署。
- `trial` / `release` API endpoint 仍为空，没有线上 UAT 或生产 URL。
- 下文 P0 代码与运行配置阻断尚未解决；即使今天拿到全部外部资源，按现有 runbook 也不能直接上线。

只有第 8 节 DoD 全部完成，才能改为 GO。

## 2. 当前功能完成度

| 能力 | 本地实现/证据 | 真实平台/线上证据 | 当前状态 |
| --- | --- | --- | --- |
| 冷启动、静默会话、用户/商户分流 | UI/API/MySQL 已闭合 | 无正式 AppID + UAT API 真机证据 | 部分完成 |
| 登录 | 微信 code2session、手机号 adapter 已实现；fake E2E 已闭合 | 真实微信登录和 getPhoneNumber 未跑 | 部分完成 |
| 退出登录 | Mini 只返回身份选择页；PC 只清本地 token | 无服务端会话撤销接口或后续 401 证据 | **未完成** |
| 浏览、菜单、详情、购物车、口味、备注 | 本地 rendered UI1 + HTTP + MySQL 已闭合 | 真实小程序体验版未跑 | 部分完成 |
| 预约今天/明天、餐段、截单、售罄 | 本地正向与失败边界已闭合 | UAT 实际门店配置未跑 | 部分完成 |
| 手机号绑定、员工/访客双价 | fake provider、双要素、Quote 快照已闭合 | 真实手机号授权和真实名单未跑 | 部分完成 |
| Quote、Prepay、支付取消/重试、支付查询兜底 | 本地 fake 微信支付完整 E2E | 真实 JSAPI、回调、Query、资金未跑 | 部分完成 |
| 订单生成、六态、自动排产、订单查询 | 本地完整 | 真实支付订单未跑 | 部分完成 |
| 用户取消、全额退款、退款中/已退款 | 本地 fake 退款及未知态已闭合 | 真实退款资金、回调、账单未跑 | 部分完成 |
| READY、二维码、手工取餐号、跨日与重复核销 | 本地完整 | 真实相机和真机扫码未跑 | 部分完成 |
| 待取餐/退款订阅消息 | provider、outbox、重试/失败本地完整 | 模板未审批，真实消息未送达 | 部分完成 |
| PC 主账号、扫码登录、RBAC、目录、导入、财务 | 本地 Chrome + HTTP + MySQL 完整 | PC 线上域名和真实扫码未跑 | 部分完成 |
| 商品图、开屏图 | file/fake 与页面边界已闭合 | 私有 COS 生产契约存在 P0/P1 缺口 | **未完成** |
| 生产部署、监控、恢复 | systemd、迁移、基础检查存在 | 无真实 CVM/CDB/COS/HTTPS 部署回执 | **未完成** |

## 3. 接手研发必须先完成的代码与交付阻断

### P0-1：集成候选并建立可追溯发布物

- [ ] 从 `b20c0fe51a4bb264703e889c6334ef7f8474a42b` 建立正式集成分支。
- [ ] 复核其相对远端 `main` 的完整 diff；不得只审最后几个 commit。
- [ ] 修完本节其他 P0 后生成新 candidate SHA。
- [ ] 在独立 detached worktree 对新 SHA 只读验证。
- [ ] 经批准后用明确 refspec 推送分支并创建 MR；不得直接推 `main`。
- [ ] MR 合并后重新绑定最终 SHA、构建物 SHA256 和部署 release 目录。

完成证据：远端 MR、合并 SHA、构建物 SHA256、干净 detached verifier receipt。

### P0-2：补真正的退出登录和服务端会话撤销

当前 Mini 的“退出”只是 `reLaunch`，PC 只是删除 `sessionStorage`，已有 token 在服务端仍可使用。
现有 PRD 未定义真正的退出语义；用户已将“登录、退出”列为交付核心能力，因此必须先补产品契约，再实现和验收，不能把现有返回身份选择页冒充退出。

- [ ] 由产品/甲方书面确认公共契约；推荐：退出只撤销当前会话，不解绑手机号、不删除用户、不删除商户绑定。
- [ ] Mini 和 PC 都提供服务端 session revoke。
- [ ] revoke 幂等；撤销后旧 token 请求必须返回 401。
- [ ] Mini 清理本地 token 后回冷启动；重新登录后从服务端恢复订单，而不是依赖客户端内存。
- [ ] PC 退出后旧 PC token 不可再访问 OWNER API。
- [ ] 覆盖 API/MySQL、Mini rendered、PC rendered、重复退出、网络失败和过期 token。

完成证据：Red→Green、旧 token 401、DB session 已失效、重新登录订单仍可查。

### P0-3：统一生产 env、SSM、CAM 与 runbook

运行时需要的配置多于当前 `order-production.env.example` 和 `order-preflight` 声明；指南/CAM 只写两个 SSM Secret，但代码还强制读取微信支付和核销 token 两个 Secret。

- [ ] 从生产 `main.go` 生成唯一必填配置清单，更新 env example、preflight、runtime check、runbook 和技术文档。
- [ ] 非敏感 env 至少覆盖数据库坐标、AppID、地域、COS 坐标及以下 11 项订阅配置：
  - `ORDER_WECHAT_SUBSCRIPTION_TEMPLATE_CONFIG_VERSION`
  - `ORDER_WECHAT_SUBSCRIPTION_READY_TEMPLATE_ID`
  - `ORDER_WECHAT_SUBSCRIPTION_READY_ORDER_NUMBER_KEY`
  - `ORDER_WECHAT_SUBSCRIPTION_READY_PICKUP_DATE_KEY`
  - `ORDER_WECHAT_SUBSCRIPTION_READY_PICKUP_TIME_KEY`
  - `ORDER_WECHAT_SUBSCRIPTION_READY_PICKUP_POINT_KEY`
  - `ORDER_WECHAT_SUBSCRIPTION_REFUND_RESULT_TEMPLATE_ID`
  - `ORDER_WECHAT_SUBSCRIPTION_REFUND_RESULT_ORDER_NUMBER_KEY`
  - `ORDER_WECHAT_SUBSCRIPTION_REFUND_RESULT_RESULT_KEY`
  - `ORDER_WECHAT_SUBSCRIPTION_MINIPROGRAM_STATE`
  - `ORDER_WECHAT_SUBSCRIPTION_LANGUAGE`
- [ ] SSM SecretName 至少统一为：
  - `order-{env}-db-password`
  - `order-{env}-wechat-miniprogram-app-secret`
  - `order-{env}-wechatpay-api-v3`
  - `order-{env}-redemption-token-key`
- [ ] UAT/prod CVM Role 仅能读取本环境这四个 Secret 和本环境 COS。
- [ ] preflight 必须在启动进程前发现缺配置，不能等 `order-api` 启动中途失败。
- [ ] 用真实 CAM Role + SSM 在 UAT 运行一次 production-mode startup，不允许长期 SecretId/SecretKey。

完成证据：production-mode 启动、无敏感输出、`/health/ready=200`、缺任一配置时稳定 fail-closed。

### P0-4：修复私有 COS 合同

规范要求私有桶和不超过 15 分钟的预签名 PUT/GET；当前 adapter 返回无签名裸 URL，私有桶必然读失败，当前上传链也不是已冻结的预签名流程。

- [ ] 桶保持私有读写，禁止匿名读取和 ListBucket。
- [ ] Admin 向 API 申请短时预签名 PUT；object key 由服务端生成。
- [ ] 限制 MIME、大小、摘要、object prefix 和有效期；上传后必须 confirm。
- [ ] Mini/PC 读取使用短时预签名 GET 或同等私有读取方案，不返回永久公有 URL。
- [ ] 商品图和开屏图均覆盖 0/1/3 图、对象不存在、过期签名、篡改 key、失败不漂移。
- [ ] Web Admin CORS 只允许 UAT/prod 指定 origin 和必要的 `PUT/GET/HEAD`，不得 `*`。
- [ ] 删除或重定义 `ORDER_COS_PUBLIC_ORIGIN`，避免继续暗示公有桶。

完成证据：真实私有 COS 上传/读取/过期/越权 Gate；桶策略 readback；页面换设备仍可见。

### P0-5：补完整部署包和 UAT 启动流程

- [ ] 增加或冻结 Nginx 配置：`:80` 仅跳 HTTPS；`:443` 承载 PC 静态文件并反代 `/api/`；`127.0.0.1:8080` 不暴露公网。
- [ ] 冻结 release artifact 内容：`order-api`、`order-migrate`、`order-bootstrap`、systemd helpers、Web Admin 静态文件、Nginx 配置和 SHA256。
- [ ] 增加初始租户 bootstrap runbook；初始化且仅初始化 storefront、折扣和首个 OWNER。
- [ ] 配置 UAT/production 的 Mini `trial` / `release` endpoint，不允许空值。
- [ ] 执行 migration → bootstrap → order-api → readiness → smoke 的单向流程。
- [ ] 验证 CLS、LogListener、CAT、云监控、证书到期、备份失败和支付对账告警。
- [ ] 完成 CDB 备份恢复演练和 CVM 重建演练；记录实际 RPO/RTO，不复述目标值。

完成证据：UAT URL、systemd 状态、Nginx/HTTPS、schema v44、bootstrap outcome、readiness、监控告警和恢复 receipt。

## 4. 甲方/云管理员现在就应发起的申请

资源申请与第 3 节代码修复并行进行。账号和资产必须归甲方企业主体；研发只获取最小、限时权限。

### 4.1 主体与基础决策

- [ ] 确认营业执照主体、腾讯云实名认证主体、域名实名/备案主体、小程序认证主体、微信支付商户主体和收款主体一致。
- [ ] 确定中国大陆地域 `TODO_TENCENT_REGION`；UAT/prod 的 CVM、CDB、COS 同地域。
- [ ] 确定 UAT/prod 月度预算上限、成本告警阈值和付款责任人。
- [ ] 确定业务 owner、云管理员、微信平台管理员、支付管理员、技术 owner、告警接收人。
- [ ] 确定 UAT 与生产域名；两环境不得共用证书、回调地址或数据库。

申请单只记录企业主体和资源标识，不附主体证件、个人证件或联系方式正文。

### 4.2 微信小程序账号与权限

- [ ] 注册并企业认证 UAT/生产小程序，取得两个 AppID；AppSecret 只进入对应环境 SSM。
- [ ] 完成餐饮类目和小程序备案。
- [ ] 配置隐私保护指引，声明手机号、头像昵称、订单联系人等用途和保存范围。
- [ ] 开通真实 `getPhoneNumber` 能力。
- [ ] 将接手研发的实名微信账号加入“开发者”；将 UAT 人员加入“体验成员”。
- [ ] 配置 UAT/prod `request`、`uploadFile`、`downloadFile` 合法域名。
- [ ] 配置客服能力，验证 `open-type="contact"`。
- [ ] 申请一次性订阅模板：
  - 待取餐提醒：订单号、取餐日期、取餐时间、取餐点。
  - 退款结果：订单号、退款结果。
- [ ] 提供模板 ID 和字段 key 的非敏感配置结果；不得把管理员密码交给研发。

完成证据：平台后台状态截图/导出记录只保留非敏感部分；开发者与体验成员可登录；体验版可打开。

### 4.3 微信支付账号、产品和 API 权限

- [ ] 开通甲方微信支付商户号并完成结算账户验证。
- [ ] 关联 UAT/生产 AppID，完成双方授权确认。
- [ ] 开通 JSAPI 支付。
- [ ] 确认商户订单查询接口可用，用于丢回调对账补单。
- [ ] 开通全额退款及退款查询。
- [ ] 开通交易账单下载/查询，用于 PC 财务对账。
- [ ] 设置 APIv3 key。
- [ ] 申请商户 API 证书/私钥，记录证书序列号。
- [ ] 获取并维护微信支付公钥/平台证书及序列号。
- [ ] 配置 UAT/prod HTTPS 支付回调和退款回调；不能使用 IP、HTTP 或共享回调。
- [ ] 批准小额真实资金 UAT 预算和执行窗口；建议上限 ¥50，实际金额由甲方书面确认。
- [ ] 指定可查询交易、退款、结算和账单的甲方支付管理员。

敏感材料由甲方管理员直接写入 SSM，或在受控会话中协助写入；研发不长期持有平台主账号。

### 4.4 腾讯云企业账号与 CAM

- [ ] 企业账号完成实名认证并开启主账号 MFA。
- [ ] 主账号不共享；为研发创建实名 CAM 子用户或联合身份。
- [ ] 人员权限拆分：
  - 部署角色：只管理指定 UAT/prod CVM、systemd 发布目录和受限运维入口。
  - 数据库运维角色：只读 CDB 状态、备份和监控；业务账号与 migration 账号分离。
  - 对象存储角色：只管理指定桶的配置和对象前缀。
  - 日志监控角色：只查询指定 CLS 日志主题、指标和告警。
  - DNS/证书角色：只管理本项目域名记录和证书绑定。
- [ ] 为 UAT/prod CVM 分别创建 CAM Role；不发放长期云 API key。
- [ ] 开启操作审计，保留关键授权、密钥轮换和资源变更记录。

### 4.5 UAT 腾讯云资源

先创建 UAT 并跑容量门，再确定生产最终 SKU。

- [ ] 独立 UAT VPC、子网、路由表和安全组。
- [ ] 一台中国大陆 CVM：公网只开放 80/443 和受限运维入口；8080、3306 不开放公网。
- [ ] 独立 TencentDB MySQL 8.0：只允许 UAT CVM 内网访问；TLS required。
- [ ] 每日全量备份，数据备份和 binlog 保留 14 天；破坏性 migration 前手动备份。
- [ ] 独立私有 COS：版本控制开启；非当前版本 30 天删除；匿名读、匿名写、列举全部关闭。
- [ ] 四个 UAT SSM Secret，SecretName 使用 `order-uat-*`，正文由甲方写入。
- [ ] UAT 域名、DNS、ICP 接入和 HTTPS 证书。
- [ ] CLS 日志集/主题和 LogListener，保留 30 天。
- [ ] 云监控和 CAT；公网每分钟探测，连续 3 次失败告警。
- [ ] 成本预算和异常费用告警。

### 4.6 生产腾讯云资源

UAT 使用候选 SKU 通过 200 并发、300 单/5 分钟容量门后再采购：

- [ ] 独立生产 VPC、子网、路由表和安全组。
- [ ] 单台生产 CVM；不预购 Kubernetes、CLB 或多 CVM。
- [ ] TencentDB MySQL 8.0 双节点、多可用区、一主一备。
- [ ] 独立生产私有 COS、SSM、CAM Role、CLS、云监控和 CAT。
- [ ] 独立生产域名、ICP备案接入、HTTPS 证书和回调地址。
- [ ] 生产备份、恢复演练、证书到期和容量告警。

一期明确不申请：Kubernetes、微服务、Redis、MQ、CLB、CDN、数据库代理、读写分离、跨地域灾备或 7×24 人工值守。只有容量或 RTO 实测失败才能另立 change。

### 4.7 UAT 业务账户、人员和经营资料

- [ ] 真实访客微信账号 1 个。
- [ ] 真实员工微信账号 1 个；对应启用的手机号 + 姓名白名单。
- [ ] 真实 OWNER 微信账号至少 2 个，避免最后 OWNER 操作测试阻塞。
- [ ] 真实 SUBACCOUNT 微信账号 1 个。
- [ ] PC 扫码登录用手机和商户小程序扫码人员。
- [ ] 真实手机号授权、头像昵称、客服和订阅测试人员均取得本人同意。
- [ ] 正式门店名称、地址、单取餐点、公告、营业状态。
- [ ] 午/晚餐截单时间、取餐时间点和服务日期。
- [ ] 正式分类、商品、价格、餐段归属、最多 3 张图片和开屏 PNG。
- [ ] 全局员工折扣率和员工名单；名单不进 Git/聊天/文档。
- [ ] 退款原因和实际运营负责人。

## 5. 资源到位后的真实用户验收

所有步骤必须在开发者/体验成员可打开的 UAT 小程序上执行，并绑定部署 SHA、AppID 环境、API 域名、订单号的脱敏引用及服务端事实。不得用本地 fake、Chrome simulate 或 HTTP 200 替代。

### 5.1 登录、手机号和身份

- [ ] 新访客首次启动：静默登录，直接进入用户首页，不弹手机号。
- [ ] 浏览菜单、商品详情和购物车仍不要求手机号。
- [ ] 第一次结算拒绝手机号：不写绑定、不建 Quote、不调支付、购物车保留。
- [ ] 再次结算同意手机号：服务端绑定成功；员工与访客价格正确。
- [ ] 附加手机号只有“手机号 + 姓名”都命中才获得员工价。
- [ ] OWNER/SUBACCOUNT 首次商户绑定和后续身份选择正确。
- [ ] 退出登录后旧 token 401；重新登录可恢复服务端订单。

### 5.2 点单、预约和支付

- [ ] 选择今天/明天、午/晚餐和离散取餐时间。
- [ ] 已截单餐段不可选；失败回时间选择且购物车保留。
- [ ] 商品餐段、上下架、当日售罄、口味、备注和数量正确。
- [ ] Quote 的原价、员工折扣、应付金额与菜单/详情一致。
- [ ] 第一次真实 `wx.requestPayment` 取消：不生成订单；同一预支付可重试。
- [ ] 第二次真实支付成功：回调或 Query 确认后才生成订单和四位取餐号。
- [ ] 人工丢弃支付回调：Query 能补建；补建 SQL 故障进入待处理而不丢支付事实。
- [ ] 距取餐恰好 30 分钟和不足 30 分钟的状态/取消入口符合契约。

### 5.3 订单、退款和订阅

- [ ] 用户能看到已预约、制作中、待取餐、已完成、退款中、已退款六态。
- [ ] 已预约且大于 30 分钟可取消；真实退款成功后进入已退款。
- [ ] 退款结果未知时保持退款中并进入后台待处理，不假成功。
- [ ] 支付成功页请求待取餐订阅；拒绝后首页和取餐码页仍有补订阅入口。
- [ ] 商户标 READY 后真实收到待取餐通知。
- [ ] 真实退款完成后收到退款结果通知。

### 5.4 核销、商户和 PC

- [ ] READY 才展示二维码；非 READY 没有 token。
- [ ] 真机相机扫码核销成功；同码重复扫返回第一次结果，不重复统计。
- [ ] 手工取餐号只核销当前营业日，跨日同号不误核销。
- [ ] SUBACCOUNT 只能执行四项现场动作，不能进入 PC OWNER API。
- [ ] OWNER PC 扫码登录、角色伪造拒绝、最后 OWNER 保护均通过。
- [ ] PC 商品/分类/图片、开屏、名单、导入、订单、退款、财务、看板和审计全部从服务端事实读取。

### 5.5 COS、账单和故障恢复

- [ ] 真实私有 COS 上传商品图和开屏图，Mini/PC 可读；换设备可见。
- [ ] 过期签名、对象删除、无权限、非法对象均可见失败且不漂移业务事实。
- [ ] 交易账单下载，对真实支付/退款逐笔核对。
- [ ] 回调乱序/重复、worker 重启、临时微信 5xx、CDB 短暂不可用均可恢复且无重复副作用。
- [ ] 备份恢复或克隆后核对 schema、订单、支付、退款、核销和审计。

## 6. 接手研发的执行顺序

1. **Day 0**：读本 handoff、PRD、技术文档和 95-case 矩阵；冻结 P0 owned paths。
2. **Day 1–3**：完成会话撤销、生产配置/SSM/CAM、私有 COS 和部署包修复；同一时间甲方申请第 4 节资源。
3. **Day 3–4**：生成新 candidate，在独立 worktree 跑 Go、Mini、PC 和 95-case exact ledger。
4. **资源 READY 后 Day 1**：部署 UAT、迁移、bootstrap、配置体验版和监控。
5. **资源 READY 后 Day 2–3**：执行第 5 节真实用户/真实支付/退款/扫码/订阅/COS 验收，修复首个真实错误并重跑。
6. **Day 4**：跑容量门、恢复演练、最终 detached verification 和客户体验版验收。
7. **客户确认后**：生产部署、提审；审核通过后由甲方确认发布。

预估：外部资源全部 READY 后仍需约 **5–7 个研发工作日**。若小程序、商户号、域名和备案从零开始，平台日历时间按 **2–4 周**准备；平台审核时间不由研发承诺。

## 7. 必须生成的非敏感证据

- [ ] 最终 Git SHA、MR、独立 verifier SHA、干净 worktree。
- [ ] Go test/vet、Mini 默认 Gate、PC Gate、95-case exact ledger。
- [ ] UAT/production artifact SHA256 和实际部署 release ID。
- [ ] production-mode startup、migration、bootstrap、readiness。
- [ ] 22 项 L4 从 `BLOCKED_EXTERNAL` 转为 `READY` 的逐项 receipt。
- [ ] 真实支付、Query、退款、交易账单的脱敏核对结果。
- [ ] 真机登录、手机号、订阅、相机扫码、PC 扫码、COS 截图/录屏或平台记录。
- [ ] CAT/CLS/云监控告警和 CDB/CVM/COS 恢复演练。
- [ ] 客户体验版验收确认、平台审核结果、客户发布确认。

证据不得包含凭据、证书正文、回调原文、完整交易号、真实手机号、OpenID 或人员名单。

## 8. 最终 GO / NO-GO DoD

以下任一项未完成均为 NO-GO：

- [ ] P0-1 至 P0-5 全部关闭且无 P0/P1 finding。
- [ ] 最终代码已合并远端主分支，并从该精确 SHA 构建/部署。
- [ ] 95 个本地 Case 在最终 SHA 重新执行为 PASS。
- [ ] 22 个 L4 external Case 有真实平台证据，不再 `BLOCKED_EXTERNAL`。
- [ ] 第 5 节所有真实用户步骤通过，包括退出后旧 token 401。
- [ ] UAT 容量门通过；生产 SKU、预算和告警阈值已据此确认。
- [ ] UAT/production 资源、数据、域名、证书、AppID、回调和 SSM 完全隔离。
- [ ] 备份、恢复、监控和应急联系人可用。
- [ ] 甲方完成体验版验收并书面确认；正式审核与发布仍由甲方管理员确认。

## 9. 当前关键路径索引

- 本地验收工具：`tools/order-acceptance/`
- Mini 环境 endpoint：`apps/wechat-miniprogram/utils/runtimeEndpointConfig.js`
- Mini 退出行为：`apps/wechat-miniprogram/components/navbar/navbar.js`
- PC 退出行为：`apps/web-admin/data/api.js`
- 生产组合根：`services/api/cmd/order-api/main.go`
- 生产配置/SSM：`services/api/internal/config/`
- COS adapter：`services/api/internal/objectstore/cos_adapter.go`
- systemd/env/preflight：`deploy/systemd/`
- 运行检查：`deploy/checks/`
- CVM 恢复：`deploy/runbooks/cvm-recovery.md`
- 客户操作指南：[`微信小程序开发和运维指南`](../微信小程序开发和运维指南/)
