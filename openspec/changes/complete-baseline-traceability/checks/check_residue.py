#!/usr/bin/env python3
"""生效 spec 残留门禁：确保 mvp-product-baseline 中没有任何 requirement 仍在肯定语境
下引用已被 2026-08-19 客户评审删除的概念——包括未被任何 delta 覆盖的 requirement。

用法: check_residue.py <tree-root>
"""
import io, os, re, sys

root = sys.argv[1]
LIVE = os.path.join(root, "openspec/specs/mvp-product-baseline/spec.md")
CHANGES = os.path.join(root, "openspec/changes")
fails = []

if not os.path.isfile(LIVE):
    print(f"LIVE_SPEC_MISSING {LIVE}"); print("RESIDUE_GATE=FAIL"); sys.exit(1)
live = io.open(LIVE, encoding="utf-8").read()

RETIRED = ["九态", "nine-state", "软预占", "soft hold", "迟到支付", "employee_price",
           "四角色", "four server-enforced", "固定取餐时段", "fixed pickup slot",
           "已支付待接单", "待取超时", "会员等级", "优惠券", "餐段库存", "库存池"]
NEG = ["MUST NOT", "不得", "不存在", "不提供", "不再", "不做", "不实现", "不支持",
       "MUST 不", "没有", "取消", "删除", "拒绝", "排除", "失效", "无需", "不触发",
       "不引入", "不生成", "不属于", "改由", "替换", "Reason", "Migration"]

def req_blocks(text):
    parts = re.split(r"(?m)^(### Requirement: .+)$", text)
    out, i = {}, 1
    while i < len(parts):
        out[parts[i].strip()[len("### Requirement: "):]] = parts[i+1]
        i += 2
    return out

# 收集所有未归档 change 的 delta 覆盖的 requirement 标题
covered = set()
if os.path.isdir(CHANGES):
    for name in sorted(os.listdir(CHANGES)):
        d = os.path.join(CHANGES, name, "specs", "mvp-product-baseline", "spec.md")
        if name == "archive" or not os.path.isfile(d):
            continue
        for m in re.finditer(r"(?m)^### Requirement: (.+)$", io.open(d, encoding="utf-8").read()):
            covered.add(m.group(1).strip())

# 生效 spec 中每条 requirement 都必须要么被某个 delta 覆盖，要么自身无残留
uncovered_hits = 0
for hdr, body in req_blocks(live).items():
    if hdr in covered:
        continue
    for chunk in re.split(r"\n\s*\n", body):
        if not chunk.strip():
            continue
        hits = [t for t in RETIRED if t in chunk]
        if hits and not any(n in chunk for n in NEG):
            uncovered_hits += 1
            fails.append(f"RESIDUE_IN_UNCOVERED_REQUIREMENT [{hdr}] {hits} :: "
                         f"{chunk.strip().splitlines()[0][:88]}")

print(f"live_requirements={len(req_blocks(live))} covered_by_pending_deltas={len(covered)} "
      f"residue_hits={uncovered_hits}")
if fails:
    print("\n".join(fails)); print("RESIDUE_GATE=FAIL"); sys.exit(1)
print("RESIDUE_GATE=PASS")
