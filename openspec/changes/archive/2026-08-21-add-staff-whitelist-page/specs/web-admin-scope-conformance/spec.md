## ADDED Requirements

### Requirement: The PC admin maintains the staff discount whitelist

PC 后台 MUST 提供员工折扣白名单页，并 MUST 从侧边导航可达。

白名单记录 MUST 只有两个可填字段：手机号（唯一识别键）与姓名。记录 MUST 另带四类非填写字段：状态（新增时默认启用，由列表行内开关切换）、加入时间（自动）、微信绑定标记与累计消费、累计单量（只读）。记录 MUST NOT 保存单位、部门、工号或备注。

保存 MUST 校验：手机号非空且为合法手机号格式；姓名非空；手机号在名单内唯一。任一不通过 MUST 拒绝保存且不修改既有记录。

编辑既有记录 MUST 只覆盖手机号与姓名，MUST NOT 重置加入时间、微信绑定标记、累计消费或累计单量。切换状态 MUST 同样不影响这些字段。

停用 MUST 保留记录，删除 MUST 需二次确认并在确认文案中说明停用与删除的区别。

#### Scenario: A whitelist entry is created

- **WHEN** 主账号提交合法的手机号与姓名
- **THEN** 记录被创建且状态为启用、微信绑定为否、累计消费与单量为零
- **AND** 加入时间由系统写入

#### Scenario: An invalid or duplicate entry is submitted

- **WHEN** 提交缺失手机号、格式错误的手机号、缺失姓名，或名单内已存在的手机号
- **THEN** 保存被拒绝并给出具体原因
- **AND** 既有记录不被修改

#### Scenario: An existing entry is edited or toggled

- **WHEN** 编辑既有记录的姓名，或切换其启用状态
- **THEN** 加入时间、微信绑定标记、累计消费与累计单量保持不变

### Requirement: The global discount rate is maintained with the whitelist

全局折扣率 MUST 在员工折扣白名单页维护，取值 MUST 为 1 到 100 的整数百分比，表示员工实付比例，100 表示无折扣。

保存 MUST 校验整数与取值范围，非整数或越界 MUST 拒绝。页面 MUST 说明该折扣对所有命中名单的用户与所有商品统一生效、逐商品先舍入到分再乘数量、且修改只影响新报价不回算历史订单。

#### Scenario: The rate is saved

- **WHEN** 主账号提交 1 到 100 之间的整数
- **THEN** 折扣率被保存并给出确认反馈

#### Scenario: An invalid rate is submitted

- **WHEN** 提交 0、大于 100 的值或小数
- **THEN** 保存被拒绝并给出具体原因
