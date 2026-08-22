# Mini-program UI runtime gates

This runbook establishes UI1 through a real locked Chromium process plus a non-WeChat mini-program simulator, and provides a UI2 receipt path through WeChat Developer Tools. A Node page harness, static WXML check, test name, or lower-level result is not UI1/UI2 evidence.

## Scope and version boundary

The current executable slice covers:

1. anonymous cold start into the user home without a phone-authorization control;
2. entry into the menu;
3. a real loopback catalog `503`, rendered error, user retry, and recovered `200` catalog;
4. category switching and opening the rendered product-selection sheet.

It does not prove real identity, availability, quote, order creation, payment, refund, fulfillment, experience build, device, production, or UI3 behavior.

## UI1: browser/simulator

Prerequisites: the repository-supported Node/npm toolchain and network access for the one-time locked browser download.

```bash
cd tools/miniprogram-ui
npm ci
npm run browser:install
npm run ui1
```

`package-lock.json` pins the runner and simulator dependencies. `browser:install` installs Playwright's exact full Chromium revision below `tools/miniprogram-ui/node_modules`; `ui1` refuses to fall back to an arbitrary system browser.

The runner loads the current `app.js`, page definitions, WXML, and component definitions. Assertions observe rendered text/class and real touch events in Chromium. The catalog store is not mocked: `wx.request` crosses a loopback HTTP fixture on `127.0.0.1:8080`, whose first catalog response is `503` and later responses are valid `200` data.

PASS requires `UI1_RESULT {"status":"PASS","scenarios":3}` and exit `0`. Port `8080` already in use, a missing locked browser, compile errors, scenario failures, or a nonzero exit are FAIL, not UI1.

## UI2: WeChat Developer Tools receipt

Prerequisites:

- WeChat Developer Tools installed at the standard macOS path, or `WECHAT_DEVTOOLS_CLI` set to its CLI;
- CLI login is true;
- the logged-in account has developer permission for the `project.config.json` project;
- loopback ports `8080` (catalog fixture) and `19420` (Developer Tools automation) are free.

Run:

```bash
cd tools/miniprogram-ui
npm ci
npm run ui2
```

Set `MINIPROGRAM_UI_RECEIPT=/absolute/path/receipt.json` to select the receipt path. Otherwise the runner writes `tools/miniprogram-ui/receipts/ui2-latest.json`; the directory is Git-ignored. The receipt contains only versions, scenario statuses, environment class, source HEAD, unverified boundaries, and a sanitized failure/recovery contract. It never records account identifiers, login/session material, request bodies, secrets, or reusable credentials.

PASS requires all three scenarios in the receipt and `UI2_RESULT {"status":"PASS",...}`. Missing developer permission is `BLOCKED_EXTERNAL`, not UI2. Recovery is singular: the mini-program AppID administrator grants the currently logged-in account developer access, `cli islogin` is reconfirmed, and the identical `ui2` command is rerun.

The runner does not invoke preview, upload, deploy, or any external platform write.

## Exact-SHA verification

In a clean detached worktree at the submitted candidate SHA, rerun the UI1 install and command above. UI2 is rerun only when its external prerequisites are present; otherwise preserve the exact `BLOCKED_EXTERNAL` receipt and recovery condition. Any source, runner, scenario, project configuration, browser revision, developer-tools version, account permission, or candidate SHA change invalidates prior evidence.
