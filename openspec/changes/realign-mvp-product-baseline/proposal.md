## Why

`openspec/specs/mvp-product-baseline/spec.md` 是一期产品行为的生效 spec，也是后续所有业务 change 的验收事实源。它成文于 2026-08-12，依据的是当时的 `docs/product/online-ordering-system-prd.md`。

2026-08-19 客户评审（`docs/product/online-ordering-system-prd-0818-review.md`，按该 spec 自身的 authority order 属第 1 顺位证据）推翻了其中大部分规则。该评审结论已于 `2f2db4a` 落入 `docs/product/online-ordering-system-prd-0818.md`，但 spec 未同步。

当前 spec 的 12 条 requirement 中有 9 条与客户已确认结论直接冲突：一期范围、数量库存与餐段库存池、15 分钟软预占与迟到支付、九态状态机、取消退款规则、员工身份识别、逐商品 employee_price、固定取餐时段、四角色。另有 1 条（authority order）需要具体化到当前生效证据。

后果是双向的：依据 spec 实现会做出客户已明确删除的能力；依据新 PRD 实现则无法通过按 spec 执行的验收。前端与后端适配在 spec 同步前无法安全开工。

## What Changes

- 把 authority order 中的「客户正式确认记录」具体化到 `online-ordering-system-prd-0818-review.md` 与 `online-ordering-system-prd-0818.md`，使 spec 的裁决链指向当前生效证据。
- 收敛一期范围：删除数量库存、即时单、接单、会员等级与优惠券、跨业务主入口；保留仅预约取餐、单取餐点、员工折扣。
- 删除数量库存、软预占与迟到支付三条 requirement；改由「按取餐日期的售罄开关」承担商品可售判定。
- 九态状态机收敛为六态；订单只在微信确认支付成功后创建，调起支付前的预支付记录不是订单。
- 新增支付对账兜底 requirement，覆盖「已支付但未生成订单」的补建与人工处理路径。
- 逐商品 `employee_price` 替换为全局单一折扣率、逐商品先舍入到分再乘数量。
- 固定取餐时段替换为离散取餐时间点，取消退款规则按新状态机重写。
- 四角色收敛为主账号 / 子账号两角色，并固定商户名单与折扣白名单分离、PC 后台微信扫码登录。
- 员工身份识别改为「浏览免手机号、首次提交订单前微信授权手机号」，并新增手工填写的附加手机号必须「手机号 + 姓名」双要素命中。

本 change 只改写 spec 文本，不改代码、API、schema、数据或任何运行行为。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `mvp-product-baseline`: 一期唯一产品行为基线按 2026-08-19 客户评审记录重新对齐，使 spec 与生效 PRD 一致，解除后续前端与后端 change 的规格阻塞。

## Impact

- Owner：branch `worktree-realign-mvp-product-baseline`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/realign-mvp-product-baseline`。
- Owned paths：`openspec/changes/realign-mvp-product-baseline/**`。
- Read-only evidence：`docs/product/online-ordering-system-prd-0818-review.md`、`docs/product/online-ordering-system-prd-0818.md`、`openspec/specs/mvp-product-baseline/spec.md`、`AGENTS.md`、`docs/quality/change-quality-gates.md`。
- Dependency：无。`base_sha` `2f2db4a31f66f992997880a02b438c9690bbb845` 已包含本 change 依据的两份产品文档。
- Blocks：后续所有前端与后端适配 change。在本 change 集成 `main` 前，任何按新 PRD 修改 `apps/**` 或 `services/**` 的 change 都无法用生效 spec 验收。
- Non-goals：不修改 `openspec/specs/mvp-product-baseline/spec.md` 本体（delta 在 archive 时应用）；不修改产品文档、合同、客户清单、其他 spec、AGENTS/Skills、代码、API、schema、部署配置或外部系统；不实现任何被本 change 重新定义的行为；不处理 `feat/member-coupon` 分支的废弃（该动作在本 change 之外单独执行）。
- Gate：`gate_type=W0`；`ui_level_target=UI0`；`ui_level_actual=UI0`；external assets none。
- 最小成功标准：delta spec 覆盖全部 9 条冲突 requirement 与 1 条 authority order 具体化，且无遗留；delta 中不出现已废止概念（数量库存、软预占、迟到支付、九态、待支付、已支付待接单、已取消、异常、employee_price、四角色、固定取餐时段、优惠券、会员等级、即时单）的肯定性表述；delta 每条 requirement 与 0818 PRD 的对应章节可逐条追溯；四类产物完整且无行为 TODO；结构与内容检查、`git diff --check` 与 owned-path 审计通过。
- 工具边界：仓库当前未安装 `openspec` CLI（`command not found`），`openspec validate --strict` 与 `openspec new change` 无法执行。产物按仓库既有 change 的结构手工建立，并以可执行的结构/内容检查替代，记为已知工具缺口而非 PASS。
