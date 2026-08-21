## 0. 门禁声明

```yaml
change: add-staff-whitelist-page
gate_type: W2
ui_level_target: UI1
ui_level_actual: UI1
owner: worktree-staff-page
worktree: .claude/worktrees/staff-page
owned_paths:
  - apps/web-admin/**
  - openspec/changes/add-staff-whitelist-page/**
base_sha: 6b29f6b
candidate_sha: external-post-commit
external_assets: none
dependencies:
  - add-product-meal-period（已归档于 base_sha）
```

门禁命令：`node openspec/changes/add-staff-whitelist-page/checks/check_staff_page.js <tree-root>`

## 1. Boundary and approval

- [x] 1.1 定位缺口。
  - Evidence: PRD §6.4 要求该页，§4.1 双要素与 §5.6 算价链都依赖它，但 PC 侧边导航只有三组、契约层对 `staff` / `whitelist` 零命中。原「会员名单」页删除的是会员券体系，员工白名单一直没补建。
- [x] 1.2 确定折扣率的表示。
  - Evidence: 见 `design.md`。存储与校验用整数百分比 1–100 表示员工实付比例，界面写「员工实付 N %」并给实时的「约 N 折」推导，避免 `discountRate` 被读成减免比例。

## 2. Red

- [x] 2.1 编写门禁并对 `base_sha` 树执行。
  - Red: `STAFF_PAGE_GATE=FAIL`（`exit=1`），六项全失败：无 `STAFF_WHITELIST` 种子、契约缺 `listStaff` 等六个方法、`pages/staff.js` 不存在。

## 3. Green

- [x] 3.1 种子与配置。
  - Evidence: `data/seed.js` 新增 `STAFF_WHITELIST`（6 条，含一条停用记录用于验证状态），字段严格为两个可填项加五个系统字段；`SETTINGS` 新增 `discountRate: 85`。
- [x] 3.2 契约层六个方法。
  - Evidence: `data/api.js` 282 → 356 行。`saveStaff` 校验手机号非空、格式（`/^1[3-9]\d{9}$/`）、姓名非空、唯一性；编辑分支只覆盖 `phone` 与 `name`；新增分支写入默认值。`saveDiscountRate` 校验整数与 1–100 范围。
- [x] 3.3 页面与导航。
  - Evidence: 新增 `pages/staff.js`（折扣率卡片 + 八列表格 + 搜索 + 行内启停 + 编辑抽屉 + 删除二次确认）；`app.js` 新增「名单」分组与 `staff` 路由并初始化内存态；`index.html` 注册脚本；`app.css` 补样式。
- [x] 3.4 重跑门禁至 Green。
  - Green: `STAFF_PAGE_GATE=PASS`（`parsed 16 javascript files`）；同一脚本对 `base_sha` 树 `exit=1`。

## 4. Refactor and writer gate

- [x] 4.1 覆盖「更新时静默丢系统字段」这一缺陷类。
  - Refactor: 门禁专设一条用例，先对某条记录的 `joinAt` / `bound` / `spend` / `orders` 取快照，编辑姓名后逐项比对，再切换状态后再比对一次。契约层因此采用 `Object.assign({}, 既有, { phone, name })` 而非整条替换。这类缺陷不写断言基本发现不了。
- [x] 4.2 UI1：浏览器实际运行。
  - UI1 渲染：侧边栏新增「名单」分组，页面标题「员工折扣白名单 · 全局折扣率与员工名单」；表格八列 `姓名 / 手机号 / 状态 / 已绑定 / 加入时间 / 累计消费 / 累计单量 / 操作`，6 行种子含一条停用记录。
  - UI1 折扣率：输入 90 时提示实时变为「约 9 折」；保存后 `__store.settings.discountRate` 为 `90`，Toast「折扣率已保存 · 员工实付 90%」。
  - UI1 校验：`saveDiscountRate(0)` → 「折扣率需在 1 到 100 之间」；`saveDiscountRate(85.5)` → 「折扣率必须是整数百分比」；`saveStaff` 重复手机号 → 「手机号 13800006620 已在名单中（林建国）」；格式错 → 「手机号格式不正确」。
  - UI1 状态切换：停用首行后 `enabled` 为 `false`，而 `joinAt` `2026-03-12`、`bound` `true`、`spend` `1286` 均未变。
  - UI1 搜索：输入「黄」后表格只剩「黄映雪」一行。
  - 控制台：`runtimeErrors` 为空数组。
- [x] 4.3 记录门禁证据与 candidate SHA。
  - Writer verdict: `{ gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 6b29f6b, candidate_sha: external-post-commit（见 5.1）, hard_blockers: 0, unverified_boundary: 折扣率对下单算价的实际作用未实现（属后端）；PC 未接后端；商户账号名单页与两个导入页仍未建 }`。

## 5. Independent verification

- [ ] 5.1 在干净 detached worktree 对精确 candidate SHA 只读验证。
- [ ] 5.2 记录 PASS/FAIL 与剩余外部边界。

## 6. 后续

第三、四件：菜品批量导入页与员工白名单批量导入页（§6.13，含契约层本地 `.xlsx` 解析）。另有 §4.4 商户账号名单页、§6.11 财务与对账、§7.3 支付待处理三页仍未建。
