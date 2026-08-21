## ADDED Requirements

### Requirement: Pending payments are records, not orders

PC 后台 MUST 提供「支付待处理」页，并 MUST 从侧边导航可达。页面承载 §7.3 对账兜底产生的条目：已发起支付、微信查得已支付、但自动补建订单失败的预支付记录。

条目 MUST NOT 是订单：MUST NOT 携带六个订单状态中的任何一个，MUST NOT 携带取餐号。取餐号在订单生成时分配（§7.8），而这些条目正是没能生成订单的那一批；带上状态或取餐号等于把 PRD 已删除的「异常订单」概念重新引入（§7.3 明确不引入 `异常` 状态）。

条目 MUST NOT 出现在订单管理的任何泳道中，MUST NOT 计入泳道计数，「支付待处理」MUST NOT 成为一条泳道。

每条 MUST 携带完整的支付事实：预支付单号、微信交易号、支付时间、支付金额、顾客联系方式、意向取餐信息与商品明细。钱已经收到，缺任一项都会导致既对不上账也退不了款。

未建单原因 MUST 为 §7.3 列举的三种之一：商品已下架、取餐时间已过、数据校验不通过。

页面 MUST 说明条目的来源是后端定时任务、且只有自动补建也失败的才会出现在此，避免主账号把该列表误认为实时的全部支付异常。

#### Scenario: A pending entry is audited

- **WHEN** 检查任一待处理条目
- **THEN** 它带有预支付单号、微信交易号、支付时间、金额、顾客、意向取餐与商品明细
- **AND** 它既没有订单状态也没有取餐号

#### Scenario: The order lanes are audited

- **WHEN** 检查订单列表与泳道计数
- **THEN** 待处理条目不出现在任何泳道中，泳道计数不含它们
- **AND** 泳道集合中没有「支付待处理」

### Requirement: Manual rebuild is refused while the cause still holds

人工建单 MUST 在提交前重新校验阻塞原因，原因仍成立时 MUST 拒绝，并 MUST 在拒绝信息中指出原因与解除办法。原因未解除就建单，等于为一道做不出来的菜分配取餐号，顾客会第二次白跑。

页面 MUST 在打开确认层时就展示当前的阻塞判定，原因仍在时 MUST NOT 提供确认按钮 —— 让主账号在点之前就知道能不能成。

建单成功时 MUST 沿用原支付事实：微信交易号、支付时间与实付金额 MUST 与预支付记录一致。订单 MUST 按 §7.8 分配 4 位取餐号且在同一取餐日期内唯一，MUST 按 §7.4 判定进入 `已预约` 或 `制作中`。

建单成功后条目 MUST 离开待处理列表。重复对同一条目建单 MUST 被拒绝，MUST NOT 生成第二笔订单，也 MUST NOT 重新占用取餐号（§7.8）。

#### Scenario: A rebuild is attempted while the product is still off the menu

- **WHEN** 对因商品已下架而未建单的条目点击人工建单
- **THEN** 确认层展示该原因与解除办法，且不提供确认按钮
- **AND** 没有产生任何订单

#### Scenario: The cause is cleared and the rebuild is retried

- **WHEN** 主账号重新上架该商品后再次建单
- **THEN** 订单生成，交易号、支付时间与实付金额与原记录一致
- **AND** 取餐号为 4 位数字且在该取餐日期内唯一
- **AND** 该条目从待处理列表中消失

#### Scenario: A rebuild is repeated

- **WHEN** 对已建单的条目再次建单
- **THEN** 请求被拒绝
- **AND** 该交易号下仍然只有一笔订单，取餐号不变

### Requirement: Voiding a pending payment refunds in full and reaches the ledger

退款作废 MUST 按原路全额退回（§7.7），契约方法 MUST NOT 接受调用方指定的金额。作废原因 MUST 必填，原因与操作人 MUST 一并记录。

作废 MUST 只把退款推进到 `退款中`，MUST NOT 自行置为 `已退款`（§7.7 只有微信确认才算完成）。

作废 MUST NOT 生成订单。作废后条目 MUST 离开待处理列表。

被作废的条目 MUST 进入财务与对账页的退款台账：顾客确实付过款，这笔退款同样出现在微信账单上。台账条目 MUST 可回溯（以预支付单号与微信交易号定位），并 MUST 与普通退单可区分。

#### Scenario: A pending payment is voided

- **WHEN** 主账号填写原因并确认作废
- **THEN** 退款金额等于已收金额，状态为 `退款中`，操作人为当前登录账号
- **AND** 没有生成任何订单，该条目从待处理列表中消失

#### Scenario: The refund ledger is checked

- **WHEN** 在财务与对账页查看退款记录
- **THEN** 被作废的条目在列，带预支付单号与微信交易号
- **AND** 它与普通退单可以区分

#### Scenario: A void is submitted without a reason

- **WHEN** 作废原因为空或只有空格
- **THEN** 请求被拒绝并说明原因必填

### Requirement: The reconciliation surfaces the received-but-unbuilt gap

对账汇总 MUST 报出区间内「已收款未建单」的笔数与金额。这些款项已经在微信账户中，却不挂在任何订单上 —— 微信账单会比实收合计正好多出这个数。

这两个数 MUST NOT 计入实收合计，净额等式 MUST 保持 `净额 === 实收合计 − 退款合计` 不变；它们 MUST 作为旁注呈现，并 MUST 指向「支付待处理」页。

字段命名 MUST 与「未到账退款笔数」区分开：同名不同义在对账页上会直接导致误读。

页面 MUST 说明该差额的成因与消解方式：条目处理完毕后差额归零。

#### Scenario: The summary is read with unresolved entries

- **WHEN** 区间内存在已收款未建单的条目
- **THEN** 汇总报出其笔数与金额，并指向支付待处理页
- **AND** 实收合计不含它们，净额等式仍然成立

#### Scenario: All entries are resolved

- **WHEN** 区间内的条目全部建单或作废完毕
- **THEN** 已收款未建单的笔数与金额归零
- **AND** 建单的金额进入实收合计，作废的金额进入退款合计
