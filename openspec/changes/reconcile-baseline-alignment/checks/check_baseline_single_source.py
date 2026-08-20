#!/usr/bin/env python3
"""基线单一事实源门禁：确保生效 spec 是唯一对齐产物，且不存在重复的对齐 change。
用法: check_baseline_single_source.py <repo-root>
"""
import io, os, re, sys

root = sys.argv[1]
LIVE = os.path.join(root, "openspec/specs/mvp-product-baseline/spec.md")
PRD = os.path.join(root, "docs/product/online-ordering-system-prd-0818.md")
POINTER = os.path.join(root, "docs/product/online-ordering-system-prd.md")
CHANGES = os.path.join(root, "openspec/changes")
fails = []
read = lambda p: io.open(p, encoding="utf-8").read()

# 1) 生效 spec 存在且覆盖两条吸收进来的 requirement
live = read(LIVE)
titles = set(re.findall(r"(?m)^### Requirement: (.+)$", live))
for t in ("Pickup identifiers and notifications follow the frozen contract",
          "Production facts and statistics come from server-confirmed data"):
    if t not in titles:
        fails.append(f"live spec missing absorbed requirement: {t}")

# 2) 不得存在第二个未归档的 mvp-product-baseline 对齐 delta
dupes = []
for name in sorted(os.listdir(CHANGES)):
    # 排除本 change 自身：它正是那份唯一的对齐 delta
    if name in ("archive", "reconcile-baseline-alignment"):
        continue
    d = os.path.join(CHANGES, name, "specs", "mvp-product-baseline", "spec.md")
    if os.path.isfile(d):
        dupes.append(name)
if dupes:
    fails.append(f"duplicate un-archived baseline delta(s): {', '.join(dupes)}")

# 3) 旧 PRD 必须是薄指针，且指向 0818
pointer = read(POINTER)
if len(pointer.splitlines()) > 40:
    fails.append(f"retired PRD is not a thin pointer ({len(pointer.splitlines())} lines)")
if "online-ordering-system-prd-0818.md" not in pointer:
    fails.append("retired PRD does not point at the 0818 baseline")

# 4) 0818 PRD 不得再自称修订提案
prd = read(PRD)
for stale in ("修订提案**，不是生效基线", "本文档**不取代**"):
    if stale in prd:
        fails.append(f"0818 PRD still calls itself a proposal: {stale}")
if "唯一有效产品基线" not in prd:
    fails.append("0818 PRD does not declare itself the sole baseline")

# 5) 已废止概念不得在生效 spec 中以肯定语境出现
RETIRED = ["九态", "软预占", "迟到支付", "employee_price", "四角色", "固定取餐时段", "已支付待接单"]
# 「失败」补自一处此前记录为已知误判的位置：可追踪性 requirement 的
# `Matrix cites a retired dimension` 场景，其 WHEN 子句必须枚举被禁术语才能禁止它们，
# 而 THEN 子句是「PRD 实施验收失败」。
NEG = ["MUST NOT", "不得", "不存在", "不再", "无", "取消", "删除", "拒绝", "排除",
       "失效", "失败", "Reason", "Migration", "改由"]
for block in re.split(r"(?m)^### Requirement: ", live)[1:]:
    title = block.splitlines()[0].strip()
    for chunk in re.split(r"\n\s*\n", block):
        hits = [t for t in RETIRED if t in chunk]
        if hits and not any(n in chunk for n in NEG):
            fails.append(f"retired concept affirmative in [{title}]: {hits}")

print(f"live_requirements={len(titles)} un_archived_baseline_deltas={len(dupes)} pointer_lines={len(pointer.splitlines())}")
if fails:
    print("\n".join(f"  {f}" for f in fails)); print("BASELINE_SINGLE_SOURCE=FAIL"); sys.exit(1)
print("BASELINE_SINGLE_SOURCE=PASS")
