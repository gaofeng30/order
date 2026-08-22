## Red

- [x] 1. 写 `checks/check_store_identity.js`，八项。
- [x] 2. base_sha `11fe9dd` 上运行 → `STORE_IDENTITY_GATE=FAIL (6/8)`。
  - 其中一项指出了一个真实错误：`the merchant order detail does not render the order pickup snapshot` —— 商户端订单详情把「门店分店名」当成了取餐地点。
- [x] 3. 记录基线：小程序 102 pass / 0 fail；归档门禁 36 项，14 项 FAIL。

## Green

- [x] 4. 两端 `STORE`：删 `branch`、`addr` 落正式值 `党政办公中心后院老食堂`。
  - 删而不是设成 `'绥安食品'`：留一个恒等于门店名的字段，下一个人仍要在 `name` 和 `branch` 之间挑一个 —— 上一个 change 里 `pickupPoint` / `pickupWindow` 的分裂就是这么来的。
- [x] 5. 六处 `store.branch` 消费点改指正确来源：门店名处用 `store.name`（menu、web-admin/app.js），取餐窗口副标题只留地址（confirm）。
- [x] 6. **修正 `admin-order-detail.wxml` 的取餐地点**：由 `{{store.branch}}` 改为 `{{o.pickupPoint}}`。
  - §7.2 要求生成订单时固化取餐点快照，§15.6.2 的 `pickupPoint` 就是它。渲染门店当前配置意味着配置一改历史订单跟着改，与快照的意义直接相反。用户端本来就读订单快照，商户端漏了。
- [x] 7. 三处硬编码的 `绥安食品 · 县前直营店` 改为读配置（home、launch、PC layer 预览）；`launch.js` 因此补 `store` 到页面数据。
- [x] 8. PRD §13.3：该行标记已确认并写入正式值，注明取餐点是该地址的北门。

## 本地验证

- [x] 9. `STORE_IDENTITY_GATE=PASS`（8/8）。
- [x] 10. `node --test tests/*.test.js` → 102 pass / 0 fail，无回归。
- [x] 11. `python3 tests/lint_wx.py .` → `WX_LINT=PASS`。
- [x] 12. 归档门禁 36 项与 base 逐行一致，**无接管**。
- [x] 13. UI1：门禁驱动真实 `Page`，断言首页读配置门店名（改配置后页面跟着变）、商户端与用户端订单详情对同一单展示同一个取餐地点。

## 本 change 之外

- `绥安食品` 仍在 `launch.wxml`、`menu.wxml`、`index.html`、`layer.js` 等约七处硬编码。**这些值是对的**，不是缺陷，只是耦合。全部改为配置读取是一次独立重构，会把本 change 改动面扩大约三倍，而收益（将来改门店名时少改几处）在单门店一期里趋近于零。本 change 只改值错的与本来就要动的那几行。
- `app.json` 的 `navigationBarTitleText` 是小程序静态配置，不经运行时数据层，不在此列。

## 独立验证

- [x] 14. 候选 SHA `397e14d`。
- [x] 15. 在干净 detached worktree 对 `397e14d` 只读验证：`DIRTY=0`；`STORE_IDENTITY_GATE=PASS`；102 pass / 0 fail；`WX_LINT=PASS`；归档门禁 36 项与 base 逐行一致。
