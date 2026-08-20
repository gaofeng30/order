## Why

2026-08-19 客户评审记录第 8 条与第 35 条删除了会员等级、优惠券与「我的优惠券」入口，改为 PC 后台维护手机号白名单加一个整单折扣。该结论已于 `e451f52` 落入生效 spec `openspec/specs/mvp-product-baseline/spec.md`：requirement `First-phase scope is closed and singular` 明确把会员等级与优惠券列入排除范围，且不得以任何形式预留；`Employee pricing uses one global discount rate applied per product` 规定定价机制为全局单一折扣率。

小程序端目前仍实现着这套被删除的能力：7 个页面（148K）、一个算价引擎、一个券票卡组件、四组 mock 数据与一整套后端接口契约，并在结算页、个人中心、商户中心和菜品编辑页有 9 处外溢引用。

这些代码百分之百要删。留着有三个具体成本：`utils/api.js` 的文件头注释仍把会员券写成「二期能力」，与生效 spec 直接矛盾；结算页的算价链仍是「原价小计 → 等级折扣 → 减券 → 应付」，与 spec 规定的「原价小计 → 员工折扣 → 应付」不一致；任何读这份代码的人都会据此误判一期范围。

## What Changes

- 删除 7 个会员券页面：`my-coupons`、`admin-levels`、`admin-members`、`admin-member-edit`、`admin-member-import`、`admin-coupons`、`admin-coupon-edit`，并从 `app.json` 移除对应路由（27 屏 → 20 屏）。
- 删除算价引擎 `utils/promo.js` 与券票卡组件 `components/coupon-card`。
- 删除 `app.js` 的 `levels` / `members` / `coupons` / `couponUsed` 全局态，以及 `utils/data.js` 的 `LEVELS` / `MEMBERS` / `COUPONS` / `MY_COUPON_USED` 种子。
- 删除 `utils/api.js` 中会员等级、会员名单与优惠券的全部接口契约。
- 结算页：删除选券弹层与优惠明细卡；算价链收敛为「商品小计 = 应付」；订单不再固化 `levelName` / `levelLabel` / `levelCut` / `couponName` / `couponCut` / `totalCut`。
- 个人中心：删除「我的优惠券」入口与会员等级胶囊。
- 商户中心：删除「会员与营销」分组。
- 菜品编辑页：删除「删除菜品时自动摘除相关券」的联动。
- UI1 测试：既有 `all-scope promo` 用例改为无券路径；新增 scope 一致性用例断言上述能力在小程序端不存在。

**等级折扣一并删除，不留占位。** 删除后结算页只显示商品小计，前端在实现全局折扣率之前不展示任何优惠。理由见 `design.md`。

## Capabilities

### New Capabilities

- `miniprogram-scope-conformance`: 微信小程序端暴露的页面、入口、全局态、接口契约与算价链必须与生效 spec 的一期范围一致，不得存在已排除能力的实现或入口。

### Modified Capabilities

无。

## Impact

- Owner：branch `worktree-remove-member-coupon`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/remove-member-coupon`。
- Owned paths：`apps/wechat-miniprogram/**`、`openspec/changes/remove-member-coupon-capability/**`。
- Read-only evidence：`openspec/specs/mvp-product-baseline/spec.md`、`docs/product/online-ordering-system-prd-0818-review.md`、`docs/product/online-ordering-system-prd-0818.md`。
- Dependency：生效 spec 的一期范围与定价 requirement，已随 `realign-mvp-product-baseline` 与 `complete-baseline-traceability` 集成于 `base_sha`。
- Non-goals：不动 `apps/web-admin/**`（PC 后台的同类删除由 `remove-member-coupon-admin-pages` 承担，该 change 的 UI1 证据受 runner 缺失阻塞）；不实现全局折扣率（属新增类 change）；不删除标签、过敏原、月售、库存位（属 `strip-retired-catalog-fields`）；不删除品牌选择页与商户端工作台（属 `remove-retired-entry-screens`）；不修改 `services/**`、生效 spec 或产品文档。
- Gate：`gate_type=W2`（删除用户可见页面与结算页优惠行，属用户可见 UI 行为变更）；`ui_level_target=UI1`；external assets none。
- 最小成功标准：会员券在小程序端无任何页面、路由、组件、全局态、种子数据与接口契约残留；结算页在无券路径下可完成加购到下单全链路；既有 UI1 回归全部通过；diff 只包含 owned paths。
- 工具边界：仓库未安装 `openspec` CLI，`openspec validate --strict` 无法执行，记 `BLOCKED_EXTERNAL`。UI1 证据来自 `apps/wechat-miniprogram` 的 Node 测试 harness，非微信开发者工具或真机，因此不声称 UI2/UI3。
