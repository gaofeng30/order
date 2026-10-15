## ADDED Requirements

### Requirement: 保留完整履约命令合同
系统 SHALL 经普通服务JWT共享PG原子RPC实现现有MARK_READY、REDEEM_TOKEN、REDEEM_CURRENT_DATE_CODE，并保留REDEEM_ORDER行为及Order Bearer、公共HTTP接口。

#### Scenario: 商户制作完成
- **GIVEN** 当前启用OWNER或SUBACCOUNT和合法PREPARING订单
- **WHEN** MARK_READY首次成功
- **THEN** Go生成安全随机token并Seal，订单READY、version加1、密文/hash/keyVersion/issuedAt与readyAt保持原合同
- **AND** audit成功receipt与订单同事务，旧key重放不重生成token或改变状态

#### Scenario: 扫码和当日取餐码定位
- **WHEN** 执行TOKEN或CODE
- **THEN** 优先依据token hash或本次Execute冻结的上海当前日期及四位号码定位live订单，缺失才按当前商户scope/action/key查原receipt目标
- **AND** 事务内复核locator，不能通过预读或跨午夜重算切换命令目标

#### Scenario: 已完成订单的命令差异
- **GIVEN** 合法COMPLETED订单且TOKEN/CODE live locator匹配
- **WHEN** 使用新key重复核销
- **THEN** 追加该命令receipt并Replay=true但不改变订单version；REDEEM_ORDER同条件新key仍按原TransitionNotAllowed拒绝

#### Scenario: 当前授权或proof改变
- **GIVEN** 旧receipt或Go已预读token proof
- **WHEN** 当前账号禁用/解绑/scope改变或锁内完整proof变化
- **THEN** 先核当前授权；未验证proof不得授权写入，合法有限重读外保持原错误且无副作用

### Requirement: READY通知意图与订单原子提交
系统 SHALL 在MARK_READY同一事务中保持现有READY consent选择、消费、不可变outbox及receipt合同；不得把通知意图视为微信送达。

#### Scenario: 无已接受授权
- **GIVEN** READY message满足原格式但没有该order/kind的ACCEPTED consent
- **WHEN** MARK_READY执行
- **THEN** 通知入队no-op，订单与命令receipt正常提交

#### Scenario: 消费最早已接受授权
- **GIVEN** 存在多个consent
- **WHEN** 为READY入队
- **THEN** 锁grant_sequence最小ACCEPTED项，核对recipient，沿order→consent→outbox顺序创建唯一不可变PENDING意图并消费该项
- **AND** 不改为最新授权、不跳过已消费项挑下一项

#### Scenario: 已有意图或中途失败
- **WHEN** 已有outbox与consent/recipient/template/message不符，已消费但无outbox，或outbox/consent/receipt步骤故障
- **THEN** 保持原fulfillment错误映射，订单/token/version、consent、outbox和receipt全部回滚；只有完全同一不可变意图才允许no-op

### Requirement: 订单读取保留真实事实与投影
系统 SHALL 实现现有orderquery五方法及其当前商户身份只读依赖，并由Go保留校验、解密、隐私和公共DTO。

#### Scenario: 列表过滤与分页
- **WHEN** 用户或商户按既有state/active/date/search/after/limit查询
- **THEN** 保留owner过滤、id倒序游标、limit+1、phone后缀LIKE转义与空数组；不得扩展筛选或推进订单状态

#### Scenario: 用户与商户详情
- **WHEN** 读取真实orders/items
- **THEN** 以同一SQL语句快照读取订单和items，完整校验金额/序号/数量/UTF8/时间历史，owner不符NotFound、phone脱敏；仅owner的READY详情通过Go解密和hash验证返回token，商户不返回token

#### Scenario: 商户身份数据异常
- **GIVEN** 原生Identity依赖的用户phone/extra身份/config/version事实损坏或缺失
- **WHEN** 商户查询
- **THEN** 保留原Unavailable或Forbidden区别，不以仅检查enabled商户绑定放宽原合同

#### Scenario: 命令后历史状态响应
- **GIVEN** 命令成功后订单已进一步推进
- **WHEN** GetMerchantAtState请求receipt对应READY或COMPLETED
- **THEN** 先校验真实当前record且仅允许具备相应历史时间的投影，保持原actions/timestamps/notification options，不伪造状态

### Requirement: 原子并发与恢复
系统 SHALL 保留当前scope/action/key请求摘要、状态和错误合同；传输不模拟跨HTTP事务。

#### Scenario: 并发与未知提交
- **WHEN** 16个同key或不同key竞争，或提交后响应丢失再以原key恢复
- **THEN** 按各命令原重放规则返回；订单只发生一次对应状态变更、READY最多一个意图/一次consent消费、各key成功receipt不重复，跨请求摘要冲突明确拒绝

### Requirement: 固定RPC权限与受控安装
系统 SHALL 仅扩展固定命名RPC和必要普通UID RLS/ACL，保留原44ledger与源码hash及严格readiness，运行端不持DDL/管理员能力。

#### Scenario: 新定义或权限漂移
- **WHEN** 新函数签名/body/security/settings、新表policy/ACL/sequence或v44事实偏离候选
- **THEN** actual metadata与Go权威对照使readiness失败，anon/其他UID不能访问业务事实或写入，禁止admin fallback

#### Scenario: 已知业务关系带有平台默认宽授权
- **GIVEN** 权威 v44 最终25业务表、独立迁移账本和16个 sequence 存在超出当前固定运行清单的默认授权
- **WHEN** root 通过受控管理版本安装本切片权限配置
- **THEN** 显式撤销这些已知资源对 PUBLIC/anon/authenticated 的表级、必要列级及 sequence 授权，再只授予当前10表/2sequence精确所需权限；管理owner、原44迁移及账本不变
- **AND** 权威资源名单一致性检查、真实普通 UID readiness、运行权限复读及管理读取全部通过；未知对象只阻断待核，不自动撤权

#### Scenario: 固定资源以外的运行权限漂移
- **GIVEN** public业务命名空间另有原44或unused relation
- **WHEN** anon或authenticated经直接/PUBLIC/继承获得固定allowlist之外表、列或sequence的有效权限
- **THEN** readiness拒绝；无运行权限的unused对象仍允许，不自动撤销未知授权，也不要求删除平台auth或pg_catalog必需资源

#### Scenario: 未审查的高权例程绕过
- **GIVEN** public业务命名空间出现固定已审签名之外的SECURITY DEFINER例程
- **WHEN** anon或authenticated经直接/PUBLIC/继承具有有效EXECUTE
- **THEN** readiness拒绝，不以受保护表RLS已开启代替高权入口隔离；撤销运行EXECUTE后允许该unused定义保留，平台auth/pg_catalog不在本扫描范围

### Requirement: 分层证据与后续边界
系统 SHALL 经真实v44原生差分、真实普通JWT UAT命令与读模型、候选及独验分别交付，不以stub或平台HTTP200代替业务通过。

#### Scenario: 当前切片通过
- **GIVEN** 本change真实履约/读取/意图证据完成
- **WHEN** 报告交付
- **THEN** 仅报告本切片，完整runtime、真实微信发送、存储最小权限、UI3和95Case/25页面保持未完成；首切片依赖未集成时不得集成本change
