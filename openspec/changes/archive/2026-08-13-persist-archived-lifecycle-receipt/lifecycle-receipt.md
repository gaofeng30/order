# Archived lifecycle receipt

## Receipt fields
receipt_schema: archived-lifecycle-receipt/v1
change_name: persist-archived-lifecycle-receipt
archive_path: openspec/changes/archive/2026-08-13-persist-archived-lifecycle-receipt
lifecycle_state: ARCHIVED
expected_historical_verdict: VERIFIED
failure_fingerprint: none
subsequent_gates: PASS
superseded_by: none
supersedes: none
repo_base_sha: 510a84baddfb5b61200c159fd2041b7c512a92db
runner_skill_git_blob: 211498668419dcb66d11f5bfacf7457ed385aa05
runner_skill_sha256: c347f88e3593ed80cea79e4b2d4885bb6d92c677d1e1012c12171fab34482d14
runner_version: unversioned
candidate_sha: d0b70a077bcaa64c401837eb0e9b6f27035210a0
integrated_sha: d0b70a077bcaa64c401837eb0e9b6f27035210a0
integration_validation: PASS
archive_sha: 44aea3b3f892acf8a6e0537f65f28f33de22cf2f
archive_validation: PASS
owned_paths_json: ["openspec/changes/persist-archived-lifecycle-receipt/**",".agents/skills/order-run-loop/SKILL.md",".agents/skills/order-run-loop/references/self-evolution.md","tools/lifecycle-receipts/**","openspec/changes/archive/2026-08-13-enable-run-loop-self-evolution/lifecycle-receipt.md","openspec/changes/archive/2026-08-13-connect-miniprogram-menu-catalog/lifecycle-receipt.md","openspec/changes/archive/2026-08-13-supersede-miniprogram-catalog-evidence/lifecycle-receipt.md"]
profile_id: lifecycle-receipt-control-v1
profile_binding_sha256: ca2fd9eccfc1eee0d597fc0950448d400d07e4fdc726dde4aed58f5a2b090001
recorded_attestation_trust: UNTRUSTED_FOR_MECHANICAL_PASS
recorded_attestation_json: {"actor_independence":"NOT_PROVEN_BY_MECHANICAL_REPLAY","binding_head_sha":"4d7e26f7a6b6930968c1d595e93bfefc6dffdd7c","claimed_verdict":"PASS","host_id":"local","source":"current exact-B G0.2 verifier final","thread_id":"019fff67-7e97-71a3-ad8c-5596a69a59da"}
mechanical_verification: REQUIRED_DERIVED
product_trees_json: {}
candidate_checkpoint_sha256: b6020ed8663303bc188c05b247475df1b0bca29634bdd70f52f30974eea44037
candidate_tasks_sha256: 8322b0102526f0f4136cb22c30c9e0c700c3e12a25ce90f34b2b9a163fb85156
candidate_open_tasks_json: ["5.1","5.2","5.3","6.1","6.2","6.3","6.4","6.5","6.6"]
post_candidate_tasks_json: {"5.1":"PASS","5.2":"PASS","5.3":"PASS","6.1":"PASS","6.2":"PASS","6.3":"PASS","6.4":"PASS","6.5":"PASS","6.6":"REQUIRED_DERIVED"}
receipt_head_verification: REQUIRED_DERIVED
observation_counts_json: {"candidate":0,"checker":2,"environment":2,"external":0}
draft_screen_decisions_json: ["DRAFT admission was not approval, candidate, verification, receipt, integration, archive or promotion evidence"]
retrospective_json: ["recorded attestation and mechanical replay do not prove actor or session independence","controlled exact-SHA replay preserves the historical FAILED verdict and layered recovery"]
unverified_boundaries_json: ["mechanical replay does not prove actor or session independence","UI2 and UI3 and real order or payment remain outside this evidence"]

## Retrospective
This receipt records the current exact-B G0.2 handoff only as untrusted audit history. Mechanical and receipt-head results remain derived, and actor/session independence is not persisted.
