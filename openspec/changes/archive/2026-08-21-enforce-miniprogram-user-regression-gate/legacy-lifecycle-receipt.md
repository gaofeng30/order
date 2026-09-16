# Archived lifecycle receipt

## Receipt fields
receipt_schema: archived-lifecycle-receipt/v1
change_name: enforce-miniprogram-user-regression-gate
archive_path: openspec/changes/archive/2026-08-21-enforce-miniprogram-user-regression-gate
lifecycle_state: ARCHIVED
expected_historical_verdict: VERIFIED
failure_fingerprint: none
subsequent_gates: PASS
superseded_by: none
supersedes: none
repo_base_sha: 2299da6013c06f4ae7dcad535373c67b66c80ebb
runner_skill_git_blob: 0f41f64ad87fc9fd410cb916b4d1562aee92e42f
runner_skill_sha256: 2781bdda1544106e30e7483c4b500d611df85c79753cc1bd3b717d91d1edaac8
runner_version: unversioned
candidate_sha: f5719e98690d0b1301ed567c8b616e074846b445
integrated_sha: f5719e98690d0b1301ed567c8b616e074846b445
integration_validation: PASS
archive_sha: b53cb520c4cac0505803a9d1e6dcc1807ac34540
archive_validation: PASS
owned_paths_json: ["openspec/changes/enforce-miniprogram-user-regression-gate/**","AGENTS.md","openspec/config.yaml","docs/quality/change-quality-gates.md","tools/harness","tools/miniprogram_gate.py","tools/tests/test_harness.py","tools/tests/test_miniprogram_gate.py",".agents/skills/order-plan-change/SKILL.md",".agents/skills/order-implement-tdd/SKILL.md",".agents/skills/order-verify-change/SKILL.md",".agents/skills/order-integrate-change/SKILL.md"]
profile_id: miniprogram-user-regression-gate-v1
profile_binding_sha256: 878dbeac46011e0ae59d568d037aa9e8b914458abcb5d26512563ddbe36cb75f
recorded_attestation_trust: UNTRUSTED_FOR_MECHANICAL_PASS
recorded_attestation_json: {"actor_independence":"NOT_PROVEN_BY_MECHANICAL_REPLAY","binding_head_sha":"07688c7cdcc957bd9fed48f8c408c929a930c23f","claimed_verdict":"PASS","source":"current exact-B independent verifier final"}
mechanical_verification: REQUIRED_DERIVED
product_trees_json: {}
candidate_checkpoint_sha256: 535a53c39b1177a795422acd6cad85ebf9c60ac5a545ff169306b6c674a564e3
candidate_tasks_sha256: da0d82d6a66830b19c6375e4cd014b0a0214d7e10a96079c91818c9cf119349e
candidate_open_tasks_json: ["6.1","6.2","6.3","7.1","7.2"]
post_candidate_tasks_json: {"6.1":"PASS","6.2":"PASS","6.3":"PASS","7.1":"PASS","7.2":"PASS"}
receipt_head_verification: REQUIRED_DERIVED
observation_counts_json: {"candidate":2,"checker":5,"environment":1,"external":0}
draft_screen_decisions_json: ["the reproducible Mini Program false-green met all five admission fields and was implemented only through its dedicated control-plane change","the exact pre-convention checkpoint incompatibility was admitted only through a separately verified nonweakening compatibility change","remaining task-overlay and metadata-drift observations were not promoted in this Goal"]
retrospective_json: ["candidate observation remains queued because an immutable verified SHA cannot mark its own post-candidate tasks","checker observations were handled without editing the frozen active runner or protected receipt judge","the Python 3.9 environment observation remains an explicit compatibility boundary","no external observation was promoted or hidden"]
unverified_boundaries_json: ["mechanical replay does not prove actor or session independence","the compatibility view does not claim the historical checkpoint recorded CANDIDATE","UI2 UI3 and real WeChat execution remain outside this receipt","remote GitHub enforcement push PR and deployment remain outside this closure"]

## Retrospective
The loop reproduced the original false-green and the later exact pre-convention checkpoint incompatibility, promoted each only through separately judged OpenSpec changes, and preserved all frozen protected bytes. The compatibility verifier reports the original IMPLEMENTING/none checkpoint facts, while mechanical and receipt-head results remain derived rather than persisted.
