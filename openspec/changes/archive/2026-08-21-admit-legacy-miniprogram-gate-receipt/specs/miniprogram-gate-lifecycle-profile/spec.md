## MODIFIED Requirements

### Requirement: Receipt closure remains ordered and externally verified

Profile candidate PASS, archive PASS and binding presence MUST NOT close the governance lifecycle receipt. A different clean-detached verifier MUST first execute exact binding-head `B` and report current `MECHANICAL_PASS`. Because this exact governance candidate has an immutable pre-convention checkpoint, only afterward MAY an authorized integrator append exactly one governance `legacy-lifecycle-receipt.md` in a later receipt-only commit `R`; another clean-detached verifier MUST run the protected standard receipt Gates plus the exact-target compatibility schema/Git/profile/chain verification at exact `R` and derive receipt-head PASS without persisting that derived result. No other change may use the legacy file or interpretation.

The receipt MUST record mechanical and receipt-head fields as `REQUIRED_DERIVED`, preserve the exact governance candidate/archive facts, the original checkpoint state `IMPLEMENTING`/candidate `none`, and excluded claims, and MUST NOT assert actor independence. Missing, stale, reordered, dirty, wrong-SHA, same-verifier, later-edited or failed binding/receipt evidence MUST keep business-module allocation `NO-GO`.

#### Scenario: Binding exists but receipt closure is incomplete
- **WHEN** `B` exists but binding-head PASS, later legacy receipt commit or separate receipt-head PASS is missing
- **THEN** governance closure remains `NO-GO`
- **AND** the next Mini Program business module is not allocated

#### Scenario: Binding and receipt heads pass in order
- **WHEN** different clean-detached verifiers pass exact `B` and later exact `R` with all declared protected v1, compatibility, mechanical and chain checks
- **THEN** governance lifecycle receipt closure may be reported complete
- **AND** the next module still captures a new runner base and treats mechanical replay as non-actor evidence
