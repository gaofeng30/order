# Ticket: implement-store-operating-status-command-core

## Fixed point

- change: `implement-store-operating-status-command-core`
- status: `IMPLEMENTING`
- owner: independent Writer in `/Users/vivix/.codex/worktrees/7e9c/order`
- branch: `codex/implement-store-operating-status-command-core`
- base SHA: `8cae09d5bc3e659d8851e7588835e579101058ac`
- candidate SHA: `NOT_CREATED`; exact SHA is bound only after the final owned-only commit
- gate: `W3`
- UI target/actual: `UI0` / `UI0`
- dependency: the exact base already contains storefront v11, merchant identity v12 and audit v13; no unintegrated change dependency

## Goal and minimum success

Create `services/api/internal/storestatus/**` as the sole production mutation seam for `storefront_settings.business_status`. The Module accepts DB, `merchantidentity.Authorizer` and a clock at construction, then hides live RBAC, singleton locking, replay/conflict, exact single-column mutation, durable success audit, deadlock retry and rollback behind one `Apply` method.

Minimum success is all frozen slices in `spec.md` Green against real MySQL, all required mutants killed, full Writer Gate passing, an owned-only Chinese commit and a clean worktree. Review, detached verification and integration remain controller-owned.

## Ownership and non-goals

Owned only:

- `.scratch/implement-store-operating-status-command-core/**`
- `services/api/internal/storestatus/**`

All existing source, migrations, package metadata and other scratch paths are read-only. The Writer will not wire HTTP/UI, change schema, touch orders/payments/sold-out state, push, open/update a PR, merge, deploy or write an external system.

## Confirmed seams

1. Public command seam: construct `Core`, call `Apply`, observe `Result/error`.
2. Real persistence adapter: fresh isolated MySQL 8.0.46 running exact v1-v13 migrations.
3. Real authorization adapter: `merchantidentity.NewRepository(db).AuthorizeInTx` over live merchant-account rows.
4. Private fault adapter: controlled commit rollback is injected internally but remains observable only through `Apply` plus the subsequent public command/database invariant.
5. Mutation seam: disposable copies modify production source; named public command tests must kill each mutant.

## Writer commands

- focused compile/behavior: `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/storestatus -count=1`
- repeated race: same package with `-race -count=20`
- fresh MySQL W3: `bash .scratch/implement-store-operating-status-command-core/verify-mysql.sh`
- mutation shield/Gate: `bash .scratch/implement-store-operating-status-command-core/verify-mutation-gate.sh`
- full API: `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/... -count=1` and the same with `-race`
- static/build/smoke: `go vet ./services/api/...`; controlled `verify-build.sh`; `bash services/api/scripts/smoke.sh`
- hygiene: gofmt on owned Go files, `git diff --check <base>`, exact owned-path allowlist, sole-mutation scan, sensitive-material scan and final clean index/worktree

## Review, verifier and integration commands

- Standards review fixed point: `git diff 8cae09d5bc3e659d8851e7588835e579101058ac...<candidate>` and `git log 8cae09d5bc3e659d8851e7588835e579101058ac..<candidate> --oneline`; review code quality, ownership, deep-module locality and Gate evidence.
- Spec review uses the same exact base/candidate and independently checks every frozen transaction, idempotency, audit, concurrency, recovery and mutation requirement.
- Any finding invalidates that SHA; the Writer fixes it, reruns all Gates and creates a replacement SHA.
- Only after both review axes report zero findings may a verifier create another fresh clean detached worktree at the exact candidate and rerun every Writer command, ownership scan and clean check read-only.
- Integration command is controller-owned and not authorized here. A future authorized integrator must accept only the reviewed and independently verified exact SHA; rebase/merge produces a new candidate and invalidates prior verification.

## Assets, recovery and rollback

External assets: `N/A_FOR_THIS_BACKEND_MODULE`. No WeChat account, device, browser, production DB, payment asset or external platform is required or accessed.

The disposable loopback Docker MySQL fixture is cleaned after each Gate. Before integration, rollback is deletion of this branch/worktree by the controller. There is no migration or external state to roll back. Failed transactions are retried only for one MySQL deadlock/lock-timeout attempt; all other failures require restoring the failed dependency and calling `Apply` again with the same key.
