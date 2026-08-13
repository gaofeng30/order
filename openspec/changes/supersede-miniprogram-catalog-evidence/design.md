## Context

历史产品 candidate `6d77bdd6319722b7c71b4726c6159955da9a84b6` 的 parent 是 legacy behavior base `94e04bf26e37e93299c26ef2c9c8aa7552619444`，其直接 archive child 是 repository base `7d01fe22ded67aeded78cb7d03de87aa12416ada`。`6d77bdd...` 与 `7d01fe2...` 的 `apps/wechat-miniprogram` tree 都是 `80d16424aefa0d4b9d4e451a1ebe5e8013627a8b`，provider `internal/catalog` tree 都是 `1867e1cb94fd38b718641d28022e1cf2e386c85b`，`internal/httpapi` tree 都是 `38f9f486156547cd547d2f3840566acfbbd4c0eb`；两者之间只有 OpenSpec archive move 和 canonical delta。

但对 `6d77bdd...` 的新 clean-detached verifier 在首个 artifact consistency Gate 即终止：proposal 第 27/30/31 行仍称 DRAFT、无 candidate、UI 未运行，checkpoint 第 11 行与 tasks 第 168-177 行却称 CANDIDATE、UI1 和 4/4 artifacts PASS。搜索命令 exit 0 只证明冲突字段存在，语义 verdict 是 FAIL，fingerprint 为 `artifact-consistency|semantic|proposal-stale-draft-not-run|6d77bdd|verifier`；后续 base Red、UI1、provider、static、scope 等 Gate 均未运行。

本 change 不修旧 artifact，也不改产品实现。它以 current repository base 新建 evidence-only candidate，在该 exact tree 重新执行完整产品 Gate，并让后续 receipt 显式引用新 candidate 的 superseding attestation。

## Goals / Non-Goals

**Goals:**

- 永久保留旧 candidate 的 verifier FAIL、首错、环境和未运行边界。
- 用一个只新增本 change 目录的新 exact candidate 证明当前 main 中同一产品实现通过完整 W2/UI1 Gate。
- 明确 `supersedes` 只作用于后续 receipt 可接受的验证证据，不作用于旧 verdict、Git 历史、产品契约或实现来源。
- 使 receipt change 只能在本 change 集成后消费新证据。

**Non-Goals:**

- 不修改业务代码、测试、provider、旧 archive、canonical product/control specs、Skills、receipt/tooling、评分或发布规则。
- 不为 `6d77bdd...` 补 PASS，不复用旧 writer 日志，不把当前树 PASS 归属到旧 SHA。
- 不证明 UI2/UI3、真实订单、支付、库存、quote、availability 或生产行为。

## Decisions

### D1. 使用三层 immutable lineage，避免把“同一实现”误写成“同一验证”

证据固定区分：

1. `legacy_behavior_base=94e04bf...`：只用于隔离重放三项预期 Red；
2. `historical_failed_candidate=6d77bdd...`：只保留 artifact-consistency FAIL，永不追溯改判；
3. `repository_base=7d01fe2...` 与其新 evidence candidate：新 candidate 只在本 change 目录增加 evidence artifacts，对其 exact SHA 重跑全部 Green Gate。

新 PASS 的语义是“当前 repository base 中相同 app/provider tree 在新 exact candidate 上通过”，不是“旧 candidate 后来通过”。相比修旧 archive或用旧 branch 再验，此方案保留审计时间线，且让 candidate SHA、tree identity 和 attestation 一一对应。

### D2. proposal/spec/design 只存稳定契约，checkpoint 是 lifecycle 运行态唯一来源

proposal、design 与 delta spec 不记录易过期的当前 state、actual UI、last decision 或 exact self-candidate SHA。`goal-checkpoint.md` 独占这些运行字段；tasks 只记录每个动作及其证据，不另行决定 lifecycle state。

candidate SHA 具有自引用问题，因此 candidate 内只存 `DERIVE_FROM_GIT_EXTERNAL_HANDOFF` 策略。单一 commit 完成后，writer 运行 `git rev-parse HEAD`、验证 parent/base、diff 与 clean status，再把完整 SHA 主动 handoff；不回写 candidate。verifier 仅消费该 immutable SHA。

### D3. evidence change 的 Red/Green 使用同一结构与内容完整性检查

批准后先在未形成 superseding result 的 tree 运行一个标准库结构检查：它必须确认旧 archive 同时存在 DRAFT/none/NOT_RUN 与 CANDIDATE/UI1/4-of-4 字段、旧 archive 相对 `7d01fe2...` byte-unchanged、对应 app/provider tree 与 `6d77bdd...` 相同，然后因 checkpoint 尚无完整 writer-Gate result 而返回预期 Red。该 Red 证明“旧历史自相矛盾且没有新的可接受 supersession”，不能以缺文件、缺模块或改坏旧 archive制造。

完成全部 current-tree Gate 后，同一检查只因本 change checkpoint/tasks 的新 evidence relation 齐备而 Green；旧 archive 仍保持相同 bytes 与 FAIL。Refactor 只去除本 change 内重复叙述，随后重跑同一结构检查和全部产品 Gate，不改断言、测试或产品文件。

### D4. current-tree acceptance 完整重跑，不从历史日志借 PASS

writer 与 independent verifier 都执行同一矩阵：

- 将 `94e04bf...` 的 `apps/wechat-miniprogram/**` 解到经 parent/name/type/non-link/entries 校验的窄 system-temp 目录，只覆盖新 exact tree 的 `tests/page-harness.js` 与 `tests/catalog-ui1.test.js`，运行 focused `legacy behavior boundary`；必须得到 3 tests/0 pass/3 fail，decisive values 为 list request `0`、queued network request `0`、unknown detail fallback `p001`，且无 missing-module error。预期非零是 Red，不是 candidate FAIL；cleanup 任一校验/删除失败则 Gate FAIL。
- 在新 exact tree 运行 `npm test --prefix apps/wechat-miniprogram`，必须 13/13 PASS；覆盖匿名 URL/HTTP、状态/retry/empty、stable order、huge string ID、integer cents、selection/snapshot/inverse、WXML，以及既有 promo/order/pay 入口 non-regression。
- 运行 `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/catalog ./services/api/internal/httpapi -count=1`。
- 对全部受影响的 51 个 `apps/wechat-miniprogram/**/*.js` 执行 `node --check`，解析该树 43 个 JSON，并单独解析 `project.config.json`，证明局部 package/lock 无第三方 dependency；Web Admin 是 byte-unchanged protected non-consumer，不混入 51/43 计数。随后运行 strict、whitespace、forbidden-field、legacy-read、fake-ID 与 sensitive audits。
- 订单、promo 和 pay 只验证既有入口未因 catalog item shape 回归；任何 mock order/pay 结果不得记为真实订单或支付 PASS。

UI2/UI3 保持 `BLOCKED_EXTERNAL`；它们不阻塞本地 UI1 supersession，但不得被降级为 PASS。

### D5. protected 与 owned-path Gate 同时绑定 base、candidate 和历史 archive

新 candidate 从 `7d01fe2...` 的 diff 必须严格只有 `openspec/changes/supersede-miniprogram-catalog-evidence/**`。下列路径相对 base 必须 diff zero：`apps/**`、`services/**`、旧 catalog archive、all canonical specs、`.agents/skills/**`、`tools/lifecycle-receipts/**`、frozen receipt change、root/stage skills 和仓库治理文件。

Git audit 还必须重算 current app/provider tree IDs、证明 base 是 candidate 的直接 parent，并确认旧 archive 的 blob 未变。任一 product/spec/task/command/base/tree/SHA 或 scope 变化都使 writer 与 independent evidence失效，从头产生新 candidate 并重验。

### D6. independent PASS、纯 fast-forward 与 deterministic archive 串行完成

只有 committed full SHA、clean writer status 和完整 handoff 存在时才建立新的 clean detached verifier worktree。verifier 首个检查再次确认旧 FAIL 与新 lineage，再从 exact-base Red 开始完整重跑；任何 FAIL 返回本 writer，新 SHA 必须重建 detached worktree并从头验证。

获得单独 integration 授权后，只允许 candidate 纯 fast-forward 到仍以其 base 为祖先的 local main；main 推进或不能 FF 时返回 writer 基于新 main 重建 candidate，不能 merge/rebase 后沿用旧 attestation。integration 验证通过后才执行 deterministic OpenSpec archive；candidate 本身不直接修改 canonical spec，canonical delta 与 dated archive 是 archive 的唯一生成输出。随后 receipt change 才能作为依赖者引用新 candidate/integrated/archive Git 事实与 external verifier handoff。

## Risks / Trade-offs

- [产品 tree 相同但验证对象不同] → 同时记录历史 SHA、repository base、tree IDs 和新 exact SHA；只 supersede evidence eligibility，不改旧 verdict。
- [搜索 exit 0 被误读为 PASS] → 固定 `shell_exit=0, semantic_verdict=FAIL, subsequent_gates=NOT_RUN` 三个独立字段。
- [candidate 试图写入自身 SHA] → SHA 只由提交后 Git 与外部 handoff 解析，任何回写产生新 SHA并使旧验证失效。
- [Node UI1 被推广成平台结论] → UI2/UI3 保持有 owner、缺失资产和恢复条件的 `BLOCKED_EXTERNAL`。
- [evidence-only change 被用来顺手修历史或产品] → exact owned-path allowlist 与 protected byte audit 一票否决。

## Migration Plan

1. DRAFT strict PASS 后等待明确 APPROVED，不运行产品 acceptance、不提交。
2. 批准后记录 meaningful structural Red，完整执行 current-tree writer Gate并得到 structural Green/Refactor。
3. 只提交 owned change directory，通过 Git/external handoff 产生 exact candidate SHA与 clean evidence。
4. 另一 clean detached worktree 对该 SHA 从头独立验证；FAIL 返回同一 writer。
5. 获得单独授权后 pure fast-forward integration，验证后 deterministic archive；不 push/PR/deploy。
6. 本 change 实际集成后，后续 receipt change 才可消费 superseding evidence。

回退只整体回退本 evidence change 的集成与 archive 输出；不触碰产品、数据、旧 archive或 Git 历史。

## Open Questions

无。旧 FAIL、三层 lineage、supersedes 边界、W2/UI1 matrix、外部阻塞、ownership 与后续 receipt 依赖已收敛。
