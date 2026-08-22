## Why

订单卡片把预约时间与取餐地点拼成了一条字符串：

```js
timeText: '预约 ' + data.orderPickupLabel(o) + ' · ' + o.pickupPoint,
```

取餐点落成正式值「党政办公中心后院老食堂北门」后，这条串超出一行宽度并折行，而承载它的 `.oc-time` 是 `align-items: center` 的 flex 行 —— 图标随之被垂直居中到两行文字的中间，看上去与任何一行都不对齐。

拼接本身才是问题。这是两条独立事实：**什么时候取**与**去哪儿取**。把它们连成一串意味着渲染宽度一变，两条信息一起塌陷；而且中间那个 `·` 会让「17:00」与地址的第一个字连读成一句话，本来就不好扫。

顺带一提，这条串的长度取决于取餐点配置 —— 也就是说排版正确与否依赖于客户填了多长的地址。这种依赖不该存在。

## What Changes

- **拆成两个字段**：`timeText`（预约时间）与 `placeText`（取餐地点），页面数据里各自独立。
- **模板渲染为两行**，各带自己的图标：时间用日历图标，地点用定位图标。
- **图标对齐首行**：`align-items: flex-start`，使任一行折行时图标仍与第一行文字齐平，而不是漂到中间。

## Capabilities

### Modified Capabilities

- `miniprogram-scope-conformance`：新增一条 requirement —— 订单卡片的取餐时间与取餐地点是两条独立展示项。

## Impact

- Owner：branch `worktree-card-meta`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/card-meta`。
- Owned paths：`apps/wechat-miniprogram/pages/orders/**`、`openspec/changes/split-order-card-meta/**`、`openspec/specs/miniprogram-scope-conformance/spec.md`。
- Non-goals：
  - 不改订单详情页（那里时间与地点本来就是分行的信息行）。
  - 不改商户端订单卡片：它展示的是联系人与下单时刻，没有这条拼接。
  - 不改取餐点取值、不改订单模型。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_card_meta.js` 七项全过；base_sha 树上五项红；小程序既有测试不回归；`lint_wx.py` 通过；归档门禁失败集合与 base 逐行一致。
