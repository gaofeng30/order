# Tasks: implement-staff-identity-snapshot-core

## Invalidation and current state

- First uncommitted attempt: `INVALIDATED_PRE_CANDIDATE` for missing mandatory evidence fields/C-T-V-R, non-vertical tests, and PASS receipts misbound to base SHA.
- Second uncommitted replacement: `INVALIDATED_PRE_CANDIDATE` because table tracers grouped independent behaviors, `C=10` lacked independent-contract evidence, and commands were not directly replayable.
- All broad mutation, race, MySQL, build, smoke, static, review, or verification receipts produced before the current implementation/spec/tasks/tests freeze are `INVALIDATED_NOT_CURRENT`. They cannot support a candidate. The granular Red/Green tree receipts below remain historical tracer evidence, not broad Gate evidence.
- The old-base post-freeze static/evidence/mutation-self-test observations, including the isolated single-mutant run, are `INVALIDATED_BY_BASE_ADVANCE`. Authoritative-base Writer behavior Gates were then run against receipt-before-docs staged source snapshot `b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc`.
- First two-axis pre-review is `INVALIDATED_AFTER_FINDING`: Standards=`P1` for missing `phase: refactor` receipts while claiming Refactor/T10; Spec=`0 findings`. It cannot support a candidate. Frozen implementation tree `f5daa28f883317494842a2f236981557366ab4ec` then passed identical focused, determinism, and race Refactor reruns, which required a completely new two-axis pre-review.
- Replacement pre-review restarted from zero on frozen staged tree `b0846d513d8d7ec181231e3c01cf654ff77879d6` and passed Standards=`0 findings`, `0/12 smells`, Spec=`0 findings`, with stable start/end HEAD/tree and no unstaged/untracked paths. It is writer pre-review, not formal exact-candidate review.
- Lifecycle is `CANDIDATE_READY_FOR_EXTERNAL_SHA_HANDOFF`. Writer candidate-ready score is `C9/T10/V8/R9=36`; `V8` means only that the next owned-only commit's externally bound exact SHA has a complete clean-detached package awaiting verifier, never independent PASS. Current machine receipts remain `candidate_sha: NOT_CREATED`; formal exact review, fresh detached verifier, integration, and archive remain pending.
- The 38 immutable Red/Green receipts below were created against historical development base `f3c4efa4cd665652d93d5da76f92d18c4bdc59ac` by `git write-tree` at each exact command boundary and rechecked read-only with `git cat-file -t <sha> = tree`. They prove only the owned pure Module's historical public-seam tracers. Pickup integration writes completely non-overlapping paths, and these receipts do not claim its behavior. No unstaged or merely remembered state is labelled immutable. The 11 additional pairs include 22 receipt bindings; `4b32503d992f34c385ffcd8c917d6d36871eb1bd` is intentionally both the invalid-UTF-8 Extra Green tree and the subsequent NFC Red tree.
- Evidence schema is frozen to `phase: red|green|refactor|writer|verifier|integration` and `exit_result: exit-<integer>|PASS|FAIL|BLOCKED_EXTERNAL|N/A`; every receipt uses one enum value only.
- Base schema is phase-aware: existing `red`/`green` receipts require historical base `f3c4efa4cd665652d93d5da76f92d18c4bdc59ac`; future `refactor`/`writer`/`verifier`/`integration` receipts require authoritative candidate comparison base `19ca1e46e106f293070f0cdf820951e31107cba6`.

## Completed granular decision slices

- [x] `SI-01` Primary, visitor, disabled-primary, Extra phone, Extra name, and Extra enabled each completed an independent public-seam Red -> minimum Green.

```yaml
task_id: SI-01
evidence_id: SI-01-primary-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveEnabledPrimaryIsStaffIgnoringName$' -count=1
exit_result: exit-1
expected: absent primary-match behavior produces the named public-seam failure
observed: named test failed before primary matching existed
sanitized_summary: enabled-primary tracer was Red
exact_sha: 5f374729baaf98361307eb82ee76d0194f2d4042
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: does not cover visitor or disabled state
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-01
evidence_id: SI-01-primary-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveEnabledPrimaryIsStaffIgnoringName$' -count=1
exit_result: exit-0
expected: enabled primary exact phone match returns versioned STAFF without using name
observed: named test passed after minimum phone-match behavior
sanitized_summary: enabled-primary tracer was Green
exact_sha: 433490d7aef07528e3a1e27efc1616b38c096d71
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: Green intentionally precedes visitor and enabled enforcement
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-01
evidence_id: SI-01-visitor-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveNoMatchIsVersionedVisitor$' -count=1
exit_result: exit-1
expected: no-match behavior fails until VISITOR preserves version
observed: named test received zero Snapshot
sanitized_summary: versioned-visitor tracer was Red
exact_sha: 0c90113906648d693b94e9eeebf8139200caf63d
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: does not cover disabled entries
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-01
evidence_id: SI-01-visitor-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveNoMatchIsVersionedVisitor$' -count=1
exit_result: exit-0
expected: no match returns VISITOR with whitelist version
observed: named test passed after minimum fallback
sanitized_summary: versioned-visitor tracer was Green
exact_sha: d0ad2e2fa64be6751816c8b76c27f9db0d4a458e
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: Green intentionally precedes disabled enforcement
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-01
evidence_id: SI-01-disabled-primary-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveDisabledPrimaryIsVisitor$' -count=1
exit_result: exit-1
expected: phone-only primary implementation incorrectly authorizes disabled record
observed: named test failed with STAFF
sanitized_summary: disabled-primary tracer was Red
exact_sha: 18147afa53cf2c9a72b3580e95973f2845e15bea
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: does not cover Extra fallback
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-01
evidence_id: SI-01-disabled-primary-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveDisabledPrimaryIsVisitor$' -count=1
exit_result: exit-0
expected: primary match additionally requires Enabled
observed: named test passed after enabled check
sanitized_summary: disabled-primary tracer was Green
exact_sha: 4cdcd566f5f8a330b108591b0cf9229feedd06a6
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: Green intentionally precedes Extra behavior
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-01
evidence_id: SI-01-extra-phone-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveDistinctExtraPhoneCanMatchAfterDisabledPrimary$' -count=1
exit_result: exit-1
expected: primary-only implementation cannot authorize distinct Extra phone
observed: named test returned VISITOR
sanitized_summary: distinct Extra-phone tracer was Red
exact_sha: 7aa3868ed32485568b556f70a7eae81015290cf8
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: does not yet enforce Extra name or enabled state
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-01
evidence_id: SI-01-extra-phone-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveDistinctExtraPhoneCanMatchAfterDisabledPrimary$' -count=1
exit_result: exit-0
expected: distinct Extra phone may match after disabled primary
observed: named test passed after minimum Extra-phone lookup
sanitized_summary: distinct Extra-phone tracer was Green
exact_sha: 37bb11b1bc003df8f9dbedd12a4ad91f72869ec6
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: Green intentionally precedes name and enabled enforcement
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-01
evidence_id: SI-01-extra-name-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveExtraNameMismatchIsVisitor$' -count=1
exit_result: exit-1
expected: Extra phone-only implementation incorrectly grants mismatched name
observed: named test returned STAFF
sanitized_summary: Extra-name tracer was Red
exact_sha: 5e0c8fbdaca96c1c5e18f4f706110aa79cf8e98b
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: does not yet enforce enabled state or normalization
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-01
evidence_id: SI-01-extra-name-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveExtraNameMismatchIsVisitor$' -count=1
exit_result: exit-0
expected: Extra requires raw exact name at this slice
observed: named mismatch test passed after name equality
sanitized_summary: Extra-name tracer was Green
exact_sha: 93afb9fe536f1604ed35a94f8bafce14a84ddba3
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: Green intentionally precedes enabled state and normalization
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-01
evidence_id: SI-01-extra-enabled-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveDisabledExtraIsVisitor$' -count=1
exit_result: exit-1
expected: phone/name-only Extra implementation incorrectly authorizes disabled record
observed: named test returned STAFF
sanitized_summary: Extra-enabled tracer was Red
exact_sha: dc2dcd5f1e7851475f8e775374739d81fa5d2759
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: does not cover normalized equality
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-01
evidence_id: SI-01-extra-enabled-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveDisabledExtraIsVisitor$' -count=1
exit_result: exit-0
expected: Extra match additionally requires Enabled
observed: named test passed after enabled check
sanitized_summary: Extra-enabled tracer was Green
exact_sha: d50f40b6620a54d2dca4dfe344f261aba3911f14
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: Green intentionally precedes normalization and validation
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

## Name normalization slices

- [x] `SI-02` Width, post-fold U+0020 deletion, and NFC composition each completed a separate public-seam Red -> minimum Green. The not-NFKC negative is a post-Green mutation sensitivity check, not an earlier implementation Red.

```yaml
task_id: SI-02
evidence_id: SI-02-width-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveFoldsNameWidth$' -count=1
exit_result: exit-1
expected: raw equality fails fullwidth Latin against ASCII
observed: named test returned VISITOR
sanitized_summary: width-fold tracer was Red
exact_sha: b396f1ebddeb184f801ec93bcbab55f2e5c31b92
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: does not cover spaces or NFC
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-02
evidence_id: SI-02-width-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveFoldsNameWidth$' -count=1
exit_result: exit-0
expected: width.Fold makes fullwidth Latin equal ASCII
observed: named test passed after width.Fold
sanitized_summary: width-fold tracer was Green
exact_sha: b8a5abcf9066bcb62e3044f002e3c7a42e4fbcb9
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: Green intentionally precedes spaces and NFC
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-02
evidence_id: SI-02-space-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveDeletesPostFoldASCIISpaces$' -count=1
exit_result: exit-1
expected: width-only implementation retains U+0020 produced by U+3000 and literal U+0020
observed: named test returned VISITOR
sanitized_summary: post-fold U+0020 deletion tracer was Red
exact_sha: 760e5137cffe60eac81b0d4ed5ce99d49238c35b
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: does not cover NFC
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-02
evidence_id: SI-02-space-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveDeletesPostFoldASCIISpaces$' -count=1
exit_result: exit-0
expected: remove only U+0020 after width folding
observed: named test passed after post-fold U+0020 deletion
sanitized_summary: post-fold U+0020 deletion tracer was Green
exact_sha: 4214c39b24b90ccc45066611cc6f4ac9cd86881f
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: Green intentionally precedes NFC and not-NFKC
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

NFC's immutable Red/Green pair is recorded with the validation pairs below. The not-NFKC contract is exercised only by the final-source reversible `norm.NFC -> norm.NFKC` mutation; it is not retroactively labelled an implementation Red.

## Completed validation and NFC slices

- [x] `SI-03` Primary, Extra, whitelist version, entry phone/name, unrelated evidence, duplicate evidence, typed error, and zero-Snapshot fail-closed behavior completed independent Red -> minimum Green tracers. NFC is accounted to `SI-02`; all other pairs in this section are `SI-03`.
- Exact pair order: invalid primary; invalid Extra phone; empty normalized Extra name; invalid UTF-8 Extra name; NFC; zero version; invalid entry phone; empty normalized entry name; invalid UTF-8 entry name; unrelated entry; duplicate canonical phone.

```yaml
task_id: SI-03
evidence_id: SI-03-invalid-primary-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsInvalidPrimaryPhone$' -count=1
exit_result: exit-1
expected: behavior is absent and the named public-seam contract fails
observed: named test exited 1 because invalid primary returned a versioned VISITOR instead of zero Snapshot
sanitized_summary: invalid-primary tracer was Red
exact_sha: a0765bd6e58cb06e1d417ac36749b89eaa9a7fb3
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-invalid-primary-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsInvalidPrimaryPhone$' -count=1
exit_result: exit-0
expected: invalid primary returns zero Snapshot and typed redacted primary-phone error
observed: named test exited 0 after the minimum behavior
sanitized_summary: invalid-primary tracer was Green
exact_sha: cb064777fe8415bc04fadec0a5ec6e1babef70d7
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-invalid-extra-phone-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsInvalidExtraPhone$' -count=1
exit_result: exit-1
expected: behavior is absent and the named public-seam contract fails
observed: named test exited 1 because invalid Extra phone was treated as VISITOR
sanitized_summary: invalid-extra-phone tracer was Red
exact_sha: 0bbcb48a1c7e5e24c3d322c28fdca807d14e51c5
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-invalid-extra-phone-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsInvalidExtraPhone$' -count=1
exit_result: exit-0
expected: invalid Extra phone returns zero Snapshot and typed redacted Extra error
observed: named test exited 0 after the minimum behavior
sanitized_summary: invalid-extra-phone tracer was Green
exact_sha: 90114ce6daec9a27a123d03d82f2348eea33b02e
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-empty-extra-name-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsExtraNameEmptyAfterNormalization$' -count=1
exit_result: exit-1
expected: behavior is absent and the named public-seam contract fails
observed: named test exited 1 because the empty normalized Extra name was treated as VISITOR
sanitized_summary: empty-extra-name tracer was Red
exact_sha: c40f4326b83f4cbbaefa3dbec06fb403c46775b8
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-empty-extra-name-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsExtraNameEmptyAfterNormalization$' -count=1
exit_result: exit-0
expected: empty normalized Extra name returns zero Snapshot and typed redacted Extra error
observed: named test exited 0 after the minimum behavior
sanitized_summary: empty-extra-name tracer was Green
exact_sha: 47ab454d82b29b9cc7b044289c045bbbd5d3a315
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-invalid-utf8-extra-name-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsInvalidUTF8ExtraName$' -count=1
exit_result: exit-1
expected: behavior is absent and the named public-seam contract fails
observed: named test exited 1 before invalid UTF-8 Extra names were rejected
sanitized_summary: invalid-utf8-extra-name tracer was Red
exact_sha: 1823b595e37529ad54f8305f349baee2cb01f84d
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-invalid-utf8-extra-name-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsInvalidUTF8ExtraName$' -count=1
exit_result: exit-0
expected: invalid UTF-8 Extra name returns zero Snapshot and typed redacted Extra error
observed: named test exited 0 after the minimum behavior
sanitized_summary: invalid-utf8-extra-name tracer was Green
exact_sha: 4b32503d992f34c385ffcd8c917d6d36871eb1bd
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-02
evidence_id: SI-02-nfc-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveComposesNamesWithNFC$' -count=1
exit_result: exit-1
expected: behavior is absent and the named public-seam contract fails
observed: named test exited 1 and returned VISITOR
sanitized_summary: nfc tracer was Red
exact_sha: 4b32503d992f34c385ffcd8c917d6d36871eb1bd
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-02
evidence_id: SI-02-nfc-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveComposesNamesWithNFC$' -count=1
exit_result: exit-0
expected: width folding followed by NFC makes canonically equivalent names match
observed: named test exited 0 after the minimum behavior
sanitized_summary: nfc tracer was Green
exact_sha: f515426cea6303d250b4ea2186d43b764cd4c9e6
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-zero-version-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsZeroWhitelistVersion$' -count=1
exit_result: exit-1
expected: behavior is absent and the named public-seam contract fails
observed: named test exited 1 before zero whitelist version was rejected
sanitized_summary: zero-version tracer was Red
exact_sha: b4288427657324d4b400c96fabd207025992d62e
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-zero-version-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsZeroWhitelistVersion$' -count=1
exit_result: exit-0
expected: zero version returns zero Snapshot and typed redacted whitelist error
observed: named test exited 0 after the minimum behavior
sanitized_summary: zero-version tracer was Green
exact_sha: 1f08d9fccd5ce3426139463d7d208a254ad732ab
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-invalid-entry-phone-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsWhitelistEntryWithInvalidPhone$' -count=1
exit_result: exit-1
expected: behavior is absent and the named public-seam contract fails
observed: named test exited 1 before invalid entry phone was rejected
sanitized_summary: invalid-entry-phone tracer was Red
exact_sha: d2e10b8a566b07f49fe59755ccf0e14bf50c23e2
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-invalid-entry-phone-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsWhitelistEntryWithInvalidPhone$' -count=1
exit_result: exit-0
expected: invalid entry phone returns zero Snapshot and typed redacted whitelist error
observed: named test exited 0 after the minimum behavior
sanitized_summary: invalid-entry-phone tracer was Green
exact_sha: 50f4212add1aac3a0a2a33d620a3250b62b49887
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-empty-entry-name-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsWhitelistEntryWithEmptyNormalizedName$' -count=1
exit_result: exit-1
expected: behavior is absent and the named public-seam contract fails
observed: named test exited 1 because malformed entry evidence incorrectly authorized STAFF
sanitized_summary: empty-entry-name tracer was Red
exact_sha: 3361e351050eb85f813fc2be6c0cde637713b1f5
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-empty-entry-name-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsWhitelistEntryWithEmptyNormalizedName$' -count=1
exit_result: exit-0
expected: empty normalized entry name returns zero Snapshot and typed redacted whitelist error
observed: named test exited 0 after the minimum behavior
sanitized_summary: empty-entry-name tracer was Green
exact_sha: cccf9e653cf550acfb61c1887477e6fee5142019
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-invalid-utf8-entry-name-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsWhitelistEntryWithInvalidUTF8Name$' -count=1
exit_result: exit-1
expected: behavior is absent and the named public-seam contract fails
observed: named test exited 1 because malformed entry evidence incorrectly authorized STAFF
sanitized_summary: invalid-utf8-entry-name tracer was Red
exact_sha: 998aebcbd556c95f7ce70f68ca807dd9b39ca994
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-invalid-utf8-entry-name-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsWhitelistEntryWithInvalidUTF8Name$' -count=1
exit_result: exit-0
expected: invalid UTF-8 entry name returns zero Snapshot and typed redacted whitelist error
observed: named test exited 0 after the minimum behavior
sanitized_summary: invalid-utf8-entry-name tracer was Green
exact_sha: e09cb5930cdd77e1cb2e47b021961f543e3cddc1
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-unrelated-entry-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsUnrelatedWhitelistEntry$' -count=1
exit_result: exit-1
expected: behavior is absent and the named public-seam contract fails
observed: named test exited 1 before unrelated entry evidence was rejected
sanitized_summary: unrelated-entry tracer was Red
exact_sha: b276ee3d6b16531e6348be5400ddf9f50810c057
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-unrelated-entry-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsUnrelatedWhitelistEntry$' -count=1
exit_result: exit-0
expected: unrelated entry returns zero Snapshot and typed redacted whitelist error
observed: named test exited 0 after the minimum behavior
sanitized_summary: unrelated-entry tracer was Green
exact_sha: 16a31a066842944c5c409fbef074e265b3ea8e01
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-duplicate-entry-phone-red
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: red
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsDuplicateWhitelistPhone$' -count=1
exit_result: exit-1
expected: behavior is absent and the named public-seam contract fails
observed: named test exited 1 because first-match behavior incorrectly authorized STAFF
sanitized_summary: duplicate-entry-phone tracer was Red
exact_sha: 64fa1f31f2fdcf45edde1dc014adc558bc672d22
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

```yaml
task_id: SI-03
evidence_id: SI-03-duplicate-entry-phone-green
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: f3c4efa4cd665652d93d5da76f92d18c4bdc59ac
candidate_sha: NOT_CREATED
phase: green
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveRejectsDuplicateWhitelistPhone$' -count=1
exit_result: exit-0
expected: duplicate canonical phone returns zero Snapshot and typed redacted whitelist error
observed: named test exited 0 after the minimum behavior
sanitized_summary: duplicate-entry-phone tracer was Green
exact_sha: 9cd14832b7f1998d7c4fa5b0296f2947458e6713
exact_object_type: tree
artifact_or_environment: immutable writer Git tree
unverified_boundary: named tracer only; broad Writer Gates remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect tree and replay exact command
```

## Post-implementation regression and mutation anchors

- `TestResolveRejectsInvalidPrimaryBeforeLowerPriorityEvidence`, `TestResolveRejectsInvalidExtraBeforeWhitelistEvidence`, `TestResolveDoesNotMutateInputAndIsDeterministic`, and `TestResolveSamePrimaryExtraDoesNotCreateFallback` were added after the business implementation was Green. They are regression/mutation anchors, not historical implementation Red evidence.
- `INVALIDATED_BY_BASE_ADVANCE` historical observation: `bash .scratch/implement-staff-identity-snapshot-core/verify-mutation-gate.sh self-test` passed against the old base. Missing-source and duplicate-source injections were rejected with `79`; simulated Go exit `2` was rejected with `82`; `real_mutations=NOT_RUN`.
- `INVALIDATED_BY_BASE_ADVANCE` historical observation: `bash .scratch/implement-staff-identity-snapshot-core/verify-mutations.sh run-one extra_phone_only_authorizes` killed that single mutant with test exit `1` and named marker `TestResolveExtraNameMismatchIsVisitor` against the old base.
- These invalidated observations earn no current Writer Gate credit. The authoritative-base replacement Writer receipts begin below.

## Authoritative-base Writer receipts

All receipts in this section use authoritative base `19ca1e46e106f293070f0cdf820951e31107cba6` and exact tree `b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc`. That tree is the immutable staged source snapshot captured before these receipt docs were written. The dynamic behavior Gates bind that pre-receipt tree; the docs-only write is followed only by bash/static/expanded-ledger/scoped-diff self-checks, while the future exact-candidate detached verifier reruns every declared dynamic Gate.

```yaml
task_id: SI-04
evidence_id: SI-04-pre-receipt-tree-freeze
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: writer
command_or_action: git cat-file -t b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc and git diff --name-only 19ca1e46e106f293070f0cdf820951e31107cba6 b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
exit_result: PASS
expected: the pre-receipt staged snapshot is a tree containing only the 14 owned changes against the authoritative base
observed: tree object exists and its authoritative-base diff is exactly the 14 owned paths
sanitized_summary: pre-receipt source freeze is bound to an immutable tree
exact_sha: b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
exact_object_type: tree
artifact_or_environment: immutable pre-receipt staged source snapshot b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
unverified_boundary: dynamic result binds the receipt-before-docs staged source tree; exact candidate review and detached full rerun remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the immutable tree and replay the exact command or action
```

```yaml
task_id: SI-05
evidence_id: SI-05-static-post-freeze
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: writer
command_or_action: bash .scratch/implement-staff-identity-snapshot-core/verify-static.sh post-freeze
exit_result: PASS
expected: authoritative-base owned static Gate sees the complete staged freeze and no unstaged or untracked path
observed: STATIC_GATE PASS with owned_files=14 staged=14 unstaged=0 untracked=0, exact go.mod promotion and unchanged go.sum
sanitized_summary: authoritative-base post-freeze static Gate passed
exact_sha: b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
exact_object_type: tree
artifact_or_environment: pre-receipt staged source snapshot b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
unverified_boundary: dynamic result binds the receipt-before-docs staged source tree; exact candidate review and detached full rerun remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the immutable tree and replay the exact command or action
```

```yaml
task_id: SI-06
evidence_id: SI-06-evidence-ledger-38
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: writer
command_or_action: bash .scratch/implement-staff-identity-snapshot-core/verify-evidence.sh
exit_result: PASS
expected: all historical granular receipts satisfy object, enum and phase-aware base rules before writer receipts are appended
observed: EVIDENCE_LEDGER PASS with records=38 and completed_tasks=3
sanitized_summary: pre-receipt evidence ledger passed with 38 historical records
exact_sha: b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
exact_object_type: tree
artifact_or_environment: pre-receipt staged source snapshot b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
unverified_boundary: dynamic result binds the receipt-before-docs staged source tree; exact candidate review and detached full rerun remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the immutable tree and replay the exact command or action
```

```yaml
task_id: SI-06
evidence_id: SI-06-focused-package
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: writer
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -count=1
exit_result: PASS
expected: the complete public-seam package contract passes
observed: focused staffidentity package tests passed
sanitized_summary: focused package Gate passed
exact_sha: b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
exact_object_type: tree
artifact_or_environment: pre-receipt staged source snapshot b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc with local Go 1.26.5
unverified_boundary: dynamic result binds the receipt-before-docs staged source tree; exact candidate review and detached full rerun remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the immutable tree and replay the exact command or action
```

```yaml
task_id: SI-06
evidence_id: SI-06-determinism-count-100
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: writer
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveDoesNotMutateInputAndIsDeterministic$' -count=100
exit_result: PASS
expected: the named immutability and determinism contract remains stable for 100 runs
observed: named determinism test passed with count=100
sanitized_summary: repeated determinism Gate passed
exact_sha: b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
exact_object_type: tree
artifact_or_environment: pre-receipt staged source snapshot b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc with local Go 1.26.5
unverified_boundary: dynamic result binds the receipt-before-docs staged source tree; exact candidate review and detached full rerun remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the immutable tree and replay the exact command or action
```

```yaml
task_id: SI-06
evidence_id: SI-06-race-package-count-20
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: writer
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/internal/staffidentity -count=20
exit_result: PASS
expected: the complete package remains race-clean across 20 runs
observed: staffidentity package race run passed with count=20
sanitized_summary: repeated package race Gate passed
exact_sha: b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
exact_object_type: tree
artifact_or_environment: pre-receipt staged source snapshot b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc with local Go 1.26.5 race runtime
unverified_boundary: dynamic result binds the receipt-before-docs staged source tree; exact candidate review and detached full rerun remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the immutable tree and replay the exact command or action
```

```yaml
task_id: SI-06
evidence_id: SI-06-mutation-first-uniqueness-fail
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: writer
command_or_action: in a temporary copy of b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc replace only the input_is_mutated replacement map literal with the prior make-map expression, then run bash .scratch/implement-staff-identity-snapshot-core/verify-mutation-gate.sh
exit_result: FAIL
expected: the infrastructure uniqueness shield rejects a replacement that still contains the original source bytes and the run is not counted PASS
observed: 16 mutants were killed, then input_is_mutated failed uniqueness with old=1 new_before=0 new_after=1; no mutation Gate PASS was credited
sanitized_summary: first full mutation run correctly stopped on uniqueness-shield failure
exact_sha: b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
exact_object_type: tree
artifact_or_environment: reversible temporary reproduction from final pre-receipt staged source snapshot b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
unverified_boundary: dynamic result binds the receipt-before-docs staged source tree; exact candidate review and detached full rerun remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the immutable tree and replay the exact command or action
```

```yaml
task_id: SI-06
evidence_id: SI-06-mutation-self-test-after-fix
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: writer
command_or_action: bash .scratch/implement-staff-identity-snapshot-core/verify-mutation-gate.sh self-test
exit_result: PASS
expected: the three infrastructure rejection shields pass after the minimum Gate-only replacement correction
observed: missing-source and duplicate-source were rejected with 79, Go exit 2 was rejected with 82, and real_mutations=NOT_RUN
sanitized_summary: mutation infrastructure self-test passed after minimum Gate-only fix
exact_sha: b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
exact_object_type: tree
artifact_or_environment: pre-receipt staged source snapshot b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
unverified_boundary: dynamic result binds the receipt-before-docs staged source tree; exact candidate review and detached full rerun remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the immutable tree and replay the exact command or action
```

```yaml
task_id: SI-06
evidence_id: SI-06-mutation-input-single-after-fix
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: writer
command_or_action: bash .scratch/implement-staff-identity-snapshot-core/verify-mutations.sh run-one input_is_mutated
exit_result: PASS
expected: the corrected input mutation is unique and reaches the named immutability assertion
observed: input_is_mutated was killed with test exit 1 and named TestResolveDoesNotMutateInputAndIsDeterministic FAIL marker
sanitized_summary: corrected input mutation single-mutant check passed
exact_sha: b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
exact_object_type: tree
artifact_or_environment: pre-receipt staged source snapshot b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
unverified_boundary: dynamic result binds the receipt-before-docs staged source tree; exact candidate review and detached full rerun remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the immutable tree and replay the exact command or action
```

```yaml
task_id: SI-06
evidence_id: SI-06-mutation-full-17
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: writer
command_or_action: bash .scratch/implement-staff-identity-snapshot-core/verify-mutation-gate.sh
exit_result: PASS
expected: all infrastructure self-tests and all 17 frozen real mutants satisfy the uniqueness and named-marker shield
observed: three infrastructure self-tests passed and all 17 of 17 real mutants were killed with exact exit 1 and their named FAIL markers
sanitized_summary: complete mutation shield passed 17 of 17
exact_sha: b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
exact_object_type: tree
artifact_or_environment: pre-receipt staged source snapshot b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
unverified_boundary: dynamic result binds the receipt-before-docs staged source tree; exact candidate review and detached full rerun remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the immutable tree and replay the exact command or action
```

```yaml
task_id: SI-06
evidence_id: SI-06-fresh-mysql-full-api
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: writer
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 bash .scratch/repair-version-scoped-mysql-migration-fixtures-v13/verify-mysql.sh full
exit_result: PASS
expected: fresh pinned loopback MySQL runs all services/api normal and race suites and removes its temporary assets
observed: mysql:8.0.46-oraclelinux9 full services/api normal and race suites passed; temporary container and credential were cleaned
sanitized_summary: fresh MySQL adjacent full API Gate passed and cleaned up
exact_sha: b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
exact_object_type: tree
artifact_or_environment: disposable loopback-only mysql:8.0.46-oraclelinux9 environment
unverified_boundary: dynamic result binds the receipt-before-docs staged source tree; exact candidate review and detached full rerun remain pending
external_asset:
  owner: writer/verifier
  missing: N/A
  recovery: rerun the same verify-mysql.sh full command with a fresh pinned container
```

```yaml
task_id: SI-06
evidence_id: SI-06-vet-all-api
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: writer
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go vet ./services/api/...
exit_result: PASS
expected: all services/api packages pass vet
observed: go vet for all services/api packages passed
sanitized_summary: all-API vet Gate passed
exact_sha: b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
exact_object_type: tree
artifact_or_environment: pre-receipt staged source snapshot b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc with local Go 1.26.5
unverified_boundary: dynamic result binds the receipt-before-docs staged source tree; exact candidate review and detached full rerun remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the immutable tree and replay the exact command or action
```

```yaml
task_id: SI-06
evidence_id: SI-06-build-all-api
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: writer
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go build ./services/api/...
exit_result: PASS
expected: all services/api packages build
observed: go build for all services/api packages passed
sanitized_summary: all-API build Gate passed
exact_sha: b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
exact_object_type: tree
artifact_or_environment: pre-receipt staged source snapshot b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc with local Go 1.26.5
unverified_boundary: dynamic result binds the receipt-before-docs staged source tree; exact candidate review and detached full rerun remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the immutable tree and replay the exact command or action
```

```yaml
task_id: SI-06
evidence_id: SI-06-controlled-api-smoke
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: writer
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/smoke.sh
exit_result: PASS
expected: the controlled order API smoke completes without leaking its canaries
observed: controlled API smoke passed
sanitized_summary: controlled API smoke Gate passed
exact_sha: b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
exact_object_type: tree
artifact_or_environment: pre-receipt staged source snapshot b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc controlled local smoke environment
unverified_boundary: dynamic result binds the receipt-before-docs staged source tree; exact candidate review and detached full rerun remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the immutable tree and replay the exact command or action
```

```yaml
task_id: SI-06
evidence_id: SI-06-final-hygiene-cleanup
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: writer
command_or_action: rerun verify-static.sh post-freeze and verify-evidence.sh, run scoped diff checks, inspect staged and unstaged and untracked path sets, and confirm verify-mysql.sh temporary container cleanup
exit_result: PASS
expected: final authoritative-base static and 38-record ledger pass with owned diff, zero unstaged and untracked paths, and no temporary MySQL residue
observed: final static and ledger passed; owned diff remained 14 staged paths with zero unstaged or untracked paths; diff and temporary-container cleanup checks passed
sanitized_summary: final writer hygiene and cleanup audit passed
exact_sha: b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
exact_object_type: tree
artifact_or_environment: pre-receipt staged source snapshot b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc after all dynamic Writer Gates
unverified_boundary: dynamic result binds the receipt-before-docs staged source tree; exact candidate review and detached full rerun remain pending
external_asset:
  owner: writer/verifier
  missing: N/A
  recovery: rerun the complete Writer Gate from static through cleanup on every new source or candidate tree
```

```yaml
task_id: SI-07
evidence_id: SI-07-quality-score-assessment
receipt_status: INVALIDATED_AFTER_STANDARDS_P1
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: writer
command_or_action: assess C, T and R against docs/quality/change-quality-gates.md section 10 using the frozen contract and current writer receipts
exit_result: N/A
expected: award only evidence-supported writer axes and do not award V or a passing total before an exact candidate and detached verifier
observed: initial C9 T10 R9 assessment lacked a phase-refactor receipt and was invalidated by Standards P1; V remained NOT_EARNED and verdict remained NOT_PASS
sanitized_summary: initial quality assessment was invalidated because no phase-refactor receipt supported the Refactor/T10 claim
exact_sha: b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc
exact_object_type: tree
artifact_or_environment: quality-table assessment over pre-receipt staged source snapshot b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc writer evidence
unverified_boundary: dynamic result binds the receipt-before-docs staged source tree; exact candidate review and detached full rerun remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the immutable tree and replay the exact command or action
```

## Refactor receipts after invalidated first pre-review

The first pre-review Standards=P1/Spec=0 result is `INVALIDATED_AFTER_FINDING` and cannot support a candidate. The following three receipts bind the identical frozen-implementation rerun to staged tree `f5daa28f883317494842a2f236981557366ab4ec`. Adding these docs resolves the ledger gap only; it does not claim a replacement pre-review PASS.

```yaml
task_id: SI-06
evidence_id: SI-06-refactor-focused-package
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: refactor
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -count=1
exit_result: PASS
expected: the frozen implementation passes the identical focused public-seam package contract after Green
observed: focused staffidentity package rerun passed with count=1
sanitized_summary: identical focused Refactor Gate passed
exact_sha: f5daa28f883317494842a2f236981557366ab4ec
exact_object_type: tree
artifact_or_environment: frozen staged implementation tree before Refactor receipt docs
unverified_boundary: first pre-review is invalidated; replacement two-axis pre-review, candidate, formal review and detached verifier remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the immutable tree and replay the exact command
```

```yaml
task_id: SI-06
evidence_id: SI-06-refactor-determinism-count-100
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: refactor
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -run '^TestResolveDoesNotMutateInputAndIsDeterministic$' -count=100
exit_result: PASS
expected: the frozen implementation passes the identical repeated immutability and determinism contract after Green
observed: named determinism rerun passed with count=100
sanitized_summary: identical repeated determinism Refactor Gate passed
exact_sha: f5daa28f883317494842a2f236981557366ab4ec
exact_object_type: tree
artifact_or_environment: frozen staged implementation tree before Refactor receipt docs
unverified_boundary: first pre-review is invalidated; replacement two-axis pre-review, candidate, formal review and detached verifier remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the immutable tree and replay the exact command
```

```yaml
task_id: SI-06
evidence_id: SI-06-refactor-race-package-count-20
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: refactor
command_or_action: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/internal/staffidentity -count=20
exit_result: PASS
expected: the frozen implementation passes the identical repeated package race contract after Green
observed: staffidentity package race rerun passed with count=20
sanitized_summary: identical repeated race Refactor Gate passed
exact_sha: f5daa28f883317494842a2f236981557366ab4ec
exact_object_type: tree
artifact_or_environment: frozen staged implementation tree before Refactor receipt docs with Go race runtime
unverified_boundary: first pre-review is invalidated; replacement two-axis pre-review, candidate, formal review and detached verifier remain pending
external_asset:
  owner: N/A
  missing: N/A
  recovery: inspect the immutable tree and replay the exact command
```

## Replacement writer pre-review and external-SHA handoff

The replacement pre-review below restarted both axes from zero on frozen staged tree `b0846d513d8d7ec181231e3c01cf654ff77879d6`. It is a writer pre-review receipt only. This governance-only delta changes the next tree, so the future committed SHA receives formal exact-candidate review and fresh detached full-Gate verification from zero through immutable external handoff.

```yaml
task_id: SI-08
handoff_task: SI-09
evidence_id: SI-08-replacement-two-axis-pre-review
change: implement-staff-identity-snapshot-core
gate_type: W3
ui_level_target: UI0
ui_level_actual: UI0
base_sha: 19ca1e46e106f293070f0cdf820951e31107cba6
candidate_sha: NOT_CREATED
phase: writer
command_or_action: restart Standards and Spec pre-review from zero against authoritative base and frozen staged tree b0846d513d8d7ec181231e3c01cf654ff77879d6, checking HEAD tree and status at both boundaries
exit_result: PASS
expected: both writer pre-review axes report zero findings with stable HEAD and tree and no unstaged or untracked paths
observed: Standards reported 0 findings and 0 of 12 smells; Spec reported 0 findings; start and end HEAD were 19ca1e46e106f293070f0cdf820951e31107cba6, tree was b0846d513d8d7ec181231e3c01cf654ff77879d6, and unstaged and untracked sets were empty
sanitized_summary: replacement two-axis writer pre-review passed with zero findings on the frozen staged tree
exact_sha: b0846d513d8d7ec181231e3c01cf654ff77879d6
exact_object_type: tree
artifact_or_environment: frozen staged tree before final candidate-ready governance docs delta
unverified_boundary: not formal exact-candidate review and not independent verification; next owned-only commit requires external SHA binding, formal exact review, and fresh clean detached full-Gate verification
external_asset:
  owner: controller for immutable external SHA handoff
  missing: exact candidate SHA does not exist before commit
  recovery: commit the owned-only candidate once, bind its full SHA externally, then run formal review and detached verification from zero
```

## Remaining implementation and lifecycle

- [x] `SI-04` Business implementation/spec/tests/Gate scripts were frozen at pre-receipt staged source tree `b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc`; only the three receipt docs change afterward.
- [x] `SI-05` Authoritative-base owned/static freeze confirmed exactly 14 staged paths with zero unstaged/untracked paths at the pre-receipt tree boundary.
- [x] `SI-06` Complete Writer behavior Gate order passed on the pre-receipt tree, with the initial mutation uniqueness failure retained as FAIL and the corrected shield rerun through 17/17 PASS.
- [x] `SI-07` Writer candidate-ready assessment is `C9/T10/V8/R9=36`. `V8` applies only to the next owned-only commit's externally bound exact SHA with a complete clean-detached package awaiting verifier; it is not independent PASS.
- [x] `SI-08` Replacement Standards/Spec writer pre-review restarted from zero on `b0846d513d8d7ec181231e3c01cf654ff77879d6` and returned dual zero findings. The first Standards=P1/Spec=0 attempt remains invalidated.
- [x] **SI-09** The next owned-only commit containing this governance update forms the candidate. The controller binds its immutable full SHA externally after commit; the commit never self-references or invents a future SHA. This handoff readiness is linked by the `SI-08` receipt's `handoff_task` field.
- [ ] `SI-10` Formal parallel Standards/Spec review and clean detached verifier bind exact candidate; any finding creates a new SHA and restarts both.
- [ ] `SI-11` Report exact candidate and explicit unverified boundary; never integrate or push in this change.
