## Why

§5.1 给首页列了六项内容，`pages/home/home.wxml` 只做到三项（门店名、地址、三个入口）。缺的三项都不是装饰：

- **门店公告**。PC 营业设置里早就有这个字段（`SETTINGS.notice`），设置页上还写着「展示在用户端首页」。商户填了，用户看不到 —— 这是一条已交付的配置能力接不上出口。
- **当前营业状态**。首页只显示一句写死的 `store.cutoff`（「今日 16:30 截单」）。商户在小程序商户端订单页把门店切成「休息中」时，用户端首页毫无反应，照样引导他去点单。
- **进行中订单提示条**。§5.1 要求「存在 `已预约` / `制作中` / `待取餐` 订单时页面顶部常驻一条」，且订单进入 `待取餐` 后文案变为「已备好，可取餐」并高亮。这条提示条同时是 §5.10 的**兜底**：订阅消息需要用户主动授权，拒绝订阅的用户只能靠首页提示条和订单状态知道餐好了。没有它，一个拒绝订阅的用户就没有任何渠道得知可以取餐了。

另有一处残留：`home.wxml` 仍保留 `item.off` 分支与「未开放」角标的渲染逻辑，是洗衣洗车入口删除后没清干净的死代码。§0.2 明确一期不显示「即将上线」一类占位。`home.js` 的 `grid` 里已无 `off` 项，所以它今天不会显示，但下一个往 `grid` 里加东西的人会以为这是可用能力。

## What Changes

- **首页展示门店公告**，取自营业设置的 `notice` 字段，不写死。
- **首页展示当前营业状态**（营业中 / 休息中 / 已截单），跟随商户端的切换。
- **新增进行中订单提示条**：存在 `已预约` / `制作中` / `待取餐` 订单时常驻，展示单数与最近一单的取餐时间，点击直达该单的取餐码页；任一单进入 `待取餐` 后文案变为「已备好，可取餐」并高亮。
- **删除 `item.off` 死分支与「未开放」角标**及其样式。

## Capabilities

### Modified Capabilities

- `miniprogram-scope-conformance`：新增两条 requirement —— 首页展示门店公告与当前营业状态且均来自配置；首页常驻进行中订单提示条并在备好时改变文案。

## Impact

- Owner：branch `worktree-home`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/home`。
- Owned paths：`apps/wechat-miniprogram/{utils/data.js,pages/home/**}`、`apps/wechat-miniprogram/tests/{home-screen-ui1.test.js,entry-screens-ui1.test.js}`、`openspec/changes/complete-home-screen/**`、`openspec/specs/miniprogram-scope-conformance/spec.md`。
- Non-goals：
  - **不接开屏图层**。§5.1 还要求首页叠加开屏透明 PNG，但该能力等服务端下发接口（§16.5），本地存储实现已按设计移除，本 change 不重新引入。
  - **不做订阅消息**。提示条是订阅被拒时的兜底，两者互补但订阅链路属另一件事（§16.5）。
  - 不改营业设置页、不改商户端的营业状态切换、不改订单模型。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_home_screen.js` 十一项全过；base_sha 树上七项红；小程序既有测试不回归；`lint_wx.py` 通过；归档门禁失败集合与 base 逐行一致。
