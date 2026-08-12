## ADDED Requirements

### Requirement: Product sources have one explicit authority order

一期产品基线 MUST 声明并执行以下优先级：真实适用合同与客户正式确认记录高于 PRD §1–§14，PRD §1–§14 高于 PRD §15、小程序源码和 PC 原型；mock 数据、内存状态及原型简化状态机不得作为生产契约。

仓库内合同证据 MUST 只表述为“仓库范围内未发现已签署证据，现实签署状态未知”。合同适用性 MUST 作为商业与上线外部依赖记录，但不得阻塞本地技术 OpenSpec 的规划、校验或提交。

#### Scenario: Conflicting sources are reviewed

- **WHEN** 同一业务规则在合同/客户正式确认、PRD §1–§14 与 §15/原型中出现不同表述
- **THEN** reviewer 按既定优先级保留唯一正式研发规则
- **AND** 较低优先级内容只能保留为原型说明，不得与正式规则并列生效

#### Scenario: Contract evidence is described

- **WHEN** PRD 或 OpenSpec 描述仓库中的合同签署状态
- **THEN** 文案只说明仓库范围内没有已签署证据且现实状态未知
- **AND** 不得断言现实中已经签署或尚未签署

#### Scenario: Mock behavior is inspected

- **WHEN** reviewer 发现前端使用独立内存订单、假支付、简化状态或撤销交互
- **THEN** 这些行为被标记为 P0 原型行为
- **AND** 正式研发只采用 PRD §1–§14 中冻结的生产规则

### Requirement: First-phase scope is closed and singular

一期 MUST 仅包含单门店到店预约自提、员工与访客、逐商品固定员工价、按日按餐段商品库存、微信支付、商户接单/制作/待取餐/核销、原路全额退款和基础后台。

一期 MUST 排除会员等级、优惠券、积分、储值、部分退款、配送、多门店、POS、打印机和叫号屏，且不得为这些排除项预留并行生效的产品规则。

#### Scenario: Included scope is traced

- **WHEN** reviewer 检查一期范围、业务规则、页面与验收矩阵
- **THEN** 每个包含项都有唯一规则和至少一个验收落点
- **AND** 简单员工价不得被归入会员等级或优惠券能力

#### Scenario: Excluded capability is proposed

- **WHEN** PRD 实施或后续 change 尝试把任一排除项作为一期行为
- **THEN** owned-path 和范围检查失败
- **AND** 该能力必须独立立项并获得新的正式范围授权

### Requirement: Inventory is keyed by service date, meal period, and product

正式库存唯一键 MUST 为 `营业日期 × 餐段 × 商品`。午餐与晚餐 MUST 使用独立库存池；同一餐段内的全部取餐时段 MUST 共享该餐段商品库存，不得增加时段库存维度。

库存剩余量、软预占量和已售量 MUST 以后端为唯一事实源；客户端展示或 mock 数值不得决定是否可售。

#### Scenario: Two meal periods sell the same product

- **WHEN** 同一商品在同一营业日期同时配置午餐和晚餐库存
- **THEN** 午餐下单只消耗午餐库存池
- **AND** 晚餐库存不受该笔午餐订单影响

#### Scenario: Two slots share one meal period

- **WHEN** 两个固定取餐时段都归属午餐
- **THEN** 两个时段的订单共同竞争同一 `营业日期 × 午餐 × 商品` 库存
- **AND** 系统不得为两个时段分别维护商品库存

#### Scenario: Client inventory conflicts with the server

- **WHEN** 客户端展示库存大于服务端可售库存
- **THEN** 服务端拒绝超出剩余库存的预占
- **AND** 客户端刷新后端事实，不得本地补单

### Requirement: Order submission uses a bounded atomic soft hold

提交订单 MUST 原子创建 15 分钟库存软预占并生成 `待支付`订单。软预占不得计为已售库存或有效预约；服务端确认微信支付成功后 MUST 将软预占原子转为实扣和有效预约，未支付超时 MUST 关闭订单并释放软预占。

迟到支付成功 MUST 先幂等确认支付事实并原子重占相同营业日期、餐段和商品库存；重占成功进入`已支付待接单`，重占失败 MUST 自动发起原路全额退款并进入`异常`。该异常自动退款成功后 MUST 转为`已退款`，退款未成功前保持`异常`并可被人工追踪。支付通知、超时释放、重占和退款触发 MUST 可重入且不重复扣减、释放或退款。

#### Scenario: Payment succeeds within the hold window

- **WHEN** 有效支付成功通知在 15 分钟软预占内到达
- **THEN** 系统只执行一次实扣并把订单转为`已支付待接单`
- **AND** 该订单成为有效预约

#### Scenario: Payment does not complete in time

- **WHEN** 15 分钟内没有可确认的支付成功事实
- **THEN** 系统把订单转为`已取消`
- **AND** 完整释放该订单的软预占

#### Scenario: Late payment can reacquire stock

- **WHEN** 已超时订单收到可验证的迟到支付成功通知且库存仍可原子重占
- **THEN** 系统重占并实扣库存
- **AND** 订单转为`已支付待接单`

#### Scenario: Late payment cannot reacquire stock

- **WHEN** 已超时订单收到可验证的迟到支付成功通知但库存无法完整重占
- **THEN** 系统不得形成有效预约
- **AND** 只发起一次原路全额退款并把订单置为`异常`
- **AND** 退款成功后订单转为`已退款`，否则保持可追踪的`异常`

### Requirement: Orders use one nine-state production state machine

生产订单状态 MUST 且仅能为：`待支付`、`已支付待接单`、`制作中`、`待取餐`、`已完成`、`已取消`、`退款中`、`已退款`、`异常`。

标准履约 MUST 为`待支付 → 已支付待接单 → 制作中 → 待取餐 → 已完成`。取消、退款和无法自动确定的钱单/库存结果 MUST 分别进入对应终态或异常态。所有转换 MUST 由服务端校验前置状态、操作者权限和幂等键并留下审计记录；生产环境 MUST 禁止撤销或回退已完成的状态转换。

#### Scenario: Merchant fulfills a paid order

- **WHEN** 支付已确认、后厨依次接单并标记备好、核销员完成核销
- **THEN** 订单严格依次经过`已支付待接单`、`制作中`、`待取餐`和`已完成`
- **AND** 任一步不得跳过前置状态

#### Scenario: A transition is repeated

- **WHEN** 相同幂等键重复提交同一状态转换
- **THEN** 系统返回第一次转换的最终结果
- **AND** 不得重复产生库存、营收、退款或核销副作用

#### Scenario: An operator attempts undo

- **WHEN** 任一客户端请求把生产订单回退到旧状态
- **THEN** 服务端拒绝该请求
- **AND** 原订单状态和审计记录保持不变

### Requirement: Cancellation and refund rules are deterministic

`待支付`订单 MUST 允许用户直接取消。`已支付待接单`且尚未到该取餐时段截单时间的订单 MUST 允许用户自助取消并立即发起原路全额退款；商户接单或到达截单时间任一先发生后，用户自助取消 MUST 关闭，只能由商户处理。

店管角色有权按业务处理结果发起原路全额退款；一期 MUST 不支持部分退款。除迟到支付重占失败的异常自动退款外，用户或店管发起退款后订单进入`退款中`，只有微信确认退款成功后才进入`已退款`。商户接单前退款成功 MUST 返还对应实扣库存；接单后商户处理的退款 MUST 不自动恢复可售库存。

#### Scenario: User cancels an unpaid order

- **WHEN** 用户取消`待支付`订单
- **THEN** 订单转为`已取消`
- **AND** 软预占被完整释放且不发起退款

#### Scenario: User cancels before acceptance and cutoff

- **WHEN** 订单处于`已支付待接单`且商户未接单、时段未截单
- **THEN** 系统接受用户取消并发起一次原路全额退款
- **AND** 退款成功后订单为`已退款`并返还实扣库存

#### Scenario: User cancels after acceptance or cutoff

- **WHEN** 商户已经接单或预约时段已经截单
- **THEN** 用户端不得提供自助取消
- **AND** 订单只能交由具有退款权限的商户人员处理

#### Scenario: Partial refund is requested

- **WHEN** 任一调用方请求小于实付金额的退款
- **THEN** 一期服务端拒绝该请求
- **AND** 不创建部分退款记录

### Requirement: Employee identity is decided by an active phone list

用户浏览 MUST 不要求手机号。首次结算 MUST 完成微信会话、手机号授权并提供姓名；服务端标准化手机号后，只有命中启用员工名单的用户才是员工，未命中者统一为访客，客户端不得自行选择身份。

订单创建 MUST 固化用户身份、标准化手机号对应的内部用户引用和员工名单版本快照；面向页面、日志和追踪台账 MUST 不保存或展示未脱敏手机号及名单个人数据。

#### Scenario: Visitor browses without identification

- **WHEN** 未登录用户浏览门店、分类和商品
- **THEN** 系统允许浏览
- **AND** 不提前强制手机号授权

#### Scenario: Active employee checks out

- **WHEN** 用户完成微信会话和手机号授权、提供姓名且手机号命中启用名单
- **THEN** 服务端把该笔报价和订单识别为员工
- **AND** 订单固化命中的名单版本快照

#### Scenario: Phone is absent or list entry is disabled

- **WHEN** 首次结算没有有效手机号与姓名，或手机号未命中启用员工名单
- **THEN** 缺少手机号或姓名时禁止结算
- **AND** 未命中或名单已停用时按访客身份结算

### Requirement: Employee price is an optional fixed per-product amount

商品 MUST 使用整数分保存正常价，并可选保存一个整数分 `employee_price`。服务端 MUST 只对已识别员工使用已配置的逐商品员工价；未配置员工价的商品和所有访客 MUST 使用正常价。

报价与订单明细 MUST 固化正常价、适用员工价、实际成交价、身份和价格版本；一期不得引入百分比折扣、会员等级、优惠券、积分或叠加算价。

#### Scenario: Employee-priced product is quoted

- **WHEN** 已识别员工购买配置了 `employee_price` 的商品
- **THEN** 服务端使用该固定员工价计算应付金额
- **AND** 订单明细同时固化正常价、员工价和成交价

#### Scenario: Visitor or unpriced product is quoted

- **WHEN** 访客购买任意商品，或员工购买未配置员工价的商品
- **THEN** 服务端使用正常价
- **AND** 客户端传入价格不得覆盖服务端报价

### Requirement: Every first-phase order uses one fixed pickup slot

一期 MUST 使用单门店、单取餐点和 `Asia/Shanghai` 门店时区。订单 MUST 选择今天或明天的午餐/晚餐固定取餐时段，不得提供“尽快取餐”；每个时段 MUST 归属一个餐段并拥有独立截单时间。

用户 MUST 可全天浏览；只有目标时段尚未截单且商品可售时才允许下单。订单 MUST 固化营业日期、餐段、时段、截单时间和取餐点快照。实际商品、库存值、时段、截单、取餐点、员工名单和价格值 MUST 作为 UAT 前生产配置，不得改变上述模型。

#### Scenario: User chooses an available slot

- **WHEN** 用户选择今天或明天、尚未截单的午餐或晚餐固定时段
- **THEN** 系统按该营业日期、餐段和时段校验并创建订单
- **AND** 订单固化完整预约与取餐点快照

#### Scenario: User requests immediate or out-of-range pickup

- **WHEN** 用户请求“尽快取餐”、后天及以后、非午晚餐或已截单时段
- **THEN** 服务端拒绝创建订单
- **AND** 不产生库存软预占

#### Scenario: Production values are not yet supplied

- **WHEN** 本地技术 OpenSpec 正在规划但真实商品、库存值、时段、截单、取餐点、名单或价格尚未配置
- **THEN** OpenSpec 仍可按固定模型完成和校验
- **AND** 对应能力进入 UAT 前必须补齐生产配置

### Requirement: Merchant permissions use four server-enforced roles

一期后台 MUST 且仅能使用`店管`、`后厨`、`核销`和`财务只读`四个业务角色。店管管理商品、员工名单、员工价、营业预约设置、订单、全额退款和基础看板；后厨只查看履约必要信息并执行接单、制作和备好；核销角色只查看待取餐必要信息并执行扫码或手工核销；财务只读角色只能查看支付、退款、结算和导出。

角色和资源权限 MUST 由服务端执行。用户不得自行进入商户端或选择后台角色；开发人员不得默认获得常驻业务角色，生产排障访问必须由客户进行限时外部授权并保留审计。

#### Scenario: Kitchen operator fulfills an order

- **WHEN** 后厨角色处理`已支付待接单`或`制作中`订单
- **THEN** 服务端只允许接单、制作和备好相关读取与转换
- **AND** 拒绝商品定价、员工名单、退款和角色管理操作

#### Scenario: Finance viewer attempts a mutation

- **WHEN** 财务只读角色请求修改订单、商品、配置或退款
- **THEN** 服务端拒绝请求
- **AND** 不产生任何业务副作用

#### Scenario: Unassigned user enters a merchant route

- **WHEN** 没有后台角色的用户访问商户页面或接口
- **THEN** 服务端拒绝访问
- **AND** 客户端身份选择不得提升权限

### Requirement: External readiness follows one twelve-gate chain

外部依赖台账 MUST 且只能按以下 12 个 Gate 顺序追踪：

1. 主体一致性；
2. 小程序注册与认证；
3. 餐饮类目；
4. 备案；
5. 项目成员与体验成员；
6. 云资源、域名实名与 ICP；
7. HTTPS 与服务器域名；
8. 商户号与 AppID 绑定；
9. API 安全状态；
10. 交易结算管理；
11. 隐私与手机号；
12. 体验版真机、审核、客户发布确认，且该 Gate 内部严格按此三步推进。

每个 Gate MUST 只记录责任方、非敏感证据引用、状态、所阻塞的阶段和更新时间。尚未取得完成证据的 Gate 状态 MUST 统一为 `BLOCKED_EXTERNAL`，取得可核查完成证据后 MUST 变为 `READY`；不得使用行为 TODO 或空值代替状态。任何台账、PRD 或 OpenSpec MUST 不包含密钥、证书内容、账号标识、手机号、员工名单或其他个人数据。外部 Gate 不得阻塞本地技术 OpenSpec，只能阻塞其声明的联调、UAT、提审或发布阶段。

#### Scenario: Local OpenSpec is planned before external readiness

- **WHEN** 一个技术 change 的行为、模型和本地验收已确定但外部 Gate 尚未完成
- **THEN** change 可继续本地规划、严格校验和提交
- **AND** 未完成 Gate 明确记录为`BLOCKED_EXTERNAL`
- **AND** 不得把未完成 Gate 伪装为已验证或越过其对应外部阶段

#### Scenario: Gate ledger is reviewed for sensitive content

- **WHEN** reviewer 检查外部依赖台账、PRD 和 OpenSpec
- **THEN** 只看到状态和非敏感证据引用
- **AND** 不存在密钥、证书正文、账号标识或个人数据

#### Scenario: Release is requested

- **WHEN** 项目准备正式发布
- **THEN** 12 个 Gate 均有可核查的完成证据
- **AND** 最后一个 Gate 已依次完成体验版真机、审核和客户发布确认

### Requirement: The baseline is traceable and has no behavioral TODO

实施后的 PRD §1–§14 MUST 提供需求—页面—状态—角色—外部依赖追踪矩阵，覆盖本 spec 全部 requirements，并为每条规则指向明确页面、状态、责任角色、外部 Gate 与验收方法；不适用维度 MUST 显式写为“不适用”。

PRD 正式基线 MUST 不存在会改变产品行为、公共契约、数据结果、授权边界或验收方式的 TODO、待定项或 A/B 方案。尚未提供的真实经营值只能列为 UAT 前配置，不得形成第二套行为模型。

#### Scenario: Traceability matrix is checked

- **WHEN** reviewer 逐条对照本 spec 与 PRD 追踪矩阵
- **THEN** 每条 requirement 至少有一个 PRD 目标位置和验收方法
- **AND** 页面、九态、四角色及 12 个外部 Gate 均无孤立项

#### Scenario: Behavioral ambiguity remains

- **WHEN** PRD 中仍存在“待确认”“可选方案”“A/B”“尽快取餐”或与本 spec 冲突的正式研发规则
- **THEN** PRD 实施验收失败
- **AND** change 不得产生候选 SHA
