## 0. 门禁声明

```yaml
change: add-bulk-import-pages
gate_type: W2
ui_level_target: UI1
ui_level_actual: UI1
owner: worktree-bulk-import
worktree: .claude/worktrees/bulk-import
owned_paths:
  - apps/web-admin/**
  - openspec/changes/add-bulk-import-pages/**
base_sha: 7f082ea
candidate_sha: external-post-commit
external_assets: none
dependencies:
  - add-product-meal-period（已归档于 base_sha）
  - add-staff-whitelist-page（已归档于 base_sha）
```

门禁命令：`node openspec/changes/add-bulk-import-pages/checks/check_bulk_import.js <tree-root>`

## 1. Boundary and approval

- [x] 1.1 确定解析实现方式。
  - Evidence: PC 后台零构建，引 SheetJS 需塞几百 KB vendor。`.xlsx` 是 ZIP + XML，deflate 有浏览器与 Node 内置的 `DecompressionStream('deflate-raw')`，因此手写约 120 行读取器，限制写在文件头。
- [x] 1.2 确定解析位置符合 PRD。
  - Evidence: §6.13.1 要求服务端解析、页面不得自行解析。后端未就位，走该节标注的 P0 原型例外：实现放在契约层 `data/api.js` 内部，页面只调契约方法。门禁断言两个页面源码不含 `Xlsx.readRows` 与 `DecompressionStream`。

## 2. Red

- [x] 2.1 编写门禁与 `.xlsx` 夹具构造器，对 `base_sha` 树执行。
  - Red: `BULK_IMPORT_GATE=FAIL`（`exit=1`），九项全失败：`data/xlsx.js` 不存在、契约缺四个方法、两个页面文件不存在。
  - 夹具构造器手写 ZIP（CRC32 + 本地头 + 中央目录 + EOCD），用 `zlib.deflateRawSync` 产生 deflate 条目并可切换为 stored，覆盖两条压缩路径。

## 3. Green

- [x] 3.1 最小 `.xlsx` 读取器。
  - Evidence: `data/xlsx.js`，ZIP 中央目录解析 + `DecompressionStream('deflate-raw')` + 正则 XML 扫描，支持 sharedStrings、inlineStr 与数值单元格，按 `r` 属性还原行列位置。
- [x] 3.2 契约层四个导入方法。
  - Evidence: `data/api.js` 新增 `previewProductImport` / `commitProductImport` / `previewStaffImport` / `commitStaffImport` 与 `MAX_IMPORT_ROWS`。`readSheet` 统一处理扩展名、表头匹配、必填列、未知列与行数上限；`stashPreview` / `takeCommit` 承担令牌与幂等。
- [x] 3.3 三步流程外壳与两个页面。
  - Evidence: `ui/import-flow.js` 共用外壳（模板列展示、选文件、计数、异常折叠、跳过异常提交、完成后回列表）；`pages/product-import.js` 与 `pages/staff-import.js` 各自只声明模板列、说明与契约方法。
- [x] 3.4 注册与样式。
  - Evidence: `index.html` 注册 `data/xlsx.js`、`ui/import-flow.js` 与两个页面；`app.js` 导航在「菜品」组加菜品批量导入、在「名单」组加员工批量导入；`app.css` 补样式。
- [x] 3.5 重跑门禁至 Green。
  - Green: `BULK_IMPORT_GATE=PASS`（`parsed 20 javascript files`）；同一脚本对 `base_sha` 树 `exit=1`。

## 4. Refactor and writer gate

- [x] 4.1 修正跨 vm realm 的结构比较。
  - Refactor: 门禁用 `assert.deepEqual` 比较沙箱内构造的数组时报「same structure but not reference-equal」。这与 `adopt-six-state-order-lifecycle` 那次是同一根因（跨 realm 原型不同）。已统一改为 `JSON.stringify` 比较，并在门禁文件头写明这条，避免下次再踩。
  - 排查时先打印了契约的实际输出，确认行号与忽略列本来就是对的 —— 失败来自断言方式而非实现。
- [x] 4.2 UI1：浏览器实际运行，上传真 `.xlsx`。
  - **菜品导入**：上传含 5 行数据的夹具（2 条新增 + 1 条同名 + 1 条售价非数值 + 1 条餐段非法）。页面显示「新增 2 条 · 更新 0 条 · 异常 3 条」；「本次将新建分类：夏日凉菜」；「图片不在模板中……请随后逐个补图」；异常三条分别为「第 4 行 菜品「商务双拼饭」已存在，导入只新增不覆盖；改价请用菜品管理的批量调价」「第 5 行 售价「三十二」不是大于 0 的数值」「第 6 行 餐段可售「中午」不是全天 / 午餐 / 晚餐之一」。
  - 点「跳过异常行，导入 2 条」后：菜品 7 → 9；分类 5 → 6 且「夏日凉菜」只出现一次；新菜 `meal='lunch'`、`price=18`、`status='on'`、`imgs.length=0`；**既有「商务双拼饭」的售价 32 与描述均未被覆盖**；自动跳回菜品管理；Toast「导入完成 · 新增 2 条」。
  - **员工导入**：上传含未知列「工号」、一条覆盖已停用员工、两条异常的夹具。页面显示「新增 1 条 · 更新 1 条 · 异常 2 条」；「已忽略未知列：工号」；异常为「第 4 行 手机号必填」「第 5 行 手机号 13900001111 在本文件中重复（第 2 行已出现）」。
  - 提交后：名单 6 → 7；**已停用的「孙丽萍」`enabled` 仍为 `false`，`joinAt` 2026-01-22 与 `spend` 214 均未变**；自动跳回名单页；Toast「导入完成 · 新增 1 条 · 更新 1 条」。
  - 控制台：两次流程 `runtimeErrors` 均为空数组。
- [x] 4.3 记录门禁证据与 candidate SHA。
  - Writer verdict: `{ gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 7f082ea, candidate_sha: external-post-commit（见 5.1）, hard_blockers: 0, unverified_boundary: 解析为 P0 原型例外，生产须移至服务端；单次行数上限 500 待 §16.3 P6 拍板；PC 未接后端 }`。

## 5. Independent verification

- [x] 5.1 在干净 detached worktree 对精确 candidate SHA 只读验证。
  - Verify: `candidate_sha=f39752d21f1f662a8372ee1c1b82e787487ad043`，验证树 clean。`BULK_IMPORT_GATE=PASS`（`parsed 20 javascript files`），同一脚本对 `base_sha` 树 `exit=1`；`STAFF_PAGE_GATE` / `MEAL_PERIOD_GATE` / `PICKUP_SETTINGS_GATE` / `ORDER_LIFECYCLE_GATE` / `ADMIN_SCOPE_GATE` / `CATALOG_FIELDS_GATE` 全部 PASS；小程序 `npm test` 65/65。diff 相对 base 为 15 files / 946 insertions / 0 deletions，全部在 owned paths 内。
- [x] 5.2 记录 PASS/FAIL 与剩余外部边界。
  - Verdict: **PASS（W2 / UI1）**。
  - 剩余外部边界：① **解析为 PRD §6.13.1 标注的 P0 原型例外**，生产必须移至服务端，后端就位时契约层实现整体替换并从 PRD 移除该例外；② 自写读取器不支持加密、宏与多工作表选择，固定读第一张表，限制写在 `data/xlsx.js` 文件头；③ 单次行数上限 500 为暂定值，§16.3 P6 待拍板；④ PC 后台仍为 mock，零网络调用；⑤ PRD §3.5 仍缺商户账号名单、财务与对账、支付待处理三页；⑥ UI1 为人工浏览器操作，未覆盖真机与真实 Excel 生成的复杂文件（如合并单元格、多工作表）。

## 6. 后续

PRD §3.5 仍缺三页：商户账号名单（§4.4，启动路由判定与 PC 扫码登录的前提）、财务与对账（§6.11）、支付待处理（§7.3）。另有 PC 后台整体接后端一事未启动。
