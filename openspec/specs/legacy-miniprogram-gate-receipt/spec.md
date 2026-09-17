# legacy-miniprogram-gate-receipt Specification

## Purpose
TBD - created by archiving change admit-legacy-miniprogram-gate-receipt. Update Purpose after archive.
## Requirements
### Requirement: One exact pre-convention Mini Program governance receipt is admitted

The repository MUST provide one explicit compatibility verifier for change `enforce-miniprogram-user-regression-gate`, candidate `f5719e98690d0b1301ed567c8b616e074846b445`, archive `b53cb520c4cac0505803a9d1e6dcc1807ac34540`, checkpoint SHA256 `535a53c39b1177a795422acd6cad85ebf9c60ac5a545ff169306b6c674a564e3`, and tasks SHA256 `da0d82d6a66830b19c6375e4cd014b0a0214d7e10a96079c91818c9cf119349e`. It MUST accept only the unique archive-local file `legacy-lifecycle-receipt.md`, MUST require the immutable checkpoint to record exactly `lifecycle=IMPLEMENTING` and `candidate_sha=none`, and MUST identify the result as an exact-only pre-convention interpretation rather than a historical `CANDIDATE` record.

The verifier MUST pin the protected v1 judge, schema, profile runner, binding registry and Mini Program wrapper Git blobs before importing them. It MUST reuse their schema, form, Git ancestry/archive/ownership/task/history and controlled mechanical profile checks; recorded attestation MUST remain untrusted and mechanical replay MUST report actor independence as `NOT_PROVEN_BY_MECHANICAL_REPLAY`.

#### Scenario: Exact legacy receipt derives PASS
- **WHEN** exact receipt-head `R` contains the unique legacy receipt, every fixed SHA/hash/blob/path/task/archive/history fact matches, the old v1 receipt Gates remain PASS, the Mini Program profile returns `MECHANICAL_PASS`, and the worktree is clean detached at `R`
- **THEN** the compatibility verifier returns `receipt_head_verification=PASS_DERIVED` without persisting PASS
- **AND** it reports recorded checkpoint state `IMPLEMENTING`, recorded candidate `none`, and the exact-only compatibility rule ID

#### Scenario: Any compatibility boundary differs
- **WHEN** the change, candidate, archive, checkpoint, task, blob, file name, directory, receipt history, binding, profile output, worktree state or exact-only field differs
- **THEN** verification returns nonzero at the first deterministic error
- **AND** standard receipt text, recorded attestation, archive presence or another PASS cannot compensate

### Requirement: Legacy compatibility does not weaken the standard receipt plane

The compatibility verifier MUST NOT modify the protected v1 judge, schema, runner, profile or binding registries, historical checkpoint/tasks, or standard receipts. The legacy filename MUST remain outside v1 enumeration, while exact-R verification MUST run both the complete existing v1 `--list`/`--chain` Gates and the explicit compatibility verifier. No later or different change MAY reuse the compatibility interpretation.

#### Scenario: Existing receipt plane remains unchanged
- **WHEN** the repair candidate and later exact receipt head are checked
- **THEN** protected control blobs equal exact-B, existing v1 `--list` and `--chain` retain their previous receipt set and results, and all five bindings remain exact
- **AND** the legacy verifier adds only the separately reported exact governance result

#### Scenario: A future change requests legacy interpretation
- **WHEN** any other change, SHA, checkpoint shape or alternate legacy receipt is supplied
- **THEN** the compatibility verifier rejects it
- **AND** the caller must use the standard lifecycle receipt contract or a new independently approved OpenSpec

### Requirement: Repair and receipt closure remain separate exact-SHA stages

The repair candidate MUST NOT add the governance legacy receipt. Two independent clean-detached Agents MUST verify the same exact repair candidate before local integration/archive. The archive MUST preserve candidate tool/test bytes and apply only the complete change move plus declared canonical deltas. Only afterward MAY the integrator add one receipt-only commit, and another independent Agent MUST derive exact receipt-head PASS.

#### Scenario: Receipt is added before repair archive
- **WHEN** the candidate, archive commit, or any commit before two exact-candidate PASS records contains the legacy receipt
- **THEN** repair promotion and governance closure remain `NO-GO`
- **AND** no mechanical output upgrades the stage

#### Scenario: All stages pass in order
- **WHEN** two independent exact-C verifiers PASS, local integration/archive is exact, `R` adds only the legacy receipt, and a different exact-R verifier passes all old and compatibility Gates
- **THEN** the original governance receipt TODO may be closed
- **AND** the result still excludes actor independence, UI2/UI3, real WeChat, push, PR and deployment
