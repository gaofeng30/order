## MODIFIED Requirements

### Requirement: Orders use one six-state production state machine

生产订单状态 MUST 且只能为 `已预约`、`制作中`、`待取餐`、`已完成`、`退款中`、`已退款`。一期 MUST NOT 存在 `待支付`、`已支付待接单`、`已取消` 或任何异常状态。

主链路 MUST 为：微信确认支付成功后进入 `已预约`，取餐时间前 30 分钟由服务端自动进入 `制作中`，商户标记备好进入 `待取餐`，核销成功进入 `已完成`。支付成功时若距取餐时间已不足 30 分钟，订单 MUST 在创建时直接进入 `制作中`。

`已预约 → 制作中` MUST 由服务端定时任务驱动。**客户端 MUST NOT 提供该转换**：商户端可执行的转换只有 `制作中 → 待取餐` 与 `待取餐 → 已完成`。

一期 MUST NOT 提供接单动作，也 MUST NOT 提供商户手动提前开做。排产定时任务 MUST 幂等并具备重试与补偿，任务漏跑 MUST NOT 导致订单卡在 `已预约`。

一期 MUST NOT 设置待取超时状态。备好后订单 MUST 保持 `待取餐` 直至核销完成或发起退款；营业日结束后仍未核销的订单 MUST 通过查询口径可筛选，且该口径 MUST NOT 是订单状态。

每次状态转换 MUST 由服务端校验前置状态、资源权限和幂等键并记录审计。相同幂等键的重复请求 MUST 返回第一次最终结果。生产 MUST NOT 提供撤销或回退已完成转换的入口，**包括 Toast 上的回退动作与任何回退契约方法**。

#### Scenario: Merchant fulfills a paid order

- **WHEN** 订单在取餐时间前 30 分钟自动进入 `制作中` 且商户标记备好
- **THEN** 订单进入 `待取餐`
- **AND** 核销成功后进入 `已完成`

#### Scenario: Payment succeeds inside the 30-minute window

- **WHEN** 支付成功时距取餐时间已不足 30 分钟
- **THEN** 订单创建即为 `制作中`
- **AND** 用户端不提供自助取消入口

#### Scenario: Client attempts to start production early

- **WHEN** 任一客户端尝试把 `已预约` 订单推进到 `制作中`
- **THEN** 请求被拒绝且订单状态不变
- **AND** 客户端的可推进转换表中不含 `已预约`

#### Scenario: A transition is repeated

- **WHEN** 同一幂等键的状态推进请求被重复提交
- **THEN** 服务端返回第一次最终结果
- **AND** 不重复产生支付、退款、营收或核销副作用

#### Scenario: An operator attempts undo

- **WHEN** 任一角色尝试撤销或回退已完成的状态转换
- **THEN** 服务端拒绝该请求
- **AND** 客户端不提供任何生产撤销入口或回退契约方法

#### Scenario: Order is never collected

- **WHEN** 营业日结束时订单仍处于 `待取餐`
- **THEN** 订单保持该状态且不自动流转
- **AND** 该订单可通过未取餐查询口径筛出，主账号可对其退款或事后核销
