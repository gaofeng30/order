## Why

PRD §6.4 要求 PC 后台维护员工折扣白名单与全局折扣率，§4.1 的附加手机号双要素依赖白名单中的姓名字段，§5.6 的算价链依赖全局折扣率。

但 **PC 后台没有这一页**。侧边导航只有经营 / 菜品 / 门店三组，契约层对 `staff` / `whitelist` 零命中。也就是说：全局折扣率没有作用对象，附加手机号双要素没有比对数据源，§6.13.3 的员工批量导入没有落地的主页面。

原「会员名单」页在 `remove-member-coupon-admin-pages` 中随会员券体系一并删除——那删的是会员等级与优惠券，员工白名单是另一件事，一直没有补建。

## What Changes

- 种子新增 `STAFF_WHITELIST`（6 条），字段严格为手机号、姓名两个可填项加 `enabled` / `joinAt` / `bound` / `spend` / `orders` 五个系统字段。
- `SETTINGS` 新增 `discountRate`（整数百分比，默认 85）。
- 契约层新增六个方法：`listStaff`（支持按手机号或姓名搜索）、`saveStaff`、`setStaffEnabled`、`deleteStaff`、`getDiscountRate`、`saveDiscountRate`。校验覆盖手机号格式与唯一性、姓名非空、折扣率整数与范围。
- 新增 PC 页面 `pages/staff.js`：顶部全局折扣率卡片（带实时的「约 N 折」提示），下方名单表格（八列）、搜索、行内启停、编辑抽屉、删除二次确认。
- `app.js` 新增「名单」导航分组；`index.html` 注册页面脚本；`app.css` 补样式。

## Capabilities

### Modified Capabilities

- `web-admin-scope-conformance`: 新增「PC 后台维护员工折扣白名单」与「全局折扣率与白名单同页维护」两条。

## Impact

- Owner：branch `worktree-staff-page`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/staff-page`。
- Owned paths：`apps/web-admin/**`、`openspec/changes/add-staff-whitelist-page/**`。
- Dependency：`add-product-meal-period`，已归档于 `base_sha`。
- Non-goals：不建商户账号名单页（§4.4，独立 change）；不建批量导入页（§6.13.3，下一步）；不接后端（PC 仍为 mock，项目方选择 B）；不实现折扣率对下单算价的实际作用（用户端算价属后端范围）。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：种子字段严格；六个契约方法齐全且校验生效；编辑与切换状态不重置系统字段；折扣率为整数百分比并校验范围；页面从侧边导航可达且不含被删除的四个字段；既有回归全部通过。
