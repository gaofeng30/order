## Why

2026-08-19 客户评审记录删除了一批入口屏与首页营销位：§1 不展示洗衣洗车也不显示「即将上线」、§34 删营销 Banner、§35 删入群与会员中心入口、§36 批注「都删掉」（推荐商品与标签一起删）、§38 删小程序商户端经营工作台且底部只保留订单/核销/菜品。

小程序端仍完整保留这些：品牌选择页、商户端工作台、首页 Banner 轮播、入群二维码面板、今日招牌横滑卡、五格商户 TabBar。

同时发现一处**跨 spec 冲突**：生效 spec `miniprogram-menu-catalog` 的 `Home and menu expose complete recoverable list states` 明确要求「首页招牌 MUST 按 server category 顺序再按各自 product 顺序 flatten 后取前四件」。该 requirement 成文于 2026-08-13，早于客户评审。评审记录按 `mvp-product-baseline` 的 authority order 属第 1 顺位，因此该条款需一并修订，否则删除代码将违反生效 spec。

## What Changes

- 删除品牌选择页 `pages/brand` 与商户端工作台 `pages/admin-dashboard`，并从 `app.json` 移除路由（20 屏 → 18 屏）。
- 删除路由封装中指向品牌选择页的 `toBrand` 方法。
- 首页删除营销 Banner 轮播、入群与会员面板、今日招牌横滑卡与推荐商品；服务宫格去掉会员中心与联系客服。
- **首页不再请求 catalog，也不再持有任何 list 状态**；商品浏览的唯一入口收敛为菜单。
- 商户端 TabBar 由五格收敛为订单 / 核销 / 菜品三格；在订单页导航栏右侧补一个「商户中心」入口，避免 `admin-profile` 及其下属页面成为孤儿页。
- 修正两处被删除页面牵连的失效引用：身份选择页的商户端入口原指向已删的工作台，改指订单管理；开屏图层编辑页原以品牌页为预览底图，收敛为只预览身份选择页。
- 删除仅由工作台使用的 `RANK` 种子。
- 修订生效 spec `miniprogram-menu-catalog`：list 状态义务收敛到菜单，首页明确 MUST NOT 请求 catalog 或渲染商品位。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `miniprogram-scope-conformance`: 追加「不得暴露已废止入口屏」「首页不得承载营销位」「商户 TabBar 只有三格且商户中心保持可达」「仅供已删页面使用的种子一并移除」四条约束。
- `miniprogram-menu-catalog`: list 状态与 retry 义务收敛到菜单；首页从该 requirement 中移除，原「首页招牌取前四件」作废。

## Impact

- Owner：branch `worktree-remove-retired-entry-screens`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/remove-retired-entry-screens`。
- Owned paths：`apps/wechat-miniprogram/**`、`openspec/changes/remove-retired-entry-screens/**`。
- Read-only evidence：`openspec/specs/mvp-product-baseline/spec.md`、`openspec/specs/miniprogram-menu-catalog/spec.md`、`docs/product/online-ordering-system-prd-0818-review.md`。
- Dependency：`remove-member-coupon-capability`（已集成于 `base_sha`，共享 `apps/wechat-miniprogram/**`，串行执行）。
- Non-goals：不动 `apps/web-admin/**`、`services/**` 或产品文档；不删除首页的「到店点单」入口（属即时单删除，与确认页取餐方式切换、订单类型和状态机同属一个 change）；不删除商户端的营业设置、开屏图层与分类管理（属迁 PC 的独立 change，且 0818 PRD §16.3 P1「营业状态切换归属」尚未拍板）；不删除标签、过敏原、月售、库存位（属 `strip-retired-catalog-fields`）。
- Gate：`gate_type=W2`（删除用户与商户可见页面、首页营销位、TabBar 项，属用户可见 UI 行为变更）；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：品牌页与工作台在路由与磁盘上均不存在；首页无任何营销位且不请求 catalog；商户 TabBar 只有三格且商户中心可达；无任何指向已删页面的失效引用；既有 UI1 回归全部通过；diff 只包含 owned paths。
- 工具边界：仓库未安装 `openspec` CLI，`openspec validate --strict` 无法执行，记 `BLOCKED_EXTERNAL`。UI1 来自 `apps/wechat-miniprogram` 的 Node 测试 harness，非微信开发者工具或真机，不声称 UI2/UI3。
