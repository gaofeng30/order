## 0. 门禁声明

```yaml
change: realign-mvp-product-baseline
gate_type: W0
ui_level_target: UI0
ui_level_actual: UI0
owner: worktree-realign-mvp-product-baseline
worktree: .claude/worktrees/realign-mvp-product-baseline
owned_paths:
  - openspec/changes/realign-mvp-product-baseline/**
base_sha: 2f2db4a31f66f992997880a02b438c9690bbb845
candidate_sha: external-post-commit
external_assets: none
dependencies: none
```

W0 依据：本 change 只写 OpenSpec change 目录内的规格文本，不改代码、公共 API、schema、持久化数据、权限或任何运行行为，因此取 W0/UI0。若实现过程中触及 `apps/**`、`services/**` 或生效 spec 本体，必须先上调 `gate_type` 并重新批准。

工具边界：仓库当前未安装 `openspec` CLI（`openspec --version` → `command not found`），`openspec new change` 与 `openspec validate --strict` 无法执行。产物按仓库既有 change 的结构手工建立，并以下述可执行门禁替代。该缺口记为 `BLOCKED_EXTERNAL`，不记为 strict PASS。

门禁命令（下文各步统一使用）：

```
python3 openspec/changes/realign-mvp-product-baseline/checks/check_realign.py <tree-root>
```

脚本随 change 一并提交，只依赖 Python 标准库，可在任意干净 worktree 中对任意树根复现。

## 1. Boundary and approval

- [x] 1.1 读取 `AGENTS.md`、`docs/quality/change-quality-gates.md`、`.agents/skills/order-plan-change/SKILL.md`、评审记录、0818 PRD 与生效 spec，确定单一能力边界与验收判据。
  - Evidence: 单一 capability `mvp-product-baseline`；单一验收判据「spec 与 2026-08-19 客户评审记录一致且无残留已废止规则」；owned paths 只有 change 目录；无依赖；无外部资产。
- [x] 1.2 确认本 change 不可按主题拆分。
  - Evidence: 见 `design.md`「为什么是一个 change」——六态依赖「支付成功才建单」，后者依赖删除软预占；删库存依赖新增按日售罄；取餐时间点依赖删除即时单。任何部分回滚都会让 spec 自相矛盾，不满足 `AGENTS.md`「可以分别验收或回滚」的拆分条件。
- [x] 1.3 确定 delta 的 MODIFIED / ADDED / REMOVED 划分。
  - Evidence: 四条标题本身编码了被推翻的事实（nine-state / per-product / slot / four roles），走 REMOVED + ADDED；两条纯删除只有 REMOVED；四条标题仍准确的走 MODIFIED。`External readiness follows one twelve-gate chain` 与 `The baseline is traceable and has no behavioral TODO` 未被客户评审触及，保持不变，理由见 `design.md`。

## 2. Red

- [x] 2.1 在 `base_sha` 的干净 detached worktree 上运行门禁，取得目标产物缺失的可观察失败。
  - Red: `git worktree add --detach <base-tree> 2f2db4a31f66f992997880a02b438c9690bbb845` 后以候选树中的门禁脚本对该树执行 → `exit=1`，输出 `ARTIFACT_MISSING proposal.md / design.md / tasks.md / specs/mvp-product-baseline/spec.md / .openspec.yaml`，`REALIGN_GATE=FAIL`。失败原因是目标规格产物不存在，非断言构造。

## 3. Green

- [x] 3.1 建立 `.openspec.yaml` 与 `proposal.md`：why、单一结果、能力、owned paths、依赖、非目标、门禁字段与最小成功标准。
  - Evidence: `proposal.md` 声明 `gate_type=W0`、`ui_level_target=UI0`、`base_sha`、owned paths 仅 change 目录、无依赖、非目标含「不修改生效 spec 本体」，并列出 9 条冲突 requirement 与 1 条 authority order 具体化。
- [x] 3.2 建立 `specs/mvp-product-baseline/spec.md` delta。
  - Evidence: 4 MODIFIED + 7 ADDED + 6 REMOVED = 17 条 requirement、44 个 scenario。全部 MODIFIED/REMOVED 标题与基线精确匹配，全部 ADDED 标题在基线中不存在（标题匹配脚本输出 `HEADER_MATCH=PASS`）。
- [x] 3.3 建立 `design.md`：单一实现取向、拆分判定、REMOVED+ADDED 取舍、十二 Gate 链不动的理由、风险与失效条件。
  - Evidence: 记录了四项 Decisions、四项 Risks 与四条 invalidation conditions。
- [x] 3.4 建立 `tasks.md` 并重跑同一门禁至 Green。
  - Green: 同一脚本对候选树执行 → `requirements: MODIFIED=4 ADDED=7 REMOVED=6 scenarios=44`，`REALIGN_GATE=PASS`，`exit=0`。同一命令、同一脚本，仅目标树不同。
  - 门禁修正（如实记录）：首次 Green 尝试报 `RETIRED_AFFIRMATIVE line283 ['即时']`，命中的是 `- **WHEN** 任一调用方请求即时取餐...`，其否定词落在同一场景的 `**THEN** 服务端拒绝该请求` 上。按单行判定属假阳性，故把「已废止概念只在否定语境出现」的判定粒度由单行改为空行分隔的段落/场景块，判定意图不变。修正后 Red 树仍为 `REALIGN_GATE=FAIL`，未削弱红线。

## 4. Refactor and writer gate

- [x] 4.1 复跑门禁并追加内容一致性检查：已废止概念只在否定语境出现、REMOVED 条目均带 Reason 与 Migration、MODIFIED/ADDED 条目均有 scenario、proposal 门禁字段齐全。
  - Refactor: 以 change 目录内提交的脚本对候选树复跑 → `REALIGN_GATE=PASS`；对 `base_sha` 树复跑 → `REALIGN_GATE=FAIL`（`exit=1`）。同一脚本、同一命令，红线未被削弱。10 条目标基线 requirement 全部被 MODIFIED 或 REMOVED 覆盖；6 条 REMOVED 均带 Reason 与 Migration；11 条 MODIFIED/ADDED 均至少一个 scenario。
- [x] 4.2 owned-path 审计与 `git diff --check`。
  - Refactor: `git status --porcelain` 只有 `openspec/changes/realign-mvp-product-baseline/`，审计输出 `changed=1 in_owned=1 outside=0`，`OWNED_PATH_AUDIT=PASS`；`git diff --stat HEAD -- openspec/specs docs apps services AGENTS.md CLAUDE.md` 为空，生效 spec、产品文档、代码与治理文件均未改动；`git diff --cached --check` 无输出，`DIFF_CHECK=PASS`。
- [x] 4.3 记录 W0/UI0 证据与 candidate SHA，checkpoint `CANDIDATE`。
  - Writer verdict: `{ gate_type: W0, ui_level_target: UI0, ui_level_actual: UI0, base_sha: 2f2db4a31f66f992997880a02b438c9690bbb845, candidate_sha: external-post-commit（不可写入自身，见 5.1 记录的精确 SHA）, hard_blockers: 0, unverified_boundary: openspec CLI 缺失导致无 strict 校验；客户书面签认未取得；本 change 不产生运行行为，故无 UI1+ 证据 }`。

## 5. Independent verification

- [ ] 5.1 在另一个干净的 detached worktree 对精确 candidate SHA 只读验证；内容、base 或 SHA 任一变化即失效。
- [ ] 5.2 记录 PASS/FAIL 与剩余外部边界（`openspec` CLI 缺失、客户书面签认未取得），不修改候选字节。

## 6. 未纳入本 change 的后续动作

以下动作在本 change 之外单独执行，记录于此避免遗漏：

- 废弃 `feat/member-coupon` 分支（对应已删除的会员券能力）。
- 前端删除类 change：会员券 7 屏 + PC 4 页、标签/过敏原/月售/库存位、品牌选择页、商户端工作台、首页营销位、TabBar 五格→三格。
- 前端与后端新增类 change：六态状态机、取餐时间选择、手机号绑定与双要素、订阅消息、扫码登录、对账兜底、折扣算价。
- 0818 PRD §16.4 C1（客户书面签认）与 C2（与现行基线维护方同步），由商务与文档提出方推进。
