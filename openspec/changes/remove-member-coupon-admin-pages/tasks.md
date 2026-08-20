## 0. 门禁声明

```yaml
change: remove-member-coupon-admin-pages
gate_type: W2
ui_level_target: UI1
ui_level_actual: UI1
owner: worktree-remove-member-coupon-admin
worktree: .claude/worktrees/remove-member-coupon-admin
owned_paths:
  - apps/web-admin/**
  - openspec/changes/remove-member-coupon-admin-pages/**
base_sha: bc40b9b61124ff45df40b219bada513a4c10e8b4
candidate_sha: external-post-commit
external_assets: none（UI1 以浏览器实际运行取得，见 4.5；仓库内仍无可提交的自动化 runner）
dependencies:
  - remove-member-coupon-capability（已集成于 base_sha，无 owned paths 交集）
```

**关于 UI1（规划判断已被推翻，如实记录）**：规划阶段依据 `docs/quality/change-quality-gates.md` 中「当前没有锁定的浏览器/微信 runner」一句，把 UI1 记为 `BLOCKED_EXTERNAL`。该判断下早了——门禁对 UI1 的定义是「浏览器或非真实平台模拟器**实际运行**主场景与错误态」，缺少可提交的自动化 runner 只意味着没有可复现的 PASS 命令，不等于取不到 UI1 证据。

实施阶段以本地静态服务器加浏览器实际运行候选树，取得了真实 UI1 主场景与错误态证据，见 4.5。`ui_level_actual` 因此为 **UI1**。仓库内仍无可提交的自动化 runner，该缺口不影响本 change 的 UI1 结论，但仍值得独立立项补齐以便后续 change 复用。

门禁命令：

```
node openspec/changes/remove-member-coupon-admin-pages/checks/check_admin_scope.js <tree-root>
```

脚本随 change 提交，只依赖 Node 标准库，可对任意树根复现。

## 1. Boundary and approval

- [x] 1.1 确定单一能力边界与验收判据。
  - Evidence: 单一 capability `web-admin-scope-conformance`；单一验收判据「会员券在 PC 后台无任何页面模块、脚本标签、导航分组、内存态、种子、契约与文案残留，且菜品/订单/分类/营业设置契约仍可调用」；owned paths 与 `remove-member-coupon-capability` 无交集。
- [x] 1.2 确定不顺手为 `apps/web-admin` 建 runner。
  - Evidence: 见 `design.md`。建 DOM 垫片属测试基础设施建设，与删除是两件可独立验收的事，`AGENTS.md` 禁止一个 change 承载两者。若补 runner 应独立立项。
- [x] 1.3 确定证据分层：数据层运行态 + 页面/导航/文案静态。
  - Evidence: `data/seed.js` 与 `data/api.js` 无 DOM 依赖，可在 Node `vm` 沙箱配 `window` 垫片真实执行；页面与导航依赖 DOM，只能静态断言。该边界写在门禁脚本文件头。

## 2. Red

- [x] 2.1 编写门禁并对 `base_sha` 树执行，取得可观察失败。
  - Red: `node checks/check_admin_scope.js <base-tree>` → `exit=1`，`ADMIN_SCOPE_GATE=FAIL`，7 项检查全部失败：`pages/levels.js still exists`、`index.html still loads pages/levels.js`、`app.js still declares the 会员与营销 nav group`、`pages/products.js still references an excluded capability`、`Seed.LEVELS still exported`、`deleteProduct` 因依赖 `s.coupons.forEach` 而崩、以及 32 处文案残留逐一指名。失败原因是目标行为缺失，非断言构造。

## 3. Green

- [x] 3.1 删除 4 个页面模块并从 `index.html` 移除脚本标签。
  - Evidence: `git rm` 移除 `pages/{levels,members,member-import,coupons}.js`（891 行）；`index.html` 78 → 74 行。
- [x] 3.2 清理导航、二期能力标记与内存态。
  - Evidence: `app.js` 195 → 183 行，删除「会员与营销」分组及其四条路由、`__store` 的四个键、`p2` 标记与顶栏「二期能力 · 不在一期合同范围」副标题文案。
- [x] 3.3 清理种子与契约层。
  - Evidence: `data/seed.js` 191 → 145 行（去掉 `LEVELS/MEMBERS/COUPONS/MY_COUPON_USED` 及含真实姓名与手机号的种子）；`data/api.js` 557 → 272 行（去掉五组契约，`deleteProduct` 去掉摘券联动与 `disabledCoupons` 返回值）。
- [x] 3.4 清理外溢文案。
  - Evidence: `pages/products.js` 删除确认文案与 toast 去掉摘券表述；`ui/drawer.js` 注释去掉已删页面名；`app.css` 去掉「二期能力标签」样式分节。
- [x] 3.5 重跑同一门禁至 Green。
  - Green: `node checks/check_admin_scope.js <candidate-tree>` → `parsed 15 javascript files`，`ADMIN_SCOPE_GATE=PASS`，`exit=0`。同一脚本、同一命令，仅目标树不同。

## 4. Refactor and writer gate

- [x] 4.1 门禁自身的两处缺陷已修复并如实记录。
  - Refactor: 首版门禁有两个 bug——异步检查（`deleteProduct` 返回 Promise）未被收集，导致 `ADMIN_SCOPE_GATE=PASS` 在未决检查完成前就打印；`vm` 沙箱缺 `setTimeout`，而 `data/api.js` 的 `ok()` 用它模拟网络往返。已改为收集 pending promise 并在 `Promise.all` 后判定，沙箱补入 `setTimeout/clearTimeout/Promise`。修复后对 `base_sha` 树复跑仍 `FAIL`，红线未被削弱。
  - 同时新增「全目录文案残留扫描」，覆盖 `apps/web-admin` 下全部 `.js`/`.html`/`.css`。该检查在实施中立即生效：抓到 `app.css` 第 471 行「二期能力标签」样式分节，逐文件人工检查会漏掉该位置。
- [x] 4.2 契约完好性与跨端回归。
  - Refactor: 门禁断言 `listProducts`/`deleteProduct`/`listOrders`/`listCategories`/`getSettings` 删除后仍为可调用函数，且 `deleteProduct` 真实执行后菜品被移除、返回值不含 `disabledCoupons`。15 个 JS 全部可解析。小程序端未被触碰，其 `npm test` 仍 `19/19`。
- [x] 4.3 owned-path 审计与 `git diff --check`。
  - Refactor: 审计输出 `changed=16 in_owned=16 outside=0`，`OWNED_PATH_AUDIT=PASS`；`git diff --cached --stat HEAD -- apps/wechat-miniprogram services openspec/specs docs` 为空；`git diff --cached --check` 无输出，`DIFF_CHECK=PASS`。净变化 16 files / 308 insertions / 1268 deletions。
- [x] 4.5 UI1：浏览器实际运行候选树的主场景与错误态。
  - UI1 主场景：以 `python3 -m http.server` 在 `127.0.0.1` 提供候选树的 `apps/`，浏览器打开 `web-admin/index.html`。工作台完整渲染（四张 KPI 卡、实时订单 6 行、今日待办、销量排行）；**侧边导航只剩「经营 / 菜品 / 门店」三组共 7 条路由，「会员与营销」分组已消失**；顶栏副标题为「今日经营概览与实时接单」，不再出现「二期能力 · 不在一期合同范围」。菜品管理页渲染 7 行菜品，页面文本不含「券」字样。
  - UI1 运行态断言（页内求值）：`__store` 键为 `store/aOrders/menu/cats/settings/layer` 六项，无 `levels/members/coupons/couponUsed`；`window.Api` 与 `window.Seed` 对 `/level|member|coupon/i` 的泄漏枚举均为空数组。
  - UI1 错误态：依次访问已删除的 `#/levels`、`#/members`、`#/members/import`、`#/coupons` 四条路由，全部**优雅回落到工作台**（`tb-title` 为「工作台」、内容区非空），`runtimeErrors` 为空数组。随后 `#/products`、`#/dashboard` 正常路由，说明回落未破坏路由表。
  - 控制台：整页重新加载后 `onlyErrors` 与全量读取均无任何消息。
  - 边界：该 UI1 为一次性人工浏览器操作，非可提交的自动化命令；未覆盖真机、微信开发者工具或真实支付，不声称 UI2/UI3。
- [x] 4.6 记录门禁证据与 candidate SHA。
  - Writer verdict: `{ gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: bc40b9b61124ff45df40b219bada513a4c10e8b4, candidate_sha: external-post-commit（见 5.1）, hard_blockers: 0, blocked_external: 0, unverified_boundary: UI1 为人工浏览器操作而非可提交 runner；未覆盖真机与微信环境；openspec CLI 缺失 }`。

## 5. Independent verification

- [x] 5.1 在干净 detached worktree 对精确 candidate SHA 只读验证。
  - Verify: `candidate_sha=d6098b3af59248d8e51d39a7ee545b8a3cd5ba99`，验证树 `git status --porcelain` 为空。候选树 `ADMIN_SCOPE_GATE=PASS`（`exit=0`，`parsed 15 javascript files`）；同一脚本对 `base_sha` 树 `exit=1`，红线仍成立。小程序端 `npm test` 仍 `19/19`，未受影响。diff 相对 base 为 17 files / 394 insertions / 1268 deletions，`OWNED_PATH=PASS`（`files=17 in_owned=17 outside=0`）。验证结束时验证树仍为 clean。
- [x] 5.2 记录 PASS/FAIL 与剩余外部边界。
  - Verdict: **PASS（W2 / UI1）**。门禁 Red→Green 双树成立，UI1 主场景与错误态经浏览器实际运行取得，见 4.5。
  - 剩余外部边界：① UI1 为一次性人工浏览器操作，仓库内仍无可提交的自动化 runner，后续 PC 端 change 需重复人工操作，建议独立立项补齐；② 未覆盖真机、微信开发者工具或真实支付，不声称 UI2/UI3；③ 仓库未安装 `openspec` CLI，strict 校验记 `BLOCKED_EXTERNAL`；④ 全局折扣率尚未实现，PC 后台当前不提供任何优惠配置入口。

## 6. 后续动作

- 为 `apps/web-admin` 引入可提交的浏览器或 DOM 级 runner（独立立项），使 UI1 从人工操作变为可复现命令。
- `strip-retired-catalog-fields`：依赖 `remove-member-coupon-capability`，两端均需处理，PC 端的 UI1 在 runner 就位前同样需人工浏览器操作。
- `remove-retired-entry-screens`：与本 change owned paths 不重叠。
- `feat/member-coupon` 分支废弃。
