## Why

`connect-miniprogram-menu-catalog@6d77bdd6319722b7c71b4726c6159955da9a84b6` 的新独立重验已因候选内 proposal 仍声明 `DRAFT`、`candidate_sha=none`、`ui_level_actual=NOT_RUN`，而 checkpoint/tasks 声明 `CANDIDATE` 与 UI1 PASS，得到终局语义 FAIL：`artifact-consistency|semantic|proposal-stale-draft-not-run|6d77bdd|verifier`。旧 FAIL 必须原样保留，但当前 local main 已包含相同产品实现；因此需要一个新的 exact candidate 对当前树完整重验，给后续 lifecycle receipt 提供可引用且不改写历史的 superseding evidence。

## What Changes

- 保留旧 candidate 的 artifact-consistency FAIL、首个决定性矛盾、clean-detached 环境与“后续 Gate 未运行”边界；不把 shell 搜索 exit 0 或旧 writer 自述改写为 verifier PASS。
- 从 clean local `main@7d01fe22ded67aeded78cb7d03de87aa12416ada` 形成只含本 change 目录的新 evidence candidate；小程序、Go provider、旧 archive、canonical specs、runner 与 receipt tooling 都保持 byte-unchanged。
- 对新 exact candidate 重新执行 legacy exact-base 三项预期 Red、完整 UI1 13/13、provider Go、JS/JSON/零依赖、strict、protected/owned/sensitive/clean Gate，再由另一 clean detached worktree 独立重跑。
- 只让新 candidate 的独立 PASS supersede 旧 candidate 的“可用于后续 receipt 的验证证据资格”；不 supersede 旧 Git 历史、旧 FAIL、产品契约或实现来源。
- 使后续 `persist-archived-lifecycle-receipt` 成为本 change 的依赖者；本 change 不编辑该 change、receipt 文件或 checker。

## Capabilities

### New Capabilities

- `miniprogram-catalog-evidence-supersession`: 定义 archived catalog candidate 验证失败后，如何保留旧 FAIL 并用当前树的新 exact candidate、完整 W2/UI1 重验和独立 attestation 形成非追溯 superseding evidence。

### Modified Capabilities

无。`miniprogram-menu-catalog` 产品契约与 `loop-engineering-control-plane` 生命周期契约保持只读；canonical 输出只允许由本 change 后续的 deterministic integration/archive 生成。

## Impact

### Outcome 与 Acceptance

- 唯一 outcome：旧 `6d77bdd...` 保持 verifier FAIL，同时一个基于 `7d01fe2...` 当前产品树的新 exact candidate 获得完整 writer Gate 与 independent exact-SHA PASS，可被后续 lifecycle receipt 明确引用为 superseding evidence。
- 单一 acceptance verdict：旧 FAIL 事实、current-tree lineage、完整 W2/UI1 Gate、exact-SHA independent PASS、纯 fast-forward integration 与 deterministic archive 缺一即 `NO-GO`；任何其他 PASS 或评分不得补偿。
- `gate_type=W2`，因为验收必须重新证明匿名 catalog provider、全部小程序 consumer、用户可见状态和错误路径；`ui_level_target=UI1`。实际运行等级与 lifecycle 状态只记录在 `goal-checkpoint.md`。
- UI2、UI3 保持 `BLOCKED_EXTERNAL`；Node UI1 不得冒充微信开发者工具、体验版、真机、真实域名或生产 PASS。

### Owner、Branch、Ownership 与 Base

- owner：`Miniprogram Catalog Evidence Supersession Writer`；从 DRAFT 到 CANDIDATE 只有该 writer。
- branch：`codex/supersede-miniprogram-catalog-evidence`。
- worktree：`/Users/vivix/.codex/worktrees/order-supersede-miniprogram-catalog-evidence.Writer`。
- writer owned paths：严格仅 `openspec/changes/supersede-miniprogram-catalog-evidence/**`。
- repository base：`7d01fe22ded67aeded78cb7d03de87aa12416ada`。candidate 的完整 SHA 必须在提交后由 Git 解析并通过外部 handoff 绑定，不写入其自身 artifact。
- read-only shared contracts：根 `AGENTS.md`、质量门禁、canonical `miniprogram-menu-catalog` 与 `loop-engineering-control-plane` specs、旧 candidate/archive Git 对象、`apps/wechat-miniprogram/**`、相关 Go provider、所有 Skills、receipt change/tooling 与 canonical specs。
- upstream：无未满足 change 依赖；`connect-miniprogram-menu-catalog` 已在 repository base 的直接祖先中集成并归档。
- downstream：`persist-archived-lifecycle-receipt` 必须在本 change 实际 `INTEGRATED` 后才能引用新 superseding evidence；分支、DRAFT、candidate 或 main 外 attestation 均不满足依赖。

### Required Assets

- UI1 所需 Node harness/tests 已存在于 repository base；writer 与 verifier 都必须在各自 exact tree 实际运行，不能复用旧日志。
- UI2 owner 为开发方与客户小程序管理员；缺少锁定的微信开发者工具/体验版、项目权限与真实 HTTPS API 域名，恢复条件是资产齐备后另行执行相同矩阵。
- UI3 owner 为 UAT owner 与客户平台管理员；缺少指定真机、受控账号/目录数据、体验版与可达域名，恢复条件是资产齐备后记录版本、设备/账号边界和最终页面结果。

### Non-Goals

- 不修改任何业务代码、测试、Go provider、公共 API、数据、订单、支付、库存、身份、UI 文案或产品 spec。
- 不修改旧 archive、旧 candidate branch、历史 checkpoint/tasks，且绝不追溯把 `6d77bdd...` 标为 PASS。
- 不修改 `order-run-loop`、stage Skills、`tools/lifecycle-receipts`、frozen receipt change、评分、push/PR/deploy 规则或其他 root tooling。
- 不新增兼容分支、缓存、fallback、runner 规则、通用 verifier 框架或相邻修复。
- 不 push、不创建或更新 PR/MR、不部署、不触发外部系统；integration/archive 需要后续单独授权。
