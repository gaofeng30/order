## ADDED Requirements

### Requirement: The PC admin drives orders through the six-state machine only

订单契约的可推进转换 MUST 只有 `制作中 → 待取餐` 与 `待取餐 → 已完成`，MUST NOT 包含 `已预约`。订单泳道 MUST 为 `已预约` / `制作中` / `待取餐` / `已完成` / `已退款` / `全部`，MUST NOT 包含任何已废止状态。

契约层 MUST NOT 导出回退方法，页面 MUST NOT 提供回退动作。

状态语义映射 MUST 只覆盖六态与非订单语义。订单种子 MUST 只使用六态，且 MUST 至少包含一条 `已预约` 订单以填充该泳道。

不可推进订单的行内说明文案 MUST NOT 与状态名拼接出重复的「已」字。

核销 MUST 拒绝已退款订单，并对非 `待取餐` 订单提示尚未备好。

#### Scenario: Contract and lanes are inspected

- **WHEN** 在无 DOM 的运行环境中加载订单契约
- **THEN** 可推进转换只有两条且不含 `已预约`，泳道为六态口径
- **AND** 不存在回退方法

#### Scenario: A reserved order is advanced by the merchant

- **WHEN** 商户尝试推进一条 `已预约` 订单
- **THEN** 请求被拒绝且状态保持 `已预约`

#### Scenario: A terminal order is advanced

- **WHEN** 商户对已处于 `已完成` 的订单再次执行推进
- **THEN** 请求被拒绝且状态不变
