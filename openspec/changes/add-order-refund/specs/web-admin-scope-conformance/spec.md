## MODIFIED Requirements

### Requirement: The reconciliation summary nets by amount

对账汇总 MUST 给出实收合计、退款合计、净额三项，且 MUST 满足 `净额 === 实收合计 − 退款合计`。三项 MUST 以整数分计算后再格式化，MUST NOT 以元为单位求和。

退款 MUST 按**金额**扣减。一期只有全额退款（§7.7），因此每笔退款金额恒等于原订单实付；但汇总 MUST NOT 因此改为按订单笔数扣减 —— 跨区间的退款（原订单支付于区间外）在区间内只有退款没有收款，按笔数扣无从扣起。

汇总 MUST 标出未到账的退款笔数（状态为 `退款中` 者），它们是净额已扣但钱尚未退出的部分。

金额为负时符号 MUST 在货币符号之前，MUST NOT 出现 `¥-12.00` 这种排版。

#### Scenario: A refund settled outside its payment range is netted

- **WHEN** 区间内含一笔退款，其原订单支付于区间之外
- **THEN** 净额扣减该退款金额，且实收合计不含该订单
- **AND** 汇总的三项满足净额等式

#### Scenario: The net goes negative

- **WHEN** 某日只有退款而无收款
- **THEN** 净额展示为负值且负号在货币符号之前

### Requirement: The refund record is traceable and complete

每条退款记录 MUST 携带退款单号、退款金额、退款状态与操作人（§6.11），并 MUST 携带可回溯到原订单的信息：订单号与微信交易号。缺少回溯信息的退款记录在对账时无法定位原单，MUST NOT 出现。

退款金额 MUST 恒等于原订单实付（§7.7 一期只支持原路全额退款）。契约层 MUST NOT 导出「是否部分退款」这类标记，页面 MUST NOT 渲染它 —— 一个恒为假的分支只会让人以为部分退款是可能的。

操作人 MUST 取自当前登录的商户账号，MUST NOT 硬编。

#### Scenario: A refund record is audited

- **WHEN** 检查任一退款记录
- **THEN** 它带有退款单号、金额、状态与操作人，且金额等于原订单实付
- **AND** 它带有原订单号与微信交易号

## ADDED Requirements

### Requirement: The owner initiates a full refund from the order

订单管理页 MUST 提供发起退款入口。退款 MUST 只能从 `已预约`、`制作中`、`待取餐`、`已完成` 四个状态发起（§7.1 旁路），从 `退款中` 或 `已退款` 发起 MUST 被拒绝且 MUST NOT 改动既有退款记录。

契约方法 MUST NOT 接受调用方指定的退款金额：一期只支持原路全额退款（§7.7），表达不出来的请求不需要校验，也不会被后续改动悄悄放开。

退款原因 MUST 必填，空白或纯空格 MUST 被拒绝。原因与操作人 MUST 一并记入退款记录，供财务对账追责。

发起退款 MUST 只把订单推进到 `退款中`。只有微信确认退款成功才是 `已退款`（§7.7），因此 PC 后台 MUST NOT 自行将订单置为 `已退款`。

重复对同一订单发起退款 MUST 被拒绝，且 MUST NOT 产生第二条退款记录、MUST NOT 改变已生成的退款单号（§7.1 幂等）。

退款 MUST 有二次确认，确认层 MUST 展示将退金额、操作人与不可撤销的后果。订单管理页 MUST NOT 提供任何撤销或回退已完成转换的入口（§7.1）。

#### Scenario: A refund is initiated from a live order

- **WHEN** 主账号对处于四个可退状态之一的订单填写原因并确认退款
- **THEN** 订单进入 `退款中`，退款记录含全额金额、原因、当前登录账号与时间戳
- **AND** 订单不会被直接置为 `已退款`

#### Scenario: A refund is attempted twice

- **WHEN** 对同一订单再次发起退款
- **THEN** 请求被拒绝并说明当前状态
- **AND** 该订单仍然只有一条退款记录，退款单号不变

#### Scenario: A refund is submitted without a reason

- **WHEN** 退款原因为空或只有空格
- **THEN** 请求被拒绝并说明原因必填

### Requirement: Uncollected is a query, not a seventh state

「未取餐」MUST 作为查询口径提供：营业日已结束仍处于 `待取餐` 的订单（§6.7）。它 MUST NOT 进入状态集合，MUST NOT 成为第七条泳道。

该筛选 MUST 只在 `待取餐` 泳道下可用，切换到其他泳道时 MUST 自动解除 —— 它是挂在 `待取餐` 上的条件，带到别处会把列表筛成空的。

营业日 MUST 由契约层单一下发，页面 MUST NOT 各自硬编日期。

#### Scenario: The uncollected filter is applied

- **WHEN** 在 `待取餐` 泳道启用「未取餐」
- **THEN** 列表只剩营业日期早于当前营业日的 `待取餐` 订单
- **AND** 当日的 `待取餐` 订单被排除

#### Scenario: The state set is audited

- **WHEN** 检查泳道集合与状态集合
- **THEN** 二者均不包含「未取餐」

#### Scenario: The lane changes while the filter is on

- **WHEN** 启用「未取餐」后切换到其他泳道，或退款后自动跳转到 `退款中`
- **THEN** 筛选自动解除，目标泳道展示其完整列表
