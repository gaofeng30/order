## Why

小程序下单成功后，「我的订单」列表和订单详情页**都会抛异常**：

```
ORDERS_LIST_THREW:  Cannot read properties of undefined (reading 'name')
ORDER_DETAIL_THREW: Cannot read properties of undefined (reading 'name')
```

原因是小程序里并存着两套商品模型。菜单页与详情页已经走服务端目录（`utils/catalogStore.js`，数字 id、`price_cents`），购物车持有的也是目录快照；但订单项只存 `[id, qty, price, ...]`，而订单列表与详情页用 `utils/data.js` 的 `itemById(id)` 回查**本地种子**（`p001`、元）。新下的单带的是目录 id，回查必然落空，`m.name` 随即抛错。种子里的历史订单因为用的是 `p001` 反而正常，所以这个缺陷在演示数据上看不出来。

回查本身才是根因。§5.6 已经确立「报价和订单明细固化原价、折扣率、折后价、身份及价格版本」—— 订单是历史记录，其显示不应依赖一张随时会变的目录表。商品下架、改名或删除之后，历史订单要么显示不出名字，要么显示成改名后的样子，两种都是错的。PC 端已经暴露了同一问题的另一面：`itemsSummary` 不得不写 `|| { name: '已删除菜品' }` 来兜底。

## What Changes

- **`items` 元组补一列名称快照**：`[id, name, qty, price, discountedPrice, flavors?, note?]`。名称在下单那一刻固化，与原价、折后价同批。
- **两端同步**：小程序与 PC 后台的种子、读取点、汇总函数一并改。两端读同一个 §15.6.2 定义，不同步就等于把契约拆成两份。
- **移除按 id 回查名称的全部路径**：小程序 `itemsSummary`、`order-detail`、`admin-order-detail`、`admin-verify`；PC `itemsSummary`。PC 的 `|| { name: '已删除菜品' }` 兜底随之删除 —— 它是回查方案的补丁，回查没了补丁也就没有存在理由。
- **PRD §15.6.2 补该列**并说明为什么名称是快照而不是外键。

## Capabilities

### Modified Capabilities

- `miniprogram-scope-conformance`：新增两条 requirement —— 订单项自带名称快照且渲染不回查目录；下单后订单列表与详情可打开。
- `web-admin-scope-conformance`：新增一条 requirement —— PC 订单项同样自带名称快照，且结算恒等式在新元组下依然成立。

## 与已归档门禁的冲突

`openspec/changes/archive/2026-08-21-align-order-model/checks/check_order_model.js:65` 按**位置**断言结算恒等式：

```js
const sub  = o.items.reduce((s, it) => s + it[1] * it[2], 0);
const paid = o.items.reduce((s, it) => s + it[1] * it[3], 0);
```

`name` 插在 index 1，上述索引整体右移一位，该门禁必然失败。这是一次**正当的事实变更**而非误报：`align-order-model` 定义元组时没有名称快照这个概念。按项目既有做法，本 change 的门禁**接管**该断言集 —— 把结算恒等式、整数分、员工折扣舍入、交易号格式、退款双向一致、无整单级口味这些仍然成立的部分按新元组重写，原门禁不再作为 PC 订单模型的权威。

## Impact

- Owner：branch `worktree-item-name`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/item-name`。
- Owned paths：
  - `docs/product/online-ordering-system-prd-0818.md`（仅 §15.6.2）
  - `apps/wechat-miniprogram/{utils/data.js,utils/util.js,pages/confirm/confirm.js,pages/order-detail/**,pages/admin-order-detail/**,pages/admin-verify/admin-verify.js}`
  - `apps/web-admin/{data/seed.js,data/api.js,pages/orders.js}`
  - `apps/wechat-miniprogram/tests/{order-item-name-ui1.test.js,catalog-ui1.test.js}`
  - `openspec/changes/snapshot-order-item-name/**`、两份生效 spec
- Non-goals：
  - **不改小程序订单的其余字段**。金额单位、`minsToPickup` / `pickupLabel` 的移除、整单级口味的移除、`pickupDate` / `paidAt` / `subtotal` 等字段的补齐，全部留给紧随其后的 `align-miniprogram-order-model`。本 change 只解决「名称从哪来」这一件事。
  - **不统一两套商品模型**。小程序的本地种子 `MENU` 与服务端目录并存是接后端时才收敛的事；本 change 让订单不再依赖任一方，正是为了让那次收敛不再牵动历史订单。
  - 不改订单项的图片来源。订单没有固化图片，商品不在本地目录时图位回落占位图。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_item_name.js` 十二项全过（含从 `check_order_model.js` 接管的五项）；base_sha 树上八项红；小程序既有测试不回归；`lint_wx.py` 通过；除被接管的 `check_order_model.js` 外，归档门禁失败集合与 base 逐行一致。
