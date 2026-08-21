## ADDED Requirements

### Requirement: The merchant order list is searchable across lanes

小程序商户端订单页 MUST 提供可输入的搜索框，按**取餐号、订单号、手机号**定位订单（联系人姓名作为同一输入框的附加匹配项）。搜索 MUST 跨全部状态泳道，MUST NOT 被当前选中的泳道限制。

搜索 MUST 是一个查询：`kw` MUST NOT 写入 `globalData`，MUST NOT 成为订单模型上的字段或状态。选择任一泳道 MUST 退出搜索态。

搜索框 MUST 是受控输入元素并绑定输入事件；MUST NOT 以静态文本冒充已交付的搜索能力。

#### Scenario: A customer reports a pickup code at the window

- **WHEN** 商户在订单页输入当前营业日的 4 位取餐号
- **THEN** 该订单出现在结果中，无论它处于哪个状态泳道
- **AND** 当前泳道的选择不影响该结果

#### Scenario: A merchant searches by order number or phone

- **WHEN** 商户输入订单号片段或手机号片段
- **THEN** 匹配的订单出现在结果中
- **AND** 结果可以跨越多个状态

#### Scenario: The merchant leaves search

- **WHEN** 商户点击任一状态泳道
- **THEN** 搜索关键词被清空
- **AND** 列表回到该泳道的完整内容

### Requirement: Pickup codes are four digits accumulated per pickup date

取餐号 MUST 为 4 位数字，MUST NOT 携带 `A` / `B` 等即时单遗留前缀。用户端与商户端展示的取餐号 MUST 使用同一格式。

取餐号 MUST 在同一取餐日期内唯一，MAY 在不同取餐日期之间重复。种子数据 MUST 包含至少一组跨营业日重复的取餐号，否则「限定当前营业日」这条约束不可证伪。

营业日 MUST 来自单一常量，MUST NOT 由运行时系统时钟推导 —— 否则同一份种子数据的断言结果会随日期翻转。

#### Scenario: Pickup code format is audited

- **WHEN** 检查用户端与商户端全部订单种子的取餐号
- **THEN** 每一个都匹配 4 位数字
- **AND** 不存在字母前缀

#### Scenario: The same code exists on two business days

- **WHEN** 检查种子数据
- **THEN** 存在两张取餐号相同但取餐日期不同的订单
- **AND** 其中一张属于当前营业日

### Requirement: A pickup code resolves only within the current business day

按取餐号定位订单 MUST 限定在当前营业日。搜索与手工核销 MUST 共用同一份解析实现，MUST NOT 各自实现该规则。

4 位纯数字输入 MUST 同时按手机号片段匹配：跨营业日歧义只是取餐号的属性，手机尾号没有该问题。

某取餐号在当前营业日无果、却存在于其他营业日时，系统 MUST 报出该事实并指出替代定位方式，MUST NOT 只返回空列表或「无效取餐号」。

#### Scenario: A stale pickup code is entered for verification

- **WHEN** 商户手工输入一个属于其他营业日的取餐号
- **THEN** 核销被拒绝
- **AND** 提示指出该号属于哪个营业日以及可改用订单号或手机号定位

#### Scenario: A four-digit phone fragment is searched

- **WHEN** 商户输入手机号的 4 位尾段
- **THEN** 持有该手机号的订单出现在结果中
- **AND** 结果不因该输入被当作取餐号而落空

#### Scenario: An order under refund is located

- **WHEN** 商户按手机号或订单号搜索一张处于 `退款中` 的订单
- **THEN** 该订单出现在结果中并显示其状态
- **AND** 商户端不因缺少 `退款中` 泳道而无法找到它
