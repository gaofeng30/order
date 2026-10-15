## MODIFIED Requirements

### Requirement: Production topology has one deployment shape

一期生产架构 MUST 使用独立 CloudBase 生产环境中的 CloudRun Go API、Shared PostgreSQL 官方 HTTPS RPC、同地域私有 COS、同源 HTTP 网关与静态托管。正式域名 MUST 使用有效 TLS；PC 唯一入口 MUST 为 `/pc/`，API 为 `/api/v1`。不得增加旧 PC 路径兼容。

运行时 MUST 使用普通专用数据库身份和固定 RPC allowlist，不得使用管理 SQL、管理 API key 或通用云凭据执行业务。材料 MUST 由 CloudRun 受保护版本配置注入；COS MUST 通过绑定本环境桶及 images 前缀的受限 STS 身份访问。源码、镜像与日志不得携带秘密。日志与指标 MUST 接入当前托管资源支持的 CLS/监控通道。

CVM/Nginx/systemd/MySQL/SSM 是被替代的历史路线，MUST NOT 作为并行发布入口。Redis、MQ、独立微服务、CLB、跨地域灾备和7×24人工值守不属于本次已确认范围。新增费用或权限不得由技术适配自动获得授权。

#### Scenario: Production request enters the system
- **WHEN** 小程序或 PC 调用正式 API
- **THEN** 请求经正式 HTTPS 网关到 CloudRun order-api
- **AND** 业务事实只持久化到独立 Shared PG，图片二进制只持久化到生产私有 COS

#### Scenario: An excluded component is proposed as a default dependency
- **WHEN** 后续实现试图引入范围外资源
- **THEN** 必须给出已有资源不满足同一验收的实测证据
- **AND** 新费用、权限及生产写入必须获得相应授权

### Requirement: Synchronous transactions preserve one durable truth boundary

同步请求的持久化副作用 MUST 通过固定 SECURITY INVOKER RPC 在单个 PostgreSQL 事务中原子完成。Handler MUST 先完成鉴权、输入与幂等键校验；业务状态、审计与必要异步意图 MUST 原子提交。外部网络调用 MUST NOT 位于数据库事务内。

微信预支付 MUST 先持久化预支付记录及稳定唯一 out_trade_no，再调用微信；只有已验证支付成功观察才可生成订单。中断后 MUST 查询或收敛原交易，不创建第二笔支付。微信是外部结算事实源，数据库 MUST 保留不可变支付观察与当前业务投影，前端跳转不构成支付成功。

#### Scenario: Business state and an asynchronous effect are accepted
- **WHEN** 请求改变业务状态且需要事务后的副作用
- **THEN** 状态、审计与通知或补偿意图同事务提交
- **AND** 外部调用在提交之后执行

#### Scenario: Prepayment call is interrupted
- **WHEN** 预支付记录提交后，微信响应前进程中断
- **THEN** 使用同一 out_trade_no 查询或重试既有操作
- **AND** 不创建第二笔预支付，也不提前创建订单

### Requirement: Inbox, outbox, and worker form one durable asynchronous boundary

支付/退款回调 MUST 验签、解密、校验环境与交易事实后，持久化到 payment_observations / refund_observations，以稳定键去重。持久化确认后 MUST 在回调到达后5秒内成功应答；验签、解密或持久化失败不得返回成功。

通知意图 MUST 随业务事务写入 notification_outbox。正常支付/退款请求、回调及通知首次发送 MUST 及时执行；补偿 MUST 仅由真实异常登记的持久待办驱动，失败后30分钟再次执行，成功停止。永久错误或需人工处理的状态 MUST 停止自动重放并告警。不得对正常待支付无限查单，不得在无异常时固定周期扫描。

任务 MUST 使用数据库原子领取、租约、版本/owner校验与幂等业务键；外部调用在领取事务之后，确认在另一事务中。消费是 at-least-once，不得宣称 exactly-once。可靠唤醒 MUST 覆盖提交待办后未发唤醒即崩溃、重启、断连与多实例交接；内存timer不能作为持久事实或唯一恢复机制。平台能力与权限未经实测不得登记完成。

自动进入制作中 MUST 依据服务端时间计算，不恢复周期制作扫描；取消资格仍要求距取餐严格大于30分钟。列表、详情、筛选、统计和写入动作 MUST 保持同一边界语义。

#### Scenario: Payment notification is delivered repeatedly
- **WHEN** 同一可信回调重复到达
- **THEN** 相同业务观察去重且只在确认持久化后应答
- **AND** 不重复创建订单、退款或核销

#### Scenario: Worker crashes after an external call
- **WHEN** 外部调用已完成但持久确认前进程退出
- **THEN** 重启或其他实例通过持久待办与租约恢复同一业务对象
- **AND** 先核实未知结果，避免重复资金副作用

#### Scenario: A retry fails and later succeeds
- **WHEN** 已登记的真实异常补偿失败
- **THEN** 持久化下一次执行时间为失败后的30分钟
- **AND** 成功后停止；无异常期间没有固定周期全局扫描

### Requirement: Schema migration is explicit, forward-only, and deployment-gated

关系迁移和函数安装 MUST 从精确候选分别生成，经审核后由单一执行者通过官方管理通道执行。关系账本、平台迁移历史、函数定义及ACL/RLS MUST 分别核对；已成功执行的迁移不得重复。执行必须使用事务与数据库锁防止并发，设置不超过30秒的锁等待预算；平台成功响应仍须读回。

运行时 MUST NOT 自动迁移、bootstrap或down。readiness MUST 通过普通身份校验函数定义、SECURITY INVOKER、角色、表/序列权限、RLS与关系账本；管理SQL成功不能证明运行身份可用。基线117实体/113可执行函数不替代最终候选实际集合；历史113/109收据不得转记新版本。

回退 MUST 使用已验证兼容当前SQL的应用版本、forward fix或经过演练的恢复。未知执行结果先只读核实；不得清dirty、删除历史或用Repair掩盖执行失败。

#### Scenario: Two migration runners start together
- **WHEN** 两个执行者尝试同一安装或迁移
- **THEN** 只有持有数据库锁者可写入
- **AND** 另一个在锁等待预算内失败且不篡改账本

#### Scenario: Application sees an incompatible schema
- **WHEN** 普通身份、定义、ACL/RLS或账本不符合候选
- **THEN** readiness返回not ready
- **AND** 不自动安装或接收业务流量

### Requirement: Development, UAT, and production are isolated by resource and identity

UAT与生产 MUST 使用独立 CloudBase EnvId、Shared PG、普通数据库用户/UID、COS桶与受限来源身份、域名、回调和运行配置。已确认的小程序AppID及商户归属不因环境适配而更换；共享商户时 MUST 校验每笔交易冻结的环境、AppID、商户和回调归属。

单进程 MUST 绑定唯一环境；UAT key/桶/UID/回调不得被生产接纳，生产身份不得访问UAT。生产不得复制UAT示例数据/人员/凭据；生产数据不原样复制到隔离验证。

#### Scenario: UAT is provisioned
- **WHEN** 配置UAT
- **THEN** 使用独立EnvId、数据库、桶、身份与回调
- **AND** 不读写生产资源

#### Scenario: Production records are needed for testing
- **WHEN** 需要生产形态样本
- **THEN** 使用合成或不可逆匿名化数据
- **AND** 不携带真实个人信息、外部支付标识和备注

### Requirement: Runtime configuration, secrets, and COS access are least-privileged

运行配置和业务材料 MUST 由 CloudRun 受保护版本配置注入，必要材料缺失/格式错误/环境不匹配必须阻止启动。仅记录安全位置与版本引用，不输出值。数据库只使用专用普通身份；COS来源身份只允许取得限定本环境images前缀的临时权限。轮换必须发布新配置并读回验证，不得回退至已撤销材料。

UAT 与生产 COS MUST 为私有桶并禁止匿名读取和列举。Admin MUST 先向 API 申请有效期不超过 15 分钟的预签名 PUT，再直传 COS 并调用确认接口；客户端图片读取 MUST 使用有效期不超过 15 分钟的预签名 GET。对象 key MUST 由服务端生成，上传 MUST 校验允许的 MIME、大小和摘要；桶 MUST 开启版本控制，非当前版本保留 30 天后删除。

仓库、文档、镜像、命令行、日志和前端配置 MUST NOT 包含长期 SecretId/SecretKey、私钥、证书正文、APIv3 key 或其他可重放凭据。

#### Scenario: Application starts without secret access
- **WHEN** CloudRun未提供当前环境的必要材料
- **THEN** `order-api` 启动失败并只记录脱敏错误类别
- **AND** 不回退到文件、命令行或内置长期凭据

#### Scenario: Client requests a product image
- **WHEN** 已授权客户端读取私有 COS 商品图片
- **THEN** API 返回有效期不超过 15 分钟的预签名 GET
- **AND** 桶保持私有且不得改为公有读私有写

### Requirement: Backup and recovery have fixed targets and honest evidence boundaries

生产 Shared PG 的每日备份、14天保留、实例销毁后7天保留以及时间点恢复能力 MUST 按目标环境实际套餐/API核实并验证；这些是验收要求，不是已提供能力的声明。COS 非当前版本 MUST 保留 30 天。团队 MUST 每季度把 Shared PG 备份恢复或克隆到隔离目标，并校验 schema version、订单、支付、退款、核销和操作日志；CloudRun实例 MUST 不保存业务事实并能由镜像、版本配置及材料安全引用重建。

文档 MUST 把以下数值写为未实测的验收目标，而非当前已达成或云厂商 SLA：Go进程与数据库故障 RTO `≤5 分钟`，计算节点丢失 RTO `≤60 分钟`，逻辑误删/损坏 RPO `≤5 分钟`、RTO `≤2 小时`。区域级灾难 MUST 明确为本阶段不承诺 RPO/RTO。

#### Scenario: Recovery targets are documented before a drill
- **WHEN** 恢复演练尚未在目标云规格执行
- **THEN** 文档只称上述 RPO/RTO 为目标且标记未实测
- **AND** 不得写成已达到、保证值或整体系统 SLA

#### Scenario: Quarterly recovery drill runs
- **WHEN** 平台 owner 从生产备份创建隔离恢复实例
- **THEN** 验证 schema 和五类关键业务记录并记录实际 RPO/RTO
- **AND** 恢复实例不得接收生产流量且演练完成后按授权清理

### Requirement: Logging, metrics, and alerts have fixed redaction and thresholds

`order-api` MUST 向 stderr 输出 JSON，CloudRun日志采集 MUST 接入 CLS 并保留 30 天。请求日志 MUST 包含 `env`、`request_id`、`method`、模板化 `path`、`status`、`duration_ms`；业务日志可增加非敏感内部 `order_id`、`payment_id`、`event_id`，但关联 ID MUST NOT 作为指标 label。

日志、证据和告警 MUST NOT 记录 query、body、Authorization、Cookie、手机号、姓名、openid、工号、用户备注、回调原文、私钥、证书、APIv3 key、完整核销 token、完整二维码或完整外部交易号。

生产告警阈值 MUST 固定为：CAT 每分钟探测且连续 3 次失败；HTTP 5xx 五分钟比例 `>1%`；读取 p95 `>500ms` 或写入 p95 `>800ms` 持续 5 分钟；持久任务最老已到期未处理事件超期 `>60 秒`（按 next_attempt_at/next_query_at/next_reconcile_at 或租约到期时间计算，30分钟退避中未到期的任务不计入超期）；任一 `DEAD`、支付对账不一致或备份失败；CloudRun CPU `>70%` 持续 15 分钟、内存 `>80%`、磁盘 `>75%`；Shared PG CPU 或连接使用率 `>70%` 持续 15 分钟、存储 `>75%`；TLS 证书剩余 `<30 天`。告警可全天发送，但不构成 7×24 人工值守承诺。

#### Scenario: Sensitive request fails
- **WHEN** 含个人信息或支付内容的请求失败并产生日志
- **THEN** 日志只保留非敏感关联 ID、模板化 path、状态、耗时和枚举原因
- **AND** 原始请求、身份、凭据和外部交易号不进入 CLS 或告警

#### Scenario: Asynchronous event becomes stale
- **WHEN** 持久任务已到期但尚未处理，超期超过 60 秒
- **THEN** 云监控立即发送包含环境和非敏感事件 ID 的告警
- **AND** 不等待 HTTP 错误率或 CPU 阈值同时触发

### Requirement: Capacity gate and upgrade order are evidence-driven

生产规格候选 MUST 通过以下容量门：200 并发用户，300 笔订单在 5 分钟内完成，数据基线至少包含 1000 日订单和 300 商品，并同时覆盖重复提交、重复/乱序支付回调、worker 重启和订单超时并发。门禁目标 MUST 为 HTTP 5xx `<0.1%`、读取 p95 `≤500ms`、写入 p95 `≤800ms`，且无重复订单、重复支付状态、重复核销、丢失 inbox/outbox 或违反库存不变量；未在真实候选规格执行前 MUST 标记未实测。

升级顺序 MUST 固定为：先检查和修复慢 SQL、索引、连接池及非必要查询；再按批准的CloudRun/Shared PG资源调整重跑同一容量门。只有上述步骤后仍不能通过容量门，或计算节点 RTO 被正式要求 `<5 分钟`，才能创建独立扩容 change。只有索引与纵向扩容后数据库读负载仍被实测为主要瓶颈，才能创建 Redis 或只读实例 change；只有数据库健康且增加 worker 并发后，最老已到期任务的超期时间仍连续 15 分钟超过 60 秒且排除外部限流，才能创建 MQ change。

#### Scenario: Initial SKU has not been load-tested
- **WHEN** 云账号、候选 SKU 或隔离压测环境尚未提供
- **THEN** 文档把容量数字写为验收门和未实测边界
- **AND** 不得声称当前 2 核 4G 或任一外部规格已经足够

#### Scenario: Team proposes horizontal expansion
- **WHEN** 团队准备加入范围外计算/缓存/只读实例或MQ
- **THEN** proposal 必须引用本 requirement 对应的真实失败证据和已完成的前置优化
- **AND** 没有证据时该架构 change 不得批准

### Requirement: Architecture documents have one vocabulary and bounded external placeholders

生产唯一执行入口 MUST 为 deploy/runbooks/order-production-release.md。UAT部署与运行文档仅维护其环境和历史回执，生产handoff只索引生产入口；旧CVM/MySQL、旧PC路径及过期worker说明不得形成有效生产指令。历史回执保留原时间/环境/版本，不得转记当前发布。

未知域名、普通身份、预算责任和平台资格 MUST 标记未核实，不得填写示例冒充正式值。公开资源标识与环境绑定可记录；任何秘密、个人信息及可重放材料不得写入仓库。官方资料记录访问日期，本地链接必须存在。

#### Scenario: Writer applies the architecture baseline
- **WHEN** 更新生产适配与文档
- **THEN** 发布路线只有CloudBase/CloudRun、Shared PG与私有COS，PC只有正式域名/pc/
- **AND** 未完成的可靠补偿、恢复、容量和真实验收明确保持NO-GO

#### Scenario: A missing production value is encoded as an example
- **WHEN** 正式域名、身份或材料尚未读回
- **THEN** 禁止把UAT或示例值作为生产配置
- **AND** 继续独立本地工作，发布项保持未执行
