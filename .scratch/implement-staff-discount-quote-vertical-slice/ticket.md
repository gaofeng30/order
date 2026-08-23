# Ticket: implement-staff-discount-quote-vertical-slice

## Fixed point and goal

- Change: `implement-staff-discount-quote-vertical-slice` (`TX-02`).
- Current exact base: `1657aa9451f612e4605fabd084ccab07542ac81a`.
- Gate: `W3`; UI target/actual: `UI0` / `UI0`.
- Goal: persist versioned staff/discount facts and immutable server-priced Quotes
  behind authenticated create and owner-read seams, plus the transaction-bound
  finalization/snapshot interface required by TX-03. Quote freezes required
  contact name plus the server-bound primary phone and exposes only its masked
  phone over HTTP. Quote v16 persists its deterministic effective expiry and
  v17 freezes the nullable product cover object key. No prepayment/order
  creation, provider call, or root composition is included.
- Candidate SHA and final Gate outcome are intentionally not self-recorded in
  this frozen tree; they belong to the post-freeze external attestation.

## Integrated dependency

The current base includes the verified store-status command. That module alone
owns production writes to `storefront_settings.business_status`; TX-02 reads
and locks the singleton while creating a Quote and does not add a competing
write seam. TX-02 exposes the locked revalidation method; TX-03 remains the
owner of prepayment creation and transaction orchestration, including its
future `deadline-now >= 1m` provider guard.

## Downstream transaction contract

`FinalizeForPrepayInTx` and `LoadSnapshotInTx` are public `internal/quote`
methods over a caller-owned `*sql.Tx`. The public Quote derives
`ExpiresAt=min(created_at.UTC()+10m,pickup_at.UTC())`; the persisted value must
match that digest-covered calculation. Finalize treats the exact deadline as
expired, locks current facts before Quote/items under the global rank, then
validates the snapshot and positive payment amount and rechecks current identity
semantics from the locked v18 primary/extra identity, product/cover/menu/store
availability, target service date, current flavor membership and cutoff without
repricing for discount drift. LoadSnapshot validates frozen rows only. The stable error set
is `NOT_FOUND`, `EXPIRED`, `PRIMARY_PHONE_REQUIRED`, `QUOTE_STALE`,
`ITEM_UNAVAILABLE`, `PICKUP_CUTOFF_PASSED`, `PAYMENT_AMOUNT_TOO_SMALL`,
`SNAPSHOT_INVALID`, and `UNAVAILABLE`. Persisted corruption is
`ErrSnapshotInvalid`; SQL/driver/transaction/1205/1213 failures are retryable
`ErrUnavailable`.

Quote creation accepts no client phone or amount. Contact names are exact
trimmed UTF-8 values of at most 64 bytes; full primary phone and contact name
are persisted and digest-covered, while responses mask the phone. Staff rates
are `1..100`, visitor rate is `100`, payable must be at least one cent, and
quantity is positive with overflow checks but no arbitrary upper cap.

Quote Create is the only TX-02 mutator. It uses `WriteMeta` plus the injected
unified operation-receipt store; receipt append is insert-last. A receipt UNIQUE
race rolls back the entire business transaction and replays from a new read-only
transaction; it never returns a conflict merely because the first request won.
Discount/whitelist mutation types and authorization belong solely to the
separate `StaffDiscount` lane.

## Owned paths

- `.scratch/implement-staff-discount-quote-vertical-slice/**`
- exact migrations `000014` through `000017`
- `services/api/internal/quote/**`
- `services/api/migrations/embed_test.go`, only to name and validate the exact
  ordered v1-v17 embedded migration chain
- `services/api/internal/catalog/migrations_test.go`, only to append the exact
  v14-v17 current-all ledger while retaining its exact historical v2-v3 prefix

Every other catalog/migrations file, all v1-v13 SQL, router/main, apps, existing
feature packages, `go.mod`, and `go.sum` are read-only. No other writer owns the
two reallocated test files during this change.

## Migration-ledger TDD decision

The accepted Red is the observed exact failure of both strict current-all
ledgers at 17 loaded migrations while expecting 13. Green must enumerate all
four new filenames in order; embedded validation must retain an exact statement
prefix for each. `>=`, prefix-length widening, globbing, future-version
acceptance, or changing the historical catalog prefix are forbidden.

## Current boundary

This replacement tree is prepared for one root-run Writer Gate. Feature tests
may create the frozen v18/v25/v29 facts after applying exact v1-v17, but TX-02
owns no v18+ migration. No push, PR, merge, deploy, external write, root
composition, prepayment/order creation, or provider integration is authorized.
