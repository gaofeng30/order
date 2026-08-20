## ADDED Requirements

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
