## Red

- [x] 1. 写 `checks/check_mp_order_model.js`，十二项。
- [x] 2. base_sha `edcadb6` 上运行 → `MP_ORDER_MODEL_GATE=FAIL (10/12)`。
  - 关键红项 `checkout keeps cents integral through a half-yuan price: total = 32.5` —— 3250 分经元的往返变成浮点数，正是要消灭的东西。
  - 两项绿为非回归守卫：无手工除法、JS 解析。
- [x] 3. 记录基线：小程序 80 pass / 0 fail；归档门禁 30 项，14 项 FAIL。
- [x] 4. 顺带发现四条种子的 `total` 与 items 之和对不上（`o1` 70/76、`o3` 62/68、`a4` 58/68、`a10` 42/40，其中 a10 是 `add-merchant-order-search` 引入时算错的）。改分时 `total` 一律由 items 计算，四条自然被修正。

## Green

- [x] 5. `utils/money.js`：唯一的分转元入口 `yuan`。非法输入返回 `—` 而非抛错或 `0.00`（`Number(null)` 是 0，不加判断会把「没有金额」显示成零元）。
- [x] 6. `components/money/money.js`：新增 `cents` 属性，组件内部完成转元；模板改渲染 `text`。八个订单金额渲染点改走 `cents`。
- [x] 7. `utils/data.js`：纯字符串日历运算（days-from-civil / civil-from-days）+ `minsToPickup(o)`、`orderPickupLabel(o)`、`pickupDateOf(pk)`、`nowStamp()`。
  - 首版用 `Date.parse` / `new Date(ms)`，虽然传的是显式时间戳、并未读时钟，但命中了 `add-merchant-order-search` 那条「营业日不得由运行时时钟推导」的归档断言。那条断言是好的，不该为绕过它去做接管，故改为无 `Date` 实现。跨月、跨年、闰年五个边界抽查通过。
- [x] 8. 两份订单种子重建为 §15.6.2 形状：金额整数分、补九个结算字段、删 `time` / `mins` / `count` / `flavor` / `flavors` / `note` / `pickupLabel` / `minsToPickup`，`note` 改名 `orderNote`，整单级口味并入 `items` 行内。
- [x] 9. `canCancelReserve` 改走 `minsToPickup(o)`；取餐文案与件数、下单时刻在各页面现算。
- [x] 10. `confirm.pay()` 直接取目录快照的 `price_cents` 写入订单，不再经过 `Number(price_text)` 的元往返；补齐九个结算字段。

## Refactor

- [x] 11. 复核：全仓无第二处分转元，无金额语境下的 `/ 100` 与 `toFixed(2)`；无任何 `o.pickupLabel` / `o.minsToPickup` / `o.flavor(s)` 读取点。

## 本地验证

- [x] 12. `MP_ORDER_MODEL_GATE=PASS`（12/12）。
- [x] 13. `node --test tests/*.test.js` → 87 pass / 0 fail。
  - 其中六条既有断言钉住了本次故意改变的事实（金额单位、`pickupLabel` 字段、`minsToPickup` 入参），已按新事实更新并保留原意：取餐文案改断推导结果，取消窗口改断「塞进记录的陈旧值不影响判定」。
- [x] 14. `python3 tests/lint_wx.py .` → `WX_LINT=PASS`。
- [x] 15. 归档门禁 30 项与 base 逐行一致，**无接管**。
- [x] 16. UI1：`tests/order-model-ui1.test.js` 七项，驱动真实 `Page` 对象。
  - 覆盖：结算写整数分且精确结算、全部种子逐单结算、格式化入口边界值、取消窗口跟时钟而非记录、无冻结取餐字段、整单只有 `orderNote` 且行内口味在商户列表可见、用户端两页端到端渲染整数分。

## 门禁自身的两次误报（均已修正）

- [x] 17. 「不复用目录格式化」原按 `/catalogStore|CatalogError/` 断言，命中的却是 `money.js` **注释里**解释「为什么不复用」的那句话。改为断结构：不 `require` 目录模块，且非法输入返回 `—`。
- [x] 18. 「无手工除法」原按裸 `/\s*100` 断言，命中日历算法的 `Math.floor(doe / 100)`。改为只在金额语境下匹配。
  - 两次都是同一个毛病：断言词而非断言事实。design.md 里写着这条，仍然踩了两次。

## 独立验证

- [x] 19. 候选 SHA `841dfe3`。
  - 首个候选 `8e90666` 因 `git add openspec` 连带纳入运行门禁生成的 `__pycache__/*.pyc` 作废；amend 剔除后重新产生候选并重跑验证。该失误本轮第三次出现，根因是归档门禁里的 Python 脚本会在仓库内生成字节码而仓库未忽略该目录。
- [x] 20. 在干净 detached worktree 对 `841dfe3` 只读验证。
  - `DIRTY=0`；`MP_ORDER_MODEL_GATE=PASS`；`node --test tests/*.test.js` → 87 pass / 0 fail；`WX_LINT=PASS`；归档门禁 30 项与 base 逐行一致。
