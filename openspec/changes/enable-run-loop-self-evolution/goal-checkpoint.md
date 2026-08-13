# Goal checkpoint

此文件只记录本 Goal/module 的可恢复运行事实；canonical spec 不承载这些运行时值。每次状态、SHA、失败指纹、验证或集成事实变化后，由当前授权角色更新。

## Module base

| field | value |
| --- | --- |
| goal | 先受控升级 `order-run-loop`，再由主 Goal 启动真实菜单目录接入 |
| module | `enable-run-loop-self-evolution` |
| state | `CANDIDATE` |
| repo_base_sha | `2209c071a21860231827b2a8c8c81d9b7745e6e1` |
| runner_skill_git_blob | `d529461de5af1bf7cc65562e59ec3c84f0750963` |
| runner_skill_sha256 | `558b549a4410d72d4c22acad621ffae96af3aeccd26adc186ede76601097aa59` |
| runner_version | `legacy front matter: no version` |
| candidate_sha | `external post-commit evidence: exact clean branch HEAD emitted without changing this candidate tree` |
| integrated_sha | `none` |
| archive_sha | `none` |
| strict | `PASS: openspec validate enable-run-loop-self-evolution --strict` |
| C/T/V/R | `C=9, T=10, V=8, R=9, total=36; writer candidate only, independent result pending` |
| hard_blockers | `0; implementation is gated by approval, not yet authorized` |
| blocked_external | `none` |
| approval | `PASS: main Agent explicitly approved on 2026-08-13 after 4/4 artifacts, strict, ownership, dependency, architecture and hard-blocker Gates passed` |
| architecture_decision | `DRAFT admission requires reproducible + (generalizable OR safety-critical) + non-weakening intent + executable regression/forward-test plans; it is not promotion. Promotion requires implemented regression PASS + clean-detached exact-SHA independent minimal-context forward-test PASS + full independent verification PASS. Activation requires local-main integration and a later module base.` |
| decision_check | `PASS: literal contract checks across proposal/design/spec/tasks plus openspec strict confirmed DRAFT admission != promotion, exact-SHA post-implementation promotion Gates, and local-main/next-module activation` |
| last_decision | `writer Gate PASS; create one owned-path candidate commit, emit exact SHA and clean status externally, then stop for an independent verifier` |

## Lane ledger

| lane | owner | branch | worktree | change | state | SHA | error_fingerprint | repeat_count | dependency | owned_paths | blocker | next |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | ---: | --- | --- | --- | --- |
| self-evolution | unique writer session | `codex/enable-run-loop-self-evolution` | `/Users/vivix/.codex/worktrees/order-run-loop-self-evolution.Writer` | `enable-run-loop-self-evolution` | `CANDIDATE` | `external post-commit evidence` | `none` | 0 | integrated canonical control plane and quality gates | change directory plus `.agents/skills/order-run-loop/**` only | none | independent verifier uses the exact SHA emitted after this record's commit |

## Observation ledger

当前模块开始后只能追加 observation；在模块结束前不得把 observation 应用到当前 runner base。

| id | class | observation | reproduction | generalization_or_safety | non_weakening_intent | regression_plan | forward_test_plan | draft_admission | promotion | activation |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| seed-01 | `checker` | 本轮冻结的七项跨平台 checker 风险需要成为初始回归基线 | 由本 change 的实现阶段 fixtures 给出 | 影响 Gate 真实性与清理安全 | 不降低任何既有 Gate | executable plan in tasks 2.2/3.4/4.2 | executable independent plan in tasks 2.3/5.2 | user-frozen input to this DRAFT, not self-admitted | pending exact-SHA post-implementation Gates | only after local-main integration, for next module |
| observed-02 | `checker` | contract checker initially required a contiguous seven-state sentence absent from the frozen legacy Skill | Green first run reproduced exit 1; the legacy contract is seven explicit transition rows | generalizable checker lesson; assertions must target actual canonical structure | preserve all seven transitions without adding production text for a test | exact transition-row assertions; same contract command now PASS | verifier will rerun contract on exact SHA; no separate runner candidate in this Goal | not screened in active module | not promoted | none |
| observed-03 | `checker` | a zsh audit loop named a variable `path`, shadowing the shell command-search array and causing `git` exit 127 | first audit reproduced `command not found: git`; renaming only the loop variable made the same audit PASS | safety-relevant but already covered by root governance against common system variables | comply with existing governance; do not add a duplicate runner rule | working-tree audit rerun with `candidate_file` PASS | verifier reruns the committed command form | not a new candidate | not promoted | none |
| observed-04 | `checker` | an inline Python f-string embedded in a quoted shell command contained backslash-escaped keys and failed before reading task progress | the first command reproduced a Python `SyntaxError`; replacing the f-string with `.format()` made the same read-only check PASS | tool-layer quoting lesson already covered by the literal-data checker contract | keep the Gate unchanged and simplify transport syntax | corrected task-progress command reported `14/21`, with only verifier/integration tasks pending | no runner-rule candidate; verifier uses repository tasks directly | not a new candidate | not promoted | none |

## Failure ledger

| attempt | command | exit_code | first_decisive_error | SHA_or_environment | fingerprint | action |
| ---: | --- | ---: | --- | --- | --- | --- |
| 1 | `git diff --no-index --check /dev/null .../proposal.md` | 3 | `proposal.md: new blank line at EOF` | DRAFT planning worktree at base `2209c071a21860231827b2a8c8c81d9b7745e6e1` | `diff-no-index-check|3|proposal-new-blank-line-at-eof|2209c071|planning` | removed only the extra final blank line; reran all artifact whitespace checks PASS, so consecutive count reset to 0 |
| 2-3 | `apply_patch` | FAIL | `checkpoint observation table context did not match` | DRAFT planning worktree at base `2209c071a21860231827b2a8c8c81d9b7745e6e1` | `apply-patch|FAIL|checkpoint-table-context-mismatch|2209c071|planning` | same tool fingerprint occurred twice; read exact lines, split the patch and the next targeted attempt PASS, so consecutive count reset before a third attempt |
| 4 | `/usr/bin/python3 .../checks/verify_contract.py --repo .` | 1 | `skill package files mismatch: references/self-evolution.md missing` | legacy runner at base `2209c071a21860231827b2a8c8c81d9b7745e6e1` | `verify-contract|1|missing-self-evolution-reference|2209c071|local-writer` | expected W1 Red; do not alter the checker, proceed to the remaining Red surfaces before minimum implementation |
| 5 | `/usr/bin/python3 .../checks/run_checker_regressions.py` | 1 | `zero_match_count is not implemented` | change-local checker at base `2209c071a21860231827b2a8c8c81d9b7745e6e1` | `checker-regressions|1|zero-match-unimplemented|2209c071|local-writer` | expected W1 Red; checker fixtures remained unchanged, proceed to forward-validator Red |
| 6 | `/usr/bin/python3 .../checks/verify_forward_test.py ... legacy-forward-result.json` | 1 | `current_module_rule_changed must be false` | synthetic legacy result for base `2209c071a21860231827b2a8c8c81d9b7745e6e1` | `forward-validator|1|current-module-rule-changed|2209c071|fixture` | expected validator Red; do not treat it as an independent session, proceed to minimum Green |
| 7 | `/usr/bin/python3 .../checks/verify_contract.py --repo .` | 1 | `legacy control invariant drifted: contiguous seven-state text missing` | first Green runner implementation | `verify-contract|1|synthetic-contiguous-state-token|2209c071|local-writer` | checker false negative; replaced only that assertion with the seven existing transition rows, then the same command PASS and repeat count reset |
| 8 | working-tree scope/whitespace audit | 127 | `zsh: command not found: git` after loop variable `path` shadowed command search | local writer zsh | `working-audit|127|zsh-path-shadowed|2209c071|local-writer` | renamed only the loop variable to `candidate_file`; the same audit PASS and repeat count reset |
| 9 | read OpenSpec apply progress with inline Python | 1 | `SyntaxError: f-string expression part cannot include a backslash` | local writer shell/Python tool boundary | `task-progress|1|f-string-backslash|2209c071|local-writer` | replaced only the inline formatter with `.format()`; corrected read-only command PASS and repeat count reset |
