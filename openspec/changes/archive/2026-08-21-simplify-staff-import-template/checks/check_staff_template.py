#!/usr/bin/env python3
"""员工白名单字段与模板一致性门禁。用法: check_staff_template.py <repo-root>"""
import io, os, re, sys
root = sys.argv[1]
PRD = os.path.join(root, "docs/product/online-ordering-system-prd-0818.md")
fails = []
prd = io.open(PRD, encoding="utf-8").read()

def need(t, tok, where):
    if tok not in t: fails.append(f"{where}: 缺少 {tok!r}")
def forbid(t, tok, where):
    if tok in t: fails.append(f"{where}: 不应出现 {tok!r}")

sec64 = prd.split("### 6.4 员工折扣与白名单", 1)[1].split("### 6.5", 1)[0]
sec6133 = prd.split("#### 6.13.3 员工白名单批量导入", 1)[1].split("#### 6.13.4", 1)[0]

# 1) §6.4 只声明两个可填字段，且不再列出被删的四个
need(sec64, "只有两个可填字段", "PRD §6.4")
for gone in ("单位 / 部门 / 工号（选填）", "备注（选填）"):
    forbid(sec64, gone, "PRD §6.4")

# 2) 模板恰好两列
rows = re.findall(r"(?m)^\| ([^|]+?) \| (✅)? *\|", sec6133)
cols = [r[0].strip() for r in rows if r[0].strip() not in ("列名", "---")]
if cols != ["姓名", "手机号"]:
    fails.append(f"PRD §6.13.3 模板列应为 ['姓名','手机号']，实际 {cols}")
need(sec6133, "**模板只有两列**", "PRD §6.13.3")
need(sec6133, "**模板 MUST NOT 包含其他任何列。**", "PRD §6.13.3")

# 3) 覆盖更新不得重新启用已停用记录
need(sec6133, "导入 MUST NOT 把已停用的记录重新启用", "PRD §6.13.3")

# 4) 数据模型同步：不再有 org/dept/jobNo/remark
model = prd.split("interface StaffWhitelist {", 1)[1].split("}", 1)[0]
for gone in ("org", "dept", "jobNo", "remark"):
    forbid(model, gone, "PRD §15.6.5 StaffWhitelist")
for keep in ("phone", "name", "enabled"):
    need(model, keep, "PRD §15.6.5 StaffWhitelist")

print(f"staff_template_cols={cols}")
if fails:
    print("\n".join(f"  {f}" for f in fails)); print("STAFF_TEMPLATE=FAIL"); sys.exit(1)
print("STAFF_TEMPLATE=PASS")
