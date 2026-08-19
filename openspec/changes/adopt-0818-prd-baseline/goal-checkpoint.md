# Goal checkpoint

## Frozen runner base

| field | value |
| --- | --- |
| goal | 将 0818 PRD §1–§14 收敛为唯一有效一期产品基线 |
| module | `adopt-0818-prd-baseline` |
| lifecycle | `CANDIDATE` |
| repo_sha | `2f2db4a31f66f992997880a02b438c9690bbb845` |
| skill_blob | `0f41f64ad87fc9fd410cb916b4d1562aee92e42f` |
| skill_sha256 | `2781bdda1544106e30e7483c4b500d611df85c79753cc1bd3b717d91d1edaac8` |
| runner_version | `unversioned` |
| owner | branch `codex/adopt-0818-prd-baseline`, worktree `/Users/vivix/.codex/worktrees/order-adopt-0818-prd-baseline.Writer` |
| dependency | none |
| blocker | none for this W0; P1–P5 only block the targeted downstream modules below |
| candidate_sha | external post-commit evidence; bind the immutable full SHA through Harness and handoff immediately after commit |
| integrated_sha | none |
| archive_sha | none |
| error_fingerprint | none |
| repeat_count | `0` |
| next | commit only owned paths, bind the exact full SHA, then hand off to a separate clean detached verifier |

## Boundary

- `gate_type=W0`; `ui_level_target=UI0`; `ui_level_actual=UI0` from the focused static content/structure check.
- External assets: none; owner/missing/recovery all `N/A`.
- Owned paths: `docs/product/online-ordering-system-prd.md`, `openspec/changes/adopt-0818-prd-baseline/**`.
- Read-only shared contracts: `docs/product/online-ordering-system-prd-0818.md`, `docs/product/online-ordering-system-prd-0818-review.md`, `openspec/specs/mvp-product-baseline/spec.md`.
- Non-goals and the single ACCEPT/REJECT verdict are frozen in `proposal.md`.
- This module uses the frozen runner above. A runner identity mismatch must be recorded and handled under the existing module contract; it cannot silently change the rules.

## Current verdict

- Main Agent review plus current strict PASS approved the DRAFT; Harness recorded `DRAFT → APPROVED → IMPLEMENTING`.
- Red PASS: the unchanged old PRD made `checks/verify_baseline.py` exit `1` because it contains `985` lines rather than an at-most-40-line thin pointer.
- Green PASS: the exact same checker passed after the minimal pointer replacement; Refactor PASS after deduplication reports `pointer_lines=11 invariants=16 removed_requirements=6 blockers=5`.
- Writer Gate PASS prepared on final candidate bytes: focused checker, strict, diff, owned-path, link/structure, read-only byte guard, recoverable base blob and Harness consistency passed; `C9/T10/V8/R9=36`, hard blockers `0`.
- Candidate SHA is bound externally after the immutable commit. Independent verification, integration and archive remain `NOT_RUN`.
- No push, deploy, external write, integration or archive is authorized.

## Downstream product-decision blockers

| item | targeted downstream module | current state | effect on this W0 |
| --- | --- | --- | --- |
| P1 | 营业状态切换的端与角色归属 | `PENDING_USER` | none |
| P2 | 跨营业日未取订单的运营处置时限 | `PENDING_USER` | none |
| P3 | PC 扫码登录会话时长、设备信任和并发登录 | `PENDING_USER` | none |
| P4 | 附加手机号数量上限与数据模型 | `PENDING_USER` | none |
| P5 | 全局折扣率对新报价的生效时机 | `PENDING_USER` | none |

每项只阻塞命中该边界的后续 change；0818 PRD §16.3 的暂定口径不得被写成客户已确认 MUST。本 W0 不依赖这些裁决。

## Observations

None.
