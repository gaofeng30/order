## ADDED Requirements

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
