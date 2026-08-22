## Why

项目方于 2026-08-22 决定：**小程序入口页为身份选择页**。普通用户点「用户端」直接浏览，商户点「商户端」触发身份验证。

这与 §4.4 现行口径不同。§4.4 写的是「启动时静默 `wx.login` 取 openid，服务端查该 openid 是否已绑定商户手机号。已绑定进身份选择页，未绑定直接进用户端首页」—— 即身份选择页只对已绑定商户可见。该口径的前提是**服务端能给出绑定判定**，而这条链路属 §16.5 待补齐项，至今未接。

在判定不存在的前提下，两种入口安排各有代价：

- 走「未绑定 → 首页」默认分支（`browse-before-identity` 的做法）：符合 §4.4 字面，但商户端在前端演示中无从进入，只能绕道个人中心。
- 走身份选择页：与 §4.4 的路由描述不符，但把选择权交回使用者，且**闸门落在真正需要它的一侧**。

项目方选择后者。这是一个产品决策，本 change 据此调整实现并回写 §4.4。

## 一个必须守住的边界

被 `browse-before-identity` 删掉的旧弹窗是**假的**：它不调用任何微信接口，`allowAuth()` 只是 `nav.go('home')`。生效 spec 因此立下一条 —— 「MUST NOT 保留不调用微信接口的假授权控件」。

本 change **不恢复那个弹窗**。商户端入口改用微信真实的 `<button open-type="getPhoneNumber">`，点击弹出的是微信自己的授权面板。授权返回的是加密数据，比对商户账号名单必须由服务端完成（§4.4），因此「允许」之后放行进商户端，但明确告知**身份校验待服务端接入** —— 演示能走通，同时不制造「已经在验证」的错觉。

## What Changes

- **入口页改回 `pages/launch/launch`**。
- **用户端卡片**：直接进首页，不索取任何身份（§14 不变）。
- **商户端卡片**：改为 `open-type="getPhoneNumber"` 的按钮，触发微信真实授权面板。
  - 用户拒绝 → 留在身份选择页并说明原因。
  - 用户允许 → 进商户端，同时提示身份校验待服务端接入。
- **PRD §4.4 回写**：入口为身份选择页；商户绑定入口除个人中心外，增加身份选择页的商户端卡片。
- **生效 spec MODIFY**：`browse-before-identity` 那条「入口页 MUST 是用户端首页」按新决策修订，其余部分（不弹授权即可浏览、无假授权控件）原样保留。

## Capabilities

### Modified Capabilities

- `miniprogram-scope-conformance`：MODIFY 一条 requirement（入口页与身份选择页的可见性），ADDED 一条（商户端入口必须触发真实微信授权且不得伪装校验结果）。

## 与已归档门禁的冲突

`openspec/changes/archive/2026-08-22-browse-before-identity/checks/check_browse_first.js` 断言 `app.json` 的首个页面是用户端首页。该断言直接由本 change 的产品决策推翻，属正当事实变更。本 change 的门禁**接管**该门禁的断言集：入口页那条按新决策重写，其余（浏览路径不索取身份、无假授权控件、返回控件可达）原样吸收。

## Impact

- Owner：branch `worktree-identity-entry`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/identity-entry`。
- Owned paths：`docs/product/online-ordering-system-prd-0818.md`（仅 §4.4）、`apps/wechat-miniprogram/{app.json,pages/launch/**,tests/{identity-entry-ui1.test.js,browse-first-ui1.test.js,entry-screens-ui1.test.js}}`、`openspec/changes/restore-identity-entry/**`、`openspec/specs/miniprogram-scope-conformance/spec.md`。
- Non-goals：
  - **不实现商户身份鉴权**。比对商户账号名单、绑定 openid、四个 `admin-*` 页的接口级鉴权全部属服务端（§4.4 末条「客户端菜单隐藏不能代替鉴权」、§16.5）。本 change 只做前端入口与授权触发，不制造已解决的错觉。
  - 不恢复被删的假授权弹窗，不新增任何不调用微信接口的「校验」控件。
  - 不改个人中心的「商户登录」入口（§4.4 原有路径，保留）。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_identity_entry.js` 十二项全过（含接管来的六项）；base_sha 树上四项红；小程序既有测试不回归；`lint_wx.py` 通过；除被接管的 `check_browse_first.js` 外归档门禁失败集合与 base 逐行一致。
