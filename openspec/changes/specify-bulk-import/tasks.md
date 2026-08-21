## 0. 门禁声明

```yaml
change: specify-bulk-import
gate_type: W0
ui_level_target: UI0
ui_level_actual: UI0
owner: worktree-bulk-import-prd
worktree: .claude/worktrees/bulk-import-prd
owned_paths:
  - docs/product/online-ordering-system-prd-0818.md
  - openspec/changes/specify-bulk-import/**
base_sha: eabba9a92cb949ff4b237cd2e190096703af5a67
candidate_sha: external-post-commit
external_assets: none
```

门禁命令：`python3 openspec/changes/specify-bulk-import/checks/check_bulk_import_prd.py <tree-root>`

## 1. Boundary and approval

- [x] 1.1 核查 PRD 现状。
  - Evidence: §6.3 只有单个新增/编辑；§4.4 未写商户账号录入方式；仅 §6.4 有一段 CSV 导入表述，继承自已删除的会员名单，客户从未表态。评审记录全文对「导入」零命中。
- [x] 1.2 确认「一一对应」约束暴露的实现缺口。
  - Evidence: PC 菜品编辑表单字段为图片/名称/售价/分类/描述，**缺 `餐段可售`**；而 PRD §6.3 要求该字段三选一，后端已有 `products.meal_period` 列与 `meal_periods` 表。表单缺失导致该字段当前无人能填。
- [x] 1.3 逐项取得用户裁决。
  - Evidence: 受理格式 `.xlsx`（2026-08-21）；菜品去重选 A（只新增不更新）；图片不进 Excel、解析后逐个上传；餐段可售取值 全天 / 午餐 / 晚餐；规格与口味选项一期不做；分类不存在则自动新建；PRD 没有则先补充再开发。

## 2. Red

- [x] 2.1 编写门禁并对 `base_sha` 树执行。
  - Red: `BULK_IMPORT_PRD=FAIL`（`exit=1`），报缺 §6.13 四个子节、缺 `.xlsx` 受理声明、缺两套去重规则、缺范围来源标注、页面清单未含两个导入页等。

## 3. Green

- [x] 3.1 修订 §6.3，固定表单字段集合并记录已知偏差。
- [x] 3.2 新增 §6.13，含 6.13.1 通用流程 / 6.13.2 菜品 / 6.13.3 员工 / 6.13.4 不做导入的对象。
- [x] 3.3 同步 §6.4、§3.5、§6.10、§15.5.3、§15.11。
- [x] 3.4 §16.3 增 P6/P7，§16.4 增 C4，§16.5 增四项前端缺口。
- [x] 3.5 重跑门禁至 Green。
  - Green: `BULK_IMPORT_PRD=PASS`（`prd_lines=1108`）；同一脚本对 `base_sha` 树仍 `exit=1`。

## 4. Refactor and writer gate

- [x] 4.1 修正一处真残留与一处门禁误伤。
  - Refactor: ① §15.5.3 的员工白名单行仍写「保留 CSV 导入与预览」，已改为「新建 · 含 `.xlsx` 批量导入入口」，同时把该行状态由「改造」改为「新建」——PC 侧该页从未建过。② 门禁最初写 `forbid(prd, "另存为 CSV UTF-8")`，命中了 §6.13.1 中解释「为何改收 xlsx」的正当引用。**这是我第五次犯「断言词而非断言事实」**，已改为禁止三条真正表示「以 CSV 为受理格式」的表述：`CSV 批量导入`、`导入页做显式检测并提示`、`| CSV 为 GBK 编码 |`。
- [x] 4.2 门禁加入模板列与表单字段的交叉检查。
  - Refactor: 从 §6.3 解析出表单字段集合，逐项断言存在；再从 §6.13.2 解析模板列，断言五列齐全且**不含菜品图片列**。这条直接执行用户提的「一一对应」约束。
- [x] 4.3 确认本 change 不触碰生效 spec。
  - Refactor: 门禁断言生效 spec 中不出现「批量导入」——对应 requirement 应在实现 change 中随代码一并建立，而非文档先行。
- [x] 4.4 记录门禁证据与 candidate SHA。
  - Writer verdict: `{ gate_type: W0, ui_level_target: UI0, ui_level_actual: UI0, base_sha: eabba9a92cb949ff4b237cd2e190096703af5a67, candidate_sha: external-post-commit（见 5.1）, hard_blockers: 0, unverified_boundary: §6.13 全节待客户确认；P6/P7 待拍板；本 change 不产生运行行为故无 UI1+ 证据 }`。

## 5. Independent verification

- [x] 5.1 在干净 detached worktree 对精确 candidate SHA 只读验证。
  - Verify: `candidate_sha=1f5486e6a979bc421b725f3ceeeaa47ca418da2f`，验证树 clean。`BULK_IMPORT_PRD=PASS`（`prd_lines=1108`），同一脚本对 `base_sha` 树 `exit=1`；小程序 `npm test` 65/65；`BASELINE_SINGLE_SOURCE=PASS`。diff 相对 base 为 6 files / 318 insertions / 7 deletions，全部在 owned paths 内。
- [x] 5.2 记录 PASS/FAIL 与剩余外部边界。
  - Verdict: **PASS（W0 / UI0）**。
  - 剩余外部边界：① **§6.13 全节待客户确认**（§16.4 C4），取得确认前不得对客户表述为已含在已付款范围内；② §16.3 P6（单文件最大行数）与 P7（导入后占位图策略）待拍板，需按真实规模与运营偏好给值；③ 本 change 只补文档，生效 spec 的对应 requirement 待实现 change 建立；④ 不产生运行行为，无 UI1 及以上证据；⑤ 仓库未安装 `openspec` CLI，strict 校验记 `BLOCKED_EXTERNAL`。

## 6. 后续实现顺序

本 change 只补文档。实现按依赖分三步，建议各自独立成 change：

1. **PC 菜品表单补 `餐段可售`** —— 后端列已存在，是纯前端 + 契约补齐，最小且解锁菜单按餐段过滤。
2. **员工折扣白名单页 + 商户账号名单页** —— 两份名单都未建，是启动路由判定、PC 扫码登录与全局折扣率的共同前提。
3. **两个批量导入页 + 服务端解析接口** —— 依赖前两步，需在 `services/api` 引入 Excel 解析库并实现 preview/commit 两组接口。
