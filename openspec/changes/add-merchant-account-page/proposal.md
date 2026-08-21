## Why

PRD §4.4 把商户账号名单定为商户身份的唯一来源：小程序端判定 openid 是否已绑商户手机号、PC 后台微信扫码登录比对手机号，两条链路都依赖它。但 §16.5 记为「PC 侧未建」—— 也就是这份决定谁能登录的名单，目前**没有任何维护入口**。

同时 PRD 只写了「手机号 + 角色」，没写清三件在实现时必须回答的事：

- **账号有哪些字段**。评审 §26 只说了主/子两种角色的权限差，没说记录本身长什么样。
- **子账号的可用范围到底是什么**。评审 §26 写「上架下架菜品、扫码」，但 §38 后来把小程序菜品页收敛为只能切可售/售罄，两处口径不一致，名单页必须给管理员一个确定的说法。
- **最后一个主账号被删/停/降级怎么办**。PRD 未提。这是一条会造成**无人能登录 PC 后台**的死锁，且解锁需要人工改库 —— 属于必须在契约层拦住的不变量，不能只靠前端隐藏按钮。

## What Changes

- **PRD §4.4** 补三条规则：账号字段（三个可填 + 三个系统维护）、两种角色的可用范围、最后一个主账号不可失效；并补停用与删除的区别。
- **PRD §15.5.3 / §16.5** 同步该页由「未建」转为已建，§16.5 行改为记录待接后端的两处（扫码登录换手机号、绑定 openid 的写入方）。
- **`apps/web-admin`** 新增商户账号名单页：七列表格（姓名 / 手机号 / 角色 / 可用范围 / 状态 / 微信绑定 / 操作）、搜索、行内启停、编辑抽屉、删除二次确认，以及只剩一个启用主账号时的提示条。
- **契约层** 新增 `listMerchantAccounts` / `saveMerchantAccount` / `setMerchantAccountEnabled` / `deleteMerchantAccount` 与 `ROLES` / `ROLE_LABEL`；最后一个主账号的三条破坏路径由契约层统一守卫，页面不参与裁决。
- **种子数据** 新增 5 条商户账号，覆盖已绑定主账号、未绑定主账号、已绑定子账号、未绑定子账号、停用子账号五种形态。
- **导航** 「名单」组由两页变三页。本页**不提供批量导入**（§6.13.4）。

## Capabilities

### Modified Capabilities

- `web-admin-scope-conformance`：新增三条 requirement —— 名单的维护规则、最后一个主账号不可失效、与折扣白名单分离且不提供批量导入。

## Impact

- Owner：branch `worktree-merchant-accounts`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/merchant-accounts`。
- Owned paths：`apps/web-admin/{index.html,app.js,data/seed.js,data/api.js,pages/accounts.js}`、`docs/product/online-ordering-system-prd-0818.md` 的 §4.4 / §15.5.3 / §16.5 三处、`openspec/changes/add-merchant-account-page/**`、`openspec/specs/web-admin-scope-conformance/spec.md`。
- Non-goals：不接后端（PC 后台仍为 P0 本地演示数据，与 §8.1 一致）；不实现微信扫码登录与 openid 绑定的写入方；不做商户账号批量导入；不改员工折扣白名单页；不改小程序端。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_merchant_accounts.js` 九项全过；base_sha 树上九项全红；小程序 65 项测试不回归；浏览器内验证最后一个主账号的删 / 停 / 降级三条路径均被拒且账号状态不变，加入第二个主账号后恢复可用。
