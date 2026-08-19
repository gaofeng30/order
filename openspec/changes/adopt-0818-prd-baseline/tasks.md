## 1. Boundary and approval

- [x] 1.1 复核 proposal/design/spec/tasks/checkpoint 在 DRAFT 一致声明 `gate_type=W0`、`ui_level_target=UI0`、`ui_level_actual=NOT_RUN`、base SHA、owner、owned paths、read-only contracts、dependencies none、external assets none、non-goals 与唯一 ACCEPT/REJECT 裁决；运行 `openspec validate adopt-0818-prd-baseline --strict`，取得明确批准后才记录 `DRAFT → APPROVED`。
  - Evidence：`openspec validate adopt-0818-prd-baseline --strict` exit `0`（`Change 'adopt-0818-prd-baseline' is valid`）；main Agent review 明确 `APPROVED`；Harness 已依次记录 `DRAFT → APPROVED → IMPLEMENTING`。此证据只证明规划可执行，不证明实现完成。

## 2. Red

- [x] 2.1 在不修改 `docs/product/online-ordering-system-prd.md` 的前提下运行 focused W0 内容检查，证明旧文件不是短薄指针，且仍把即时单/数量库存/软预占/九态/接单/四角色/会员券/逐商品员工价写成有效正文；命令必须非零退出并保存首个决定性失败。
  - Check contract：标准库脚本要求旧 PRD 不超过 40 行，包含 `online-ordering-system-prd-0818.md`、`已废止`、`§13.2 外部 Gate` 和 `openspec/specs/mvp-product-baseline/spec.md`，并拒绝任何未置于废止说明中的第二套业务正文。
  - Red evidence：`PYTHONDONTWRITEBYTECODE=1 python3 openspec/changes/adopt-0818-prd-baseline/checks/verify_baseline.py` exit `1`；首个决定性失败为 `retired PRD must be a thin pointer with at most 40 lines; found 985 lines`。`candidate_sha=not-yet-created`；外部资产 `N/A`；未证明 Green、运行行为、真实微信或生产结果。

## 3. Green

- [x] 3.1 仅将 `docs/product/online-ordering-system-prd.md` 替换为明确废止的薄指针：唯一指向 0818 PRD §1–§14，说明 review 的证据身份，并用 `§13.2 外部 Gate` 锚点重定向 canonical `mvp-product-baseline` 十二 Gate requirement；不得复制旧业务正文。
  - Evidence：只修改 owned PRD，将 `985` 行旧正文替换为 `13` 行废止指针；0818 PRD/review、canonical spec 和业务文件未改。
- [x] 3.2 重跑 2.1 完全相同的 focused check 并取得零退出；同时逐项确认 delta 覆盖 I1–I16，且库存/软预占/即时单/接单/九态/四角色/会员券/逐商品员工价只保留为已删除或禁止语义。
  - Green evidence：同一命令 `PYTHONDONTWRITEBYTECODE=1 python3 openspec/changes/adopt-0818-prd-baseline/checks/verify_baseline.py` exit `0`；`BASELINE_CHECK=PASS pointer_lines=13 invariants=16 removed_requirements=6 blockers=5`。外部资产 `N/A`；不证明运行行为、真实微信、支付、UAT、部署或生产结果。

## 4. Refactor and writer gate

- [x] 4.1 精简重复表述后重跑同一 focused check，并 byte-guard `docs/product/online-ordering-system-prd-0818.md`、`docs/product/online-ordering-system-prd-0818-review.md`、`openspec/specs/mvp-product-baseline/spec.md` 相对 `2f2db4a31f66f992997880a02b438c9690bbb845` 未变。
  - Refactor evidence：去重后同一 focused command exit `0`；`BASELINE_CHECK=PASS pointer_lines=11 invariants=16 removed_requirements=6 blockers=5`。`git diff --exit-code 2f2db4a31f66f992997880a02b438c9690bbb845 -- <three-read-only-contracts>` exit `0`，三份只读共享契约 byte-identical。外部资产 `N/A`；不证明 UI1/真实平台/业务运行结果。
- [x] 4.2 运行 `openspec validate adopt-0818-prd-baseline --strict`、`git diff --check 2f2db4a31f66f992997880a02b438c9690bbb845` 与包含 untracked files 的 owned-path audit；结果只能包含 `docs/product/online-ordering-system-prd.md` 和 `openspec/changes/adopt-0818-prd-baseline/**`。
  - Writer checks：focused checker PASS；change strict PASS；diff check PASS；8 个 intended paths 的 owned-path audit PASS；三份只读共享契约 byte guard PASS；两个 PRD 链接和 canonical spec 目标存在；base PRD blob 可读取；Harness consistency 在把已完成 current task 从 `4.1` 更新到 `4.2` 后 PASS。Go、JS、API、数据库、UI1、微信和生产检查因本 W0 不修改对应影响面均为 `N/A`，未写成 PASS。
- [x] 4.3 记录真实 Red/Green/Refactor/writer 证据与 C/T/V/R，确认硬阻断为零后只提交 owned paths，记录完整 `candidate_sha` 并确认 worktree/index clean；未提交 SHA 不得进入 verifier。
  - Writer verdict prepared：`C9/T10/V8/R9=36`，每项 `>=8`，hard blockers `0`；`candidate_sha=external-post-commit`，由不可变提交后的 Harness checkpoint 和交接记录绑定。未验证边界：independent exact-SHA verification、UI1/微信/支付/UAT/部署/生产、integration 与 archive 均为 `NOT_RUN`。

## 5. Independent exact-SHA verification

- [ ] 5.1 verifier 在另一 clean detached worktree 检出完整 `candidate_sha`，确认 HEAD 精确匹配、worktree clean、只读共享契约相对 base 未变，并从头重跑 2.1/3.2 的 focused check、strict、diff check 和 owned-path audit。
- [ ] 5.2 verifier 只记录 exact-SHA `PASS` 或首个真实 `FAIL` 与未验证边界，不修改 candidate bytes；proposal/design/spec/tasks、验收命令、base 或 SHA 任一变化使旧验证立即失效并返回 writer。

## 6. Authorized integration boundary

- [ ] 6.1 仅在 exact candidate 获未失效的 independent PASS、当前 main 仍满足 base/无依赖条件且用户单独授权集成后，按仓库流程集成本地 main；main 已推进时产生新 candidate 并重跑 writer 与 independent Gates。
- [ ] 6.2 集成后只核对 main 包含已验证内容和 owned paths，记录 `INTEGRATED`；本 change 明确禁止 archive、push、deploy、外部写入和权限变更。
