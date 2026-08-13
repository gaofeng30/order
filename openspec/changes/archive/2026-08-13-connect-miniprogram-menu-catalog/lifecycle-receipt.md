# Archived lifecycle receipt

## Receipt fields
receipt_schema: archived-lifecycle-receipt/v1
change_name: connect-miniprogram-menu-catalog
archive_path: openspec/changes/archive/2026-08-13-connect-miniprogram-menu-catalog
lifecycle_state: ARCHIVED
expected_historical_verdict: FAILED
failure_fingerprint: artifact-consistency|semantic|proposal-stale-draft-not-run|6d77bdd|verifier
subsequent_gates: NOT_RUN
superseded_by: 109c8e828f6f5a10adff33ccdb73d4fd784b2f3d
supersedes: none
repo_base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444
runner_skill_git_blob: 211498668419dcb66d11f5bfacf7457ed385aa05
runner_skill_sha256: c347f88e3593ed80cea79e4b2d4885bb6d92c677d1e1012c12171fab34482d14
runner_version: unversioned
candidate_sha: 6d77bdd6319722b7c71b4726c6159955da9a84b6
integrated_sha: 6d77bdd6319722b7c71b4726c6159955da9a84b6
integration_validation: NOT_RUN
archive_sha: 7d01fe22ded67aeded78cb7d03de87aa12416ada
archive_validation: NOT_RUN
owned_paths_json: ["apps/wechat-miniprogram/app.js","apps/wechat-miniprogram/components/customize/customize.js","apps/wechat-miniprogram/components/customize/customize.wxml","apps/wechat-miniprogram/package-lock.json","apps/wechat-miniprogram/package.json","apps/wechat-miniprogram/pages/confirm/confirm.js","apps/wechat-miniprogram/pages/confirm/confirm.wxml","apps/wechat-miniprogram/pages/detail/detail.js","apps/wechat-miniprogram/pages/detail/detail.wxml","apps/wechat-miniprogram/pages/detail/detail.wxss","apps/wechat-miniprogram/pages/home/home.js","apps/wechat-miniprogram/pages/home/home.wxml","apps/wechat-miniprogram/pages/home/home.wxss","apps/wechat-miniprogram/pages/menu/menu.js","apps/wechat-miniprogram/pages/menu/menu.wxml","apps/wechat-miniprogram/pages/menu/menu.wxss","apps/wechat-miniprogram/tests/catalog-ui1.test.js","apps/wechat-miniprogram/tests/page-harness.js","apps/wechat-miniprogram/utils/catalogApi.js","apps/wechat-miniprogram/utils/catalogStore.js","apps/wechat-miniprogram/utils/util.js","openspec/changes/connect-miniprogram-menu-catalog/**"]
profile_id: old-menu-artifact-fail-v1
profile_binding_sha256: 17841bbfa1f9a21ff39e9c669dcf2c14e7c34c31b32948effab4e878b52de4ea
recorded_attestation_trust: UNTRUSTED_FOR_MECHANICAL_PASS
recorded_attestation_json: {"claimed_verdict":"FAIL","fingerprint":"artifact-consistency|semantic|proposal-stale-draft-not-run|6d77bdd|verifier","source":"historical exact-SHA re-verification"}
mechanical_verification: REQUIRED_DERIVED
product_trees_json: {"apps/wechat-miniprogram":"80d16424aefa0d4b9d4e451a1ebe5e8013627a8b","services/api/internal/catalog":"1867e1cb94fd38b718641d28022e1cf2e386c85b","services/api/internal/httpapi":"38f9f486156547cd547d2f3840566acfbbd4c0eb"}
candidate_checkpoint_sha256: 8fccd8d5707a2e404b9cfa9077de6df1835130f129f9168d129a08a3332729e1
candidate_tasks_sha256: a9ba01d15121ae3aeac293b8fe6e508fec1284188b1ddb641dcb1f2ffb870f43
candidate_open_tasks_json: ["7.1","7.2","7.3","7.4","8.1","8.2"]
post_candidate_tasks_json: {"7.1":"NOT_RUN","7.2":"NOT_RUN","7.3":"NOT_RUN","7.4":"NOT_RUN","8.1":"NOT_RUN","8.2":"NOT_RUN"}
receipt_head_verification: REQUIRED_DERIVED
observation_counts_json: {"candidate":0,"checker":1,"environment":0,"external":2}
draft_screen_decisions_json: ["historical exact candidate remains FAILED and all later Gates remain NOT_RUN"]
retrospective_json: ["artifact fields conflict at the exact candidate","the later supersession is linked without changing this failure"]
unverified_boundaries_json: ["UI2 and UI3 remain BLOCKED_EXTERNAL","real order and payment remain unverified"]

## Retrospective
The old candidate stays FAILED. Neither archive presence, product-tree identity nor the reciprocal supersession link can convert this history into PASS.
