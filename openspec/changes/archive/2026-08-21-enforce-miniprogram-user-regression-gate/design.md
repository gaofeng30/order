## Context

现有质量协议能告诉执行者“应该做什么”，但无法阻止一个 change 用勾选框或自由文本绕过。决定性缺口有两个：`tools/harness` 不理解小程序影响面/TDD/UI 语义；真实用户回归通常发生在 candidate 形成后，却没有绑定 exact SHA 的结构化凭证。GitHub required check 当前不存在，且远端配置不在本 change 授权范围。

本 change 是 dedicated runner/control-plane evolution。它必须遵循冻结基线、non-weakening、正反例回归、clean-detached exact-SHA forward-test 和独立验证；集成前不激活，现有业务模块不原地切换 judge。

## Goals / Non-Goals

**Goals:**

- 任何新 candidate 只要实际 diff 触及 `apps/wechat-miniprogram/**`，就不能省略结构化 TDD/用户回归计划。
- TDD 在形成 candidate 前机器校验；用户回归在 exact candidate 后执行并在进入 `INDEPENDENT_VERIFIED` 前机器校验。
- 普通小程序模块最低 UI2；平台原生能力最低 UI3。
- 旧 change、旧 candidate 和当前 session 模块不被追溯改写；main 推进时按现有规则形成新 candidate 后适用新 Gate。
- 保留现有生命周期、lane、授权、独立验证、敏感信息和业务 Gate，不以新规则削弱任何旧约束。

**Non-Goals:**

- 自动控制微信开发者工具、体验版或真机；自动证明人工没有伪造外部运行结果。
- 修改小程序业务代码、当前 session candidate、远端 GitHub workflow/branch protection。
- 新增第二套生命周期、评分、runner 或第三方依赖。

## Decisions

### 1. 每个小程序 change 使用一个结构化 `miniprogram-gates.json`

文件固定放在 `openspec/changes/<change>/miniprogram-gates.json`，顶层精确包含：

- `schema_version: 1`、`change`、完整 `base_sha`；
- `tdd.red/green/refactor`，每项精确包含 `task_id` 与非空 `command`；
- `user_regression.minimum_ui_level`、`native_capabilities` 与 `scenarios`；
- scenarios 至少包含一个 `primary` 和一个 `regression`，每项有稳定 `id`、非空 `steps` 与 `expected`。

选择 JSON 是为了使用 Python 标准库获得确定解析、精确字段和反例测试。Harness 不执行 manifest 中任意命令；writer/verifier 仍按 stage Skill 实际运行，避免把仓库文本变成隐式 shell 执行面。

候选检查使用 manifest 的 `base_sha...candidate_sha` diff。若 diff 触及小程序但 candidate 内没有合法 manifest，直接失败；因此即使 planning 阶段有人漏声明，也无法形成通过的 candidate。

### 2. UI 等级由影响面下限决定，不由 Agent 自评降低

无平台原生能力的小程序 change 的 `minimum_ui_level` 必须至少 UI2。`native_capabilities` 非空时必须为 UI3；checker 还扫描 candidate diff 中的 `wx.login`、`getPhoneNumber`、`wx.requestPayment`、`wx.scanCode` 与 `wx.requestSubscribeMessage` 标记，发现漏报即失败。

UI1 Harness 仍是 candidate 前本地用户路径回归和 TDD 证据，但不能替代 UI2/UI3。缺少真实资产时保持 `BLOCKED_EXTERNAL`，可以保留代码 candidate，但不得进入 `INDEPENDENT_VERIFIED` 或 `INTEGRATED`。

### 3. 用户回归使用 candidate 外结构化 receipt，避免自引用 SHA

真实用户回归只能在 candidate SHA 已存在后执行。调用方通过 `./tools/harness checkpoint ... --user-regression-receipt <file>` 提交一个本地 JSON receipt，精确包含：

- `schema_version`、`change`、`candidate_sha`、`ui_level`、`environment`；
- manifest 的全部 scenario ID 且每项 `result=PASS`；
- 总 `result=PASS` 与非空 `unverified_boundary`。

Harness 读取真实普通文件（拒绝 symlink/非文件），校验 exact SHA、等级和场景全集，只在 Git common-dir operational state 中保存规范化字段与 SHA256。candidate bytes 保持不变；candidate、manifest、scenario 或环境变化使 receipt 失效。自由文本、截图数量或低等级 PASS 不能替代该 receipt。

receipt 仍是外部运行声明，不单独证明真实性；`order-verify-change` 必须把它作为输入重新核对，`order-integrate-change` 再运行 Harness Gate。远端 required check/branch protection 是进一步防人为绕过的独立外部动作。

### 4. 在现有四命令 Harness 中扩展 checkpoint，不新增控制面

保留 `status/check/checkpoint/observe` 四个命令。`checkpoint` 新增一个可选 receipt 参数；`validate_record` 根据激活状态、candidate、manifest、tasks 和 receipt fail closed。状态输出只显示 Gate 状态/等级/SHA，不复制外部原文。

规则的激活 marker 是 candidate 基线中已存在的 `tools/miniprogram_gate.py`。本控制 change 的 base 不含 marker，因此不会自审判；只有其集成后的新模块基线才启用。旧 candidate 的 parent/base 不含 marker，仍由旧规则读取；但 main 一旦推进，旧业务 candidate 必须按既有 moving-main 规则重建，其新 base 将包含 marker。

### 5. checker 独立于 Harness 编排并可被 verifier/未来 CI 复用

`tools/miniprogram_gate.py` 只负责 manifest、candidate diff、TDD task、native capability 与 receipt 语义，不写状态、不执行业务命令。Harness 调用其函数；独立 verifier 可直接运行其 CLI。这样未来 GitHub required check 可以复用同一 checker，不重写规则。

## Risks / Trade-offs

- [结构化 receipt 仍可能被人伪造] → exact-SHA verifier 必须核对实际运行环境与证据；后续单独授权 GitHub required check/branch protection 才能防直接绕过本地流程。
- [UI2/UI3 资产缺失导致代码 candidate 停留] → 明确允许 `CANDIDATE`，但 `BLOCKED_EXTERNAL` 阻断更高状态；不把低层证据冒充完成。
- [规则追溯应用会让历史 active changes 全部失败] → 仅对 base 已包含 marker 的模块激活；moving-main 规则自然使待集成旧 candidate 重建。
- [静态 native token 扫描不覆盖间接包装] → manifest 仍要求作者声明，verifier 审查调用链；扫描只做已知直接调用的 fail-closed 补强，不声称完整程序分析。
- [Harness 复杂度增加] → 语义集中在一个标准库模块，Harness 只传递记录并保持四命令；不新增服务、依赖或第二状态机。

## Migration Plan

1. 在隔离 writer worktree 取得当前 false-green Red。
2. 实现 checker、Harness 集成、规则/Skill 最小更新及完整正反例。
3. exact candidate 在 clean detached worktree执行 minimal-context forward-test 与全部 non-weakening/回归。
4. 未获单独集成授权前保持 main 不变；授权纯快进后，规则只对后续冻结的新模块生效。
5. 当前 session change 若继续集成，因 main 已推进必须从新 base 形成 candidate、补 manifest 和 UI3 用户回归。

回滚只需撤销本控制 change；产品代码与数据不受影响。已经按此规则形成的业务证据仍可保留，但回滚后不再由 Harness 强制。

## Open Questions

无。用户已经确认采用“普通小程序 UI2、平台原生能力 UI3”的推荐硬 Gate；远端 required check/branch protection 需要后续单独授权。
