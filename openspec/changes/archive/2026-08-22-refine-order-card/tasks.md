## Red

- [x] 1. 先定位截图里那行乱码的成因，不靠推断。
  - `pages/orders/orders.js:21` 为 `o.items.reduce((a, b) => a + b[1], 0)`。`snapshot-order-item-name` 把名称插到 index 1 后，这个下标从数量变成了商品名，于是 `0 + '商务双拼饭' + '山药排骨汤'` 拼成字符串。
  - 全仓排查同类写法：只此一处，`admin-orders.js` 已用 `it[2]`、PC 侧无此类聚合。**是我在那个 change 里改了两处中的一处。**
- [x] 2. 写 `checks/check_order_card.js`，九项。
- [x] 3. base_sha `ff4d0fa` 上运行 → `ORDER_CARD_GATE=FAIL (5/9)`。
  - 其中一项直接把缺陷现场打了出来：`o1.itemCount looks like a number concatenated with a product name: 0商务双拼饭山药排骨汤`。
  - 该项首版用 `/^\d+[^\d.\s:¥]/` 判定，误伤了 `pickupDate = '2026-08-21'`。收紧为「数字紧跟中文」后才指向真正的失败模式。
- [x] 4. 记录基线：小程序 102 pass / 0 fail；归档门禁 34 项，14 项 FAIL。

## Green

- [x] 5. 删除订单卡片的「共 N 件」及 `orders.js` 里的 `itemCount`、`orders.wxss` 里的 `.oc-count`。
  - 删而不是改成 `b[2]`：用户明确说不需要，留一个没人渲染的字段会让下一个人以为它有用途。
- [x] 6. 取餐号徽章去掉「号」标签与 `.oc-code-lbl` 样式，只留号码；`.oc-code` 原有的双向居中保持不变。
- [x] 7. 「取消预约」加 `white-space: nowrap` 与 `flex: 0 0 auto`。
  - 两条缺一不可：只加 nowrap，按钮仍会被 flex 压窄而让文字溢出。
- [x] 8. 门禁新增结构规则：对 `items` 的 `reduce` 聚合只能选下标 2 / 3 / 4。
  - 这条比「断言 `itemCount` 等于 3」强 —— 后者只锁一个字段，前者锁的是**这一类写法**，下次元组再变形会被直接拦下。

## 自查发现（提交前修正）

- [x] 9. 删「共 N 件」时把 `.oc-total-lbl` 的 `margin-left: auto` 一并删了。件数原本是行首元素，「合计」靠这个 auto 把整组推到右边；删掉后整行会塌成左对齐。已还原，并补了第九项断言锁住它。
  - 这处不是门禁抓到的，是我自己复核样式时发现的 —— 说明视觉布局的回归仍缺乏自动化覆盖，只能靠人看。

## 本地验证

- [x] 10. `ORDER_CARD_GATE=PASS`（9/9）。
- [x] 11. `node --test tests/*.test.js` → 102 pass / 0 fail，无回归。
- [x] 12. `python3 tests/lint_wx.py .` → `WX_LINT=PASS`。
- [x] 13. 归档门禁 34 项与 base 逐行一致，**无接管**。
- [x] 14. UI1：在页面 harness 中驱动真实 `Page`，确认行字段不再含 `itemCount`、`total` 为整数分数字、`summary` 与 `timeText` 正常。

## 独立验证

- [x] 15. 候选 SHA `39dd516`。
- [x] 16. 在干净 detached worktree 对 `39dd516` 只读验证：`DIRTY=0`；`ORDER_CARD_GATE=PASS`；102 pass / 0 fail；`WX_LINT=PASS`；归档门禁 34 项与 base 逐行一致。
