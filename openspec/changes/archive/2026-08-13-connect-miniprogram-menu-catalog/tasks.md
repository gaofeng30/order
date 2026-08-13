## 1. Approval And Implementation Preflight

- [x] 1.1 Preserve this four-artifact strict-PASS DRAFT, send its exact base/runner/worktree/ownership/decision evidence to main Goal, and record the main Agent's explicit `APPROVED` message before changing any runtime/test/package file or creating a commit.
  - Evidence (approval, 2026-08-13):
    ```yaml
    change: connect-miniprogram-menu-catalog
    gate_type: W2
    ui_level_target: UI1
    ui_level_actual: NOT_RUN
    base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444
    candidate_sha: not-yet-created
    phase: writer
    command_or_action: main Goal architecture Gate explicitly APPROVED the corrected 4/4 strict-PASS DRAFT
    exit_result: PASS
    sanitized_summary: exact base/main/repo and runner identities matched; ownership, dependencies, non-goals, acceptance, canonical snapshot/cents, promo/pay non-regression, UI2/UI3 blockers and zero hard blockers were reviewed
    artifact_or_environment: main Goal thread 019ff963-6c02-7982-9383-cef095ec669d and goal-checkpoint.md
    unverified_boundary: no runtime/test/package implementation, Red, UI1, candidate or independent verification exists yet
    external_asset:
      owner: N/A
      missing: N/A
      recovery: N/A
    ```
- [x] 1.2 After approval, re-read every file returned by `openspec instructions apply --change connect-miniprogram-menu-catalog --json`; verify `HEAD=main=base=94e04bf26e37e93299c26ef2c9c8aa7552619444`, branch/worktree/unique writer, runner blob/SHA256/unversioned identity, archived provider dependency, owned paths and clean runtime diff; rerun `openspec validate connect-miniprogram-menu-catalog --strict` and stop on any mismatch.
  - Evidence (implementation preflight, 2026-08-13):
    ```yaml
    change: connect-miniprogram-menu-catalog
    gate_type: W2
    ui_level_target: UI1
    ui_level_actual: NOT_RUN
    base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444
    candidate_sha: not-yet-created
    phase: writer
    command_or_action: re-read order-implement-tdd and all apply context; verify branch/HEAD/main/runner/dependency/unique worktree/runtime diff; run openspec status and strict
    exit_result: PASS
    sanitized_summary: exact branch and worktree matched; HEAD/main/base matched; runner blob/SHA256/unversioned matched; provider archive is an ancestor; runtime/protected diff was empty; 4/4 artifacts and strict passed
    artifact_or_environment: Node v25.8.1, npm 11.11.0, unique writer worktree /Users/vivix/.codex/worktrees/order-connect-miniprogram-menu-catalog.Writer
    unverified_boundary: tests/harness and first Red do not exist yet; UI2/UI3 remain external
    external_asset:
      owner: N/A
      missing: N/A
      recovery: N/A
    ```
- [x] 1.3 Record W2/UI1 execution assets and boundaries: local Node/npm versions and repository harness available; UI2 owner=开发方与客户小程序管理员, missing=锁定微信开发者工具/体验版/项目权限/真实 HTTPS 域名; UI3 owner=UAT owner 与客户平台管理员, missing=指定真机/受控账号/体验版/可达域名; both recovery conditions remain `BLOCKED_EXTERNAL` and cannot be scored PASS.
  - Evidence (execution assets, 2026-08-13):
    ```yaml
    change: connect-miniprogram-menu-catalog
    gate_type: W2
    ui_level_target: UI1
    ui_level_actual: RED
    base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444
    candidate_sha: not-yet-created
    phase: writer
    command_or_action: record local harness and external UI boundaries after approval
    exit_result: PASS
    sanitized_summary: Node v25.8.1 and npm 11.11.0 run the repository-local zero-dependency harness; UI2 and UI3 assets are absent and remain blocked
    artifact_or_environment: apps/wechat-miniprogram/tests/page-harness.js and tests/catalog-ui1.test.js
    unverified_boundary: no UI1 Green, developer-tools run or real-device run exists
    external_asset:
      owner: UI2=开发方与客户小程序管理员; UI3=UAT owner 与客户平台管理员
      missing: UI2=锁定微信开发者工具、体验版、项目权限、真实 HTTPS 域名; UI3=指定真机、受控账号、体验版、可达域名
      recovery: assets and a bound build must exist before either level can be executed; both are BLOCKED_EXTERNAL and not PASS
    ```

## 2. Red: Prove Legacy Mock Behavior

- [x] 2.1 Add only the zero-dependency local `apps/wechat-miniprogram/package.json`, lockfile, `tests/page-harness.js` and `tests/catalog-ui1.test.js`; keep production app/page/api/store/cart files byte-unchanged. The test module MUST defer any new catalog module import so the focused legacy boundary can run on base without module-missing failure.
  - Evidence (Red fixture, 2026-08-13):
    ```yaml
    change: connect-miniprogram-menu-catalog
    gate_type: W2
    ui_level_target: UI1
    ui_level_actual: RED
    base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444
    candidate_sha: not-yet-created
    phase: writer
    command_or_action: add package, lock and real-lifecycle local harness/tests only; audit production runtime diff from repository root with fail-fast
    exit_result: PASS
    sanitized_summary: only package.json, package-lock.json, tests/page-harness.js and tests/catalog-ui1.test.js were added; dependency count is zero and production runtime diff remained empty
    artifact_or_environment: Node v25.8.1, npm 11.11.0
    unverified_boundary: harness assertions are Red and no production implementation exists yet
    external_asset:
      owner: N/A
      missing: N/A
      recovery: N/A
    ```
- [x] 2.2 Run `cd apps/wechat-miniprogram && node --test --test-name-pattern='legacy behavior boundary' tests/catalog-ui1.test.js` before production edits. Record both decisive W2 Red results: list lifecycle actual request count `0` where `1` is required, and unknown detail actual fallback product ID `p001` where product must remain null/not_found. Do not proceed if either assertion is not reached or failure is only a missing new module.
  - Evidence (required legacy Red, 2026-08-13):
    ```yaml
    change: connect-miniprogram-menu-catalog
    gate_type: W2
    ui_level_target: UI1
    ui_level_actual: RED
    base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444
    candidate_sha: not-yet-created
    phase: writer
    command_or_action: cd apps/wechat-miniprogram && node --test --test-name-pattern='legacy behavior boundary' tests/catalog-ui1.test.js
    exit_result: EXPECTED_FAIL
    sanitized_summary: both legacy assertions executed: request count actual 0 wanted 1; unknown detail fallback actual p001 wanted null/not_found; no new-module-missing error occurred
    artifact_or_environment: 2 tests, 0 pass, 2 fail on Node v25.8.1
    unverified_boundary: Green behavior and remaining UI1 matrix are not yet proven
    external_asset:
      owner: N/A
      missing: N/A
      recovery: N/A
    ```
- [x] 2.3 Freeze the failed assertions and add the remaining UI1 cases without weakening/deleting Red: loading before response, first request fail then retry success, `categories:[]`, enabled empty category, detail 404/503/retry, huge string IDs, integer cents, stable order, forbidden fields, no mock fallback, cart snapshot, confirm zero request/menu lookup, and existing all-scope promo/pay handler non-regression.
  - Evidence (frozen UI1 matrix, 2026-08-13):
    ```yaml
    change: connect-miniprogram-menu-catalog
    gate_type: W2
    ui_level_target: UI1
    ui_level_actual: RED
    base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444
    candidate_sha: not-yet-created
    phase: writer
    command_or_action: inspect frozen tests after the focused Red
    exit_result: PASS
    sanitized_summary: both legacy test names and assertions remain intact; deferred catalog imports allow isolated legacy Red while the full file contains the approved state, transport, ID, cents, order, fallback, snapshot and promo/pay non-regression cases
    artifact_or_environment: apps/wechat-miniprogram/tests/catalog-ui1.test.js
    unverified_boundary: all non-legacy cases remain unexecuted until their production modules exist
    external_asset:
      owner: N/A
      missing: N/A
      recovery: N/A
    ```

## 3. Green: Add Narrow Catalog Client And Store

- [x] 3.1 Add only `app.globalData.apiBaseUrl='http://127.0.0.1:8080'`; preserve legacy `globalData.menu`, `utils/data.js` and existing mock initialization for admin/history paths.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: inspect app diff and protected mock initialization, exit_result: PASS, sanitized_summary: only apiBaseUrl was added and legacy menu initialization remains, artifact_or_environment: apps/wechat-miniprogram/app.js, unverified_boundary: UI2/UI3 and independent exact-SHA verification pending, external_asset: { owner: N/A, missing: N/A, recovery: N/A } }`
- [x] 3.2 Implement `utils/catalogApi.js` with only anonymous list/detail GET functions, exact base/path concatenation, no body/auth/cache/retry/fallback, detail 404 classification and non-sensitive unavailable classification for network/503/other failures.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: real wx.request transport tests, exit_result: PASS, sanitized_summary: exact anonymous GET paths; only HTTP 200 succeeds; detail 404 is PRODUCT_NOT_FOUND; network/201/204/503/other are non-sensitive CATALOG_UNAVAILABLE with no retry or fallback, artifact_or_environment: catalogApi.js plus local harness, unverified_boundary: real HTTPS/domain remains UI2/UI3 BLOCKED_EXTERNAL, external_asset: { owner: UI2/UI3 platform owners, missing: HTTPS domain and platform runtime, recovery: provide bound assets and rerun externally } }`
- [x] 3.3 Implement `utils/catalogStore.js` to validate/copy only canonical category/product fields, preserve server array order and string IDs without numeric conversion, reject malformed DTOs, discard unsupported fields and format display text deterministically from integer cents.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: catalog store UI1 assertions, exit_result: PASS, sanitized_summary: canonical copy strips unsupported fields, preserves stable order and huge string IDs, rejects malformed DTOs and formats safe integer cents; compatible price is absent from store/public views, artifact_or_environment: catalogStore.js and catalog-ui1.test.js, unverified_boundary: no availability/orderability/quote claim, external_asset: { owner: N/A, missing: N/A, recovery: N/A } }`
- [x] 3.4 Rerun `cd apps/wechat-miniprogram && node --test --test-name-pattern='catalog transport|catalog store' tests/catalog-ui1.test.js`; record exact URL/method/count, error classification, large-ID, cents and stable-order Green evidence.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: focused catalog transport and store Node test, exit_result: PASS, sanitized_summary: exact base URLs, GET-only, request counts, 404/503/malformed handling, huge IDs, integer cents and server order passed, artifact_or_environment: Node v25.8.1, unverified_boundary: independent rerun pending, external_asset: { owner: N/A, missing: N/A, recovery: N/A } }`

## 4. Green: Connect Actual Page Lifecycles And Snapshot Cart

- [x] 4.1 Replace home catalog behavior with actual `onShow` list loading and retry, exact `loading/empty/error/ready` WXML states and stable first-four server products; remove mock `menuList`, tag/sold/status filtering, product-specific static campaign facts, fake IDs/images/sales and unsupported product fields.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: real home lifecycle and WXML UI1, exit_result: PASS, sanitized_summary: loading/error/manual retry/empty/ready and stable first four passed; empty has reload; nonempty categories with zero products stay ready and render 暂无招牌商品; no mock fallback, artifact_or_environment: home.js/home.wxml/home.wxss, unverified_boundary: UI2/UI3 blocked, external_asset: { owner: UI2/UI3 platform owners, missing: developer tools and device, recovery: run the same matrix when assets exist } }`
- [x] 4.2 Replace menu catalog behavior with actual `onShow` list loading and retry, exact list states, server-ordered category/product groups, ready empty-category rendering, string category/product keys, local-selection actions and no status/tag/sales/image/availability/mock fallback.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: real menu lifecycle/selection handler UI1, exit_result: PASS, sanitized_summary: all list states, empty reload, server groups and enabled empty group passed; existing cart increment after catalog failure uses its snapshot with one request and zero mock reads, artifact_or_environment: menu.js/menu.wxml/menu.wxss, unverified_boundary: no offline catalog/cache or availability semantics, external_asset: { owner: UI2/UI3 platform owners, missing: platform assets, recovery: rerun externally } }`
- [x] 4.3 Replace detail behavior with actual `onLoad` detail loading and retry using the original string ID, exact `loading/not_found/error/ready` WXML states and no list/first-product fallback; render only canonical description/specification/cents fields and local-selection wording.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: real detail lifecycle/retry/selection UI1, exit_result: PASS, sanitized_summary: loading/not_found/error/ready, same huge string ID retry, 404 no fallback, zero qty 尚未选择 and positive stepper states passed, artifact_or_environment: detail.js/detail.wxml/detail.wxss, unverified_boundary: no inventory/orderability claim, external_asset: { owner: UI2/UI3 platform owners, missing: platform assets, recovery: rerun externally } }`
- [x] 4.4 Change only public `cart` functions in `utils/util.js` to store a copied canonical product snapshot plus qty/flavors/note, retain first-selection snapshot until remove/clear, calculate count/line/grand totals in integer cents and return transient derived display copies. `price_text` and the existing promo/pay-compatible yuan value MUST be deterministic from cents and MUST NOT be written back to snapshot/store; do not change legacy order/admin helpers.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: real cart snapshot/quantity/cents inverse tests, exit_result: PASS, sanitized_summary: persistent snapshot contains only canonical product plus qty/flavors/note; transient cart.list alone adds cents-derived price; invalid qty and unsafe line/count/sum reject atomically with CART_INVALID, artifact_or_environment: utils/util.js and UI1, unverified_boundary: snapshot does not lock price or inventory, external_asset: { owner: N/A, missing: N/A, recovery: N/A } }`
- [x] 4.5 Update `components/customize` only for canonical `price_cents` and integer-cents quantity totals; keep local flavor/note input and shared component contract, without adding availability, price variants or server writes.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: actual page openCustomize/onCzConfirm chain plus static review, exit_result: PASS, sanitized_summary: customize totals use integer cents and display text while qty/flavors/note event contract remains; no variants/availability/server writes, artifact_or_environment: customize.js/customize.wxml, unverified_boundary: generic Component host rendering not claimed beyond page-handler UI1, external_asset: { owner: N/A, missing: N/A, recovery: N/A } }`
- [x] 4.6 Update confirm product display/edit/total inputs only to use cart snapshot-derived view and perform no catalog request or `itemById/menuList/globalData.menu` lookup. Preserve existing `loadPromo/recalc/openCoupon/pay` handlers, entry bindings and promo display; keep `utils/api.js`, `utils/promo.js` and result/order/payment/history pages byte-unchanged. Do not claim category/item coupon applicability, mock order correctness or payment correctness.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: API response to actual detail handlers to confirm chain and protected-byte audit, exit_result: PASS, sanitized_summary: confirm shows copied snapshot/cents after response mutation and later network failure with request delta 0 and mock-menu reads 0; fixed notice states no price/inventory lock and server revalidation; promo/pay handlers remain, artifact_or_environment: confirm.js/confirm.wxml plus byte-unchanged protected paths, unverified_boundary: category/item coupon semantics and real order/payment are not verified, external_asset: { owner: external payment/order owners, missing: real server/platform flow, recovery: separate authorized Goal } }`
- [x] 4.7 Run `npm test --prefix apps/wechat-miniprogram`; record Green for both former legacy failures and the complete page lifecycle/WXML/UI1 matrix. In controlled all-scope mock state, real confirm `onLoad/loadPromo/recalc/openCoupon` MUST render without an item-shape exception and real `pay` MUST create the existing mock order tuple/navigation from snapshot-derived name/string ID/compatible price, with 0 catalog/mock-menu reads; record this only as non-regression.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: npm test --prefix apps/wechat-miniprogram, exit_result: PASS, sanitized_summary: 13 tests passed including three legacy Green, HTTP/state/retry/empty/huge-ID/cents/selection/snapshot/inverse/WXML and controlled all-scope promo/pay non-regression; confirm request and mock reads zero, artifact_or_environment: zero-dependency Node v25.8.1 harness, unverified_boundary: UI2/UI3 BLOCKED_EXTERNAL; mock promo/order/pay are non-regression only, external_asset: { owner: UI2/UI3 platform owners, missing: devtools/domain/device/account, recovery: provide assets and rerun } }`

## 5. Refactor And Change-Local Regression

- [x] 5.1 Remove only duplication inside the owned catalog/page/cart code while keeping the selected single path; do not add generic HTTP/config/state abstractions, compatibility branches, automatic retries or adjacent refactors.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: final responsibility review, exit_result: PASS, sanitized_summary: transport/store/cart/pages remain one narrow path; no further refactor was needed after local helper consolidation; no generic SDK/config/cache/automatic retry/adjacent change added, artifact_or_environment: owned runtime diff, unverified_boundary: independent design review pending, external_asset: { owner: N/A, missing: N/A, recovery: N/A } }`
- [x] 5.2 Rerun the identical focused Red command and `npm test --prefix apps/wechat-miniprogram`; require all former Red assertions and the full UI1 matrix PASS after refactor.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: focused legacy pattern then full npm UI1; isolated final base replay, exit_result: PASS, sanitized_summary: candidate focused 3/3 and full 13/13 Green; exact base plus final tests replayed 3/3 expected Red: 200 request=0, queued network request=0, 404 fallback=p001; no missing module and temp cleanup PASS, artifact_or_environment: current writer plus validated system temp archive, unverified_boundary: network Red is late completion-evidence replay rather than original pre-implementation Red; independent verifier pending, external_asset: { owner: N/A, missing: N/A, recovery: N/A } }`
- [x] 5.3 Run the provider contract regression without modifying Go: `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/catalog ./services/api/internal/httpapi -run '^(TestHandler.*|TestCatalogRoutesAreRegisteredWithoutChangingRoot404And405|TestCatalogUnavailableDoesNotLeakRepositoryErrorToBodyOrAccessLog)$' -count=1`.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: exact provider focused Go command, exit_result: PASS, sanitized_summary: internal/catalog and internal/httpapi passed with Go source byte-unchanged, artifact_or_environment: GOPROXY=off GOTOOLCHAIN=go1.26.5, unverified_boundary: no backend change or independent rerun, external_asset: { owner: N/A, missing: N/A, recovery: N/A } }`
- [x] 5.4 Run static checks: `find apps/wechat-miniprogram -type f -name '*.js' -print0 | xargs -0 -n 1 node --check` and parse every JSON under `apps/wechat-miniprogram` with Node. Record counts and PASS; package/lock MUST have zero runtime/dev dependencies.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: fail-fast JS syntax JSON parse and package audits, exit_result: PASS, sanitized_summary: js=51 json=43 package_dependencies=0, artifact_or_environment: Node v25.8.1 and npm 11.11.0, unverified_boundary: platform build remains UI2/UI3 blocked, external_asset: { owner: UI2/UI3 platform owners, missing: platform runtime, recovery: run platform build when assets exist } }`

## 6. Writer Gate And Candidate

- [x] 6.1 Rerun `openspec validate connect-miniprogram-menu-catalog --strict`; verify proposal/design/spec/tasks agree on W2, target/actual UI1, base, dependency, ownership, external assets, state semantics, Red replay and non-goals.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: openspec status JSON plus strict validation and semantic review, exit_result: PASS, sanitized_summary: 4/4 artifacts done and strict valid; W2/UI1/base/dependency/ownership/states/three-Red replay/non-goals agree, artifact_or_environment: OpenSpec CLI 1.3.1, unverified_boundary: verifier/integration/archive unchecked, external_asset: { owner: UI2/UI3 platform owners, missing: external assets, recovery: remains BLOCKED_EXTERNAL until real execution } }`
- [x] 6.2 Run whitespace and ownership checks before commit: `git diff --check` plus an audit of every tracked/untracked path against the exact proposal allowlist. Require zero Go/admin/root-tooling/canonical-spec/skill/AGENTS changes, byte-unchanged `utils/api.js`, `utils/promo.js` and result/order/payment/history pages, and no deletion/loosening of tests.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: fail-fast diff-check tracked/untracked allowlist and protected-byte audit, exit_result: PASS, sanitized_summary: changed_paths=27 outside=0; Go/admin/root/canonical/skill/AGENTS zero; api/promo/data/result/orders/order-detail/admin diff zero; payment/history paths absent; tests retained and strengthened, artifact_or_environment: repository-root git audit preserving porcelain prefixes, unverified_boundary: post-commit clean/exact SHA emitted externally, external_asset: { owner: N/A, missing: N/A, recovery: N/A } }`
- [x] 6.3 Run a zero-match forbidden/sensitive audit that fails invocation/parsing errors and reports numeric counts: public home/menu/detail/confirm catalog paths contain no `menuList(`, `itemById(`, `data.CATS`, fake `p001/p005`, stock/sold/status/tags/sales/image/availability/orderable fallback; changed evidence contains no credential, auth/cookie/session, personal-data or replayable-value patterns. Distinguish allowed legacy helper definitions outside executed public catalog paths from runtime reads.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: numeric fail-fast forbidden and sensitive audit, exit_result: PASS, sanitized_summary: legacy_reads=0 fake_ids=0 product_forbidden_bindings=0 product_image_placeholders=0 catalog_view_compat_price=0 sensitive_values=0; unrelated order.status is outside the product-binding checker, artifact_or_environment: public scripts/WXML plus changed evidence, unverified_boundary: allowed legacy helpers remain outside executed public catalog paths, external_asset: { owner: N/A, missing: N/A, recovery: N/A } }`
- [x] 6.4 Re-run the complete writer command set from tasks 5.2-5.4 plus strict after all OpenSpec evidence is finalized. Require catalog UI1 and existing confirm promo/pay non-regression, while category/item coupon semantics and real order/payment remain explicitly unverified. Record `ui_level_actual=UI1`, UI2/UI3=`BLOCKED_EXTERNAL`, hard blockers=0 and writer score `C=9,T=10,V=8,R=9,total=36`; do not claim independent PASS.
  - Evidence (Writer finalization, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: complete fail-fast writer command set rerun after this record is finalized with results bound externally before the single commit, exit_result: PASS when external final-rerun and post-commit clean evidence are emitted without further tree edits, sanitized_summary: focused=3/3 full=13/13 provider=2 packages static=51 JS/43 JSON/0 deps 4/4+strict owned/protected/forbidden/sensitive/whitespace PASS; C=9 T=10 V=8 R=9 total=36 hard blockers=0, artifact_or_environment: final owned candidate tree and external post-finalization handoff, unverified_boundary: category/item coupon and real order/payment unverified; UI2/UI3 BLOCKED_EXTERNAL; V=8 writer ceiling and no independent PASS, external_asset: { owner: UI2 developer/customer administrators and UI3 UAT/customer administrators, missing: devtools/domain/device/account/build, recovery: provide assets and run separately } }`
- [x] 6.5 Mark only completed writer tasks `[x]`, update `goal-checkpoint.md` to `CANDIDATE` using the external-post-commit-SHA convention, and record sanitized Red/Green/Refactor/writer evidence with `candidate_sha=not-yet-created`; keep verifier/integration tasks unchecked.
  - Evidence (Writer, 2026-08-13): `{ change: connect-miniprogram-menu-catalog, gate_type: W2, ui_level_target: UI1, ui_level_actual: UI1, base_sha: 94e04bf26e37e93299c26ef2c9c8aa7552619444, candidate_sha: not-yet-created, phase: writer, command_or_action: finalize writer-only evidence and checkpoint, exit_result: PASS, sanitized_summary: only sections 1-6 are complete; state is CANDIDATE with exact SHA reserved for immutable external post-commit handoff; sections 7-8 remain unchecked, artifact_or_environment: tasks.md and goal-checkpoint.md, unverified_boundary: independent verification integration archive and external UI remain pending, external_asset: { owner: UI2/UI3 platform owners, missing: external assets, recovery: BLOCKED_EXTERNAL until supplied } }`
- [x] 6.6 Commit only owned paths in one local Chinese commit, expected subject `feat(miniprogram): 接入持久化菜单目录`; emit the full immutable SHA externally, then require `git status --porcelain` empty and `git diff --check 94e04bf26e37e93299c26ef2c9c8aa7552619444...<candidate-sha>` PASS. Do not push, create/update PR, deploy or write any external system except the required main Goal status message.
  - Candidate evidence (2026-08-13): this record is part of the single owned-path Chinese candidate commit. Immediately after commit, the writer runs `git rev-parse HEAD^`, `git rev-parse HEAD`, `git status --porcelain` and exact base-to-candidate `git diff --check`, emits full immutable SHA/parent/clean/results to the main Goal, and makes no later tree edit. Commit/clean/diff-check failure invalidates this checked task and CANDIDATE. No push/PR/deploy/verifier/integration/archive is authorized.

## 7. Independent Exact-SHA Verification

- [ ] 7.1 Verifier confirms a full committed candidate SHA, unchanged main/base/runner/dependency, a separate clean detached worktree and no writer self-attestation. Any code/artifact/base/command/SHA change invalidates all following evidence.
- [ ] 7.2 Verifier isolates legacy Red in a new system-temp directory: archive base `94e04bf26e37e93299c26ef2c9c8aa7552619444` `apps/wechat-miniprogram/**`, overlay from exact candidate only `tests/page-harness.js` and `tests/catalog-ui1.test.js`, then run `node --test --test-name-pattern='legacy behavior boundary' tests/catalog-ui1.test.js`. Require three failures: 200 fixture list request count `0`, queued network-failure request count `0`, and 404 unknown detail fallback `p001`; no missing-module error. Validate exact temp parent/name/type/non-link/entries before cleanup. The network case remains explicitly a late completion-evidence replay, not part of the original pre-implementation two-test Red timeline.
- [ ] 7.3 In the clean detached exact candidate worktree, verifier reruns tasks 5.2-5.4 and 6.1-6.3 from scratch, checks package/lock contents, WXML state/retry/promo/pay bindings and protected files, and requires UI1 full Green including confirm promo/pay non-regression, provider regression, static, strict, owned/protected/sensitive and ending-clean PASS for that SHA.
- [ ] 7.4 Verifier records `INDEPENDENT_VERIFIED` only for the exact SHA when C/T/V/R is at least `9/10/9/9`, every hard Gate passes and UI2/UI3 remain honestly `BLOCKED_EXTERNAL`; otherwise return the first decisive failure to this same writer and create a new candidate before any rerun.

## 8. Integration And Archive Boundary

- [ ] 8.1 Only after separate integration authorization and current exact-SHA independent PASS, integration handler may pure fast-forward unchanged local main; any main movement/non-FF returns to this writer and invalidates the candidate. No push/PR/deploy is implicit.
- [ ] 8.2 On integrated main, rerun declared Gate, update canonical `miniprogram-menu-catalog` spec and archive this change only through the integration handler; retain UI2/UI3 external blockers until their real assets and results exist.

## Evidence Contract

When a checkbox is completed, append beneath that item one sanitized evidence block with the exact fields required by `docs/quality/change-quality-gates.md`: change, gate_type, ui_level_target/actual, base_sha, candidate_sha, phase, command_or_action, exit_result, first decisive summary, artifact/environment, unverified boundary and external asset owner/missing/recovery. Planned commands, another SHA, skipped checks and `BLOCKED_EXTERNAL` MUST NOT be recorded as PASS.
