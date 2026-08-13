# Archived lifecycle receipt

## Receipt fields
receipt_schema: archived-lifecycle-receipt/v1
change_name: supersede-miniprogram-catalog-evidence
archive_path: openspec/changes/archive/2026-08-13-supersede-miniprogram-catalog-evidence
lifecycle_state: ARCHIVED
expected_historical_verdict: VERIFIED_SUPERSESSION
failure_fingerprint: none
subsequent_gates: PASS
superseded_by: none
supersedes: 6d77bdd6319722b7c71b4726c6159955da9a84b6
repo_base_sha: 7d01fe22ded67aeded78cb7d03de87aa12416ada
runner_skill_git_blob: 211498668419dcb66d11f5bfacf7457ed385aa05
runner_skill_sha256: c347f88e3593ed80cea79e4b2d4885bb6d92c677d1e1012c12171fab34482d14
runner_version: unversioned
candidate_sha: 109c8e828f6f5a10adff33ccdb73d4fd784b2f3d
integrated_sha: 109c8e828f6f5a10adff33ccdb73d4fd784b2f3d
integration_validation: PASS
archive_sha: 510a84baddfb5b61200c159fd2041b7c512a92db
archive_validation: PASS
owned_paths_json: ["openspec/changes/supersede-miniprogram-catalog-evidence/**"]
profile_id: menu-supersession-v1
profile_binding_sha256: dab5c5d69791056564c79ee5829934f71cbd018b7abd2951ec8a15ff8502c728
recorded_attestation_trust: UNTRUSTED_FOR_MECHANICAL_PASS
recorded_attestation_json: {"claimed_verdict":"PASS","source":"historical exact-SHA verifier handoff"}
mechanical_verification: REQUIRED_DERIVED
product_trees_json: {"apps/wechat-miniprogram":"80d16424aefa0d4b9d4e451a1ebe5e8013627a8b","services/api/internal/catalog":"1867e1cb94fd38b718641d28022e1cf2e386c85b","services/api/internal/httpapi":"38f9f486156547cd547d2f3840566acfbbd4c0eb"}
candidate_checkpoint_sha256: 63d5b263012cbf666cc195d01e2207fe06986243ecf1c0468a822663082a4a3a
candidate_tasks_sha256: c1d6a5d22cdaa13fe9eac9305bee55a6fb06d96926a2431bae1feced9f0f0e1f
candidate_open_tasks_json: ["6.1","6.2","6.3","7.1","7.2","7.3","8.1"]
post_candidate_tasks_json: {"6.1":"PASS","6.2":"PASS","6.3":"PASS","7.1":"PASS","7.2":"PASS","7.3":"PASS","8.1":"PASS"}
receipt_head_verification: REQUIRED_DERIVED
observation_counts_json: {"candidate":0,"checker":1,"environment":0,"external":2}
draft_screen_decisions_json: ["replacement evidence preserves the old FAIL and independently replays the current product tree"]
retrospective_json: ["legacy Red, UI1, provider, static and scope evidence are mechanically replayable","current delivery recovery remains layered and does not repair the old candidate"]
unverified_boundaries_json: ["mechanical replay does not re-prove verifier actor or session independence","UI2 and UI3 and real order or payment remain outside this evidence"]

## Retrospective
The reciprocal link and exact tree identity support only a mechanically reproducible current delivery when the replacement profile passes. They never relabel the historical failed candidate.
