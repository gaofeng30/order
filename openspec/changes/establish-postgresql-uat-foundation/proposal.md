# Establish PostgreSQL UAT foundation

Status: UAT_SCOPE_CONFIRMED / IMPLEMENTING. The latest user instruction confirms full UAT workflows, adversarial review, deployment and online smoke; the user selects Sol to execute the prepared plan. No repeated scope approval is required. UAT pay-as-you-go was actually disabled and verified. This is not a Writer Gate PASS.

## Why
The existing API persists to MySQL while the authorized UAT has CloudBase PostgreSQL. A driver-only replacement cannot preserve transactions, migrations, recovery and trusted payment observations.

## What Changes
- Port the existing persistence, forward migration/history/readiness and transactional invariants to PostgreSQL, without a dual-write or runtime SQL translator.
- Reuse the three necessary existing fixes after local revalidation; preserve the public business contract and API.
- Establish isolated local PostgreSQL fault/regression evidence before any cloud deployment.

## Gate and Ownership
- gate_type: W3; ui_level target/actual: UI0 for this backend-only change.
- owner/controller: closeout executor 01a11042-3601-7850-a38f-974660ec7462; branch codex/order-uat-closeout; worktree /Users/vivix/.codex/worktrees/order-uat-closeout/order. The former writer is preserved read-only at /Users/vivix/.codex/worktrees/order-client-cloud-onboarding-checklist; all 457 owned source files matched on transfer.
- base_sha: d29475de47281b17802ecb73e5a329450f850deb; governance-only snapshot/head: affb06cafeac5a922b7f436837337eeae7e7079a; candidate_sha: NOT_CREATED.
- owned paths: services/api/internal and cmd persistence/config/migration tests, services/api/migrations, go.mod/go.sum, new scoped local test/deployment scripts, this change, .scratch/order-uat-closeout. Historical .scratch/order-pg-uat-mvp evidence remains read-only in the former writer. UI apps and the eight existing governance files are not owned.
- read-only contracts: CONTEXT.md, openspec/specs, existing user-confirmed business rules, current AGENTS/quality gates.
- dependencies: none for local implementation; UAT requires cost/access/runtime identity gates. main integration is not authorized.
- non-goals: production operations during the current UAT stage, remote publication, new business features, quantity reservations, real financial tests without specific approval.
- external assets: local PG16.13 is available (not PG17 target evidence); PG17 UAT/native identity and real UI/financial assets remain BLOCKED_EXTERNAL at their dependent stages.

## Planning checkpoint
The confirmed business scope remains unchanged. Historical local native44/session/schema/transport/storage/financial/HTTP/worker evidence is indexed in the former writer's `.scratch/order-pg-uat-mvp/execution-checkpoints.md`; it is not current-candidate proof. Runtime is PostgreSQL only; immutable historical MySQL SQL and test-only algorithm assertions are retained. The protected research originals remain untouched in the former writer and were not copied into this dedicated worktree. The 2026-10-06 local two-axis review found one Standards and one Spec defect; both have native-PG Red/Green evidence in `.scratch/order-uat-closeout`, with final affected review and current local Writer checks now passed; candidate formation and exactSHA detached verification remain pending. Candidate, independent, deployment and full95-case status remain unproven. Cloud access is now working through the authorized official MCP; native PG/TLS/Go identity, cost, UI and real financial acceptance remain dependent gates. Production architecture conflicts require their own scoped UAT delta before dependent deployment.
