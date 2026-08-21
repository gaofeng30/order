## ADDED Requirements

### Requirement: PC order lines carry the same product name snapshot

PC 后台的订单项 MUST 使用与小程序相同的 §15.6.2 元组 `[id, name, qty, price, discountedPrice, flavors?, note?]`。两端 MUST NOT 各自定义订单项形状。

PC 的订单摘要 MUST NOT 按 id 回查商品名称，因此 MUST NOT 保留「商品已删除」一类的回查兜底 —— 该兜底是回查方案的补丁，回查移除后它不再有触发条件，留着会误述当前行为。

§15.6.2 的三条结算恒等式在新元组下 MUST 依然成立：逐行 `qty × price` 之和等于 `subtotal`；逐行 `qty × discountedPrice` 之和等于 `total`；`subtotal - discountCut` 等于 `total`。

#### Scenario: PC order lines are audited

- **WHEN** 检查 PC 订单种子的每一行订单项
- **THEN** 形状与小程序一致，第二项为非空字符串名称
- **AND** 三条结算恒等式逐单成立

#### Scenario: An order summary is rendered

- **WHEN** PC 渲染订单摘要或订单详情
- **THEN** 名称取自订单自身
- **AND** 代码中不存在按 id 取名称的回查或其兜底分支
