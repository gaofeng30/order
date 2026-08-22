## Why

用户在真机上看订单列表，指出三处展示问题。其中第一处不只是样式：

**「共 N 件」这一行是错的。** 截图里显示的是「共 0商务双拼饭山药排骨汤 件」。原因是 `pages/orders/orders.js:21`：

```js
itemCount: o.items.reduce((a, b) => a + b[1], 0),
```

`snapshot-order-item-name` 把名称插到了 `items` 元组的 index 1，这个下标从「数量」变成了「商品名」，于是 `0 + '商务双拼饭' + '山药排骨汤'` 拼成了一个字符串。我在同一个 change 里改对了 `admin-orders.js` 的同类聚合，**漏了这一处**；门禁与测试都只覆盖了元组形状和渲染名称，没有覆盖任何基于 `items` 的数值聚合，所以一路绿灯。

用户要求删掉这一行，所以修法是删除；但缺陷的**成因**要留下防线 —— 对 `items` 的聚合必须选数值列，这条规则要能被门禁检出，否则下一次元组变形还会重演。

另外两处是纯展示：取餐号徽章里的「号」字多余，号码应居中；「取消预约」四个字被挤成两行。

## What Changes

- **删除订单卡片的「共 N 件」**，以及 `orders.js` 里产生它的 `itemCount`。
- **取餐号徽章只留号码并居中**，去掉「号」标签与其样式。
- **「取消预约」单行展示**：按钮不换行、不被压缩。
- **门禁新增一条结构规则**：对 `items` 的聚合只能选数值列（下标 2 / 3 / 4），选到名称列即报错。

## Capabilities

### Modified Capabilities

- `miniprogram-scope-conformance`：新增一条 requirement —— 订单卡片的展示元素与对订单项的数值聚合规则。

## Impact

- Owner：branch `worktree-order-card`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/order-card`。
- Owned paths：`apps/wechat-miniprogram/pages/orders/**`、`openspec/changes/refine-order-card/**`、`openspec/specs/miniprogram-scope-conformance/spec.md`。
- Non-goals：
  - **不改取餐点内容**。「取餐地点有且仅有一个」是同批提出的另一件事，属配置数据与 PRD §13.3，另开 change。
  - 不改订单详情页的徽章（那是取餐码主视觉，与列表徽章的信息密度要求不同）。
  - 不改订单卡片的其余信息（订单号、状态、摘要、取餐时间、合计）。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_order_card.js` 九项全过；base_sha 树上五项红；小程序既有测试不回归；`lint_wx.py` 通过；归档门禁失败集合与 base 逐行一致。
