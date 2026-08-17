## Context

The repository has moved from a pure P0 prototype to a hybrid state: the Go service owns MySQL migrations and an anonymous menu catalog, and three mini-program read paths consume it, while transaction, payment, identity and merchant-management flows remain mock or unimplemented. The code is current, but five developer-facing documents still describe the earlier all-mock state. The independently verified lightweight Harness also needs a short user entry, without changing protected governance or representing its local ledger as proof.

This documentation-only change starts from Harness candidate `79c4f1d8ebf300bf2f9f4c226fcd2c2aa2643963`, owns only the declared Markdown/OpenSpec paths, and depends on that change for integration.

## Goals / Non-Goals

**Goals:**

- Give a first-time developer one consistent answer for what is API-backed, what is mock, and what is not implemented.
- Put operational commands near the code they operate without duplicating AGENTS or Skill rules.
- Preserve explicit verification and external-runtime boundaries.
- Keep all local Markdown links valid.

**Non-Goals:**

- No code, API, schema, product, contract, customer, deployment or governance change.
- No new documentation hierarchy, changelog or general-purpose checker.
- No claim of real MySQL, WeChat, payment, UAT or production completion.

## Decisions

### Edit the existing reader entry points

The root README remains the repository and developer entry; the mini-program README describes its hybrid runtime; the Demo guide describes what a presenter can actually demonstrate; the technical document states current as-built implementation beside the target architecture; the docs index routes readers. This is shorter and less fragile than creating parallel integration, architecture, runbook and handoff documents for a small repository.

### Describe one hybrid boundary everywhere

All five documents use the same boundary: catalog read is API-backed; cart snapshots and the remaining user/merchant flows are local; Web Admin is a static mock client. Product target behavior remains in PRD and is not rewritten as current behavior.

### Keep Harness guidance thin

Only `status` and `check` appear as newcomer commands. `checkpoint` and `observe` remain discoverable through `--help` and OpenSpec because they mutate the local operational index and should be used in the controlled workflow. Root AGENTS and stage Skills stay untouched.

### Use executable content and link checks

Red/Green evidence uses exact stale/required phrase checks plus relative-link resolution. Repository strict validation and diff/owned-path audits cover structure; no artificial code test is added for Markdown.

## Risks / Trade-offs

- [Risk] The documentation change could land before its Harness dependency. → Integration is blocked until `add-lightweight-harness-loop` is in current `main`.
- [Risk] P0 customer documents intentionally describe mock behavior. → Only developer-facing current-state wording changes; contracts, customer checklists and archives remain untouched.
- [Risk] A future code change can make prose stale again. → Acceptance binds the current code paths and stale phrases; future behavior changes must update these same entry points through their own OpenSpec change.

## Migration Plan

1. Validate the documentation change on its dependency candidate.
2. Integrate `add-lightweight-harness-loop` first.
3. Refresh this branch on the resulting `main`, create a new candidate and rerun verification.
4. Integrate this docs-only change, then archive it separately.

Rollback is a revert of only the owned documentation and OpenSpec paths; runtime behavior is unaffected.

## Open Questions

None.
