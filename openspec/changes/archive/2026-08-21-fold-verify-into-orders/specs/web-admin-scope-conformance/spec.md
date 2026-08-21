## ADDED Requirements

### Requirement: The PC admin has no standalone verify screen

PC 后台 MUST NOT 提供独立的扫码核销页（§15.5.3）。扫码在手机上进行（小程序商户端，评审 §23），PC 侧不需要一个模拟扫码枪的输入页。

PC 页面集合 MUST 与 PRD §3.5 的页面清单逐项一致，页数 MUST 相等。多一页少一页都意味着两边有一侧未同步。

删除 MUST 是能力上的删除：页面文件、导航路由、脚本挂载与该页专属样式 MUST 全部移除，MUST NOT 留下无入口的孤儿页或无选择器命中的死样式。

#### Scenario: The page set is audited

- **WHEN** 检查侧边导航路由、`pages/` 目录与 `index.html` 的脚本挂载
- **THEN** 三处均不存在独立核销页
- **AND** 路由集合与 PRD §3.5 的页面清单逐项一致且页数相等

### Requirement: Manual verify is located through order search

核销 MUST 仍是 `待取餐 → 已完成` 的唯一路径（§6.6），其入口 MUST 保留在订单详情。独立核销页删除后，定位待核销订单的能力 MUST 由订单管理页的搜索承担，MUST NOT 因删页而丢失。

订单管理 MUST 提供按取餐号、订单号、手机号的搜索（§6.6 末条、§6.7）。搜索 MUST 跨泳道：发起核销时并不知道该单当前处于哪个状态。

搜索 MUST NOT 成为一条泳道，MUST NOT 进入订单状态集合。

搜索与泳道 MUST NOT 同时决定列表内容：选择泳道 MUST 退出搜索态，否则使用者无法判断当前看到的是哪一种结果集。

#### Scenario: An order is located for verification

- **WHEN** 主账号在订单管理搜索取餐号、订单号或手机号
- **THEN** 匹配的订单出现在列表中，且结果可跨多个状态
- **AND** 选中后详情仍提供核销动作

#### Scenario: The lane is clicked while a search is active

- **WHEN** 处于搜索态时点击任一泳道
- **THEN** 搜索被清空，列表回到该泳道的完整结果

### Requirement: A pickup code only resolves within the current business day

按取餐号搜索 MUST 只匹配当前营业日的订单（§6.6）。取餐号按取餐日期从 `0001` 累计，跨营业日可能重复（§7.8）—— 不加限定时一个 4 位数字会同时命中多天的订单，凭它核销就会核错单。

订单号与手机号 MUST NOT 受该限制：订单号全局唯一，手机号不存在跨日歧义。4 位数字 MUST 同时按取餐号与手机号片段匹配，但只有取餐号那一半受营业日限制。

当某取餐号在当前营业日无匹配、却存在于其他营业日时，页面 MUST 报出该事实并指出可用的定位方式（订单号 / 手机号搜索，或「未取餐」筛选）。空结果 MUST NOT 被呈现为「不存在该订单」。

#### Scenario: A code from an earlier business day is entered

- **WHEN** 输入一个只存在于更早营业日的取餐号
- **THEN** 列表不返回该订单
- **AND** 页面提示该取餐号在哪个营业日存在，并给出可用的定位方式

#### Scenario: The same order is located by its order number

- **WHEN** 改用订单号搜索同一笔跨营业日订单
- **THEN** 该订单被找到

### Requirement: Verification refuses refunded orders and never double-counts

核销 MUST 拒绝处于 `退款中` 或 `已退款` 的订单（§6.6 已退款订单不得核销）。该校验 MUST 由契约层承担，MUST NOT 依附于任何单一页面 —— 否则删页即删校验。

重复核销 MUST NOT 改变订单状态、MUST NOT 重复计入营收或订单量（§6.6 幂等）。

#### Scenario: A refunded order is verified

- **WHEN** 对 `退款中` 或 `已退款` 的订单发起核销
- **THEN** 请求被拒绝且订单状态不变

#### Scenario: Verification is repeated

- **WHEN** 对已核销的订单再次核销
- **THEN** 请求被拒绝且订单保持 `已完成`
