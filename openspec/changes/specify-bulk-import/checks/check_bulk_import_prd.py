#!/usr/bin/env python3
"""批量导入 PRD 一致性门禁。用法: check_bulk_import_prd.py <repo-root>"""
import io, os, re, sys

root = sys.argv[1]
PRD = os.path.join(root, "docs/product/online-ordering-system-prd-0818.md")
LIVE = os.path.join(root, "openspec/specs/mvp-product-baseline/spec.md")
fails = []
prd = io.open(PRD, encoding="utf-8").read()
live = io.open(LIVE, encoding="utf-8").read()

def need(text, token, where):
    if token not in text:
        fails.append(f"{where}: 缺少 {token!r}")

def forbid(text, token, where):
    if token in text:
        fails.append(f"{where}: 不应出现 {token!r}")

# 1) §6.13 存在且四个子节齐全
for t in ("### 6.13 批量导入（PC 后台）", "#### 6.13.1 通用流程", "#### 6.13.2 菜品批量导入",
          "#### 6.13.3 员工白名单批量导入", "#### 6.13.4 不做批量导入的对象"):
    need(prd, t, "PRD")

# 2) 格式决定：收 xlsx、不收 CSV，且旧的 GBK 转存表述已清除
need(prd, "`.xlsx`", "PRD")
need(prd, "**不接受 CSV**", "PRD")
# 禁的是「以 CSV 为受理格式」这一事实，而非「CSV」三个字母 ——
# §6.13.1 需要引用旧做法来说明为何改收 xlsx。
for stale in ("CSV 批量导入", "导入页做显式检测并提示", "| CSV 为 GBK 编码 |"):
    forbid(prd, stale, "PRD")

# 3) 去重规则：菜品只新增、员工按手机号覆盖
need(prd, "**去重规则：只新增，不更新。**", "PRD §6.13.2")
need(prd, "**去重规则：按手机号覆盖更新。**", "PRD §6.13.3")

# 4) 图片不进模板、分类自动新建、商户账号不导入
need(prd, "**图片 MUST NOT 进入模板。**", "PRD §6.13.2")
need(prd, "**分类自动新建**", "PRD §6.13.2")
need(prd, "**商户账号名单不提供批量导入**", "PRD §6.13.4")

# 5) 解析在服务端
need(prd, "解析 MUST 在服务端完成", "PRD §6.13.1")

# 6) 范围来源标注为待客户确认，且 §16.4 有对应条目
need(prd, "开发方提出的范围新增，待客户确认", "PRD §6.13")
need(prd, "批量导入的范围确认", "PRD §16.4")

# 7) 菜品模板列与 §6.3 表单字段一一对应
form = re.search(r"\*\*商品编辑表单字段固定为五项\*\*：(.+?)。", prd)
if not form:
    fails.append("PRD §6.3: 未声明表单字段集合")
else:
    for f in ("菜品图片", "菜品名称", "售价", "分类", "餐段可售", "描述"):
        need(form.group(1), f, "PRD §6.3 表单字段")
    tpl = prd.split("#### 6.13.2 菜品批量导入", 1)[1].split("#### 6.13.3", 1)[0]
    for col in ("菜品名称", "售价", "分类", "餐段可售", "描述"):
        need(tpl, f"| {col} ", "PRD §6.13.2 模板列")
    forbid(tpl.split("| 描述 ")[0], "| 菜品图片 ", "PRD §6.13.2 模板列")

# 8) 页面清单已同步为 12 页且含两个导入页
need(prd, "**PC 网页后台（12 页）**", "PRD §3.5")
for t in ("**菜品批量导入**", "**员工白名单批量导入**"):
    need(prd, t, "PRD §3.5")

# 9) 生效 spec 未被本轮修改（本 change 只补 PRD）
if "批量导入" in live:
    fails.append("生效 spec 不应在本轮引入批量导入条款")

print(f"prd_lines={len(prd.splitlines())}")
if fails:
    print("\n".join(f"  {f}" for f in fails)); print("BULK_IMPORT_PRD=FAIL"); sys.exit(1)
print("BULK_IMPORT_PRD=PASS")
