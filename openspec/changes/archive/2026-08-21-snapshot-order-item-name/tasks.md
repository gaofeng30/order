## Red

- [x] 1. 先复现缺陷，不靠推断。用页面 harness 走「加购目录商品 → 结算 → 打开订单」：
  - `ORDERS_LIST_THREW: Cannot read properties of undefined (reading 'name')`
  - `ORDER_DETAIL_THREW: Cannot read properties of undefined (reading 'name')`
  - `order.total = "32.00" (string)` 而种子订单 `total = 70 (number)`
- [x] 2. 写 `checks/check_item_name.js`，十二项，其中五项标 `[takeover]`。
  - 「不回查」断的是行为：给一个本地目录查不到的 id 配上名称，摘要必须报出该名称；再辅以结构断言「摘要实现内不出现 `itemById`」。不断「已删除菜品」这类字面量。
- [x] 3. base_sha `f92acfb` 上运行 → `ITEM_NAME_GATE=FAIL (8/12)`。
  - 八项红覆盖两端元组、回查路径、下单后可打开、字段类型、结算恒等式、折扣舍入、PRD 定义。
  - 四项绿为非回归守卫：字段齐备、交易号与退款记录双向、无整单级口味、JS 解析。
- [x] 4. 记录基线：小程序 73 pass / 0 fail；归档门禁 30 项，14 项 FAIL。

## Green

- [x] 5. PRD §15.6.2：`items` 补 `name` 列，并写明它是快照而非外键、以及为什么必须排在必填段内。
- [x] 6. PC `data/seed.js`：28 处订单项补名称快照，名称从同文件 `MENU` 提取而非手抄。
- [x] 7. PC `data/api.js` 的 `itemsSummary` 改读快照，删除 `|| { name: '已删除菜品' }` 兜底；`pages/orders.js` 的两处位置索引右移并改具名解构。
- [x] 8. 小程序 `utils/data.js`：16 个 `items` 块共 30 行补名称与折后价。
  - 前两次尝试用正则改写，都在含 `['口味']` 数组的行上截断出错，`git checkout` 回滚后改用括号匹配扫描器 + 顶层逗号切分，并跳过非种子的注释块。
- [x] 9. 小程序四处读取点改读快照：`utils/util.js` 的 `itemsSummary`、`order-detail`、`admin-order-detail`、`admin-verify`。
  - `order-detail` 保留一处 `itemById`，只为取图片，允许返回 `null` 回落占位图 —— 订单没有固化图片，这是诚实降级。
- [x] 10. `pages/confirm/confirm.js`：`items` 写入名称与折后价；`total` / `subtotal` 由格式化字符串改为数字，与种子订单同类型。
- [x] 11. 更新 `tests/catalog-ui1.test.js` 中被本次事实变更钉住的三条断言（价格位置、`total` / `subtotal` 类型），保留其原意并补断名称快照。该文件原不在 owned paths，已补入。

## Refactor

- [x] 12. 全仓 `itemById` 调用点复核：只剩 `admin-products`（菜品页本身）与 `order-detail`（取图，带 null 回落）；PC 只剩 `dashboard.js` 的销量排行，属 §6.12 另一件事，不在本 change。

## 本地验证

- [x] 13. `ITEM_NAME_GATE=PASS`（12/12）。
- [x] 14. `node --test tests/*.test.js` → 80 pass / 0 fail（既有 73 项不回归 + 新增 7 项）。
- [x] 15. `python3 tests/lint_wx.py .` → `WX_LINT=PASS`。
- [x] 16. 归档门禁与 base `f92acfb` 逐行 diff：**仅 `align-order-model/check_order_model.js` 由 PASS 转 FAIL**，即 proposal 记录的接管，其余 29 项完全一致。
- [x] 17. UI1：`tests/order-item-name-ui1.test.js` 七项，驱动真实 `Page` 对象。
  - 覆盖：结算固化名称、订单列表可打开、订单详情可打开且图位回落、新旧订单字段类型一致、目录改名后历史订单不变、两端种子每行都带名称、商户端在菜品表清空后仍能渲染名称。
  - 其中第二、三项正是 Red 阶段复现的两处崩溃。

## 独立验证

- [x] 18. 候选 SHA `b2cf964`。
  - 首个候选 `814ceb7` 因 `git add -A` 再次误纳 `__pycache__/*.pyc`（owned paths 之外的构建产物）作废；amend 剔除后重新产生候选并重跑验证。同一失误第二次发生，后续改为按路径显式 `git add`。
- [x] 19. 在干净 detached worktree 对 `b2cf964` 只读验证。
  - `DIRTY=0`；`ITEM_NAME_GATE=PASS`；`node --test tests/*.test.js` → 80 pass / 0 fail；`WX_LINT=PASS`；归档门禁与 base 逐行 diff 仅 `check_order_model.js` 一行由 PASS 转 FAIL，即已记录的接管。
