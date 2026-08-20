## MODIFIED Requirements

### Requirement: The baseline is traceable and has no behavioral TODO

实施后的 PRD §1–§14 MUST 提供需求—页面—状态—角色—外部依赖追踪矩阵，覆盖本 spec 全部 requirements，并为每条规则指向明确页面、状态、责任角色、外部 Gate 与验收方法；不适用维度 MUST 显式写为“不适用”。

PRD 正式基线 MUST 不存在会改变产品行为、公共契约、数据结果、授权边界或验收方式的 TODO、待定项或 A/B 方案。尚未提供的真实经营值只能列为 UAT 前配置，不得形成第二套行为模型。

追踪矩阵引用的状态集合 MUST 为六态，角色集合 MUST 为主账号与子账号两角色；矩阵 MUST NOT 引用任何已被客户评审删除的概念作为待覆盖维度。

#### Scenario: Traceability matrix is checked

- **WHEN** reviewer 逐条对照本 spec 与 PRD 追踪矩阵
- **THEN** 每条 requirement 至少有一个 PRD 目标位置和验收方法
- **AND** 页面、六态、两角色及 12 个外部 Gate 均无孤立项

#### Scenario: Matrix cites a retired dimension

- **WHEN** 追踪矩阵把九态、四角色、数量库存、软预占、固定取餐时段或优惠券列为待覆盖维度
- **THEN** PRD 实施验收失败
- **AND** 该维度必须先按当前生效 spec 重新表述

#### Scenario: Behavioral ambiguity remains

- **WHEN** PRD 中仍存在“待确认”“可选方案”“A/B”“尽快取餐”或与本 spec 冲突的正式研发规则
- **THEN** PRD 实施验收失败
- **AND** change 不得产生候选 SHA
