## Why

§15.5.3 给订单管理列了三项改造：六态泳道（已完成）、补「未取餐」筛选口径、补退款入口。后两项是本 change。

退款是这套系统里**唯一由主账号发起的资金动作**，也是财务与对账页里 `操作人` 这一列的唯一写入方 —— 在此之前该列只能靠种子硬编。

同时要纠正上一个 change 引入的一处错误：`add-finance-page` 在种子里造了一笔部分退款（实付 38.00 退 12.00），并据此写了一条「部分退款是合法情形」的规格。而 PRD 在六处明写**一期只支持原路全额退款**（§7.7「任何部分退款请求必须拒绝且不创建退款记录」、§6.7「一期无部分退款」、§3.2 排除清单、§0 一期范围、§14 规则速查）。这条错误已随上一个 change 合入 main，本 change 一并改回。

## What Changes

- **契约层新增 `refundOrder(id, reason)`**：只从 `已预约 / 制作中 / 待取餐 / 已完成` 四态发起；**不接受金额入参**；原因必填；操作人取自当前登录账号；只推进到 `退款中`；重复发起被拒且不产生第二条记录。
- **契约层新增 `today()` 与 `currentAccount()`**：营业日与登录账号各有单一来源，页面不再硬编日期、不再用装饰用的 `Seed.MANAGER`。
- **`listOrders(lane, { uncollected })`**：§6.7 的「未取餐」口径，是筛选条件不是第七个状态，不进 `LANES` 也不进 `ACT`。
- **订单管理页**：详情页脚加「发起退款」，带二次确认层（展示将退金额、操作人、必填原因、不可撤销后果）；`待取餐` 泳道加「未取餐」开关，切换泳道或退款跳转时自动解除。
- **纠正部分退款**：种子那笔改回全额；契约去掉 `partial` 标记；财务页去掉「部分退款」渲染。
- **种子补一笔跨营业日未取的 `待取餐` 单**，「未取餐」口径才有数据可验。

## 与已归档门禁的冲突

`archive/2026-08-21-add-finance-page` 的 `check_finance.js` 有一条断言依赖那笔部分退款种子（「seed has no partial refund to distinguish the two」），用它来区分「净额按金额扣」与「按笔数扣」两种实现。

那笔种子数据本身违反 PRD §7.7，是上一个 change 的错误。归档产物不可变，因此本 change 的门禁**接管**该项：改用 PRD 允许的区分数据 —— 跨区间的退款（原订单支付于区间外，区间内只有退款没有收款，按笔数扣无从扣起）。原断言中的三项数值、净额等式与整数分要求原样保留。

接管后 `check_finance.js` 的该项会失败，这是纠错的正常结果。

## Capabilities

### Modified Capabilities

- `web-admin-scope-conformance`：修订两条（对账汇总的净额理由改为跨区间退款而非部分退款；退款记录新增「金额恒等于订单实付、不得导出部分退款标记、操作人取自登录账号」），新增两条（主账号发起全额退款、未取餐是查询口径不是状态）。

## Impact

- Owner：branch `worktree-refund`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/refund`。
- Owned paths：`apps/web-admin/{data/seed.js,data/api.js,pages/orders.js,pages/finance.js}`、`openspec/changes/add-order-refund/**`、`openspec/specs/web-admin-scope-conformance/spec.md`。
- Non-goals：不建「支付待处理」页（§7.3，下一件）；不实现微信退款回调与 `退款中 → 已退款` 的推进（由支付回调驱动，非 PC 端动作）；不实现事后核销；不接后端。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_refund.js` 十二项全过；base_sha 树上十一项红；小程序 65 项不回归；已归档的订单模型、财务、商户账号门禁不回归；浏览器内验证四态可退、重复退款被拒、空原因被拒、退款后跳转到 `退款中` 且页脚只剩打印小票、退款即时出现在财务页并带操作人与原因、每笔退款金额均等于订单实付。
