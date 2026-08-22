# establish-miniprogram-ui-gates

## Status

APPROVED by the delegated user instruction on 2026-08-23.

## Goal

Establish a repository-reproducible mini-program UI1 gate that actually runs the current user-facing slice and its recoverable network error in a non-WeChat simulator. When the local WeChat Developer Tools and project authorization are available, run the same behavior contract at UI2 and emit a sanitized receipt.

## Gate declaration

- `gate_type`: `W1` — this change adds isolated quality-runner behavior and does not change a product page, public API, persisted result, money, authorization, or concurrency invariant.
- `ui_level_target`: `UI1`.
- `ui_level_actual`: `UI1` after the recorded Green and Refactor commands passed 3/3 scenarios in locked Chromium.
- Optional higher evidence: `UI2`; current status is `BLOCKED_EXTERNAL` because the logged-in WeChat Developer Tools account is not a developer for the configured project. It is PASS only after permission is restored and the UI2 receipt records all declared scenarios.
- owner: independent quality-infrastructure writer session for `establish-miniprogram-ui-gates`.
- worktree: `/Users/vivix/.codex/worktrees/b683/order` at detached HEAD.
- `base_sha`: `2c549f2c413a06ec160b4e886fb16cd62e18176f`.
- `candidate_sha`: external post-commit value; the immutable commit cannot contain its own SHA.
- owned paths:
  - `.scratch/establish-miniprogram-ui-gates/**`
  - `tools/miniprogram-ui/**`
  - `docs/quality/miniprogram-ui-runbook.md` only after a runner has actually passed.
- read-only shared contracts:
  - `AGENTS.md`
  - `CONTEXT.md`
  - `docs/quality/change-quality-gates.md`
  - `docs/product/online-ordering-system-prd-0818.md` sections 10 and 14
  - `project.config.json`
  - `apps/wechat-miniprogram/**`
- dependencies: Node.js, a locked UI1 browser/simulator dependency set under `tools/miniprogram-ui`; UI2 additionally requires the locally installed WeChat Developer Tools, CLI login, and access to the configured project AppID.
- non-goals: product code or configuration changes; real order/payment/fulfillment claims; UI3; preview/upload/deploy; external platform writes.

## Pre-agreed TDD seams

The delegated instruction already fixes the public seam, so no additional product decision is needed:

1. UI1 observes the actual page definitions and WXML through rendered DOM in a real Chrome process backed by a non-WeChat mini-program simulator. Assertions use visible text/class and user clicks, not private page methods, the existing Node page harness, static WXML inspection, or test names.
2. Network behavior crosses a real loopback HTTP fixture boundary. The runner observes loading/error/retry/ready in rendered UI; it does not replace the catalog store with a mock.
3. UI2 observes the same visible contract through WeChat Developer Tools automation and writes a sanitized receipt containing tool/runner versions, scenario outcomes, environment class, and unverified boundaries. It never records login/session material, account identifiers, request bodies, or other secrets.

## Required scenarios and current version boundary

1. Cold-start anonymous browsing: launch into the current user home without a phone authorization prompt, then enter the menu.
2. Recoverable network error: the real catalog request fails, the rendered menu shows `目录加载失败`, the user taps `重试`, and the rendered catalog becomes ready.
3. Menu interaction: after recovery, the user changes category and triggers the current product-selection interaction.

The exact base only integrates anonymous catalog browsing and local cart-selection behavior. This gate therefore does not claim real identity, availability, quote, order creation, payment, refund, fulfillment, production, or UI3.

## Minimal success standard

- A locked, isolated runner under `tools/miniprogram-ui` launches a real browser and passes all three UI1 scenarios from a clean install.
- The Red failure, Green pass, and Refactor rerun are recorded in `tasks.md` using the repository evidence fields.
- WeChat Developer Tools `36.6.0`, CLI login, and project configuration are present. The permission probe found that the logged-in account is not a developer for this project, so UI2 records `BLOCKED_EXTERNAL` with one recovery path: the AppID administrator grants developer access, then the same `ui2` command is rerun.
- The diff stays inside owned paths, contains no sensitive data, passes applicable repository gates, is committed, and is rerun in a clean detached worktree at the exact candidate SHA.
