## 0. 门禁声明

```yaml
change: remove-member-coupon-capability
gate_type: W2
ui_level_target: UI1
ui_level_actual: UI1
owner: worktree-remove-member-coupon
worktree: .claude/worktrees/remove-member-coupon
owned_paths:
  - apps/wechat-miniprogram/**
  - openspec/changes/remove-member-coupon-capability/**
base_sha: e451f527a00c4203ab2297282aac13623cf9045f
candidate_sha: external-post-commit
external_assets: none
dependencies:
  - 生效 spec mvp-product-baseline 的一期范围与定价 requirement（已集成于 base_sha）
```

W2 依据：删除用户可见页面、结算页优惠明细卡与个人中心入口，属用户可见 UI 行为变更。按质量门禁决策表 W2 最低 UI1。

UI1 证据来源：`apps/wechat-miniprogram` 的 Node 测试 harness（`tests/page-harness.js`），非微信开发者工具或真机，因此不声称 UI2/UI3。

工具边界：仓库未安装 `openspec` CLI，`openspec validate --strict` 无法执行，记 `BLOCKED_EXTERNAL`。

门禁命令：

```
cd apps/wechat-miniprogram && npm test
```

## 1. Boundary and approval

- [x] 1.1 读取生效 spec、评审记录、0818 PRD 与质量门禁，确定单一能力边界与验收判据。
  - Evidence: 单一 capability `miniprogram-scope-conformance`；单一验收判据「会员券在小程序端无任何页面、路由、组件、全局态、种子数据与接口契约残留，且结算链路仍可走通」；owned paths 为小程序应用与本 change 目录。
- [x] 1.2 按 UI1 证据可得性拆分前端删除工作。
  - Evidence: 小程序端有 Node harness（基线 13 项通过），可取真实 UI1；`apps/web-admin` 仓库内无任何 runner 与 `package.json`，只能取静态解析级证据。W2 硬阻断 UI0，故 PC 后台的同类删除拆为 `remove-member-coupon-admin-pages`，其 UI1 记 `BLOCKED_EXTERNAL`，不与本 change 混同。经用户确认采用该拆法。
- [x] 1.3 确定等级折扣与优惠券一并删除，不留占位。
  - Evidence: 见 `design.md`。生效 spec 的机制是全局单一折扣率，与 `Level` 表驱动的等级折扣是不同数据模型；保留占位同时违反 spec 的「不得预留」与 `AGENTS.md` 的「禁止过渡双轨」。经用户确认。
- [x] 1.4 声明与 `strip-retired-catalog-fields` 的路径冲突处置。
  - Evidence: 两者都需要 `utils/api.js` 与 `utils/data.js`。本 change 先取得所有权，后者声明依赖，不并行。

## 2. Red

- [x] 2.1 新增 UI1 用例 `tests/scope-conformance-ui1.test.js`，逐项断言排除能力在小程序端不存在，取得可观察失败。
  - Red: `node --test tests/scope-conformance-ui1.test.js` → `tests 6 / pass 0 / fail 6`。六项分别因路由与页面目录存在、`utils/promo.js` 与 `components/coupon-card` 存在、`globalData` 含 `levels/members/coupons/couponUsed`、`data.js` 导出 `LEVELS/MEMBERS/COUPONS/MY_COUPON_USED` 且 `api.js` 导出会员券方法、结算页含 `couponId/calc/isMember` 且订单固化优惠字段、模板含选券入口而失败。断言逐项指名残留位置，不是快照对比。

## 3. Green

- [x] 3.1 删除 7 个会员券页面、`utils/promo.js`、`components/coupon-card`，并从 `app.json` 移除路由与全局组件。
  - Evidence: `app.json` pages 27 → 20，`usingComponents` 去掉 `coupon-card` 后剩 10 个；7 个页面目录与 2 个模块经 `git rm` 移除。
- [x] 3.2 清理全局态、种子数据与接口契约。
  - Evidence: `app.js` 83 → 73 行（去掉 `coupon` 与四个会员券全局键及其初始化）；`utils/data.js` 207 → 153 行（去掉 `LEVELS/MEMBERS/COUPONS/MY_COUPON_USED` 及其含真实姓名与手机号的种子）；`utils/api.js` 394 → 94 行（去掉会员等级、会员名单、名单导入、优惠券、用户卡包五组契约，并去掉 `deleteProduct` 的摘券联动），文件头注释由「会员等级 / 会员名单 / 优惠券 / 菜品，二期能力」改为「菜品」。
- [x] 3.3 结算页改为无优惠机制路径。
  - Evidence: `confirm.js` 去掉 `promo` 与 `api` 引用、`loadPromo/recalc/openCoupon/closeCoupon/pickCoupon/confirmCoupon`，改为 `refreshItems` 直接由 `cart.totalCents()` 计算 `subtotal_cents/subtotal_text/payable_cents/payable_text`；订单去掉 `levelName/levelLabel/levelCut/couponName/couponCut/totalCut` 与券计次。`confirm.wxml` 181 → 113 行（优惠明细卡收敛为单行商品小计，删除整个选券弹层）；`confirm.wxss` 119 → 82 行；`confirm.json` 去掉 `coupon-card`。
- [x] 3.4 清理个人中心、商户中心与菜品编辑的外溢引用。
  - Evidence: `profile.js` 42 → 21 行、`profile.wxml` 70 → 61 行（去掉券入口、等级胶囊与会员文案）；`admin-profile.js` 18 → 15 行、`admin-profile.wxml` 92 → 70 行（去掉「会员与营销」整组及其「二期能力」胶囊）；`admin-product-edit.js` 删除确认文案与 toast 去掉摘券表述。
- [x] 3.5 重跑同一门禁至 Green。
  - Green: `node --test tests/scope-conformance-ui1.test.js` → `tests 6 / pass 6 / fail 0`。同一用例文件、同一命令。

## 4. Refactor and writer gate

- [x] 4.1 修复既有 UI1 回归，改写与删除范围一一对应。
  - Refactor: 删除动作打破 2 项既有用例。`cart snapshot drives confirm, existing all-scope promo and mock pay` 改名为 `subtotal-only pricing`，券过滤改为断言 `globalData.coupons` 不存在，`isMember/calc.usable/openCoupon/cpVisible` 四条断言换为 `subtotal_cents/subtotal_text/payable_cents/payable_text` 与 `openCoupon` 未定义，并**新增**订单 `total`/`subtotal` 断言。`WXML exposes exact recoverable states` 中 `assert.match(confirmWXML, /bindtap="openCoupon"/)` 改为 `doesNotMatch`，并**新增** `{{subtotal_text}}` 与 `<money v="{{payable_text}}"` 两条断言。改写只置换被删能力对应的断言，其余原样保留，断言总数未减少。
- [x] 4.2 全量 UI1 与静态检查。
  - Refactor: `npm test`（已把新用例挂入 test script）→ `tests 19 / pass 19 / fail 0`。43 个 JS 全部 `node --check` 通过；全部 JSON 解析通过；`app.json` 20 条路由对应的页面文件全部存在，10 个全局组件全部可解析；20 个页面级 `usingComponents` 无失效引用。
- [x] 4.3 owned-path 审计与 `git diff --check`。
  - Refactor: 审计输出 `changed=53 in_owned=53 outside=0`，`OWNED_PATH_AUDIT=PASS`；`git diff --cached --stat HEAD -- apps/web-admin services openspec/specs docs` 为空，PC 后台、后端、生效 spec 与产品文档均未改动；`git diff --cached --check` 无输出，`DIFF_CHECK=PASS`。净变化 53 files / 333 insertions / 2819 deletions。
- [x] 4.4 记录 W2/UI1 证据与 candidate SHA，checkpoint `CANDIDATE`。
  - Writer verdict: `{ gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: e451f527a00c4203ab2297282aac13623cf9045f, candidate_sha: external-post-commit（见 5.1）, hard_blockers: 0, unverified_boundary: openspec CLI 缺失；UI1 来自 Node harness 而非微信开发者工具或真机，不声称 UI2/UI3；PC 后台同类删除未做 }`。

## 5. Independent verification

- [ ] 5.1 在干净 detached worktree 对精确 candidate SHA 只读验证。
- [ ] 5.2 记录 PASS/FAIL 与剩余外部边界。

## 6. 后续动作

- `remove-member-coupon-admin-pages`：PC 后台的同类删除，UI1 受 runner 缺失阻塞。
- `strip-retired-catalog-fields`：依赖本 change（共享 `utils/api.js`、`utils/data.js`）。
- 全局折扣率的实现属新增类 change；在其完成前，结算页不展示任何优惠。
- `feat/member-coupon` 分支废弃：做的正是本 change 删除的能力，需单独处理。
