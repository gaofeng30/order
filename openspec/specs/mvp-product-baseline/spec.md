# mvp-product-baseline Specification

## Purpose
定义一期正式研发必须共同遵守、可追踪并可验收的唯一产品行为与外部依赖基线，统一一期范围、预约取餐、订单履约、身份与定价、权限责任和外部 Gate，避免后续 change 依据 P0 mock 或简化状态产生冲突契约。基线依据 2026-08-19 客户正式确认记录 `docs/product/online-ordering-system-prd-0818-review.md` 与 `docs/product/online-ordering-system-prd-0818.md`。
## Requirements
### Requirement: Product sources have one explicit authority order

一期产品基线 MUST 声明并执行以下优先级：真实适用合同与客户正式确认记录高于 PRD §1–§14，PRD §1–§14 高于 PRD §15、小程序源码和 PC 原型；mock 数据、内存状态及原型简化状态机不得作为生产契约。

当前生效的客户正式确认记录 MUST 为 `docs/product/online-ordering-system-prd-0818-review.md`；当前生效的 PRD §1–§14 MUST 为 `docs/product/online-ordering-system-prd-0818.md`。`docs/product/online-ordering-system-prd.md` 中与前两者冲突的条款 MUST 视为失效，不得作为实现或验收依据。

仓库内合同证据 MUST 只表述为“仓库范围内未发现已签署证据，现实签署状态未知”。合同适用性 MUST 作为商业与上线外部依赖记录，但不得阻塞本地技术 OpenSpec 的规划、校验或提交。

#### Scenario: Conflicting sources are reviewed

- **WHEN** 同一业务规则在客户正式确认记录、PRD §1–§14 与 §15/原型中出现不同表述
- **THEN** reviewer 按既定优先级保留唯一正式研发规则
- **AND** 较低优先级内容只能保留为原型说明，不得与正式规则并列生效

#### Scenario: Superseded baseline clause is cited

- **WHEN** 任一 change 引用 `online-ordering-system-prd.md` 中被客户评审记录推翻的条款作为实现或验收依据
- **THEN** 该引用被判定为无效依据
- **AND** reviewer 要求改用当前生效的评审记录与 0818 PRD

#### Scenario: Contract evidence is described

- **WHEN** PRD 或 OpenSpec 描述仓库中的合同签署状态
- **THEN** 文案只说明仓库范围内没有已签署证据且现实状态未知
- **AND** 不得断言现实中已经签署或尚未签署

#### Scenario: Mock behavior is inspected

- **WHEN** reviewer 发现前端使用独立内存订单、假支付、简化状态或撤销交互
- **THEN** 这些行为被标记为 P0 原型行为
- **AND** 正式研发只采用 PRD §1–§14 中冻结的生产规则

### Requirement: First-phase scope is closed and singular

一期 MUST 仅包含单门店单取餐点到店自提、仅预约取餐、员工与访客身份、全局单一折扣率的员工优惠、按取餐日期的商品售罄开关、微信支付、自动排产与制作、备好、扫码或手工核销、原路全额退款和基础后台。

一期 MUST 排除数量库存与库存预占、即时取餐、接单、会员等级、优惠券、积分、储值、部分退款、配送、多门店、POS、打印机、叫号屏和跨业务主入口，且不得为这些排除项预留并行生效的产品规则、UI 占位或兼容分支。

#### Scenario: Included scope is traced

- **WHEN** reviewer 检查一期范围、业务规则、页面与验收矩阵
- **THEN** 每个包含项都有唯一规则和至少一个验收落点
- **AND** 全局折扣率不得被归入会员等级或优惠券能力

#### Scenario: Excluded capability is proposed

- **WHEN** PRD 实施或后续 change 尝试把任一排除项作为一期行为
- **THEN** owned-path 和范围检查失败
- **AND** 该能力必须独立立项并获得新的正式范围授权

#### Scenario: Excluded capability is proposed as a UI placeholder

- **WHEN** 任一 change 以“置灰提示”“即将上线”或预留入口的形式引入排除项
- **THEN** 范围检查失败
- **AND** 一期不接受任何排除项的 UI 占位例外

### Requirement: Product availability uses a per-service-date sellout switch

一期 MUST NOT 实现数量库存、库存预占、自动扣减、可售量或超卖控制。商品可售性 MUST 只由两个独立维度决定：上下架（长期，仅 PC 后台）与售罄（按取餐日期，两端均可切换）。

售罄标记 MUST 按取餐日期生效且只影响该营业日期的下单，MUST NOT 影响其他可预约日期；次日 MUST 自然回到可售，不需要人工恢复。售罄 MUST 可自由开关。

售罄商品在用户端 MUST 可见、带蒙层且不可加入购物车；下架商品在用户端 MUST 不展示。

一期 MUST NOT 提供任何自动的产能或超卖保护，该风险 MUST 由商户手工标记售罄承担。

#### Scenario: Sellout is marked during service

- **WHEN** 商户在营业日 D 把某商品标记为售罄
- **THEN** 该商品在 D 当日不可下单
- **AND** 该商品在 D+1 的预约不受影响

#### Scenario: Next service date begins

- **WHEN** 进入下一个营业日期
- **THEN** 前一日的售罄标记不再生效
- **AND** 商户无需手工恢复即可售出

#### Scenario: Quantity inventory is proposed

- **WHEN** 任一 change 引入份数、剩余量、预占或自动扣减
- **THEN** 范围检查失败
- **AND** 该能力必须独立立项并获得新的范围授权

### Requirement: Orders exist only after confirmed WeChat payment

服务端 MUST 只在确认微信支付成功后创建订单并分配取餐号。调起微信支付前创建的预支付记录 MUST NOT 被视为订单：它不得进入任何订单列表、不得占用取餐号、不得对用户可见。

提交订单前 MUST 重新校验身份、商品上下架与当日售罄、服务端价格、取餐日期，以及目标取餐时间所属餐段是否仍在截单前；任一校验不通过 MUST 拦截支付。目标餐段已截单时 MUST 提示已截单并把用户送回取餐时间选择，且 MUST 保留购物车内容。

前端支付成功回调 MUST NOT 作为支付成功事实。支付回调与订单创建 MUST 幂等：同一 `out_trade_no` 的重复通知 MUST NOT 重复建单或重复占号。

支付结果未确认期间，用户端 MUST 只显示临时加载提示，MUST NOT 生成订单，也 MUST NOT 展示异常状态。

#### Scenario: Payment succeeds

- **WHEN** 服务端确认微信支付成功
- **THEN** 系统创建订单并分配该取餐日期的取餐号
- **AND** 订单固化商品、价格、折扣率、身份、取餐日期与时间和取餐点快照

#### Scenario: Payment is cancelled or fails

- **WHEN** 用户取消支付或支付失败
- **THEN** 系统不创建订单
- **AND** 预支付记录作废且不占用取餐号

#### Scenario: Cutoff passes while the user is at checkout

- **WHEN** 用户提交订单时目标取餐时间所属餐段已过截单时刻
- **THEN** 服务端拦截支付并提示该餐段已截单
- **AND** 用户被送回取餐时间选择且购物车内容保留

#### Scenario: Payment notification is repeated

- **WHEN** 同一 `out_trade_no` 的支付成功通知被重复投递
- **THEN** 服务端返回第一次结果
- **AND** 不重复创建订单、不重复分配取餐号

### Requirement: Unmatched payments are reconciled into orders or manual handling

服务端 MUST 提供定时对账任务，扫描已发起支付但超过约定时长仍未生成订单的预支付记录，并调用微信支付结果查询接口核对真实支付结果。

查得未支付 MUST 作废该预支付记录。查得已支付 MUST 幂等补建订单并分配取餐号。补建失败 MUST 把该条目转入后台待处理列表交由主账号处理，MUST NOT 静默丢弃，也 MUST NOT 引入异常订单状态。

该链路 MUST 对用户端不可见；补建成功后订单 MUST 出现在用户订单列表中。

#### Scenario: Payment callback is lost

- **WHEN** 用户已支付成功但服务端未收到回调、订单未生成
- **THEN** 对账任务查得已支付并补建订单
- **AND** 用户在订单列表中看到该单

#### Scenario: Reconciliation cannot rebuild the order

- **WHEN** 对账查得已支付但补建订单失败
- **THEN** 该条目进入后台待处理列表
- **AND** 主账号可对其发起退款或人工建单

#### Scenario: Reconciliation runs repeatedly

- **WHEN** 对账任务对同一预支付记录重复执行
- **THEN** 已补建的订单不被重复创建
- **AND** 取餐号不被重复分配

### Requirement: Orders use one six-state production state machine

生产订单状态 MUST 且只能为 `已预约`、`制作中`、`待取餐`、`已完成`、`退款中`、`已退款`。一期 MUST NOT 存在 `待支付`、`已支付待接单`、`已取消` 或任何异常状态。

主链路 MUST 为：微信确认支付成功后进入 `已预约`，取餐时间前 30 分钟由服务端自动进入 `制作中`，商户标记备好进入 `待取餐`，核销成功进入 `已完成`。支付成功时若距取餐时间已不足 30 分钟，订单 MUST 在创建时直接进入 `制作中`。

`已预约 → 制作中` MUST 由服务端定时任务驱动。**客户端 MUST NOT 提供该转换**：商户端可执行的转换只有 `制作中 → 待取餐` 与 `待取餐 → 已完成`。

一期 MUST NOT 提供接单动作，也 MUST NOT 提供商户手动提前开做。排产定时任务 MUST 幂等并具备重试与补偿，任务漏跑 MUST NOT 导致订单卡在 `已预约`。

一期 MUST NOT 设置待取超时状态。备好后订单 MUST 保持 `待取餐` 直至核销完成或发起退款；营业日结束后仍未核销的订单 MUST 通过查询口径可筛选，且该口径 MUST NOT 是订单状态。

每次状态转换 MUST 由服务端校验前置状态、资源权限和幂等键并记录审计。相同幂等键的重复请求 MUST 返回第一次最终结果。生产 MUST NOT 提供撤销或回退已完成转换的入口，**包括 Toast 上的回退动作与任何回退契约方法**。

#### Scenario: Merchant fulfills a paid order

- **WHEN** 订单在取餐时间前 30 分钟自动进入 `制作中` 且商户标记备好
- **THEN** 订单进入 `待取餐`
- **AND** 核销成功后进入 `已完成`

#### Scenario: Payment succeeds inside the 30-minute window

- **WHEN** 支付成功时距取餐时间已不足 30 分钟
- **THEN** 订单创建即为 `制作中`
- **AND** 用户端不提供自助取消入口

#### Scenario: Client attempts to start production early

- **WHEN** 任一客户端尝试把 `已预约` 订单推进到 `制作中`
- **THEN** 请求被拒绝且订单状态不变
- **AND** 客户端的可推进转换表中不含 `已预约`

#### Scenario: A transition is repeated

- **WHEN** 同一幂等键的状态推进请求被重复提交
- **THEN** 服务端返回第一次最终结果
- **AND** 不重复产生支付、退款、营收或核销副作用

#### Scenario: An operator attempts undo

- **WHEN** 任一角色尝试撤销或回退已完成的状态转换
- **THEN** 服务端拒绝该请求
- **AND** 客户端不提供任何生产撤销入口或回退契约方法

#### Scenario: Order is never collected

- **WHEN** 营业日结束时订单仍处于 `待取餐`
- **THEN** 订单保持该状态且不自动流转
- **AND** 该订单可通过未取餐查询口径筛出，主账号可对其退款或事后核销

### Requirement: Cancellation and refund rules are deterministic

支付成功前 MUST 不存在订单，因此一期 MUST 不提供“取消未支付订单”这一行为；用户取消支付或支付失败 MUST 只作废预支付记录。

处于 `已预约` 且距取餐时间大于 30 分钟的订单 MUST 允许用户自助取消并立即发起一次原路全额退款。订单进入 `制作中` 后，用户自助取消 MUST 关闭，只能由具有退款权限的商户人员处理。

主账号 MUST 有权对任一已生成订单发起原路全额退款；一期 MUST 不支持部分退款。发起退款后订单进入 `退款中`，只有微信确认退款成功后才进入 `已退款`。退款结果无法确定或退款失败时，订单 MUST 保持 `退款中` 并在后台标记为待处理，MUST NOT 进入任何异常状态。

一期没有数量库存，退款 MUST NOT 触发任何库存返还或可售量调整。

#### Scenario: User cancels before payment completes

- **WHEN** 用户在微信支付页取消支付或支付失败
- **THEN** 系统不生成订单，也不占用取餐号
- **AND** 对应预支付记录作废且用户订单列表无该笔记录

#### Scenario: User cancels more than 30 minutes before pickup

- **WHEN** 订单处于 `已预约` 且距取餐时间大于 30 分钟
- **THEN** 系统接受用户取消并发起一次原路全额退款
- **AND** 订单进入 `退款中`，微信确认退款成功后为 `已退款`

#### Scenario: User cancels within 30 minutes of pickup

- **WHEN** 订单已进入 `制作中`
- **THEN** 用户端不得提供自助取消入口
- **AND** 订单只能交由主账号处理

#### Scenario: Refund result cannot be confirmed

- **WHEN** 退款已受理但微信未返回可确认的成功结果
- **THEN** 订单保持 `退款中` 并在后台标记为待处理
- **AND** 系统不把订单置为 `已退款`，也不引入异常状态

#### Scenario: Partial refund is requested

- **WHEN** 任一调用方请求小于实付金额的退款
- **THEN** 一期服务端拒绝该请求
- **AND** 不创建部分退款记录

### Requirement: Employee identity is decided by an active phone list

用户浏览 MUST 不要求手机号，小程序启动时 MUST NOT 强制手机号授权。首次提交订单前 MUST 完成微信会话与手机号授权；服务端标准化手机号后，只有命中启用员工折扣白名单的用户才是员工，未命中或记录停用者统一为访客，客户端不得自行选择身份。

用户 MAY 在个人中心手工填写附加手机号用于员工匹配。手工填写的手机号 MUST 只有在“手机号 + 姓名”同时命中同一条启用白名单记录时才授予员工身份；姓名 MUST 在去空格与全半角归一后精确匹配。微信授权取得的主手机号 MUST 命中即生效，其姓名只作记录。

员工折扣白名单与商户账号名单 MUST 分开维护，互不影响。

订单创建 MUST 固化用户身份、标准化手机号对应的内部用户引用、折扣率快照和白名单版本快照；面向页面、日志和追踪台账 MUST 不保存或展示未脱敏手机号及名单个人数据。

#### Scenario: Visitor browses without identification

- **WHEN** 未登录用户浏览门店、分类和商品
- **THEN** 系统允许浏览
- **AND** 启动与浏览阶段都不索取手机号授权

#### Scenario: Active employee is identified by the authorized phone

- **WHEN** 用户完成微信手机号授权且该手机号命中启用白名单
- **THEN** 服务端把该笔报价和订单识别为员工
- **AND** 订单固化命中的白名单版本快照与折扣率快照

#### Scenario: Extra phone matches only the phone number

- **WHEN** 用户手工填写的附加手机号命中白名单但姓名不一致
- **THEN** 系统不授予员工身份
- **AND** 该用户按访客原价结算

#### Scenario: Extra phone matches both factors

- **WHEN** 用户手工填写的附加手机号与姓名同时命中同一条启用白名单记录
- **THEN** 系统授予员工身份
- **AND** 后续报价按员工折扣计算

#### Scenario: Phone is absent or list entry is disabled

- **WHEN** 用户未完成手机号授权，或手机号未命中启用白名单
- **THEN** 未授权手机号时禁止提交订单
- **AND** 未命中或白名单已停用时按访客身份结算

### Requirement: Employee pricing uses one global discount rate applied per product

商品 MUST 使用整数分保存原价，且 MUST NOT 保存逐商品员工价。员工优惠 MUST 由一个全局单一折扣率承担，对所有命中白名单的用户和所有商品统一生效；一期 MUST NOT 引入会员等级、优惠券、积分、叠加算价或“不参与折扣”的商品开关。

算价链 MUST 固定为：原价小计 → 员工折扣 → 应付。折扣 MUST 逐商品计算：先用单价乘以折扣率并四舍五入到分，再乘数量求和，使菜单与详情页展示的员工价逐项等于结算明细中的成交价。

折扣率修改 MUST 只影响新报价，MUST NOT 回算历史订单。访客 MUST 按原价结算。未完成手机号绑定前，用户端 MUST 只展示原价且不展示划线价。

#### Scenario: Employee is quoted

- **WHEN** 已识别员工购买任意商品
- **THEN** 服务端按全局折扣率逐商品计算成交价
- **AND** 订单明细同时固化原价、折扣率和成交价

#### Scenario: Displayed employee price equals the charged price

- **WHEN** 员工把若干商品加入购物车并进入结算
- **THEN** 结算明细中每一项的成交价等于菜单与详情页展示的员工价
- **AND** 逐项成交价之和等于应付金额

#### Scenario: Visitor is quoted

- **WHEN** 访客购买任意商品
- **THEN** 服务端使用原价
- **AND** 客户端传入价格不得覆盖服务端报价

#### Scenario: Identity is not yet known

- **WHEN** 用户尚未完成手机号绑定
- **THEN** 菜单与详情页只展示原价
- **AND** 不展示划线价，也不先按员工价展示再在结算时打回原价

### Requirement: Every first-phase order uses one discrete pickup time

一期 MUST 只提供预约取餐，MUST NOT 提供即时取餐或“尽快”模式。可预约营业日期 MUST 只有今天与明天。门店时区 MUST 固定为 `Asia/Shanghai`。

餐段 MUST 只有午餐与晚餐，每个餐段 MUST 有一个固定截单时刻，该餐段内全部取餐时间共用该截单时刻，MUST NOT 随取餐时间滚动。

取餐时间 MUST 为餐段范围内的离散时间点，粒度 MUST 可由商户配置。取餐时间 MUST 定义为约定时刻而非必须到场的窗口：商品备好后由订阅消息通知，用户凭通知取餐。

用户 MUST 可全天浏览，但只有目标取餐时间所属餐段尚未截单且商品当日可售时才能提交订单。一期 MUST 为单门店单取餐点，MUST NOT 提供多点选择或分单路由。

#### Scenario: User chooses an available pickup time

- **WHEN** 用户在今天或明天选择一个未截单餐段内的取餐时间点
- **THEN** 系统接受该取餐时间
- **AND** 菜单按该取餐时间所属餐段过滤可售商品

#### Scenario: Meal period is past its cutoff

- **WHEN** 用户查看已过截单时刻的餐段
- **THEN** 该餐段的全部取餐时间不可选并标注截止时刻
- **AND** 当日两个餐段均已截单时，该日期整体不可选

#### Scenario: Immediate pickup is requested

- **WHEN** 任一调用方请求即时取餐或不带取餐时间的订单
- **THEN** 服务端拒绝该请求
- **AND** 一期不提供即时单路径

#### Scenario: Production values are not yet supplied

- **WHEN** 真实截单时刻、取餐时间范围与粒度尚未配置
- **THEN** 这些值作为 UAT 前配置记录
- **AND** 不得改变单取餐点、午晚两餐段、仅预约取餐与可预约今天明天的模型

### Requirement: Merchant permissions use two server-enforced roles

商户后台 MUST 且只能使用主账号与子账号两个角色，且 MUST 由服务端执行资源权限；客户端菜单隐藏 MUST NOT 代替鉴权。

主账号 MUST 可配置多个。主账号 MUST 拥有小程序端全部能力，以及 PC 后台的商品与分类配置、上下架、价格、全局折扣率、员工折扣白名单、商户账号名单、退款、财务与对账、营业设置、开屏图层和看板。

子账号 MUST 只能使用小程序端，能力 MUST 限于查看订单、标记备好、扫码或手工核销、切换商品可售与售罄；MUST NOT 进入 PC 后台，MUST NOT 改价、上下架、退款、修改配置或查看财务。

商户身份来源 MUST 为 PC 后台维护的商户账号名单。小程序端 MUST 按 openid 是否已绑定商户手机号决定启动落地页；PC 后台 MUST 使用微信扫码登录并只允许主账号通过，MUST NOT 建设独立密码体系。

普通用户 MUST NOT 进入商户端。开发人员 MUST NOT 默认持有常驻业务角色，生产排障只能使用客户明确授权、限时且有审计的访问。

#### Scenario: Sub-account fulfills an order

- **WHEN** 子账号在小程序端查看订单并标记备好
- **THEN** 操作成功且订单进入 `待取餐`
- **AND** 同一子账号发起退款、改价或上下架的请求被服务端拒绝

#### Scenario: Sub-account attempts PC login

- **WHEN** 子账号手机号用于 PC 后台微信扫码登录
- **THEN** 登录被拒绝
- **AND** 不产生任何业务副作用

#### Scenario: Unassigned user enters a merchant route

- **WHEN** 不在商户账号名单中的用户请求商户端路由或接口
- **THEN** 服务端拒绝该请求
- **AND** 该用户的启动落地页仍为用户端首页

#### Scenario: Merchant binds on first use

- **WHEN** 商户首次在个人中心触发商户登录并完成手机号授权
- **THEN** 命中商户账号名单时绑定该 openid
- **AND** 之后启动直接进入身份选择页

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

追踪矩阵引用的状态集合 MUST 为六态，角色集合 MUST 为主账号与子账号两角色；矩阵 MUST NOT 引用任何已被客户评审删除的概念作为待覆盖维度。

#### Scenario: Traceability matrix is checked

- **WHEN** reviewer 逐条对照本 spec 与 PRD 追踪矩阵
- **THEN** 每条 requirement 至少有一个 PRD 目标位置和验收方法
- **AND** 页面、六态、两角色及 12 个外部 Gate 均无孤立项

#### Scenario: Matrix cites a retired dimension

- **WHEN** 追踪矩阵把九态、四角色、数量库存、软预占、固定取餐时段或优惠券列为待覆盖维度
- **THEN** PRD 实施验收失败
- **AND** 该维度必须先按当前生效 spec 重新表述

#### Scenario: Behavioral ambiguity remains

- **WHEN** PRD 中仍存在“待确认”“可选方案”“A/B”“尽快取餐”或与本 spec 冲突的正式研发规则
- **THEN** PRD 实施验收失败
- **AND** change 不得产生候选 SHA

### Requirement: Pickup identifiers and notifications follow the frozen contract

取餐号 MUST 为四位数字并按取餐日期从 `0001` 累计；同一订单重复处理 MUST 返回原号且不得重复占号。手工取餐号核销 MUST 只匹配当前营业日期的 `待取餐` 订单（I14）。二维码 token MUST 在订单进入 `待取餐` 时生成、服务端不透明绑定订单、不设时间过期并在核销后立即失效；扫码与手工核销 MUST 共用同一幂等规则。

一期 MUST 只有待取餐提醒与退款结果两类一次性订阅消息，且均由用户主动订阅（I15）。支付成功页 MUST 请求待取餐提醒，取消确认弹窗 MUST 请求退款结果；拒绝订阅时 MUST 保留首页进行中订单提示条、订单状态与取餐码页补订阅入口。下单成功、支付成功、订单完成、临时停餐或商品变化 MUST 不发送其他订阅消息。

#### Scenario: Cross-day pickup numbers collide

- **WHEN** 当前营业日期与历史营业日期存在相同四位取餐号
- **THEN** 手工核销只匹配当前营业日期的 `待取餐` 订单
- **AND** 历史未取订单不得被误核销

#### Scenario: Redemption is repeated

- **WHEN** 同一二维码或取餐号被重复核销
- **THEN** 服务端返回第一次核销结果
- **AND** 不重复完成订单或统计营收

#### Scenario: User declines a subscription

- **WHEN** 用户拒绝支付成功页的待取餐提醒订阅
- **THEN** 首页进行中订单提示条和订单状态仍提供取餐信息
- **AND** 取餐码页保留再次请求待取餐提醒的入口

### Requirement: Production facts and statistics come from server-confirmed data

身份、商品、价格、销售状态、订单、支付、退款和核销 MUST 以后端为唯一事实源；订单 MUST 固化商品、价格、身份、折扣率、取餐日期/时间和取餐点快照，配置变化不得回算历史订单。mock、客户端内存状态、前端支付结果和 §15 简化状态 MUST 只用于 P0 演示，不得覆盖生产事实（I16）。

统计 MUST 只来自服务端确认的订单、支付、退款和核销数据；`退款中`、`已退款` MUST 按明确口径单独统计且不混入有效营收。营业日结束后仍为 `待取餐` 的订单 MUST 以“未取餐”查询口径单独可查，不得计入已完成订单数或创建新状态。

#### Scenario: Client state conflicts with the server

- **WHEN** mock、客户端内存订单、前端支付动画或本地状态与服务端事实不同
- **THEN** 页面与后续处理采用服务端结果
- **AND** 客户端不得本地创建生产订单、推进状态或确认支付退款成功

#### Scenario: Dashboard calculates effective revenue

- **WHEN** 基础看板统计订单与营收
- **THEN** 只使用服务端确认的数据并单列退款中、已退款与未取餐口径
- **AND** 未取餐不计为已完成，退款中或已退款不混入有效营收
