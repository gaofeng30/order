## Why

PRD §6.13 定义了菜品与员工白名单两个批量导入页，PC 后台一个都没有。§3.5 要求 12 页，实际 9 页（含前两件补的菜品餐段与员工白名单）。

## What Changes

- 新增 `data/xlsx.js`：契约层内部的最小 `.xlsx` 读取器。手写 ZIP 解析（stored 与 deflate 两种压缩方式），deflate 走浏览器与 Node 均内置的 `DecompressionStream('deflate-raw')`，XML 用正则扫描。**不引第三方库** —— PC 后台是零构建静态站点。
- 契约层新增四个方法：`previewProductImport` / `commitProductImport` / `previewStaffImport` / `commitStaffImport`，以及 `MAX_IMPORT_ROWS`。预览不写数据、返回一次性令牌；提交按令牌幂等。
- 新增 `ui/import-flow.js`：三步流程外壳，两个导入页共用。
- 新增 `pages/product-import.js` 与 `pages/staff-import.js`，各自只声明模板列与契约方法。
- `app.js` 导航新增两项，`index.html` 注册四个新脚本，`app.css` 补样式。

## Capabilities

### Modified Capabilities

- `web-admin-scope-conformance`: 新增「导入走预览—提交两步流程」「菜品导入只新增不覆盖」「员工导入按手机号覆盖但不重置系统字段」三条。

## Impact

- Owner：branch `worktree-bulk-import`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/bulk-import`。
- Owned paths：`apps/web-admin/**`、`openspec/changes/add-bulk-import-pages/**`。
- Dependency：`add-staff-whitelist-page`（员工导入的主页面）与 `add-product-meal-period`（菜品模板的餐段列），均已归档于 `base_sha`。
- Non-goals：不实现服务端解析（PRD §6.13.1 的 P0 例外允许契约层本地解析）；不接后端；不建商户账号名单、财务与对账、支付待处理三页。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：读取器能解析 stored 与 deflate 两种 `.xlsx`；非 `.xlsx` 与缺必填列被拒；预览计数、行号、原因、未知列、将新建分类均正确且不写数据；提交幂等；菜品只新增不覆盖、分类只建一次；员工覆盖不重置系统字段且不重新启用已停用记录；两页从导航可达且不自行解析。
