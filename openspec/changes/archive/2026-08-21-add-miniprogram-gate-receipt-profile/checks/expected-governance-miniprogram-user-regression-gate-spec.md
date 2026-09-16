# miniprogram-user-regression-gate Specification

## Purpose
TBD - created by archiving change enforce-miniprogram-user-regression-gate. Update Purpose after archive.
## Requirements
### Requirement: Mini-program changes declare structured TDD and user regression gates

规则激活后的 OpenSpec candidate 只要 `base_sha...candidate_sha` diff 触及 `apps/wechat-miniprogram/**`，candidate MUST 包含合法的 `openspec/changes/<change>/miniprogram-gates.json`。manifest MUST 绑定 change 与完整 base SHA，声明 Red、Green、Refactor 的 task ID/command，并声明用户主路径、相邻回归、平台原生能力和最低 UI 等级。

每个 TDD task ID MUST 唯一存在于 candidate 的 `tasks.md`。manifest、tasks 或 proposal 的工作树 bytes 与 candidate 不一致时 MUST 拒绝使用移动中的本地文件替代 exact candidate。

#### Scenario: Candidate touches Mini Program without a manifest
- **WHEN** candidate diff 包含 `apps/wechat-miniprogram/**` 且 candidate 内不存在合法 manifest
- **THEN** candidate Gate 返回非零
- **AND** change 不得通过 `CANDIDATE` 推进

#### Scenario: Manifest references incomplete TDD work
- **WHEN** Red、Green 或 Refactor task 缺失、重复、未完成或 command 为空
- **THEN** candidate Gate 返回非零并指明第一个缺失阶段
- **AND** 任务勾选总数、测试数量或自由文本 PASS 不得补偿

### Requirement: User regression minimum is UI2 or UI3 by affected capability

普通小程序模块的用户侧回归 MUST 至少为 UI2。manifest 声明任一平台原生能力，或 candidate diff 直接包含 `wx.login`、`getPhoneNumber`、`wx.requestPayment`、`wx.scanCode`、`wx.requestSubscribeMessage` 时，最低等级 MUST 为 UI3，且 manifest MUST 完整列出检测到的能力。

UI1 Harness MUST 继续作为本地 TDD/用户路径证据，但 MUST NOT 代替 UI2 或 UI3。用户回归 scenarios MUST 至少包含一个 `primary` 和一个 `regression`，每个场景具有稳定 ID、实际步骤和预期结果。

#### Scenario: Ordinary page change declares UI1 only
- **WHEN** 普通小程序 change 的最低用户回归等级低于 UI2
- **THEN** manifest 校验失败
- **AND** Node Harness PASS 不得升级为微信开发者工具或体验版 PASS

#### Scenario: Native capability declares UI2
- **WHEN** manifest 或 candidate diff 表明 change 涉及平台原生能力但最低等级低于 UI3
- **THEN** manifest/candidate Gate 失败
- **AND** change 必须等待指定真机、真实账号和真实结果的 UI3 回归

#### Scenario: Native capability is omitted from manifest
- **WHEN** candidate diff 检测到受控原生 token 而 `native_capabilities` 未声明该能力
- **THEN** candidate Gate 失败
- **AND** verifier 不得把漏报解释为普通 UI2 change

### Requirement: Exact candidate user regression receipt gates higher lifecycle states

小程序 change 从 `CANDIDATE` 进入 `INDEPENDENT_VERIFIED` 前 MUST 提供结构化用户回归 receipt。receipt MUST 绑定同一完整 candidate SHA、达到 manifest 最低 UI 等级、覆盖全部场景且每项/总结果均为 PASS，并记录非敏感环境边界与未验证范围。Harness MUST 保存 receipt 的规范化字段与 SHA256，不把原始外部内容复制进普通日志。

candidate SHA、manifest、场景、receipt 环境或运行结果变化 MUST 使旧 receipt 失效。缺失、低等级、旧 SHA、场景不全、失败或 `BLOCKED_EXTERNAL` MUST 阻断 `INDEPENDENT_VERIFIED` 与 `INTEGRATED`，评分和其他测试不得补偿。

#### Scenario: Candidate has no user regression receipt
- **WHEN** 小程序 candidate 请求进入 `INDEPENDENT_VERIFIED` 但没有当前 receipt
- **THEN** Harness 返回非零并保持原生命周期
- **AND** candidate 可以保留但不得被称为模块完成

#### Scenario: Receipt belongs to another SHA or misses a scenario
- **WHEN** receipt SHA 不等于 candidate 或 scenario ID 集合不等于 manifest 全集
- **THEN** Harness 拒绝该 receipt
- **AND** 不修改原 checkpoint 状态

#### Scenario: Complete exact-SHA receipt passes
- **WHEN** receipt 绑定 exact candidate、达到最低等级并覆盖全部场景 PASS
- **THEN** user-regression Gate 通过并保存 digest/规范化摘要
- **AND** exact-SHA repository verifier 仍必须完成其独立 Gate

### Requirement: Gate activation is forward-only and non-weakening

本规则 MUST 只对 runner base 已包含固定 checker marker 的模块生效。旧 change/candidate 不得被追溯改写为新规则 PASS；main 推进导致旧业务 candidate 需要更新时，新 candidate MUST 使用含 marker 的 base 并接受本 Gate。

实现 MUST 保留现有 lifecycle、OpenSpec strict、owned paths、敏感信息、评分硬阻断、exact-SHA verifier、集成授权和 self-evolution Gate。任何旧 Gate 删除、跳过或降级 MUST 使本控制 change promotion 失败。

#### Scenario: Historical candidate predates marker
- **WHEN** 已存在 candidate 的冻结 base 不包含 checker marker
- **THEN** Harness 不追溯要求新 manifest/receipt
- **AND** 不把该豁免表述为新规则验证通过

#### Scenario: Main advances before an old Mini Program candidate integrates
- **WHEN** 控制规则进入 main 后，旧 candidate 因 moving-main 规则需要重建
- **THEN** replacement candidate 的新 base 包含 marker并适用全部新 Gate
- **AND** 旧测试、receipt 或验证不得复用
