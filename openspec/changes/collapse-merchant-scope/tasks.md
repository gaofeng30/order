## 0. 门禁声明

```yaml
change: collapse-merchant-scope
gate_type: W2
ui_level_target: UI1
ui_level_actual: UI1
owner: worktree-merchant-scope
worktree: .claude/worktrees/merchant-scope
owned_paths:
  - apps/wechat-miniprogram/**
  - openspec/changes/collapse-merchant-scope/**
  - docs/product/online-ordering-system-prd-0818.md
base_sha: 63e161e487493930450d909a42bf6ca93c53a674
candidate_sha: external-post-commit
external_assets: none
dependencies:
  - reconcile-baseline-alignment（已归档于 base_sha）
```

门禁命令：`cd apps/wechat-miniprogram && npm test`

## 1. Boundary and approval

- [x] 1.1 确认三处 spec 不一致并定范围。
  - Evidence: ① 菜品页超出 §38 授权（上架新菜 / 编辑 / 上下架）；② 商户中心及其三个配置页超出 §26；③ `admin-profile` 的两个「建设中」占位从未实现。用户于 2026-08-21 确认全部处理，第 2 项按本文档建议裁决。
- [x] 1.2 裁决 0818 PRD §16.3 P1。
  - Evidence: 评审 §26 划归 PC 的是「完整商品配置、退款、名单和财务」，不含营业状态；现场临时休息是手机场景。裁决为保留在小程序订单页，其余迁 PC，商户端恰好收敛为 §3.5 的 4 屏。PRD §16.3 P1 已标记为已定。
- [x] 1.3 确认上架新菜与人员管理的实现位置。
  - Evidence: **上架新菜**在 PC `pages/products.js` 已实现（按钮 + 抽屉表单 + `Api.saveProduct`），删除小程序端不丢功能。**人员管理两端都没有**：小程序侧只是 toast 占位；PC 侧导航只有经营 / 菜品 / 门店三组七页，员工折扣白名单与商户账号名单均未建，契约层也无对应接口。该缺口记入本 change 的非目标与后续。

## 2. Red

- [x] 2.1 新增 `tests/merchant-scope-ui1.test.js`。
  - Red: `tests 6 / pass 1 / fail 5`。五项分别因商户端仍有 9 屏、菜品页仍暴露 `newProduct`/`edit`/`toggleShelf`、订单页无营业状态切换、订单页仍指向商户中心、全端仍引用已删页面与「建设中」占位而失败。第 3 条「售罄切换端到端可用」为回归保护，此时即应通过。

## 3. Green

- [x] 3.1 删除 5 个 PC-only 页面并收敛路由。
  - Evidence: `git rm` 移除 `admin-product-edit` / `admin-categories` / `admin-settings` / `admin-layer` / `admin-profile`；`app.json` pages 18 → 13，商户端 9 屏 → 4 屏。
- [x] 3.2 菜品页收敛为只有售罄切换。
  - Evidence: `admin-products.js` 49 → 38 行（移除 `edit` / `newProduct` / `toggleShelf`）；`.wxml` 60 → 46 行（移除上架新菜横幅、编辑入口、图片点击进编辑、上下架开关）；`.wxss` 移除对应样式。
- [x] 3.3 营业状态切换迁至订单页，导航栏改切换身份。
  - Evidence: `admin-orders.js` 新增 `BIZ` 常量、`biz`/`storeStatus` 数据、`setBiz` 与 `reset`，移除 `toProfile`；`.wxml` 顶部新增三段状态切换，导航栏右侧由「商户中心」改为「切换身份」；`.wxss` 补样式。
- [x] 3.4 移除本机存储的开屏图层与占位入口。
  - Evidence: `git rm` 移除 `utils/layer.js` 与 `components/layer-overlay`；`launch.wxml` 去掉 `<layer-overlay />` 与相关注释，`launch.json`、`app.json` 去掉组件注册；用户端 `profile` 移除「设置」占位（PRD §5.9 未定义）。
- [x] 3.5 重跑至 Green。
  - Green: `node --test tests/merchant-scope-ui1.test.js` → 6/6。

## 4. Refactor and writer gate

- [x] 4.1 新 lint 在提交前抓到我重犯的同一个错。
  - Refactor: 删除「上架新菜」样式时又一次用了按行正则，`.new-banner` 是多行规则，只删掉首行选择器留下孤儿规则体，`admin-products.wxss` 花括号变为 `{=23 }=24`。**上一轮刚加的 `lint_wx.py` 在本地就报了 FAIL**，未流到微信开发者工具。已删除孤儿体，`WX_LINT=PASS`。
- [x] 4.2 四条既有断言按新事实重写。
  - Refactor: 删页后有四条断言的对象不存在。逐条改为等价或更强形式——「菜品编辑页无库存字段」→「小程序端不存在菜品编辑页」（数量库存由 PC 门禁继续覆盖）；「商户中心保持可达」→「商户中心不存在且订单页提供切换身份」；「开屏图层编辑页只预览存在的页面」→「本机存储图层实现整体不存在」；排除能力在商户中心/编辑页的缺席改由页面不存在保证。均非放宽。
- [x] 4.3 裁决 P1 并更新 PRD。
  - Refactor: §16.3 P1 标记为已定并写明裁决依据；§16.5 的开屏图层条目由「需改服务端下发」更新为「本机实现已整体移除，待接口就位后重新接入」。
- [x] 4.4 全量回归。
  - Refactor: `npm test` → `tests 65 / pass 65 / fail 0`（新增 6 条，既有 59 条经改写后无回归）。`app.json` 13 条路由与 9 个全局组件、全部页面级组件引用均有效。`WX_LINT=PASS`。
- [x] 4.5 记录门禁证据与 candidate SHA。
  - Writer verdict: `{ gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 63e161e487493930450d909a42bf6ca93c53a674, candidate_sha: external-post-commit（见 5.1）, hard_blockers: 0, unverified_boundary: 用户端暂无开屏图层展示（待服务端接口）；PC 侧两份名单未建；UI1 来自 Node harness 与结构 lint，未覆盖微信开发者工具与真机 }`。

## 5. Independent verification

- [x] 5.1 在干净 detached worktree 对精确 candidate SHA 只读验证。
  - Verify: `candidate_sha=8b6118cb804cda73402526d59fbf09f7b8a4d459`，验证树 clean。小程序 `npm test` 65/65；商户端路由恰为 `admin-orders` / `admin-order-detail` / `admin-verify` / `admin-products` 四屏，总屏数 13 且无缺失文件、无失效组件引用；五个门禁全部 PASS（`BASELINE_SINGLE_SOURCE` / `PICKUP_SETTINGS_GATE` / `ORDER_LIFECYCLE_GATE` / `ADMIN_SCOPE_GATE` / `CATALOG_FIELDS_GATE`）；`go build ./...` 通过。diff 相对 base 为 46 files / 370 insertions / 1108 deletions，`OWNED=PASS`。
- [x] 5.2 记录 PASS/FAIL 与剩余外部边界。
  - Verdict: **PASS（W2 / UI1）**。
  - 剩余外部边界：① 用户端暂无开屏图层展示，待服务端下发接口就位后重新接入；② 已有设备上的陈旧图层文件不会被自动清理，但小程序不再读取该 storage key，图片不再渲染；③ PC 侧员工折扣白名单与商户账号名单均未建，是当前最大功能缺口；④ UI1 来自 Node harness 与 WXSS/WXML 结构 lint，未覆盖微信开发者工具与真机；⑤ 仓库未安装 `openspec` CLI，strict 校验记 `BLOCKED_EXTERNAL`。
  - **待与客户确认（不在已确认范围内）**：批量导入的格式与范围。PRD §6.4 的 CSV 导入继承自已删除的会员名单，客户从未就此表态；商户账号名单与菜品批量导入 PRD 中根本没有。用户已表示明日讨论。

## 6. 后续

- **PC 侧两份名单均未建，是当前最大功能缺口**：员工折扣白名单（PRD §6.4，全局折扣率的作用对象）与商户账号名单（PRD §4.4、评审 §32，启动路由判定与 PC 扫码登录的前提）。契约层亦无对应接口。建议作为下一个 change。
- 开屏图层的服务端下发接口就位后，重新接入用户端首页浮层与身份选择页展示。
- 仍未处理：`feat/member-coupon` 分支废弃；`apps/web-admin` 的可提交 runner。
