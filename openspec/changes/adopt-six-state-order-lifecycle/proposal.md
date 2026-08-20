## Why

生效 spec `mvp-product-baseline` 的 `Orders use one six-state production state machine` 与 `Every first-phase order uses one discrete pickup time` 目前被两端实现直接违反。

小程序的状态机是 `待制作 → 待取餐 → 已完成` 三态，且推进后的 Toast 提供**撤销**——生效 spec 明确「生产 MUST NOT 提供撤销或回退已完成转换的入口」。PC 后台同样是三态，并且契约层导出了 `revertOrder`，两个页面各挂了一处回退回调。

两端都还保留即时单：`orderMode` 全局态、订单的 `type` 字段、首页「到店点单」入口、结算页「尽快取餐 / 预约取餐」切换、「尽快 · 约 17:10」文案。评审记录 §7 批注「没有现场点单，只有预约点单」已删除该模式。

订单种子仍带 `待支付`、`已取消`、`待制作` 三个已废止状态，用户端订单筛选段与商户端泳道也按旧状态划分。

## What Changes

- 两端状态机收敛为六态。商户可执行的转换只有 `制作中 → 待取餐` 与 `待取餐 → 已完成`；`已预约 → 制作中` 由服务端定时推进，**客户端不提供该转换**。
- 移除撤销能力：小程序 Toast 组件去掉 `onUndo` 与撤销按钮，PC 契约层去掉 `revertOrder`，两端所有调用点一并清理。
- 移除即时单：`orderMode`、订单 `type` 字段、首页「到店点单」入口、结算页取餐方式切换与「尽快」文案。结算页恒为预约流程。
- 订单种子迁移到六态，并各补一条 `已预约` 订单填充新泳道。
- 用户端订单筛选段改为 全部 / 已预约 / 制作中 / 待取餐 / 已完成 / 已退款；PC 泳道同口径加 `全部`。
- 核销二维码只在 `待取餐` 展示；`已预约` 与 `制作中` 只展示取餐号、取餐时间与状态。
- 结算生成订单时按 §7.4 判定：距取餐不足 30 分钟直接进 `制作中`，否则 `已预约`；取餐号改为 4 位数字无前缀。
- 修正一处被本 change 放大的既有文案缺陷：不可推进订单的行内说明拼出「该订单已已完成」。

## Capabilities

### Modified Capabilities

- `mvp-product-baseline`: 六态 requirement 补充「客户端 MUST NOT 提供 `已预约 → 制作中`」与「禁止撤销包括 Toast 回退动作与回退契约方法」，并新增对应 scenario。
- `miniprogram-scope-conformance`: 追加「只走六态且无撤销」「只提供预约下单且二维码按 `待取餐` 门控」两条。
- `web-admin-scope-conformance`: 追加「只走六态、无回退方法、泳道六态口径」一条。

## Impact

- Owner：branch `worktree-six-state-lifecycle`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/six-state-lifecycle`。
- Owned paths：`apps/wechat-miniprogram/**`、`apps/web-admin/**`、`openspec/changes/adopt-six-state-order-lifecycle/**`。
- Dependency：删除类三件套，均已集成并归档于 `base_sha`。
- Non-goals：不实现服务端定时排产（无后端，`已预约 → 制作中` 在原型中只体现为种子状态与客户端不可推进）；不实现退款流程与 `退款中 → 已退款` 的触发（PC 退款入口属独立 change）；不实现取餐时间点选择的新 UI（结算页沿用现有日期与时段控件，改造属 `switch-to-pickup-time-selection`）；不实现支付对账兜底；不改后端。
- Gate：`gate_type=W2`；`ui_level_target=UI1`；`ui_level_actual=UI1`。
- 最小成功标准：两端状态语义、可推进转换、泳道与筛选段均为六态口径；两端无任何撤销/回退入口或方法；即时单在两端零残留；结算生成带取餐时间的预约单；核销二维码按 `待取餐` 门控；既有 UI1 回归全部通过；diff 只包含 owned paths。
- 工具边界：仓库未安装 `openspec` CLI，strict 校验记 `BLOCKED_EXTERNAL`。小程序 UI1 来自 Node harness，PC UI1 来自本地静态服务器加浏览器实际运行，均不构成微信开发者工具或真机证据。
