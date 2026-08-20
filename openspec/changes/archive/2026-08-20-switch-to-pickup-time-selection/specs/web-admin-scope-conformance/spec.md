## ADDED Requirements

### Requirement: The PC admin configures cutoffs per meal period

营业设置 MUST NOT 保留门店级的单一截单时刻或营业起止时间。设置 MUST 按餐段配置截单时刻、取餐起止时间，并 MUST 提供一个全局的取餐时间粒度。

保存 MUST 校验：粒度大于 0、餐段列表非空、每个餐段的截单与取餐起止均已填写、取餐结束时间不早于开始时间。任一校验不通过 MUST 拒绝保存。

#### Scenario: Settings are loaded and inspected

- **WHEN** 在无 DOM 的运行环境中加载营业设置
- **THEN** 不存在门店级单一截单时刻或营业起止时间
- **AND** 每个餐段都带截单时刻与取餐起止，且存在取餐时间粒度

#### Scenario: An invalid period is saved

- **WHEN** 提交取餐结束早于开始的餐段、非正的粒度或空餐段列表
- **THEN** 保存被拒绝并给出具体原因
- **AND** 既有设置不被修改
