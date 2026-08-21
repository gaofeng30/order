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
