## 0. 门禁声明

```yaml
change: switch-to-pickup-time-selection
gate_type: W2
ui_level_target: UI1
ui_level_actual: UI1
owner: worktree-pickup-time-selection
worktree: .claude/worktrees/pickup-time-selection
owned_paths:
  - apps/wechat-miniprogram/**
  - apps/web-admin/**
  - openspec/changes/switch-to-pickup-time-selection/**
base_sha: 87098b9e55aba7cfeaaf2f6d28cdfd8313269347
candidate_sha: external-post-commit
external_assets: none
dependencies:
  - adopt-six-state-order-lifecycle（已归档于 base_sha）
```

门禁命令：

```
cd apps/wechat-miniprogram && npm test
node openspec/changes/switch-to-pickup-time-selection/checks/check_pickup_settings.js <tree-root>
```

## 1. Boundary and approval

- [x] 1.1 确定单一验收判据。
  - Evidence: 「可预约日期为今天与明天；取餐时间点由范围与粒度推导；已截餐段折叠；菜单顶部条选定后结算只读复用；提交时截单拦截且购物车保留；PC 后台按餐段配置且校验生效」。
- [x] 1.2 确定按餐段过滤商品为硬边界。
  - Evidence: spec 要求菜单按所选取餐时间的餐段过滤商品，但商品来自 API，`catalogStore` 契约无餐段字段，服务端不下发。前端硬做只能靠本地 mock 覆盖服务端数据，而 `miniprogram-menu-catalog` 明令禁止。声明为非目标，待后端扩展商品 schema。
- [x] 1.3 确定顺带修复上一 change 的三处遗留。
  - Evidence: `adopt-six-state-order-lifecycle` 删除订单 `type` 后遗留三处读取点，其中 `canCancelReserve` 判 `o.type === 'reserve'` 导致**取消预约按钮永远不出现**（功能性回归）。三处均落在本 change 的语义范围（取消规则与取餐展示）内，一并修复。

## 2. Red

- [x] 2.1 小程序端：新增 `tests/pickup-time-ui1.test.js`。
  - Red: `node --test tests/pickup-time-ui1.test.js` → `tests 10 / pass 0 / fail 10`。覆盖可预约日期、餐段固定截单与可配粒度、时间点推导、截单判定、默认取餐时间、菜单顶部条与弹层、分组折叠、日期全截标注、结算只读复用、截单拦截、取消资格不依赖已删除的 `type`。
- [x] 2.2 PC 端：新增 `checks/check_pickup_settings.js`。
  - Red: 对 `base_sha` 树执行 → `exit=1`。四项失败：设置仍是门店级单一 `cutoff` 与营业起止、契约无餐段校验、设置页仍编辑 `f-open`/`f-close`/`f-cut`、多个模块仍读被移除的字段。

## 3. Green

- [x] 3.1 重建小程序预约数据模型。
  - Evidence: `utils/data.js` 的 `RESERVE_DATES` 收敛为今天与明天；删除写死的 `RESERVE_SLOTS`；新增 `MEAL_PERIODS`（午餐 11:30 截 / 11:30–13:30，晚餐 17:00 截 / 17:00–19:00）与 `PICKUP_STEP_MIN = 30`；新增 `pickupTimes`（由范围与粒度推导）、`isPeriodCutOff`、`isDateCutOff`、`defaultPickup`、`dateLabel`、`pickupLabel`；`canCancelReserve` 去掉已删除的 `type` 判定。
- [x] 3.2 新增跨页共享的取餐时间访问器。
  - Evidence: `utils/util.js` 新增 `pickup`（`get`/`set`/`label`），默认值取 `data.defaultPickup()`。
- [x] 3.3 菜单顶部取餐时间条与选择弹层。
  - Evidence: `menu.js` 125 → 171 行，新增 `syncPickup` / `buildPicker` / `openPicker` / `closePicker` / `pickPickerDate` / `pickPickerTime`；`menu.wxml` 122 → 169 行，新增顶部条与按餐段分组的弹层，已截组只渲染标题与截止时刻（`times` 为空数组），全截日期标注「今日已截单」；`menu.wxss` 补样式。
- [x] 3.4 结算页改为只读复用。
  - Evidence: `confirm.js` 删除 `slotsFor` / `dates` / `dateIdx` / `slots` / `slot` / `pickDate` / `pickSlot` / `syncPayLabel`，改为 `syncPickup` + `editPickup`（回菜单）；`pay()` 提交前调用 `isPeriodCutOff` 拦截并保留购物车，订单新增 `mealPeriod`；`confirm.wxml` 97 → 94 行删除日期段与时段横滑控件，改为只读展示 + 修改入口；`confirm.wxss` 同步。
- [x] 3.5 PC 后台按餐段配置。
  - Evidence: `data/seed.js` 的 `SETTINGS` 由 `{ openTime, closeTime, cutoff }` 改为 `{ pickupStepMin, mealPeriods[] }`；`pages/settings.js` 89 → 99 行改为逐餐段编辑截单与取餐起止 + 一个粒度输入；`data/api.js` 的 `saveSettings` 补齐四项校验（粒度大于 0、餐段非空、字段必填、结束不早于开始）。
- [x] 3.6 两端门禁至 Green。
  - Green: 小程序 10/10；PC `PICKUP_SETTINGS_GATE=PASS`，对 `base_sha` 树 `exit=1`。

## 4. Refactor and writer gate

- [x] 4.1 修复上一 change 遗留的三处 `type` 读取点，并补通用断言。
  - Refactor: ① `canCancelReserve` 判 `o.type === 'reserve'` —— `type` 已删除，该函数恒返回 false，**取消预约按钮永远不出现**。② `pages/result/result.js` 按 `o.type` 分支渲染标题、图标、文案与按钮，一期只有预约单，全部收敛为预约分支。③ `pages/order-detail/order-detail.wxml` 绑定 `{{reserve}}`，而该字段已从 data 中删除，渲染的是错误分支（显示「预计取餐 + store.pickup」而非实际取餐时间）。
  - 补了两条通用断言：**全端不得读 `o.type`/`item.type`/`order.type`**（扩展到 7 个文件）；**模板不得绑定页面从未设置的字段**（遍历所有页面对 9 个已删字段做交叉检查）。后者把「删了 data 忘了删模板」变成可执行检查。
- [x] 4.2 修正一处依赖隐式默认值的既有用例。
  - Refactor: `adopt-six-state-order-lifecycle` 的 `checkout creates a reserved order` 未显式设定取餐时间，依赖旧默认值。本 change 把默认值改为「当前时刻之后第一个未截单时间点」= 今天 17:00，距取餐 12 分钟，按 §7.4 支付成功即进 `制作中` —— 行为正确，用例假设失效。改为显式 `pickup.set({ off: 1, period: 'lunch', time: '12:00' })`，并**新增**一条 `checkout inside the 30-minute window creates a producing order` 覆盖 30 分钟分支。断言总数增加。
- [x] 4.3 全量回归。
  - Refactor: 小程序 `npm test` → `tests 58 / pass 58 / fail 0`（新用例 11 条挂入 test script，既有 47 条无回归）。PC 门禁 `PICKUP_SETTINGS_GATE=PASS`。
- [x] 4.4 UI1：PC 后台浏览器实际运行。
  - UI1 渲染：营业设置页展示按餐段的三列表单 —— 午餐 截单 `11:30` / 取餐自 `11:30` / 至 `13:30`，晚餐 截单 `17:00` / 取餐自 `17:00` / 至 `19:00`，外加「取餐时间粒度（分钟）= 30」；页面副标题为「状态、餐段截单、取餐时间、取餐点与门店公告」。
  - UI1 保存链路：把晚餐截单改为 `16:30`、粒度改为 `15` 后保存，`__store.settings` 实际变为 `dinner.cutoff = '16:30'`、`pickupStepMin = 15`，Toast 为「设置已保存」。
  - UI1 校验：提交取餐结束早于开始的餐段被拒，错误信息「午餐 的取餐结束时间不能早于开始时间」；提交粒度 0 被拒，错误信息「取餐时间粒度需大于 0」。
  - UI1 残留：`['openTime','closeTime','cutoff']` 在运行态 settings 上的存在性检查为空数组。
  - 控制台：`runtimeErrors` 为空数组。
- [x] 4.5 记录门禁证据与 candidate SHA。
  - Writer verdict: `{ gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 87098b9e55aba7cfeaaf2f6d28cdfd8313269347, candidate_sha: external-post-commit（见 5.1）, hard_blockers: 0, unverified_boundary: 按餐段过滤商品受后端契约阻塞；服务端定时截单与排产无实现可验；`NOW_MINS` 为演示时钟；openspec CLI 缺失 }`。

## 5. Independent verification

- [ ] 5.1 在干净 detached worktree 对精确 candidate SHA 只读验证。
- [ ] 5.2 记录 PASS/FAIL 与剩余外部边界。

## 6. 集成与后续

- 两份 delta MUST 与本 change 一并 archive 后应用到生效 spec。
- **前端能独立推进的部分到此为止。** 后续新增类全部依赖后端：按餐段过滤商品（需扩展商品契约）、按取餐日期的售罄开关、全局折扣率、手机号绑定与双要素、订阅消息、退款流程、支付对账兜底、PC 扫码登录。`services/api` 目前只有 catalog。
