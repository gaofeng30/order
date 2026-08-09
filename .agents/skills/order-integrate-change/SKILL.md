---
name: order-integrate-change
description: Prepare and integrate one independently verified OpenSpec change in the order repository. Use when a candidate is ready to merge, when main advanced after verification, or when deciding whether a completed change can be archived.
---

# Integrate an Order Change

## Check the gate

Require all of the following:

- OpenSpec artifacts and tasks are complete and strict validation passes;
- dependencies are integrated or pinned exactly as declared;
- the candidate full SHA has an independent PASS attestation;
- the diff contains only owned paths and intentional shared-contract edits;
- required review and repository checks pass.

Do not treat another change's unfinished work as a blocker unless there is a declared dependency or ownership conflict.

## Handle a moving main

1. Compare the verified candidate base with current `main`.
2. If integration requires rebase, merge or conflict resolution, return the update to the writer worktree.
3. Treat the updated commit as a new candidate SHA.
4. Rerun local and independent verification before integration.

Never carry verification forward across a SHA change.

## Integrate and archive

1. Follow the repository's authorized review and merge path. Do not push, create a PR or change external systems without explicit permission.
2. After merge, verify the integrated main SHA and required checks separately from the candidate result.
3. Only then run `openspec archive <change-name>` and commit the archive result as the repository process requires.

Report candidate SHA, integrated SHA, verification evidence and archive status as separate facts.
