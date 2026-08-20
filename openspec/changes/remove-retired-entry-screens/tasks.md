## 0. 门禁声明

```yaml
change: remove-retired-entry-screens
gate_type: W2
ui_level_target: UI1
ui_level_actual: UI1
owner: worktree-remove-retired-entry-screens
worktree: .claude/worktrees/remove-retired-entry-screens
owned_paths:
  - apps/wechat-miniprogram/**
  - openspec/changes/remove-retired-entry-screens/**
base_sha: cc484df89a3079272f8fd32ce01b976f2bc0801f
candidate_sha: external-post-commit
external_assets: none
dependencies:
  - remove-member-coupon-capability（已集成于 base_sha，共享 apps/wechat-miniprogram/**，串行）
```

W2 依据：删除用户与商户可见页面、首页营销位与 TabBar 项，属用户可见 UI 行为变更。

门禁命令：

```
cd apps/wechat-miniprogram && npm test
```

工具边界：仓库未安装 `openspec` CLI，strict 校验记 `BLOCKED_EXTERNAL`。UI1 来自 Node 测试 harness，非微信开发者工具或真机，不声称 UI2/UI3。

## 1. Boundary and approval

- [x] 1.1 确定单一能力边界与验收判据。
  - Evidence: 验收判据「品牌页与工作台在路由与磁盘上均不存在；首页无营销位且不请求 catalog；商户 TabBar 三格且商户中心可达；无指向已删页面的失效引用」。
- [x] 1.2 与 `strip-retired-catalog-fields` 的冲突处置。
  - Evidence: 两者在 `utils/data.js`、`utils/util.js`、`pages/admin-dashboard/*`、`tests/*` 四处重叠。本 change 先执行并删除 `admin-dashboard` 整页，后者的重叠面因此缩小；经用户确认串行执行，不并行。
- [x] 1.3 发现并处置跨 spec 冲突。
  - Evidence: 生效 spec `miniprogram-menu-catalog` 的 `Home and menu expose complete recoverable list states` 要求「首页招牌 MUST 按 server category 顺序再按各自 product 顺序 flatten 后取前四件」，与客户评审 §36 批注「都删掉」直接冲突。该 requirement 成文于 2026-08-13，早于评审；评审记录按 authority order 属第 1 顺位。因此本 change 带一条 MODIFIED delta，理由与「为何不单开 spec change」见 `design.md`。

## 2. Red

- [x] 2.1 新增 UI1 用例 `tests/entry-screens-ui1.test.js`，逐项断言已废止入口屏与营销位不存在，并断言可达性。
  - Red: `node --test tests/entry-screens-ui1.test.js` → `tests 7 / pass 1 / fail 6`。六项分别因路由与目录存在、首页仍有 Banner/入群/招牌、宫格仍含会员中心与联系客服、TabBar 仍为五格、商户中心在收敛后不可达、`RANK` 仍被导出而失败。第 7 条为回归保护用例，此时即应通过。

## 3. Green

- [x] 3.1 删除品牌选择页与商户端工作台，收敛路由。
  - Evidence: `git rm` 移除 `pages/brand`、`pages/admin-dashboard`；`app.json` pages 20 → 18；`utils/util.js` 去掉 `toBrand`。
- [x] 3.2 首页清空营销位并停止请求 catalog。
  - Evidence: `home.js` 65 → 32 行（去掉 `CAMPAIGNS`、`bannerIdx`、`onBanner`、`dotTap`、`openBanner`、`toDetail`、`signature`、`listState`、`loadCatalog`、`retryCatalog`，宫格去掉 `member` 与 `service`）；`home.wxml` 131 → 51 行；`home.wxss` 66 → 29 行。
- [x] 3.3 商户 TabBar 收敛为三格并保持商户中心可达。
  - Evidence: `components/tabbar/tabbar.js` 商户端分支由五格改为订单/核销/菜品；`admin-orders.wxml` 导航栏右侧 slot 补「商户中心」入口，`admin-orders.js` 补 `toProfile`。
- [x] 3.4 删除仅由工作台使用的种子。
  - Evidence: `utils/data.js` 去掉 `RANK` 及其导出，153 → 149 行。
- [x] 3.5 建立两份 spec delta。
  - Evidence: `miniprogram-scope-conformance` 追加 4 条 requirement / 5 个 scenario；`miniprogram-menu-catalog` 1 条 MODIFIED，主语由「首页与菜单」收敛为「菜单」，新增首页 `MUST NOT` 请求 catalog，菜单侧义务一条未减。
- [x] 3.6 重跑同一门禁至 Green。
  - Green: `node --test tests/entry-screens-ui1.test.js` → 9/9 通过（含 Refactor 阶段新增的 2 条）。

## 4. Refactor and writer gate

- [x] 4.1 修正一条基于错误假设写的 Red 断言，并如实记录。
  - Refactor: Red 阶段我写了 `home still loads the catalog after the marketing rewrite`，假设首页删掉招牌后仍应请求 catalog。读 `miniprogram-menu-catalog` 生效 spec 后确认该假设错误——首页不应再有任何 list 义务。该断言改写为 `home no longer touches the catalog`，并追加 `listState` 与 `retryCatalog` 缺失断言。这是纠正错误前提，不是为了让实现通过而放宽。
- [x] 4.2 修复删除动作造成的两处失效引用，并各补一条断言。
  - Refactor: 残留扫描发现 `launch.wxml` 的商户端入口 `data-to="admin-dashboard"` 指向已删页面（商户点击即断路），改指 `admin-orders`；`admin-layer` 以已删的品牌页为预览底图之一，收敛为只预览身份选择页，并改掉「同时展示在业务选择页与身份选择页」的说明文案。新增 `the identity screen routes merchants to a page that exists` 与 `the launch-layer editor previews only screens that exist` 两条用例防回归。
- [x] 4.3 改写既有 UI1 中覆盖首页 catalog 生命周期的用例。
  - Refactor: 6 条既有用例因首页不再请求 catalog 而失败。逐条改写为菜单侧等强度断言：`legacy behavior boundary: list lifecycle sends a catalog request` 与 `network failure is retryable` 的目标页改为菜单；`home list lifecycle...` 改名为 `menu list lifecycle covers loading, error, retry, ready and server order` 并把「前四件」断言换为菜单按 server 顺序 flatten 的全部五件；`non-200 2xx` 与 `list empty` 两条中的首页片段改为菜单（其中一处与既有菜单段重复，已删重复而非保留两份）；`WXML exposes...` 中首页的 list-state 断言改为 `doesNotMatch(homeWXML, /listState|retryCatalog|signature/)`。**并新增** `home performs no catalog work at all` 一条。断言总数未减少。
- [x] 4.4 全量 UI1 与静态检查。
  - Refactor: 新用例挂入 `npm test` 后 `tests 29 / pass 29 / fail 0`。42 个 JS 全部 `node --check` 通过；`app.json` 18 条路由对应页面文件全部存在，10 个全局组件与 18 个页面级组件引用均有效；全端对 `pages/brand`、`admin-dashboard`、`'brand'` 的扫描无命中。
- [x] 4.5 owned-path 审计与 `git diff --check`。
  - Refactor: `changed=27 outside=0`，`OWNED_PATH=PASS`；`git diff --cached --stat HEAD -- apps/web-admin services openspec/specs docs` 为空；`git diff --check` 初次报 `home.wxss:29 new blank line at EOF`，已修，复跑 `DIFF_CHECK=PASS`。净变化 27 files / 220 insertions / 628 deletions。
- [x] 4.6 记录门禁证据与 candidate SHA。
  - Writer verdict: `{ gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: cc484df89a3079272f8fd32ce01b976f2bc0801f, candidate_sha: external-post-commit（见 5.1）, hard_blockers: 0, unverified_boundary: UI1 来自 Node harness，未覆盖微信开发者工具与真机；openspec CLI 缺失 }`。

## 5. Independent verification

- [x] 5.1 在干净 detached worktree 对精确 candidate SHA 只读验证。
  - Verify: `candidate_sha=91bb8d9f7e533a3be5663830908a7a00c7fdaa6d`，验证树 `git status --porcelain` 为空。`npm test` → `tests 29 / pass 29 / fail 0`；对 `pages/brand`、`admin-dashboard`、`入群`、`今日招牌`、`CAMPAIGNS`、`signature`、`RANK` 的全目录扫描零命中；`app.json` 18 条路由对应页面全部存在，页面级组件引用无失效。diff 相对 base 为 30 files / 406 insertions / 628 deletions，`OWNED_PATH=PASS`（`files=30 outside=0`）。验证结束时验证树仍为 clean。
  - 交叉复核：`mvp-product-baseline` 的残留门禁仍只有那一处已记录的良性误判（`Matrix cites a retired dimension` 的 WHEN 子句枚举被禁术语），本 change 未新增残留。
- [x] 5.2 记录 PASS/FAIL 与剩余外部边界。
  - Verdict: **PASS（W2 / UI1）**。
  - 剩余外部边界：① UI1 来自 `apps/wechat-miniprogram` 的 Node 测试 harness，未覆盖微信开发者工具、体验版或真机，不声称 UI2/UI3；② 仓库未安装 `openspec` CLI，strict 校验记 `BLOCKED_EXTERNAL`；③ 首页「到店点单」入口仍在，即时单能力尚未删除（声明为非目标）；④ 商户端营业设置、开屏图层与分类管理仍在小程序端，迁 PC 受 0818 PRD §16.3 P1 阻塞。

## 6. 集成与后续

- 两份 delta MUST 与本 change 一并 archive 后应用到生效 spec。
- 后续：`strip-retired-catalog-fields`（标签/过敏原/月售/库存位，两端）；即时单删除（含首页「到店点单」入口与确认页取餐方式切换）；商户端功能迁 PC（受 0818 PRD §16.3 P1 阻塞）；`feat/member-coupon` 分支废弃。
