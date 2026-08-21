## Why

要建 PRD §6.11 的财务与对账页，得先有可对账的订单。当前 `apps/web-admin` 的订单记录与 PRD §15.6.2 差得很远：

- **金额单位是元不是分。** `total: 58` 这样的整数元。§15.6.2 明写「整数分」。对账要按分核微信账单，元为单位的中间态在求和与折扣舍入上必然产生一分钱的差。
- **支付事实一个都没有。** 没有 `paidAt`、没有微信交易号、没有折扣率与身份快照。§6.11 要求展示的三项（支付金额、支付时间、微信交易号）里有两项无处可取。
- **退款事实一个都没有。** §6.11 要求退款单号、退款金额、退款状态与操作人；种子里 `已退款` 的订单只有一句 `note: '用户取消'`。
- **取餐信息缺失且带过期派生量。** 没有 `pickupDate` / `pickupTime` / `mealPeriod` / `pickupPoint`，却存了 `mins`（距取餐分钟数）—— §15.6.2 已把 `minsToPickup` 列入删除项，它随时间变化，存下来必然过期。
- **整单级口味未清理。** §15.6.2 规定口味与备注绑定在 `items` 行内，整单级只有 `orderNote`，但记录仍有 `flavor` 字段且页面在读它。

另有一处由此暴露的缺口：泳道只有五个状态，`退款中` 不在其中，只有切到「全部」才看得见 —— 而它恰是唯一需要人工盯到到账的状态。

本 change 只做模型对齐，不建财务页；财务页在其后的 change 里做。

## What Changes

- **种子订单**重写为 §15.6.2 的形态：11 笔，金额全部整数分，含支付时间、微信交易号、折扣率与身份快照、四项取餐信息；`items` 行扩为 `[id, 数量, 原价单价, 折后单价, 口味, 备注]`。
- **退款记录**落到 `refund` 块（退款单号 / 金额 / 状态 / 操作人 / 时间 / 原因），退款中与已退款各一笔，其中已退款的一笔是**部分退款**（退 12.00 于实付 38.00），用于验证净额是按金额而非按订单数算。
- **契约层**新增 `yuan(cents)` 作为唯一的分转元展示函数；`LANES` 补入 `退款中`，六态各一条泳道。
- **订单管理页**详情改展示支付时间、取餐日期与时间点、取餐点、微信交易号，员工折扣单额外显示折扣率与减免；列表「等待」列改为「取餐」（时间点 + 日期）；口味带改读行内口味。
- **工作台**同步「等待」列改为「取餐」。
- **PRD §16.5** 记录一处已知遗留：菜品价格仍以元存储（§15.6.1 要求分），订单不读菜品价而读自身快照，故不影响对账，但导入模板与菜品表单的单位对齐是另一件事。

## 与已归档门禁的冲突

`archive/2026-08-20-adopt-six-state-order-lifecycle` 的 `check_order_lifecycle.js` 把 `LANES` 钉死为 `['已预约','制作中','待取餐','已完成','已退款','全部']` —— 五个状态。而 PRD §15.5.3 对订单管理页的要求是「**六态泳道**」。

这不是本 change 越界，是那个 change 少给了一格：`退款中` 从未出现在泳道里。它在当时不可见，因为种子里根本没有 `退款中` 的订单。

归档产物不可变，因此本 change 的门禁**接管**该项断言：泳道集合改为六态加「全部」，原断言中的其余部分（`NEXT` 推进图、`已预约` 不可由前端推进、五个废弃状态不得出现在泳道里）原样保留。接管后 `check_order_lifecycle.js` 的该项会失败，这是事实变更的正常结果，不是回归。

## Capabilities

### Modified Capabilities

- `web-admin-scope-conformance`：新增三条 requirement —— 金额以分存储且恒等式成立、订单携带支付与退款事实、六态各一条泳道。

## Impact

- Owner：branch `worktree-order-model`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/order-model`。
- Owned paths：`apps/web-admin/{app.css,data/seed.js,data/api.js,pages/orders.js,pages/dashboard.js}`、`docs/product/online-ordering-system-prd-0818.md` 的 §16.5、`openspec/changes/align-order-model/**`、`openspec/specs/web-admin-scope-conformance/spec.md`。
- Non-goals：不建财务与对账页；不实现发起退款（§15.5.3 列为订单管理的改造项，另做）；不改菜品价格的存储单位；不改小程序端；不接后端。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_order_model.js` 十项全过；base_sha 树上九项红（第十项「所有 JS 可解析」是非回归守卫，本就应绿）；小程序 65 项不回归；历史归档门禁无新增失败；浏览器内验证员工折扣单 27.20 + 23.80 − 9.00 = 51.00 精确成立、六条泳道计数正确、退款中单可直接进入。
