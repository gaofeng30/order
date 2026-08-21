## 0. 门禁声明

```yaml
change: add-product-meal-period
gate_type: W2
ui_level_target: UI1
ui_level_actual: UI1
owner: worktree-meal-period
worktree: .claude/worktrees/meal-period
owned_paths:
  - apps/web-admin/**
  - apps/wechat-miniprogram/utils/data.js
  - openspec/changes/add-product-meal-period/**
  - docs/product/online-ordering-system-prd-0818.md
base_sha: 27b09bb
candidate_sha: external-post-commit
external_assets: none
```

门禁命令：`node openspec/changes/add-product-meal-period/checks/check_meal_period.js <tree-root>`

## 1. Boundary and approval

- [x] 1.1 定位缺口来源。
  - Evidence: PRD §6.3 要求餐段可售三选一、§5.2 要求菜单按餐段过滤、后端已有 `products.meal_period` 与 `meal_periods` 表，但 PC 表单无该字段、契约不接受，字段无人能填。缺口由「模板列与表单一一对应」的核对暴露。
- [x] 1.2 确认前后端策略。
  - Evidence: 项目负责人选择 B —— 四个新页面先做 mock，后端就绪再接。本 change 因此只改 PC 契约层与页面，不接后端。
- [x] 1.3 更新 PRD 范围标注。
  - Evidence: §6.13 由「待客户确认」改为「项目负责人 2026-08-21 确认纳入一期」；§16.4 C4 标记已确认；§6.13.1 增 P0 原型例外，允许契约层内本地解析 `.xlsx`，明确不构成生产契约。标注为项目方确认而非客户确认——手上无客户就此事的书面记录。

## 2. Red

- [x] 2.1 编写门禁并对 `base_sha` 树执行。
  - Red: `MEAL_PERIOD_GATE=FAIL`（`exit=1`），三项失败：`p001.meal = undefined`、契约保存后丢失 `meal`、编辑表单无餐段控件。

## 3. Green

- [x] 3.1 两端种子补 `meal`。
  - Evidence: `apps/web-admin/data/seed.js` 与 `apps/wechat-miniprogram/utils/data.js` 各 7 条商品补齐。
- [x] 3.2 契约层必填校验与单一标签来源。
  - Evidence: `data/api.js` 新增 `MEALS`/`MEAL_LABEL` 并导出；`saveProduct` 校验三选一、`patch` 透传；批量调价路径同步透传；body 注释同步。
- [x] 3.3 页面渲染与提交。
  - Evidence: 表格新增「餐段」列（从 `Api.MEAL_LABEL` 渲染）；编辑表单新增必填下拉（选项从 `Api.MEALS` 渲染）并在提交时带上；页面副标题更新为「上下架、售罄、餐段与价格」。
- [x] 3.4 重跑门禁至 Green。
  - Green: `MEAL_PERIOD_GATE=PASS`；同一脚本对 `base_sha` 树 `exit=1`。

## 4. Refactor and writer gate

- [x] 4.1 修正一处断言位置错误。
  - Refactor: 门禁首版断言「`pages/products.js` 中出现『全天』」，但标签由契约层的 `MEAL_LABEL` 提供，页面从契约渲染，源码中自然没有该字面。**这是同一类错误的又一次——断言字面而非事实。** 已拆为两条：一条按运行态断言契约导出的取值集合与标签映射，一条断言页面从 `Api.MEALS` / `Api.MEAL_LABEL` 渲染并在提交时带上该字段。
- [x] 4.2 覆盖共用契约的连带影响。
  - Refactor: `saveProduct` 是批量调价复用的共用契约。新增必填字段后若调价路径不透传，整批调价会失败或抹掉 `meal`。已同步透传，门禁的契约用例覆盖缺失与非法两种拒绝路径。
- [x] 4.3 UI1：浏览器实际运行。
  - UI1 表格：列为 `菜品 / 分类 / 餐段 / 售价 / 状态 / 操作`；前四行餐段依次为 全天 / 午餐 / 全天 / 全天。
  - UI1 表单：字段为 `菜品图片 / 菜品名称 / 售价 / 餐段可售 * / 分类 * / 描述`；下拉选项为 `all=全天`、`lunch=午餐`、`dinner=晚餐`，当前值 `all`。
  - UI1 保存：改选晚餐后保存，`__store.menu[0].meal` 变为 `dinner`，Toast「已保存」。
  - UI1 校验：缺餐段与传 `中午` 均被拒，错误信息「请选择餐段可售」。
  - 控制台：`runtimeErrors` 为空数组。
- [x] 4.4 记录门禁证据与 candidate SHA。
  - Writer verdict: `{ gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 27b09bb, candidate_sha: external-post-commit（见 5.1）, hard_blockers: 0, unverified_boundary: 用户端菜单按餐段过滤未实现（需后端在 catalog 响应下发该字段）；PC 仍为 mock 未接后端 }`。

## 5. Independent verification

- [ ] 5.1 在干净 detached worktree 对精确 candidate SHA 只读验证。
- [ ] 5.2 记录 PASS/FAIL 与剩余外部边界。

## 6. 后续

按依赖，下两步为：员工折扣白名单页（含全局折扣率设置）→ 两个批量导入页（含契约层本地 `.xlsx` 解析）。
