---
name: order-verify-change
description: Independently verify an exact candidate SHA for one OpenSpec change in the order repository. Use after the implementation worktree has committed and self-tested a candidate, or whenever a rebase, merge, spec edit, task edit, or code edit invalidates earlier verification.
---

# Verify an Order Change

## Require exact inputs

Obtain the repository, change name and full candidate SHA. Do not verify an uncommitted working tree, branch name alone, or a moving ref.

Read and enforce `docs/quality/change-quality-gates.md`. Reject a candidate whose declared highest `gate_type`, actual `ui_level`, required external evidence, score or hard-blocker result is missing or inconsistent.

## Create an isolated verification surface

1. Create a separate clean detached worktree at the exact candidate SHA. Never reuse the writer worktree.
2. Confirm `git rev-parse HEAD` equals the requested SHA and `git status --porcelain` is empty.
3. Read `AGENTS.md` and all proposal, design, spec and task artifacts from that SHA.

Use a narrowly named temporary directory. Before removal, confirm it is the exact Git-managed verification worktree, is clean, is not a symlink and is outside the repository worktree being developed.

## Verify without repairing

1. Run `openspec validate <name> --strict`.
2. Inspect the candidate diff for declared dependencies, owned paths, public contracts and accidental unrelated files.
3. Execute every acceptance command required by specs and tasks, including failure scenarios and affected regression tests.
4. Confirm the worktree remains clean after verification.

Do not edit business code, specs or tasks. On failure, report the first decisive error with command and evidence; return the change to its writer. The writer creates a new SHA and verification starts again.

Implementation, spec, tasks, base, dependency, acceptance-command or SHA changes invalidate prior verification. For a repaired SHA, reuse the verifier session only with a newly clean detached worktree and a full rerun; repeated-error escalation remains owned by `order-run-loop`.

## Attest the result

Report:

- exact verified SHA;
- change name;
- commands and pass/fail results;
- unverified limitations, if any;
- `PASS` or `FAIL`.

Any later code, spec, task, rebase or merge change invalidates the result. If recording evidence creates a metadata-only commit, independently rerun the final SHA before integration.
