## Why

两条并行工作线各自把生效 spec `mvp-product-baseline` 对齐到 0818 PRD，产生了重复且不兼容的产物。

- 远端线 `adopt-0818-prd-baseline`（`f716348`）：产物完整但**未归档**，delta 从未应用到生效 spec；同时它把 `docs/product/online-ordering-system-prd.md` 改成了指向 0818 的薄指针，该动作已生效。
- 本地线 `realign-mvp-product-baseline` + `complete-baseline-traceability`（`e451f52` 归档）：已应用到生效 spec，且后续五个已集成 change 的 delta 全部基于其 requirement 标题。

两者零文件冲突但语义重叠：requirement 集合高度一致，标题与措辞不同。远端 delta 的 `MODIFIED` / `REMOVED` 按标题匹配，而那些标题在生效 spec 中已被本地线 REMOVED——此时归档远端 delta 会失败或产生重复 requirement。

同时，0818 PRD 仍自称「对现行基线的修订提案，不是生效基线」，与远端已完成的废止动作矛盾。

## What Changes

- 吸收远端 delta 的两条**独有** requirement 到生效 spec：`Pickup identifiers and notifications follow the frozen contract`（取餐号编号与跨日核销、二维码 token 生命周期、两类一次性订阅消息及其兜底）与 `Production facts and statistics come from server-confirmed data`（服务端唯一事实源、统计口径与未取餐查询）。逐字吸收，不改写。
- 把 `adopt-0818-prd-baseline` 归档为 `2026-08-21-adopt-0818-prd-baseline-superseded`，并附 `SUPERSEDED.md` 记录取代原因、裁决依据与成果去向。
- 更新 0818 PRD：顶部定位与 §0.4 由「修订提案」改为「唯一有效产品基线」，并记录 §16.4 C2「与现行基线维护方对齐」已完成。
- 新增门禁 `check_baseline_single_source.py`：生效 spec 覆盖两条吸收项、不得存在第二份未归档的 baseline delta、旧 PRD 必须是指向 0818 的薄指针、0818 PRD 不得再自称提案、生效 spec 中不得有已废止概念的肯定性表述。

## Capabilities

### Modified Capabilities

- `mvp-product-baseline`: 追加取餐凭证与通知契约、生产事实与统计口径两条 requirement，13 → 15 条。

## Impact

- Owner：branch `worktree-reconcile-baseline`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/reconcile-baseline`。
- Owned paths：`openspec/specs/mvp-product-baseline/**`、`openspec/changes/reconcile-baseline-alignment/**`、`openspec/changes/adopt-0818-prd-baseline/**`（归档移动）、`docs/product/online-ordering-system-prd-0818.md`。
- Dependency：本地 merge `e74a454` 已把远端 10 个 commit 集成到 `base_sha`。
- Non-goals：不修改 `services/**`（远端后端实现）；不修改其他已归档 change 的产物；不实现远端后端已落地能力对应的前端接入（属后续 change）。
- Gate：`gate_type=W0`（只改 spec 与产品文档，不改运行行为）；`ui_level_target=UI0`；`ui_level_actual=UI0`。
- 最小成功标准：生效 spec 15 条且含两条吸收项；仓库中不存在第二份未归档的 baseline delta；旧 PRD 为薄指针；0818 PRD 自述为唯一基线；小程序 59 项与四个前端门禁、Go build 均不回归。
- 工具边界：仓库未安装 `openspec` CLI，strict 校验记 `BLOCKED_EXTERNAL`。
