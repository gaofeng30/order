## ADDED Requirements

### Requirement: Production topology has one deployment shape

一期生产架构 MUST 固定为单台中国境内 CVM：Nginx 在 `:443` 终止 TLS、承载 Web Admin 静态文件并反向代理至只监听 `127.0.0.1:8080` 的 `order-api`；HTTP `:80` 只重定向 HTTPS。`order-api` 与同进程 worker MUST 作为同一个 Go/Gin 模块化单体由 systemd 管理。

生产数据 MUST 使用同地域 TencentDB MySQL 8.0 双节点多可用区、一主一备；商品图片 MUST 使用同地域私有 COS；密钥 MUST 使用 SSM 与 CAM 临时身份；日志、指标和公网探测 MUST 分别使用 CLS、云监控和 CAT。CDB MUST 只允许来自生产 CVM 的内网访问，公网安全组 MUST 只开放业务 HTTPS 和受限运维入口。

本基线 MUST 排除 Kubernetes、微服务、CloudBase Run、Docker 运行依赖、Redis、MQ、读写分离、数据库代理、CLB、CDN、跨地域灾备和 7×24 人工值守；后续 change 不得把这些组件解释为本基线的隐藏前置或默认扩展。

#### Scenario: Production request enters the system
- **WHEN** 小程序或 Web Admin 调用生产 API
- **THEN** 请求经 HTTPS 到达 Nginx，再转发给本机 `order-api`
- **AND** 业务事实只持久化到生产 MySQL，图片二进制只持久化到生产私有 COS

#### Scenario: An excluded component is proposed as a default dependency
- **WHEN** 后续 change 在没有本 spec 声明的实测升级条件时引入 Redis、MQ、CLB、CDN、只读实例或新的服务部署单元
- **THEN** 架构依赖检查失败
- **AND** 该组件必须拆为独立 change 并提供触发证据

### Requirement: Synchronous transactions preserve one durable truth boundary

同步请求的持久化副作用 MUST 只发生在 MySQL 事务中。Handler MUST 先完成鉴权、输入与幂等键校验，再在单个事务内写业务状态、审计记录及必要 outbox；外部网络调用 MUST NOT 位于 MySQL 事务内，事务提交后才能返回已持久化结果或执行外部调用。

微信预支付 MUST 使用稳定且唯一的 `out_trade_no`：先在 MySQL 事务创建订单与支付尝试，提交后调用微信；进程中断后 MUST 以同一 `out_trade_no` 重试或查单，不得创建第二笔支付尝试。微信支付是外部结算事实源，MySQL MUST 保存应用侧不可变支付事件和当前业务投影；前端跳转不得作为支付成功依据。

#### Scenario: Business state and an asynchronous effect are accepted
- **WHEN** 同步请求既改变业务状态又要求事务提交后执行副作用
- **THEN** 业务状态、审计和 outbox 在同一个 MySQL 事务原子提交
- **AND** 外部副作用只由事务提交后的 worker 执行

#### Scenario: Prepayment call is interrupted
- **WHEN** 订单与支付尝试已提交，但微信预支付响应前进程中断
- **THEN** 重试使用原 `out_trade_no` 或先向微信查单
- **AND** 不得新增第二笔支付尝试或长时间持有数据库事务

### Requirement: Inbox, outbox, and worker form one durable asynchronous boundary

外部支付与退款通知 MUST 在验签和解密通过后，以稳定通知键写入 `inbox_events` 唯一记录；inbox 事务提交成功后 MUST 在通知到达后的 5 秒内返回 HTTP 200/204。唯一键冲突 MUST 视为已持久化重复通知并返回成功；验签、解密或持久化失败 MUST NOT 返回成功。

业务事务产生的异步工作 MUST 同事务写入 `outbox_events`。同进程 worker MUST 使用 MySQL 8.0 `SELECT ... FOR UPDATE SKIP LOCKED`、`status`、`available_at`、`lease_until` 和 `attempts` 分批租用任务；消费语义 MUST 为 at-least-once，所有消费者 MUST 幂等，且 MUST NOT 宣称 exactly-once。

外部调用 MUST 在领取事务提交后执行；结果 MUST 在新事务确认。任务失败 MUST 指数退避且单次退避不超过 15 分钟；同一事件第 10 次执行失败后 MUST 进入 `DEAD` 并立即告警。worker 停止时 MUST 停止领取新任务，未确认任务 MUST 在 lease 到期后重试；事件正文不得保存敏感凭据或个人信息。

#### Scenario: Payment notification is delivered repeatedly
- **WHEN** 微信以同一稳定通知键重复发送已通过验签和解密的通知
- **THEN** `inbox_events` 只保留一条唯一事件且每次均在持久化确认后成功应答
- **AND** 业务消费者不得重复扣减、释放、退款或推进订单状态

#### Scenario: Worker crashes after an external call
- **WHEN** worker 已完成外部调用但在确认 outbox 前崩溃
- **THEN** lease 到期后事件会再次领取
- **AND** 幂等键使重试不产生重复副作用

#### Scenario: Event fails ten executions
- **WHEN** 同一 inbox 或 outbox 事件第 10 次执行仍失败
- **THEN** 事件状态进入 `DEAD` 且不再自动执行
- **AND** 系统立即产生包含非敏感事件 ID 和枚举原因的告警

### Requirement: Schema migration is explicit, forward-only, and deployment-gated

数据库 migration MUST 由独立入口 `order-migrate` 在部署新版本前执行。每个 migration MUST 使用单调递增编号的 forward-only SQL，并记录到 `schema_migrations`；runner MUST 通过 `GET_LOCK('order_schema_migrate', 30)` 防止并发迁移。

生产 `order-api` 启动时 MUST NOT 自动执行 migration 或 down migration。`/health/ready` MUST 同时验证数据库连接和应用支持的 schema version；数据库不可达、版本缺失或版本不兼容时 MUST 返回 not ready。

回滚 MUST 使用兼容当前 schema 的上一版二进制、forward fix 或已验证备份恢复；生产不得自动 down。删除列、缩窄字段或不可逆数据改写前 MUST 创建手动备份并确认恢复点。

#### Scenario: Two migration runners start together
- **WHEN** 两个 `order-migrate` 同时尝试迁移同一数据库
- **THEN** 只有取得命名锁的 runner 执行 migration
- **AND** 未取得锁的 runner 在 30 秒内失败退出而不修改 schema

#### Scenario: Application sees an incompatible schema
- **WHEN** `order-api` 启动后发现数据库连接失败或 schema version 不兼容
- **THEN** readiness 返回 not ready
- **AND** 应用不得自动迁移、自动 down 或对不兼容 schema 接受业务流量

### Requirement: Development, UAT, and production are isolated by resource and identity

dev、UAT、prod MUST 使用不同的运行配置、数据库、COS 桶、域名、SSM 前缀和微信身份。UAT 与 prod MUST 使用不同 VPC、CVM、CDB 实例、CAM Role、证书及回调地址；UAT MySQL 引擎版本 MUST 与生产一致，规格值由白名单外部占位符提供。

生产数据 MUST NOT 原样复制到 dev 或 UAT；需要样本时 MUST 先匿名化。单个进程 MUST 只绑定一个环境，且不得通过运行时参数同时访问两个环境。

#### Scenario: UAT is provisioned
- **WHEN** 平台 change 配置 UAT
- **THEN** UAT 使用独立资源标识、权限、域名、桶、数据库和微信身份
- **AND** 不读取或写入任何生产资源

#### Scenario: Production records are needed for testing
- **WHEN** 测试需要生产形态的数据样本
- **THEN** 只能使用经过不可逆匿名化的数据副本
- **AND** 原始个人信息、支付标识和用户备注不得进入 dev 或 UAT

### Requirement: Runtime configuration, secrets, and COS access are least-privileged

非敏感配置 MUST 存入 root-owned、权限不宽于 `0640` 的 systemd EnvironmentFile。数据库密码、微信商户私钥、APIv3 key、会话签名 key 等密钥 MUST 存入 `/order/{env}/...` SSM 路径；应用 MUST 通过只能读取本环境前缀的 CVM CAM Role 在启动时获取，获取失败 MUST 阻止启动。密钥轮换 MUST 通过更新 SSM 后重启进程加载。

UAT 与生产 COS MUST 为私有桶并禁止匿名读取和列举。Admin MUST 先向 API 申请有效期不超过 15 分钟的预签名 PUT，再直传 COS 并调用确认接口；客户端图片读取 MUST 使用有效期不超过 15 分钟的预签名 GET。对象 key MUST 由服务端生成，上传 MUST 校验允许的 MIME、大小和摘要；桶 MUST 开启版本控制，非当前版本保留 30 天后删除。

仓库、文档、镜像、命令行、日志和前端配置 MUST NOT 包含长期 SecretId/SecretKey、私钥、证书正文、APIv3 key 或其他可重放凭据。

#### Scenario: Application starts without secret access
- **WHEN** CAM Role 无权读取当前环境的必要 SSM secret
- **THEN** `order-api` 启动失败并只记录脱敏错误类别
- **AND** 不回退到文件、命令行或内置长期凭据

#### Scenario: Client requests a product image
- **WHEN** 已授权客户端读取私有 COS 商品图片
- **THEN** API 返回有效期不超过 15 分钟的预签名 GET
- **AND** 桶保持私有且不得改为公有读私有写

### Requirement: Backup and recovery have fixed targets and honest evidence boundaries

生产 CDB MUST 每日自动全量备份，数据备份和 binlog MUST 保留 14 天；实例销毁后的备份 MUST 保留 7 天。COS 非当前版本 MUST 保留 30 天。团队 MUST 每季度把 CDB 备份恢复或克隆到隔离实例，并校验 schema version、订单、支付、退款、核销和操作日志；CVM MUST 不保存业务事实并能由发布物、Nginx/systemd 配置和 SSM 重建。

文档 MUST 把以下数值写为未实测的验收目标，而非当前已达成或云厂商 SLA：Go 进程与 CDB 主备故障 RTO `≤5 分钟`，CVM 丢失 RTO `≤60 分钟`，逻辑误删/损坏 RPO `≤5 分钟`、RTO `≤2 小时`。区域级灾难 MUST 明确为本阶段不承诺 RPO/RTO。

#### Scenario: Recovery targets are documented before a drill
- **WHEN** 恢复演练尚未在目标云规格执行
- **THEN** 文档只称上述 RPO/RTO 为目标且标记未实测
- **AND** 不得写成已达到、保证值或整体系统 SLA

#### Scenario: Quarterly recovery drill runs
- **WHEN** 平台 owner 从生产备份创建隔离恢复实例
- **THEN** 验证 schema 和五类关键业务记录并记录实际 RPO/RTO
- **AND** 恢复实例不得接收生产流量且演练完成后按授权清理

### Requirement: Logging, metrics, and alerts have fixed redaction and thresholds

`order-api` MUST 向 stderr 输出 JSON，LogListener MUST 采集到 CLS 并保留 30 天。请求日志 MUST 包含 `env`、`request_id`、`method`、模板化 `path`、`status`、`duration_ms`；业务日志可增加非敏感内部 `order_id`、`payment_id`、`event_id`，但关联 ID MUST NOT 作为指标 label。

日志、证据和告警 MUST NOT 记录 query、body、Authorization、Cookie、手机号、姓名、openid、工号、用户备注、回调原文、私钥、证书、APIv3 key、完整核销 token、完整二维码或完整外部交易号。

生产告警阈值 MUST 固定为：CAT 每分钟探测且连续 3 次失败；HTTP 5xx 五分钟比例 `>1%`；读取 p95 `>500ms` 或写入 p95 `>800ms` 持续 5 分钟；inbox/outbox 最老待处理事件 `>60 秒`；任一 `DEAD`、支付对账不一致或备份失败；CVM CPU `>70%` 持续 15 分钟、内存 `>80%`、磁盘 `>75%`；CDB CPU 或连接使用率 `>70%` 持续 15 分钟、存储 `>75%`；TLS 证书剩余 `<30 天`。告警可全天发送，但不构成 7×24 人工值守承诺。

#### Scenario: Sensitive request fails
- **WHEN** 含个人信息或支付内容的请求失败并产生日志
- **THEN** 日志只保留非敏感关联 ID、模板化 path、状态、耗时和枚举原因
- **AND** 原始请求、身份、凭据和外部交易号不进入 CLS 或告警

#### Scenario: Asynchronous event becomes stale
- **WHEN** 最老待处理 inbox/outbox 事件年龄超过 60 秒
- **THEN** 云监控立即发送包含环境和非敏感事件 ID 的告警
- **AND** 不等待 HTTP 错误率或 CPU 阈值同时触发

### Requirement: Capacity gate and upgrade order are evidence-driven

生产规格候选 MUST 通过以下容量门：200 并发用户，300 笔订单在 5 分钟内完成，数据基线至少包含 1000 日订单和 300 商品，并同时覆盖重复提交、重复/乱序支付回调、worker 重启和订单超时并发。门禁目标 MUST 为 HTTP 5xx `<0.1%`、读取 p95 `≤500ms`、写入 p95 `≤800ms`，且无重复订单、重复支付状态、重复核销、丢失 inbox/outbox 或违反库存不变量；未在真实候选规格执行前 MUST 标记未实测。

升级顺序 MUST 固定为：先检查和修复慢 SQL、索引、连接池及非必要查询；再纵向升级 CVM/CDB 并重跑同一容量门。只有上述步骤后仍不能通过容量门，或计算节点 RTO 被正式要求 `<5 分钟`，才能创建独立 CLB/多 CVM change。只有索引与纵向扩容后数据库读负载仍被实测为主要瓶颈，才能创建 Redis 或只读实例 change；只有数据库健康且增加 worker 并发后，最老任务仍连续 15 分钟超过 60 秒且排除外部限流，才能创建 MQ change。

#### Scenario: Initial SKU has not been load-tested
- **WHEN** 云账号、候选 SKU 或隔离压测环境尚未提供
- **THEN** 文档把容量数字写为验收门和未实测边界
- **AND** 不得声称当前 2 核 4G 或任一外部规格已经足够

#### Scenario: Team proposes horizontal expansion
- **WHEN** 团队准备加入 CLB、多 CVM、Redis、只读实例或 MQ
- **THEN** proposal 必须引用本 requirement 对应的真实失败证据和已完成的前置优化
- **AND** 没有证据时该架构 change 不得批准

### Requirement: Architecture documents have one vocabulary and bounded external placeholders

`design.md`、`docs/product/online-ordering-system-technical.md` 与`docs/微信小程序开发和运维指南/腾讯云操作指南.md` MUST 对本 spec 的组件、数据流、迁移、环境、安全、COS、恢复、观测、容量和非目标使用同一唯一口径。腾讯云指南 MUST 删除 MySQL/TDSQL-C 二选一、公有读 COS、长期 SecretId/SecretKey 交付和默认 CDN；较早文案不得作为并行可选方案继续存在。

三个文档只允许以下外部占位符：`TODO_CLOUD_ACCOUNT_ID`、`TODO_TENCENT_REGION`、`TODO_VPC_ID`、`TODO_UAT_CVM_INSTANCE_TYPE`、`TODO_PROD_CVM_INSTANCE_TYPE`、`TODO_UAT_CDB_INSTANCE_TYPE`、`TODO_PROD_CDB_INSTANCE_TYPE`、`TODO_UAT_DOMAIN`、`TODO_PROD_DOMAIN`、`TODO_UAT_COS_BUCKET`、`TODO_PROD_COS_BUCKET`、`TODO_CERTIFICATE_ID`、`TODO_UAT_WECHAT_APPID`、`TODO_PROD_WECHAT_APPID`、`TODO_WECHAT_MCH_ID`、`TODO_ALERT_RECIPIENTS`、`TODO_MONTHLY_BUDGET_CNY`。这些值 MUST 由真实云账号、客户平台身份、实际采购或预算提供，不得把行为、架构或安全决策伪装成占位符。

文档 MUST 使用腾讯云和微信支付官方 HTTPS 直链，并记录访问日期；本地相对链接 MUST 指向存在文件。正文 MUST NOT 包含真实账号标识、域名、桶名、AppID、商户号、IP、手机号、用户名、密码、token、私钥、证书或回调报文。

#### Scenario: Writer applies the architecture baseline
- **WHEN** writer 修改技术文档与腾讯云指南
- **THEN** 三份文档通过必备术语、禁用歧义、外部占位符白名单、链接、敏感信息和 owned-path 检查
- **AND** MySQL/TDSQL-C 二选一、公有读 COS、长期 SecretId/SecretKey 与默认 CDN 文案均不存在

#### Scenario: A missing architecture decision is encoded as a placeholder
- **WHEN** 文档出现白名单外占位符或使用“待定”“二选一”“按需选择”“视情况”等词留下实现选择
- **THEN** 内容完整性 Gate 失败
- **AND** change 不得形成 CANDIDATE
