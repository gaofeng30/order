## Why

客户给出了取餐地点的正式值：**党政办公中心后院老食堂北门**，并明确「有且仅有一个」。

§13.3 把「单取餐点名称、地址和说明」列为 UAT 前配置、责任方是客户现场负责人 —— 现在这个值到手了，必须落进文档与两端数据，否则演示数据里那套虚构的「县前直营店 / 县前大厦 1F · 2 号取餐窗口」会一直跟到 UAT。

同时暴露一处实现不一致：`pages/confirm/confirm.js:99` 写的是

```js
pickupPoint: data.STORE.pickupWindow,      // '县前大厦 1F · 2 号取餐窗口'
```

而种子订单存的是点位**名称**（`'县前直营店'`）。于是新下的单在订单详情里 `PICKUP_POINTS.find(x => x.name === o.pickupPoint)` 必然查不到，回落到门店地址。同一个字段在两条写入路径上装着不同语义的值。

根子在于数据模型仍是多点形态：一个 `PICKUP_POINTS` 数组，每项带 `name` 与 `addr`。§3.1 明写「一期为单门店单取餐点，不提供多点选择或分单路由」，§5.5 亦然。既然只有一个点、且客户只给了一个字符串，名称与地址的区分就没有对应的现实事实，只会让两条写入路径各自挑一个。

## What Changes

- **取餐点统一为单一常量** `PICKUP_POINT = '党政办公中心后院老食堂北门'`，两端同源。
- **`PICKUP_POINTS` 数组收敛**：一期单点，不再维护 `name` / `addr` 二元结构；订单详情因此只展示一次取餐地点，不再重复输出同一串文字。
- **两端全部订单快照与配置**改为该值；`confirm.pay()` 写入同一常量，与种子一致。
- **PC 的三点位表收敛为单点**：`data/seed.js` 里还留着「县前直营店 / 绥芬河北站取餐点 / 青云镇综合市场点」三条，营业设置页据此渲染了一个 `<select>` —— 那正是 §3.1「不提供多点选择」排除的东西。改为只读展示。
- **删除 `STORE.pickupWindow`**：它与新常量表达同一件事。一个事实存两处，正是 `confirm.pay()` 与种子各挑一个字段的成因。
- **PRD §13.3 记录已确认值**，把该行从「待客户提供」变为「已确认」，并新增一行标记门店名与门店地址仍为演示值。

## Capabilities

### Modified Capabilities

- `miniprogram-scope-conformance`：新增一条 requirement —— 取餐点是单一常量，所有写入路径与展示路径同源。
- `web-admin-scope-conformance`：新增一条 requirement —— PC 使用同一取餐点常量。

## Impact

- Owner：branch `worktree-pickup-point`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/pickup-point`。
- Owned paths：`docs/product/online-ordering-system-prd-0818.md`（仅 §13.3）、`apps/wechat-miniprogram/{utils/data.js,pages/confirm/**,pages/order-detail/**}`、`apps/web-admin/{data/seed.js,pages/settings.js}`、`openspec/changes/set-single-pickup-point/**`、两份生效 spec。
- Non-goals：
  - **不改门店名与门店地址**。`STORE.branch = '县前直营店'`、`STORE.addr = '绥芬河市青云镇通商路'` 同为演示值，但客户这次只给了取餐地点。凭空替换会把一个虚构值换成另一个虚构值，反而更难发现。这两项单独向客户索取，本 change 不动。
  - 不做多取餐点能力，也不为将来预留 —— §3.1 明确排除。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_pickup_point.js` 九项全过；base_sha 树上八项红；两端既有测试不回归；`lint_wx.py` 通过；归档门禁失败集合与 base 逐行一致。
