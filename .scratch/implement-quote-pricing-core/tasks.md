# Tasks: implement-quote-pricing-core

## Candidate invalidation

- `7a5412546e9d1c59e1213ea668245e60db52e63e` is `INVALIDATED_BY_INDEPENDENT_STANDARDS_AND_SPEC_REVIEW`.
- P1: eleven completed tasks previously had only three aggregate records, without per-task unified evidence or an explicit `green` phase.
- P2: Spec/tests previously did not freeze or distinguish `INVALID_RATE → EMPTY_LINES → per-line price → quantity → arithmetic` priority.
- All writer review and detached receipts bound to the invalidated SHA are void; it must not be integrated. Replacement remains `candidate_sha: external-post-commit` until immutable handoff.
- `8650359395bd0b5117217dee967ec6b09d831a0b` is `INVALIDATED_BY_INDEPENDENT_STANDARDS_REVIEW` because the aggregate-count evidence checker admitted per-record missing fields/phases and did not require receipt-void semantics.
- Its Spec 0-finding receipt and all writer/review receipts are void; it must not be integrated. The next replacement also remains `candidate_sha: external-post-commit` until immutable handoff.

## Completed tasks and unified evidence
- [x] `QP-01` 核验 exact base、clean detached start、目标 branch 不存在后创建独立 writer branch。

```yaml
task_id: QP-01
evidence_id: QP-01-writer-start
evidence_origin: historical_action_recovered_from_writer_session
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: writer
command_or_action: >-
  Historical exact action: git rev-parse HEAD; git status --short --branch; git show-ref --verify --quiet refs/heads/codex/implement-quote-pricing-core; then git switch -c codex/implement-quote-pricing-core.
exit_result: PASS
sanitized_summary: exact base and clean detached HEAD confirmed; target branch was absent and created without overwrite
artifact_or_environment: initial writer worktree before any owned file existed
unverified_boundary: historical branch provenance only; not a replacement writer/review/verifier receipt
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect git ancestry and immutable replacement handoff
```
- [x] `QP-02` 完整读取适用 AGENTS/CONTEXT、canonical PRD、质量 Gate 与三项 Skill。

```yaml
task_id: QP-02
evidence_id: QP-02-source-read
evidence_origin: historical_read_action_plus_current_structural_replay
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: writer
command_or_action: >-
  Historical action: fully read AGENTS.md, CONTEXT.md, docs/quality/change-quality-gates.md, canonical PRD sections 5.6/6.4/15.6.2/15.6.5, codebase-design SKILL.md plus DEEPENING.md, tdd SKILL.md/tests.md/mocking.md, and code-review SKILL.md; current replay command: test -f AGENTS.md && test -f CONTEXT.md && test -f docs/quality/change-quality-gates.md && test -f docs/product/online-ordering-system-prd-0818.md && test -f .agents/skills/codebase-design/DEEPENING.md && test -f .agents/skills/tdd/SKILL.md && test -f .agents/skills/code-review/SKILL.md
exit_result: PASS
sanitized_summary: source hierarchy, W3 evidence rules, domain vocabulary, deep Module seam, TDD and two-axis review were applied
artifact_or_environment: repository documentation at fixed base plus replacement worktree
unverified_boundary: structural replay proves source availability; historical action records completed reading
external_asset:
  owner: N/A
  missing: docs/agents/issue-tracker.md remains governance pending by explicit delegation
  recovery: initialize tracker only in a separately authorized governance change
```
- [x] `QP-03` 冻结 W3/UI0、Interface、数学/错误、owned/read-only、非目标、RGR、Gate 与外部资产。

```yaml
task_id: QP-03
evidence_id: QP-03-structure-green
evidence_origin: current_replacement_structural_replay
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: green
command_or_action: >-
  test -f .scratch/implement-quote-pricing-core/DRAFT.md && test -f .scratch/implement-quote-pricing-core/spec.md && rg -q 'gate_type.*W3' .scratch/implement-quote-pricing-core/DRAFT.md && rg -q 'Calculate\(input Input\) \(Result, error\)' .scratch/implement-quote-pricing-core/{DRAFT.md,spec.md} && rg -q 'candidate_sha: external-post-commit' .scratch/implement-quote-pricing-core/{DRAFT.md,spec.md,tasks.md}
exit_result: exit-0
sanitized_summary: DRAFT and Spec contain fixed W3/UI0 Interface, ownership, formula, priority, errors, Gate commands and external boundaries
artifact_or_environment: owned scratch design artifacts
unverified_boundary: structural check does not prove Go behavior or independent verification
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun exact structural command
```
- [x] `QP-04` 完成初始 owned-path 与冲突审计。

```yaml
task_id: QP-04
evidence_id: QP-04-conflict-audit
evidence_origin: historical_action_recovered_from_writer_session
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: writer
command_or_action: >-
  Historical exact action before creation: test ! -e .scratch/implement-quote-pricing-core; test ! -e services/api/internal/quotepricing; then rg searched quotepricing and frozen field names in read-only services/api, scratch and docs.
exit_result: PASS
sanitized_summary: both owned roots were absent on exact base and no existing quotepricing Module or same-name Interface was found
artifact_or_environment: initial exact-base writer worktree
unverified_boundary: absence cannot be replayed after implementation; current owned/protected diff is verified by QP-11
external_asset:
  owner: N/A
  missing: N/A
  recovery: reproduce in disposable detached base worktree if historical absence must be re-audited
```
- [x] `QP-05` 取得 public-seam 编译 Red，再只补齐可编译 surface。

```yaml
task_id: QP-05
evidence_id: QP-05-red
evidence_origin: historical_red_recovered_from_writer_session
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: red
command_or_action: >-
  GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/quotepricing -run '^$' -count=1
exit_result: exit-1
sanitized_summary: public_seam_test.go failed to compile because Input, Result and Calculate were undefined
artifact_or_environment: writer worktree with doc.go and external public seam test
unverified_boundary: historical compile Red does not prove replacement Green or behavior
external_asset:
  owner: N/A
  missing: N/A
  recovery: current compile Green is replayable below
```
```yaml
task_id: QP-05
evidence_id: QP-05-green
evidence_origin: current_replacement_replay
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: green
command_or_action: >-
  GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/quotepricing -run '^$' -count=1
exit_result: exit-0
sanitized_summary: public Calculate Interface and supporting exported types compile through external package seam
artifact_or_environment: replacement writer worktree with Go 1.26.5
unverified_boundary: compile-only Green does not prove pricing behavior
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun exact command
```
- [x] `QP-06` 完成 half-up、逐商品非整单舍入、数量在舍入后相乘的 Red/Green。

```yaml
task_id: QP-06
evidence_id: QP-06-red
evidence_origin: historical_first_decisive_red_recovered_from_writer_session
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: red
command_or_action: >-
  GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/quotepricing -run '^TestCalculateRoundsUnitHalfUpBeforeQuantity$' -count=1
exit_result: exit-1
sanitized_summary: named test observed zero Result instead of worked example 101x85 percent quantity 2 producing 202/172/30
artifact_or_environment: writer worktree at first pricing tracer
unverified_boundary: first decisive Red only; grouped Green covers all three named behaviors
external_asset:
  owner: N/A
  missing: N/A
  recovery: mutation Gate reproducibly proves sensitivity
```
```yaml
task_id: QP-06
evidence_id: QP-06-green
evidence_origin: current_replacement_replay
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: green
command_or_action: >-
  GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/quotepricing -run '^(TestCalculateRoundsUnitHalfUpBeforeQuantity|TestCalculateRoundsEachUnitBeforeSumming|TestCalculateMultipliesQuantityAfterUnitRounding)$' -count=1
exit_result: exit-0
sanitized_summary: named tests pass 101x85 percent, two 1-cent lines at 50 percent, and quantity-after-rounding examples
artifact_or_environment: replacement writer worktree public-seam tests
unverified_boundary: focused Green does not prove invalid input, overflow, race or adjacent packages
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun exact focused command
```
- [x] `QP-07` 完成 rate/price/quantity/empty 合法边界与单一非法输入 Red/Green。

```yaml
task_id: QP-07
evidence_id: QP-07-red
evidence_origin: historical_first_decisive_red_recovered_from_writer_session
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: red
command_or_action: >-
  GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/quotepricing -run '^TestCalculateRejectsRateOutsideMathematicalRange$' -count=1
exit_result: exit-1
sanitized_summary: named test failed to compile because stable ErrorKind/Error types were not yet defined
artifact_or_environment: writer worktree at first invalid-input tracer
unverified_boundary: first decisive Red only; grouped Green covers named boundary and single-invalid tests
external_asset:
  owner: N/A
  missing: N/A
  recovery: mutation Gate reproducibly proves rate/empty/error sensitivity
```
```yaml
task_id: QP-07
evidence_id: QP-07-green
evidence_origin: current_replacement_replay
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: green
command_or_action: >-
  GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/quotepricing -run '^(TestCalculateRejectsRateOutsideMathematicalRange|TestCalculateAcceptsZeroAndOneHundredPercent|TestCalculateRejectsNegativeUnitPrice|TestCalculateAcceptsZeroPrice|TestCalculateRejectsNonPositiveQuantity|TestCalculateRejectsEmptyLines)$' -count=1
exit_result: exit-0
sanitized_summary: named tests accept 0/100 percent and zero price while rejecting invalid rate, negative price, non-positive quantity and empty lines
artifact_or_environment: replacement writer worktree public-seam tests
unverified_boundary: single-invalid Green does not prove multi-invalid priority; QP-13 covers it
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun exact focused command
```
- [x] `QP-08` 完成乘法/加法 overflow 与零 Result fail-closed Red/Green。

```yaml
task_id: QP-08
evidence_id: QP-08-red
evidence_origin: historical_first_decisive_red_recovered_from_writer_session
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: red
command_or_action: >-
  GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/quotepricing -run '^TestCalculateRejectsOriginalLineMultiplicationOverflow$' -count=1
exit_result: exit-1
sanitized_summary: unchecked MaxInt64 times 2 wrapped negative instead of zero Result plus OVERFLOW
artifact_or_environment: writer worktree at first overflow tracer
unverified_boundary: first decisive Red only; grouped Green covers named arithmetic/fail-closed tests
external_asset:
  owner: N/A
  missing: N/A
  recovery: mutation Gate injects named arithmetic faults
```
```yaml
task_id: QP-08
evidence_id: QP-08-green
evidence_origin: current_replacement_replay
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: green
command_or_action: >-
  GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/quotepricing -run '^(TestCalculateRejectsOriginalLineMultiplicationOverflow|TestCalculateRejectsDiscountMultiplicationOverflow|TestCalculateRejectsCrossLineSumOverflow|TestCalculateDiscardsPartialResultOnLaterInvalidLine)$' -count=1
exit_result: exit-0
sanitized_summary: named overflow and later-invalid tests return exact zero Result with stable expected error kind
artifact_or_environment: replacement writer worktree public-seam tests
unverified_boundary: focused arithmetic Green does not prove race cleanliness or adjacent packages
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun exact focused command
```
- [x] `QP-09` 覆盖顺序、输入不变、重复/并发确定性并完成 Refactor focused/race Gate。

```yaml
task_id: QP-09
evidence_id: QP-09-refactor
evidence_origin: current_replacement_replay
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: refactor
command_or_action: >-
  set -e; GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/quotepricing -count=1; GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/internal/quotepricing -count=20
exit_result: exit-0
sanitized_summary: full package including line order/input preservation, repeated success/error determinism and 32-worker concurrency passed; race count 20 passed
artifact_or_environment: replacement writer worktree with Go race detector
unverified_boundary: pure Module race evidence does not prove caller, DB, router, order or payment integration
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun exact two-command shell action
```
- [x] `QP-10` mutation shield 拒绝基础设施失败，11 个可逆 mutant 全部由命名行为断言杀死。

```yaml
task_id: QP-10
evidence_id: QP-10-mutation
evidence_origin: current_replacement_replay
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: refactor
command_or_action: >-
  bash .scratch/implement-quote-pricing-core/verify-mutation-gate.sh
exit_result: exit-0
sanitized_summary: infrastructure exit 2 was rejected via harness exit 82; all 11 mutants exited exactly 1 with named FAIL marker and source count exactly one
artifact_or_environment: isolated temporary mutation copies cleaned by harness
unverified_boundary: mutation sensitivity proves only 11 named faults
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun exact mutation Gate
```
- [x] `QP-11` fresh MySQL 全 API test/race 与 vet/build/smoke/static/scope/sensitive Gate。

```yaml
task_id: QP-11
evidence_id: QP-11-mysql-writer
evidence_origin: current_replacement_replay
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: writer
command_or_action: >-
  GOPROXY=off GOTOOLCHAIN=go1.26.5 bash .scratch/repair-version-scoped-mysql-migration-fixtures-v13/verify-mysql.sh full
exit_result: exit-0
sanitized_summary: fresh mysql:8.0.46-oraclelinux9 reports 8.0.46 on loopback-only ephemeral binding; full services/api test and race pass
artifact_or_environment: disposable loopback-only MySQL container cleaned by existing harness
unverified_boundary: adjacent MySQL regression does not prove pure pricing Module or production data
external_asset:
  owner: writer/verifier
  missing: verifier must independently recreate database
  recovery: rerun full profile from immutable replacement SHA
```
```yaml
task_id: QP-11
evidence_id: QP-11-static-writer
evidence_origin: current_replacement_replay
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: writer
command_or_action: >-
  set -e; GOPROXY=off GOTOOLCHAIN=go1.26.5 go vet ./services/api/...; GOPROXY=off GOTOOLCHAIN=go1.26.5 go build ./services/api/...; GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/smoke.sh; test -z "$(gofmt -l services/api/internal/quotepricing)"; bash -n .scratch/implement-quote-pricing-core/verify-evidence.sh .scratch/implement-quote-pricing-core/verify-evidence-gate.sh .scratch/implement-quote-pricing-core/verify-mutations.sh .scratch/implement-quote-pricing-core/verify-mutation-gate.sh; bash .scratch/implement-quote-pricing-core/verify-evidence-gate.sh; git diff --cached --check; changed_paths=$(git diff --cached --name-only); printf '%s\n' "${changed_paths}" | awk '!/^\.scratch\/implement-quote-pricing-core\// && !/^services\/api\/internal\/quotepricing\// {bad=1} END {exit bad}'; if git diff --cached | rg -i 'authorization[[:space:]]*:|cookie[[:space:]]*:|begin [a-z ]*private key|api[_-]?v3[_-]?key[[:space:]]*[:=]|openid[[:space:]]*[:=]|session[_-]?code[[:space:]]*[:=]' >/dev/null; then exit 72; fi; git diff --quiet; test -z "$(git ls-files --others --exclude-standard)"
exit_result: exit-0
sanitized_summary: vet/build zero, smoke PASS, format/shell/evidence/diff/owned/protected/sensitive and unstaged-clean audits pass
artifact_or_environment: staged replacement candidate in writer worktree
unverified_boundary: pre-commit writer checks do not prove post-commit exact SHA
external_asset:
  owner: N/A
  missing: replacement SHA and detached receipt pending
  recovery: commit only staged owned paths then review and verify exact SHA
```
- [x] `QP-12` 修复 P1：每个完成 task 就地附统一 evidence record，并以逐 record checker 和负向 failure shield 验证完整性。

```yaml
task_id: QP-12
evidence_id: QP-12-evidence-failure-shield-red
evidence_origin: current_replacement_structural_red
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: red
command_or_action: >-
  bash .scratch/implement-quote-pricing-core/verify-evidence-gate.sh
exit_result: exit-1
sanitized_summary: named EvidenceMissingTopLevelField mutant survived with exit 0 because the old checker compared only whole-file field totals
artifact_or_environment: writer worktree after adding the negative Gate but before replacing aggregate-count validation
unverified_boundary: first decisive structural Red; Green below covers all per-record and invalidation negative cases
external_asset:
  owner: N/A
  missing: N/A
  recovery: replay the committed negative Gate against the replacement checker
```

```yaml
task_id: QP-12
evidence_id: QP-12-evidence-structure-green
evidence_origin: current_replacement_replay
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: green
command_or_action: >-
  bash .scratch/implement-quote-pricing-core/verify-evidence-gate.sh
exit_result: exit-0
sanitized_summary: every fenced record has each top-level and external_asset field exactly once with valid phase/exit enums; required phase pairs and both invalidated-candidate receipt semantics are explicit; five missing-field/phase/invalidation mutants are rejected nonzero
artifact_or_environment: owned tasks.md plus deterministic per-record checker and negative failure-shield Gate
unverified_boundary: structural Gate proves evidence shape and declared provenance, not the underlying commands that are replayed separately
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun exact evidence failure-shield Gate
```
- [x] `QP-13` 修复 P2：冻结多重非法输入优先级，并以两条组合测试和两个 priority mutant 闭合。

```yaml
task_id: QP-13
evidence_id: QP-13-rate-empty-red
evidence_origin: current_replacement_temporary_adversarial_red
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: red
command_or_action: >-
  In isolated temporary module copy, move empty-lines before rate validation; run GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/quotepricing -run '^TestCalculateRejectsInvalidRateBeforeEmptyLines$' -count=1
exit_result: exit-1
sanitized_summary: named test returned EMPTY_LINES instead of INVALID_RATE with exact FAIL marker
artifact_or_environment: isolated temporary adversarial copy moved to local Trash after capture
unverified_boundary: temporary Red proves only rate-versus-empty priority sensitivity
external_asset:
  owner: N/A
  missing: N/A
  recovery: final mutation harness injects same fault
```
```yaml
task_id: QP-13
evidence_id: QP-13-price-quantity-red
evidence_origin: current_replacement_temporary_adversarial_red
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: red
command_or_action: >-
  In isolated temporary module copy, move quantity before price validation; run GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/quotepricing -run '^TestCalculateRejectsNegativePriceBeforeNonPositiveQuantity$' -count=1
exit_result: exit-1
sanitized_summary: named test returned INVALID_QUANTITY instead of INVALID_PRICE with exact FAIL marker
artifact_or_environment: isolated temporary adversarial copy moved to local Trash after capture
unverified_boundary: temporary Red proves only price-versus-quantity priority sensitivity
external_asset:
  owner: N/A
  missing: N/A
  recovery: final mutation harness injects same fault
```
```yaml
task_id: QP-13
evidence_id: QP-13-priority-green
evidence_origin: current_replacement_replay
change: implement-quote-pricing-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8bcdf3d6b1ea41529adaa54f463cc118c69e0e25
candidate_sha: external-post-commit
phase: green
command_or_action: >-
  GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/quotepricing -run '^(TestCalculateRejectsInvalidRateBeforeEmptyLines|TestCalculateRejectsNegativePriceBeforeNonPositiveQuantity)$' -count=1
exit_result: exit-0
sanitized_summary: public Calculate seam returns INVALID_RATE before EMPTY_LINES and INVALID_PRICE before INVALID_QUANTITY with zero Result
artifact_or_environment: replacement writer worktree external-package combination tests
unverified_boundary: two combinations do not enumerate every later-line/arithmetic combination; Spec freezes full traversal order and full tests cover traversal
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun exact focused command plus mutation Gate
```
## Pending post-commit tasks

- [ ] `QP-14` 只提交 owned paths，中文 replacement commit；immutable external handoff 绑定新 exact SHA。
- [ ] `QP-15` 对 replacement exact SHA 从零并行完成 Standards/Spec 双轴审查，零 finding。
- [ ] `QP-16` 在 fresh clean detached worktree 对 replacement exact SHA 从头重跑全部 Gate，writer/verifier 均 clean。

> QP-14/QP-15/QP-16 在实际 external receipt 前保持 pending，不预宣 PASS。integration 不在本 change 授权范围内。
