## Why

小程序商户端暴露的能力超出客户授权。2026-08-19 评审 §26 规定「完整商品配置、退款、名单和财务功能留在 PC 后台」，§38 规定「菜品页只允许切换可售 / 售罄」，0818 PRD §3.5 规定小程序商户端共 4 屏。

实际有 9 屏：除订单管理、订单详情、扫码核销、菜品四屏外，还有菜品编辑、分类管理、营业设置、开屏图层、商户中心。菜品页除售罄切换外还提供「上架新菜」横幅、每行「编辑」入口、点图片进编辑、上下架开关——这些按 §26 都属 PC 能力。

`admin-profile` 还留着两个从未实现的占位入口（「交班对账 · 建设中」「成员管理 · 建设中」），用户端个人中心也有一个「设置建设中」，均不在 PRD §5.9 定义的内容里。

`utils/layer.js` 的开屏图层用 `wx.setStorageSync` + `USER_DATA_PATH` 实现，违反 PRD §6.9「配置存服务端，向所有用户下发；不使用本机存储」。编辑页迁往 PC 后它失去唯一写入方，只会渲染一张无法清除的陈旧图片——这已在真实设备上发生。

## What Changes

- 删除 5 个页面：`admin-product-edit`、`admin-categories`、`admin-settings`、`admin-layer`、`admin-profile`。小程序 18 屏 → 13 屏，商户端 9 屏 → 4 屏。
- 菜品页收敛为只有售罄切换：移除 `newProduct` / `edit` / `toggleShelf` 及其模板入口与样式。
- 营业状态切换从已删的 `admin-settings` 移到订单页顶部（PRD §16.3 P1 由此裁决）。
- 订单页导航栏右侧由「商户中心」改为「切换身份」，保证收敛后仍可返回身份选择页。
- 删除 `utils/layer.js` 与 `components/layer-overlay`，身份选择页不再渲染本机存储的图层。
- 删除用户端个人中心的「设置」占位入口（PRD §5.9 未定义该项）。
- 裁决 0818 PRD §16.3 P1，并更新 §16.5 的开屏图层条目。

## Capabilities

### Modified Capabilities

- `miniprogram-scope-conformance`: 修订「商户 TabBar 三格」requirement（商户中心改为不得存在，切换身份移至订单页）；新增「商户端恰好四屏」与「不得渲染本机存储的开屏图层」两条。

## Impact

- Owner：branch `worktree-merchant-scope`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/merchant-scope`。
- Owned paths：`apps/wechat-miniprogram/**`、`openspec/changes/collapse-merchant-scope/**`、`docs/product/online-ordering-system-prd-0818.md`。
- Dependency：`reconcile-baseline-alignment`，已集成并归档于 `base_sha`。
- Non-goals：不动 `apps/web-admin/**`（上架新菜、分类管理、营业设置、开屏图层在 PC 侧均已实现，本 change 只是收回小程序端的重复入口）；不新建员工折扣白名单与商户账号名单（PC 侧两份名单均未建，属独立 change）；不实现开屏图层的服务端下发接口。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：小程序商户端恰好 4 屏；菜品页只有售罄切换且该链路端到端可用；营业状态可从订单页切换；切换身份可达；全端无对已删页面的引用与「建设中」占位；无本机存储图层；既有回归全部通过。
- 工具边界：仓库未安装 `openspec` CLI，strict 校验记 `BLOCKED_EXTERNAL`。UI1 来自 Node harness 与 WXSS/WXML 结构 lint，非微信开发者工具或真机。
