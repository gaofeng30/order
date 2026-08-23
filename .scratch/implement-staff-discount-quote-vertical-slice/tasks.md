# TX-02 execution evidence

- [x] Historical WIP-base work is retained only as Red/Green development
  history and grants no current candidate status.
- [x] Current authoritative candidate base frozen as
  `1657aa9451f612e4605fabd084ccab07542ac81a`.
- [x] Prior candidate `9a6d540` invalidated by final Review findings; no evidence
  from it grants replacement-candidate status after these edits.
- [x] Integrated store-status dependency reviewed: it solely writes
  `storefront_settings.business_status`; Quote retains a consistent locked read.
- [x] Read root `AGENTS.md`, canonical 0818 PRD, `CONTEXT.md`, `$tdd`,
  `$codebase-design`, integrated staffidentity/quotepricing/storefront/menu/catalog,
  and migrations v1-v13.
- [x] Freeze Quote's four entrypoints and keep StaffDiscount administration
  entirely outside TX-02.
- [x] v14 staff whitelist static Red -> Green.
- [x] v15 discount settings static Red -> Green.
- [x] v16 quotes static Red -> Green.
- [x] v17 quote items static Red -> Green.
- [x] Exact v1-v17 embedded/catalog ledger Red -> Green under reallocated ownership.
- [x] Authenticated staff quote create/read tracer Red -> Green.
- [x] Visitor rate 100 and per-line half-up Red -> Green.
- [x] 80 -> 75 immutability Red -> Green.
- [x] Idempotent replay/conflict and rollback Red -> Green.
- [x] Ownership, PII-free fail-closed, and complete digest snapshot Red -> Green.
- [x] Light race/mutation/static/full-package review.
- [x] TX-03 transaction-seam Red: focused compile failed on missing
  `FinalizeForPrepayInTx`, `LoadSnapshotInTx`, and stable finalization errors.
- [x] TX-03 transaction-seam Green: exact
  `min(created_at UTC + 10m, pickup_at UTC)` boundary,
  current identity/product/store/menu checks, discount/version-only immunity,
  frozen-only snapshot loading, and caller-owned rollback semantics pass.
- [x] Contact/payment P1 Red: focused tests failed to compile before public
  contact snapshot/error fields existed; missing/untrimmed/client phone,
  64-byte boundary, digest tamper, zero payment, and quantity 100 were then
  observable failures.
- [x] Contact/payment P1 Green: required contact and server primary phone are
  immutable/digest-covered, HTTP masks the phone, rate is `1..100`, payable is
  positive at create/finalize, and quantity remains positive/overflow-only.
- [x] Effective-deadline Red -> Green: tests first failed on missing
  `ExpiresAt`; exact earlier-pickup expiry and pickup-before-creation now fail
  closed, and frozen v16 persists the deterministic value under a strict CHECK.
- [x] Error-classification Red -> Green: corrupt snapshot tests first failed on
  missing `ErrSnapshotInvalid`; digest/persisted corruption now uses that stable
  error while simulated 1205/1213 reads remain `ErrUnavailable`.
- [x] Frozen schema/interface delta: v14 persists the NFKC/whitespace-free name
  key, v16 persists digest-covered effective expiry, and v17 persists the
  nullable item cover object key.
- [x] Frozen WriteMeta/receipt contract: Quote Create alone appends one non-PII
  transaction-bound receipt last. Receipt UNIQUE losers rollback all Quote
  writes and replay in a distinct read-only transaction.
- [x] Frozen Finalize lock order: locate without locks, lock current facts by
  rank, then lock Quote/items and recheck locator/digest. Current cover drift is
  stale; LoadSnapshot never reads current facts.
- [x] Final Review S1 Red/Green: an AST contract first reported all Quote-owned
  admin provider/mutator/receipt symbols; the package now owns only Quote Create
  receipt semantics and has no StaffDiscount command vocabulary.
- [x] Final Review S2 Red/Green: the exact v14 contract first failed on
  `idx_staff_whitelist_enabled_id`; the unbudgeted `(enabled,id)` index was
  removed without weakening any exact migration assertion.
- [x] Final Review identity Red/Green: an extra-phone+name whitelist match first
  produced `VISITOR/100`; Quote now locks the complete v18 identity group and
  resolves primary/extra candidates in one `staffidentity.Resolve` call, while
  Finalize rejects extra-identity semantic drift.
- [x] Final Review service-date Red/Green: missing and closed rows first created
  Quotes; Create now writes zero rows and Finalize returns stale for either fact.
- [x] Final Review flavor Red/Green: duplicate, unavailable and forged frozen
  duplicate flavors were first accepted; Create/Load/Finalize now enforce
  uniqueness, current membership and config drift at their correct boundaries.
- [x] Final Review DTO Red/Green: focused HTTP output first returned `meal`;
  POST/GET now expose only `pickup.meal_period`.
- [x] Fresh MySQL seam Green: Quote/item locks block a competing update until
  caller rollback; product/status/identity drift, exact deadline and pickup
  cutoff return the stable expected errors.
- [x] Fresh MySQL exact v1-v17 history and Quote W3 behavior.
- [x] Re-run full API normal/race, vet, build and smoke after the seam.
- [ ] Post-freeze external Writer Gate: 26 mutations, fresh MySQL Quote receipt race,
  full normal/race, owned/sensitive/cleanup. Its result is intentionally not
  self-recorded in this candidate tree.
- [ ] Post-freeze candidate commit and exact-SHA attestation; intentionally not
  self-recorded in this candidate tree.

## Evidence log

Append one decisive command/result for each completed Red and Green pair.

- v14 Red: `go test ./services/api/internal/quote -run '^TestStaffWhitelistMigrationContract$' -count=1` failed because `000014_create_staff_whitelist.sql` did not exist.
- v14 Green: the same command passed after adding the single-statement table migration.
- v15 Red/Green: `TestDiscountSettingsMigrationContract` failed on the missing v15 asset, then passed with singleton rate/discount/whitelist versions.
- v16 Red/Green: `TestQuotesMigrationContract` failed on the missing v16 asset, then passed with no TTL/prepay/order fields.
- v17 Red/Green: `TestQuoteItemsMigrationContract` failed on the missing v17 asset, then passed with immutable product snapshots and no product foreign key.
- Migration-chain Red/Green: `TestMigrationSetLoadsExactlyThroughV17` isolated the loader-rejected v16 constraint name; renaming only the constraint to avoid the reserved `SOURCE ` token made the exact v1-v17 load pass.
- HTTP tracer Red/Green: `TestAuthenticatedUserCreatesAndReadsOwnImmutableQuote` first failed to compile without the feature types/handler, then passed with strict authenticated create/read seams.
- HTTP fail-closed Red/Green: client-owned money, duplicate JSON keys, invalid application snapshots, owner hiding, stable typed errors, and GET bodies each failed against the permissive adapter before the minimum strict boundary passed.
- Provider snapshot Red/Green: the first locked staff create failed before `NewProvider`; staff `101 * 80%` now rounds half-up to 81 per unit before quantity, while visitors use 100.
- P5 Red/Green: `TestLaterDiscountFactAffectsOnlyQuotesCreatedAfterIt` exposed nil empty-flavor snapshots during immutable re-read; after canonicalizing empty arrays, an old 80/11 Quote remains 162 cents and a later 75/12 Quote is 152 cents.
- Replay/rollback/read Green extensions: exact replay returns the stored full snapshot without source reads, different input with the same key conflicts, injected failure rolls back the transaction, and non-owner reads return not found.
- Concurrent light gate: `TestConcurrentQuoteCreationNeverMixesDiscountRateAndVersion -count=20` observed only 80/11 or 75/12 pairs.
- Light package gate: MySQL environment variables explicitly unset; `go vet ./services/api/internal/quote`, `go test -race ./services/api/internal/quote`, and direct quotepricing/staffidentity/menu/storefront regressions passed.
- Initial mutation gate: `.scratch/implement-staff-discount-quote-vertical-slice/mutation.sh` killed five mutants covering staff rate, idempotency conflict, client money, snapshot digest, and empty flavor arrays; the seam expansion later brings the declared total to eight.
- Fresh MySQL test was compiled and skipped before the token. Its current scope
  covers exact v1-v17 apply/repeat/history, staff/visitor pricing, 80->75
  immutability, idempotency, ownership, atomic rollback, concurrent source
  pairs, and digest corruption.
- Reallocated ledger Red: before ownership changed,
  `TestEmbeddedMigrationChainIsExactAndRecoverable` failed with
  `embedded migrations = 17, want exact v1-v13 chain`, and
  `TestCatalogMigrationSet` failed with `migration count = 17, want 13`.
- Reallocated ledger Green: appended the four exact names
  `000014_create_staff_whitelist.sql`, `000015_create_discount_settings.sql`,
  `000016_create_quotes.sql`, and `000017_create_quote_items.sql`; the embedded
  ledger also requires exact `CREATE TABLE ` prefixes. No count widening,
  wildcard, or future-version acceptance was added, and catalog's historical
  `wantNames[1:3]` v2-v3 prefix remains unchanged.
- Pre-seam heavy evidence: fresh MySQL Quote W3, full `services/api` normal/race,
  vet, build and smoke passed on the current base before the downstream seam;
  this evidence is retained only as diagnostic history and grants no final
  candidate status.
- Transaction-seam Red/Green: focused tests first failed to compile on the
  missing public interface/methods/errors, then passed for strict expiry,
  identity/product/status/cutoff changes, discount/whitelist-version immunity,
  frozen-only loading and caller-owned rollback.
- Expanded mutation Green: all eight declared mutations are killed by their
  exact behavior tests; the three seam mutants change `>=` to `>`, make discount
  version drift stale, and make `LoadSnapshotInTx` read mutable settings.
- P1 mutation Green: all sixteen then-declared mutations were killed. The eight added
  mutants cover client phone acceptance, contact digest omission, zero payment
  at create/finalize, earlier-pickup expiry, pickup-before-creation, zero staff
  discount, and corruption incorrectly collapsed into retryable unavailability.
- Frozen-delta mutation set: four earlier mutants covered cover/expiry digest
  inclusion, receipt UNIQUE replay, current cover drift, and frozen behavior.
- Final Review mutation Green: the current 26-mutant set additionally kills
  omission of extra identity, Create/Finalize service-date checks, duplicate or
  unavailable flavors, Finalize flavor drift, and the `meal_period` DTO key.
- Replacement local regression: focused Quote normal and race passed; all 26
  current mutations were killed; `go vet ./services/api/...` and isolated API
  binary build exited zero. Full API normal/race reached all packages but the
  sandbox denied localhost binds in the existing `internal/app`, `wechat` and
  `wechatpay` listener tests; the root-run replacement Writer Gate remains the
  authority for those tests and fresh MySQL.
- Fresh MySQL seam Green: exact v1-v17 history still passes; finalization locks
  header/items until caller rollback, while current product/status/identity and
  exact cutoff failures remain stable and PII-free.
- Historical pre-review MySQL W3: isolated MySQL 8.0.46 applied and replayed the
  exact clean v1-v17 history, enforced the real `1..100` discount constraint,
  froze server contact, rejected zero payment with zero rows, retained caller
  locks, and expired an earlier-pickup Quote at the exact effective deadline.
- Historical pre-review repository regression: full `go test ./services/api/...`,
  full `go test -race ./services/api/...`, `go vet ./services/api/...`, isolated
  builds, and `services/api/scripts/smoke.sh` all exited zero on the current
  authoritative base plus the pre-finding owned delta; it is not replacement
  Writer evidence.

## TX-02-MIGRATION-LEDGER-01 receipt

- `receipt_status`: `HISTORICAL_LIGHT_PASS_INVALIDATED_BY_BASE_ADVANCE`
- `phase`: `green`
- `historical_development_base_sha`:
  `historical WIP base; not current evidence`
- `current_authoritative_base_sha`:
  `1657aa9451f612e4605fabd084ccab07542ac81a`
- `owned_delta`: exactly `services/api/migrations/embed_test.go` and
  `services/api/internal/catalog/migrations_test.go`, limited to strict v14-v17
  ledger entries and the exact v1-v17 diagnostic.
- `red`: the two focused tests above exited nonzero with exact `17, want 13`
  failures before this delta.
- `green`: `go test ./services/api/migrations -run
  '^TestEmbeddedMigrationChainIsExactAndRecoverable$' -count=1` and `go test
  ./services/api/internal/catalog -run '^TestCatalogMigrationSet$' -count=1`
  both exited zero.
- `read_only`: every other catalog/migrations file remains unchanged.
- `unverified_boundary`: this old-base receipt grants no current-base Writer,
  fresh MySQL, candidate, review, or detached exact-SHA verification status.
