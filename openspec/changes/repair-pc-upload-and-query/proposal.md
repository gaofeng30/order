# Change: Close the confirmed PC UAT repair scope

## Why

Existing PC behavior diverges from confirmed server facts: image upload uses an obsolete multipart request; late order and finance reads replace the current selection; failed settings reads lack recovery; static account identity survives login changes. Review of the pinned historical draft also reproduced current401 cleanup being skipped on a malformed body, and successful outer-await continuations restoring authorized caches after that cleanup.

## What Changes

- Consume the approved image intent, raw PUT and JSON confirmation protocol with one content identity, preserving private storage and phase-failure shields.
- Preserve the latest lane/date/search result, authoritative six-state global counts and existing immutable order notes, flavors and payable line subtotals.
- Bind finance amounts, rows, tabs and CSV filenames to the selected server-date query; show unavailable facts and visible retry on read failure. Restore settings recovery without invented values.
- Render the authenticated account identity and existing visible server logout. Clear current401 authorization independently of body parsing; preserve403/network/5xx semantics. Check the original session immediately before bootstrap, settings, catalog, pending-payment, sold-out and image cache commits.
- Preserve all twelve routes, existing public API and business rules. No automatic retries, new backend capability, production configuration, Mini Program edits or canonical CaseID promotion.

## Ownership and dependency

Gate W3 / local browser UI1. Sole writer: branch `codex/order-pc-uat-closeout`, worktree `/Users/vivix/.codex/worktrees/order-pc-uat-closeout/order`. Base and unchanged backend dependency: `e95b8f01f6aa38871b87d4a0f2e5d8595be60141`, already independently verified locally. Integration still requires the backend dependency to be integrated, exact PC candidate verification and applicable integration authorization.

The previous 14-file draft was adopted byte-identically from detached archive snapshot `b1e29343f121dfe2cad094c4d655e02e9b2ae9ba`; that archive commit is not a candidate. Historical frontend baseline was `affb06cafeac5a922b7f436837337eeae7e7079a`. Original main, archived worktree and other writer/index state are protected.

Owned paths:

- `apps/web-admin/{app.css,app.js,index.html,data/api.js,pages/finance.js,pages/orders.js,pages/settings.js}`
- `apps/web-admin/tests/{http-contract.test.js,transactions-pages.test.js,composed-ui1-pc05-pc07-pc12-closure-runner.mjs}`
- This change's `proposal.md`, `design.md`, `tasks.md` and `specs/pc-upload-consumer/spec.md`.

Temporary evidence lives outside tracked source in the current task artifact directory. Backend, Mini Program, canonical matrix and authoritative baseline specs are read-only. Routine implementation, local review and UAT work are already authorized; no stage permission prompt is required. Local review uses the repository's current code-review skill and does not replay the historical refused remote review service.

## Delivery boundary

The PC candidate has not yet been created. Current passing tests do not prove browser acceptance, clean detached exact-SHA verification, main integration or deployed Cloud UAT. Actual CloudBase PG principal/TLS, CloudRun identity adapter and real WeChat/payment gates remain separate deployment dependencies; local synthetic providers never satisfy those gates.
