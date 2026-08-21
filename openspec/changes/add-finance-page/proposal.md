## Why

PRD §6.11 要求 PC 后台有财务与对账页，§3.5 的 12 页清单里也列着它，但一直未建 —— 商户没有任何地方能看到「今天收了多少、退了多少、和微信账单对不对得上」。

前一个 change（`align-order-model`）已把订单模型对齐到 §15.6.2，支付时间、微信交易号、折扣快照与退款记录都已就位，本 change 建页。

## What Changes

- **契约层**新增四个只读方法：`listPayments` / `listRefunds` / `financeSummary` / `buildPaymentExport`。归集口径写在契约层而不是页面 —— 后端就位时只替换实现，页面不动。
- **新增 `pages/finance.js`**：日期区间筛选（含今天 / 近 7 天快捷）、四张汇总卡（实收 / 退款 / 净额 / 员工折扣）、收款明细与退款记录两个页签、明细导出、以及一段说明本页能核什么、不能核什么的口径文案。
- **导航**「经营」组由三页变四页，位置在订单管理与扫码核销之间。
- **种子数据**补两笔用于把归集口径钉死的订单：一笔前一日支付、次日取餐的预约单；一笔支付次日才退到账的退款。
- **PRD §16.5** 记录「与微信支付账单自动核对」为后端缺口。

## 两条归集口径

这页的全部价值是和微信商户平台的交易账单对得上，差一分就是废的。两条口径都容易写反，因此都由门禁用跨日数据钉死：

- **收款按支付日期归集，不是营业日期。** 微信账单以交易时间为准。预约单可以今天付、明天取，按营业日期归集会让这笔跑到错误的一天。
- **退款按退款到账日期归集，不是原订单的支付日期。** 跨日退款在微信账单里出现在到账那天，跟着原单走会让两天都错。

净额一律按金额相减，不按订单笔数 —— 部分退款是合法情形。

## Capabilities

### Modified Capabilities

- `web-admin-scope-conformance`：新增五条 requirement —— 按支付日期与退款日期分别归集、汇总按金额净额、退款记录可回溯、导出能被 Excel 正确打开、页面声明自身对账边界。

## Impact

- Owner：branch `worktree-finance`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/finance`。
- Owned paths：`apps/web-admin/{index.html,app.js,data/seed.js,data/api.js,pages/finance.js}`、`docs/product/online-ordering-system-prd-0818.md` 的 §16.5、`openspec/changes/add-finance-page/**`、`openspec/specs/web-admin-scope-conformance/spec.md`。
- Non-goals：不实现自动拉取微信账单（需后端调支付查询接口）；不实现发起退款（§15.5.3 列为订单管理的改造项）；不建「支付待处理」页（§7.3，另做）；不接后端。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_finance.js` 十项全过；base_sha 树上八项红；小程序 65 项不回归；浏览器内验证 08-21 实收 510.80 − 退款 36.00 = 净额 474.80、08-22 只有跨日退款且净额为 −¥12.00、08-20 含那笔次日取餐的预约单、非法区间被拒、导出带 BOM 且行数正确。
