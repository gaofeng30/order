# Minimal-context fresh-session forward-test

This scenario is an independent verifier Gate, not a writer self-test. Run it only in a clean detached worktree at the exact candidate SHA.

## Context budget

Give the fresh session only these repository files:

1. `AGENTS.md`
2. `.agents/skills/order-run-loop/SKILL.md`
3. `.agents/skills/order-run-loop/references/self-evolution.md`
4. this scenario

Also give it the exact candidate SHA and require read-only operation. Do not give it prior chat, writer claims, `tasks.md`, design rationale or expected prose.

## Scenario

Ask the session to coordinate a synthetic module with these observations:

- a reproducible checker portability defect whose candidate has a generalizable rationale, non-weakening intent, an executable regression plan and an executable forward-test plan;
- an otherwise similar candidate missing its forward-test plan;
- one machine-specific tool failure;
- one unavailable real credential;
- one checker defect from which a new candidate must be derived without changing the original class.

The current module froze its runner before these observations. A separately promoted rule exists in two variants: one is only independently verified outside main, and one is explicitly integrated into local main before a later module starts.

Require the session to return only a JSON object accepted by `checks/verify_forward_test.py`. It must compute the runner identity from the detached worktree, keep the current module unchanged, classify all four observation types, keep the incomplete candidate in observations, label the complete candidate `draft_admissible_not_promoted`, keep a verified-but-not-integrated rule inactive, activate only the local-main-integrated rule for the next module, and preserve the four existing stage routes.

Use this exact result shape and replace the identity placeholders with values computed from the detached worktree:

```json
{
  "candidate_sha": "<exact candidate SHA>",
  "repository_clean": true,
  "current_module_rule_changed": false,
  "read_files": [
    "AGENTS.md",
    ".agents/skills/order-run-loop/SKILL.md",
    ".agents/skills/order-run-loop/references/self-evolution.md",
    "openspec/changes/enable-run-loop-self-evolution/forward-test.md"
  ],
  "runner_base": {
    "repo_sha": "<exact candidate SHA>",
    "skill_blob": "<40-character Skill blob SHA>",
    "skill_sha256": "<64-character Skill content SHA256>",
    "runner_version": "unversioned"
  },
  "observations": [
    {"id": "candidate-1", "class": "candidate", "current_module_applied": false},
    {"id": "environment-1", "class": "environment", "current_module_applied": false},
    {"id": "checker-1", "class": "checker", "current_module_applied": false, "derived_candidate_id": "candidate-2"},
    {"id": "external-1", "class": "external", "current_module_applied": false}
  ],
  "incomplete_screen": "stay_observation",
  "complete_screen": "draft_admissible_not_promoted",
  "verified_outside_main": "inactive",
  "integrated_local_main": "activate_next_module_only",
  "routes": {
    "DRAFT": "$order-plan-change",
    "APPROVED": "$order-implement-tdd",
    "CANDIDATE": "$order-verify-change",
    "INDEPENDENT_VERIFIED": "$order-integrate-change"
  },
  "hard_gates_weakened": false
}
```

## Verifier command

Store the JSON outside the repository in a narrowly named validated temporary directory, then run:

```sh
/usr/bin/python3 openspec/changes/enable-run-loop-self-evolution/checks/verify_forward_test.py \
  --candidate-sha <exact-40-character-candidate-sha> \
  <validated-result.json>
```

A PASS proves only this exact SHA under the declared minimal context. It does not prove integration, activation on main, external platform behavior or another SHA.
