## ADDED Requirements

### Requirement: The PC admin reconciles payments against the wechat bill

PC 后台 MUST 提供财务与对账页，并 MUST 从侧边导航可达。页面 MUST 支持按日期区间筛选，起始日期晚于结束日期 MUST 被拒绝且不改变当前区间。

收款 MUST 按**支付日期**归集，MUST NOT 按营业日期归集：微信商户平台的交易账单以交易时间为准，而预约单可以今天支付、次日取餐，两者按不同日期落账则必然对不上。每笔收款 MUST 同时展示支付时间与营业日期，避免管理员自行推断。

退款 MUST 按**退款到账日期**归集，MUST NOT 跟随原订单的支付日期：跨日退款在微信账单里出现在到账当天。页面 MUST 说明「今天的退款里可能含前几天的订单」。

收款明细 MUST 逐笔展示订单号、支付时间、微信交易号与实付金额（§6.11）。

#### Scenario: A reservation paid the day before is grouped

- **WHEN** 一笔订单支付于某日、取餐营业日期为次日，按支付当日筛选
- **THEN** 该笔出现在支付当日的收款明细中
- **AND** 按营业日期当日筛选时，该笔不出现在收款明细中

#### Scenario: A refund settles on a later day

- **WHEN** 一笔退款的到账日期晚于原订单的支付日期
- **THEN** 该退款出现在到账当日的退款记录中
- **AND** 不出现在原订单支付当日的退款记录中

#### Scenario: An invalid date range is submitted

- **WHEN** 起始日期晚于结束日期
- **THEN** 变更被拒绝并给出原因
- **AND** 页面保持原有区间与数据

### Requirement: The reconciliation summary nets by amount

对账汇总 MUST 给出实收合计、退款合计、净额三项，且 MUST 满足 `净额 === 实收合计 − 退款合计`。三项 MUST 以整数分计算后再格式化，MUST NOT 以元为单位求和。

退款 MUST 按**金额**扣减，MUST NOT 按订单笔数扣减：部分退款是合法情形，按笔数扣会扣掉整单实付。

汇总 MUST 标出未到账的退款笔数（状态为 `退款中` 者），它们是净额已扣但钱尚未退出的部分。

金额为负时符号 MUST 在货币符号之前，MUST NOT 出现 `¥-12.00` 这种排版。

#### Scenario: A partial refund is netted

- **WHEN** 区间内含一笔退款金额小于订单实付的部分退款
- **THEN** 净额只扣减实际退回的金额
- **AND** 汇总的三项满足净额等式

#### Scenario: The net goes negative

- **WHEN** 某日只有退款而无收款
- **THEN** 净额展示为负值且负号在货币符号之前

### Requirement: The refund record is traceable and complete

每条退款记录 MUST 携带退款单号、退款金额、退款状态与操作人（§6.11），并 MUST 携带可回溯到原订单的信息：订单号与微信交易号。缺少回溯信息的退款记录在对账时无法定位原单，MUST NOT 出现。

部分退款 MUST 在记录上标明，并同时展示原订单实付金额。

#### Scenario: A refund record is audited

- **WHEN** 检查任一退款记录
- **THEN** 它带有退款单号、金额、状态与操作人
- **AND** 它带有原订单号与微信交易号

### Requirement: The payment detail export survives Excel

导出 MUST 产出可直接用 Excel 打开的明细文件，且 MUST 防住两处会让文件变得不可用的陷阱：

- 文件 MUST 带 UTF-8 BOM。缺少 BOM 时 Excel 会把中文识别为乱码。
- 微信交易号与退款单号是长数字串，MUST 以文本形式导出。直接写入数字 Excel 会转成科学计数法，导致交易号无法与账单比对。

导出 MUST 含表头，且 MUST 每笔收款一行。金额 MUST 以元导出，MUST NOT 直接导出分。

#### Scenario: The export is opened in Excel

- **WHEN** 导出某日期区间的收款明细
- **THEN** 文件以 UTF-8 BOM 开头，含表头且行数等于收款笔数加一
- **AND** 微信交易号以原样长数字串出现，未被转为科学计数法

### Requirement: The page states what it does not reconcile

页面 MUST 明写「自动拉取并比对微信账单一期未实现」，并说明本页数字只汇总本系统内的订单、不代表已与微信核平。缺少该声明时管理员会把汇总数字当作系统核过的结果。

该缺口 MUST 同时记录在 PRD 的前端与后端缺口清单中。

#### Scenario: The page is audited for its own limits

- **WHEN** 检查财务与对账页
- **THEN** 页面写明自动核对未实现，并给出人工核对的口径
- **AND** PRD 的缺口清单中有对应条目
