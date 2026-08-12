> 状态：`CANDIDATE`。主 Agent 已依据用户授权于 2026-08-12 将 change 从 `DRAFT` 批准为 `APPROVED`，writer 完成 preflight 后进入 `IMPLEMENTING`；writer-owned tasks 和本地 Gate 完成后进入 `CANDIDATE`。每完成一项，必须在该项下记录决定性命令、结果或设计理由。

## 1. Approval, Ownership and Red Evidence

- [x] 1.1 取得用户对本 change 的明确批准，把状态从 `DRAFT` 更新为 `APPROVED`；确认唯一 writer 仍在 `codex/formalize-mvp-product-baseline` 的既有独立 worktree，HEAD 包含规划 commit，且 `docs/product/online-ordering-system-prd.md` 没有并行 writer 或未吸收的变化。
  - 证据：主 Agent 明确传达用户授权批准；`git branch --show-current` 返回 `codex/formalize-mvp-product-baseline`，`git rev-parse HEAD` 返回规划 SHA `6bc6f734312e9d0c30a7b16b91c3565b9f4616dc`，`git status --short --branch` 仅显示当前 clean 分支。
- [x] 1.2 重新完整读取 `proposal.md`、`design.md`、`specs/mvp-product-baseline/spec.md` 与本文件，运行 `openspec validate formalize-mvp-product-baseline --strict`，确认无行为 Open Question 后才进入 `IMPLEMENTING`。
  - 证据：`openspec instructions apply --change formalize-mvp-product-baseline --json` 返回四类 context files、23 个 tasks 且 state=`ready`；四文件共 539 行已完整读取；`openspec validate formalize-mvp-product-baseline --strict` 返回 valid，design `Open Questions` 为“无”。
  - 设计同步：批准消息明确外部未就绪状态统一为 `BLOCKED_EXTERNAL`；已在 spec/design/4.1 required terms 中写成唯一规则，strict 复验仍 valid。
- [x] 1.3 在修改 PRD 前运行 4.1 的正式基线检查并记录 Red；失败必须来自当前 §1–§14 缺少三维库存键、统一九态、四角色、12 Gate 或仍含行为待确认，而不是命令或环境错误。
  - Red：4.1 Python 检查正常执行并以 exit 1 结束；报告缺少合同证据边界、`employee_price`、15 分钟软预占、迟到支付、九态中的`已支付待接单`、四角色、`Asia/Shanghai`、12 Gate、`BLOCKED_EXTERNAL` 与 UAT 配置等 35 个决定性词，同时命中`需要和客户讨论`和`部分退款可作为`两个禁止项。
- [x] 1.4 逐条对照 spec 与当前 PRD，记录现有 §1–§14 中关于员工身份、预约、库存扣减、版本拆分、状态机、退款和后台角色的冲突位置，作为后续同文件最小改写边界。
  - 证据：`sed -n '1,1068p' ... | rg -n ...` 定位到员工信息可选/访客临时标识（154–155）、平台超级管理员（188）、提交即扣库存（413）、目标使用待备餐（453）、部分退款 V1（663）、员工/餐段/截单/每日库存被放入 V1（745–751）及 §13“需要和客户讨论”（943）。

## 2. Minimal PRD Baseline Implementation

- [x] 2.1 仅修改 `docs/product/online-ordering-system-prd.md` 的 §2–§3，写入一期唯一包含/排除范围、来源优先级、mock 非生产契约，以及“仓库范围内未发现已签署证据、现实签署状态未知”的严格证据边界。
  - Green：PRD §2–§3 已写入单一闭环、包含/排除清单、四级来源顺序和严格合同证据措辞；4.1 检查确认三条证据边界词及 `mock` 全部存在。
- [x] 2.2 更新 §4–§5：浏览免手机号；首次结算需要微信会话、手机号和姓名；服务端按启用名单识别员工；逐商品固定员工价；仅今天/明天、午/晚固定预约时段且无“尽快取餐”。
  - Green：PRD §4.1、§5.3–§5.5 已形成唯一身份、预约、价格和支付前置规则；4.1 检查命中微信会话/手机号/姓名/名单/今天/明天/午晚餐/`employee_price`，正式区未命中禁止的即时取餐词。
- [x] 2.3 更新 §6：库存唯一键为营业日期×餐段×商品、午晚餐独立且同餐段时段共享；员工价使用整数分并固化价格快照；营业设置使用 `Asia/Shanghai`、逐时段截单、单门店单取餐点；后台权限固定店管/后厨/核销/财务只读。
  - Green：PRD §6.2–§6.7 已覆盖三维库存、午晚独立/餐段共享、价格快照、固定时段、单取餐点与四角色；矩阵检查同时确认四角色均有追踪落点。
- [x] 2.4 更新 §7：写入 15 分钟原子软预占、支付实扣、超时释放、迟到支付重占/失败自动全额退款、统一九态、服务端单向转换及生产禁止撤销。
  - Green：PRD §7.1–§7.4 已写入九态、标准单向链、15 分钟预占、正常/迟到支付及禁止撤销；4.1 检查九态全部词和相关不变量通过。
- [x] 2.5 更新 §7.5–§7.6：未支付直接取消；已支付未接单且未截单自动原路全退；接单或截单后仅商户处理；退款中/已退款语义、接单前库存返还、接单后不自动返库及一期无部分退款。
  - Green：PRD §7.5–§7.6 固定唯一取消/全退/返库规则；4.1 正式区不再含“部分退款可作为”，矩阵退款行覆盖`已取消/退款中/已退款/异常`。
- [x] 2.6 更新 §8–§12：把全部已冻结行为放入 MVP 正式开发必做和正式业务不变量，从 V1/二期移除冲突项；补幂等、后端事实源、服务端资源权限、审计和敏感数据边界。
  - Green：PRD §8.2 汇总一期必做，§8.3 只保留独立后续能力，§9/§12 固定事实源、幂等、权限、审计、敏感数据和 12 条不变量；旧 V1 正式分支已删除。
- [x] 2.7 把 §13 改为生产配置与外部依赖，按 design D7 写入固定 12 Gate、责任方、非敏感证据引用、阻塞阶段和更新时间字段；将真实商品/库存/时段/截单/取餐点/员工名单/价格值明确为 UAT 前配置。
  - Green：PRD §13 包含 12 行有序 Gate，未就绪统一为`BLOCKED_EXTERNAL`，并列出 7 类 UAT 前配置、责任方、最晚阶段和固定模型；敏感数据静态检查 PASS。
- [x] 2.8 更新 §14 的可执行验收和需求—页面—状态—角色—外部依赖追踪矩阵，覆盖 spec 全部 requirements；在 §15 章首及冲突位置明确 P0 as-built 低于 §1–§14，不重写其视觉和演示实现细节。
  - Green：矩阵脚本返回 `traceability matrix check PASS: 12 requirements / 12 complete rows`；§15 从规划 SHA 完整恢复原 P0 细节，仅在章首、蓝本说明、状态机、U05 假支付与章末补充非生产边界。

## 3. Refactor and Consistency Pass

- [x] 3.1 在 §1–§14 删除或改写所有与正式规则冲突的“待确认”“建议”“可选方案”和 V1/二期归类，保证行为只有一条生效路径；外部未就绪项只保留为具有责任方和最晚阶段的 Gate 或 UAT 配置。
  - Refactor：4.1 检查正式区对 `TODO/TBD/A/B/需要和客户讨论/待确认/尽快取餐/部分退款可作为` 全部为零，并返回 `PRD baseline check PASS`。
- [x] 3.2 统一全文正式术语：九态名称、四角色名称、`营业日期 × 餐段 × 商品`、15 分钟软预占、逐商品 `employee_price`、单门店单取餐点和 `Asia/Shanghai`；§15 的不同术语必须显式标记为原型。
  - Refactor：4.1 与矩阵检查共同覆盖统一术语；§15 状态机警告改为指向 §7.1，U05 明确是假支付，章首声明冲突不构成生产契约。
- [x] 3.3 检查 PRD 和 change artifacts 不含密钥、证书正文、账号标识、真实手机号、员工名单或其他个人数据；只保留非敏感证据引用与就绪状态。
  - Refactor：Python 对 PRD 与 change Markdown 扫描 private-key header、具体 AppID 格式和中国大陆手机号格式，返回 `sensitive-data static check PASS`。

## 4. Local Verification

- [x] 4.1 运行以下只读检查，验证 PRD §1–§14 包含全部决定性规则且没有行为 TODO；命令必须返回 `PRD baseline check PASS`：

  ```bash
  python3 - <<'PY'
  from pathlib import Path

  path = Path("docs/product/online-ordering-system-prd.md")
  text = path.read_text(encoding="utf-8")
  target, marker, prototype = text.partition("\n# 15.")
  if not marker:
      raise SystemExit("missing PRD section 15 boundary")

  required = [
      "真实适用合同", "仓库范围内未发现已签署证据", "现实签署状态未知",
      "mock", "单门店", "预约自提", "逐商品", "employee_price",
      "营业日期", "餐段", "商品", "午餐", "晚餐", "15 分钟", "软预占",
      "迟到支付", "待支付", "已支付待接单", "制作中", "待取餐", "已完成",
      "已取消", "退款中", "已退款", "异常", "禁止撤销", "全额退款",
      "微信会话", "手机号", "姓名", "员工名单", "今天", "明天",
      "Asia/Shanghai", "店管", "后厨", "核销", "财务只读",
      "主体一致性", "小程序注册", "餐饮类目", "备案", "项目成员",
      "云资源", "HTTPS", "服务器域名", "商户号", "AppID", "API 安全状态",
      "交易结算", "隐私", "体验版真机", "审核", "客户发布确认", "BLOCKED_EXTERNAL",
      "需求", "页面", "状态", "角色", "外部依赖", "UAT 前配置",
  ]
  forbidden = ["TODO", "TBD", "A/B", "需要和客户讨论", "待确认", "尽快取餐", "部分退款可作为"]
  missing = [value for value in required if value not in target]
  present_forbidden = [value for value in forbidden if value in target]
  if missing or present_forbidden:
      raise SystemExit(f"missing={missing}; forbidden={present_forbidden}")
  print("PRD baseline check PASS")
  PY
  ```

  - 证据：使用上方脚本并追加批准要求的`BLOCKED_EXTERNAL`必需词运行，输出 `PRD baseline check PASS`，exit 0。

- [x] 4.2 逐条对照 `specs/mvp-product-baseline/spec.md` 与 PRD §14 追踪矩阵，确认每条 requirement 都有 PRD 位置、页面、状态、角色、外部 Gate 和验收方法；不适用维度显式标注，不允许空单元格。
  - 证据：Python 解析 spec 的 12 个 requirement 和矩阵 12 行，断言每行恰有 7 个非空单元格，并检查 U01/U03/U05/U06/U07/U08、M01/M03/M04/M09、九态、四角色、Gate 1–12 与“不适用”，输出 `traceability matrix check PASS: 12 requirements / 12 complete rows`。
- [x] 4.3 运行 `openspec validate formalize-mvp-product-baseline --strict`，并核对 `openspec status --change formalize-mvp-product-baseline --json` 所有 artifacts 完整。
  - 证据：在最新本地 main 上重跑 strict，输出 `Change 'formalize-mvp-product-baseline' is valid`；status 返回 `isComplete=true`，proposal/design/specs/tasks 四类 artifact 均为 `done`。
- [x] 4.4 吸收当前本地 main 后运行 owned-path 检查；以精确 main SHA 与三点 diff 区分本 change 和已集成的并行 change，仅允许 `openspec/changes/formalize-mvp-product-baseline/**` 与 `docs/product/online-ordering-system-prd.md`，并确认客户清单、合同、technical.md 和业务代码无本 change diff。
  - 证据：本地 main 最终由 integrator 前移至归档 SHA `69cc9b6437dc3181681603d1bb060c07acba97f1`，其 parent 为 `76e30b9e4a2dd7a9034cc37023a68e68487cebc3`；writer 保存未提交证据后运行 `git rebase main`，无冲突完成。`git merge-base --is-ancestor 69cc9b6437dc3181681603d1bb060c07acba97f1 HEAD` exit 0；`git diff --name-only main...HEAD` 只列出 PRD 和本 change 目录，归档路径不计入本 change，protected-path 检查 PASS。
- [x] 4.5 运行 `git diff --check`、检查 Markdown 标题/表格/链接，并把 4.1–4.4 的决定性结果记录到已完成任务下；任何规则、spec 或任务调整后重跑全部本地 Gate。
  - 证据：`git diff --check main...HEAD` exit 0；Markdown 检查确认 5 个 owned Markdown 文件代码围栏闭合、相对链接有效、正式章节严格为 §1–§14 且 §15 边界唯一；原型保真检查确认相对 main 的 §15 仅有 5 处获批生产边界注释；敏感数据检查 PASS。

## 5. Candidate and Independent Verification

- [x] 5.1 将完成证据和状态更新写入本 change，只暂存 owned paths，提交一个完整 `CANDIDATE`，记录完整 SHA 并确认 writer worktree clean；不得推送、创建 PR 或修改外部系统。
  - 证据：候选提交前已在 `main@69cc9b6437dc3181681603d1bb060c07acba97f1` 上完成 4.1–4.5，且只暂存 owned paths；提交后以 `git rev-parse HEAD`、`git status --short --branch`、`git diff --exit-code` 和 `git diff --cached --exit-code` 生成精确候选 SHA 与 clean 证据并在 handoff 回传。候选不能在自身内容中记录自己的最终 SHA，否则写回会产生新候选。
- [ ] 5.2 verifier 在另一个干净 detached worktree 检出 5.1 的精确 SHA，只读重跑 4.1–4.5、严格 OpenSpec 和完整 diff 审查，并确认 worktree 在验证后仍 clean。
- [ ] 5.3 若需要把独立验证结果写回 artifacts，写回提交视为新候选；必须由 verifier 对新的完整 SHA 再跑 5.2，只有最终 SHA 可进入 `INDEPENDENT_VERIFIED`，任何 PRD/spec/tasks/rebase/merge 变化都使旧验证失效。
