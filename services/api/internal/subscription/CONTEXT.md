# Subscription module context

## Frozen seam and scope

- Public seam: `RecordConsent`, `EnqueueInTx`, `RunDue`.
- HTTP owner (outside this package): `POST /api/v1/orders/:id/subscriptions`.
- MySQL schema owner (outside this package): v38 `notification_consents`, v39 `notification_outbox`.
- Provider seam: `SendSubscription`; the deterministic fake is local/test injection only.

## Invariants

- Only `READY` and `REFUND_RESULT` exist; only an `ACCEPTED` consent can be consumed.
- Lock order is order -> consent -> outbox. The command receipt is inserted last and is never locked first.
- One accepted consent is consumed by at most one outbox row; `(order,kind)` has one immutable intent.
- Claim commits a short lease/version CAS before provider send. Provider send is always outside a DB transaction.
- Temporary failure returns to `PENDING`; success and permanent failure are terminal. An expired lease may be sent again, so local uniqueness is not WeChat exactly-once.
- Persisted message JSON uses a typed non-PII payload and never contains openid, phone, provider secret, or raw provider error text.

## TDD evidence

- [x] Red: `TestRecordConsentPersistsAcceptedDecision` failed to compile before the public seam existed.
- [x] Green: accepted/rejected decision persistence and stable validation.
- [x] Red -> Green: real SQL order lock -> consent lock -> insert -> command receipt last.
- [x] Red -> Green: enqueue order lock -> accepted consent lock -> outbox lock/write -> consent consume.
- [x] Red -> Green: worker lease claim/commit -> provider send -> SENT CAS.
- [x] Red -> Green: temporary/permanent provider errors persist only stable redacted codes.
- [x] Red -> Green: malformed date/time payload is rejected before persistence.
- [x] Fast Gate: `go test ./services/api/internal/subscription -count=1`.
- [x] Race Gate: `go test -race ./services/api/internal/subscription -count=1`.

## External and deferred verification

- Real WeChat subscription template application/send is `BLOCKED_EXTERNAL` and is not proven by the fake provider.
- Fresh MySQL v38/v39 migration execution and real lock/deadlock races are integration Gate work; this lane does not run Docker or own migrations.
