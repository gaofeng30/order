# Tasks: implement-pickup-options-read-api

## Lifecycle and checklist

- lifecycle: `PRE_CANDIDATE_WIP`
- `candidate_sha`: `NOT_CREATED`
- `formal_exact_sha_review`: `PENDING`
- `clean_detached_verifier`: `PENDING`
- `integration`: `PENDING_NOT_AUTHORIZED`
- `ui_level_target`: `UI1`
- `ui_level_actual`: `NOT_RUN` (current)
- `ui1_historical_receipt`: `PO-08-ui1`
- `ui1_receipt_status`: `INVALIDATED_NOT_CURRENT` after replay/Gate/receipt/spec/tasks changes; exact historical observation was Chrome for Testing `151.0.7922.34`, `TOTAL 3 SUCCESS`.
- `ui1_pre_receipt_writer_receipt`: `PO-08-writer-post-freeze`; UI1 3/3 passed on `source_tree_sha256: 15a2a659c40e6fc3ff05a9cad26e8e46e63341aeff6ae63389ed33113cb30b8c`, while this tasks insertion leaves final staged-tree UI1/Writer evidence pending rerun.
- `post_freeze_writer_gate`: `INITIAL_PASS_RECORDED_FOR_PRE_RECEIPT_STAGED_TREE`; this receipt insertion changes tasks, so the final staged-tree pre-review and Writer Gate rerun remain `PENDING`.
- [x] `PO-01` exact base / ancestry / clean detached start / unique writer branch。
- [x] `PO-02` 完整读取 rules、quality Gate、PRD、CONTEXT 与三项 skills；tracker 诚实为 `GOVERNANCE_PENDING`。
- [ ] `PO-03` 当前 W2/UI1、seams、owned/read-only、contract、RGR、命令与资产已写入治理工件；最终冻结与 post-freeze receipt 尚未完成，不追认未保存的历史中间树。
- [x] `PO-04` replay-only infrastructure self-check + 6 条 Red 已由 `PO-04-replay-current` exact receipt current PASS；该 receipt 写入后仍必须在 full post-freeze Writer Gate 内重跑，不升级为完整 Writer PASS。
- [x] `PO-05` 首轮 post-freeze Writer Gate 已重跑 Green/Refactor、focused/race/determinism、all API test/race、vet/build/smoke。
- [x] `PO-06` 首轮 post-freeze Writer Gate 的 infrastructure shield + 8/8 mutations PASS。
- [x] `PO-07` 首轮 post-freeze Writer Gate 的 fresh loopback MySQL `mysql:8.0.46-oraclelinux9` PASS。
- [x] `PO-08` receipt 写入前 staged tree 的完整 Writer Gate PASS，exact receipt 为 `PO-08-writer-post-freeze`；两次此前失败均为已失效 infrastructure attempts，不是业务 failure。
- [ ] `PO-08-final` stage 本 receipt 后对最终治理树重跑 Standards/Spec 双轴 pre-review 至 0 finding，再对完全相同 staged tree 重跑 Writer Gate；最后 terminal receipt 不反写。
- [ ] `PO-09` 仅 owned paths 提交完整 candidate，worktree clean。
- [ ] `PO-10` exact candidate Standards/Spec 双轴 review 零 finding。
- [ ] `PO-11` 另一 clean detached worktree 对 exact candidate SHA 只读重跑全部 Gate。

## Evidence contract

每条 receipt 都包含质量协议强制字段和显式 `receipt_status`。`candidate_sha: NOT_CREATED` 是当前真实 lifecycle 状态；形成提交后只新增 exact candidate receipts，不追溯改写 Red source tree。`exit_result` 为单值。标记为 historical/invalidated 的运行证据不是 current PASS；`PO-08-writer-post-freeze` 精确绑定 receipt 写入前 staged tree，本次 tasks 变化后仍须按 post-freeze 顺序完成最终 review 与 Writer rerun。

```yaml
task_id: PO-01
receipt_status: CURRENT_FOUNDATION
allowed_phase: writer
phase: writer
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: NOT_RUN
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
source_commit_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
command_or_action: pwd; git status --short --branch; git rev-parse HEAD; git merge-base --is-ancestor exact-base HEAD; git diff --check; git switch -c codex/implement-pickup-options-read-api
exit_result: exit-0
expected: exact clean detached base with true ancestry and unique writer branch
observed: exact base matched; status was clean; ancestry exit 0; branch created in current independent worktree
sanitized_summary: exact-base writer identity established before owned files existed
artifact_or_environment: /Users/vivix/.codex/worktrees/869e/order at source_commit_sha
unverified_boundary: no feature behavior, database, UI, review or verifier evidence
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect immutable base commit and branch reflog
```

```yaml
task_id: PO-02
receipt_status: CURRENT_FOUNDATION
allowed_phase: writer
phase: writer
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: NOT_RUN
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
source_commit_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
command_or_action: full reads of AGENTS.md; CONTEXT.md; change-quality-gates; PRD 5.5, 5.6 and 12; codebase-design plus DEEPENING; tdd plus tests/mocking; code-review; tracker existence check
exit_result: exit-0
expected: required sources readable and missing tracker recorded without fabrication
observed: all required sources read; docs/agents/issue-tracker.md absent
sanitized_summary: W2/UI1, deep Module seam, public-interface TDD and two-axis review rules applied; governance remains pending
artifact_or_environment: repository documents at source_commit_sha
unverified_boundary: reading sources does not prove implementation behavior
external_asset:
  owner: workflow owner
  missing: docs/agents/issue-tracker.md
  recovery: configure tracker only in separately authorized governance work
```

```yaml
task_id: PO-03
receipt_status: HISTORICAL_SEQUENCE_NOT_IMMUTABLY_PRESERVED
allowed_phase: writer
phase: writer
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: NOT_RUN
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
source_commit_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
command_or_action: audit whether an immutable intermediate governance source tree exists for the historical ticket.md, spec.md and initial tasks.md creation
exit_result: FAIL
expected: an immutable source tree proving the governance freeze preceded the first product test
observed: the intermediate uncommitted governance tree was not saved, so chronology is not claimed as an immutable receipt
sanitized_summary: current contract is explicit, but the historical freeze sequence lacks an immutable source-tree binding
artifact_or_environment: exact intermediate tree unavailable; source_commit_sha identifies only the unchanged product base
unverified_boundary: this audit record does not prove freeze-before-Red chronology or runtime behavior
external_asset:
  owner: N/A
  missing: immutable intermediate governance source tree and candidate SHA
  recovery: do not reconstruct history; freeze the current staged governance tree and use the declared post-freeze sequence
```

```yaml
task_id: PO-04-route
receipt_status: INVALIDATED_BY_REPLAY_GATE_CHANGE
allowed_phase: red
phase: red
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: NOT_RUN
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
source_tree_sha256: 038a6c961d29d2a1e40bcd76c1abe29f0d8e59fbe401f60b9d1ddf3ffe9ef560
command_or_action: replay-rgr.sh removes the unique pickup-options registration then runs go test ./services/api/internal/httpapi -run '^TestPickupOptionsRouteFailsClosedAnonymously$' -count=1
exit_result: exit-1
expected: named public route test fails because exact route is absent
observed: named FAIL marker TestPickupOptionsRouteFailsClosedAnonymously; response assertion observed base-equivalent 404 behavior
sanitized_summary: immutable replay proves route capability is required
artifact_or_environment: isolated replay source tree identified by source_tree_sha256; removed after test
unverified_boundary: this Red does not prove success DTO or other projection rules
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun replay-rgr.sh from unchanged Green product tree
```

```yaml
task_id: PO-04-dates
receipt_status: INVALIDATED_BY_DATE_REPLAY_DEFINITION_CHANGE
allowed_phase: red
phase: red
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: NOT_RUN
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
source_tree_sha256: c375e6defe3c2551b6d7d45aacdc1eb396c21ba272940f53940a7a4bac2dc54a
command_or_action: historical replay replaced the valid success projection with a generic unavailable result, then ran go test ./services/api/internal/menu -run '^TestPickupOptionsUsesShanghaiTodayAndTomorrow$' -count=1
exit_result: exit-1
expected: UTC instant crossing Shanghai midnight must yield 2026-08-20 and 2026-08-21 plus no-store
observed: named FAIL marker TestPickupOptionsUsesShanghaiTodayAndTomorrow; replay returned generic unavailable instead of projected dates
sanitized_summary: historical replay did not isolate the Shanghai date projection and is not accepted as current Red evidence
artifact_or_environment: isolated replay source tree identified by source_tree_sha256; removed after test
unverified_boundary: meal enumeration and cutoff facts are separate receipts
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun replay-rgr.sh from unchanged Green product tree
```

```yaml
task_id: PO-04-endpoint
receipt_status: INVALIDATED_BY_REPLAY_GATE_CHANGE
allowed_phase: red
phase: red
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: NOT_RUN
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
source_tree_sha256: 049c82898163f0dac47cd953096d75260a921271c701a38026f312884212b004
command_or_action: replay-rgr.sh changes the range loop from <= end to < end then runs go test ./services/api/internal/menu -run '^TestPickupOptionsIncludesEveryMealAndClosedRangeEndpoint$' -count=1
exit_result: exit-1
expected: both meal groups include exact closed endpoints 13:30 and 19:00 on both dates
observed: named FAIL marker TestPickupOptionsIncludesEveryMealAndClosedRangeEndpoint
sanitized_summary: immutable replay proves closed-range endpoint sensitivity
artifact_or_environment: isolated replay source tree identified by source_tree_sha256; removed after test
unverified_boundary: non-default interval and cutoff are separate receipts
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun replay-rgr.sh from unchanged Green product tree
```

```yaml
task_id: PO-04-interval
receipt_status: INVALIDATED_BY_REPLAY_GATE_CHANGE
allowed_phase: red
phase: red
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: NOT_RUN
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
source_tree_sha256: b0e16019acee8b7b7841b6004cd122a2e7ecd9c5bf0c4eeb24155b156b3810c4
command_or_action: replay-rgr.sh replaces configured interval with 30 then runs go test ./services/api/internal/menu -run '^TestPickupOptionsHonorsConfiguredInterval$' -count=1
exit_result: exit-1
expected: legal 20-minute range yields 11:00, 11:20, 11:40 and 12:00
observed: named FAIL marker TestPickupOptionsHonorsConfiguredInterval
sanitized_summary: immutable replay proves the stored interval is the only enumeration step source
artifact_or_environment: isolated replay source tree identified by source_tree_sha256; removed after test
unverified_boundary: exact cutoff is a separate receipt
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun replay-rgr.sh from unchanged Green product tree
```

```yaml
task_id: PO-04-cutoff
receipt_status: INVALIDATED_BY_REPLAY_GATE_CHANGE
allowed_phase: red
phase: red
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: NOT_RUN
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
source_tree_sha256: 8f1bc327d02f2c50dc6611f311d179467acec5d47e5b06320e0cc7f290246605
command_or_action: replay-rgr.sh changes now.Before(cutoffAt) to inclusive cutoff then runs go test ./services/api/internal/menu -run '^TestPickupOptionsUsesStrictCutoffAndDateOrderabilityOR$/^exact$' -count=1
exit_result: exit-1
expected: exact cutoff meal.orderable is false
observed: named FAIL marker TestPickupOptionsUsesStrictCutoffAndDateOrderabilityOR
sanitized_summary: immutable replay proves strict Before boundary sensitivity
artifact_or_environment: isolated replay source tree identified by source_tree_sha256; removed after test
unverified_boundary: complete fail-closed and database scenarios are separate receipts
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun replay-rgr.sh from unchanged Green product tree
```

```yaml
task_id: PO-05
receipt_status: INVALIDATED_BY_POST_RECEIPT_ARTIFACT_AND_GATE_CHANGES
allowed_phase: refactor
phase: refactor
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: NOT_RUN
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
product_go_tree_sha256: 8df3e2b1f5db9ca1a1f650a9df9cd2a6cd0649efdf2e6131f453b46cdd564f39
command_or_action: focused normal; focused race; focused count 20 on menu and httpapi after complete Green/Refactor
exit_result: exit-0
expected: exact public DTO/error states, single Reader/clock, no List, stable complete projection, legacy menu regression and determinism all pass
observed: both packages passed normal/race and 20 repeated runs
sanitized_summary: Green product tree satisfies provider, consumer, fail-closed and deterministic public contract
artifact_or_environment: pre-candidate Go source tree identified by product_go_tree_sha256
unverified_boundary: Go source hash excludes governance receipts; final Writer gate must rerun after tasks freeze
external_asset:
  owner: N/A
  missing: candidate SHA
  recovery: rerun verify-writer.sh then form immutable candidate
```

```yaml
task_id: PO-06
receipt_status: INVALIDATED_BY_POST_RECEIPT_ARTIFACT_AND_GATE_CHANGES
allowed_phase: refactor
phase: refactor
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: NOT_RUN
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
product_go_tree_sha256: 8df3e2b1f5db9ca1a1f650a9df9cd2a6cd0649efdf2e6131f453b46cdd564f39
command_or_action: verify-mutation-gate.sh validates synthetic go exit 2 becomes harness exit 82, then runs eight isolated reversible mutants
exit_result: exit-0
expected: unique replacement; each go test exact exit 1; matching named FAIL marker for all eight mutants
observed: infrastructure shield PASS expected_exit 82; eight MUTATION_KILLED lines each reported exit 1 and expected marker
sanitized_summary: tests reject missing tomorrow/end/interval, inclusive cutoff, UTC date, removed closed meals, broken date OR and overlap partial success
artifact_or_environment: temporary mutation copies derived from product_go_tree_sha256; removed by trap
unverified_boundary: mutation sensitivity is not real database, UI or independent verification
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun verify-mutation-gate.sh from unchanged product tree
```

```yaml
task_id: PO-07
receipt_status: INVALIDATED_BY_POST_RECEIPT_ARTIFACT_AND_GATE_CHANGES
allowed_phase: refactor
phase: refactor
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: NOT_RUN
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
product_go_tree_sha256: 8df3e2b1f5db9ca1a1f650a9df9cd2a6cd0649efdf2e6131f453b46cdd564f39
command_or_action: verify-mysql.sh provisions loopback mysql 8.0.46 and executes existing services/api/scripts/menu-integration.sh under race
exit_result: exit-0
expected: fresh v1-v13; default 5+5; custom range/interval; cutoff before/exact/after; missing/misaligned/overlap 503; GET no-write; legacy menu unchanged
observed: exact MySQL 8.0.46; menu/catalog/httpapi integration packages passed; follow-up found no container or credential file
sanitized_summary: real isolated MySQL contract passed without migration or persistent configuration writes
artifact_or_environment: disposable mysql:8.0.46-oraclelinux9, loopback ephemeral port and random owned schemas; cleaned
unverified_boundary: production DB, deployed binary, real menu data and real UAT are not covered
external_asset:
  owner: Writer and Verifier
  missing: N/A for local Gate
  recovery: verify-mysql.sh recreates and cleans the exact local asset per source tree
```

```yaml
task_id: PO-08-ui1-initial
receipt_status: HISTORICAL_FAILED_ATTEMPT_NOT_CURRENT
allowed_phase: writer
phase: writer
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: NOT_RUN
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
product_go_tree_sha256: 8df3e2b1f5db9ca1a1f650a9df9cd2a6cd0649efdf2e6131f453b46cdd564f39
command_or_action: npm --prefix tools/miniprogram-ui run ui1
exit_result: exit-1
expected: locked Chromium runner executes three existing adjacent scenarios
observed: ERR_MODULE_NOT_FOUND for karma because current worktree had no node_modules runtime asset
sanitized_summary: first UI1 attempt failed on missing local runtime dependency before browser or scenario execution
artifact_or_environment: current writer worktree before approved runtime-asset reuse
unverified_boundary: environment failure proves neither UI1 PASS nor product/UI failure
external_asset:
  owner: quality runtime owner
  missing: current-worktree locked node_modules
  recovery: verify lock hashes, temporarily reuse exact staging runtime asset, rerun original UI1, remove symlink
```

```yaml
task_id: PO-08-ui1
receipt_status: INVALIDATED_NOT_CURRENT
allowed_phase: writer
phase: writer
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: UI1
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
product_go_tree_sha256: 8df3e2b1f5db9ca1a1f650a9df9cd2a6cd0649efdf2e6131f453b46cdd564f39
command_or_action: verify-ui1.sh checks current/staging package and lock hashes, links locked runtime, runs npm --prefix tools/miniprogram-ui run ui1, then unlinks
exit_result: exit-0
expected: locked real Chromium runs 3 scenarios and runtime symlink is removed
observed: Chrome for Testing 151.0.7922.34; TOTAL 3 SUCCESS; UI1_RESULT PASS scenarios 3; cleanup PASS
sanitized_summary: existing launch/menu/catalog 503-retry/category-sheet UI1 has adjacent no-regression evidence
artifact_or_environment: source-exact tracked tools/apps plus hash-matched staging node_modules; temporary symlink removed
unverified_boundary: future pickup-options mini-program consumer, UI2, UI3, production and real-menu UAT are not proven
external_asset:
  owner: quality runtime owner
  missing: N/A for local UI1
  recovery: rerun verify-ui1.sh while exact staging locked runtime remains hash-compatible
```

```yaml
task_id: PO-08-writer-initial
receipt_status: HISTORICAL_FAILED_ATTEMPT_NOT_CURRENT
allowed_phase: writer
phase: writer
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: UI1
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
product_go_tree_sha256: 8df3e2b1f5db9ca1a1f650a9df9cd2a6cd0649efdf2e6131f453b46cdd564f39
command_or_action: verify-writer.sh runs RGR replay, focused/race/determinism, mutations, fresh MySQL, full test/race/vet/build, smoke, UI1, cleanup, format/diff/owned/sensitive checks
exit_result: exit-98
expected: all declared Writer Gates pass on final frozen tasks/spec/product tree
observed: RGR, focused/race/determinism, mutations, MySQL, full test/race/vet/build, smoke and UI1 passed; final sensitive scan matched its own regex source and exited 98
sanitized_summary: infrastructure self-match invalidated the aggregate Writer receipt without exposing a secret or establishing a product failure
artifact_or_environment: pre-candidate WIP; exact candidate not created
unverified_boundary: staged pre-review, commit, exact candidate review and detached verifier remain pending
external_asset:
  owner: Writer and quality runtime owner
  missing: self-excluding sensitive audit and candidate SHA
  recovery: exclude only verify-writer.sh from its own literal regex, retain all owned product/evidence inputs, then rerun verify-writer.sh from the first RGR replay
```

```yaml
task_id: PO-08-writer-final
receipt_status: INVALIDATED_BY_OWN_RECEIPT_AND_LATER_GATE_CHANGES
allowed_phase: writer
phase: writer
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: UI1
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
product_go_tree_sha256: 8df3e2b1f5db9ca1a1f650a9df9cd2a6cd0649efdf2e6131f453b46cdd564f39
command_or_action: verify-writer.sh after self-excluding sensitive audit fix
exit_result: exit-0
expected: RGR replay, focused/race/determinism, mutations, fresh MySQL, full test/race/vet/build, smoke, UI1, cleanup, format/diff/owned/sensitive all pass
observed: five RGR hashes matched; focused/race/count20 passed; shield plus eight mutants passed; MySQL 8.0.46 passed; full test/race/vet/build and smoke passed; UI1 3 SUCCESS; cleanup/owned/format/diff/sensitive passed; WRITER_GATE PASS printed
sanitized_summary: complete post-freeze Writer Gate passed after the infrastructure-only self-match repair
artifact_or_environment: pre-candidate WIP with product_go_tree_sha256 and current complete governance files
unverified_boundary: this recorded run precedes its own receipt insertion; the identical full Gate must rerun once more before pre-review
external_asset:
  owner: Writer and quality runtime owner
  missing: candidate SHA and independent verifier receipt
  recovery: rerun verify-writer.sh after this receipt insertion; abort pre-review on any nonzero result
```

```yaml
task_id: PO-04-replay-current
receipt_status: CURRENT_REPLAY_ONLY_PENDING_POST_RECEIPT_STAGE_AND_WRITER_RERUN
allowed_phase: red
phase: red
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: NOT_RUN
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
replay_script_sha256: fd4129fa214b126b701173d731fe7e8c780e92ee05685e634361bd434fabb994
command_or_action: .scratch/implement-pickup-options-read-api/replay-rgr.sh
exit_result: exit-0
expected: byte-level infrastructure shield proves missing source exit 79, empty needle exit 78 and unique target-only replacement preserves Query; six isolated Red replays each produce exact go-test exit 1 plus its named FAIL marker
observed: infrastructure PASS missing_exit=79 empty_exit=78 query_count=1; route_missing=068ac8f4542013e8aea953c2a48668b15707288ee21ee43cdfbcd100d653a9fb; shanghai_dates_missing=b4e483a4432d73e20b890ea916e966bcfa5c1f4d31dc499ae7e9b0d35e6bc6cc; configured_pickup_point_missing=43195855a19b132b8b8208f905878239b8f4b778ffeefe679348c47e3faf084b; closed_endpoint_missing=85bccf0f0ca27f6aca1b712b76f9847cfd4bf3d201aae665a25d2d4d7a5aa4c2; configured_interval_missing=5851f1385722799ceed3559bc478c2f1efa9489e92afc417d80548d8e24482bb; strict_cutoff_missing=2a910b37fd5d55ad7706b99c2bb472658e683338c83f5615452c1e1e877239cc; all six reported exit 1 and their exact named FAIL marker; script exit 0
sanitized_summary: byte-exact replay infrastructure no longer corrupts Query and all six current Red definitions are observable and independently named
artifact_or_environment: six isolated temporary services/api source trees identified by the observed SHA-256 values; replay root removed by EXIT trap
unverified_boundary: this replay-only receipt does not prove Green/refactor, mutations, MySQL, race, UI1, complete Writer Gate, candidate review or detached verification; tasks receipt insertion requires restage, staged dual pre-review and full Writer Gate rerun
external_asset:
  owner: N/A
  missing: candidate SHA and current full Writer Gate receipt
  recovery: stage this receipt, rerun staged Standards/Spec pre-review to zero finding, then rerun the full Writer Gate on that exact staged tree
```

```yaml
task_id: PO-08-writer-attempt-testmain
receipt_status: INVALIDATED_INFRASTRUCTURE_ATTEMPT_NOT_BUSINESS_FAILURE
allowed_phase: writer
phase: writer
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: NOT_RUN
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
head_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
command_or_action: bash .scratch/implement-pickup-options-read-api/verify-writer.sh
exit_result: exit-1
expected: complete Writer Gate passes on the exact staged source tree
observed: RGR 6/6 and focused normal passed; focused race stopped at runtime/race package testmain cannot find package before later stages
diagnosis_or_recovery: no code or Gate changed; the exact focused race command then passed once and passed 5/5 further serial executions
sanitized_summary: transient local race-toolchain infrastructure attempt was invalidated and is not a business or contract failure
artifact_or_environment: exact base/head staged writer worktree before candidate creation
unverified_boundary: this attempt proves neither a complete Writer PASS nor any later MySQL, full API, UI1, review or verifier stage
external_asset:
  owner: local Go toolchain runtime owner
  missing: stable race invocation during this attempt
  recovery: exact command stability was diagnosed with one PASS plus serial 5/5 PASS before the full Writer command was retried
```

```yaml
task_id: PO-08-writer-attempt-colima
receipt_status: INVALIDATED_INFRASTRUCTURE_ATTEMPT_NOT_BUSINESS_FAILURE
allowed_phase: writer
phase: writer
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: NOT_RUN
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
head_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
command_or_action: bash .scratch/implement-pickup-options-read-api/verify-writer.sh
exit_result: exit-1
expected: complete Writer Gate provisions fresh loopback MySQL and continues through all later stages
observed: RGR 6/6, focused/race/determinism and mutations 8/8 passed; MySQL stopped before container creation with permission denied connecting to the Colima Docker socket
diagnosis_or_recovery: Colima was stopped during this attempt; after Colima started, the unchanged original full Writer command passed
sanitized_summary: stopped local container runtime invalidated this infrastructure attempt and is not a business, schema or contract failure
artifact_or_environment: local Colima socket unavailable on exact base/head staged writer worktree
unverified_boundary: this attempt proves neither MySQL cases nor any later full API, UI1, cleanup, review or verifier stage
external_asset:
  owner: local container runtime owner
  missing: running Colima Docker API during this attempt
  recovery: start Colima, retain the same staged tree and rerun the unchanged full Writer command
```

```yaml
task_id: PO-08-writer-post-freeze
receipt_status: PASS_PRE_RECEIPT_STAGED_TREE_FINAL_STAGE_REVIEW_AND_RERUN_PENDING
allowed_phase: writer
phase: writer
change: implement-pickup-options-read-api
gate_type: W2
ui_level_target: UI1
ui_level_actual: UI1
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
head_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
source_tree_sha256: 15a2a659c40e6fc3ff05a9cad26e8e46e63341aeff6ae63389ed33113cb30b8c
command_or_action: bash .scratch/implement-pickup-options-read-api/verify-writer.sh
exit_result: exit-0
expected: RGR 6/6; mutation shield and 8/8 mutations; fresh MySQL 8.0.46; focused/determinism and all API normal/race; vet/build/smoke; UI1 3/3; cleanup, owned, diff, format and sensitive checks; exact terminal Writer PASS
observed: RGR 6/6 PASS; mutations 8/8 PASS; fresh MySQL image mysql:8.0.46-oraclelinux9 PASS; focused/determinism and all services/api test/race PASS; vet/build/smoke PASS; locked Chromium UI1 3/3 PASS; temporary MySQL/container credential/UI symlink cleanup and owned/diff/format/sensitive checks PASS; terminal WRITER_GATE=PASS base_sha=f3c4efa4cd665652d93d5da76f92d18c4bdc59ac head_sha=f3c4efa4cd665652d93d5da76f92d18c4bdc59ac source_tree_sha256=15a2a659c40e6fc3ff05a9cad26e8e46e63341aeff6ae63389ed33113cb30b8c
sanitized_summary: unchanged exact staged product and Gate tree completed the full W2 Writer sequence after transient Go-race and stopped-Colima infrastructure attempts were independently cleared
artifact_or_environment: exact pre-receipt staged index identified by source_tree_sha256; fresh loopback mysql:8.0.46-oraclelinux9; locked Chromium UI1 runtime; all temporary assets cleaned
unverified_boundary: receipt insertion changes tasks and invalidates final candidate eligibility until this receipt is staged, final staged Standards/Spec pre-review returns zero finding and the full Writer Gate reruns on that identical final staged tree; candidate, exact-SHA formal review, detached verifier, integration, UI2/UI3, production and real-menu UAT remain unproven
external_asset:
  owner: Writer and quality runtime owner
  missing: candidate SHA, final staged-tree review receipt and final terminal Writer rerun receipt
  recovery: stage only this tasks receipt, rerun staged Standards/Spec pre-review to zero finding, then rerun the unchanged Writer Gate on that exact staged tree without writing the terminal receipt back
```

## Current required replay and post-freeze sequence

The following are Gate declarations, not PASS receipts:

- Shanghai-date Red performs the unique minimal source replacement `now := handler.now().In(shanghaiLocation)` -> `now := handler.now()`. It keeps the handler on the success path and makes `TestPickupOptionsUsesShanghaiTodayAndTomorrow` observe the wrong UTC-derived dates; the named date assertion, rather than a generic 503 assertion, produced exact go-test exit 1 and its own `--- FAIL` marker. Status: `CURRENT_REPLAY_ONLY_PASS_PENDING_FULL_POST_FREEZE_WRITER_RERUN`.
- Complete-enumeration Red removes only the first internal configured pickup point from the successful public DTO while retaining range start/end, the configured interval step, route, Shanghai dates and cutoff calculation. Only `TestPickupOptionsEnumeratesEveryConfiguredPickupTime` ran; its single assertion observed missing `12:00`, with exact go-test exit 1 and its own `--- FAIL` marker. Status: `CURRENT_REPLAY_ONLY_PASS_PENDING_FULL_POST_FREEZE_WRITER_RERUN`.
- The complete changed-path audit is the union of `git diff --name-only <base>...HEAD`, `git diff --cached --name-only`, `git diff --name-only`, and `git ls-files --others --exclude-standard`; the Writer Gate rejects unstaged/untracked source, checks committed/staged/unstaged diffs including `git diff --cached --check`, and emits a stable `source_tree_sha256` from the complete staged index before/after the run.
- After ticket/spec/tasks and all Gate scripts were stage-frozen and staged Standards/Spec pre-review returned zero finding, the first exact full Writer command passed as `PO-08-writer-post-freeze`. Status: `INITIAL_PASS_RECORDED_FOR_PRE_RECEIPT_STAGED_TREE`.
- This receipt insertion invalidates both the first pre-review and Writer Gate for final candidate eligibility: stage the final governance tree, rerun staged Standards/Spec pre-review to zero finding, then rerun the identical Writer Gate on that same final staged tree. The final exact exit-0 terminal receipt is not written back into the frozen governance files; it is the candidate-formation input. Any implementation/governance/Gate/staged-tree change invalidates the affected prior evidence and restarts this order; any nonzero result leaves lifecycle `PRE_CANDIDATE_WIP`.

## Writer C/T/V/R pre-candidate score

- `C = 9`：当前 HTTP DTO、时区/日期/餐段/时间点、orderable 语义、统一错误、既有 `/menu`、非目标和 consumer 缺口均已明确。
- `T = 0`：`PO-08-writer-post-freeze` 已证明 receipt 写入前 staged tree 的完整 Writer Gate PASS；但本次 tasks receipt 写入改变最终治理树，最终 staged-tree pre-review 与 full Writer rerun 仍 `PENDING`，因此当前 candidate-eligible T 分仍不提高。
- `V = 0`：candidate 尚未创建，不能获得 quality protocol 的 candidate exact-SHA 最低 `V = 8`。
- `R = 8`：当前失败回流、证据失效、临时 MySQL/凭据/UI symlink 的清理与重跑条件已明确；历史清理观察不升级其他维度为 PASS。
- `total = 17`；当前不满足每项 `>= 8` 或总分 `>= 36`，不能形成 candidate。只有 post-freeze Writer Gate、candidate exact SHA、正式双轴 review 和 clean detached independent verification 实际完成后，才按新 receipts 更新分数。
