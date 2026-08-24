# Core acceptance L1/L2 receipt

## Scope

- Base: `1049799c184a1f42a49c2781d04a230787a462be`
- Owned paths:
  - `services/api/cmd/order-api/acceptance_core_invariants_e2e_test.go`
  - `.scratch/acceptance-core-invariants-l1-l2/receipt.md`
- Production code, migrations, router, apps, acceptance manifest and coverage ledger were not changed.
- Evidence uses root-composed HTTP handlers, a fresh v1-v44 MySQL schema per L2 test, and local fake providers. It is not Mini Program UI L3 or real-WeChat L4 evidence.

## Red -> Green

1. Baseline inventory validation passed with 95 cases; the cases below still lacked the named L1/L2 evidence. The current PC selector already covered AC-18, so this change deliberately does not duplicate it.
2. First compile Red: the new migration helper incorrectly called `ReadDir`/`ReadFile` as methods on `embed.FS`. It was reduced to the standard `io/fs` API; focused static execution then passed.
3. First MySQL Red: the test expected an invented `RETRY` notification state. The existing durable contract is `PENDING` with `last_error_code=RATE_LIMITED` and non-null `next_attempt_at`; the assertion was corrected without production changes.
4. Green:

   ```text
   GOTOOLCHAIN=go1.26.5 GOPROXY=off go test ./services/api/cmd/order-api -run 'TestAcceptanceCore' -count=1
   PASS

   ORDER_TEST_MYSQL_HOST=127.0.0.1 ORDER_TEST_MYSQL_PORT=33093 \
   ORDER_TEST_MYSQL_USER=root ORDER_TEST_MYSQL_TLS_MODE=disabled \
   ORDER_TEST_MYSQL_INSTANCE=order-mysql-w3 ORDER_TEST_MYSQL_ISOLATED=YES \
   GOTOOLCHAIN=go1.26.5 GOPROXY=off \
   go test -race -p=1 ./services/api/cmd/order-api -run '^TestAcceptanceCore' -count=1 -v
   PASS (3 tests, 2.658s)
   ```

5. Cleanup: `SHOW DATABASES LIKE 'order_acceptance_%'` returned zero rows after Green.

## Case evidence

| Case | New level | Expected fact | Failure shield |
|---|---:|---|---|
| PAGE-U05 | L2 | bound primary phone snapshot, contact, flavor, note and cover key are in the server quote/digest | unbound, cutoff, fact drift and client-owned price cannot create prepayment |
| PAGE-U09 | L2 business fallback | server order history starts empty and only materialized paid orders appear | client/local fake success cannot create order or navigation fact |
| AC-05 | L2 | menu, detail and quote resolve the same integer-cent staff price | client price input is rejected |
| AC-09 | L2 | initial paid orders within 30 minutes remain `PAID`; exact 30 minutes advances them once | replay is idempotent and does not advance again |
| AC-16 | L2 | last owner remains owner | disable, downgrade and delete of the last owner return `LAST_OWNER` |
| AC-18 | existing selector, no new claim | current `TestAcceptancePCPagesCloseWithDerivedFactsAndFailureShields` already asserts derived revenue/metrics and query non-mutation | unclaimed orders are excluded; no duplicate selector added |
| AC-19 | L1+L2 local | audit rows cover privileged actions without persisting raw phone, session token, pickup token or provider ciphertext | fake external completion tables and cosmetic identity columns are absent |
| BE-10 | L2 | owner/sub-admin permissions and last-owner invariant are enforced through HTTP | visitor/non-merchant and sub-admin owner-approval attempts are rejected without mutation |
| BE-17 | L2 | refunded order clears pickup-token ciphertext while retaining a hash for audit | old token cannot redeem a refunded order |
| INV-01 | L1+L2 | one singleton storefront, one pickup point and reservation pickup are the server facts | delivery, instant pickup and a second point are rejected |
| INV-11 | L2 | orders strictly before 30 minutes remain `PAID`; at the boundary the worker advances once | no early preparation and no replay mutation |
| INV-14 | L2 | accepted READY/REFUND_RESULT intents and rejected/temporary-failure outcomes are durable | rejected consent creates no outbox; provider failure does not mutate order state |
| INV-15 | L2 | date-scoped pickup sequence resets across dates and concurrent confirmations allocate distinct numbers | forced `9999` exhaustion fails closed with no order, no `10000`, and no burned sequence |
| INV-16 | L1+L2 | all authoritative identity, price, payment, order, refund and notification facts are server persisted | unknown client fields, in-memory/provider fake success and forbidden columns cannot become facts |

## Intentionally not closed

- No target case is promoted to full ready by this receipt alone.
- PAGE-U05/PAGE-U09/AC-05/AC-09/INV-01 and other page-facing scenarios still require their matrix-specified Mini Program L3 evidence.
- AC-19 still requires L3 client-visible audit/error-path evidence; any real WeChat/cloud L4 remains external.
- This change does not claim full 95-case acceptance or submission readiness.
