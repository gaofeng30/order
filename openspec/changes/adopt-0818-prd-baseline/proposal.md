## Why

2026-08-19 客户澄清已完整落入 `docs/product/online-ordering-system-prd-0818.md`，但旧 canonical PRD 和现行 `mvp-product-baseline` 仍要求数量库存、软预占、九态、四角色和逐商品员工价等相反规则。继续保留多套有效基线会让后续 change 在范围、资金状态、权限和验收上选择性引用冲突契约，因此必须先把客户已确认的一期口径收敛成唯一产品基线。

## What Changes

- **BREAKING**：把 `docs/product/online-ordering-system-prd-0818.md` §1–§14（含其已经吸收的 2026-08-19 客户澄清）确立为仓库唯一有效产品 PRD；`docs/product/online-ordering-system-prd-0818-review.md` 只保留为其裁决证据。
- 把 `docs/product/online-ordering-system-prd.md` 改为薄指针和明确废止说明，不再并列维护旧业务正文，也不得继续作为实现或验收依据。
- 对现行 `mvp-product-baseline` 提供完整 delta：冻结 0818 PRD §3 的一期包含/排除范围以及 §12 的全部 16 条业务不变量，删除数量库存/软预占/迟到支付/九态/接单/四角色/逐商品员工价等旧要求。
- 保留 0818 PRD §16.3 的 P1–P5 为下游模块的产品决策阻塞，不在本 change 内替客户决定；这些边界不影响本 W0 基线收敛。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `mvp-product-baseline`: 改为以 0818 PRD §1–§14 为唯一有效产品基线，完整冻结一期范围与 16 条业务不变量，并移除旧库存、支付、状态、角色和员工价契约。

## Impact

- Primary outcome：仓库只剩一个有效产品基线，所有后续产品、API、数据、前端和验收 change 均只能从 0818 PRD §1–§14 与归档后的 canonical `mvp-product-baseline` 推导。
- Owner：branch `codex/adopt-0818-prd-baseline`，worktree `/Users/vivix/.codex/worktrees/order-adopt-0818-prd-baseline.Writer`。
- Base SHA：`2f2db4a31f66f992997880a02b438c9690bbb845`。
- Owned paths：`docs/product/online-ordering-system-prd.md`、`openspec/changes/adopt-0818-prd-baseline/**`。
- Read-only shared contracts：`docs/product/online-ordering-system-prd-0818.md`、`docs/product/online-ordering-system-prd-0818-review.md`、`openspec/specs/mvp-product-baseline/spec.md`。
- Dependencies：none。
- External assets：none；asset owner/recovery 均为 `N/A`，不存在 `BLOCKED_EXTERNAL`。
- Non-goals：不修改 0818 PRD、客户评审记录、现有 canonical specs、业务代码、API、schema、运行行为、AGENTS、Skills、Harness 或其他 existing changes；不替 P1–P5 做产品裁决；不推送、部署、集成或归档。
- Gate：`gate_type=W0`；`ui_level_target=UI0`；`ui_level_actual=UI0`（focused 内容/结构检查已在 Green 与 Refactor 真实运行；不证明 UI1、运行行为、真实微信或生产结果）。
- 唯一接受裁决：仅当旧 PRD 已成为只指向 0818 PRD 的废止薄指针、完整 delta 与 0818 PRD §3/§12 逐项一致、P1–P5 未被擅自决定、strict/内容/owned-path/diff 检查全部通过且 exact candidate 获得独立只读验证时，结果为 `ACCEPT`；任一条件不满足即 `REJECT`。
