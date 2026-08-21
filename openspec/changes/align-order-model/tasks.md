## 1. 门禁（Red）

- [x] 1.1 写 `checks/check_order_model.js` 十项
- [x] 1.2 在 base_sha `2299da6` 树上确认九项红、`exit=1`（第十项为 JS 解析守卫，本就应绿）

## 2. 数据模型

- [x] 2.1 种子订单重写为 §15.6.2 形态：11 笔，金额整数分，`items` 行扩为六元组
- [x] 2.2 补支付事实：`paidAt`、`txnId`、`discountRate`、`isStaff`
- [x] 2.3 补取餐事实：`pickupDate`、`pickupTime`、`mealPeriod`、`pickupPoint`；删 `mins`
- [x] 2.4 补 `refund` 块，退款中与已退款各一笔，后者为部分退款
- [x] 2.5 删整单级 `flavor`，口味与备注下沉到 `items` 行内
- [x] 2.6 校验 11 笔订单三条恒等式与折扣舍入全部成立

## 3. 契约与页面

- [x] 3.1 契约层加 `yuan(cents)` 并导出
- [x] 3.2 `LANES` 补 `退款中`，六态各一条泳道
- [x] 3.3 `pages/orders.js`：详情改支付时间 / 取餐时间 / 取餐点 / 交易号；折扣单显示折扣率与减免；列表列改「取餐」；口味带读行内
- [x] 3.4 `pages/dashboard.js` 同步列改「取餐」
- [x] 3.5 `app.css` 详情行内口味带与菜名分开
- [x] 3.6 门禁在候选树 `ORDER_MODEL_GATE=PASS`

## 4. 文档与规格

- [x] 4.1 `web-admin-scope-conformance` 三条 ADDED requirement
- [x] 4.2 PRD §16.5 记录菜品价格单位仍为元这一已知遗留

## 5. UI1 证据

- [x] 5.1 `python3 -m http.server 8092` 起站
- [x] 5.2 员工折扣单 0131：27.20 + 23.80 = 51.00，折扣行 −9.00，原价 60.00
- [x] 5.3 六条泳道计数 1/4/2/2/1/1，全部 11
- [x] 5.4 退款中泳道可直接进入并选中 0085
- [x] 5.5 详情展示支付时间、取餐时间、取餐点、微信交易号
- [x] 5.6 行内口味带 `display:inline; margin-left:8px`，与菜名不粘连
- [x] 5.7 工作台列同步为「取餐」，控制台无 error

## 6. 回归

- [x] 6.1 小程序 `npm test` 65/65
- [x] 6.2 历史归档门禁：唯一新增失败为 `check_order_lifecycle.js` 的泳道集合项，属事实变更，断言已由本门禁接管（见 proposal「与已归档门禁的冲突」）；其余失败集合与 base_sha 树完全一致

## 7. 交付

- [ ] 7.1 提交
- [ ] 7.2 clean worktree 独立验证
- [ ] 7.3 合入 main、应用 delta、归档
