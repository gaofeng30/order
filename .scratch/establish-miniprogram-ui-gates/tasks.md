# establish-miniprogram-ui-gates evidence

## Checklist

- [x] Confirm exact base SHA and read applicable rules, quality contract, PRD sections 10/14, and required Skills.
- [x] Declare scope, W/UI Gate, external assets, non-goals, acceptance commands, and public TDD seams.
- [x] Red: cold-start anonymous browse fails because the UI1 runner does not exist.
- [x] Green: cold-start anonymous browse runs in the real browser/simulator.
- [x] Red/Green: network error is visible and user retry recovers to rendered catalog.
- [x] Red/Green: category and product-selection interactions run from rendered UI.
- [x] Refactor: rerun the same UI1 scenarios and affected regression/static checks.
- [x] UI2: emit a sanitized `BLOCKED_EXTERNAL` receipt after the developer-tools permission probe failed before page compilation.
- [x] Two-axis Standards/Spec review found and drove fixes for the real launch route, UI2 external-asset receipts, and vulnerable transitive dependencies; final candidate is reviewed again after commit.
- [ ] Commit candidate and rerun declared gates in a clean detached worktree at the exact SHA.

## Evidence

Evidence entries are appended only after the command/action actually runs.

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: 586180ff04b9aebd31e0939bb3ca08932c743fa4
phase: red
command_or_action: npm --prefix tools/miniprogram-ui audit --json
exit_result: exit-1
sanitized_summary: first candidate reported 17 vulnerabilities: 3 critical, 6 high, 7 moderate, and 1 low
artifact_or_environment: official miniprogram-simulate@1.6.2 and miniprogram-automator@0.12.1 dependency tree
unverified_boundary: no downgrade or forced audit rewrite was accepted; the first candidate is invalid
external_asset:
  owner: quality-infrastructure writer
  missing: lockfile-compatible remediation for stale transitive dependencies
  recovery: apply exact compatibility-tested overrides, clean-install, rerun audit, UI1, UI2 probe, and repository regression gates
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: 586180ff04b9aebd31e0939bb3ca08932c743fa4
phase: red
command_or_action: npm --prefix tools/miniprogram-ui run ui1
exit_result: exit-1
sanitized_summary: the new cold-start assertion failed because the runner had no launch-page definition and therefore skipped the configured first route
artifact_or_environment: locked Chromium 151.0.7922.34 and actual pages/launch/launch.wxml
unverified_boundary: actual launch-to-user-to-home navigation had not run
external_asset:
  owner: quality-infrastructure writer
  missing: launch-page registration in the UI1 scenario
  recovery: register the actual launch page, drive its rendered user entry, and rerun the identical command
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: external-post-commit
phase: green
command_or_action: clean npm ci; npm audit --json; npm run browser:install; npm run ui1; npm run ui2
exit_result: audit-0; UI1-3-of-3; UI2-BLOCKED_EXTERNAL
sanitized_summary: seven exact overrides removed all 17 audit findings without downgrading the official tools; Chromium ran launch-to-user-to-home-to-menu plus 503 retry and menu interaction; UI2 emitted the expected permission receipt
artifact_or_environment: order-miniprogram-ui-gates@1.0.0, miniprogram-simulate@1.6.2, miniprogram-automator@0.12.1, Chromium 151.0.7922.34, Developer Tools 36.6.0
unverified_boundary: UI2 pages did not compile or run because project developer permission remains absent
external_asset:
  owner: mini-program AppID administrator
  missing: developer permission for the currently logged-in WeChat Developer Tools account
  recovery: grant that account developer access, confirm cli islogin, then rerun npm --prefix tools/miniprogram-ui run ui2
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: external-post-commit
phase: refactor
command_or_action: mini-program 65/65; OpenSpec all strict 20/20; JS and JSON static; Go format/test/race/vet/build/smoke
exit_result: PASS
sanitized_summary: all governance-infrastructure regression gates passed after launch-route, receipt, and audit remediation
artifact_or_environment: final owned writer tree before the replacement candidate commit
unverified_boundary: clean detached exact-SHA verification and UI2 remain outstanding
external_asset:
  owner: mini-program AppID administrator for optional UI2
  missing: project developer permission
  recovery: grant permission and rerun the identical UI2 command
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: not-yet-created
phase: green
command_or_action: npm --prefix tools/miniprogram-ui run ui1
exit_result: exit-0
sanitized_summary: locked Chromium ran cold anonymous browse, recoverable network failure, category switch, and rendered product-selection sheet; 3/3 passed
artifact_or_environment: order-miniprogram-ui-gates@1.0.0, miniprogram-simulate@1.6.2, Chromium 151.0.7922.34
unverified_boundary: simulator evidence does not prove WeChat Developer Tools, experience build, device, real account, payment, or production behavior
external_asset:
  owner: N/A
  missing: N/A
  recovery: npm ci then npm run browser:install and npm run ui1 from tools/miniprogram-ui
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: not-yet-created
phase: refactor
command_or_action: npm --prefix tools/miniprogram-ui run ui1
exit_result: exit-0
sanitized_summary: warning-free runner rerun passed the identical 3/3 browser/simulator scenarios
artifact_or_environment: order-miniprogram-ui-gates@1.0.0, miniprogram-simulate@1.6.2, Chromium 151.0.7922.34
unverified_boundary: WeChat Developer Tools and higher platform behavior remain outside UI1
external_asset:
  owner: N/A
  missing: N/A
  recovery: npm ci, install the locked browser, and rerun ui1
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: not-yet-created
phase: writer
command_or_action: npm --prefix tools/miniprogram-ui run ui2
exit_result: BLOCKED_EXTERNAL
sanitized_summary: developer-tools permission probe confirmed the logged-in account is not a developer for the configured project; no UI2 page compiled or ran
artifact_or_environment: sanitized receipt at tools/miniprogram-ui/receipts/ui2-latest.json, excluded from Git
unverified_boundary: all UI2 scenario behavior
external_asset:
  owner: mini-program AppID administrator
  missing: developer permission for the currently logged-in WeChat Developer Tools account
  recovery: grant that account developer access, confirm cli islogin, then rerun npm --prefix tools/miniprogram-ui run ui2
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: not-yet-created
phase: red
command_or_action: npm --prefix tools/miniprogram-ui run ui2
exit_result: exit-1
sanitized_summary: the UI2 command was absent, so no WeChat Developer Tools run or receipt occurred
artifact_or_environment: local developer tools 36.6.0 with CLI login already confirmed
unverified_boundary: all UI2 behavior
external_asset:
  owner: quality-infrastructure writer
  missing: UI2 automator and receipt runner
  recovery: add the isolated UI2 runner and rerun the identical command without preview or upload
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: NOT_RUN
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: not-yet-created
phase: red
command_or_action: npm --prefix tools/miniprogram-ui run ui1
exit_result: exit-254
sanitized_summary: package.json was absent, so the requested UI1 runner could not start
artifact_or_environment: local writer worktree at exact base
unverified_boundary: no browser, simulator, page, or scenario ran
external_asset:
  owner: quality-infrastructure writer
  missing: repository UI1 runner
  recovery: add the locked isolated runner and rerun the identical command
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1-partial
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: not-yet-created
phase: green
command_or_action: npm --prefix tools/miniprogram-ui run ui1
exit_result: exit-0
sanitized_summary: real loopback 503 rendered the error; user retry crossed the same transport and a 200 response rendered the recovered catalog, 2/2 scenarios passed
artifact_or_environment: locked Chromium 151.0.7922.34, actual menu WXML/page definition, loopback fixture
unverified_boundary: menu selection interaction and WeChat runtime are not yet covered
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun the identical command after a clean install
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: external-post-commit
phase: writer
command_or_action: UI1 3/3; existing mini-program 65/65; OpenSpec all strict 20/20; JS and JSON static; Go format/test/race/vet/build/smoke; owned-path and sensitive-value audits
exit_result: PASS
sanitized_summary: all applicable writer gates passed; change-specific OpenSpec validation is N/A because the user-owned path contract forbids openspec edits and explicitly requires a scratch ticket; C9/T10/V8/R9=36 with hard blockers 0
artifact_or_environment: final owned candidate tree before immutable commit
unverified_boundary: V8 requires clean detached exact-SHA verification; UI2 is BLOCKED_EXTERNAL on missing project developer permission; no preview, upload, integration, push, deploy, real order/payment, production, or UI3 proof
external_asset:
  owner: mini-program AppID administrator for optional UI2
  missing: developer permission for the currently logged-in WeChat Developer Tools account
  recovery: grant that account developer access, confirm cli islogin, then rerun npm --prefix tools/miniprogram-ui run ui2
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1-partial
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: not-yet-created
phase: red
command_or_action: npm --prefix tools/miniprogram-ui run ui1
exit_result: exit-1
sanitized_summary: Chrome rendered the catalog failure and accepted the retry touch, but recovery stayed in error because no loopback transport existed
artifact_or_environment: locked Chromium 151.0.7922.34 and actual menu WXML/page definition
unverified_boundary: no successful catalog response or ready-state menu rendered
external_asset:
  owner: quality-infrastructure writer
  missing: loopback catalog fixture and browser wx.request adapter
  recovery: add the isolated loopback transport and rerun the identical command
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1-partial
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: not-yet-created
phase: green
command_or_action: npm --prefix tools/miniprogram-ui run ui1
exit_result: exit-0
sanitized_summary: locked Chromium 151.0.7922.34 rendered the actual home page; anonymous cold start and touch navigation passed 1/1
artifact_or_environment: order-miniprogram-ui-gates@1.0.0 with miniprogram-simulate@1.6.2
unverified_boundary: network recovery and menu interaction are not implemented yet; no WeChat runtime ran
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun the identical command after a clean install
```
