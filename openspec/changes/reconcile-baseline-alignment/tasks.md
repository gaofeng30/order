## 0. 门禁声明

```yaml
change: reconcile-baseline-alignment
gate_type: W0
ui_level_target: UI0
ui_level_actual: UI0
owner: worktree-reconcile-baseline
worktree: .claude/worktrees/reconcile-baseline
owned_paths:
  - openspec/specs/mvp-product-baseline/**
  - openspec/changes/reconcile-baseline-alignment/**
  - openspec/changes/adopt-0818-prd-baseline/**
  - docs/product/online-ordering-system-prd-0818.md
base_sha: e74a454fffa9f784bcbc185141899873c6ad1443
candidate_sha: external-post-commit
external_assets: none
```

门禁命令：

```
python3 openspec/changes/reconcile-baseline-alignment/checks/check_baseline_single_source.py <tree-root>
```

## 1. Boundary and approval

- [x] 1.1 确认冲突性质与取代判据。
  - Evidence: 两线零文件冲突、语义重叠。远端 delta 未归档，取代成本低；本地线已生效且后续五个已集成 change 的 delta 依赖其标题，回滚成本高。用户于 2026-08-21 裁决保留本地线。
- [x] 1.2 确认远端线的两条独有 requirement 值得吸收。
  - Evidence: 覆盖本地线未写全的取餐号编号与跨日核销、二维码 token 生命周期、两类订阅消息及兜底、统计口径与未取餐查询。逐字吸收不改写。

## 2. Red

- [x] 2.1 编写门禁并对合并后的树执行。
  - Red: `BASELINE_SINGLE_SOURCE=FAIL`，四项命中：生效 spec 缺两条吸收项、存在重复的未归档 baseline delta（`adopt-0818-prd-baseline`）、以及可追踪性 requirement 中已废止概念的疑似肯定性表述。

## 3. Green

- [x] 3.1 吸收两条 requirement 到生效 spec。
  - Evidence: `openspec/specs/mvp-product-baseline/spec.md` 13 → 15 条，新增 5 个 scenario。两条均为 ADDED，未改动任何既有标题。
- [x] 3.2 归档并说明取代。
  - Evidence: `adopt-0818-prd-baseline` → `archive/2026-08-21-adopt-0818-prd-baseline-superseded`，附 `SUPERSEDED.md`。
- [x] 3.3 更新 0818 PRD 的基线地位。
  - Evidence: 顶部定位与 §0.4 由「修订提案 / 不取代旧基线」改为「唯一有效产品基线」，并记录 §16.4 C2 已完成。
- [x] 3.4 重跑门禁至 Green。
  - Green: `live_requirements=15 un_archived_baseline_deltas=0 pointer_lines=11`，`BASELINE_SINGLE_SOURCE=PASS`。

## 4. Refactor and writer gate

- [x] 4.1 修正门禁自身两处缺陷。
  - Refactor: ① 重复 delta 扫描把本 change 自己也算了进去，改为按名排除自身。② 否定词表缺「失败」，导致对 `Matrix cites a retired dimension` 场景误判——该缺口在 `complete-baseline-traceability` 的 design.md 中被记录为「留给下一个触及本 spec 的 change」，本 change 即是，已补齐。
- [x] 4.2 更正 `SUPERSEDED.md` 中一处不准确表述。
  - Refactor: 初稿写「`verify_baseline.py` 对当前树仍为 PASS」。实测归档后该脚本按自身路径深度推导仓库根（`parents[4]`），目录多一层后解析失效，报 `missing required file`。已改为如实记录，并说明其约束已由新门禁继承扩展。归档产物不修改。
- [x] 4.3 全量回归。
  - Refactor: 小程序 `npm test` 59/59；四个前端门禁全部 PASS（`PICKUP_SETTINGS_GATE` / `ORDER_LIFECYCLE_GATE` / `ADMIN_SCOPE_GATE` / `CATALOG_FIELDS_GATE`）；`go build ./...` 通过。
- [x] 4.4 记录门禁证据与 candidate SHA。
  - Writer verdict: `{ gate_type: W0, ui_level_target: UI0, ui_level_actual: UI0, base_sha: e74a454fffa9f784bcbc185141899873c6ad1443, candidate_sha: external-post-commit（见 5.1）, hard_blockers: 0, unverified_boundary: openspec CLI 缺失；本 change 不产生运行行为故无 UI1+ 证据 }`。

## 5. Independent verification

- [x] 5.1 在干净 detached worktree 对精确 candidate SHA 只读验证。
  - Verify: `candidate_sha=4168d74fc78f850efecf7cae46fb56f83fccfa8a`，验证树 clean。`BASELINE_SINGLE_SOURCE=PASS`（`live_requirements=15 un_archived_baseline_deltas=0 pointer_lines=11`）；小程序 `npm test` 59/59；四个前端门禁全部 PASS；`go build ./...` 通过、`go test ./services/api/...` 无 FAIL。diff 相对 base 为 16 files / 345 insertions / 6 deletions。验证结束时验证树仍为 clean。
- [x] 5.2 记录 PASS/FAIL 与剩余外部边界。
  - Verdict: **PASS（W0 / UI0）**。
  - 剩余外部边界：① 取代判定为用户裁决，远端线维护方尚未确认，`SUPERSEDED.md` 已完整记录判据与回滚路径；② 归档后的 `verify_baseline.py` 因路径深度失效，归档产物不修改，其约束已由新门禁继承；③ 仓库未安装 `openspec` CLI，strict 校验记 `BLOCKED_EXTERNAL`；④ 本 change 不产生运行行为，无 UI1 及以上证据。

## 6. 后续

- 远端后端已落地会话、手机号绑定、预约菜单可用性，并加了 `products.meal_period`、`meal_periods`、`product_sold_out_dates` 三处 schema。此前声明为「受后端阻塞」的三项已解除，前端接入属后续 change。
- 仍未处理：`feat/member-coupon` 分支废弃；`apps/web-admin` 的可提交 runner。
