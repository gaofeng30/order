# Spec: implement-store-operating-status-command-core

## 1. Authority and scope

- Canonical product authority: `docs/product/online-ordering-system-prd-0818.md` §6.1, §6.9, §6.10 and §16.3 P1.
- The explicit change contract resolves the §6.9/§6.10 role wording for this Module: both `OWNER` and `SUBACCOUNT` may change operating status. An `OWNER`/`SUBACCOUNT` role change is serialized and the committed live role is audited; disabled, deleted or invalid-role accounts are rejected.
- Fixed base: `8cae09d5bc3e659d8851e7588835e579101058ac`.
- Gate: `W3`; UI target/actual: `UI0` / `UI0`.
- This is a backend-only deep Module. It proves the command transaction, authorization, idempotency, audit and recovery behavior only.

## 2. Owned, read-only and non-goals

Only these paths are owned:

- `.scratch/implement-store-operating-status-command-core/**`
- `services/api/internal/storestatus/**`

Read-only dependencies and contracts:

- `services/api/internal/storefront/**`
- `services/api/internal/merchantidentity/**`
- all `services/api/migrations/**`, especially v11 and v13
- router, main, apps, `go.mod`, `go.sum`, every other scratch path and every other repository path

If `merchantidentity.Authorizer` or the v13 audit schema cannot express the frozen transaction, implementation stops and reports the exact blocker. No ownership expansion is permitted.

Non-goals: HTTP, router/main wiring, UI, migrations, merchant-account management, orders, payment/refund, product sold-out state, external platforms, production data, push, PR, merge, deploy or integration.

## 3. Public Interface and seam

The only behavior seam is the constructed `Core` and its `Apply` method:

```go
type Command struct {
	UserID          uint64
	DesiredStatus   storefront.BusinessStatus
	IdempotencyKey  string
	RequestID       string
}

type Result struct {
	Before  storefront.BusinessStatus
	After   storefront.BusinessStatus
	Changed bool
}

func New(db *sql.DB, authorizer merchantidentity.Authorizer, clock func() time.Time) *Core
func (*Core) Apply(context.Context, Command) (Result, error)
```

The Interface also exposes the minimum stable error modes `ErrInvalidCommand`, `ErrIdempotencyConflict` and `ErrUnavailable`; live authorization errors from `merchantidentity.Authorizer` remain distinguishable with `errors.Is`.

Tests observe behavior through `Apply`. Real MySQL and the real `merchantidentity.Repository` are the production adapters. A private commit adapter may be used only to make controlled commit failure observable through the same `Apply` seam.

## 4. Input and dependency contract

Before beginning a database transaction, `Apply` MUST reject:

- `UserID == 0`;
- `DesiredStatus` outside `open|closed|cutoff`;
- empty, invalid UTF-8 or leading/trailing-whitespace `IdempotencyKey`;
- `RequestID` that is empty, over 64 bytes, invalid UTF-8 or has leading/trailing whitespace;
- nil DB, nil Authorizer, nil clock, or a zero clock value.

The clock is sampled once per attempt, converted to UTC and truncated to microseconds. Invalid commands MUST win over invalid dependencies so malformed input is rejected without DB or Authorizer access.

## 5. Transaction and idempotency contract

Each attempt MUST use exactly one transaction in this order:

1. call `AuthorizeInTx(ctx, tx, UserID, ActionStoreStatusWrite, Target{Type:"storefront_settings", ID:1})`;
2. read singleton `storefront_settings.id=1` with `FOR UPDATE` and validate current status;
3. read the first audit with the same actor user, action and raw `SHA256(IdempotencyKey)`; validate its target, result, reason and status pair;
4. if the first audit has target `storefront_settings/1` and its `state_after` equals `DesiredStatus`, return its original `{Before,After,Changed}` after a read-only commit;
5. if that key is already bound to another resource or desired status, return `ErrIdempotencyConflict` and roll back;
6. for a new key, update only `storefront_settings.business_status` when `Before != DesiredStatus`; no-op skips the UPDATE;
7. insert exactly one `SUCCEEDED` audit with live merchant account, role, auth version, actor user, exact action/target, request ID, key hash, before/after state, occurred-at and reason `OPERATING_STATUS_CHANGED` or `OPERATING_STATUS_UNCHANGED`;
8. commit once, then return the result.

Audit role is derived only from the live authorization actor: `merchant_owner -> OWNER`, `merchant_subaccount -> SUBACCOUNT`. Unknown/zero authorization facts fail closed.

The singleton row lock serializes every status command before replay inspection. Concurrent same-key/same-desired commands converge to one write and one audit. Concurrent same-key/different-desired commands produce one committed winner and one conflict. Different keys form a committed before/after chain.

MySQL deadlock or lock-timeout errors (`1213`/`1205`) retry the complete transaction once, including live authorization. No other error retries. Missing/bad singleton data, authorization failure, query/update/audit failure or final commit failure returns no success; every controllably failed transaction is rolled back. A production commit error is reported as unavailable because its outcome cannot be inferred locally.

## 6. Mutation ownership invariant

After this change, production Go source MUST contain exactly one mutation statement for `storefront_settings`, inside this Module, and it MUST update only `business_status`. Migrations, fixtures and tests are not mutation seams.

The mutation Gate MUST kill at least these reversible mutants:

1. accept an illegal status enum;
2. bypass live authorization;
3. authorize with the wrong action;
4. authorize the wrong target;
5. remove singleton `FOR UPDATE`;
6. let a duplicate key rewrite state/audit;
7. let a conflicting key return success;
8. update another storefront column;
9. omit or corrupt the success audit;
10. commit after audit insertion failure.

Every mutation runs in a disposable copy, requires one exact source match, must exit through the named behavior assertion, and must leave the writer tree unchanged. The harness first proves its infrastructure-failure shield.

## 7. Required W3 slices

Each slice uses one public-seam test, records a real Red before the minimum Green, and binds both observations to immutable Git tree objects:

1. illegal input is rejected before DB/authorization;
2. real MySQL OWNER changes status and every other storefront column remains byte/value-identical;
3. real MySQL SUBACCOUNT changes status and every other storefront column remains identical;
4. disabled, deleted and invalid-role accounts fail closed; OWNER/SUBACCOUNT commit ordering uses the live role snapshot;
5. no-op, sequential replay and conflicting desired status;
6. concurrent same-key convergence, conflicting-key winner and different-key serialization;
7. audit exactness and rollback when the audit table is unavailable;
8. controlled commit rollback, real deadlock retry, missing row, invalid row, DB failure and successful next-attempt recovery.

Refactor occurs only after all slices are Green, then reruns the same focused, race, fresh-MySQL, mutation and repository regression commands.

## 8. Evidence and limits

- No HTTP/UI/platform asset exists or is required: external assets are `N/A_FOR_THIS_BACKEND_MODULE`.
- Fresh loopback-only `mysql:8.0.46-oraclelinux9` is a disposable test fixture, not production or external evidence.
- UI actual remains `UI0`; no browser, Mini Program, device, UAT or production behavior is claimed.
- OpenSpec is `N/A`: root `AGENTS.md` makes `openspec/**` historical and it is outside owned paths.
- Candidate status is recorded externally after the owned-only commit so the commit never self-references its own SHA.
