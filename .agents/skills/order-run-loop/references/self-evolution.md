# Control runner evolution at module boundaries

Use this protocol only for runner evolution. Keep lifecycle execution in the four existing `$order-*` stage skills.

## Freeze one runner base

Before allocating work in every module, compute and persist:

| field | evidence |
| --- | --- |
| `repo_sha` | full immutable checked-out repository SHA |
| `skill_blob` | `git rev-parse <repo_sha>:.agents/skills/order-run-loop/SKILL.md` |
| `skill_sha256` | SHA256 of the checked-out `SKILL.md` bytes |
| `runner_version` | front matter version when present, otherwise an explicit unversioned marker |

Record the module, lifecycle state, candidate/integrated/archive SHA, dependency, blocker, error fingerprint/repeat count and next action in the repository checkpoint. On resume, rebuild state from this checkpoint, OpenSpec tasks, exact SHAs and current repository facts. Do not use old chat memory as evidence.

If the current Skill identity differs after the base is frozen, retain the frozen base, record the mismatch and stop or continue under the existing module contract as governance permits. Never silently switch rules in a running module.

## Record observations without self-editing

Append observations during the module. Do not edit the active runner because of them. Assign each record exactly one immutable class:

| class | use |
| --- | --- |
| `candidate` | one explicit, testable runner-rule proposal |
| `environment` | machine, tool version, network or transient local state |
| `checker` | checker correctness, portability, parsing or false-green behavior |
| `external` | unavailable facts, qualifications, accounts, secrets or irreversible authority |

Derive a new `candidate` from an `environment`, `checker` or `external` record without changing the source class. Preserve the source ID.

## Separate DRAFT admission, promotion and activation

At a module boundary, admit a candidate to a queued or created dedicated OpenSpec `DRAFT` only when all five fields exist:

1. reproducible evidence;
2. a generalizable rationale or safety-critical rationale;
3. explicit non-weakening intent;
4. an executable regression plan;
5. an executable minimal-context forward-test plan.

DRAFT admission is not promotion. It does not approve, implement, verify or activate a rule. Keep an incomplete record in the observation ledger.

After the dedicated change implements the rule, promote its exact candidate only when all of these are current PASS for the same SHA:

- reproducibility and generalizability or safety impact are revalidated;
- non-weakening checks PASS;
- implemented regression PASS;
- a fresh session in a clean detached exact-SHA worktree passes the minimal-context forward-test;
- complete independent verification PASS.

Reject promotion if any item is missing, failed or `BLOCKED_EXTERNAL`. Never use C/T/V/R, coverage or another PASS to compensate. Treat skipping tests, deleting failure cases, loosening assertions, replacing tested production logic with mocks, false-green constructs or bypassing exact-SHA independence as hard blockers.

A promoted rule remains inactive until its change is integrated into local main. Activate it only when the next module captures a new runner base after that integration. A verified rule outside main stays inactive, and an active module never switches runners.

## Keep checker evidence portable and truthful

Apply these contracts to regression and Gate tooling:

- **zero-match count**: return numeric `0` for a valid search with no matches; still fail invocation or parsing errors.
- **Markdown field parsing**: parse an exact structural field, ignore fenced lookalikes, and fail missing or duplicate fields.
- **literal backticks**: transport fixture text through argument vectors or standard input; never evaluate backticks, `$()` or metacharacters through a shell.
- **explicit case semantics**: normalize intentionally case-insensitive fields, including `awk` checks; keep case-sensitive fields exact.
- **bounded health polling**: set attempt and time bounds; return ready only after an observed healthy state and fail at the bound.
- **safe temporary directories**: create a narrow system-temp target; resolve and verify its parent, name, type, link state and entries before cleanup; stop on validation or cleanup failure.
- **archive trailing newline**: remove only one LF or CRLF record terminator and compare every remaining character exactly.

Do not use `|| true` or another mechanism that turns an unexpected failure into PASS. Record the first decisive error and apply the existing same-fingerprint stop rule.

## Close the module

Write a retrospective that lists observations by immutable class, DRAFT-screen decisions, missing evidence and the target later Goal/module. Queue DRAFT-admissible lessons when no runner-change lane is authorized. Do not call them promoted and do not reopen the Skill within the same Goal.
