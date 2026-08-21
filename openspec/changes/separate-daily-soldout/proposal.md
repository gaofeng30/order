## Why

§15.6.1 写得很清楚：`status: 'on'|'off'` —— 上下架，**`soldout` 移出本字段**；「当日售罄不落在 Product 上，由服务端按 `(productId, 营业日期)` 维护独立记录，次日自然清零」。

两端都没有照做。`apps/web-admin/data/seed.js` 与 `apps/wechat-miniprogram/utils/data.js` 都把 `soldout` 塞进 `status`，于是它成了商品的一个**长期属性**。后果不是「字段放错了地方」，而是两条业务规则直接失效：

1. **售罄不会次日清零。** §6.5 要求「次日自然清零，无需手工恢复」。存成商品属性后，商户今晚标的售罄明早还在，得记着挨个恢复 —— 忘一个就是一整天卖不出去，而且没有任何东西会提醒他。
2. **售罄会误伤明天的预约。** §6.5 明写「商户在营业日 D 标记售罄，只屏蔽 D 当天的下单，**不影响 D+1 的预约**」。当前实现里售罄是全局的，用户预约明天取餐也会被挡。一期只有预约单，这条正是它的主场景。

三个维度被压进了一个枚举：上下架是长期的、由 PC 独占；售罄是当日的、两端可切。压在一起，任何一方的写入都会覆盖另一方的意图。

## What Changes

- **`status` 收敛为 `'on' | 'off'`**，两端种子与全部读写点同步。
- **新增按取餐日期的售罄记录** `PRODUCT_SOLD_OUT_DATES`，形如 `{ productId, serviceDate }`，唯一键为二者之组合 —— 与后端 `serve-reservation-menu-availability` 的 `product_sold_out_dates` 表同形。
- **新增派生** `isSoldOut(productId, serviceDate)`。可售性 = 上架 **且** 该取餐日期无售罄记录，两个维度分别判断。
- **两端菜品页的售罄开关**改为写入/删除当前营业日的记录，不再触碰 `status`；上下架仍只在 PC。
- **种子补一条昨日售罄记录**，使「次日自然清零」可证伪 —— 没有它，清零与从未售罄不可区分。

## Capabilities

### Modified Capabilities

- `miniprogram-scope-conformance`：新增两条 requirement —— 商品的上下架与当日售罄是两个独立维度；小程序商户端的售罄开关按营业日写入且不影响次日。
- `web-admin-scope-conformance`：新增两条 requirement —— PC 同一模型；PC 的补建订单校验按取餐日期判断售罄。

## 与已归档门禁的冲突

`openspec/changes/archive/2026-08-20-strip-retired-catalog-fields/checks/check_catalog_fields.js` 断言：

```js
return w.Api.setProductStatus(id, 'soldout')
  .then(() => { assert.equal(w.__store.menu[0].status, 'soldout'); ... });
```

它把「售罄是 `status` 的第三个取值」当成契约钉住了 —— 而这正是 §15.6.1 禁止、本 change 要纠正的事实。这是正当的事实变更而非误报：`strip-retired-catalog-fields` 当时只负责删除库存 / 月售 / 标签 / 过敏原，没有触及售罄的存储位置。

按项目既有做法，本 change 的门禁**接管**该断言集：四条仍然成立的断言（种子无废止字段、契约不含数量、菜品页无库存与销量列、看板无库存告急）原样吸收，并把冲突的那一条改写为「上下架只接受 `on` / `off`，第三个取值必须被拒绝」。

## 依赖

- **契约对齐（非阻塞）**：后端 change `serve-reservation-menu-availability`（owner: `codex/serve-reservation-menu-availability`，owned paths 全在 `services/api/**`）已定义 `product_sold_out_dates` 表与 `sold_out` 响应字段，并要求「日期 D 的售罄商品仍返回，但带 `sold_out=true`，且同一记录不影响 D+1」。本 change 的客户端模型按该形状命名与建模，**不修改其任何 owned path**，两者路径零重叠，可并行。
- 用户端菜单的可售性由该后端的 `/api/v1/menu` 提供，接入属另一件事；本 change 只让客户端模型先就位。

## Impact

- Owner：branch `worktree-soldout`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/soldout`。
- Owned paths：`apps/web-admin/{data/seed.js,data/api.js,pages/products.js}`、`apps/wechat-miniprogram/{utils/data.js,utils/api.js,pages/admin-products/**}`、`apps/wechat-miniprogram/tests/daily-soldout-ui1.test.js`、`openspec/changes/separate-daily-soldout/**`、两份生效 spec。
- Non-goals：
  - **不接 `/api/v1/menu`**。用户端菜单的按日可售性依赖后端 change，接入是另一件事。
  - **不做数量库存或超卖保护**。§6.5 明确一期没有，商户只能现场手工标售罄。
  - 不改商品的其余字段、不改批量导入模板、不动订单。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_daily_soldout.js` 十六项全过（含接管来的四项）；base_sha 树上十一项红；两端既有测试不回归；`lint_wx.py` 通过；归档门禁失败集合与 base 逐行一致。
