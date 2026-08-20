## Why

生效 spec `mvp-product-baseline` 的 `First-phase scope is closed and singular` 已把会员等级与优惠券列入一期排除范围且不得预留；`Employee pricing uses one global discount rate applied per product` 规定定价机制为全局单一折扣率。

`remove-member-coupon-capability` 已删除小程序端的该能力（`bc40b9b`）。PC 网页后台仍保留同一套：4 个页面（891 行）、侧边导航「会员与营销」分组、四组 mock 种子（含真实姓名与手机号）、契约层五组接口，以及删除菜品时的摘券联动。

两端的 mock 数据层相互独立，因此这一保留不造成跨端数据不一致，但它使 PC 后台仍向商户暴露一期不交付的功能入口，且契约层与小程序端已经分叉。

## What Changes

- 删除 4 个页面模块：`pages/levels.js`、`pages/members.js`、`pages/member-import.js`、`pages/coupons.js`，并从 `index.html` 移除对应 `<script>` 标签。
- 删除侧边导航的「会员与营销」分组、其四条路由，以及 `p2` 二期能力标记与顶栏副标题对应文案。
- 删除 `window.__store` 的 `levels` / `members` / `coupons` / `couponUsed` 初始化。
- 删除 `data/seed.js` 的 `LEVELS` / `MEMBERS` / `COUPONS` / `MY_COUPON_USED` 种子。
- 删除 `data/api.js` 中会员等级、会员名单、名单导入、优惠券与用户卡包五组契约，以及 `deleteProduct` 的摘券联动与 `disabledCoupons` 返回值。
- 清理 `pages/products.js` 的摘券确认文案与 toast、`ui/drawer.js` 的过时注释、`app.css` 的二期能力样式分节。

## Capabilities

### New Capabilities

- `web-admin-scope-conformance`: PC 网页商户后台加载的页面模块、导航分组、内存态、数据契约与文案必须与生效 spec 的一期范围一致，不得存在已排除能力的实现、入口或表述。

### Modified Capabilities

无。

## Impact

- Owner：branch `worktree-remove-member-coupon-admin`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/remove-member-coupon-admin`。
- Owned paths：`apps/web-admin/**`、`openspec/changes/remove-member-coupon-admin-pages/**`。
- Read-only evidence：`openspec/specs/mvp-product-baseline/spec.md`、`docs/product/online-ordering-system-prd-0818-review.md`、`docs/quality/change-quality-gates.md`。
- Dependency：`remove-member-coupon-capability`，已集成于 `base_sha`。本 change 不共享其 owned paths。
- Non-goals：不动 `apps/wechat-miniprogram/**`、`services/**`、生效 spec 或产品文档；不实现全局折扣率；不删除标签、过敏原、月售、库存位；不为 `apps/web-admin` 建设浏览器 runner（属独立立项）。
- Gate：`gate_type=W2`（删除商户可见页面与导航入口，属用户可见 UI 行为变更）；`ui_level_target=UI1`；`ui_level_actual=UI0`；**UI1 = `BLOCKED_EXTERNAL`**。
- **已知硬边界**：`docs/quality/change-quality-gates.md` 的决策表把 W2×UI0 标为硬阻断，而 `apps/web-admin` 仓库内没有任何浏览器或等价 runner，该文档自身亦记载「当前没有锁定的浏览器/微信 runner；缺少 UI1 资产即 `BLOCKED_EXTERNAL`，没有当前 PASS 命令」。因此本 change 无法取得 UI1 PASS。恢复条件：为 `apps/web-admin` 引入浏览器或 DOM 级 runner（独立 change）。本 change 取到的最强证据为数据层的**运行态**断言加页面/导航/文案的静态断言，不冒充 UI1。
- 最小成功标准：会员券在 PC 后台无任何页面模块、脚本标签、导航分组、内存态、种子数据、契约方法与文案残留；菜品、订单、分类与营业设置契约在删除后仍可调用；全部 JS 可解析；diff 只包含 owned paths。
- 工具边界：仓库未安装 `openspec` CLI，`openspec validate --strict` 无法执行，记 `BLOCKED_EXTERNAL`。
