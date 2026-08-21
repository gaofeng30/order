## Why

PRD §6.3 要求商品有「餐段可售：全天 / 午餐 / 晚餐，三选一」，§5.2 要求菜单按用户所选取餐时间的餐段过滤商品，后端已就绪 `products.meal_period` 列与 `meal_periods` 表（`000004`、`000005`、`000006`）。

但 **PC 菜品编辑表单没有这个字段，契约层也不接受它** —— 该字段当前无人能填，菜单按餐段过滤因此无从实现。

这个缺口是在核对「Excel 模板列与前端表单一一对应」时暴露的：§6.13.2 的模板有「餐段可售」列，而表单里没有对应项。补齐它是批量导入的前置条件。

## What Changes

- 两端商品种子补 `meal` 字段，七条商品各自赋值。
- PC 契约层：`saveProduct` 把 `meal` 视为必填并校验三选一；`patch` 透传该字段；批量调价路径同步透传。
- PC 契约层新增 `MEALS` 与 `MEAL_LABEL` 两个导出，作为取值集合与展示标签的**单一来源**。
- PC 菜品管理页：表格新增「餐段」列；编辑表单新增必填的「餐段可售」下拉，选项与标签均从契约渲染；提交时带上该字段。
- PRD 三处标注更新：§6.13 范围来源由「待客户确认」改为「项目负责人 2026-08-21 确认纳入一期」；§16.4 C4 标记为已确认；§6.13.1 增加 P0 原型例外，允许在 PC 契约层内做本地 `.xlsx` 解析。

## Capabilities

### Modified Capabilities

- `web-admin-scope-conformance`: 新增「每个商品都声明餐段可售」一条。

## Impact

- Owner：branch `worktree-meal-period`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/meal-period`。
- Owned paths：`apps/web-admin/**`、`apps/wechat-miniprogram/utils/data.js`、`openspec/changes/add-product-meal-period/**`、`docs/product/online-ordering-system-prd-0818.md`。
- Dependency：`simplify-staff-import-template`，已归档于 `base_sha`。
- Non-goals：不实现菜单按餐段过滤商品（用户端菜单来自 API，需后端在 catalog 响应中下发该字段）；不接后端（PC 仍为 mock，经项目负责人选择 B）；不建名单页与导入页。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：种子每条商品都有合法 `meal`；契约必填并校验；表格与表单从契约渲染并可提交；既有回归全部通过。
