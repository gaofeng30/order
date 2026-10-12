# Shared PG read and report requirements

Status: APPROVED_REQUIREMENTS / IMPLEMENTATION_IN_PROGRESS / FINAL_GATE_NOT_RUN / CANDIDATE_NOT_CREATED。gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN。

## ADDED Requirements

### Requirement: Preserve all existing public ports
The implementation SHALL implement the original billing five methods, adminreport ten methods, audit Search and storestatus Apply through exactly the fifteen named RPCs in design; financial commands SHALL delegate to the existing applications.

#### Scenario: An existing public operation is invoked
- **WHEN** An existing public operation is invoked
- **THEN** Its native validation, authorization, output and error ordering are preserved without new business rules.

### Requirement: Preserve report read stages
Read RPCs SHALL preserve native statement order, scan failures and later-statement visibility without introducing a single joined snapshot or new locks.

#### Scenario: A concurrent writer commits between header and item reads
- **WHEN** A concurrent writer commits between header and item reads
- **THEN** The subsequent statement follows the native visibility and scan-error contract.

### Requirement: Preserve time and financial codecs
The implementation SHALL preserve full uint64, native int64 arithmetic, nullable facts, integer pgx timestamp consumption, billing fixed +08 dates, admin IANA dates and original Local/UTC CSV bytes.

#### Scenario: Boundary dates, large amounts or invalid stored scan facts are consumed
- **WHEN** Boundary dates, large amounts or invalid stored scan facts are consumed
- **THEN** Native outcomes and CSV bytes remain unchanged; no new range limit is introduced.

### Requirement: Keep reconciliation atomic
Reconciliation SHALL retain intrinsic advisory lock, replay priority, original pending projections and late audit clock in one transaction; provider calls SHALL run outside it.

#### Scenario: A mismatch commits or a late audit write fails
- **WHEN** A mismatch commits or a late audit write fails
- **THEN** Mismatch returns after committed facts; a late failure rolls back every projection, and replay does not duplicate evidence.

### Requirement: Preserve store status receipts
Apply SHALL retain current authorization and singleton checks before receipt, per-attempt clock, unchanged receipts and only original retry classes.

#### Scenario: An old key is retried after authority revocation or after lost acknowledgment
- **WHEN** An old key is retried after authority revocation or after lost acknowledgment
- **THEN** Current authorization is enforced and committed receipts prevent duplicate writes.

### Requirement: Preserve explicit response growth
The nine named growing responses in design SHALL use strict single-JSON success decoding without an artificial body cap; the other six and error/auth responses retain their approved bounds.

#### Scenario: A healthy native catalogue or report grows
- **WHEN** A healthy native catalogue or report grows
- **THEN** No rows, fields or CSV pages are truncated and no new business size limit is added.

### Requirement: Enforce dependency and least privilege
Final registry installation and candidate gates SHALL wait for verified Seventh exact-SHA adoption, then validate all 105 fixed bodies and least privilege under the ordinary service UID.

#### Scenario: Parallel reports development uses a temporary fixture
- **WHEN** Parallel reports development uses a temporary fixture
- **THEN** It is identified as development evidence and cannot satisfy final integration or candidate gates.

### Requirement: Require layered verification
The change SHALL complete native Red/Green, unified regression, ordinary-UAT cleanup, writer gates and clean detached exact-SHA verification before integration.

#### Scenario: Only local or synthetic-provider tests pass
- **WHEN** Only local or synthetic-provider tests pass
- **THEN** Real provider delivery, full runtime, UI3 and complete 95-case/25-page UAT remain unverified.
