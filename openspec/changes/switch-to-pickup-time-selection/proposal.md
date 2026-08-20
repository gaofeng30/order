## Why

生效 spec `mvp-product-baseline` 的 `Every first-phase order uses one discrete pickup time` 规定：仅预约取餐、可预约今天与明天、每餐段一个固定截单时刻、取餐时间为离散时间点且粒度商户可配、取餐时间是约定时刻而非到场窗口。

两端实现与之不符：小程序的 `RESERVE_DATES` 仍是今天/明天/后天，取餐时段是一条写死的数组且与餐段无关，没有截单判定，日期与时间在结算页选择而不是菜单顶部。PC 后台的营业设置只有一个门店级 `cutoff` 与营业起止时间，没有按餐段的配置，也没有取餐时间粒度。

`adopt-six-state-order-lifecycle` 删除订单 `type` 字段后还留下三处未清理的读取点，其中 `canCancelReserve` 判定 `o.type === 'reserve'` 导致**取消预约按钮永远不出现**——那是上一 change 的回归，本 change 一并修复。

## What Changes

- 数据层重建预约模型：`RESERVE_DATES` 收敛为今天与明天；新增 `MEAL_PERIODS`（含固定截单时刻与取餐起止）与 `PICKUP_STEP_MIN`；新增 `pickupTimes` / `isPeriodCutOff` / `isDateCutOff` / `defaultPickup` / `pickupLabel`。取餐时间点由范围与粒度推导，不再写死。
- 新增跨页共享的 `pickup` 访问器，默认值为当前时刻之后第一个未截单的时间点。
- 菜单页新增顶部取餐时间条与选择弹层：日期两段、时间按餐段分组，已截餐段整组折叠并标注截止时刻，全截日期标注「今日已截单」。
- 结算页改为只读展示所选取餐时间，提供回菜单修改的入口，删除原有的日期与时段选择控件；提交时重新校验截单，已截则拦截并保留购物车。
- 订单新增 `mealPeriod` 字段，`pickupLabel` 由共享态推导。
- PC 后台营业设置由门店级单一截单改为**按餐段配置**（截单时刻、取餐起止）加一个全局取餐时间粒度，契约层补齐校验。
- 修复 `adopt-six-state-order-lifecycle` 遗留的三处已删除 `type` 字段读取点：`canCancelReserve`、`result.js`、`order-detail.wxml` 的 `reserve` 绑定。

## Capabilities

### Modified Capabilities

- `miniprogram-scope-conformance`: 追加「取餐时间一次选定并跨页共享」「已截餐段整组折叠且提交时重新校验」「取消资格只依赖状态与剩余时间」三条。
- `web-admin-scope-conformance`: 追加「按餐段配置截单与取餐时间」一条。

## Impact

- Owner：branch `worktree-pickup-time-selection`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/pickup-time-selection`。
- Owned paths：`apps/wechat-miniprogram/**`、`apps/web-admin/**`、`openspec/changes/switch-to-pickup-time-selection/**`。
- Dependency：`adopt-six-state-order-lifecycle`，已集成并归档于 `base_sha`。
- Non-goals：**不实现按餐段过滤商品**——`catalogStore` 的商品契约没有餐段字段，服务端不下发，加不了；该能力需后端先扩展商品 schema。不实现服务端定时排产与自动截单；不实现取餐点多选；不改后端。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：可预约日期为今天与明天；取餐时间点由范围与粒度推导；已截餐段折叠且日期全截可识别；菜单顶部条选定后结算只读复用；提交时截单拦截且购物车保留；PC 后台按餐段配置且校验生效；已删除的 `type` 字段全端零读取；既有 UI1 回归全部通过；diff 只包含 owned paths。
- 工具边界：仓库未安装 `openspec` CLI，strict 校验记 `BLOCKED_EXTERNAL`。小程序 UI1 来自 Node harness，PC UI1 来自本地静态服务器加浏览器实际运行。
