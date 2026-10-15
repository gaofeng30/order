## MODIFIED Requirements

### Requirement: Production topology has one deployment shape

一期生产架构 MUST 使用独立 CloudBase 生产环境中的 CloudRun Go API、Shared PostgreSQL 官方 HTTPS RPC、同地域私有 COS、同源 HTTP 网关与静态托管。正式域名 MUST 使用有效 TLS；PC 唯一入口 MUST 为 `/pc/`，API 为 `/api/v1`。不得增加旧 PC 路径兼容。

运行时 MUST 使用普通专用数据库身份和固定 RPC allowlist，不得使用管理 SQL、管理 API key 或通用云凭据执行业务。材料 MUST 由 CloudRun 受保护版本配置注入；COS MUST 通过绑定本环境桶及 images 前缀的受限 STS 身份访问。源码、镜像与日志不得携带秘密。运行日志 MUST 脱敏，关键业务操作 MUST 可查询审计；不得把特定收费监控产品作为默认前置。

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

UAT 与生产 COS MUST 为私有桶并禁止匿名读取和列举。Admin MUST 先向 API 申请有效期不超过 15 分钟的预签名 PUT，再直传 COS 并调用确认接口；客户端图片读取 MUST 使用有效期不超过 15 分钟的预签名 GET。对象 key MUST 由服务端生成，上传 MUST 校验允许的 MIME、大小和摘要。生产图片 MUST 使用唯一 key、禁止覆盖及持久确认，MUST NOT 依赖 COS 版本号或开启历史版本保留；图片误删恢复不属于原始 PRD 要求。此变更需同步 API/SQL 并验证后发布，不能将本地规格当作线上状态；既有 UAT 历史版本不在本次变更范围。

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

核心数据 MUST 有可执行的备份与恢复演练方案；MUST 核实当前 Shared PG 已有导出/备份路径，在隔离目标验证 schema、函数、ACL/RLS、序列及订单、支付、退款、核销和审计。MUST 记录来源时间、数据一致性、实际恢复窗口与耗时，不继承旧 CDB 固定保留期、恢复频率或 RPO/RTO 数值。CloudRun 实例 MUST 不保存业务事实，并能从镜像、保护配置及材料引用重建。

平台备份入口不可用 MUST NOT 被直接解释为所有导出恢复路线不可用；现有授权内的替代能力须按同一完整性要求实测。任何新增费用或覆盖生产的恢复操作 MUST 获得相应授权。图片历史版本与误删恢复 MUST NOT 成为发布前置。

#### Scenario: Recovery targets are documented before a drill
- **WHEN** 恢复演练尚未完成
- **THEN** 文档记录拟用路径及未验证项，不承诺固定恢复时限或云厂商 SLA
- **AND** 不将设计方案记为已验证恢复能力

#### Scenario: An isolated recovery drill runs
- **WHEN** 使用已有导出或备份工件恢复到隔离目标
- **THEN** 校验结构与核心业务记录、外部资金事实，并记录实际数据窗口与耗时
- **AND** 隔离目标不得接收生产流量，清理按授权执行

### Requirement: Logging, metrics, and alerts have fixed redaction and thresholds

运行日志 MUST 保留足以定位问题的脱敏错误与关联信息；关键业务操作 MUST 可查询、可审计、可人工处理。日志、证据与告警 MUST NOT 包含请求原文、个人信息、凭据、完整核销 token、二维码或完整外部交易号。

CLS、CAT、固定日志保留天数和完整阈值矩阵 MUST NOT 作为新增发布门。预算告警、用量和证书状态 MUST 按用户要求只读核实；已配置与未配置须区分，新增收费能力须另行确认。不承诺未经建立的自动告警或全天人工值守。

#### Scenario: Sensitive request fails
- **WHEN** 含个人信息或支付内容的请求失败并产生日志
- **THEN** 只记录必要的非敏感关联、错误类别与执行结果
- **AND** 原始请求、身份和凭据不进入日志或告警

#### Scenario: A background task needs manual action
- **WHEN** 自动补偿停止且仍有未解决事项
- **THEN** 持久状态保留查询、审计与人工处理依据
- **AND** 不以删除日志产品依赖为理由丢弃业务记录

### Requirement: Capacity gate and upgrade order are evidence-driven

容量验证 MUST 依据原始 PRD 的已知规模：5–10 分钟集中 100–300 单、每日 500–1000 单。MUST 保留交易原子性、重复/并发操作幂等和核心交易不被统计任务阻塞的检查；MUST 记录实际资源、到达模型、耗时、错误及业务不变量。未执行的规模 MUST 标记未实测，不得以配额替代容量证明。

3000 人、固定 200 并发、300 商品、固定延迟及错误率阈值 MUST NOT 作为未获产品确认的新发布门。新增资源前 MUST 先诊断已有资源是否满足已知规模，排除非必要查询或实现依赖；不得默认增配缓存、MQ、只读实例或付费扩容。

#### Scenario: Initial resources have not been load-tested
- **WHEN** 尚未取得当前规格的隔离验证证据
- **THEN** 记录未实测范围并执行已知规模的必要验证
- **AND** 不声称当前规格已满足全部容量要求

#### Scenario: Team proposes horizontal expansion
- **WHEN** 团队准备新增资源
- **THEN** 必须给出已有资源不满足已确认规模的实测证据
- **AND** 新增费用或权限须获得相应授权

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
