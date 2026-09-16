## Context

当前 profile registry 固定四项，bindings loader 只接受三个历史 binding 或再加一项 receipt-control bootstrap binding。双 Gate governance candidate `f5719e98690d0b1301ed567c8b616e074846b445` 已以单 parent 归档到 `b53cb520c4cac0505803a9d1e6dcc1807ac34540`，但当时主动冻结了 lifecycle profile/binding 路径，所以今天没有合法 profile 可以导出其机械证据。当前规则又禁止业务 change 定义自己的 judge，因此必须先由一个独立控制面 candidate 固定 replay、archive trust 与第五 binding admission。

本 change 为 `W1/UI0`。它修改 profile registry、runner、一个新 wrapper 和聚焦测试，不修改产品、双 Gate checker、Harness、receipt judge/schema、现有 binding/receipt 或 run-loop 规则。现有四个 binding 与 receipt 结果必须保持 byte/semantic non-weakening。

## Goals / Non-Goals

**Goals:**

- 为 exact governance candidate 提供固定、零网络、temp-only、bounded-output 的机械 replay。
- 让 candidate 阶段合法包含一个未绑定的第五 profile definition，同时保持现有四项 binding 有效。
- 只在 profile change exact candidate 已独立验证并精确归档后，允许一个 later binding-only commit 追加唯一 governance binding。
- 用 exact profile candidate 中的 authority bytes 验证既有 governance archive 与 profile change 自身 archive，拒绝工作树自证。
- 保持 binding-head、receipt-head 和 actor/session independence 为后续独立 Gate。

**Non-Goals:**

- Mini Program、Admin、Go 产品逻辑、PRD、双 Gate checker/Harness、receipt schema/judge 或 run-loop 规则变更。
- candidate 内写 binding、receipt、canonical spec 或历史 PASS。
- generic profile plugin、任意第六项 binding、兼容分支、远端 required check、推送、PR、部署或外部写入。

## Decisions

### 1. Runner 只接受一个固定第五 profile/binding 阶段

`mechanical-profiles-v1.json` 精确增加 `miniprogram-user-regression-gate-v1`。`profile_runner.py` 把合法 registry 从四项提高为五项，但 bindings 只允许两个精确序列：当前 `[三个历史项, lifecycle-receipt-control-v1]`，或 later `[当前四项, miniprogram-user-regression-gate-v1]`。第五项的 change、target、tool path 和定义全部固定，重复、换序、第六项、未知项或 mutation 立即失败。

不采用通用“registry 有什么就能绑定什么”，因为那会让任意仓库文本扩大 judge 面；也不在 candidate 中预写未验证 binding，因为 profile source/archive 顺序尚未成立。

### 2. exact candidate `C` 同时固定两条 archive trust 检查

Candidate-owned `checks/` 保存只读 checker 与完整 expected canonical bytes：

- governance 历史归档：固定 `T=f5719e98690d0b1301ed567c8b616e074846b445`、`TA=b53cb520c4cac0505803a9d1e6dcc1807ac34540`，要求 `TA` 唯一 parent 为 `T`，active governance change 全部以同相对路径 `R100` 移入唯一 dated archive，只有声明的三个 canonical specs 可修改且分别逐字节等于 `C` 中 fixture，profile/binding/runner/judge/run-loop/product bytes 在 `T→TA` 不变。
- profile change 归档：固定 dispatcher 提供的 full `C` 与未来 `A`，要求 `A` 唯一 parent 为 `C`，active change 全部 `R100` 移入唯一 dated archive，只有新 capability canonical spec 修改且等于 `C` 中 fixture，candidate-owned runner/registry/wrapper/tests、bindings/judge/run-loop/product bytes在 `C→A` 不变。

第五 binding loader 从 exact `C` Git blob 执行 checker authority；`TA`、`A` 或当前 worktree 的 checker/runner 副本都只作比较对象。这样历史 archive 先于 checker 出现仍可被后来的 independently verified authority 精确审计，但不会倒推 actor independence。

### 3. Profile wrapper 重放最小完整机械矩阵

`miniprogram_user_regression_gate.py` 只接受 `--repo/--temp-root/--git/--python/--node/--npm/--go/--module-cache-download` 固定 argv。它首先确认 target worktree exact `HEAD=T`、detached、clean，然后运行：

- `tools.tests.test_miniprogram_gate` 与 `tools.tests.test_harness`；
- `npm test --prefix apps/wechat-miniprogram`；
- Go format、`go test`、`go test -race`、`go vet`、`go build` 与 API smoke；
- target change strict结构/owned-path/protected-byte/敏感信息的仓库内确定性检查，不调用网络或用户目录工具。

所有 cache/home/tmp 只能位于 runner 提供的 profile temp；wrapper 合并子命令结果，只在全部成功后输出精确一行 `miniprogram-user-regression-gate-v1=MECHANICAL_PASS`。profile 明确排除 OpenSpec CLI 可变实现、Skill quick_validate、UI2/UI3、真实微信与 actor independence。

### 4. Binding 与 receipt 保持候选之后的分段提交

Profile change 先按普通流程形成 `C`，由 minimal-context verifier 与普通 verifier 在不同 clean detached worktree 验证，随后经单独授权 pure-FF 集成并以 exact `C→A` 归档。只有此后 integrator 才能追加 binding-only `B`；不同 verifier 必须在 exact `B` 执行 profile 得到 `MECHANICAL_PASS`。再由 integrator 追加 governance receipt-only `R`，另一 verifier 在 exact `R` 运行 chain/receipt derivation。

`B` 或 `R` 不是本 candidate 的完成任务，也不允许写进 candidate。任一 source、profile definition、target、archive、binding、receipt、runner、test、command 或 SHA 变化使旧验证失效。

## Risks / Trade-offs

- [Runner 固定常量继续增长] → 本 change 只增加一个审计过的第五项；不提供 generic escape hatch，下一项仍需独立 OpenSpec。
- [历史 archive 在 checker 产生前已发生] → checker/fixtures 由 later exact `C` 的两个独立 verifier 审核，并从 Git object 执行；只导出机械事实，不追认 actor independence。
- [完整 Go/race 矩阵较慢] → 接受 bounded 运行时间，避免 profile 只验证 Python 控制面而遗漏治理 candidate 声明的非弱化回归。
- [archive CLI 固定多生成一个 EOF 空行] → `C` 预存通过 `git diff --check` 的完整 expected bytes；archive 必须先运行仓库安装的 CLI，证明输出只比 fixture 多一个 terminal blank line，再仅删除该空行。任何其他差异均阻断 archive/binding，且不得在 A 阶段修改 fixture。
- [第五 binding 会触及 bootstrap binding 历史] → loader 明确允许一个 later fifth-only append，同时继续验证第四 bootstrap 的既有 C/A/B 信任链且禁止其内容变化。

## Migration Plan

1. 在独立 writer worktree 以真实 Red 覆盖第五 profile 缺失、四-binding loader 固定计数和 archive/binding negatives。
2. 最小实现 registry、runner、wrapper、checker/fixtures，重跑相同 focused 与完整回归，形成 exact candidate `C`。
3. 由两个 clean-detached verifier 分别执行 minimal-context forward-test 与普通 exact-C Gate；任一失败返回原 writer。
4. 取得单独授权后 pure-FF 集成；运行仓库安装的 OpenSpec archive，确认 canonical 仅多一个 terminal blank line，删除该空行后按 exact `C→A` archive；archive Gate 从 exact `C` authority bytes 验证。
5. 另行授权 integrator 追加唯一 binding-only `B`，different verifier 通过 binding-head；再追加 receipt-only `R`，another verifier 导出 receipt-head PASS。
6. 在 `R` 关闭前不分配下一业务 module。回滚仅限 `B` 前普通 revert；`B/R` 后不得改写历史 receipt，修复必须另开 change。

## Open Questions

无。target SHA、archive SHA、profile ID、命令面、顺序、owner 与 non-goals 均已由仓库事实和现行 run-loop 规则固定。
