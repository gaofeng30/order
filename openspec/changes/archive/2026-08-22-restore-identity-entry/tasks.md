## Red

- [x] 1. 写 `checks/check_identity_entry.js`，十二项（六项新增 + 六项接管）。
- [x] 2. base_sha `349f79b` 上运行 → `IDENTITY_ENTRY_GATE=FAIL (4/12)`。
  - 四项红：入口页仍是首页、商户端入口不触发微信授权、拒绝与允许两条回调路径都不存在。
- [x] 3. 记录基线：小程序 111 pass / 0 fail；归档门禁 39 项，16 项 FAIL。

## Green

- [x] 4. `app.json` 入口页改回 `pages/launch/launch`，`pages/home/home` 保留在清单中。
- [x] 5. 商户端卡片由 `<view bindtap="go">` 改为 `<button open-type="getPhoneNumber" bindgetphonenumber="onMerchantPhone">`。
  - **关键边界**：不恢复被 `browse-before-identity` 删掉的自绘弹层。那个弹层不调任何微信接口，「允许」只是跳页；生效 spec 因此立下「MUST NOT 保留不调用微信接口的假授权控件」。用微信自己的控件既满足项目方要求，又不触碰那条。
- [x] 6. `onMerchantPhone` 两条分支：
  - 拒绝（detail 无 `code` / `encryptedData`）→ 留在本页并说明商户端为何需要验证。**不渲染成失败** —— 拒绝是合法选择，渲染成失败会诱导用户反复尝试。
  - 允许 → 进商户端，同时 toast「已授权 · 身份校验待服务端接入」。这句不能省：省掉它，演示现场看到的就是一个「验证通过」的假象，而实际什么都没验。
- [x] 7. `launch.wxss` 补 `.id-card--btn`（去掉微信 `<button>` 的默认边框、背景与 `::after` 描边）与 `.id-hint`。花括号 25/25 平衡。
- [x] 8. PRD §4.4 回写：入口为身份选择页；商户端入口触发手机号授权；明写「服务端比对能力就位前前端不得声称校验已发生」；保留个人中心「商户登录」作为第二条入口；启动仍应静默 `wx.login` 且不得弹授权界面。

## 本地验证

- [x] 9. `IDENTITY_ENTRY_GATE=PASS`（12/12）。
- [x] 10. `node --test tests/*.test.js` → 117 pass / 0 fail（既有 111 项 + 新增 6 项）。
  - 两条活测试钉住了旧事实：`browse-first-ui1.test.js` 的入口页断言、`entry-screens-ui1.test.js` 的 `data-to="admin-orders"`。均按新事实更新并保留原意 —— 后者改为断「模板有 getPhoneNumber 控件 + 脚本落点是 admin-orders 且不是已删除的 admin-dashboard」。
- [x] 11. `python3 tests/lint_wx.py .` → `WX_LINT=PASS`。
- [x] 12. 归档门禁 39 项与 base 逐行 diff：**仅 `browse-before-identity/check_browse_first.js` 由 PASS 转 FAIL**，即 proposal 记录的接管。
- [x] 13. UI1：`tests/identity-entry-ui1.test.js` 六项，驱动真实 `Page` 对象。
  - 覆盖：入口页与首页可达、用户端零索取、商户端是微信控件而非自绘弹层、拒绝留在本页且不是错误、允许进商户端且明示校验待服务端、前端无任何角色判定。

## 明确未做

- **商户身份鉴权本身**。比对商户账号名单、绑定 openid、四个 `admin-*` 页的接口级鉴权全部属服务端（§4.4 末条：客户端菜单隐藏不能代替鉴权；§16.5 待补齐）。本 change 只做入口与授权触发，门禁另有一项断言前端不得出现 `isMerchant` / `checkRole` 一类判定，避免把「前端拦了一下」误当成已鉴权。

## 独立验证

- [x] 14. 候选 SHA `bb138bb`。
- [x] 15. 在干净 detached worktree 对 `bb138bb` 只读验证：`DIRTY=0`；`IDENTITY_ENTRY_GATE=PASS`；117 pass / 0 fail；`WX_LINT=PASS`；归档门禁 39 项与 base 逐行 diff 仅 `check_browse_first.js` 一行由 PASS 转 FAIL，即已记录的接管。
