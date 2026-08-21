## ADDED Requirements

### Requirement: Shelf state and daily sell-out are two independent dimensions

商品的 `status` MUST 只有 `'on'` 与 `'off'` 两个取值，表示长期上下架。当日售罄 MUST NOT 存在商品对象上。

当日售罄 MUST 存为按取餐日期的独立记录，唯一键为 `(productId, serviceDate)`。记录的**存在**即表示该商品在该取餐日期售罄；MUST NOT 用布尔字段表达，否则「昨天标过又取消」与「今天还没标过」会成为两种形态表示同一件事。

可售性 MUST 由两个维度现算：上架且该取餐日期无售罄记录。MUST NOT 引入第三个综合状态枚举。

#### Scenario: The product model is audited

- **WHEN** 检查商品种子与全部读写点
- **THEN** `status` 只出现 `'on'` 与 `'off'`
- **AND** 商品对象上不存在任何售罄字段

#### Scenario: Yesterday's sell-out does not survive

- **WHEN** 某商品存在昨日的售罄记录而无今日记录
- **THEN** 它在今日可售
- **AND** 无需任何手工恢复动作

### Requirement: The merchant sell-out toggle is scoped to the business day

小程序商户端菜品页的售罄开关 MUST 写入或删除当前营业日的售罄记录，MUST NOT 修改 `status`。

在营业日 D 标记售罄 MUST NOT 影响 D+1 的可售性 —— 一期只有预约单，屏蔽次日预约会直接误伤主场景。

菜品页 MUST 仍然只提供售罄切换，MUST NOT 提供上下架、编辑或新增（§6.5、§15.5.2）。

#### Scenario: A merchant sells out at the window

- **WHEN** 商户在营业日 D 把某商品标为售罄
- **THEN** 该商品在 D 日不可售
- **AND** 该商品在 D+1 仍可售
- **AND** 该商品的 `status` 未被改动

#### Scenario: A merchant restocks in the evening

- **WHEN** 商户把已售罄的商品切回可售
- **THEN** 当日售罄记录被移除
- **AND** 该商品当日重新可售
