## Why

`realign-mvp-product-baseline`（已集成 `939c925`）把一期产品基线对齐到 2026-08-19 客户评审记录，但其 delta 遗漏了一条 requirement。

`The baseline is traceable and has no behavioral TODO` 当时被判定为「未被客户评审触及」而保持不变。该判定只看了 requirement 的正文，没有看它的 scenario：其中一条断言写着「页面、**九态**、**四角色**及 12 个外部 Gate 均无孤立项」。

九态与四角色已被同一批 delta 删除。结果是生效 spec 内部自相矛盾：状态机 requirement 规定六态、权限 requirement 规定两角色，而可追踪性 requirement 要求追踪矩阵覆盖九态与四角色。任何据此建立追踪矩阵的后续 change 都会被要求覆盖不存在的维度。

上一轮门禁只检查 delta 自身，不检查「未被 delta 覆盖的生效 requirement 是否仍有残留」，因此漏检。

## What Changes

- MODIFIED `The baseline is traceable and has no behavioral TODO`：追踪矩阵引用的状态集合改为六态、角色集合改为两角色，并新增一条禁止矩阵引用已删除维度的 scenario。
- 新增残留门禁 `checks/check_residue.py`：对生效 spec 中**未被任何待集成 delta 覆盖**的 requirement 执行已废止概念检查，关闭上一轮的漏检路径。

本 change 只改写 spec 文本与门禁脚本，不改代码、API、schema、数据或任何运行行为。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `mvp-product-baseline`: 补齐 `realign-mvp-product-baseline` 遗漏的可追踪性 requirement，消除生效 spec 内部关于状态数与角色数的自相矛盾。

## Impact

- Owner：branch `worktree-complete-baseline-traceability`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/complete-baseline-traceability`。
- Owned paths：`openspec/changes/complete-baseline-traceability/**`。
- Read-only evidence：`openspec/specs/mvp-product-baseline/spec.md`、`openspec/changes/realign-mvp-product-baseline/**`、`docs/product/online-ordering-system-prd-0818-review.md`、`docs/product/online-ordering-system-prd-0818.md`。
- Dependency：`realign-mvp-product-baseline`，已集成于 `base_sha` `939c9255c488e2f40468c17d557d79c523d3dda5`。两个 change 的 delta MUST 在同一次 archive 中一并应用到生效 spec。
- Non-goals：不修改 `openspec/specs/mvp-product-baseline/spec.md` 本体；不修改 `realign-mvp-product-baseline` 的已集成产物；不修改产品文档、代码、schema 或部署配置；不处理 `feat/member-coupon` 分支的废弃。
- Gate：`gate_type=W0`；`ui_level_target=UI0`；`ui_level_actual=UI0`；external assets none。
- 最小成功标准：残留门禁对 `base_sha` 树 FAIL、对候选树 PASS；生效 spec 应用两个 delta 后共 13 条 requirement，且不存在任何已废止概念的肯定性引用；diff 只包含 owned paths。
- 工具边界：仓库未安装 `openspec` CLI，`openspec validate --strict` 无法执行，记 `BLOCKED_EXTERNAL`。
