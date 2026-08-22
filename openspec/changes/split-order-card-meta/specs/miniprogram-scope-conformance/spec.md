## ADDED Requirements

### Requirement: Pickup time and pickup place are separate lines on the order card

用户端订单卡片 MUST 把**取餐时间**与**取餐地点**作为两条独立展示项，各占一行、各带图标。MUST NOT 拼接进同一个字段或同一行。

任一字段 MUST NOT 包含另一条信息：取餐时间的文案里 MUST NOT 出现取餐地点，取餐地点的文案里 MUST NOT 出现时刻。

信息行的图标 MUST 对齐首行文字，MUST NOT 垂直居中于整块 —— 取餐点是客户可配值，任何一行都可能因为配置变长而折行，此时居中会让图标与两行都不对齐。

取餐地点 MUST 取自订单自身的 `pickupPoint` 快照。

#### Scenario: An order with a long pickup place is rendered

- **WHEN** 取餐地点较长
- **THEN** 取餐时间与取餐地点分别显示在各自的行上
- **AND** 图标与该行首行文字对齐

#### Scenario: The card fields are audited

- **WHEN** 检查订单卡片的页面数据
- **THEN** 取餐时间与取餐地点是两个字段
- **AND** 任一字段都不包含另一条信息
