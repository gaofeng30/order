> 状态：`DRAFT`。当前提交只规划任务；批准前不得勾选、执行任务或修改 PRD。每完成一项，必须在该项下记录决定性命令、结果或设计理由。

## 1. Approval, Ownership and Red Evidence

- [ ] 1.1 取得用户对本 change 的明确批准，把状态从 `DRAFT` 更新为 `APPROVED`；确认唯一 writer 仍在 `codex/formalize-mvp-product-baseline` 的既有独立 worktree，HEAD 包含规划 commit，且 `docs/product/online-ordering-system-prd.md` 没有并行 writer 或未吸收的变化。
- [ ] 1.2 重新完整读取 `proposal.md`、`design.md`、`specs/mvp-product-baseline/spec.md` 与本文件，运行 `openspec validate formalize-mvp-product-baseline --strict`，确认无行为 Open Question 后才进入 `IMPLEMENTING`。
- [ ] 1.3 在修改 PRD 前运行 4.1 的正式基线检查并记录 Red；失败必须来自当前 §1–§14 缺少三维库存键、统一九态、四角色、12 Gate 或仍含行为待确认，而不是命令或环境错误。
- [ ] 1.4 逐条对照 spec 与当前 PRD，记录现有 §1–§14 中关于员工身份、预约、库存扣减、版本拆分、状态机、退款和后台角色的冲突位置，作为后续同文件最小改写边界。

## 2. Minimal PRD Baseline Implementation

- [ ] 2.1 仅修改 `docs/product/online-ordering-system-prd.md` 的 §2–§3，写入一期唯一包含/排除范围、来源优先级、mock 非生产契约，以及“仓库范围内未发现已签署证据、现实签署状态未知”的严格证据边界。
- [ ] 2.2 更新 §4–§5：浏览免手机号；首次结算需要微信会话、手机号和姓名；服务端按启用名单识别员工；逐商品固定员工价；仅今天/明天、午/晚固定预约时段且无“尽快取餐”。
- [ ] 2.3 更新 §6：库存唯一键为营业日期×餐段×商品、午晚餐独立且同餐段时段共享；员工价使用整数分并固化价格快照；营业设置使用 `Asia/Shanghai`、逐时段截单、单门店单取餐点；后台权限固定店管/后厨/核销/财务只读。
- [ ] 2.4 更新 §7：写入 15 分钟原子软预占、支付实扣、超时释放、迟到支付重占/失败自动全额退款、统一九态、服务端单向转换及生产禁止撤销。
- [ ] 2.5 更新 §7.5–§7.6：未支付直接取消；已支付未接单且未截单自动原路全退；接单或截单后仅商户处理；退款中/已退款语义、接单前库存返还、接单后不自动返库及一期无部分退款。
- [ ] 2.6 更新 §8–§12：把全部已冻结行为放入 MVP 正式开发必做和正式业务不变量，从 V1/二期移除冲突项；补幂等、后端事实源、服务端资源权限、审计和敏感数据边界。
- [ ] 2.7 把 §13 改为生产配置与外部依赖，按 design D7 写入固定 12 Gate、责任方、非敏感证据引用、阻塞阶段和更新时间字段；将真实商品/库存/时段/截单/取餐点/员工名单/价格值明确为 UAT 前配置。
- [ ] 2.8 更新 §14 的可执行验收和需求—页面—状态—角色—外部依赖追踪矩阵，覆盖 spec 全部 requirements；在 §15 章首及冲突位置明确 P0 as-built 低于 §1–§14，不重写其视觉和演示实现细节。

## 3. Refactor and Consistency Pass

- [ ] 3.1 在 §1–§14 删除或改写所有与正式规则冲突的“待确认”“建议”“可选方案”和 V1/二期归类，保证行为只有一条生效路径；外部未就绪项只保留为具有责任方和最晚阶段的 Gate 或 UAT 配置。
- [ ] 3.2 统一全文正式术语：九态名称、四角色名称、`营业日期 × 餐段 × 商品`、15 分钟软预占、逐商品 `employee_price`、单门店单取餐点和 `Asia/Shanghai`；§15 的不同术语必须显式标记为原型。
- [ ] 3.3 检查 PRD 和 change artifacts 不含密钥、证书正文、账号标识、真实手机号、员工名单或其他个人数据；只保留非敏感证据引用与就绪状态。

## 4. Local Verification

- [ ] 4.1 运行以下只读检查，验证 PRD §1–§14 包含全部决定性规则且没有行为 TODO；命令必须返回 `PRD baseline check PASS`：

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
      "交易结算", "隐私", "体验版真机", "审核", "客户发布确认",
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

- [ ] 4.2 逐条对照 `specs/mvp-product-baseline/spec.md` 与 PRD §14 追踪矩阵，确认每条 requirement 都有 PRD 位置、页面、状态、角色、外部 Gate 和验收方法；不适用维度显式标注，不允许空单元格。
- [ ] 4.3 运行 `openspec validate formalize-mvp-product-baseline --strict`，并核对 `openspec status --change formalize-mvp-product-baseline --json` 所有 artifacts 完整。
- [ ] 4.4 以 `c47135b660a9ca3f9f9ee6ded6b09fbf0ee6f1af` 为边界运行 owned-path 检查；`git diff --name-only c47135b660a9ca3f9f9ee6ded6b09fbf0ee6f1af` 只能列出 `openspec/changes/formalize-mvp-product-baseline/**` 与 `docs/product/online-ordering-system-prd.md`，并确认客户清单、合同、technical.md 和业务代码无 diff。
- [ ] 4.5 运行 `git diff --check`、检查 Markdown 标题/表格/链接，并把 4.1–4.4 的决定性结果记录到已完成任务下；任何规则、spec 或任务调整后重跑全部本地 Gate。

## 5. Candidate and Independent Verification

- [ ] 5.1 将完成证据和状态更新写入本 change，只暂存 owned paths，提交一个完整 `CANDIDATE`，记录完整 SHA 并确认 writer worktree clean；不得推送、创建 PR 或修改外部系统。
- [ ] 5.2 verifier 在另一个干净 detached worktree 检出 5.1 的精确 SHA，只读重跑 4.1–4.5、严格 OpenSpec 和完整 diff 审查，并确认 worktree 在验证后仍 clean。
- [ ] 5.3 若需要把独立验证结果写回 artifacts，写回提交视为新候选；必须由 verifier 对新的完整 SHA 再跑 5.2，只有最终 SHA 可进入 `INDEPENDENT_VERIFIED`，任何 PRD/spec/tasks/rebase/merge 变化都使旧验证失效。
