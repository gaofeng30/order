## 0. 门禁声明

```yaml
change: remove-member-coupon-admin-pages
gate_type: W2
ui_level_target: UI1
ui_level_actual: UI0
ui_level_UI1: BLOCKED_EXTERNAL
owner: worktree-remove-member-coupon-admin
worktree: .claude/worktrees/remove-member-coupon-admin
owned_paths:
  - apps/web-admin/**
  - openspec/changes/remove-member-coupon-admin-pages/**
base_sha: bc40b9b61124ff45df40b219bada513a4c10e8b4
candidate_sha: external-post-commit
external_assets:
  - name: browser-or-dom-runner-for-web-admin
    owner: 开发方
    available: false
    recovery: 为 apps/web-admin 引入浏览器或 DOM 级 runner（独立立项）
dependencies:
  - remove-member-coupon-capability（已集成于 base_sha，无 owned paths 交集）
```

**W2 硬边界（必须先读）**：`docs/quality/change-quality-gates.md` 的决策表把 W2×UI0 标为硬阻断，同一文档又记载「当前没有锁定的浏览器/微信 runner；缺少 UI1 资产即 `BLOCKED_EXTERNAL`，没有当前 PASS 命令」。`apps/web-admin` 仓库内没有任何 runner 或 `package.json`。

因此本 change **取不到 UI1 PASS**，按文档规定记 `BLOCKED_EXTERNAL` 并写明恢复条件，不用静态结果冒充。是否在此状态下集成属 lane 决策，本 change 不单方面认定为已完成 W2 验收。

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
- [x] 4.4 记录门禁证据与 candidate SHA。
  - Writer verdict: `{ gate_type: W2, ui_level_target: UI1, ui_level_actual: UI0, ui_level_UI1: BLOCKED_EXTERNAL, base_sha: bc40b9b61124ff45df40b219bada513a4c10e8b4, candidate_sha: external-post-commit（见 5.1）, hard_blockers: 0, blocked_external: 1（web-admin 无浏览器/DOM runner）, unverified_boundary: 页面渲染与导航回退行为未经运行时验证；openspec CLI 缺失 }`。

## 5. Independent verification

- [ ] 5.1 在干净 detached worktree 对精确 candidate SHA 只读验证。
- [ ] 5.2 记录 PASS/FAIL 与剩余外部边界。

## 6. 后续动作

- 为 `apps/web-admin` 引入浏览器或 DOM 级 runner（独立立项）。补齐后本 change 的 UI1 `BLOCKED_EXTERNAL` 应重新评估。
- `strip-retired-catalog-fields`：依赖 `remove-member-coupon-capability`，两端均需处理，PC 端将面临同样的 UI1 边界。
- `remove-retired-entry-screens`：与本 change owned paths 不重叠。
- `feat/member-coupon` 分支废弃。
