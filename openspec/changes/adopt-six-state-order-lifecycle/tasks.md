## 0. 门禁声明

```yaml
change: adopt-six-state-order-lifecycle
gate_type: W2
ui_level_target: UI1
ui_level_actual: UI1
owner: worktree-six-state-lifecycle
worktree: .claude/worktrees/six-state-lifecycle
owned_paths:
  - apps/wechat-miniprogram/**
  - apps/web-admin/**
  - openspec/changes/adopt-six-state-order-lifecycle/**
base_sha: d0e17d6417817f48833b82081173eb411dbccba0
candidate_sha: external-post-commit
external_assets: none
dependencies:
  - remove-member-coupon-capability / remove-member-coupon-admin-pages /
    remove-retired-entry-screens / strip-retired-catalog-fields（均已归档于 base_sha）
```

门禁命令：

```
cd apps/wechat-miniprogram && npm test
node openspec/changes/adopt-six-state-order-lifecycle/checks/check_order_lifecycle.js <tree-root>
```

## 1. Boundary and approval

- [x] 1.1 确认生效 spec 正被违反，确定单一验收判据。
  - Evidence: `mvp-product-baseline` 的六态 requirement 自 `realign-mvp-product-baseline` 起生效，两端实现仍为 `待制作 → 待取餐 → 已完成` 三态且带撤销。验收判据「两端状态语义、可推进转换、泳道与筛选段均为六态口径；无任何撤销入口或方法；即时单零残留」。
- [x] 1.2 确定 `已预约 → 制作中` 不进 NEXT。
  - Evidence: 见 `design.md`。让商户手点「开做」等于把评审 §4/§27 明确删除的接单换名加回。门禁为此加了专门断言：推进 `已预约` 必须被拒绝且状态不变。
- [x] 1.3 确定即时单的删除边界。
  - Evidence: 首页「到店点单」入口在 `remove-retired-entry-screens` 中被显式列为非目标，理由是它与结算取餐方式切换、`type` 字段和状态机入口态同属一件事——本 change 正是那件事，一并删除。结算页的日期与时段控件本身不动，改造属 `switch-to-pickup-time-selection`。

## 2. Red

- [x] 2.1 小程序端：新增 `tests/order-lifecycle-ui1.test.js`。
  - Red: `node --test tests/order-lifecycle-ui1.test.js` → `tests 8 / pass 0 / fail 8`。八项分别覆盖状态语义、单向无撤销、终态不再前进、种子状态与 `type`、即时单零残留、结算生成预约单、订单筛选段、二维码门控。
- [x] 2.2 PC 端：新增 `checks/check_order_lifecycle.js`（数据层 Node 运行态 + 页面层静态）。
  - Red: 对 `base_sha` 树执行 → `ORDER_LIFECYCLE_GATE=FAIL`（`exit=1`），五项失败：NEXT/LANES 不符、`contract still exports revertOrder`、状态色阶不符、`a1 has retired status 待制作`、推进链路取不到 `制作中` 订单。

## 3. Green

- [x] 3.1 小程序状态机与撤销。
  - Evidence: `utils/util.js` 的 `STATUS_MAP` 收敛为六态与非订单语义；`NEXT` 改为 `{ 制作中: '待取餐', 待取餐: '已完成' }`，`ACT` 为 `已预约` 提供只读标签「待开做」；`advanceOrder` 去掉 `prev` 与 `onUndo`；`advanceMeta` 的 `isView` 改为 `!NEXT[status]`；`orderMode` 整体删除。`components/toast/toast.js` 去掉 `_onUndo`/`undoable`，`.wxml` 去掉撤销按钮，`.wxss` 去掉对应样式。
- [x] 3.2 小程序页面与种子。
  - Evidence: `confirm.js` 去掉 `mode`、恒走预约、按 `minsToPickup <= 30` 判定 `制作中`/`已预约`、取餐号改 4 位无前缀、订单去掉 `type`；`confirm.wxml` 去掉取餐方式切换与「尽快」块（113 → 97 行），`.wxss` 同步；`orders.js` 筛选段改六态并去掉 `goPay`；`home.js` 去掉「到店点单」；`order-detail` 去掉 `reserve` 派生、二维码按 `o.status === '待取餐'` 门控、取消卡按 `已预约` 门控；`utils/data.js` 订单种子迁六态、删 `待支付`/`已取消` 两单、补一条 `已预约`。
- [x] 3.3 PC 端状态机、撤销与页面。
  - Evidence: `data/api.js` 的 `NEXT`/`ACT`/`LANES`/`STATUS_MAP` 六态化，`advanceOrder` 去掉 `prev` 返回，`revertOrder` 整个函数与导出删除，`advanceMeta` 的 `isView` 改为 `!NEXT[status]`；`pages/orders.js` 默认泳道改 `已预约`、去掉两处 `onUndo`；`pages/dashboard.js` 去掉 `onUndo`、待办与实时订单口径改六态；`pages/verify.js` 的状态判定改为「已退款拒绝核销 / 非待取餐提示尚未备好」；`data/seed.js` 种子迁六态并补一条 `已预约`。
- [x] 3.4 两端门禁至 Green。
  - Green: 小程序 `node --test tests/order-lifecycle-ui1.test.js` → 8/8；PC `ORDER_LIFECYCLE_GATE=PASS`。

## 4. Refactor and writer gate

- [x] 4.1 修正两处自己写的断言缺陷。
  - Refactor: ① 小程序用例断言 `doesNotMatch(util.js, /onUndo|撤销/)`，但 `util.js` 的注释里正当地写着「生产禁止撤销」，属假阳性。断言收紧为只查 `onUndo`，并**新增**对 `toast.js` 的 `onUndo|undoable` 与 `toast.wxml` 的撤销文案两条断言——把判定从「词不出现」改为「能力不存在」，强度提高。② 用例夹具用了非规范商品 ID `x1`，`catalogStore` 要求大整数字符串，改用与其他用例一致的 `9007199254740993`。
  - PC 门禁另有一处：`assert.deepEqual` 比较 vm 沙箱内的对象会因跨 realm 原型不同而误报，改用 `JSON.stringify` 比较。修正后对 `base_sha` 树复跑仍 `FAIL`，红线未削弱。
- [x] 4.2 修复被本 change 放大的既有文案缺陷。
  - Refactor: PC 订单页对不可推进订单渲染 `该订单已${o.status}`，而状态名本身以「已」开头。旧模型下 isView 只有 `已完成`/`已取消`，六态把 `已预约`/`退款中`/`已退款` 也纳入只读分支，缺陷在默认泳道即可见。改为 `该订单${o.status}`，门禁加断言防回归。
- [x] 4.3 全量回归。
  - Refactor: 小程序 `npm test` → `tests 42 / pass 42 / fail 0`（新用例挂入 test script，既有 34 条无回归）。PC 三个门禁全部 PASS：`ORDER_LIFECYCLE_GATE`、`ADMIN_SCOPE_GATE`、`CATALOG_FIELDS_GATE`。两端全部 JS 解析通过。
- [x] 4.4 UI1：PC 后台浏览器实际运行。
  - UI1 泳道：`已预约 1 / 制作中 4 / 待取餐 2 / 已完成 1 / 已退款 1 / 全部 9`，页面副标题为「履约流转：已预约 → 制作中 → 待取餐 → 已完成」。
  - UI1 只读态：选中 `已预约` 订单 `0145`，右侧详情无推进按钮（`reservedHasAdvance: false`），行内说明为「该订单已预约」（修复前为「该订单已已预约」）。
  - UI1 推进链路：切到 `制作中` 泳道，推进按钮标签为「备好」；点击后 `a1` 由 `制作中` 变为 `待取餐`，泳道计数由 `制作中 4 / 待取餐 2` 变为 `制作中 3 / 待取餐 3`。
  - UI1 无撤销：推进后的 Toast DOM 中不含撤销动作，全页 `/撤销/` 匹配为 false。
  - UI1 运行态：`window.Api.NEXT` 为两条转换、`LANES` 为六态口径、`Object.hasOwn(Api, 'revertOrder')` 为 false、`advanceMeta('已预约')` 为 `{ label: '待开做', isView: true }`。
  - 控制台：`runtimeErrors` 为空数组。
- [x] 4.5 owned-path 审计与 `git diff --check`。
  - 见 5.1 的验证记录。
- [x] 4.6 记录门禁证据与 candidate SHA。
  - Writer verdict: `{ gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: d0e17d6417817f48833b82081173eb411dbccba0, candidate_sha: external-post-commit（见 5.1）, hard_blockers: 0, unverified_boundary: 服务端定时排产与退款触发无实现可验；小程序 UI1 来自 Node harness、PC UI1 为人工浏览器操作；openspec CLI 缺失 }`。

## 5. Independent verification

- [ ] 5.1 在干净 detached worktree 对精确 candidate SHA 只读验证。
- [ ] 5.2 记录 PASS/FAIL 与剩余外部边界。

## 6. 集成与后续

- 三份 delta MUST 与本 change 一并 archive 后应用到生效 spec。
- 后续新增类：取餐时间点选择、按取餐日期的售罄开关、全局折扣率、手机号绑定与双要素、订阅消息、退款流程、支付对账兜底、PC 扫码登录。多数依赖后端，`services/api` 目前只有 catalog。
