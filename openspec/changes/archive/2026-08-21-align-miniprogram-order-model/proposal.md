## Why

PC 后台的订单已按 §15.6.2 改造完毕（`align-order-model`，2026-08-21），小程序没跟上。同一份 §15.6.2 定义，两端各写一半：

| §15.6.2 要求 | PC | 小程序 |
| --- | --- | --- |
| 金额整数分 | 是 | **否，元** |
| `pickupDate` / `pickupTime` / `mealPeriod` | 有 | 仅商户端有 `pickupDate` |
| `paidAt` / `subtotal` / `discountRate` / `discountCut` / `isStaff` | 有 | **全无** |
| `orderNote` | 有 | 叫 `note` |
| 删除 `minsToPickup` / `pickupLabel` | 已删 | **仍在** |
| 整单级只有 `orderNote`，无整单级口味 | 已删 | **仍有 `flavor` / `flavors`** |

这不只是「字段没补齐」。金额单位分裂意味着两端算出的同一笔钱可能不同：§5.6 要求「金额一律以整数分保存与计算」「逐商品折扣先四舍五入到分再乘数量」，而小程序用元的浮点数，`32.5 × 3` 这类运算会产生 `97.49999999999999`。接后端时两端把各自的数交给同一个支付接口，差额无处可查。

`minsToPickup` 是另一处隐患：它把一个**随时间变化的量冻结成了记录上的字段**。`canCancelReserve` 用它判断 §7.6 的「取餐前 30 分钟可取消」。当前 `NOW_MINS` 是写死的演示时钟（16:48），全应用的时间都不流动，所以这个缺陷**今天还观察不到**；但换成真实时钟的那一刻，取消窗口就会按**下单时刻**而不是当前时刻判定，直接放行本该拒绝的取消。§15.6.2 因此明令删除该字段，剩余时间必须现算。

## What Changes

- **金额改整数分**：`subtotal` / `discountCut` / `total` 与 `items` 行的原价、折后价。新增 `utils/money.js` 的 `yuan(cents)` 作为唯一格式化入口，`money` 组件加 `cents` 属性，页面 MUST NOT 自己写 `/ 100`。
- **补齐 §15.6.2 字段**：`pickupDate`（用户端）、`pickupTime`、`mealPeriod`、`paidAt`、`subtotal`、`discountRate`、`discountCut`、`isStaff`、`orderNote`。
- **删除 §15.6.2 明令删除的字段**：`minsToPickup`、`pickupLabel`。取餐文案与剩余时间改为**从 `pickupDate` + `pickupTime` 实时推导**，取消窗口因此恢复有效。
- **删除整单级口味** `flavor` / `flavors`。展示改为聚合 `items` 行内的口味与备注，与 PC `pages/orders.js` 的做法一致 —— 信息不丢，模型合规。
- **删除派生冗余** `count`（由 items 求和）、`mins`（无任何使用点）；`time` 并入 `paidAt`。

## Capabilities

### Modified Capabilities

- `miniprogram-scope-conformance`：新增四条 requirement —— 小程序订单携带 §15.6.2 的全部结算事实且金额为整数分；取消窗口从取餐时刻实时推导而非下单时冻结；整单级只有 `orderNote`；金额渲染只经由单一格式化入口。

## Impact

- Owner：branch `worktree-mp-order`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/mp-order`。
- Owned paths：`apps/wechat-miniprogram/{utils/data.js,utils/util.js,utils/money.js,components/money/**,pages/confirm/**,pages/orders/**,pages/order-detail/**,pages/result/**,pages/admin-orders/**,pages/admin-order-detail/**,pages/admin-verify/**}`、`apps/wechat-miniprogram/tests/{order-model-ui1.test.js,catalog-ui1.test.js,pickup-time-ui1.test.js,order-lifecycle-ui1.test.js}`、`openspec/changes/align-miniprogram-order-model/**`、`openspec/specs/miniprogram-scope-conformance/spec.md`。
- Non-goals：
  - **不改菜品价格的存储单位**。`MENU.price` 仍是元，§16.5 已把它列为与后端 `products.price` 一并对齐的另一件事。订单不读菜品价而读自身的分级快照，因此本 change 之后订单侧已经完全是分。
  - **不实现员工折扣**。`discountRate` 恒为 100、`discountCut` 恒为 0、`isStaff` 恒为 false，因为身份识别链路尚未接后端（§16.5）。字段先立住，算价链路属另一件事。
  - 不改 PC；不改服务端目录层（`catalogStore` 本来就是分）。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_mp_order_model.js` 十二项全过；base_sha 树上十项红；小程序既有测试不回归；`lint_wx.py` 通过；归档门禁失败集合与 base 逐行一致。
