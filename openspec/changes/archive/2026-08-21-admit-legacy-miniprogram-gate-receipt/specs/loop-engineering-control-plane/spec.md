## MODIFIED Requirements

### Requirement: Archived recovery uses one append-only receipt

After existing candidate verification, integration, and archive Gates, the control plane MUST add exactly one `lifecycle-receipt.md` in the dated archive, keep candidate checkpoint/tasks byte-identical, and store `receipt_head_verification=REQUIRED_DERIVED`. Recovery MUST verify exact Git objects/ancestry, archive path/diff, ownership, candidate artifact digests/tasks, retrospective, unique receipt add, and no later receipt touch.

The sole exception is `enforce-miniprogram-user-regression-gate@f5719e98690d0b1301ed567c8b616e074846b445`, whose immutable checkpoint predates the runtime-section convention. It MUST use exactly one `legacy-lifecycle-receipt.md` and an independently verified exact-target compatibility verifier that pins and reuses the protected v1 control plane, preserves the recorded `IMPLEMENTING`/`none` facts, and passes both old v1 Gates and its separate receipt-head Gate. This exception MUST NOT be generalized or consumed by v1 `--list`.

#### Scenario: Standard receipt history is valid
- **WHEN** a non-exempt change's structural/Git checks and required controlled mechanical replay all pass
- **THEN** recovery reports the immutable standard receipt head and layered evidence result
- **AND** writes neither the derived head nor PASS back to repository evidence

#### Scenario: Exact legacy receipt history is valid
- **WHEN** the sole exempt change's fixed compatibility facts, old v1 Gates, structural/Git checks and controlled mechanical replay all pass at exact `R`
- **THEN** recovery reports the immutable legacy receipt head, original checkpoint facts and exact-only compatibility rule
- **AND** writes neither the derived head nor PASS back to repository evidence

#### Scenario: Receipt is stale or tampered
- **WHEN** the required standard or exact legacy receipt is missing, duplicated, edited later, ancestry-inconsistent, task-inconsistent, structurally ambiguous or supplied to the wrong verifier
- **THEN** recovery returns non-zero `NO-GO`
- **AND** archive presence, recorded attestation or the exact exception name alone does not close recovery
