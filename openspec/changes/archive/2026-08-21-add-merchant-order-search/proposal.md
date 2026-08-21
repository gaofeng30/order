## Why

小程序商户端订单页顶部有一个搜索框，但它是**死的** —— `pages/admin-orders/admin-orders.wxml:21` 是一个静态 `<text class="search-ph">搜索 取餐号 / 订单号 / 手机号</text>`，不是 `<input>`，没有绑定任何事件，`admin-orders.js` 里也没有对应 handler。§6.6 末条要求「商户端订单列表提供按取餐号、订单号、手机号搜索」，现在点上去什么都不会发生。这是一个**看起来已交付实则未交付**的能力，比缺失更危险。

同时取餐号还是旧格式。§7.8 明写「取餐号为 4 位数字，按取餐日期从 `0001` 顺序累计。即时单已删除，不再有前缀区分」，但两端种子数据仍是 `A126` / `B208` / `A118` 这种字母加三位的即时单遗留格式。PC 后台已经全量切到 4 位数字，两端现在对不上号。

这两件事必须一起做。取餐号一旦变成 4 位纯数字，§7.8 最后一条描述的风险就成立了 ——「跨营业日的取餐号可能重复，因此手工核销必须限定当前营业日期」。而当前商户端订单**根本没有日期字段**，这条约束无法表达，也无法验证。所以「按取餐号搜索」和「取餐号 4 位数字化」共享同一个正确性前提：营业日。分开做，中间那个版本会引入 §7.8 明确警告的歧义。

## What Changes

- **搜索落地**：`<text>` 占位换成受控 `<input>`，按取餐号 / 订单号 / 手机号 / 联系人匹配，跨泳道。点泳道即退出搜索态。
- **取餐号 4 位数字化**：商户端与用户端两份种子数据的 `code` 全部改为 4 位数字，去掉 `A` / `B` 前缀；`admin-verify` 的模拟扫码号与输入框 placeholder 一并更新。
- **补营业日字段**：商户端订单新增 `pickupDate`，`utils/data.js` 导出单一 `BUSINESS_DAY` 常量。种子补一条**跨营业日且取餐号与当日重复**的 `待取餐` 单，使「限定当前营业日」这条约束可证伪。
- **取餐号解析限定当前营业日**：搜索与手工核销两条路径共用同一规则。跨营业日命中时报出该事实并指出定位办法，而不是让使用者看到空列表或「无效取餐号」。
- **补一条 `退款中` 种子单**：§7.1 允许订单长期停在 `退款中`，种子里此前没有该状态的样本，它在「全部」泳道与搜索结果中的呈现无法验证。

## Capabilities

### Modified Capabilities

- `miniprogram-scope-conformance`：新增三条 requirement —— 商户订单列表可按取餐号 / 订单号 / 手机号跨泳道搜索；取餐号在两端均为 4 位数字并按取餐日期累计；取餐号解析限定当前营业日且跨日命中给出可执行提示。

## Impact

- Owner：branch `worktree-merchant-search`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/merchant-search`。
- Owned paths：`apps/wechat-miniprogram/{utils/data.js,utils/util.js,pages/admin-orders/**,pages/admin-verify/**}`、`apps/wechat-miniprogram/tests/merchant-search-ui1.test.js`、`openspec/changes/add-merchant-order-search/**`、`openspec/specs/miniprogram-scope-conformance/spec.md`。
- Non-goals：
  - **不加 `退款中` 泳道**。§6.6 第一条明确列出五档泳道且不含 `退款中`，`tests/order-lifecycle-ui1.test.js:142` 亦已钉死该数组。经与用户确认按 PRD 执行，本 change 只保证该状态的单在「全部」泳道与搜索结果中可被找到。
  - **不接 `wx.scanCode`**。§6.6 要求扫码在手机上进行，但真机扫码只能在联调阶段验证，模拟扫码按钮保留原样。
  - **不改核销的写入路径**。`admin-verify.js` 的 `confirm()` 目前直接改内存状态、未走 `utils/api.js`、无幂等键，这是接后端时的工作，不在本 change 范围。
  - 不改 PC 后台；不改用户端订单页交互（仅同步 `code` 字面值）。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_merchant_search.js` 十二项全过；base_sha 树上十一项红，第十二项（JS 解析）为非回归守卫；小程序既有 65 项测试不回归；`lint_wx.py` 通过；历史归档门禁失败集合与 base 逐行一致。
