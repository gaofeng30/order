## Red

- [x] 1. 核实用户反馈，不靠推断。
  - `app.json` 的 `pages[0]` 是 `pages/launch/launch` —— 所有人启动即落身份选择页。
  - 该页 `用户端` 卡片绑 `openAuth`（弹授权），`商户端` 卡片绑 `go`（直接 `navigateTo`，零验证）。**需要手机号的一侧敞开，不需要的一侧拦着。**
  - 与 §14 第一条「启动时不弹手机号授权」、§4.4「未绑定直接进用户端首页」直接冲突。
- [x] 2. 同页另发现两处：`toBrand()` 调用的 `nav.toBrand` 已随 §0.2 删除品牌选择页而不存在，点返回键抛 `TypeError`；`home.wxml` 的 `<navbar exit>` 给所有人留了一条回身份选择页的路。
- [x] 3. 写 `checks/check_browse_first.js`，九项。
- [x] 4. base_sha `7865793` 上运行 → `BROWSE_FIRST_GATE=FAIL (6/9)`，六项红覆盖入口页、弹窗状态与处理器、用户端卡片绑定、假授权控件、首页 exit、死处理器。
- [x] 5. 记录基线：小程序 102 pass / 0 fail；归档门禁 36 项，14 项 FAIL。

## Green

- [x] 6. `app.json` 页面顺序调整，`pages/home/home` 成为入口；身份选择页保留在清单中（§3.5 用户端 9 屏之一）。
- [x] 7. `launch.js` 重写：删 `auth` 状态、`openAuth` / `closeAuth` / `allowAuth`、内联微信图标常量；`toBrand` 改为 `nav.back()`。
- [x] 8. `launch.wxml`：用户端卡片改 `data-to="home" bindtap="go"`，删整个弹窗块，返回键改绑 `back`。
- [x] 9. `launch.wxss`：按规则边界整块删除 12 条 `.auth-*` 样式，删后花括号 22/22 平衡。
- [x] 10. `home.wxml`：`<navbar>` 去掉 `exit` 属性（不改 `navbar` 组件本身，其他页面可能仍需该能力）。

## 门禁自身的一次收窄

- [x] 11. 「无假授权控件」原按 `/授权/` 判定，命中了 `profile.wxml` 的 `<pill s="已授权">` —— 那是**状态标签**不是控件。收窄为只匹配「申请获取 / 授权登录 / 获取…手机号」这类请求话术。又一次断言词而非断言事实。

## 本地验证

- [x] 12. `BROWSE_FIRST_GATE=PASS`（9/9）。
- [x] 13. `node --test tests/*.test.js` → 111 pass / 0 fail（既有 102 项不回归 + 新增 9 项）。
- [x] 14. `python3 tests/lint_wx.py .` → `WX_LINT=PASS`。
- [x] 15. 归档门禁 36 项与 base 逐行一致，**无接管**。
- [x] 16. UI1：`tests/browse-first-ui1.test.js` 九项，驱动真实 `Page` 对象。
  - 覆盖：入口页是首页、首页开屏无任何授权状态且不导走用户、浏览四页不索取手机号、身份选择页无弹窗残留、用户端卡片直达首页、商户端入口仍可用、模板绑定的处理器全部存在、个人中心仍能回到身份选择页、首页无 exit 捷径。

## 本 change 之外

- **商户端仍无鉴权**。四个 `admin-*` 页任何人 `navigateTo` 即可进入。§4.4 末条明写「客户端菜单隐藏不能代替鉴权」，真正的防线是这四页的数据接口各自由服务端校验角色。§16.5 已列为待补齐，本 change 不制造已解决的错觉。
- **不给商户端卡片加授权弹窗**。§4.4 的绑定入口在个人中心的「商户登录」，在身份选择页新增一条授权路径属 PRD 之外的新增设计。
- `profile.wxml` 的 `<pill s="已授权">` 在身份链路接通前恒为「已授权」，与实际状态无关；属该页内容问题，另行处理。

## 独立验证

- [ ] 17. 提交产生候选 SHA。
- [ ] 18. 在干净 detached worktree 对该精确 SHA 只读重跑 12–16。
