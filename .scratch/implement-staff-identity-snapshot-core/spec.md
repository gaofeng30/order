# Spec: staff identity snapshot core

## Module and interface

`services/api/internal/staffidentity` is a backend-only, pure in-process Module. Its only business function Interface is:

```go
func Resolve(input Input) (Snapshot, error)
```

The supporting public data types are frozen exactly as follows:

```go
type Input struct {
    PrimaryPhone       string
    Extra              *ExtraClaim
    WhitelistVersion   uint64
    CandidateEntries   []Entry
}

type ExtraClaim struct { Phone string; Name string }
type Entry struct { Phone string; Name string; Enabled bool }
type Kind string
type Snapshot struct { Kind Kind; WhitelistVersion uint64 }
```

`Kind` has only `VISITOR` and `STAFF`. Typed/redacted `Error` and `ErrorKind` expose only `Kind()` and stable categories `INVALID_PRIMARY_PHONE`, `INVALID_EXTRA_CLAIM`, and `INVALID_WHITELIST_SNAPSHOT`. No helper business function is exported. Inputs and results contain no user ID, record ID, source, discount rate, or pass-through phone/name.

This is a deep Module: callers provide one already-consistent candidate view, while canonical validation, Unicode normalization, duplicate/unrelated-evidence rejection, precedence, and fail-closed behavior remain local behind one seam.

## Canonical phone and name

- Every accepted phone is already canonical and matches the complete ASCII pattern `+[1-9][0-9]{0,14}`. The Module never rewrites phone input.
- Name normalization is exactly: apply `width.Fold`, then NFC canonical composition, then remove every U+0020 produced/preserved after width folding.
- It does not apply NFKC, case folding, pinyin, simplified/traditional conversion, punctuation folding, or removal/trim of any other Unicode whitespace.
- Consequently U+3000 folds to U+0020 and is removed; full-/half-width forms and half-width kana plus voiced mark normalize through width folding and NFC. `①` remains distinct from `1`, and `ﬃ` remains distinct from `ffi`.
- A claim/record name is valid only when its source string is valid UTF-8 and its normalized value is non-empty. Entries always validate their name even when primary-phone matching would otherwise ignore it.

## Validation priority and fail-closed errors

Validation occurs completely before identity resolution, with stable top-level priority:

1. Validate `PrimaryPhone`; failure returns zero `Snapshot{}` plus `INVALID_PRIMARY_PHONE`.
2. Validate `Extra`; `nil` is valid. A non-nil claim must have phone and name both present, a canonical phone, valid UTF-8 name, and non-empty normalized name. Any failure returns zero `Snapshot{}` plus `INVALID_EXTRA_CLAIM`.
3. Validate the whitelist snapshot; version must be nonzero. Then inspect entries in input order: phone must be canonical; name must be valid UTF-8 and normalize non-empty; phone must equal primary or the supplied extra phone; and no canonical phone may occur more than once. Any failure returns zero `Snapshot{}` plus `INVALID_WHITELIST_SNAPSHOT`.

All validation failures return the first category above and an exact zero Snapshot. Error text contains only the stable category, never PII or input values. There is no first/last-wins behavior. An empty `CandidateEntries` slice is valid.

`CandidateEntries` is the caller-provided complete consistent view for the primary phone and optional extra phone. An entry for any other phone makes that evidence untrustworthy. Duplicate canonical phones are invalid whether enabled or disabled. A malformed or unrelated later entry invalidates the whole snapshot even if an earlier enabled primary entry could match.

## Resolution

After all validation succeeds:

1. If the unique primary-phone entry exists and is enabled, return `STAFF` with the input `WhitelistVersion`; its name is ignored only for matching, not validation.
2. Otherwise, if Extra exists, its phone is distinct from PrimaryPhone, the unique entry for that extra phone exists and is enabled, and the equally normalized claim/record names match exactly, return `STAFF` with the version.
3. Otherwise return `VISITOR` with the version.

A disabled primary does not block a distinct enabled extra match. Disabled/missing entries and name mismatch are visitor results, not errors. If Extra repeats PrimaryPhone, it is not a distinct fallback; the valid snapshot resolves solely from the primary entry. Both visitor and staff preserve `WhitelistVersion`.

## Determinism and immutability

`Resolve` does not modify `Input`, `Extra`, `CandidateEntries`, their backing storage, or strings. Equal input produces equal Snapshot/error kind across repeated and concurrent calls. The implementation reads no clock, environment, random source, file, network, database, or global mutable state.

## RGR and mutation contract

- Red begins at the agreed public seam and must be a real compile/behavior failure caused by missing behavior.
- Green adds the minimum behavior for each vertical tracer.
- Primary match, versioned visitor, disabled primary, Extra phone, Extra name, Extra enabled, width folding, post-fold U+0020 deletion, NFC composition, and each validation behavior are separate named public-seam tracers with separate Red/Green boundaries. A state that was not saved in an immutable tree object is not immutable evidence.
- The not-NFKC negative contract becomes observably Red only through the post-Green single-source reversible `norm.NFC` to `norm.NFKC` mutation. It is a regression/mutation sensitivity anchor and is never described as an earlier implementation Red.
- The later primary-priority, Extra-priority, input-immutability/determinism, and same-primary-Extra tests are post-implementation regression/mutation anchors. Their focused Green results do not create or imply historical implementation Red evidence.
- Refactor reruns the identical public-seam focused contract set, repeated determinism, and race checks.
- The reversible mutation Gate must enforce exactly one source replacement per mutant; a kill is valid only when `go test` exits exactly `1` and output contains the named `--- FAIL: Test...` marker. Exit `2` or a missing marker is infrastructure failure, not a kill.
- At least twelve mutants cover: primary incorrectly requiring name; extra phone-only authorization; enabled ignored; disabled authorization; width-fold removal; ASCII-space removal; NFKC over-normalization; name-mismatch authorization; duplicate last-wins; visitor dropping version; malformed evidence treated as visitor; error returning partial Snapshot; and validation priority.

## Gates and evidence binding

- historical development base SHA: `f3c4efa4cd665652d93d5da76f92d18c4bdc59ac`. All 38 immutable granular `red`/`green` receipts retain this base because that is where those exact tree objects were produced and tested. They prove only the owned pure Module tracers; they do not claim pickup integration coverage.
- authoritative candidate comparison base SHA: `19ca1e46e106f293070f0cdf820951e31107cba6`. Final owned diff/static Gates, candidate review, formal review, and detached verification compare this base to the future exact candidate. The pickup integration write paths do not overlap this change's owned paths.
- lifecycle status: `CANDIDATE_READY_FOR_EXTERNAL_SHA_HANDOFF`; candidate SHA is still `not-yet-created`, so machine receipts use `candidate_sha: NOT_CREATED`. The next owned-only commit containing this governance delta forms the candidate, and the immutable full SHA is bound externally by the controller after commit because a commit cannot contain its own SHA. No future SHA is guessed or self-referenced.
- evidence enum schema: `phase` is exactly one of `red|green|refactor|writer|verifier|integration`; `exit_result` is exactly one of `exit-<integer>|PASS|FAIL|BLOCKED_EXTERNAL|N/A`. `red`/`green` records require the historical development base; `refactor`/`writer`/`verifier`/`integration` records require the authoritative candidate comparison base.
- the old-base post-freeze static/evidence/mutation-self-test observations, including the isolated `extra_phone_only_authorizes` run, are `INVALIDATED_BY_BASE_ADVANCE`. They remain historical observations only and earn no current Writer Gate credit.
- current authoritative-base Writer behavior Gates passed against receipt-before-docs staged source snapshot `b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc`: static/ledger, focused, determinism, race, mutation shield with 17/17 named kills, fresh loopback MySQL full API normal/race, vet/build/smoke, and cleanup. The first mutation run correctly remained FAIL after 16 kills when the `input_is_mutated` replacement violated the uniqueness shield; a minimum Gate-only replacement fix was followed by self-test, single-mutant, and full 17/17 reruns.
- the first two-axis pre-review is `INVALIDATED_AFTER_FINDING`: Standards reported one P1 because the ledger had no `phase: refactor` receipt while claiming Refactor/T10 complete; Spec reported zero findings. Neither axis from that attempt may support a candidate.
- the frozen implementation was then rerun identically at staged tree `f5daa28f883317494842a2f236981557366ab4ec`: focused package `count=1`, named determinism `count=100`, and package race `count=20` all passed and are recorded as three independent `phase: refactor` receipts. This resolved the evidence gap but did not itself constitute a new pre-review, so both axes were required to review again.
- replacement two-axis pre-review restarted from zero on frozen staged tree `b0846d513d8d7ec181231e3c01cf654ff77879d6` and passed: Standards=`0 findings`, `0/12 smells`; Spec=`0 findings`; start/end HEAD remained `19ca1e46e106f293070f0cdf820951e31107cba6`, start/end tree remained `b0846d513d8d7ec181231e3c01cf654ff77879d6`, with zero unstaged/untracked paths. This is writer pre-review only, never formal exact-candidate review.
- this final governance-only delta intentionally makes the next candidate tree differ from pre-review tree `b0846d513d8d7ec181231e3c01cf654ff77879d6`. The accepted immutable external-SHA handoff may form that candidate, but no formal review or independent result transfers from the pre-review tree. Formal Standards/Spec exact-candidate review and a fresh clean detached full-Gate verifier decide the committed SHA from zero.
- writer candidate-ready score is `C9/T10/V8/R9=36`. `V8` means only that, once the next owned-only commit supplies an exact externally bound SHA, the clean detached verification package is complete and awaiting verifier execution; it is not independent PASS. Until that external binding occurs, the ledger correctly retains `candidate_sha: NOT_CREATED`. Formal exact review, detached verifier, integration, and archive remain pending.
- owned/diff freeze Gate accounts separately for staged changes against base, unstaged tracked changes, and untracked files. In `post-freeze` mode it requires a non-empty allowed staged set and zero unstaged/untracked files before any behavioral Writer Gate.
- Completed Writer behavior Gate order on pre-receipt tree `b7ad5a82ae5aa5f1bdf1ef5eaab1759b04d392bc`: owned/diff/static `post-freeze` -> evidence-ledger consistency -> focused -> repeated determinism -> race -> mutation infrastructure shield -> fresh-MySQL adjacent/full API -> vet -> build -> smoke -> final sensitive/owned/diff/clean checks. Identical Refactor focused/determinism/race reruns passed on `f5daa28f883317494842a2f236981557366ab4ec`; replacement two-axis writer pre-review passed on `b0846d513d8d7ec181231e3c01cf654ff77879d6`. Formal exact-candidate review and detached verification remain pending.
- focused: `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/staffidentity -count=1`
- race repeated: same package with `-race -count=20`
- determinism repeated: focused determinism tests with `-count=100`
- mutation shield: `bash .scratch/implement-staff-identity-snapshot-core/verify-mutation-gate.sh`
- all API tests/race: fresh pinned `mysql:8.0.46` via the repository fixture plus `go test ./services/api/...` and `go test -race ./services/api/...`; this is adjacent/full regression evidence only, not DB transaction proof for this Module.
- static/build/smoke: `go vet ./services/api/...`, `go build ./services/api/...`, controlled API smoke.
- hygiene: gofmt, diff check, owned-path allowlist, sensitive scan, exact `go.mod` change, unchanged `go.sum`.
- review/verifier: Standards and Spec reviews plus another clean detached worktree compare `19ca1e46e106f293070f0cdf820951e31107cba6...<exact-candidate>`.
- integration: not authorized and not performed.

External assets: UI, production whitelist storage/API, caller wiring, pricing/order/payment flow and real production data are `N/A_FOR_THIS_PURE_MODULE`; their future owners must supply their own Gates. Docker plus a fresh pinned MySQL fixture is required only for the declared all-API adjacent regression and must be loopback-only and cleaned up.
