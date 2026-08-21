## Why

§7.3 定了一条对账兜底链路，用来堵「用户已付款、系统无订单」的资损口子：定时任务扫描预支付记录 → 调微信查询接口 → 幂等补建订单 → **补建失败的转入 PC 后台「支付待处理」列表，由主账号人工处理**。

那个列表一直没建。§3.5 的 12 页清单里列着它，§15.5.3 也记为「新增」，但对应的页面不存在 —— 也就是说这条兜底链路的**人工出口是断的**：自动补建失败之后，钱在微信账户里、订单不存在、没有任何界面能看到这笔，更谈不上处理。

上一个 change 做完发起退款后，这是 §15.5.3 清单上最后一项未建页面。

## What Changes

- **新增 `pages/pending.js`**：八列表格（预支付单号 / 支付时间 / 金额 / 顾客 / 意向取餐 / 菜品 / 未建单原因 / 操作），两个动作对应 §7.3 的「人工建单」与「退款作废」，各带二次确认层。
- **契约层新增** `listPendingPayments` / `pendingPaymentCount` / `rebuildOrder` / `refundPendingPayment` / `blockingReason`。
- **建单前重新校验阻塞原因**：商品仍下架或售罄、取餐时间已过、数据校验未通过时拒绝，并指出解除办法。页面在打开确认层时就展示判定结果，原因仍在时不提供确认按钮。
- **建单沿用原支付事实**（交易号、支付时间、实付金额），按 §7.8 分配当日唯一的 4 位取餐号，按 §7.4 判定进 `已预约` 或 `制作中`。
- **退款作废**沿用 §7.7 的全额规则：无金额入参、原因必填、只到 `退款中`、记录操作人；不生成订单；进入财务页的退款台账并标出来源。
- **财务与对账页**新增「已收款未建单」旁注：净额卡下标注笔数与金额，底部说明该差额的成因与消解方式。字段命名为 `unbuiltCount / unbuiltAmount`，与既有的「未到账退款笔数」`pendingCount` 区分。
- **种子**新增 3 条待处理记录，覆盖 §7.3 列举的三种原因；其中「商品已下架」那条指向本就处于下架态的 `p007`，使「解除原因后重试」可被实际走通。
- **PRD §16.5** 记录扫描链路（定时任务 + 微信查询接口）仍属后端缺口。

## Capabilities

### Modified Capabilities

- `web-admin-scope-conformance`：新增四条 requirement —— 待处理条目不是订单、建单在原因未解除时必须拒绝、作废全额退款并入台账、对账报出已收款未建单的差额。

## Impact

- Owner：branch `worktree-pending`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/pending`。
- Owned paths：`apps/web-admin/{index.html,app.js,data/seed.js,data/api.js,pages/pending.js,pages/finance.js}`、`docs/product/online-ordering-system-prd-0818.md` 的 §16.5、`openspec/changes/add-pending-payments/**`、`openspec/specs/web-admin-scope-conformance/spec.md`。
- Non-goals：不实现 §7.3 前半段（定时扫描任务与微信支付查询接口，属后端）；不实现 `退款中 → 已退款` 的推进（由支付回调驱动）；不改订单管理页；不接后端。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：`check_pending.js` 十二项全过；base_sha 树上十一项红；小程序 65 项不回归；历史归档门禁失败集合与 base 完全一致；浏览器内验证三种原因下建单均被拒且无确认按钮、上架后重试建单得到当日唯一取餐号 0189、作废后不生成订单但进入退款台账、财务页净额卡标注「另有 2 笔已收款未建单 ¥66.00」且处理完毕后差额归零。
