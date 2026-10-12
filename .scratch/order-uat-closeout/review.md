# Local two-axis review

Fixed point: d29475de47281b17802ecb73e5a329450f850deb. Draft review command: git diff d29475de47281b17802ecb73e5a329450f850deb -- go.mod go.sum services/api openspec/changes/establish-postgresql-uat-foundation; plus task-owned untracked files from inherited-source-manifest.json. Governance snapshot affb06cafeac5a922b7f436837337eeae7e7079a is not a business candidate. Root is sole writer; both reviewers performed read-only review.

## Standards

Initial P2: product and launch-layer commands acquire a second shared-pool connection inside their transaction for image confirmations. Both are repaired by immutable receipt prefetch before BeginTx, preserving owner authorization, catalog receipt replay/error order and unconfirmed-image rejection. True single-connection PG Red/Green receipts are retained.

Follow-up due cutoff concern is repaired: claimQuery obtains leaseNow after the row lock, preserves parameter now for batch eligibility, and uses leaseNow for expiry and update timestamps. Public RunDue regression demonstrates batch rescheduling and next-batch recovery.

Full initial refund failure has concrete five PG connection-cap rejections during its execution window. No deterministic lock cycle or nested pool access was found in the eight same-key Request→claimCreate→finishCreate path. The harness limits package concurrency while preserving eight business goroutines and PG max_connections25. Original failed gate is retained; subsequent full gate must supply Green.

Reviewer did not execute tests or mutate state. Initial/affected local review only, not exact-SHA independent evidence.

## Spec

Final Spec review reports no new defect; original stale lease issue is closed. No contract weakening or scope creep found. claimQuery reads leaseNow after the row lock and retains batch cutoff for due eligibility. Owner/version fencing, expired-result rejection, next-batch recovery and paid-query idempotent materialization remain.

Public RunDue regression covers rescheduling after the cutoff, next-batch recovery and absence of duplicate facts. PG PATH plus GOFLAGS=-mod=readonly -p=1 adjust local tool discovery, module write prevention and package scheduling; eight business goroutines remain.

Reviewer verified four key PG tests Green and the current ordinary receipt1441PASS/4opt-inSKIP/0FAIL. Race/vet/build are root's separate evidence. This is read-only uncommitted-source review, not exact-SHA independent verification or cloud UAT acceptance.

Standards: original1finding resolved, follow-up cutoff concern resolved, no current unresolved finding. Spec: original1finding resolved, no current unresolved finding. Root verification receipts separately show current ordinary/race1441PASS/4SKIP each, vet/build/smoke PASS.
