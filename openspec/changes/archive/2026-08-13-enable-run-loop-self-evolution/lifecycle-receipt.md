# Archived lifecycle receipt

## Receipt fields
receipt_schema: archived-lifecycle-receipt/v1
change_name: enable-run-loop-self-evolution
archive_path: openspec/changes/archive/2026-08-13-enable-run-loop-self-evolution
lifecycle_state: ARCHIVED
expected_historical_verdict: VERIFIED
failure_fingerprint: none
subsequent_gates: PASS
superseded_by: none
supersedes: none
repo_base_sha: 2209c071a21860231827b2a8c8c81d9b7745e6e1
runner_skill_git_blob: d529461de5af1bf7cc65562e59ec3c84f0750963
runner_skill_sha256: 558b549a4410d72d4c22acad621ffae96af3aeccd26adc186ede76601097aa59
runner_version: legacy-unversioned
candidate_sha: 7a5e8bb261b994d68ce9af5eada347df6700c490
integrated_sha: 7a5e8bb261b994d68ce9af5eada347df6700c490
integration_validation: PASS
archive_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444
archive_validation: PASS
owned_paths_json: [".agents/skills/order-run-loop/**","openspec/changes/enable-run-loop-self-evolution/**"]
profile_id: self-evolution-v1
profile_binding_sha256: 012c4a414e59b371a375a8b5cf2026099173f41a7b121deccb1bc2e0686e4a59
recorded_attestation_trust: UNTRUSTED_FOR_MECHANICAL_PASS
recorded_attestation_json: {"claimed_verdict":"PASS","source":"historical exact-SHA verifier handoff"}
mechanical_verification: REQUIRED_DERIVED
product_trees_json: {}
candidate_checkpoint_sha256: 7c59c03c3fcfc0945ff22f1435099dbd9bc3e6965980ddedab4b5730f669ce27
candidate_tasks_sha256: ced70b3f7cba944a7a32e39efab85d84964f4057d907f9d70e12e01012f7505b
candidate_open_tasks_json: ["5.1","5.2","5.3","6.1","6.2","6.3","6.4"]
post_candidate_tasks_json: {"5.1":"PASS","5.2":"PASS","5.3":"PASS","6.1":"PASS","6.2":"PASS","6.3":"PASS","6.4":"PASS"}
receipt_head_verification: REQUIRED_DERIVED
observation_counts_json: {"candidate":0,"checker":7,"environment":0,"external":0}
draft_screen_decisions_json: ["historical runner rule was admitted, independently verified, integrated and archived"]
retrospective_json: ["deterministic contract and seven positive/seven negative checker cases are mechanically replayable","fresh actor independence remains outside mechanical replay"]
unverified_boundaries_json: ["mechanical replay does not re-prove verifier actor or session independence"]

## Retrospective
This backfill preserves the historical PASS only as audit history. Current recovery derives deterministic repository evidence and never upgrades it into actor-independence proof.
