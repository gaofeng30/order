## Why

2026-08-19 客户评审记录删除了商品目录上的四类字段：§9 不展示商品标签、§10 不展示过敏原、§16 用户端不展示月售、§3 不做数量库存与自动扣减。生效 spec `mvp-product-baseline` 的 `Product availability uses a per-service-date sellout switch` 已明确规定商品可售性只由上下架与按取餐日期的售罄开关决定，且 MUST NOT 实现数量库存、预占或超卖控制。

两端仍在保存与展示这四类字段：种子里每条商品都带 `tags` / `allergens` / `sold` / `stock`；接口契约做库存校验并为新建商品填充标签与过敏原；商户端菜品列表展示「库存 N ·告急」与「月售 N」；菜品编辑提供库存输入；PC 工作台把「库存告急」计入待办。

用户端已经干净——`menu` 与 `detail` 早已切到 API 目录，既有 UI1 用例的 `UNSUPPORTED_PRODUCT_FIELDS` 已经禁止这些字段出现在用户端商品对象上。因此本 change 的范围只剩商户端与两端的 mock 种子。

## What Changes

- 两端种子逐条移除商品的 `tags` / `allergens` / `sold` / `stock`，保留 `status`。
- 两端契约层移除库存校验与数量入参，新建商品不再填充标签与过敏原。
- 小程序商户端：菜品列表移除库存位、告急标记与月售；菜品编辑移除库存输入与数字校验。
- PC 后台：菜品管理表格移除库存列与销量列，编辑表单移除库存输入，批量改价不再透传数量；页面副标题由「上下架、售罄、价格与库存」改为「上下架、售罄与价格」。
- PC 工作台待办移除「库存告急」，并**一并移除「待取超时」**——理由见 `design.md`。
- 两端状态语义色阶移除 `库存告急` 与 `待取超时` 两个已废止条目。

售罄与上下架控件、销量排行数据源 `RANK` 均保留。

## Capabilities

### Modified Capabilities

- `miniprogram-scope-conformance`: 追加「商品记录不含已废止目录字段」「商户端菜品页面不展示这些字段且售罄链路完好」两条。
- `web-admin-scope-conformance`: 追加「PC 后台目录不含已废止字段」「工作台待办只统计待制作数」两条。

## Impact

- Owner：branch `worktree-strip-retired-catalog-fields`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/strip-retired-catalog-fields`。
- Owned paths：`apps/wechat-miniprogram/**`、`apps/web-admin/**`、`openspec/changes/strip-retired-catalog-fields/**`。
- Read-only evidence：`openspec/specs/mvp-product-baseline/spec.md`、`openspec/specs/miniprogram-scope-conformance/spec.md`、`openspec/specs/web-admin-scope-conformance/spec.md`、`docs/product/online-ordering-system-prd-0818-review.md`。
- Dependency：`remove-member-coupon-capability`、`remove-member-coupon-admin-pages`、`remove-retired-entry-screens`，均已集成并归档于 `base_sha`。三者与本 change 共享 `apps/**`，串行执行。
- Non-goals：不实现按取餐日期的售罄开关（属新增类 change，本 change 只删除数量维度）；不改动 PC 工作台的销量排行数据源（PRD §6.12 要求改接真实订单数据，属独立 change）；不删除即时单；不迁移商户端配置页到 PC。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：两端商品记录与契约不含四类字段；商户端与 PC 后台不展示库存与月售；售罄与上下架链路完好；PC 待办只剩待制作数；既有 UI1 回归全部通过；diff 只包含 owned paths。
- 工具边界：仓库未安装 `openspec` CLI，strict 校验记 `BLOCKED_EXTERNAL`。小程序 UI1 来自 Node 测试 harness，PC 后台 UI1 来自本地静态服务器加浏览器实际运行，均非微信开发者工具或真机，不声称 UI2/UI3。
