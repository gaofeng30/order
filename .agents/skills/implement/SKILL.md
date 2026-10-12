---
name: implement
description: "Implement a piece of work based on a spec or set of tickets."
disable-model-invocation: true
---

Implement the authorized work described by the spec or tickets. Read the [repository workflow bridge](../../../docs/agents/skills.md); Matt artifacts do not replace existing Writer, independent-verifier or integration gates.

Use $tdd at existing or approved public seams, following the repository's Red → Green → Refactor evidence requirements. Do not repeat confirmation for decisions already settled by the contract.

Run focused checks during implementation and every applicable check required by the [quality gates](../../../docs/quality/change-quality-gates.md) before a candidate. Record actual results, skips and missing assets. A baseline failure requires the same check on the unchanged base under equivalent conditions; it remains a failed required gate and does not authorize unrelated repairs.

Use $code-review with the declared base SHA and originating spec to review the work. In an already authorized implementation, return in-scope findings to the writer for repair and rerun affected checks; standalone review remains read-only.

After the Writer Gate passes, when the repository permits and the user has not prohibited it, create a local candidate commit on the task branch in its dedicated worktree. Inspect the staged and unstaged diff and include only this task's changes. If existing staged content or mixed-file edits cannot be safely separated, stop automatic commit without clearing or resetting the user's index. Record the full candidate SHA, then obtain independent verification in another clean detached worktree. Neither review nor candidate commit grants push, PR, merge or deployment authorization.
