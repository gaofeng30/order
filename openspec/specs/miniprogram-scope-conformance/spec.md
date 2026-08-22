# miniprogram-scope-conformance Specification

## Purpose
定义微信小程序端暴露的页面、路由、入口、全局态、接口契约与算价链必须与生效 spec `mvp-product-baseline` 的一期范围保持一致，不得存在已排除能力的实现、入口或残留表述。
## Requirements
### Requirement: The mini program exposes no route for an excluded capability

`apps/wechat-miniprogram/app.json` 的 `pages` 列表 MUST NOT 包含任何实现生效 spec 排除能力的路由。会员等级、会员名单、名单导入、会员编辑、优惠券列表、优惠券编辑与用户卡包的页面目录 MUST NOT 存在于仓库中。

#### Scenario: Route list is audited

- **WHEN** 检查 `app.json` 的 `pages` 列表
- **THEN** 其中不存在 `my-coupons`、`admin-levels`、`admin-members`、`admin-member-edit`、`admin-member-import`、`admin-coupons`、`admin-coupon-edit`
- **AND** 这些页面对应的目录在仓库中不存在

#### Scenario: Excluded page is reintroduced

- **WHEN** 任一 change 为排除能力新增页面目录或 `app.json` 路由
- **THEN** scope 一致性检查失败
- **AND** 该能力必须先取得新的正式范围授权

### Requirement: The mini program holds no state or contract for an excluded capability

小程序 MUST NOT 在 `app.js` 的 `globalData` 中保存会员等级、会员名单、优惠券或券使用计数。`utils/data.js` MUST NOT 导出这些能力的种子数据。`utils/api.js` MUST NOT 声明这些能力的后端接口契约。算价引擎 `utils/promo.js` 与券票卡组件 MUST NOT 存在。

#### Scenario: Global state is inspected

- **WHEN** 加载小程序 App 实例并检查 `globalData`
- **THEN** 不存在 `levels`、`members`、`coupons`、`couponUsed` 任一键

#### Scenario: Contract layer is inspected

- **WHEN** 加载 `utils/api.js` 并枚举其导出
- **THEN** 不存在会员等级、会员名单、名单导入、优惠券或用户卡包的任何方法

#### Scenario: Seed data is inspected

- **WHEN** 加载 `utils/data.js` 并枚举其导出
- **THEN** 不存在 `LEVELS`、`MEMBERS`、`COUPONS`、`MY_COUPON_USED` 任一导出

### Requirement: Checkout pricing shows only what the first-phase scope defines

结算页 MUST NOT 提供选券入口、券弹层或等级折扣展示。在全局折扣率实现之前，结算页 MUST 只展示商品小计并令应付金额等于商品小计。

订单 MUST NOT 固化会员等级或优惠券相关字段。客户端 MUST NOT 依赖任何算价引擎模块完成结算。

#### Scenario: User checks out without any discount mechanism

- **WHEN** 用户加购商品后进入结算页并提交订单
- **THEN** 结算页展示商品小计且应付金额等于商品小计
- **AND** 生成的订单不含等级或优惠券字段

#### Scenario: Checkout template is audited

- **WHEN** 检查结算页 WXML
- **THEN** 不存在选券入口、券弹层或等级折扣行
- **AND** 结算页脚本不引用算价引擎模块

### Requirement: User and merchant entries drop excluded capabilities

个人中心 MUST NOT 展示「我的优惠券」入口或会员等级胶囊。商户中心 MUST NOT 展示「会员与营销」分组。菜品编辑 MUST NOT 包含删除菜品时摘除优惠券适用范围的联动。

#### Scenario: Profile entries are audited

- **WHEN** 检查个人中心页面与模板
- **THEN** 不存在优惠券入口、等级胶囊或券数量字段

#### Scenario: Merchant center entries are audited

- **WHEN** 检查商户中心页面与模板
- **THEN** 不存在「会员与营销」分组及其下属入口

#### Scenario: Product deletion no longer touches coupons

- **WHEN** 商户在菜品编辑页删除一个菜品
- **THEN** 操作不引用任何优惠券数据
- **AND** 不产生与券适用范围相关的提示

### Requirement: The mini program exposes no retired entry screen

品牌选择页与小程序商户端经营工作台 MUST NOT 存在于 `app.json` 的 `pages` 列表或仓库中。路由封装 MUST NOT 提供指向品牌选择页的跳转方法。

#### Scenario: Retired routes are audited

- **WHEN** 检查 `app.json` 的 `pages` 列表与页面目录
- **THEN** 不存在 `pages/brand/brand` 与 `pages/admin-dashboard/admin-dashboard`
- **AND** 路由封装不含指向品牌选择页的方法

### Requirement: The home screen carries no marketing surface

首页 MUST NOT 渲染营销 Banner 轮播、入群模块、会员面板、推荐商品或今日招牌。首页脚本 MUST NOT 保留这些能力对应的数据、轮播索引或事件处理器。

首页服务宫格 MUST NOT 提供会员中心或联系客服入口。

#### Scenario: Home surfaces are audited

- **WHEN** 检查首页脚本与模板
- **THEN** 不存在 Banner、轮播索引、入群、会员面板、招牌位或推荐商品
- **AND** 服务宫格不含会员中心与联系客服

### Requirement: The merchant tab bar carries only first-phase tabs

小程序商户端底部导航 MUST 只有订单、核销、菜品三格，MUST NOT 包含工作台或商户中心。

商户中心页 MUST NOT 存在于小程序端：其承载的营业设置、分类管理与开屏图层均属 PC 后台能力。收敛后仍需保留的「切换身份」MUST 由订单页导航栏右侧承担，MUST NOT 留下无入口的孤儿页。

#### Scenario: Merchant tab bar is audited

- **WHEN** 检查底部导航组件的商户端分支
- **THEN** 只声明订单、核销、菜品三格
- **AND** 不声明工作台或商户中心

#### Scenario: The identity screen stays reachable

- **WHEN** 商户进入订单页
- **THEN** 导航栏右侧提供返回身份选择页的入口
- **AND** 商户中心页不存在于仓库中

### Requirement: Seed data owned only by a removed screen is dropped

仅供已删除页面使用的种子数据 MUST 一并移除，MUST NOT 作为无主常量保留在种子模块中。

#### Scenario: Orphan seed is audited

- **WHEN** 加载种子模块并枚举导出
- **THEN** 不存在仅由已删除的经营工作台使用的销量排行常量

### Requirement: Product records carry no retired catalog field

商品记录 MUST NOT 保存标签、过敏原、月售或数量库存。种子数据、接口契约与页面数据 MUST 一并不含这四类字段。

商品的销售状态字段 MUST 保留：可售性由上下架与售罄开关承担，见生效 spec `mvp-product-baseline` 的 `Product availability uses a per-service-date sellout switch`。

#### Scenario: Seed products are inspected

- **WHEN** 加载种子模块并逐条检查商品
- **THEN** 不存在 `tags`、`allergens`、`sold`、`stock` 任一字段
- **AND** 每条商品仍带销售状态字段

#### Scenario: Product contract is inspected

- **WHEN** 检查商品接口契约的入参、校验与新建默认值
- **THEN** 不接受数量入参、不做库存校验、不为新建商品填充标签或过敏原
- **AND** 销售状态契约仍然可调用

### Requirement: Merchant product screens show no retired catalog field

商户端菜品列表 MUST NOT 展示库存数、库存告急标记或月售。菜品编辑 MUST NOT 提供库存输入项。

菜品列表 MUST 保留售罄与上下架控件。

#### Scenario: Merchant product list is audited

- **WHEN** 检查商户端菜品列表的页面数据与模板
- **THEN** 行数据不含 `stock`、`sold`、`tags`、`allergens` 或告急标记
- **AND** 售罄切换控件仍然存在

#### Scenario: Merchant product editor is audited

- **WHEN** 检查菜品编辑页的脚本与模板
- **THEN** 不存在库存字段、库存输入或数量校验

#### Scenario: Sale status still works after the fields are gone

- **WHEN** 商户对某商品切换售罄再切回可售
- **THEN** 该商品的销售状态随之变化
- **AND** 过程不依赖任何数量字段

### Requirement: The mini program drives orders through the six-state machine only

小程序的状态语义映射 MUST 只覆盖六态与非订单语义，MUST NOT 保留 `待支付`、`待制作`、`已取消` 等已废止状态。

商户端可推进的转换 MUST 只有 `制作中 → 待取餐` 与 `待取餐 → 已完成`。推进 MUST 单向：到达终态后再次推进 MUST 不改变状态。

Toast 组件 MUST NOT 提供回退动作，路由与工具层 MUST NOT 保留任何回退回调。

订单种子与运行态订单 MUST 只使用六态。

#### Scenario: Status semantics are inspected

- **WHEN** 检查状态语义映射
- **THEN** 六态全部有语义色，且不存在已废止状态

#### Scenario: An order is advanced past its terminal state

- **WHEN** 商户对已处于 `已完成` 的订单再次执行推进
- **THEN** 状态保持 `已完成`

#### Scenario: Undo is searched for

- **WHEN** 检查 Toast 组件与工具层
- **THEN** 不存在回退回调、回退按钮或撤销文案

### Requirement: The mini program offers reservation ordering only

一期 MUST NOT 提供即时取餐。首页服务宫格、结算页取餐方式切换、订单列表与结果页 MUST NOT 出现即时单入口、订单类型字段或「尽快」文案。工具层 MUST NOT 导出下单模式状态。

结算 MUST 生成带取餐日期与取餐时间的预约单；订单 MUST NOT 携带订单类型字段。

核销二维码 MUST 只在订单进入 `待取餐` 后展示。

#### Scenario: Checkout creates a reservation

- **WHEN** 用户加购商品后提交订单
- **THEN** 订单为 `已预约`（距取餐不足 30 分钟时为 `制作中`）且带取餐时间
- **AND** 订单不含订单类型字段

#### Scenario: Immediate ordering is searched for

- **WHEN** 检查首页、结算页、结果页与订单列表
- **THEN** 不存在下单模式状态、即时单入口或「尽快」文案

#### Scenario: The QR code is gated on readiness

- **WHEN** 订单处于 `已预约` 或 `制作中`
- **THEN** 订单详情不渲染核销二维码，只展示取餐号、取餐时间与状态

### Requirement: Pickup time is chosen once and shared across the ordering flow

可预约营业日期 MUST 只有今天与明天。每个餐段 MUST 有一个固定截单时刻，餐段内全部取餐时间共用；取餐时间 MUST 为由餐段范围与粒度推导的离散时间点，粒度 MUST 可配置而非写死。

取餐时间 MUST 在菜单顶部条选定并跨页共享；结算页 MUST 只读展示该选择，MUST NOT 再提供第二套日期或时间选择器。

默认取餐时间 MUST 为当前时刻之后第一个未截单的时间点；当日全部餐段截单时 MUST 落到下一个可选日期。

#### Scenario: Menu opens with a usable default

- **WHEN** 用户进入菜单
- **THEN** 顶部条展示当前时刻之后第一个未截单的取餐时间
- **AND** 点击该条展开取餐时间选择弹层

#### Scenario: Checkout reuses the chosen time

- **WHEN** 用户在菜单顶部条选定取餐时间后进入结算
- **THEN** 结算页只读展示该取餐时间并提供回菜单修改的入口
- **AND** 结算页不含日期或时间选择控件

#### Scenario: Times are derived from range and step

- **WHEN** 由餐段的取餐起止与粒度推导时间点
- **THEN** 结果为该范围内按粒度均分的离散时刻
- **AND** 粒度变化时时间点随之变化

### Requirement: A cut-off meal period is folded, not itemised

取餐时间选择弹层 MUST 按餐段分组。已截单餐段 MUST 整组折叠并标注其截止时刻，MUST NOT 逐条渲染不可选时间点。当日全部餐段截单时，该日期 MUST 标注为已截单。

提交订单时 MUST 重新校验目标取餐时间所属餐段是否仍在截单前；已截单 MUST 拦截提交、提示重选，且 MUST 保留购物车内容。

#### Scenario: A period is past its cutoff

- **WHEN** 当前时刻已过某餐段的固定截单时刻且所选日期为今天
- **THEN** 该餐段在弹层中整组折叠并标注截止时刻
- **AND** 该组不渲染任何可选时间点

#### Scenario: Submission targets a cut-off period

- **WHEN** 用户提交订单而所选取餐时间所属餐段已截单
- **THEN** 提交被拦截且不生成订单
- **AND** 购物车内容保留

### Requirement: Cancel eligibility depends only on state and remaining time

自助取消资格 MUST 只由订单状态与距取餐分钟数决定：`已预约` 且距取餐大于 30 分钟。判定 MUST NOT 读取任何随即时单删除的订单类型字段。

#### Scenario: Cancel eligibility is evaluated

- **WHEN** 判定一条 `已预约` 且距取餐 102 分钟的订单
- **THEN** 允许自助取消
- **AND** 同样状态但距取餐 18 分钟、或状态为 `制作中` 的订单不允许

### Requirement: The mini program merchant surface is exactly four screens

小程序商户端 MUST 只有订单管理、订单详情、扫码核销、菜品四屏。完整商品配置、分类管理、营业设置、开屏图层与任何名单管理 MUST NOT 存在于小程序端，它们属 PC 后台能力。

菜品页 MUST 只提供可售 / 售罄切换，MUST NOT 提供上架新菜、编辑菜品或上下架开关。

营业状态切换 MUST 保留在小程序商户端的订单页：现场临时休息需要手机可操作。

小程序端 MUST NOT 保留任何「建设中」占位入口。

#### Scenario: Merchant routes are audited

- **WHEN** 检查 `app.json` 的商户端路由与页面目录
- **THEN** 只存在订单管理、订单详情、扫码核销、菜品四屏
- **AND** 菜品编辑、分类管理、营业设置、开屏图层与商户中心均不存在

#### Scenario: A merchant opens the product screen

- **WHEN** 商户进入菜品页
- **THEN** 每行只提供可售 / 售罄切换
- **AND** 不存在上架新菜入口、编辑入口或上下架开关

#### Scenario: A merchant needs to pause service on site

- **WHEN** 商户在订单页切换营业状态
- **THEN** 门店状态随之变化
- **AND** 该能力不依赖 PC 后台

### Requirement: The mini program renders no locally stored launch layer

小程序端 MUST NOT 以本机存储实现开屏装饰图层。图层配置与图片 MUST 由服务端下发；在该接口就位前，小程序 MUST 不渲染任何图层，MUST NOT 保留只能读取本机存储的图层模块或组件。

#### Scenario: The launch layer is audited

- **WHEN** 检查小程序的图层模块、组件与身份选择页
- **THEN** 不存在读取本机存储的图层实现
- **AND** 身份选择页不渲染图层

### Requirement: The merchant order list is searchable across lanes

小程序商户端订单页 MUST 提供可输入的搜索框，按**取餐号、订单号、手机号**定位订单（联系人姓名作为同一输入框的附加匹配项）。搜索 MUST 跨全部状态泳道，MUST NOT 被当前选中的泳道限制。

搜索 MUST 是一个查询：`kw` MUST NOT 写入 `globalData`，MUST NOT 成为订单模型上的字段或状态。选择任一泳道 MUST 退出搜索态。

搜索框 MUST 是受控输入元素并绑定输入事件；MUST NOT 以静态文本冒充已交付的搜索能力。

#### Scenario: A customer reports a pickup code at the window

- **WHEN** 商户在订单页输入当前营业日的 4 位取餐号
- **THEN** 该订单出现在结果中，无论它处于哪个状态泳道
- **AND** 当前泳道的选择不影响该结果

#### Scenario: A merchant searches by order number or phone

- **WHEN** 商户输入订单号片段或手机号片段
- **THEN** 匹配的订单出现在结果中
- **AND** 结果可以跨越多个状态

#### Scenario: The merchant leaves search

- **WHEN** 商户点击任一状态泳道
- **THEN** 搜索关键词被清空
- **AND** 列表回到该泳道的完整内容

### Requirement: Pickup codes are four digits accumulated per pickup date

取餐号 MUST 为 4 位数字，MUST NOT 携带 `A` / `B` 等即时单遗留前缀。用户端与商户端展示的取餐号 MUST 使用同一格式。

取餐号 MUST 在同一取餐日期内唯一，MAY 在不同取餐日期之间重复。种子数据 MUST 包含至少一组跨营业日重复的取餐号，否则「限定当前营业日」这条约束不可证伪。

营业日 MUST 来自单一常量，MUST NOT 由运行时系统时钟推导 —— 否则同一份种子数据的断言结果会随日期翻转。

#### Scenario: Pickup code format is audited

- **WHEN** 检查用户端与商户端全部订单种子的取餐号
- **THEN** 每一个都匹配 4 位数字
- **AND** 不存在字母前缀

#### Scenario: The same code exists on two business days

- **WHEN** 检查种子数据
- **THEN** 存在两张取餐号相同但取餐日期不同的订单
- **AND** 其中一张属于当前营业日

### Requirement: A pickup code resolves only within the current business day

按取餐号定位订单 MUST 限定在当前营业日。搜索与手工核销 MUST 共用同一份解析实现，MUST NOT 各自实现该规则。

4 位纯数字输入 MUST 同时按手机号片段匹配：跨营业日歧义只是取餐号的属性，手机尾号没有该问题。

某取餐号在当前营业日无果、却存在于其他营业日时，系统 MUST 报出该事实并指出替代定位方式，MUST NOT 只返回空列表或「无效取餐号」。

#### Scenario: A stale pickup code is entered for verification

- **WHEN** 商户手工输入一个属于其他营业日的取餐号
- **THEN** 核销被拒绝
- **AND** 提示指出该号属于哪个营业日以及可改用订单号或手机号定位

#### Scenario: A four-digit phone fragment is searched

- **WHEN** 商户输入手机号的 4 位尾段
- **THEN** 持有该手机号的订单出现在结果中
- **AND** 结果不因该输入被当作取餐号而落空

#### Scenario: An order under refund is located

- **WHEN** 商户按手机号或订单号搜索一张处于 `退款中` 的订单
- **THEN** 该订单出现在结果中并显示其状态
- **AND** 商户端不因缺少 `退款中` 泳道而无法找到它

### Requirement: Order lines carry a product name snapshot

订单项 MUST 形如 `[id, name, qty, price, discountedPrice, flavors?, note?]`，其中 `name` 是下单当刻固化的商品名称快照。

渲染订单的任何路径 MUST NOT 按 `id` 回查商品表取名称。`id` 保留用于退款、对账与销量归集，MUST NOT 作为显示名称的来源。

名称 MUST 与原价、折后价同批固化：§5.6 要求订单固化价格事实，名称属于同一类事实，两者 MUST NOT 采用不同的存储策略。

`name` MUST 位于必填段内（`id` 之后、`qty` 之前），MUST NOT 追加在 `flavors` / `note` 这两个可选尾项之后 —— 否则元组不再有稳定 arity，未填口味的订单会取到错误的列。

#### Scenario: A product is renamed after an order was placed

- **WHEN** 商品在下单后改名或从目录移除
- **THEN** 历史订单仍显示下单当时的名称
- **AND** 渲染过程不发起任何按 id 的商品查询

#### Scenario: The order line shape is audited

- **WHEN** 检查两端订单种子的每一行订单项
- **THEN** 每行至少五项，第二项为非空字符串名称
- **AND** 数量、原价、折后价三项均为整数

### Requirement: A freshly placed order opens on every user surface

结算写出的订单 MUST 能被订单列表与订单详情直接渲染，MUST NOT 抛异常。

结算写出的字段类型 MUST 与种子订单一致：同一字段 MUST NOT 在一处是数字、在另一处是格式化字符串。

商品图片不在订单中固化，因此商品不在本地目录时订单详情 MUST 回落占位图，MUST NOT 因此报错。

#### Scenario: A user places an order and opens it

- **WHEN** 用户完成结算后打开「我的订单」与订单详情
- **THEN** 两个页面都正常渲染，列出商品名称与数量
- **AND** 两个页面都不抛异常

#### Scenario: Order totals keep one type

- **WHEN** 比较结算新建的订单与种子订单的金额字段
- **THEN** 两者类型相同
- **AND** 不存在一处为字符串、一处为数字的字段

### Requirement: Mini program orders carry the settlement facts in integer cents

小程序订单 MUST 携带 §15.6.2 的全部结算事实：`pickupDate`、`pickupTime`、`mealPeriod`、`pickupPoint`、`paidAt`、`subtotal`、`discountRate`、`discountCut`、`total`、`isStaff`、`contact`、`phone`、`items`。

`subtotal`、`discountCut`、`total` 与 `items` 行的原价、折后价 MUST 为整数分，MUST NOT 为元。结算恒等式 MUST 成立：逐行 `qty × price` 之和等于 `subtotal`；逐行 `qty × discountedPrice` 之和等于 `total`；`subtotal − discountCut` 等于 `total`。

结算写出的订单 MUST 与种子订单携带同一组字段、同一单位，MUST NOT 少写任一结算事实。

身份识别链路就位前 `isStaff` MUST 为 false、`discountRate` MUST 为 100、`discountCut` MUST 为 0 —— 这是「所有人都是访客价」这一真实业务状态的表达，不是占位符。

#### Scenario: Every order is audited against 15.6.2

- **WHEN** 检查用户端与商户端全部订单种子
- **THEN** 每单携带全部结算字段
- **AND** 三条结算恒等式逐单成立
- **AND** 金额为整数分而非元

#### Scenario: A user completes checkout

- **WHEN** 用户结算下单
- **THEN** 新订单携带与种子订单相同的字段集合与单位
- **AND** 三条结算恒等式在新订单上同样成立

### Requirement: Time to pickup is derived, never stored

订单 MUST NOT 携带 `minsToPickup` 或 `pickupLabel` 字段。距取餐的剩余时间与取餐文案 MUST 从 `pickupDate` 与 `pickupTime` 现算。

§7.6 的取消窗口 MUST 依据**当前时刻**判定，MUST NOT 依据下单时刻冻结的值 —— 否则时钟一旦真实流动，本该拒绝的取消会被放行。

#### Scenario: The cancel window is evaluated

- **WHEN** 判断一张 `已预约` 订单能否自助取消
- **THEN** 判定依据是当前时刻与取餐时刻之差
- **AND** 订单记录上不存在任何冻结的剩余分钟数

#### Scenario: Pickup text is rendered

- **WHEN** 订单列表、订单详情或支付结果页展示取餐时间
- **THEN** 文案由 `pickupDate` 与 `pickupTime` 推导
- **AND** 订单记录上不存在 `pickupLabel` 字段

### Requirement: Only the order note lives at order level

订单 MUST NOT 携带整单级口味字段。口味 MUST 绑定在 `items` 行内（§15.6.4），整单级 MUST 只有 `orderNote`。

展示整单口味时 MUST 聚合各行口味与备注，MUST NOT 因删除整单级字段而丢失信息。

#### Scenario: Order level fields are audited

- **WHEN** 检查订单种子与结算写出的订单
- **THEN** 不存在 `flavor` 或 `flavors` 字段
- **AND** 存在 `orderNote`

#### Scenario: A merchant looks at an order with per-item flavors

- **WHEN** 商户查看一张各菜品口味不同的订单
- **THEN** 每行的口味与备注都可见
- **AND** 整单备注单独可见

### Requirement: Money is formatted through exactly one entry point

整数分转元 MUST 只有一处实现。页面与模板 MUST NOT 自行做除法或 `toFixed`，MUST 经由该入口渲染。

该入口 MUST NOT 复用目录层的格式化函数 —— 后者在输入非法时抛目录不可用错误，会把显示问题伪装成网络故障。

#### Scenario: Money rendering is audited

- **WHEN** 检查全部页面与组件
- **THEN** 不存在第二处分转元实现
- **AND** 不存在按 100 做除法的页面代码

### Requirement: Shelf state and daily sell-out are two independent dimensions

商品的 `status` MUST 只有 `'on'` 与 `'off'` 两个取值，表示长期上下架。当日售罄 MUST NOT 存在商品对象上。

当日售罄 MUST 存为按取餐日期的独立记录，唯一键为 `(productId, serviceDate)`。记录的**存在**即表示该商品在该取餐日期售罄；MUST NOT 用布尔字段表达，否则「昨天标过又取消」与「今天还没标过」会成为两种形态表示同一件事。

可售性 MUST 由两个维度现算：上架且该取餐日期无售罄记录。MUST NOT 引入第三个综合状态枚举。

#### Scenario: The product model is audited

- **WHEN** 检查商品种子与全部读写点
- **THEN** `status` 只出现 `'on'` 与 `'off'`
- **AND** 商品对象上不存在任何售罄字段

#### Scenario: Yesterday's sell-out does not survive

- **WHEN** 某商品存在昨日的售罄记录而无今日记录
- **THEN** 它在今日可售
- **AND** 无需任何手工恢复动作

### Requirement: The merchant sell-out toggle is scoped to the business day

小程序商户端菜品页的售罄开关 MUST 写入或删除当前营业日的售罄记录，MUST NOT 修改 `status`。

在营业日 D 标记售罄 MUST NOT 影响 D+1 的可售性 —— 一期只有预约单，屏蔽次日预约会直接误伤主场景。

菜品页 MUST 仍然只提供售罄切换，MUST NOT 提供上下架、编辑或新增（§6.5、§15.5.2）。

#### Scenario: A merchant sells out at the window

- **WHEN** 商户在营业日 D 把某商品标为售罄
- **THEN** 该商品在 D 日不可售
- **AND** 该商品在 D+1 仍可售
- **AND** 该商品的 `status` 未被改动

#### Scenario: A merchant restocks in the evening

- **WHEN** 商户把已售罄的商品切回可售
- **THEN** 当日售罄记录被移除
- **AND** 该商品当日重新可售

### Requirement: The home screen shows the store notice and business status from configuration

首页 MUST 展示门店公告，内容 MUST 取自营业设置的公告字段，MUST NOT 在页面中写死。

首页 MUST 展示当前营业状态，取值为 `营业中` / `休息中` / `已截单`，MUST 跟随商户端的切换。营业状态 MUST NOT 由截单时刻派生 —— §6.9 允许主账号手动覆盖营业时间规则，派生值只是默认值而非事实。

首页 MUST NOT 保留任何「未开放」「即将上线」一类的占位角标或其渲染分支（§0.2）。

#### Scenario: The merchant edits the notice

- **WHEN** 营业设置里的公告内容变化
- **THEN** 首页展示新的公告
- **AND** 首页脚本与模板中不存在写死的公告文本

#### Scenario: The merchant pauses service

- **WHEN** 商户在小程序商户端把门店切为 `休息中`
- **THEN** 用户端首页展示 `休息中`
- **AND** 该展示不依赖截单时刻的推导

### Requirement: The home screen keeps a standing banner for orders in flight

存在 `已预约` / `制作中` / `待取餐` 订单时，首页顶部 MUST 常驻一条提示，展示进行中单数与**最近一单的取餐时刻**；无此类订单时 MUST NOT 渲染该提示。

「最近一单」MUST 按取餐时刻排序确定，MUST NOT 按下单时间 —— 用户关心的是下一顿何时能取。

任一订单进入 `待取餐` 时，提示 MUST 改为「已备好，可取餐」并高亮，MUST NOT 在该文案中继续强调单数而稀释行动号召。

点击提示 MUST 直达对应订单的取餐码页。

该提示是 §5.10 订阅消息被拒时的兜底：餐饮类目只能一次性订阅，拒绝授权的用户 MUST 仍能从首页得知餐已备好。

#### Scenario: A user has orders in flight

- **WHEN** 用户存在 `已预约` 或 `制作中` 订单且无 `待取餐`
- **THEN** 首页顶部展示进行中单数与最近一单的取餐时刻
- **AND** 点击进入该订单

#### Scenario: A meal is ready

- **WHEN** 用户的任一订单进入 `待取餐`
- **THEN** 提示文案变为「已备好，可取餐」并高亮
- **AND** 点击进入该订单的取餐码页

#### Scenario: A user has no order in flight

- **WHEN** 用户没有 `已预约` / `制作中` / `待取餐` 订单
- **THEN** 首页不渲染该提示

### Requirement: WXML tag structure is enforced by a static gate

每个 `.wxml` 文件的标签 MUST 成对闭合。静态门禁 MUST 检出以下三类结构错误，且 MUST 分别报告而非合并为一句「不配平」：

- 栈已空却出现闭合标签（孤立闭合）；
- 闭合标签与栈顶开启标签名不一致（交叉嵌套）；
- 文件结束时仍有未闭合标签。

不匹配的报告 MUST 同时给出**成对另一端的行号** —— 开发者工具只能指到错误暴露的位置，而需要修改的往往是另一端。

检查 MUST 正确处理属性值内的 `>`（如 `wx:if="{{a > 0}}"`），MUST NOT 因此产生假阳性。

标签配对与既有的 `wx:elif` 同级检查 MUST 共用同一份 void 元素清单与同一次遍历，MUST NOT 各自维护，否则同一文件会在两项检查中被解析成不同的树。

#### Scenario: A deleted block leaves an orphan closing tag

- **WHEN** 某个区块被删除但其闭合标签留在原地
- **THEN** 门禁在提交前报出该孤立闭合标签及其行号
- **AND** 不依赖开发者工具在人工打开该页面时才暴露

#### Scenario: A template uses a comparison inside an attribute

- **WHEN** 模板中出现 `wx:if="{{qty > 0}}"` 一类的属性
- **THEN** 该标签被正确识别
- **AND** 不产生任何结构错误报告

#### Scenario: The repository is audited

- **WHEN** 对全部 `.wxml` 运行门禁
- **THEN** 不存在孤立闭合、交叉嵌套或未闭合标签

### Requirement: The order card shows no piece count and aggregates items numerically

用户端订单列表卡片 MUST NOT 展示商品件数，MUST NOT 在页面数据里保留产生它的字段。

对 `items` 的任何数值聚合 MUST 选取数值列 —— 数量、原价或折后价。MUST NOT 选取 id 列或名称列：名称是字符串，累加会得到拼接结果而非数字，且该错误在渲染前不会抛异常。

取餐号徽章 MUST 只展示号码本身并居中，MUST NOT 附加「号」一类的说明标签。

订单卡片上的操作按钮 MUST 单行展示，MUST NOT 因所在行的空间分配而折行或被压缩至文字溢出。

#### Scenario: The order list is rendered

- **WHEN** 用户打开我的订单
- **THEN** 卡片不展示件数
- **AND** 取餐号徽章内只有号码
- **AND** 「取消预约」在一行内完整显示

#### Scenario: Item aggregation is audited

- **WHEN** 检查全部页面中对 `items` 的聚合
- **THEN** 每一处选取的都是数值列
- **AND** 不存在选取名称列的聚合

### Requirement: The pickup point is a single shared constant

一期为单门店单取餐点（§3.1、§5.5）。取餐点 MUST 是一个常量字符串，两端 MUST 引用同一取值，MUST NOT 在任何页面硬编码第二处取餐点文字。

取餐点 MUST NOT 拆成名称与地址两个字段：客户只提供一个字符串，拆分会迫使写入方各自挑一个字段，从而在不同写入路径上装入不同语义的值。

订单生成时 MUST 固化取餐点快照（§7.2），且该快照 MUST 与配置中的取餐点一致 —— 结算写入的值与种子订单携带的值 MUST NOT 是两种东西。

订单详情 MUST 只展示一次取餐地点。

#### Scenario: A user places an order and opens it

- **WHEN** 用户完成结算并打开订单详情
- **THEN** 订单快照里的取餐点与配置中的取餐点一致
- **AND** 取餐地点在页面上只出现一次

#### Scenario: Pickup point data is audited

- **WHEN** 检查两端的取餐点配置与全部订单快照
- **THEN** 只存在一个取餐点取值
- **AND** 不存在名称与地址分离的二元结构
