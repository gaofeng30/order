## ADDED Requirements

### Requirement: Order money is stored in cents and settles exactly

订单金额 MUST 以整数分存储，MUST NOT 出现浮点或以元为单位的中间态。涉及金额的字段为 `subtotal`（原价小计）、`discountCut`（折扣减免）、`total`（实付），以及 `items` 行内的原价单价与折后单价。

每笔订单 MUST 同时满足三条恒等式：`subtotal - discountCut === total`；`items` 逐行「原价单价 × 数量」之和 `=== subtotal`；`items` 逐行「折后单价 × 数量」之和 `=== total`。任一不成立即为数据错误。

折扣 MUST 逐商品先舍入到分再乘数量：折后单价 `=== round(原价单价 × discountRate / 100)`。整单先乘后舍入 MUST NOT 使用 —— 两种口径在多行订单上会差到分。

契约层 MUST 导出唯一的分转元展示函数，页面 MUST 经由它渲染金额，MUST NOT 自行做除以 100 的算术。

#### Scenario: Every order is audited for settlement

- **WHEN** 遍历订单数据
- **THEN** 三条恒等式全部成立，且三个金额字段均为整数
- **AND** 员工折扣单的逐行折后单价等于原价单价乘折扣率后舍入到分

#### Scenario: A page renders an amount

- **WHEN** 检查任何渲染订单金额的页面
- **THEN** 金额经由契约层的分转元函数产出
- **AND** 页面源码中不存在手写的除以 100

### Requirement: Every order carries the payment and refund facts

订单记录 MUST 携带对账所需的支付事实：`paidAt`（支付成功时间，精确到秒）、`txnId`（微信交易号，全局唯一）、`discountRate` 与 `isStaff`（身份与折扣率快照）。

取餐信息 MUST 由 `pickupDate`（营业日期）、`pickupTime`（时间点 `hh:mm`）、`mealPeriod`（`lunch` 或 `dinner`）、`pickupPoint` 四项承载，MUST NOT 保留「距取餐还有几分钟」这类派生量 —— 它随时间变化，存下来必然过期。

处于 `退款中` 或 `已退款` 的订单 MUST 携带退款记录，含退款单号、退款金额、退款状态、操作人、退款时间与退款原因；退款状态 MUST 与订单状态一致。其余状态的订单 MUST NOT 携带退款记录。

退款金额 MUST 为正整数分且 MUST NOT 超过订单实付 —— 部分退款是合法情形。

口味与备注 MUST 绑定在 `items` 行内，整单级 MUST 只有 `orderNote`，MUST NOT 存在整单级口味字段。

#### Scenario: A refunded order is audited

- **WHEN** 检查状态为 `退款中` 或 `已退款` 的订单
- **THEN** 它带有完整的退款记录且退款状态与订单状态一致
- **AND** 退款金额为正整数分且不超过订单实付

#### Scenario: A non-refunded order is audited

- **WHEN** 检查其余四个状态的订单
- **THEN** 它不带退款记录

#### Scenario: The order detail is opened

- **WHEN** 主账号在订单管理页选中一笔订单
- **THEN** 详情展示支付时间、取餐日期与时间点、取餐点与微信交易号
- **AND** 员工折扣单额外展示折扣率与减免金额，且原价小计减减免等于实付

### Requirement: Each of the six order states has its own lane

订单管理页 MUST 为六个状态各设一条泳道，外加一条「全部」。`退款中` MUST 自成一格：它是唯一需要人工盯到到账的状态，混在「全部」里等同于没有入口。

每条泳道的计数 MUST 等于该状态的订单数，「全部」的计数 MUST 等于订单总数。

#### Scenario: The lanes are audited

- **WHEN** 检查订单管理页的泳道集合与计数
- **THEN** 六个状态各有一条泳道，另有一条「全部」
- **AND** 每条泳道的计数与该状态的订单数一致
