## ADDED Requirements

### Requirement: Mini Program TDD and user regression are separate hard gates

质量协议 MUST 将小程序 TDD 与用户侧回归记录为两个不可互相替代的硬 Gate。TDD MUST 在形成 candidate 前提供真实 Red、Green、Refactor；用户侧回归 MUST 在 exact candidate 上按影响面达到 UI2 或 UI3。UI1、本地单元/契约测试、覆盖率、C/T/V/R 或另一个模块 PASS MUST NOT 代替用户侧回归。

#### Scenario: TDD passes but user regression is absent
- **WHEN** 小程序 change 已完成 Red/Green/Refactor，但 exact candidate 没有达到目标等级的用户回归
- **THEN** 质量结论是 `BLOCKED_EXTERNAL` 或 FAIL，而不是模块完成
- **AND** change 不得进入 `INDEPENDENT_VERIFIED` 或 `INTEGRATED`

#### Scenario: User regression passes without TDD
- **WHEN** UI2/UI3 用户路径通过但 candidate 缺真实 Red、Green 或 Refactor
- **THEN** writer Gate 失败
- **AND** 外部运行 PASS 不得补偿缺失的开发证据
