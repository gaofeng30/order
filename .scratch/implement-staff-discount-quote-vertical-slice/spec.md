# TX-02 Staff Discount Quote Vertical Slice

## Exact base and ownership

- Current authoritative integration base:
  `1657aa9451f612e4605fabd084ccab07542ac81a`.
- Earlier WIP-base evidence is development history only and is not current
  Writer verification.
- Owned: this scratch directory, migrations v14-v17, the new
  `services/api/internal/quote` package, and exactly
  `services/api/migrations/embed_test.go` plus
  `services/api/internal/catalog/migrations_test.go` for strict v14-v17
  migration-ledger updates only.
- Root composition, existing production packages, every other catalog or
  migrations file, v1-v13 SQL, apps, and module files remain read-only.

## Frozen product rules

- A saved global rate is immediately effective for later quote creation only.
- Existing quotes are immutable. A rate/version change never recalculates or
  makes an existing quote stale by itself.
- Visitor payable rate is `100`; staff payable rate is the current singleton
  rate, and a saved staff rate is restricted to `1..100`.
- Each line rounds `unit_price_cents * rate / 100` half-up before multiplying
  quantity.
- Client monetary values are not an input to quote creation.
- Staff pricing resolves the locked v18 user identity through one
  `staffidentity.Resolve` call: an enabled primary-phone match is sufficient;
  an extra-phone match requires the frozen extra name to match the same enabled
  whitelist row. The trusted primary phone remains mandatory for contact.
- `contact_name` is required, valid UTF-8, already trimmed, non-empty, and at
  most 64 bytes. The bound primary phone is read inside the Quote transaction;
  a client phone is never accepted. Both are immutable digest-covered facts,
  while HTTP responses expose only contact name and a masked phone.
- A calculated payable amount below one cent fails as
  `PAYMENT_AMOUNT_TOO_SMALL` before any Quote row is written. Finalization
  rechecks the same invariant so a downstream caller creates zero prepayments.
- Quantity has no invented inventory cap: each line requires only a positive
  integer, with all arithmetic overflow failing closed.
- Quote v16 persists `expires_at`; it is deterministically calculated as
  `min(created_at.UTC()+10m, pickup_at.UTC())`; `observed_at >= expires_at` is
  `EXPIRED`. The persisted value is digest-covered and must equal the
  deterministic calculation; a pickup instant before creation is invalid.
- TX-03, not this change, will require at least one minute remaining before its
  provider Create call. TX-02 creates no prepayment or provider request.
- Target `service_dates` rows are current facts: a missing or closed row rejects
  Quote Create and makes Finalize stale. Each selected flavor is unique and a
  member of the locked singleton `flavor_options_json`; removal is current-fact
  drift. HTTP emits pickup `meal_period`, never `meal`.
- Prepay must later revalidate price, identity, whitelist, listed/sold-out,
  service date, storefront/flavors, business status, and cutoff. This module
  does not create prepay or order records.

## Deep module and seams

The `quote` module is the locality for consistent source reads, staff identity
resolution, pricing, immutable persistence, idempotency, ownership, and HTTP
projection.

Confirmed external HTTP seam:

- `POST /api/v1/quotes`: strict Bearer, strict JSON, one idempotency key,
  server-owned source reads, create or replay.
- `GET /api/v1/quotes/:id`: strict Bearer and owner-only immutable read; a
  non-owner observes the same not-found result as an absent quote.

Frozen ownership boundary:

- `StaffDiscount` is the sole owner of discount and whitelist mutations. TX-02
  contains no admin provider, merchant authorizer, staff/discount command type,
  or HTTP-adjacent administration vocabulary and does not edit
  `internal/staffdiscount/**`.
- Quote Create accepts only `WriteMeta{ActorUserID,IdempotencyKey,RequestID}`
  and appends its non-PII `quote.create` operation receipt last through the
  injected transaction-bound store.
- If the receipt UNIQUE key loses a first-write race, the complete business
  transaction rolls back. Replay then occurs from a new read-only transaction
  with no business locks; a same-key different request is the only public
  idempotency conflict.

The production MySQL provider and HTTP adapter are the two adapters at the HTTP
application seam.

The current base includes the independently verified store-status command. It
is the production owner of `storefront_settings.business_status` writes. Quote
creation takes a consistent locked read of that singleton and fails closed
unless it is `open`; TX-02 neither duplicates that write seam nor modifies the
store-status package. Future prepay still revalidates business status.

Confirmed downstream transaction seam for TX-03:

- `FinalizeForPrepayInTx(ctx, tx, userID, quoteID, observedAt) (Quote, error)`
  performs an unlocked locator read, locks current facts in the frozen global
  rank, then locks Quote/items and revalidates the locator/digest. It enforces
  the effective deadline and positive payment amount and rechecks current
  full v18 primary/extra identity semantics, product price/cover/listing/
  category/sold-out state, target service date, store/flavor facts, and pickup
  schedule/cutoff. The caller owns commit and rollback.
- `LoadSnapshotInTx(ctx, tx, quoteID) (Quote, error)` reads and validates only
  the frozen Quote header/items. It never reads current product, discount,
  identity, whitelist, store, or menu facts.
- Both methods create no prepayment, order, provider request, or root route.
  Stable errors are `NOT_FOUND`, `EXPIRED`, `PRIMARY_PHONE_REQUIRED`,
  `QUOTE_STALE`, `ITEM_UNAVAILABLE`, `PICKUP_CUTOFF_PASSED`,
  `PAYMENT_AMOUNT_TOO_SMALL`, `SNAPSHOT_INVALID`, and `UNAVAILABLE`.
- Persisted snapshot validation or digest corruption returns
  `ErrSnapshotInvalid`. SQL, driver, transaction, 1205/1213, and temporary DB
  failures return `ErrUnavailable`; these categories are never collapsed.
- Discount rate/version drift never invalidates or reprices a Quote. Whitelist
  source-version drift alone also passes when current resolved `STAFF` or
  `VISITOR` semantics still match the frozen Quote.

## Request and snapshot

Quote creation accepts contact name, pickup date/time, order note, and ordered
items containing only product id, positive quantity, unique ordered flavor
strings, and line note. Duplicate product ids and duplicate flavors are
rejected; every flavor must be a member of the locked current singleton option
set. Product name, phone, and all money come from server-owned sources.

One transaction freezes:

- owner user id, contact name, and full bound primary phone;
- identity kind and global whitelist source version;
- applied rate and discount version;
- store name/address, pickup point, date/time, and meal;
- product id/name, nullable cover object key, content-derived product source
  version, original and discounted unit price, quantity, line totals, flavors,
  and notes;
- quote totals, canonical request digest, full snapshot digest, created time,
  and a deterministically derived effective expiry returned by the public Quote.

The product source version is a SHA-256 content version of the locked menu row;
existing product tables have no mutable version column. Future prepay must still
re-read the source facts rather than trusting this digest alone.

## Error and idempotency contract

- Errors expose stable PII-free categories only.
- Corrupt immutable rows/digest are durable `ErrSnapshotInvalid`; infrastructure
  failures remain retryable `ErrUnavailable`.
- An exact `(user_id, idempotency_key_hash)` replay with the same canonical
  request digest returns the stored quote from a new read-only replay
  transaction after releasing its per-key named lock.
- The same key with a different canonical request fails closed as conflict.
- Any transaction failure commits neither quote nor quote items.
- Authentication failure, unavailable/invalid persisted facts, and ownership
  failure do not disclose phone, name, SQL, DSN, or another user's quote.

## Acceptance

- Static migration contracts for v14-v17 and exact ordered v1-v17 embedded and
  catalog ledgers; historical catalog checks remain the exact v2-v3 prefix.
- v14 has only its primary key and unique phone key; its exact non-unique
  secondary-index count is zero.
- HTTP/provider behavior for staff and visitor quotes, half-up line rounding,
  fake client amount rejection, exact replay/conflict, owner-only reads, and
  immutable complete snapshots.
- Discount `80 -> 75`: old quote remains 80; later quote is 75.
- Concurrent upstream discount changes/Quote creates never pair a rate with the
  wrong version.
- Injected transaction failure leaves no half quote.
- Concurrent first writes for Quote commit one aggregate mutation and one
  receipt; losers rollback and replay the first PII-free response from a
  distinct read-only transaction.
- Cover object key and persisted effective expiry are digest-covered; Finalize
  rejects current cover drift while LoadSnapshot remains source-independent.
- Contact/primary-phone snapshots are digest-covered and only the masked phone
  crosses HTTP; payable zero and client phone/money produce zero Quote rows.
- The prepay transaction seam enforces the exact effective deadline boundary,
  holds Quote/item locks until the caller completes the transaction, revalidates
  service-date/flavor/full-identity and all other required current facts, and
  leaves `LoadSnapshotInTx` source-independent.
- Fresh MySQL applies and checks the exact clean v1-v17 history, then exercises
  Quote concurrency, idempotency, rollback and immutable snapshots. Feature
  fixtures add only the frozen post-v17 identity/storefront/service-date facts;
  TX-02 adds no v18+ migration. The authoritative v1-v44 gate remains a root CP
  integration responsibility.
