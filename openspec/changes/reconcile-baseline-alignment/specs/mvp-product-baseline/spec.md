## ADDED Requirements

### Requirement: Pickup identifiers and notifications follow the frozen contract

取餐号 MUST 为四位数字并按取餐日期从 `0001` 累计；同一订单重复处理 MUST 返回原号且不得重复占号。手工取餐号核销 MUST 只匹配当前营业日期的 `待取餐` 订单（I14）。二维码 token MUST 在订单进入 `待取餐` 时生成、服务端不透明绑定订单、不设时间过期并在核销后立即失效；扫码与手工核销 MUST 共用同一幂等规则。

一期 MUST 只有待取餐提醒与退款结果两类一次性订阅消息，且均由用户主动订阅（I15）。支付成功页 MUST 请求待取餐提醒，取消确认弹窗 MUST 请求退款结果；拒绝订阅时 MUST 保留首页进行中订单提示条、订单状态与取餐码页补订阅入口。下单成功、支付成功、订单完成、临时停餐或商品变化 MUST 不发送其他订阅消息。

#### Scenario: Cross-day pickup numbers collide

- **WHEN** 当前营业日期与历史营业日期存在相同四位取餐号
- **THEN** 手工核销只匹配当前营业日期的 `待取餐` 订单
- **AND** 历史未取订单不得被误核销

#### Scenario: Redemption is repeated

- **WHEN** 同一二维码或取餐号被重复核销
- **THEN** 服务端返回第一次核销结果
- **AND** 不重复完成订单或统计营收

#### Scenario: User declines a subscription

- **WHEN** 用户拒绝支付成功页的待取餐提醒订阅
- **THEN** 首页进行中订单提示条和订单状态仍提供取餐信息
- **AND** 取餐码页保留再次请求待取餐提醒的入口

### Requirement: Production facts and statistics come from server-confirmed data

身份、商品、价格、销售状态、订单、支付、退款和核销 MUST 以后端为唯一事实源；订单 MUST 固化商品、价格、身份、折扣率、取餐日期/时间和取餐点快照，配置变化不得回算历史订单。mock、客户端内存状态、前端支付结果和 §15 简化状态 MUST 只用于 P0 演示，不得覆盖生产事实（I16）。

统计 MUST 只来自服务端确认的订单、支付、退款和核销数据；`退款中`、`已退款` MUST 按明确口径单独统计且不混入有效营收。营业日结束后仍为 `待取餐` 的订单 MUST 以“未取餐”查询口径单独可查，不得计入已完成订单数或创建新状态。

#### Scenario: Client state conflicts with the server

- **WHEN** mock、客户端内存订单、前端支付动画或本地状态与服务端事实不同
- **THEN** 页面与后续处理采用服务端结果
- **AND** 客户端不得本地创建生产订单、推进状态或确认支付退款成功

#### Scenario: Dashboard calculates effective revenue

- **WHEN** 基础看板统计订单与营收
- **THEN** 只使用服务端确认的数据并单列退款中、已退款与未取餐口径
- **AND** 未取餐不计为已完成，退款中或已退款不混入有效营收
