# Design

## Confirmed scope and contracts

Use existing public HTTP and page seams. Image uploads hash the literal bytes, send the existing MIME/size/SHA256 to intent, PUT unchanged bytes with only returned storage headers, then confirm the server object key with the same idempotency identity. Cache only a matching confirmed image. Never forward Admin bearer or cookies to storage.

The shared order read generation retains only the latest requested cache snapshot while each caller still receives its own response. The page paint generation retains the latest lane/date/search rendering. Global counts read all cursor pages independently; unavailable or incomplete facts remain unknown. Preserve six states, immutable paid amounts, notes/flavor facts and server payable line subtotals. Product required-field checks explain the already-existing server boundary before an invalid write.

Finance captures selected dates and tab for each paint, drops obsolete/detached responses, and retains unavailable state rather than manufacturing zero amounts. CSV filenames remain bound to the dates at the export click. Settings loading and failure states expose manual retry and render only authoritative successful fields. Account name/avatar/title follow the current account rather than prototype literals; menu containment and existing routes remain intact.

## Authorization cache commits

Existing visible logout awaits current-bearer server revocation.204 or current401 clears local authorization; unavailable server/network preserves failed state and manual retry. Current401 must clear token/account/caches and notify the shell even when JSON/text parsing fails. A403 or503 malformed response must preserve the current workspace, and a late old401 must not clear a new session.

The request fence alone is insufficient: after request resolves, another concurrent current401 can run before the caller's continuation commits cached facts. Capture the original session generation/bearer and call the existing private `assertSession` check immediately before each confirmed cache-writing continuation. A shared check keeps the existing SESSION_EXPIRED contract consistent without adding a new public interface or retry policy. Bootstrap, settings, product reads/saves, sold-out state, categories, pending payments and final image confirmation are the observed affected sites. Existing order generations already protect order/count caches.

## Evidence and lifecycle

The pinned old archive retains earlier upload/query/auth and finance/settings Red/Green runs, failed fixtures and raw receipts. Its 59 affected Node tests passing did not cover the two newly reproduced review findings. New public-seam Red logs reproduce malformed401 cleanup, bootstrap old account, six additional cache locations and final upload private-URL restoration. The test payloads match server response shapes and execute no external fetches. Adjacent checks cover malformed403/503, late malformed401 after new login and concurrently approved replacement login.

Fresh actual UI1 uses the unchanged backend candidate binaries, current PC bytes and an owned loopback/SCRAM PostgreSQL instance with synthetic identities/providers. Record backend SHA separately from frontend candidate/source manifest, observed HTTP/SQL facts, UI state, failed checks and bounded cleanup. No production/cloud connection and no real-money claim. Browser interactions use the enabled CUA surface.

Finish only the necessary local checks/review and browser main/error paths, then form a local candidate with full SHA and clean state. Verify that exact SHA read-only in another clean detached worktree. Dependency integration and deployed Cloud UAT remain later gates. The historical remote review refusal is evidence of that old run, not a prohibition on the current supported local review route.

No separate business refactor is needed. The only small extraction is the identical session check required by the confirmed commit sites; all public behavior and existing assertions remain unchanged.
