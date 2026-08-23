# Tasks: implement-store-operating-status-command-core

## Checklist

- [x] Read applicable `AGENTS.md`, `$tdd`, `$codebase-design`, `CONTEXT.md`, canonical PRD clauses, quality Gate, current storefront/merchantidentity and v11/v13 migrations.
- [x] Confirm exact base/HEAD/clean, target branch absence and independent branch creation before writing files.
- [x] Freeze full Spec/ticket/tasks with dependencies, ownership, read-only paths, non-goals, W3/UI0, seams, Writer/dual-review/detached/integration commands and external assets N/A.
- [x] Slice SS-01: invalid input is rejected before DB/authorization.
- [x] Slice SS-02: real MySQL OWNER and SUBACCOUNT updates mutate only `business_status`.
- [x] Slice SS-03: disabled/deleted/role-drift and role-change ordering are live and fail closed where required.
- [x] Slice SS-04: no-op, replay and conflict preserve first-result idempotency.
- [x] Slice SS-05: concurrent same/conflicting/different keys serialize correctly.
- [x] Slice SS-06: exact audit and audit failure atomicity.
- [x] Slice SS-07: controlled commit, deadlock, missing/bad row and DB failure recovery.
- [x] Mutation infrastructure shield and all ten required mutants pass.
- [x] Refactor and rerun focused, race, fresh MySQL, mutation, full API, vet, controlled build, smoke and hygiene Gates.
- [ ] Create one complete Chinese owned-only commit and confirm clean candidate handoff.
- [ ] Controller: exact candidate Standards review has zero findings.
- [ ] Controller: exact candidate Spec review has zero findings.
- [ ] Controller: fresh clean detached exact-SHA verifier passes all Gates.
- [ ] Controller: separately authorize and perform integration; Writer does not push/PR/merge/deploy.

## Evidence ledger

Only append evidence that actually occurred. Each Red/Green record must name the exact command and immutable Git tree object containing the tested source and test. Evidence never contains credentials, personal data, raw requests or external payloads.

```yaml
task_id: SS-00
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: writer
command_or_action: exact HEAD/base/status verification followed by target branch absence check and branch creation
exit_result: PASS
sanitized_summary: detached HEAD equaled the full fixed base, worktree and index were clean, and the previously absent target branch was created from that SHA
artifact_or_environment: local independent Writer worktree
unverified_boundary: no implementation behavior existed yet
external_asset:
  owner: N/A
  missing: N/A
  recovery: N/A
```

```yaml
task_id: SS-01
evidence_id: SS-01-invalid-status-red
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/storestatus -run '^TestApplyRejectsInvalidStatusBeforeDependencies$' -count=1
exit_result: exit-1
sanitized_summary: the public command seam did not exist, so the named invalid-status tracer failed to compile with undefined New, Command, Result and ErrInvalidCommand
exact_sha: 0458a0efa5f468162dcfea973abfc62092b12923
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree
unverified_boundary: no valid command or database behavior existed
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command
```

```yaml
task_id: SS-01
evidence_id: SS-01-invalid-status-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/storestatus -run '^TestApplyRejectsInvalidStatusBeforeDependencies$' -count=1
exit_result: exit-0
sanitized_summary: the minimum public Core, Command, Result and invalid-status guard rejected an illegal enum before touching nil dependencies
exact_sha: 3a99b7941e72ff31bf978f3b93faa7df9fcbd1f2
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree
unverified_boundary: the remaining malformed fields and every valid transaction path were not implemented
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command
```

```yaml
task_id: SS-01
evidence_id: SS-01-malformed-command-red
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/storestatus -run '^TestApplyRejectsMalformedCommandBeforeDependencies$' -count=1
exit_result: exit-1
sanitized_summary: all eight malformed user, key and request cases reached the unavailable fallback instead of the invalid-command guard
exact_sha: 0f9ddbd1d02f43c86082f237f8df9a15a65cf104
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree
unverified_boundary: valid dependency and database behavior remained absent
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command
```

```yaml
task_id: SS-01
evidence_id: SS-01-malformed-command-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/storestatus -run '^(TestApplyRejectsInvalidStatusBeforeDependencies|TestApplyRejectsMalformedCommandBeforeDependencies)$' -count=1
exit_result: exit-0
sanitized_summary: all frozen user, status, idempotency-key and request-ID invalid forms now return a zero result and ErrInvalidCommand before nil dependencies are touched
exact_sha: 85df578190d038de05d9c8bf8c5f476ed64df2be
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree
unverified_boundary: valid commands still had no transaction behavior
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command
```

```yaml
task_id: SS-03
evidence_id: SS-03-disabled-account-red
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-1
sanitized_summary: after restoring the local Docker runtime, fresh MySQL v1-v13 showed that a disabled OWNER returned generic unavailable instead of the live Authorizer account-not-available result
exact_sha: 405aa1656192f70ded2b837da8e28bef728a00ce
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus disposable loopback mysql:8.0.46-oraclelinux9
unverified_boundary: enabled accounts and status mutation remained absent
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-03
evidence_id: SS-03-disabled-account-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-0
sanitized_summary: the minimum transaction now invokes the real transaction-bound Authorizer with the frozen action and singleton target; disabled OWNER is rejected with no status or audit write
exact_sha: ddb88501a465605b033e9616fad4b12ab9f368bb
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: enabled accounts still returned unavailable without mutation
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-02
evidence_id: SS-02-owner-single-column-red
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-1
sanitized_summary: fresh MySQL authorization succeeded for an enabled OWNER, but Apply still returned unavailable and did not produce the requested open-to-closed result
exact_sha: 33c17c0535ed38118ab50ce153e4a67737c0c493
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: no enabled-account write or success audit existed
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-02
evidence_id: SS-02-owner-single-column-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-0
sanitized_summary: an enabled OWNER now commits open-to-closed through Apply, exactly one success audit exists, and all ten non-status singleton columns remain identical
exact_sha: 6f35f16cc9d41a0148cf6167efbab454355a86d3
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: SUBACCOUNT, exact audit facts, idempotency and concurrency remained unimplemented
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-02
evidence_id: SS-02-subaccount-single-column-red
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-1
sanitized_summary: the enabled SUBACCOUNT was live-authorized but the owner-only implementation returned unavailable instead of committing closed-to-cutoff
exact_sha: 2fddede5ebfc612b43353c0a09fa2b73bce5eca4
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: SUBACCOUNT write support and audit-role facts were absent
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-02
evidence_id: SS-02-subaccount-single-column-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-0
sanitized_summary: both live OWNER and SUBACCOUNT now commit through the same Apply seam, mutate only business_status and leave every other singleton column unchanged
exact_sha: 96f3e72f3dfc0e85195f891d087c733b2f9b1603
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: audit snapshots, idempotency, role competition and concurrency remained incomplete
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-06
evidence_id: SS-06-exact-audit-red
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-1
sanitized_summary: the success audit existed but its account, role, auth-version, target and key-hash facts were null, so the exact durable audit scan failed
exact_sha: ad7335f01817d894402a68558894278f7885a3f7
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: exact audit persistence and replay proof were absent
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-06
evidence_id: SS-06-exact-audit-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-0
sanitized_summary: success now durably records the exact live account, SUBACCOUNT role, auth version, actor, frozen action/target, raw-key SHA256, request, before/after state, reason and microsecond UTC time
exact_sha: 976bd65fca39c327f0273a0aa66b7647de456891
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: no-op reason, replay, conflict and failure rollback were not yet covered
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-04
evidence_id: SS-04-noop-red
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-1
sanitized_summary: an open-to-open command executed a zero-row UPDATE and returned unavailable instead of the required unchanged first result and success audit
exact_sha: 49fddbbffbb92945d1335a8fe03ee6008e510ee7
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: no-op and later replay behavior were absent
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-04
evidence_id: SS-04-noop-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-0
sanitized_summary: a no-op skips UPDATE, returns open-to-open with Changed false, preserves all singleton columns and writes one exact unchanged success audit
exact_sha: 97fdf853f19db8c1f889e5c4d8b7d3a2a3e97a10
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: duplicate-key replay and conflict remained absent
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-04
evidence_id: SS-04-replay-red
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-1
sanitized_summary: after an intervening command restored open, replaying the original key rewrote status to closed and produced another audit instead of returning the first result read-only
exact_sha: d0fea4fe1aa33fcfd44f9e2e3edaeab550c69c65
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: matching replay lookup and conflict handling were absent
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-04
evidence_id: SS-04-replay-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-0
sanitized_summary: matching actor/action/resource/key-hash/desired-state replay now returns the first open-to-closed result after an intervening command without rewriting status or adding an audit
exact_sha: 804fcd0092387f363a53246c01c9fde3abf72b10
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: same-key different-desired conflict still fell through as a new command
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-04
evidence_id: SS-04-conflict-red
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-1
sanitized_summary: the same actor/action/key with a different desired status fell through as a new closed-to-cutoff success instead of returning a zero idempotency conflict
exact_sha: e195c1be318dc5833e45ffeabf01b9f9be53d8f2
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: first-audit target conflict inspection was absent
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-04
evidence_id: SS-04-conflict-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-0
sanitized_summary: replay now inspects the first same-actor/action/key audit and rejects a different resource or desired status without a state or audit write
exact_sha: 5c1b25eb17453d370bd4b62d434601b9ff2c33bb
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: concurrent callers could still inspect replay state without a singleton row lock
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-05
evidence_id: SS-05-concurrent-replay-red
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-1
sanitized_summary: a real two-transaction barrier let both same-key calls read before serialization; one caller returned unavailable instead of converging to the first result
exact_sha: 824692cbec82b83b08343af1d6721be052cd5ae4
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: singleton FOR UPDATE serialization was absent
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-05
evidence_id: SS-05-concurrent-replay-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-0
sanitized_summary: singleton SELECT FOR UPDATE now serializes two barrier-synchronized same-key transactions; both return the first open-to-closed result with one state write and one audit
exact_sha: 809d8bf4764541122678ddd528f02fcd377c37ba
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: concurrent conflicting and distinct keys still required explicit regression assertions
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-03
evidence_id: SS-03-live-account-coverage-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-0
sanitized_summary: fresh MySQL additionally rejects deleted and schema-drifted invalid-role accounts; an OWNER transaction blocks a competing OWNER-to-SUBACCOUNT change, then the next request succeeds and audits the new SUBACCOUNT auth version
exact_sha: d05ebdeb015931c42ef224c98419c5ef0f3d6114
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: commit/deadlock and storage-failure recovery were still pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-05
evidence_id: SS-05-concurrency-coverage-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-0
sanitized_summary: the same row-lock implementation also yields one winner plus one conflict for concurrent same-key different-desired calls and an exact committed before/after chain for concurrent different keys
exact_sha: d05ebdeb015931c42ef224c98419c5ef0f3d6114
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: this is expanded Green coverage on the prior concurrency Red/Green implementation, not a separate implementation Red
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-06
evidence_id: SS-06-audit-failure-recovery-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-0
sanitized_summary: hiding the audit table makes Apply fail unavailable and rolls the status update back; restoring the table and retrying the same key commits exactly one state change and audit
exact_sha: 4233a10dcf285b70c85cb82d92ed5963f99fb8e8
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: controlled commit failure and real deadlock retry were pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-07
evidence_id: SS-07-row-recovery-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-0
sanitized_summary: missing and invalid singleton status rows fail with zero audit; restoring a valid singleton lets the same key succeed on the next attempt, and a zero clock fails before transaction side effects
exact_sha: 4233a10dcf285b70c85cb82d92ed5963f99fb8e8
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: these are expanded recovery assertions on existing fail-closed paths; commit and deadlock had not completed Red-Green
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-07
evidence_id: SS-07-commit-failure-red
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-1
sanitized_summary: the controlled commit adapter was ignored, so Apply returned a committed open-to-closed success instead of unavailable with rollback
exact_sha: 3d6c22e37fafa34fcb7413769bd148382611cd7e
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: commit failure was not routed through the private fault seam
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-07
evidence_id: SS-07-commit-failure-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-0
sanitized_summary: both new-command and replay commits now pass through the private commit adapter; controlled rollback returns unavailable with zero state/audit, then the same key succeeds exactly once through the production adapter
exact_sha: 20789be48659a4d34f84ed6d8ffb80b2a21d9795
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: real deadlock still returned unavailable without retry
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-07
evidence_id: SS-07-deadlock-red
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-1
sanitized_summary: a weighted real account/storefront lock cycle selected Apply as the deadlock victim; Apply returned unavailable instead of retrying and observing the committed SUBACCOUNT auth version
exact_sha: 7c385789c79c891cdc7cea59ae0aa494511a67ce
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46 and performance_schema lock proof
unverified_boundary: retry classification and full transaction restart were absent
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-07
evidence_id: SS-07-deadlock-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-0
sanitized_summary: one MySQL 1213/1205 restarts the entire transaction once; the weighted real deadlock then succeeds only after fresh authorization observes committed SUBACCOUNT auth version 2 and produces one exact audit
exact_sha: 00490f75f487f9812ccb3599e3580db56d65ad38
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46 and performance_schema lock proof
unverified_boundary: DB disconnect recovery and mutation Gate were still pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: MUT-00
evidence_id: mutation-gate-audit-atomicity-red
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mutation-gate.sh
exit_result: exit-1
sanitized_summary: the infrastructure shield and first nine mutants passed, but the initial audit-failure mutant did not actually force the failed transaction down a successful commit return and therefore survived the named assertion
exact_sha: 4ebff3a788cb9c546fc150a958bdc3de6d730714
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus disposable source copy and fresh loopback MySQL 8.0.46
unverified_boundary: the tenth mutant required a stronger exact replacement before the mutation Gate could be claimed
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-06
evidence_id: SS-06-audit-insert-atomicity-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-0
sanitized_summary: a real BEFORE INSERT rejection lets replay SELECT succeed but makes the success-audit write fail; the status update rolls back, trigger removal lets the same key recover, and exactly one final audit exists
exact_sha: 0f1e6f38b99001519cebc580c211423b788795a6
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: repository-wide Refactor Gates had not run
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-07
evidence_id: SS-07-database-recovery-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh
exit_result: exit-0
sanitized_summary: a closed database handle returns zero unavailable without state or audit; opening a new connection to the restored isolated schema lets the same key commit exactly once
exact_sha: 0f1e6f38b99001519cebc580c211423b788795a6
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus fresh disposable loopback MySQL 8.0.46
unverified_boundary: this is expanded Green coverage on the SS-07 commit/deadlock Red-Green implementation rather than a new implementation Red
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: MUT-00
evidence_id: mutation-gate-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-mutation-gate.sh
exit_result: exit-0
sanitized_summary: the zero-match infrastructure shield passed; all ten exact-match disposable mutants were killed by their named assertions, including illegal enum, authorization action/target, row lock, replay/conflict, single-column update, exact audit and audit-insert rollback; writer source remained byte-identical
exact_sha: 0f1e6f38b99001519cebc580c211423b788795a6
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree plus disposable source copy and fresh loopback MySQL 8.0.46
unverified_boundary: repository-wide Refactor Gates had not run
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command in a fresh fixture
```

```yaml
task_id: SS-08
evidence_id: writer-refactor-gates-green
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: refactor
command_or_action: focused normal; focused race count 20; fresh MySQL focused race count 20; fresh MySQL full services/api normal and race; local full services/api normal and race; go vet; controlled verify-build.sh; repository smoke
exit_result: all exit-0
sanitized_summary: unchanged production behavior passed focused and repository-wide regression, race and W3 integration checks; real MySQL package race count 20 completed in 64.7 seconds; vet and controlled two-binary build were clean; smoke printed PASS
exact_sha: fc528e3db4f11c561f8f71232b25b71ec8ed7eca
exact_object_type: tree
artifact_or_environment: immutable pre-receipt Writer Git tree plus disposable loopback MySQL 8.0.46 and controlled private build directory
unverified_boundary: tracked receipts and the consolidated final Writer Gate script are later task-only changes, so this pre-receipt tree is not the candidate and the complete Gate must rerun before commit
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact commands
```

```yaml
task_id: SS-08
evidence_id: consolidated-writer-gate-red
change: implement-store-operating-status-command-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 8cae09d5bc3e659d8851e7588835e579101058ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: bash .scratch/implement-store-operating-status-command-core/verify-writer.sh
exit_result: exit-2
sanitized_summary: the consolidated Gate stopped before behavior commands because its sensitive-pattern shell quoting had an unmatched quote; no test, database, build or external side effect ran
exact_sha: 752919b93e3664f28d1d814301031f4433a6a0a9
exact_object_type: tree
artifact_or_environment: immutable Writer Git tree and local shell syntax shield
unverified_boundary: the corrected final tree required the complete Writer Gate from the beginning
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the tree object and replay the exact command
```

## Candidate scoring target

- C >= 9: one frozen Interface, exact transaction order, live role semantics, error modes and ownership are explicit.
- T >= 10: every requested invalid, transaction, idempotency, concurrency, audit, recovery and mutation case is executable.
- V = 8 at Writer handoff: complete exact-SHA clean-detached package awaits controller review/verifier; no independent PASS is claimed.
- R >= 9: deadlock retry, dependency restoration, duplicate suppression, rollback checks and invalidation rules are reproducible.
