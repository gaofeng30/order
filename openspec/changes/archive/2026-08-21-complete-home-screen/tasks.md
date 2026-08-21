## Red

- [x] 1. 写 `checks/check_home_screen.js`，十一项。
- [x] 2. base_sha `b19930c` 上运行 → `HOME_SCREEN_GATE=FAIL (7/11)`。
  - 四项绿中，「无进行中订单时不渲染提示」在 base 上是**平凡通过**（base 根本没有提示条）。它作为非回归守卫有价值，但不算 Red 证据，此处如实记录。
- [x] 3. 记录基线：小程序 94 pass / 0 fail；归档门禁 32 项，14 项 FAIL。

## Green

- [x] 4. `STORE` 种子补 `notice`，取值与 PC `SETTINGS.notice` 一致 —— 接后端时两端本就是同一条记录。
- [x] 5. `app.js` 的 `globalData.store` 由「只有一个状态字段」改为承载完整门店信息，商户可配置项才有落点。
- [x] 6. `home.js` 重写：公告与营业状态读配置不写死；营业状态直接取 `store.status`，不从截单时刻派生（§6.9 允许人工覆盖，派生值只是默认值）。
- [x] 7. 进行中订单提示条：按取餐时刻排序取最近一单，`待取餐` 优先并改为「已备好，可取餐」高亮，点击直达该单。
- [x] 8. 删除 `item.off` 死分支、「未开放」角标及其三条样式（`.grid-item.off`、`.grid-ico.off`、`.grid-tag`）。
  - 按完整规则整块删除而非逐行正则，删后校验花括号 21/21 平衡。

## 命名冲突（发现并修正）

- [x] 9. 首版把提示条命名为 `banner`，被 `entry-screens-ui1.test.js` 的营销 Banner 守卫拦下。
  - **这次拦得对**：`banner` 正是 §0.2 废止的营销 Banner 的名字，在同一个文件里复用一个已删除概念的名字，下一个人读到时无从分辨。改名 `ongoing` / `tapOngoing`，门禁文案同步改为 in-flight strip。本轮第二次同类冲突（前一次是 `sold` 撞月售），两次都说明废止清单里的词不能拿来复用。

## 门禁自身的问题（发现并修正）

- [x] 10. 「营业状态跟随商户」一项原先在同一个 check 里交错创建多个宿主，而 `mp()` 会重置 `global.Page` 与 `require.cache`，后建的宿主让先建的再也拿不到页面定义，报成 `Cannot read properties of null`。改为串行创建。

## 本地验证

- [x] 11. `HOME_SCREEN_GATE=PASS`（11/11）。
- [x] 12. `node --test tests/*.test.js` → 102 pass / 0 fail（既有 94 项不回归 + 新增 8 项）。
- [x] 13. `python3 tests/lint_wx.py .` → `WX_LINT=PASS`。
- [x] 14. 归档门禁 32 项与 base 逐行一致，**无接管**。
- [x] 15. UI1：`tests/home-screen-ui1.test.js` 八项，驱动真实 `Page` 对象。
  - 覆盖：公告来自配置且未写死、营业状态跟随三个取值、无进行中订单不渲染、只计三态、按取餐时刻而非下单时刻排序、备好时改文案且不再强调单数、点击跳对应单、三入口且无占位。

## 非目标（本 change 明确不做）

- 开屏图层：§5.1 也要求首页叠加透明 PNG，但它依赖服务端下发接口（§16.5）。本地存储实现已按 `remove-retired-entry-screens` 的结论移除，重新引入会退回「只会渲染无法清除的陈旧图片」的状态。
- 订阅消息：提示条是订阅被拒时的兜底，两者互补，但订阅链路属另一件事。

## 独立验证

- [x] 16. 候选 SHA `9b00eef`（一次通过，未误纳构建产物）。
- [x] 17. 在干净 detached worktree 对 `9b00eef` 只读验证。
  - `DIRTY=0`；`HOME_SCREEN_GATE=PASS`；`node --test tests/*.test.js` → 102 pass / 0 fail；`WX_LINT=PASS`；归档门禁 32 项与 base 逐行一致。
