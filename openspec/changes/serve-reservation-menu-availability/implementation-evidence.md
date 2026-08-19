# Implementation evidence

## 2026-08-20 approval and entry

- Main-session review explicitly approved the frozen DB-driven contract and authorized this writer to enter `IMPLEMENTING`; it did not authorize push, PR, deploy, integration or archive.
- Frozen base, writer branch/worktree, dependency, owned paths, W3/UI0, `required_external_assets=none`, public contract and non-goals remain exactly as declared in `proposal.md`.
- Entry commands passed: `git rev-parse HEAD`, `git rev-parse main` and the ancestry check resolved to `babd1ef662811e3df6a75aa28995268352531438`; `openspec validate serve-reservation-menu-availability --strict` reported the change valid; `git diff --check` exited zero.
- Entry state was `order-mysql-w3=NOT_ESTABLISHED`, real MySQL W3=`NOT_RUN`; runtime establishment and foundation preflight were therefore completed before the first Red.

## 2026-08-20 writer runtime preflight

- Live Official Registry enumeration selected the frozen latest `mysql:8.0.46-oraclelinux9`, manifest `sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b`, arm64 platform `sha256:213bbfaf699693a40a20a12bb4342d2589a15a3dc7153db698eaed252a92458e`.
- The owned runtime is Colima 0.10.3 profile `order-mysql-w3`, aarch64/VZ/Docker, 2 CPU, 4 GiB memory, 10 GiB disk, no host mounts, no network address. Its exact-digest MySQL container is healthy on a loopback random port with a 0600 random credential file, zero mounts and a noexec/nosuid 1 GiB tmpfs; MySQL reports 8.0.46 and owned schema residue is zero.
- The first foundation script attempt exposed `invalid_connection_field` because the temporary TLS mode was `false`; changing only that env value to the existing accepted value `disabled` made the identical script pass. No secret or DSN was printed.

Structured task-level evidence is recorded under tasks 1.1 and 1.2. Subsequent Red, Green, Refactor and writer Gate evidence is appended to the corresponding task rather than inferred from this entry record.

## 2026-08-20 config and selection Red

- Focused tests froze current-config discrete points, closed endpoints, cutoff orderability, non-default valid data and missing/duplicate/unknown/negative/cross-day/nonzero-second/interval/order/alignment/overlap failures.
- The first focused command failed at the absent `MealPeriodRecord` and `ResolveMeal` production symbols. This is the expected observable Red; migrations, repository and HTTP remain untouched.
- The same focused command passed after the minimum validator/resolver, then passed again after a naming/formatting-only refactor. No default schedule or fallback was added.

## 2026-08-20 migrations Red, Green and Refactor

- Static Red observed exact v1-v7 chain and v4-v7 files missing. Real W3 Red then observed migration count 3 instead of 7 on the owned MySQL 8.0.46 runtime after the tests froze v7 schema, initial rows, CHECK, key and repeat behavior.
- Minimum v4-v7 statements made the static command and real catalog integration pass. A first Green expectation assumed lexical enum sorting; correcting that test to MySQL enum declaration order produced the same-command PASS.
- Refactor evidence reran the static migration command, foundation integration and catalog integration successfully. v1-v3 bytes were not edited.

## 2026-08-20 writer Gate

- The exact focused command and executable `menu-integration.sh` passed after refactor. Full `go test`, full race, vet, build, smoke, foundation integration, catalog integration and menu integration then all exited zero with the owned MySQL environment present.
- Strict, untracked-aware owned-path audit, catalog production/v1-v3 byte guards, whitespace/gofmt, one-versioned-route and executable-script checks passed. The first static checker used zsh's special `path` variable and lost `PATH`; renaming that checker variable was the only repair before identical PASS.
- Container identity remained healthy exact arm64 digest on loopback with zero mounts and owned schema residue zero. `C9/T10/V8/R9=36`, hard blockers `0`; V remains 8 until a different verifier checks the exact committed SHA.
