## MODIFIED Requirements

### Requirement: Product sources have one explicit authority order

仓库 MUST 以 `docs/product/online-ordering-system-prd-0818.md` §1–§14 作为唯一有效产品 PRD；该文件已经吸收 `docs/product/online-ordering-system-prd-0818-review.md` 中 2026-08-19 的客户确认与逐轮澄清，review 文件 MUST 只作为裁决证据而不形成第二套产品正文。`docs/product/online-ordering-system-prd.md` MUST 只保留薄指针、废止说明和 0818 PRD §13.2 所需的外部 Gate 重定向，不得保留可被解释为仍生效的旧业务规则。

真实适用合同与客户正式确认记录 MUST 高于唯一 PRD；唯一 PRD §1–§14 MUST 高于 §15 前端现状与改造说明；mock、客户端内存态、前端支付结果及原型简化状态 MUST 不得作为生产契约（I16）。仓库内合同证据 MUST 只表述为“仓库范围内未发现已签署证据，现实签署状态未知”。

#### Scenario: A reader enters through the retired PRD

- **WHEN** reviewer 或后续 writer 打开 `docs/product/online-ordering-system-prd.md`
- **THEN** 文档明确声明自身已废止并指向 0818 PRD §1–§14
- **AND** reader 不会看到即时单、数量库存、软预占、九态、四角色、会员券或逐商品员工价仍是有效一期要求

#### Scenario: Product sources conflict

- **WHEN** 客户正式确认、0818 PRD §1–§14、§15 或原型对同一规则表述不同
- **THEN** reviewer 按“真实适用合同/客户正式确认 → 0818 PRD §1–§14 → §15 → 原型”的顺序保留唯一正式研发规则
- **AND** 低顺位内容不得与正式规则并列生效

#### Scenario: Contract evidence is described

- **WHEN** PRD 或 OpenSpec 描述仓库中的合同签署状态
- **THEN** 文案只说明仓库范围内没有已签署证据且现实状态未知
- **AND** 不得断言现实中已经签署或尚未签署

### Requirement: First-phase scope is closed and singular

一期 MUST 且仅能包含：单门店、单取餐点、到店自提且仅预约取餐；今天/明天两个营业日期；午餐/晚餐两个餐段及离散取餐时间点；员工与访客；微信小程序用户端、小程序商户端四屏和 PC 网页商户后台；商品分类、商品、最多三张图片、上下架、按取餐日期售罄、餐段可售和整数分价格；手机号白名单与全局折扣率；购物车、订单、微信支付、支付对账兜底；自动排产、备好、扫码/手工核销；规则内取消和原路全额退款；两类订阅消息；基础看板、财务对账和关键操作审计；服务端下发的开屏装饰图层。

一期 MUST 排除数量库存、库存预占、自动扣减、即时取餐、接单、手动提前开做、会员等级、优惠券、逐商品员工价、积分、储值、部分退款、配送、多门店、多档口、多商户、POS、打印机、叫号屏、企业月结、发票、第三方 ERP/财务/食堂系统对接及跨业务主入口。排除项 MUST 不得以隐藏开关、兼容分支、占位页或“后续启用”进入一期契约。

#### Scenario: Included scope is traced

- **WHEN** reviewer 对照 0818 PRD §3.1、§8.2 与一期页面/验收矩阵
- **THEN** 每个包含项都有唯一规则和至少一个验收落点
- **AND** 三端共享服务端事实而不形成独立业务模型

#### Scenario: Excluded capability is proposed

- **WHEN** 后续 change 尝试把任一排除项作为一期行为
- **THEN** 范围验收失败
- **AND** 该能力必须独立立项并获得新的正式范围授权

### Requirement: Cancellation and refund rules are deterministic

订单处于 `已预约` 且距取餐时间大于 30 分钟时，用户 MUST 可自助取消并只发起一次原路全额退款；订单已进入 `制作中` 后，用户自助取消 MUST 关闭，只能联系商户处理（I9）。主账号 MUST 可按业务处理结果发起原路全额退款；一期 MUST 拒绝部分退款。

用户自助取消或主账号发起退款后，订单 MUST 进入 `退款中`；只有微信确认退款成功后才 MUST 进入 `已退款`。退款结果失败或不确定时 MUST 保持 `退款中` 并在 PC 后台产生待处理标记，不得创建 `异常` 状态。由于一期没有数量库存，退款 MUST 不得执行库存返还（I10）。

#### Scenario: User cancels before automatic production

- **WHEN** 订单为 `已预约` 且距取餐时间大于 30 分钟
- **THEN** 系统接受用户取消、幂等发起一次原路全额退款并进入 `退款中`
- **AND** 微信确认退款成功后转为 `已退款`

#### Scenario: User attempts cancellation after production starts

- **WHEN** 订单已进入 `制作中`、`待取餐` 或 `已完成`
- **THEN** 用户端不得提供自助取消
- **AND** 订单只能交由主账号按业务结果处理

#### Scenario: Refund result is not confirmed

- **WHEN** 微信尚未确认退款成功或退款调用失败
- **THEN** 订单保持 `退款中` 且后台可查询待处理标记
- **AND** 系统不得显示 `已退款`、创建 `异常` 状态或返还库存

#### Scenario: Partial refund is requested

- **WHEN** 任一调用方请求小于订单实付金额的退款
- **THEN** 一期服务端拒绝该请求
- **AND** 不创建部分退款记录

### Requirement: Employee identity is decided by an active phone list

用户浏览门店、分类、商品与公开营业信息 MUST 不要求手机号；首次提交订单前 MUST 建立微信会话并完成手机号授权，服务端标准化手机号后以启用的员工折扣白名单决定员工/访客身份。微信授权主手机号命中即生效；用户手工填写的附加手机号 MUST 以“手机号 + 归一化姓名”同时命中同一启用记录才生效，附加手机号数量上限在 P4 决定前 MUST 不由实现者假设（I11）。

订单创建 MUST 固化内部用户引用、员工/访客身份、折扣率和名单版本快照；客户端不得让用户自选员工身份。页面、日志和追踪台账 MUST 不保存或展示未脱敏手机号及名单个人数据。

#### Scenario: Visitor browses without identification

- **WHEN** 未绑定手机号的用户启动并浏览商品
- **THEN** 系统允许浏览且不在启动时请求手机号授权
- **AND** 页面只显示原价

#### Scenario: Authorized primary phone matches

- **WHEN** 用户提交订单前授权的微信主手机号命中启用白名单
- **THEN** 服务端将报价识别为员工并固化名单版本
- **AND** 姓名不作为主手机号命中的附加前置条件

#### Scenario: Manually entered phone only partially matches

- **WHEN** 手工附加手机号只命中手机号、姓名不匹配或记录已停用
- **THEN** 该附加手机号不得取得员工身份
- **AND** 客户端不得绕过双要素校验

### Requirement: External readiness follows one twelve-gate chain

外部依赖台账 MUST 且只能按以下 12 个 Gate 顺序追踪：主体一致性；小程序注册与认证；餐饮类目；备案；项目成员与体验成员；云资源、域名实名与 ICP；HTTPS 与服务器域名；商户号与 AppID 绑定；API 安全状态；交易结算管理；隐私与手机号；体验版真机、审核、客户发布确认（最后一项内部严格按此三步推进）。会员等级与优惠券的原合同 Gate MUST 注销，不得保留第 13 Gate。

每个 Gate MUST 只记录责任方、非敏感证据引用、状态、阻塞阶段和更新时间。尚未取得完成证据的 Gate MUST 为 `BLOCKED_EXTERNAL`，取得可核查证据后才 MUST 为 `READY`；外部 Gate 只阻塞对应联调、UAT、提审或发布阶段，不得阻塞本地产品/技术 OpenSpec。订阅消息外部准备 MUST 只申请待取餐提醒和退款结果两类模板；微信支付外部准备 MUST 包含退款与支付结果查询能力。

#### Scenario: Local baseline is adopted before external readiness

- **WHEN** 12 个外部 Gate 尚未齐备但本 W0 产品基线内容已确定
- **THEN** 本 change 可继续严格校验、候选和独立验证
- **AND** 未完成 Gate 不得被写成当前 PASS

#### Scenario: External ledger is reviewed

- **WHEN** reviewer 检查外部依赖台账
- **THEN** 只看到 12 个 Gate、对应状态及非敏感证据引用
- **AND** 不存在会员券 Gate、密钥、证书正文、账号标识、手机号或名单数据

#### Scenario: Release is requested

- **WHEN** 项目准备正式发布
- **THEN** 12 个 Gate 均有可核查完成证据
- **AND** 最后一个 Gate 已依次完成体验版真机、审核和客户发布确认

### Requirement: The baseline is traceable and has no behavioral TODO

唯一产品 PRD §1–§14 与本 spec MUST 逐项覆盖一期范围及 I1–I16，且每条规则 MUST 能追踪到页面/调用方、状态、角色、外部 Gate 和可执行验收；不适用维度 MUST 明确为“不适用”。旧 PRD MUST 不得残留第二套行为正文。

0818 PRD §16.3 的 P1–P5 MUST 保持未决，且 MUST 分别阻塞其后续受影响模块：P1 阻塞营业状态操作归属；P2 阻塞跨营业日未取订单运营处置时限；P3 阻塞 PC 后台扫码登录会话与设备信任；P4 阻塞附加手机号数量模型；P5 阻塞全局折扣率生效时机。它们 MUST 不阻塞本 W0 基线收敛，也不得被当前文档暂定口径伪装成客户确认。

#### Scenario: Invariants are audited

- **WHEN** reviewer 对照 0818 PRD §12 的 16 条编号不变量检查 delta
- **THEN** I1–I16 每条都有且只有一个一致的规范落点及可执行场景
- **AND** 即时单、数量库存、软预占、九态、四角色、会员券和逐商品员工价没有残留为有效要求

#### Scenario: A downstream module touches P1 through P5

- **WHEN** 后续 change 的行为或验收依赖 P1、P2、P3、P4 或 P5 任一事项
- **THEN** 该下游 change 在对应客户决策被记录前保持产品决策阻塞
- **AND** writer 不得采用 0818 PRD §16.3 的暂定口径替客户拍板

#### Scenario: This W0 change is reviewed

- **WHEN** 本 change 只执行旧 PRD 废止指针和完整 OpenSpec delta 的内容/结构收敛
- **THEN** P1–P5 不构成本 change 的 `BLOCKED_EXTERNAL` 或实现阻塞
- **AND** strict、内容映射、diff 与 owned-path 检查仍可独立获得结果

## ADDED Requirements

### Requirement: Product availability has no quantity inventory

一期 MUST 不建立数量库存、软预占、自动扣减、库存返还或超卖判断（I3）。商品可售性 MUST 只有长期的上下架与按取餐日期生效的售罄两个维度；营业日 D 的售罄 MUST 只屏蔽 D 当天的订单、不得影响 D+1 预约，并 MUST 在次日自然恢复。售罄 MUST 可由授权商户手工自由切换，系统不得从订单数量推导或自动切换。

#### Scenario: Merchant marks a product sold out

- **WHEN** 商户在营业日 D 把商品标为售罄
- **THEN** 商品在 D 当天仍可展示但不可下单
- **AND** D+1 的预约不受影响且次日自然恢复可售

#### Scenario: Demand exceeds kitchen capacity

- **WHEN** 某商品已收订单量超过现场备餐能力
- **THEN** 系统不得声称存在自动超卖保护或数量拦截
- **AND** 商户只能手工售罄并人工处理已收订单

### Requirement: Only confirmed payment creates an order

调起微信支付前 MUST 只创建用户不可见、不进订单列表、不占取餐号的内部预支付记录；只有服务端确认微信支付成功后才 MUST 幂等生成订单并分配取餐号。用户取消支付、支付失败或结果尚未确认 MUST 不生成订单，前端成功回调 MUST 不得作为支付成功事实（I5）。

系统 MUST 定时扫描已发起支付但未生成订单的预支付记录并查询微信真实支付结果：未支付则作废；已支付则幂等补建订单；补建失败则进入 PC 后台支付待处理列表，由主账号人工退款或建单。该对账链路 MUST 不引入 `异常` 订单状态（I6）。

#### Scenario: User cancels or payment fails

- **WHEN** 微信支付被用户取消、明确失败或尚无服务端成功事实
- **THEN** 系统不生成订单、不分配取餐号
- **AND** 用户端只显示可恢复的支付结果确认状态

#### Scenario: Payment callback is repeated

- **WHEN** 同一 `out_trade_no` 的服务端支付成功通知重复到达
- **THEN** 系统只生成一张订单并返回同一取餐号
- **AND** 不重复产生订单、支付或编号副作用

#### Scenario: Callback is lost but payment succeeded

- **WHEN** 定时对账查询到预支付记录已由微信确认支付但系统尚无订单
- **THEN** 系统幂等补建订单并按取餐时间判定初始状态
- **AND** 若补建失败则创建支付待处理条目而不是异常订单

### Requirement: Orders use one six-state production state machine

生产订单状态 MUST 且仅能为 `已预约`、`制作中`、`待取餐`、`已完成`、`退款中`、`已退款`（I7）。主链路 MUST 为“服务端确认支付成功 → `已预约` → 取餐前 30 分钟自动 `制作中` → 商户备好 `待取餐` → 核销 `已完成`”；支付成功时距取餐不足 30 分钟 MUST 直接生成 `制作中` 订单。系统 MUST 不存在接单或手动提前开做动作（I8）。

所有状态转换 MUST 由服务端校验前置状态、资源权限与幂等键并记录审计；生产 MUST 禁止撤销和回退。营业日结束后仍为 `待取餐` 的订单 MUST 通过“未取餐”查询口径处理，该口径 MUST 不是新状态。

#### Scenario: Scheduled production starts

- **WHEN** `已预约` 订单到达取餐时间前 30 分钟
- **THEN** 服务端定时任务幂等推进为 `制作中`
- **AND** 不需要商户接单或提前开做

#### Scenario: Payment completes inside thirty minutes

- **WHEN** 服务端确认支付成功时距取餐时间不足 30 分钟
- **THEN** 新订单初始状态直接为 `制作中`
- **AND** 用户端不提供自助取消

#### Scenario: Merchant fulfills an order

- **WHEN** 商户标记 `制作中` 订单备好并随后核销
- **THEN** 订单依次进入 `待取餐` 与 `已完成`
- **AND** 二者不得跳过合法前置状态

#### Scenario: A transition is repeated or reversed

- **WHEN** 相同幂等键重复转换，或客户端尝试撤销/回退生产状态
- **THEN** 重复请求返回第一次结果而回退请求被拒绝
- **AND** 不重复产生退款、营收、通知或核销副作用

### Requirement: Employee price uses one global discount rate

金额 MUST 以整数分由服务端保存与计算。算价链 MUST 固定为“原价小计 → 员工折扣 → 应付”；命中员工折扣白名单时 MUST 对每个商品按“原价单价 × 全局单一折扣率”四舍五入到分，再乘数量求和，页面逐商品员工价 MUST 等于结算成交价。访客 MUST 使用原价；一期 MUST 不得存在等级、优惠券、逐商品员工价或不参与折扣开关（I12）。

报价与订单明细 MUST 固化原价、折扣率、折后价、身份和价格版本；后续配置变化 MUST 不回算历史订单。全局折扣率何时对新报价生效属于 P5，在客户确认前 MUST 不由实现者假设即时、定时或按营业日生效。

#### Scenario: Employee-priced product is quoted

- **WHEN** 已识别员工请求商品报价
- **THEN** 服务端先把该商品原价单价乘全局折扣率并舍入到分，再乘数量
- **AND** 菜单、详情和结算明细的逐商品价格一致

#### Scenario: Visitor or client-supplied price is quoted

- **WHEN** 访客购买商品或客户端提交自定义价格
- **THEN** 服务端按原价计算且忽略客户端价格
- **AND** 不应用会员、优惠券或逐商品员工价

### Requirement: Every first-phase order uses one reserved pickup time point

一期 MUST 使用单门店、单取餐点、到店自提且仅预约取餐（I1），门店时区 MUST 为 `Asia/Shanghai`。可预约日期 MUST 只有今天与明天，餐段 MUST 只有午餐与晚餐（I2）；每餐段 MUST 有一个固定截单时刻，餐段内所有离散取餐时间点 MUST 共用该截单时刻。取餐时间点 MUST 是预计备好约定时刻而不是到场窗口（I4）。

用户 MUST 可直接进入菜单并默认选中当前时刻之后第一个未截单时间点；当天全部截单时 MUST 默认落到明天。提交支付前服务端 MUST 校验取餐日期、餐段和固定截单时刻，已截单时 MUST 拦截支付、保留购物车并返回时间选择。订单 MUST 固化取餐日期、餐段、时间点、截单时刻和取餐点快照。

#### Scenario: User chooses an available pickup time

- **WHEN** 用户选择今天或明天、午餐或晚餐中尚未截单的离散时间点
- **THEN** 系统允许进入最终支付校验并固化完整取餐快照
- **AND** 菜单只展示该餐段可售商品

#### Scenario: All pickup periods today are closed

- **WHEN** 今天午餐和晚餐均已过固定截单时刻
- **THEN** “今天”整体置灰且默认选择明天第一个时间点
- **AND** 已截单餐段按组折叠并展示截止时刻

#### Scenario: User requests an invalid pickup mode

- **WHEN** 用户请求即时取餐、后天及以后、早餐、非离散时间点或已截单餐段
- **THEN** 服务端拒绝进入支付
- **AND** 不生成订单或任何库存占用

### Requirement: Merchant permissions use two server-enforced roles

一期商户角色 MUST 且仅能为主账号与子账号，主账号 MUST 可配置多个（I13）。子账号 MUST 仅能在小程序商户端查看订单、标记备好、扫码或手工核销、切换商品可售/售罄；主账号 MUST 具备这些能力，并独占 PC 后台商品/分类配置、价格、折扣率、名单、退款、财务、营业设置、开屏图层、看板和支付待处理能力。子账号 MUST 不得进入 PC 后台。

商户账号名单 MUST 与员工折扣白名单分开；小程序通过微信手机号与商户名单绑定，PC 后台使用微信扫码登录且仅主账号可通过。所有资源权限 MUST 由服务端执行；PC 会话时长、设备信任与并发登录策略属于 P3，营业状态切换归属属于 P1，在客户确认前 MUST 保持下游阻塞。

#### Scenario: Sub-account performs field operations

- **WHEN** 子账号查看订单、标记备好、核销或切换商品可售/售罄
- **THEN** 服务端允许对应小程序端动作
- **AND** 拒绝改价、上下架、退款、配置、财务与 PC 后台访问

#### Scenario: Main account enters PC administration

- **WHEN** 微信扫码身份命中启用的主账号记录
- **THEN** 服务端允许进入 PC 后台授权资源
- **AND** 不因该用户是否命中员工折扣白名单而改变商户权限

#### Scenario: Unassigned user enters a merchant route

- **WHEN** 普通用户或未命中商户名单的用户访问商户页面或接口
- **THEN** 服务端拒绝访问
- **AND** 客户端入口或身份选择不得提升权限

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

## REMOVED Requirements

### Requirement: Inventory is keyed by service date, meal period, and product

**Reason**: 客户确认一期不做数量库存、预占、自动扣减或超卖保护；继续保留库存键会直接违反 I3。

**Migration**: 使用新增 requirement `Product availability has no quantity inventory`，只保留上下架与按取餐日期的手工售罄。

### Requirement: Order submission uses a bounded atomic soft hold

**Reason**: 客户确认删除 15 分钟软预占、待支付订单、迟到支付重占与自动退款；支付成功前不存在订单。

**Migration**: 使用新增 requirement `Only confirmed payment creates an order`，以内部预支付记录、服务端确认支付和支付对账兜底处理资金事实。

### Requirement: Orders use one nine-state production state machine

**Reason**: 客户确认公开生产订单只有六态，并删除待支付、已支付待接单、已取消和异常。

**Migration**: 使用新增 requirement `Orders use one six-state production state machine`。

### Requirement: Employee price is an optional fixed per-product amount

**Reason**: 客户确认一期使用全局单一折扣率，不做逐商品 `employee_price`、等级或优惠券。

**Migration**: 使用新增 requirement `Employee price uses one global discount rate`，逐商品舍入到分后再乘数量。

### Requirement: Every first-phase order uses one fixed pickup slot

**Reason**: 客户澄清取消固定取餐时段模型，改为午晚餐段内共享固定截单时刻的离散取餐时间点。

**Migration**: 使用新增 requirement `Every first-phase order uses one reserved pickup time point`。

### Requirement: Merchant permissions use four server-enforced roles

**Reason**: 客户澄清一期只有主账号和子账号两个商户角色，四角色模型已废止。

**Migration**: 使用新增 requirement `Merchant permissions use two server-enforced roles`。
