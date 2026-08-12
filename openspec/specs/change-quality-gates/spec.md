# change-quality-gates Specification

## Purpose

定义单个 OpenSpec change 如何按最高风险选择最低质量证据、形成 candidate、触发 exact-SHA 验证并满足集成 Gate。

## Requirements
### Requirement: Every change declares its highest risk gate

每个 OpenSpec change MUST 在 DRAFT 中声明唯一 `gate_type`，并按 W0 结构、W1 内部逻辑、W2 公共契约/UI、W3 数据/资金/并发的最高命中风险取值；分类不得按文件数量、实现难度、平均风险、测试数量或覆盖率向下降级。

#### Scenario: Change touches more than one risk class
- **WHEN** 一个 change 同时包含内部逻辑和持久化数据、资金、权限或并发结果变更
- **THEN** `gate_type` 等于 W3
- **AND** writer 执行 W3 的全部最低 Gate

#### Scenario: Higher risk appears during implementation
- **WHEN** writer 发现实现需要触及比已批准类型更高的风险面
- **THEN** writer 在继续实现前更新 proposal、design、spec 和 tasks
- **AND** 旧批准与旧验证按变更后的 artifact 失效

### Requirement: Risk classes have fixed minimum Red Green Refactor evidence

质量协议 MUST 为 W0-W3 固定最低 Red、Green、Refactor：W0 使用结构/链接/内容完整性；W1 使用单元、边界和错误路径；W2 使用 provider/全部调用方契约、兼容/错误态和至少 UI1；W3 使用并发、幂等、事务、恢复及故障注入或等价可执行证据。所有 Red MUST 是目标行为缺失产生的可观察失败，覆盖率数字不得替代任一阶段证据。

#### Scenario: W0 document or structure change is implemented
- **WHEN** W0 writer 开始改变文档、链接、文件结构或内容完整性
- **THEN** writer 先记录同一结构、链接、schema、内容完整性或白名单检查的真实失败
- **AND** Green 与 Refactor 后重跑该检查、owned-path 和 diff 检查

#### Scenario: W1 internal behavior is implemented
- **WHEN** W1 writer 改变不涉及公共契约或持久化结果的内部逻辑
- **THEN** Red 覆盖最小单元、边界或错误路径
- **AND** Refactor 后重跑 focused test 与受影响回归，共享状态或并发代码还必须运行 race

#### Scenario: W2 public contract or UI is implemented
- **WHEN** W2 writer 改变公共 API/schema、调用方契约或用户可见 UI 行为
- **THEN** Red/Green 覆盖 provider、全部受影响调用方、兼容行为、错误态和至少 UI1
- **AND** 同时修改两端断言但没有独立契约证据不得视为通过

#### Scenario: W3 high-risk behavior is implemented
- **WHEN** W3 writer 改变数据、migration、权限、订单、支付、库存、核销、退款、事务或并发结果
- **THEN** Red/Green/Refactor 覆盖相关并发、幂等、事务、恢复、非法状态及故障注入或等价可执行证据
- **AND** 验收确认业务不变量成立且没有重复副作用

### Requirement: UI evidence is graded by actual runtime

每个 change MUST 声明一个 `ui_level`：UI0 静态、UI1 浏览器/模拟器、UI2 微信开发者工具/体验版、UI3 真机/真实平台。质量协议 MUST 包含 W0-W3 × UI0-UI3 的 16 格决策表；每格必须先满足风险类型 Gate，再追加实际执行的运行证据。W2 的 UI0 MUST 是硬阻断，未实际运行的更高等级不得记录为 PASS。

#### Scenario: W2 declares only static UI evidence
- **WHEN** W2 change 的最高已执行 UI 证据只有 UI0
- **THEN** writer Gate 返回 FAIL 或 `BLOCKED_EXTERNAL`
- **AND** change 不得形成通过的 candidate

#### Scenario: Mini-program runtime asset is unavailable
- **WHEN** 小程序专属验收需要目标 UI 等级，但微信工具、体验版、权限、账号、真机或平台资产不可用
- **THEN** 结果记录为 `BLOCKED_EXTERNAL`
- **AND** 记录 asset owner、缺失证据和恢复条件，不得以静态或低一级证据冒充 PASS

#### Scenario: Real platform evidence is reported
- **WHEN** reviewer 检查一个 UI3 PASS
- **THEN** 证据包含实际版本、环境、账号/设备边界、操作结果和未验证范围
- **AND** PASS 不推广为未覆盖机型、账号、环境或生产全量正确性

### Requirement: Writer Gate is permanent and evidence based

所有 writer MUST 在形成 CANDIDATE 前通过 OpenSpec strict、真实 Red/Green/Refactor、owned-path audit、敏感信息检查、适用于影响面的当前 Go/static checks 和 change 声明的全部 Gate；提交后 MUST 记录完整 candidate SHA 并保持 worktree clean。任何未执行命令不得写成 PASS。

#### Scenario: Writer prepares a candidate
- **WHEN** writer 完成 change-local implementation tasks
- **THEN** strict、RGR、owned paths、当前适用 Go/static checks、评分和硬阻断检查全部有可复现结果
- **AND** candidate evidence 包含完整 SHA 与 clean status

#### Scenario: Required external Gate cannot run
- **WHEN** change 验收依赖当前不可用的数据库、微信、支付、真机、CI、监控或其他外部资产
- **THEN** writer 记录 `BLOCKED_EXTERNAL` 而不是 PASS
- **AND** 在该 Gate 成为非必要或恢复条件满足前不得把 change 判为通过

#### Scenario: Coverage is high but business acceptance is missing
- **WHEN** 测试覆盖率或测试数量达到任意数值，但类别最低失败路径、业务不变量或运行证据缺失
- **THEN** writer Gate 仍然失败
- **AND** 评分不得补偿缺失证据

### Requirement: Evidence uses one sanitized template

每项 Gate 证据 MUST 至少记录 change、`gate_type`、`ui_level`、`base_sha`、阶段、命令或运行操作、退出结果、脱敏后的首个决定性错误或 PASS 摘要、artifact/环境和未验证边界。协议 MUST 给 W0-W3 分别提供可复制的命令/证据模板，并区分当前可执行命令与等待外部资产后启用的 Gate。

#### Scenario: Command evidence is recorded
- **WHEN** writer 或 verifier 将一项命令记录为 PASS 或 FAIL
- **THEN** 记录包含实际执行的命令、退出结果、对应 SHA/阶段和脱敏摘要
- **AND** 不用计划命令、推断或历史运行替代当前证据

#### Scenario: Conditional Gate is documented
- **WHEN** 质量文档列出 Playwright、微信、数据库、支付、真机、CI 或监控 Gate
- **THEN** 文档只记录启用所需资产、owner 和恢复条件
- **AND** 在仓库没有确定入口时不得编造已可执行命令

### Requirement: Independent verification is narrowly triggered

仓库生命周期的 worktree verifier MUST 只验证已提交的 CANDIDATE 完整 SHA，并在另一 clean detached worktree 重跑全部声明 Gate。探索、DRAFT、APPROVED、未提交 diff、branch 名或 moving ref MUST NOT 启动 verifier。待发布公共契约或高风险外部结论 MUST 在 candidate acceptance 中声明独立契约或运行证据，但没有 candidate SHA 时不得获得 `INDEPENDENT_VERIFIED` 状态。

#### Scenario: Draft asks for verifier
- **WHEN** change 仍是探索、DRAFT、APPROVED、未提交实现或没有完整 candidate SHA
- **THEN** 当前 planner/writer 只做本阶段自检
- **AND** 不创建 exact-SHA worktree verifier

#### Scenario: Public contract needs independent evidence
- **WHEN** candidate 将发布公共契约，或包含支付、真实数据、真实 UAT 等高风险外部结论
- **THEN** candidate acceptance 包含独立 reviewer/运行观察证据
- **AND** 该证据与 exact-SHA verifier 的仓库状态边界分别记录

### Requirement: Verification failure and invalidation return to the writer

verifier FAIL MUST 返回原 writer 修复；任何实现、spec、tasks、base、依赖、验收命令、rebase、merge 或 candidate SHA 变化 MUST 使旧验证失效。修复生成新 SHA 后，verifier MUST 在新的 clean detached worktree 从头重验。verifier session 复用和同一错误指纹第三次升级 MUST 引用 `loop-engineering-control-plane` 主 spec，不得在本协议复制跨 change 调度算法。

#### Scenario: Writer repairs a failed candidate
- **WHEN** verifier 返回 FAIL 且 writer 提交修复后的新 candidate SHA
- **THEN** 旧 SHA 的结果不适用于新 SHA
- **AND** verifier 为新 SHA 重建 clean detached worktree并完整重跑

#### Scenario: Same error fingerprint repeats
- **WHEN** 相同错误指纹连续第三次出现
- **THEN** stage skill 将升级与停止决策交给 `order-run-loop` 的权威规则
- **AND** 不执行第四次盲目重试或另建一套计数协议

#### Scenario: Runtime environment changes after PASS
- **WHEN** 数据库 schema、运行配置、微信 AppID/基础库/体验版、支付证书/回调、测试账号或真机条件发生变化
- **THEN** 相应 UI 或外部运行证据失效
- **AND** 新环境必须重新执行该 Gate 才能报告 PASS

### Requirement: Integration accepts only current main dependencies and an exact PASS

集成 Gate MUST 只接受所有声明依赖已经在当前 main 进入 `INTEGRATED`、candidate exact SHA 获得未失效 PASS、required review 满足且集成已获授权的 change。依赖的 branch、candidate 或 main 外 independent PASS 不得替代 `INTEGRATED`。

#### Scenario: Dependency has only a verified candidate
- **WHEN** 一个依赖只有 main 外 candidate 和 exact-SHA PASS，但尚未进入当前 main
- **THEN** 下游 change 不得集成
- **AND** 依赖保持未满足

#### Scenario: Main advances after candidate verification
- **WHEN** main 在 candidate 验证后推进，导致集成需要 rebase、merge 或冲突处理
- **THEN** 原 writer 在原 change worktree 更新并形成新 candidate SHA
- **AND** writer Gate 与 independent verification 从头重跑

### Requirement: Quality score cannot override hard blockers

单个 change MUST 使用 C/T/V/R 四维各 10 分评分：C 契约正确性、T 测试证据、V 验证独立性、R 可恢复性。只有总分 `>= 36`、每项 `>= 8` 且硬阻断为零时才能评分 PASS。敏感信息泄漏、必要 Gate 未运行、SHA 不符、P0 业务不变量失败、越过 owned paths 或未经授权写入/推送/部署 MUST 一票否决。

#### Scenario: Score passes but a hard blocker exists
- **WHEN** C/T/V/R 总分和单项均达到阈值，但存在任一硬阻断
- **THEN** verdict 是 FAIL 或 `BLOCKED_EXTERNAL`
- **AND** 分数、覆盖率或其他 PASS 不得抵消硬阻断

#### Scenario: Score is reported
- **WHEN** writer 或 verifier 报告 C/T/V/R
- **THEN** 每项分数绑定可追溯的 artifact、命令或独立运行证据
- **AND** 单个 change 评分不替代 `order-run-loop` 的主 Goal readiness rubric

### Requirement: Sensitive information never enters ordinary evidence

普通日志、tasks、命令输出、trace artifact 和主动回传 MUST NOT 包含 Authorization、Cookie、session/login code、私钥、证书、APIv3 key、手机号、姓名、openid、工号、用户备注、原始 query/body、支付/退款回调原文、完整核销 token/二维码或完整外部交易号。发现泄漏 MUST 立即使 Gate FAIL，并由 writer 清理后产生新 SHA。

#### Scenario: Sensitive value appears in test evidence
- **WHEN** reviewer 在普通日志、task 证据、trace 或回传中发现任一禁止值
- **THEN** 当前 Gate 一票否决
- **AND** 日志访问权限、脱敏承诺或评分不得豁免该失败

#### Scenario: Safe correlation is needed
- **WHEN** 测试或运行证据需要关联请求与业务事件
- **THEN** 使用服务端 request/trace/event ID、内部订单 ID、模板化 path、status、duration 或枚举化错误
- **AND** 关联 ID 不作为高基数指标 label

### Requirement: Stage skills reference one quality protocol without copying orchestration

批准后的实现 MUST 创建 `docs/quality/change-quality-gates.md`，并只对 `order-plan-change`、`order-implement-tdd`、`order-verify-change`、`order-integrate-change` 四个 stage skill 增加各自阶段的最小引用/检查。实现 MUST NOT 修改 `order-run-loop`、根 `AGENTS.md`、业务代码或产品/架构文档，也不得复制 `loop-engineering-control-plane` 的 lane、scheduler、ledger、checkpoint 或 session 规则。

#### Scenario: Four skills are inspected
- **WHEN** reviewer 检查四个 stage skill
- **THEN** 每个 skill 都引用 `docs/quality/change-quality-gates.md` 并只执行本阶段检查
- **AND** `order-run-loop` 的跨 change 调度规则没有被复制或替代

#### Scenario: Owned paths are audited
- **WHEN** writer 或 verifier 比较本 change 与基线的 changed paths
- **THEN** 所有路径均属于 proposal 声明的固定 owned paths
- **AND** `.agents/skills/order-run-loop/**`、根 `AGENTS.md`、业务代码和产品/架构文档没有变化
