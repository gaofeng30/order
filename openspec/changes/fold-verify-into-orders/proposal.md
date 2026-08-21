## Why

侧边栏有 13 页，PRD §3.5 写的是 12 页。多出来的是 PC 端的**扫码核销页** —— §15.5.3 早已判它「删除」，理由是评审 §23「扫码是在手机上扫」，手工核销并入订单管理。删除一直没做。

但直接删会造成功能倒退：`pages/verify.js` 是当前**唯一**能按取餐号找到一张单的地方，而订单管理页至今没有任何搜索框。§6.6 末条要求「商户端订单列表提供按取餐号、订单号、手机号搜索」，§6.7 要求主账号可按取餐号、订单号、手机号查询全部订单 —— 两条都没落地。

所以「删除」和「并入」必须一起做，否则删掉的是能力而不是冗余。

## What Changes

- **删除 PC 扫码核销页**：`pages/verify.js`、导航路由、`index.html` 挂载、以及 `app.css` 第 13 节该页专属的 13 条 `.vf-*` 样式（整节删除，不做逐行正则）。
- **订单管理补搜索**：按取餐号 / 订单号 / 手机号 / 联系人，跨泳道。点泳道即退出搜索态。
- **取餐号限当前营业日**：§7.8 明写跨营业日的取餐号可能重复，§6.6 因此规定手工输入只匹配当前营业日。订单号与手机号不受此限。
- **跨日取餐号给提示**：某取餐号当日无果却存在于其他营业日时，报出该事实并指出定位办法，而不是让使用者看到一个空列表。
- **契约层新增** `searchOrders(q)` 与 `codeHint(q)`。

## Capabilities

### Modified Capabilities

- `web-admin-scope-conformance`：新增四条 requirement —— PC 无独立核销页且页面集合与 §3.5 一致、手工核销经由订单搜索定位、取餐号只在当前营业日解析、核销拒绝已退款单且幂等。

## Impact

- Owner：branch `worktree-drop-verify`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/drop-verify`。
- Owned paths：`apps/web-admin/{index.html,app.js,app.css,data/api.js,pages/orders.js}`、删除 `apps/web-admin/pages/verify.js`、`openspec/changes/fold-verify-into-orders/**`、`openspec/specs/web-admin-scope-conformance/spec.md`。
- Non-goals：不改小程序商户端的扫码核销（那是评审 §23 指定的扫码位置，保留）；不实现按取餐日期区间查询（§6.7 提到，但「未取餐」筛选已覆盖主要场景）；不接后端。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_fold_verify.js` 十一项全过；base_sha 树上七项红（四项为非回归守卫）；小程序 65 项不回归；历史归档门禁失败集合与 base 逐行一致；`app.css` 花括号平衡且无残留 `.vf-` 选择器；浏览器内验证侧边栏 12 页与 §3.5 逐项一致、当日取餐号可搜到、跨日取餐号返回空并给出提示、同一单可用订单号找到、手机尾号可搜到、搜索结果跨三个状态。
