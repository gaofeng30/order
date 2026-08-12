## Why

客户已认可的 P0 原型允许使用本地 mock 和简化状态，但正式研发仍缺少一个覆盖一期范围、订单履约、身份价格、预约库存、权限责任与外部依赖的唯一产品行为基线。若直接按现有 PRD、§15 原型或前端内存状态进入后续业务 change，同一规则会被不同 writer 解释成不同公共契约并造成返工。

## What Changes

- 在 PRD §1–§14 中建立一期正式研发的唯一产品行为基线，并明确真实适用合同/客户正式确认、PRD 目标章节、§15/原型与 mock 的优先级。
- 固定一期包含与排除范围，不把员工价混同为会员等级，也不把 P0 mock 当作生产契约。
- 固定库存、下单软预占、支付确认、超时和迟到支付、九态订单状态机、取消退款、员工/访客识别、逐商品员工价、预约取餐与后台四角色规则。
- 固定外部依赖的 12 项顺序链、责任边界、进入后续阶段的 Gate，以及不得在任何台账中保存密钥、证书、账号标识或个人数据的安全边界。
- 为需求、页面、状态、角色和外部依赖建立追踪矩阵，并把后续 PRD 更新拆成可执行、可检查、可回滚的实现任务。
- 本规划保持 `DRAFT`；批准前不修改 PRD，不执行任何实现任务。

## Capabilities

### New Capabilities

- `mvp-product-baseline`: 定义一期正式研发必须共同遵守、可追踪并可验收的唯一产品行为与外部依赖基线。

### Modified Capabilities

无。

## Impact

- 状态：`DRAFT`；本提交只完成规划，不代表规则已经写入 PRD 或获准实施。
- owner：`codex/formalize-mvp-product-baseline` 分支及当前既有独立 worktree 的唯一 writer。
- owned paths：`openspec/changes/formalize-mvp-product-baseline/**`、`docs/product/online-ordering-system-prd.md`；本轮只写 change 目录，批准后的实现才允许修改 PRD。
- shared contracts：后续将修改 PRD §1–§14 的一期范围、业务规则、验收标准与文档优先级；它会成为身份、商品、库存、订单、支付、退款、履约、核销、后台权限和上线准备等后续 changes 的产品输入，但本 change 不定义这些能力的 API 或数据库实现。
- 基线：`c47135b660a9ca3f9f9ee6ded6b09fbf0ee6f1af`；现有 `bootstrap-api-service` 只提供进程基线，不构成本 change 的行为依赖。
- 外部依赖：合同适用性、客户正式确认、主体资质、平台账号、域名备案、支付安全状态、真实经营配置与外部授权只限制对应联调、UAT、提审或上线 Gate，不阻塞本地 OpenSpec 规划和 PRD 基线编写。
- 非目标：不修改客户确认清单、合同、技术文档或任何业务代码；不实现 API、数据库、支付、库存、定时任务、页面或部署；不登记真实密钥、证书、账号标识、员工名单或其他个人数据；不替代客户的签署、认证、收款、退款和发布责任。
- 最小成功标准：四类 OpenSpec 产物完整且内部无冲突、无行为 TODO；`openspec validate formalize-mvp-product-baseline --strict` 通过；需求—页面—状态—角色—外部依赖追踪矩阵完整；PRD 实施任务具有可执行检查；diff 仅包含 owned paths。
