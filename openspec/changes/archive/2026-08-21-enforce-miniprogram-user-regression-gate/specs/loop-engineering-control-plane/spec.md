## ADDED Requirements

### Requirement: Control plane blocks Mini Program promotion on missing gate evidence

控制面 MUST 在小程序 change 的 `IMPLEMENTING → CANDIDATE` 转换校验结构化 manifest、candidate diff 和完整 TDD tasks，并在 `CANDIDATE → INDEPENDENT_VERIFIED` 转换校验绑定同一 SHA 的用户回归 receipt。任一校验失败 MUST 原子拒绝状态变更；`BLOCKED_EXTERNAL` MUST 保留既有 blocker 记录、在 task 证据中保留 owner/恢复条件并阻断更高状态。

#### Scenario: Candidate transition lacks complete TDD gates
- **WHEN** 小程序 change 请求进入 `CANDIDATE` 且 manifest/TDD task/candidate 绑定任一不满足
- **THEN** checkpoint 返回非零且原状态不变
- **AND** evidence 文本或手工 task 总完成数不能绕过

#### Scenario: Verification transition lacks exact user regression
- **WHEN** 小程序 candidate 请求进入 `INDEPENDENT_VERIFIED` 但 receipt 缺失、过期、低等级或不完整
- **THEN** checkpoint 返回非零且保持 `CANDIDATE`
- **AND** integration handler 不能继续

#### Scenario: Complete gate evidence advances normally
- **WHEN** TDD candidate Gate、exact-SHA user receipt、repository verifier 和既有全部 Gate 当前 PASS
- **THEN** 控制面可按原七态相邻推进
- **AND** 本规则不新增生命周期或绕过集成授权
