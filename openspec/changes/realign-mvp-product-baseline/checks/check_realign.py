#!/usr/bin/env python3
"""realign-mvp-product-baseline 结构与内容门禁。用法: check_realign.py <repo-root>"""
import io, os, re, sys

root = sys.argv[1]
CH = os.path.join(root, "openspec/changes/realign-mvp-product-baseline")
BASE = os.path.join(root, "openspec/specs/mvp-product-baseline/spec.md")
DELTA = os.path.join(CH, "specs/mvp-product-baseline/spec.md")
fails = []

# 1. 四类产物齐全
for f in ("proposal.md", "design.md", "tasks.md", "specs/mvp-product-baseline/spec.md", ".openspec.yaml"):
    if not os.path.isfile(os.path.join(CH, f)):
        fails.append(f"ARTIFACT_MISSING {f}")
if fails:
    print("\n".join(fails)); print("REALIGN_GATE=FAIL"); sys.exit(1)

base = io.open(BASE, encoding="utf-8").read()
delta = io.open(DELTA, encoding="utf-8").read()
base_hdrs = {m.group(1).strip() for m in re.finditer(r"^### Requirement: (.+)$", base, re.M)}

# 2. 解析 delta 分段
sec, cur, bodies = {}, None, {}
for line in delta.splitlines():
    m = re.match(r"^## (MODIFIED|ADDED|REMOVED) Requirements$", line)
    if m:
        cur = m.group(1); sec.setdefault(cur, []); continue
    if cur:
        bodies.setdefault(cur, []).append(line)
        r = re.match(r"^### Requirement: (.+)$", line)
        if r: sec[cur].append(r.group(1).strip())

# 3. MODIFIED/REMOVED 标题必须精确命中基线；ADDED 必须是新标题
for kind in ("MODIFIED", "REMOVED"):
    for h in sec.get(kind, []):
        if h not in base_hdrs: fails.append(f"HEADER_NOT_IN_BASE {kind} {h}")
for h in sec.get("ADDED", []):
    if h in base_hdrs: fails.append(f"ADDED_DUPLICATES_BASE {h}")

# 4. 10 条目标基线 requirement 必须全部被处理
TARGETS = [
    "Product sources have one explicit authority order",
    "First-phase scope is closed and singular",
    "Inventory is keyed by service date, meal period, and product",
    "Order submission uses a bounded atomic soft hold",
    "Orders use one nine-state production state machine",
    "Cancellation and refund rules are deterministic",
    "Employee identity is decided by an active phone list",
    "Employee price is an optional fixed per-product amount",
    "Every first-phase order uses one fixed pickup slot",
    "Merchant permissions use four server-enforced roles",
]
touched = set(sec.get("MODIFIED", [])) | set(sec.get("REMOVED", []))
for t in TARGETS:
    if t not in touched: fails.append(f"TARGET_NOT_HANDLED {t}")

# 5. MODIFIED/ADDED 每条至少一个 Scenario；REMOVED 每条须有 Reason 与 Migration
for kind in ("MODIFIED", "ADDED"):
    blocks = re.split(r"^### Requirement: ", "\n".join(bodies.get(kind, [])), flags=re.M)[1:]
    for b in blocks:
        title = b.splitlines()[0].strip()
        if "#### Scenario:" not in b: fails.append(f"NO_SCENARIO {kind} {title}")
rem = re.split(r"^### Requirement: ", "\n".join(bodies.get("REMOVED", [])), flags=re.M)[1:]
for b in rem:
    title = b.splitlines()[0].strip()
    if "**Reason**:" not in b: fails.append(f"NO_REASON {title}")
    if "**Migration**:" not in b: fails.append(f"NO_MIGRATION {title}")

# 6. 已废止概念只能出现在否定语境
RETIRED = ["待支付", "已支付待接单", "已取消", "软预占", "迟到支付", "employee_price",
           "即时", "优惠券", "会员等级", "接单", "库存"]
NEG = ["MUST NOT", "不得", "不存在", "不提供", "不再", "不做", "不实现", "不支持",
       "MUST 不", "没有", "取消", "删除", "拒绝", "排除", "失效", "无需", "不触发",
       "不引入", "不生成", "不属于", "改由", "替换"]
# 粒度为空行分隔的段落/场景块：一个 WHEN/THEN/AND 三元组是一个语义单元，
# 否定词可能落在 THEN 而非 WHEN 上，按单行判定会产生假阳性。
live = "\n".join(bodies.get("MODIFIED", []) + bodies.get("ADDED", []))
for chunk in re.split(r"\n\s*\n", live):
    if not chunk.strip(): continue
    hits = [t for t in RETIRED if t in chunk]
    if hits and not any(n in chunk for n in NEG):
        fails.append(f"RETIRED_AFFIRMATIVE {hits} :: {chunk.strip().splitlines()[0][:90]}")

# 7. proposal 必须声明门禁字段
prop = io.open(os.path.join(CH, "proposal.md"), encoding="utf-8").read()
for token in ("gate_type=W0", "ui_level_target=UI0", "Owned paths", "Non-goals", "base_sha"):
    if token not in prop: fails.append(f"PROPOSAL_MISSING {token}")

print(f"requirements: MODIFIED={len(sec.get('MODIFIED',[]))} "
      f"ADDED={len(sec.get('ADDED',[]))} REMOVED={len(sec.get('REMOVED',[]))} "
      f"scenarios={delta.count('#### Scenario:')}")
if fails:
    print("\n".join(fails)); print("REALIGN_GATE=FAIL"); sys.exit(1)
print("REALIGN_GATE=PASS")
