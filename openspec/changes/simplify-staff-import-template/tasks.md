## 0. 门禁声明

```yaml
change: simplify-staff-import-template
gate_type: W0
ui_level_target: UI0
ui_level_actual: UI0
owner: worktree-staff-template
worktree: .claude/worktrees/staff-template
owned_paths:
  - docs/product/online-ordering-system-prd-0818.md
  - openspec/changes/simplify-staff-import-template/**
base_sha: 228c6a3
candidate_sha: external-post-commit
external_assets: none
dependencies:
  - specify-bulk-import（已归档于 base_sha）
```

门禁命令：`python3 openspec/changes/simplify-staff-import-template/checks/check_staff_template.py <tree-root>`

## 1. Boundary and approval

- [x] 1.1 确认收敛范围连带页面字段。
  - Evidence: 用户此前定下「Excel 字段与前端页面一一对应」。只收模板会留下「页面有字段、模板填不了」的不对应状态，因此 §6.4 的可填字段同步收敛为手机号与姓名两项。
- [x] 1.2 确认状态字段的处置。
  - Evidence: §6.4 要求「停用保留记录但暂停折扣，用于离职人员」，状态不能删；但停用是对已有记录的操作而非新建时填的数据，因此留在页面开关、不进模板。

## 2. Red

- [x] 2.1 编写门禁并对 `base_sha` 树执行。
  - Red: `STAFF_TEMPLATE=FAIL`（`exit=1`），报 §6.4 仍列出单位/部门/工号/备注、模板列不等于两列、缺「不得重新启用已停用记录」、`StaffWhitelist` 仍含 `org`/`dept`/`jobNo`/`remark`。

## 3. Green

- [x] 3.1 §6.4 可填字段收敛为两项，并显式区分三类非填写字段。
- [x] 3.2 §6.13.3 模板收敛为姓名、手机号两列，明写不得含其他列，未知列忽略并在预览提示。
- [x] 3.3 §6.13.3 补「覆盖更新 MUST 保留状态，导入不得重新启用已停用记录」。
- [x] 3.4 §15.6.5 `StaffWhitelist` 删除 `org`/`dept`/`jobNo`/`remark`；§16.5 建页要求同步。
- [x] 3.5 重跑门禁至 Green。
  - Green: `staff_template_cols=['姓名', '手机号']`，`STAFF_TEMPLATE=PASS`；同一脚本对 `base_sha` 树 `exit=1`。

## 4. Refactor and writer gate

- [x] 4.1 门禁按结构解析模板列，而非匹配字面。
  - Refactor: 从 §6.13.3 的表格用正则提取列名数组，断言恰为 `['姓名','手机号']`。这样多一列少一列或改序都会被点出，不依赖某个词是否出现——本轮之前已连续五次犯「断言词而非断言事实」，此处按结构断言。
- [x] 4.2 记录门禁证据与 candidate SHA。
  - Writer verdict: `{ gate_type: W0, ui_level_target: UI0, ui_level_actual: UI0, base_sha: 228c6a3, candidate_sha: external-post-commit（见 5.1）, hard_blockers: 0, unverified_boundary: §6.13 全节待客户确认；本 change 不产生运行行为 }`。

## 5. Independent verification

- [ ] 5.1 在干净 detached worktree 对精确 candidate SHA 只读验证。
- [ ] 5.2 记录 PASS/FAIL 与剩余外部边界。
