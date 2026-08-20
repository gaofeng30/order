## 0. 门禁声明

```yaml
change: complete-baseline-traceability
gate_type: W0
ui_level_target: UI0
ui_level_actual: UI0
owner: worktree-complete-baseline-traceability
worktree: .claude/worktrees/complete-baseline-traceability
owned_paths:
  - openspec/changes/complete-baseline-traceability/**
base_sha: 939c9255c488e2f40468c17d557d79c523d3dda5
candidate_sha: external-post-commit
external_assets: none
dependencies:
  - realign-mvp-product-baseline（已集成于 base_sha）
```

W0 依据：只写 change 目录内的规格文本与门禁脚本，不改代码、公共 API、schema、持久化数据、权限或任何运行行为。

工具边界：仓库未安装 `openspec` CLI，`openspec validate --strict` 无法执行，记 `BLOCKED_EXTERNAL`，不记为 strict PASS。

门禁命令：

```
python3 openspec/changes/complete-baseline-traceability/checks/check_residue.py <tree-root>
```

脚本随 change 提交，只依赖 Python 标准库，可对任意树根复现。

## 1. Boundary and approval

- [x] 1.1 定位缺陷并确定最小修正范围。
  - Evidence: `realign-mvp-product-baseline` 把 `The baseline is traceable and has no behavioral TODO` 判为「未被客户评审触及」，该判定在 requirement 正文层面成立，在 scenario 层面不成立——`Traceability matrix is checked` 断言「页面、九态、四角色及 12 个外部 Gate 均无孤立项」，而九态与四角色已被同一批 delta 删除。生效 spec 因此自相矛盾。
- [x] 1.2 确定不回退重做上一个 change。
  - Evidence: 见 `design.md`。`realign-mvp-product-baseline` 已完成独立验证并集成，其验证结论对当时声明的范围仍然有效；缺陷是范围划少，属新发现问题，按新 change 处理可追溯，改写已集成历史则会让那次验证的 SHA 失去指向。
- [x] 1.3 确定门禁必须换一个方向。
  - Evidence: 上一轮门禁 `check_realign.py` 只检查 delta 内部一致性，对「delta 没写的部分」无感，而遗漏按定义不在 delta 里。新门禁以生效 spec 为输入、减去待集成 delta 覆盖的标题，对剩余部分执行残留检查。两者互补，都保留。

## 2. Red

- [x] 2.1 对 `base_sha` 树运行残留门禁，取得该缺陷的可观察失败。
  - Red: `python3 checks/check_residue.py <base-tree>` → `exit=1`；输出 `live_requirements=12 covered_by_pending_deltas=17 residue_hits=1` 与 `RESIDUE_IN_UNCOVERED_REQUIREMENT [The baseline is traceable and has no behavioral TODO] ['九态', '四角色'] :: - **WHEN** reviewer 逐条对照本 spec 与 PRD 追踪矩阵`，`RESIDUE_GATE=FAIL`。失败原因是生效 spec 中存在未被任何 delta 覆盖且仍引用已删除概念的 requirement，非断言构造。

## 3. Green

- [x] 3.1 建立 delta：MODIFIED `The baseline is traceable and has no behavioral TODO`。
  - Evidence: 追踪矩阵引用的状态集合改为六态、角色集合改为两角色；正文新增「MUST NOT 引用任何已被客户评审删除的概念作为待覆盖维度」；新增 scenario `Matrix cites a retired dimension`，使该约束可检查。原有两个 scenario 的其余断言保持不变。
- [x] 3.2 建立 `proposal.md`、`design.md` 与门禁脚本。
  - Evidence: proposal 声明 W0/UI0、owned paths 仅 change 目录、依赖 `realign-mvp-product-baseline` 且两 delta 必须同批 archive；design 记录不回退的理由、门禁换向的理由与三项风险。
- [x] 3.3 重跑同一门禁至 Green。
  - Green: `python3 checks/check_residue.py <candidate-tree>` → `live_requirements=12 covered_by_pending_deltas=18 residue_hits=0`，`RESIDUE_GATE=PASS`，`exit=0`。同一脚本、同一命令，仅目标树不同。

## 4. Refactor and writer gate

- [x] 4.1 复跑两个门禁，确认互不破坏。
  - Refactor: 残留门禁对候选树 `RESIDUE_GATE=PASS`（`residue_hits=0`，`covered_by_pending_deltas=18`）、对 `base_sha` 树 `exit=1` 红线仍成立；上一 change 的 `check_realign.py` 对候选树仍 `REALIGN_GATE=PASS`（`MODIFIED=4 ADDED=7 REMOVED=6 scenarios=44`）。两个门禁互补且互不破坏。
- [x] 4.2 owned-path 审计与 `git diff --check`。
  - Refactor: 审计输出 `changed=6 in_owned=6 outside=0`，`OWNED_PATH_AUDIT=PASS`；`git diff --cached --stat HEAD -- openspec/specs openspec/changes/realign-mvp-product-baseline docs apps services` 为空，生效 spec、上一 change 的已集成产物、产品文档与代码均未改动；`git diff --cached --check` 无输出，`DIFF_CHECK=PASS`。
- [x] 4.3 记录 W0/UI0 证据与 candidate SHA，checkpoint `CANDIDATE`。
  - Writer verdict: `{ gate_type: W0, ui_level_target: UI0, ui_level_actual: UI0, base_sha: 939c9255c488e2f40468c17d557d79c523d3dda5, candidate_sha: external-post-commit（见 5.1 记录的精确 SHA）, hard_blockers: 0, unverified_boundary: openspec CLI 缺失导致无 strict 校验；本 change 不产生运行行为，故无 UI1+ 证据 }`。

## 5. Independent verification

- [x] 5.1 在干净 detached worktree 对精确 candidate SHA 只读验证。
  - Verify: `candidate_sha=9a86306e4f00594ba47c6c62e503f73064932a48`，验证树 `git status --porcelain` 为空。残留门禁对候选树 `RESIDUE_GATE=PASS`（`residue_hits=0`）、对 `base_sha` 树 `RESIDUE_GATE=FAIL`（`residue_hits=1`，点名 `The baseline is traceable and has no behavioral TODO`）；`check_realign.py` 对候选树仍 `REALIGN_GATE=PASS`。diff 为 6 files / 246 insertions / 0 deletions，全部位于 owned paths 内。验证结束时验证树仍为 clean。
- [x] 5.2 记录 PASS/FAIL 与剩余外部边界。
  - Verdict: **PASS**（W0 / UI0）。
  - 剩余外部边界：① 仓库未安装 `openspec` CLI，`openspec validate --strict` 未执行，记 `BLOCKED_EXTERNAL`；② 2026-08-19 客户评审记录尚未取得书面签认（0818 PRD §16.4 C1）；③ 与 `online-ordering-system-prd.md` 维护方的同步尚未完成（§16.4 C2）；④ 本 change 不产生运行行为，无 UI1 及以上证据。
  - 残留门禁的覆盖边界：只作用于 `mvp-product-baseline`，且词表按当前已删除概念枚举，不保证发现未来新增的废止概念。

## 6. 集成前置

两个 delta MUST 在同一次 archive 中一并应用到 `openspec/specs/mvp-product-baseline/spec.md`。若只应用 `realign-mvp-product-baseline`，生效 spec 会处于「状态机规定六态、可追踪性要求覆盖九态」的自相矛盾中间态。
