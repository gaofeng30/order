## Why

上一个 change 把取餐点落成了正式值，同时在 PRD §13.3 单列一行标记「门店名称与门店地址仍为演示值、待客户提供」。客户现已给出：

- 门店名：**绥安食品**（没有分店，`branch` 这个概念本身不成立）
- 地址：**党政办公中心后院老食堂**

当前两端都还带着 `branch: '县前直营店'` 与 `addr: '绥芬河市青云镇通商路'`，并在六处 UI 上输出。其中三处是硬编码的 `绥安食品 · 县前直营店` 字面量，配置改了也不会跟着变。

顺带发现一处真实错误：`pages/admin-order-detail/admin-order-detail.wxml:44` 的**取餐地点**渲染的是 `{{store.branch}}` —— 商户端订单详情把「门店分店名」当成了取餐地点。订单自身携带 `pickupPoint` 快照（§7.2 要求生成订单时固化取餐点），却没有被使用。分店名一旦变化，历史订单展示的取餐地点会跟着变，这与「快照」的意义直接相反。

## What Changes

- **`STORE.addr` 落正式值**；**删除 `branch` 字段**：门店名就是 `name`，一期单门店无分店概念，留着一个恒等于门店名的字段只会让下一个人再挑一次。
- **六处 `store.branch` 消费点**改为正确来源：门店名处用 `store.name`，取餐地点处用订单自身的 `pickupPoint`。
- **三处硬编码的 `绥安食品 · 县前直营店`** 改为读配置。
- **PRD §13.3** 该行标记为已确认。

## Capabilities

### Modified Capabilities

- `miniprogram-scope-conformance`：新增一条 requirement —— 门店标识只有一处事实来源，且订单的取餐地点取自订单快照而非门店配置。

## Impact

- Owner：branch `worktree-store-identity`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/store-identity`。
- Owned paths：`docs/product/online-ordering-system-prd-0818.md`（仅 §13.3）、`apps/wechat-miniprogram/{utils/data.js,pages/home/**,pages/launch/**,pages/menu/**,pages/confirm/**,pages/admin-order-detail/**}`、`apps/web-admin/{data/seed.js,app.js,pages/layer.js}`、`openspec/changes/set-store-identity/**`、`openspec/specs/miniprogram-scope-conformance/spec.md`。
- Non-goals：
  - **不把所有 `绥安食品` 字面量改为配置读取**。门店名在 `launch.wxml`、`menu.wxml`、`index.html`、`layer.js` 等处硬编码，但那些值**是对的** —— 它们不是缺陷，只是耦合。本 change 只改错的值，以及本来就要改的那几行；顺手重构其余处属相邻问题。
  - 不改门店公告、营业时间、取餐点（上一个 change 已定）。
  - 不改 `app.json` 的 `navigationBarTitleText`：它是小程序静态配置，不经运行时数据层。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_store_identity.js` 八项全过；base_sha 树上六项红；两端既有测试不回归；`lint_wx.py` 通过；归档门禁失败集合与 base 逐行一致。
