## ADDED Requirements

### Requirement: Pickup time is chosen once and shared across the ordering flow

可预约营业日期 MUST 只有今天与明天。每个餐段 MUST 有一个固定截单时刻，餐段内全部取餐时间共用；取餐时间 MUST 为由餐段范围与粒度推导的离散时间点，粒度 MUST 可配置而非写死。

取餐时间 MUST 在菜单顶部条选定并跨页共享；结算页 MUST 只读展示该选择，MUST NOT 再提供第二套日期或时间选择器。

默认取餐时间 MUST 为当前时刻之后第一个未截单的时间点；当日全部餐段截单时 MUST 落到下一个可选日期。

#### Scenario: Menu opens with a usable default

- **WHEN** 用户进入菜单
- **THEN** 顶部条展示当前时刻之后第一个未截单的取餐时间
- **AND** 点击该条展开取餐时间选择弹层

#### Scenario: Checkout reuses the chosen time

- **WHEN** 用户在菜单顶部条选定取餐时间后进入结算
- **THEN** 结算页只读展示该取餐时间并提供回菜单修改的入口
- **AND** 结算页不含日期或时间选择控件

#### Scenario: Times are derived from range and step

- **WHEN** 由餐段的取餐起止与粒度推导时间点
- **THEN** 结果为该范围内按粒度均分的离散时刻
- **AND** 粒度变化时时间点随之变化

### Requirement: A cut-off meal period is folded, not itemised

取餐时间选择弹层 MUST 按餐段分组。已截单餐段 MUST 整组折叠并标注其截止时刻，MUST NOT 逐条渲染不可选时间点。当日全部餐段截单时，该日期 MUST 标注为已截单。

提交订单时 MUST 重新校验目标取餐时间所属餐段是否仍在截单前；已截单 MUST 拦截提交、提示重选，且 MUST 保留购物车内容。

#### Scenario: A period is past its cutoff

- **WHEN** 当前时刻已过某餐段的固定截单时刻且所选日期为今天
- **THEN** 该餐段在弹层中整组折叠并标注截止时刻
- **AND** 该组不渲染任何可选时间点

#### Scenario: Submission targets a cut-off period

- **WHEN** 用户提交订单而所选取餐时间所属餐段已截单
- **THEN** 提交被拦截且不生成订单
- **AND** 购物车内容保留

### Requirement: Cancel eligibility depends only on state and remaining time

自助取消资格 MUST 只由订单状态与距取餐分钟数决定：`已预约` 且距取餐大于 30 分钟。判定 MUST NOT 读取任何随即时单删除的订单类型字段。

#### Scenario: Cancel eligibility is evaluated

- **WHEN** 判定一条 `已预约` 且距取餐 102 分钟的订单
- **THEN** 允许自助取消
- **AND** 同样状态但距取餐 18 分钟、或状态为 `制作中` 的订单不允许
