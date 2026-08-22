## ADDED Requirements

### Requirement: PC uses the same pickup point constant

PC 后台的取餐点 MUST 取自与小程序同值的常量，MUST NOT 各自维护。营业设置页展示的取餐点 MUST 是该常量，MUST NOT 是另一个字段（如取餐窗口描述）。

PC 订单与支付待处理条目的取餐点快照 MUST 与该常量一致。

#### Scenario: The settings page renders the pickup point

- **WHEN** 主账号打开营业设置
- **THEN** 页面展示的取餐点即该常量
- **AND** 与小程序展示的是同一串文字
