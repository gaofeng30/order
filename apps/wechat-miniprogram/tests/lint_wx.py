#!/usr/bin/env python3
"""WXSS/WXML 静态结构检查：抓 Node harness 看不见的编译期错误。
用法: lint_wx.py <miniprogram-root>"""
import io, os, re, sys

root = sys.argv[1]
fails = []

def walk(ext):
    for d, _, fs in os.walk(root):
        if 'node_modules' in d: continue
        for f in fs:
            if f.endswith(ext): yield os.path.join(d, f)

# ---- WXSS：括号配平 + 孤儿规则体（选择器被删、只剩声明与右括号）----
for p in walk('.wxss'):
    src = io.open(p, encoding='utf-8').read()
    rel = os.path.relpath(p, root)
    body = re.sub(r'/\*.*?\*/', '', src, flags=re.S)
    if body.count('{') != body.count('}'):
        fails.append(f"{rel}: 花括号不配平 {{={body.count('{')} }}={body.count('}')}")
    depth = 0
    for i, line in enumerate(body.splitlines(), 1):
        t = line.strip()
        if depth == 0 and t and not t.startswith('@') and '{' not in t and t.endswith((';', '}')):
            fails.append(f"{rel}:{i}: 顶层出现声明或孤立右括号（选择器缺失）:: {t[:60]}")
        depth += line.count('{') - line.count('}')
        if depth < 0:
            fails.append(f"{rel}:{i}: 右括号多于左括号")
            depth = 0

# ---- WXML：标签配对 + wx:elif / wx:else 必须有同级前驱 wx:if ----
# 两项检查共用同一次遍历与同一份 void 清单：两份清单迟早会漂移，
# 届时同一个文件在两项检查里会被解析成不同的树。
TAG = re.compile(r'<([a-zA-Z][\w-]*)((?:[^<>"\']|"[^"]*"|\'[^\']*\')*?)(/?)>|</([a-zA-Z][\w-]*)>', re.S)
VOID_TAGS = {'image', 'input', 'icon', 'br'}
for p in walk('.wxml'):
    src = io.open(p, encoding='utf-8').read()
    rel = os.path.relpath(p, root)
    src_nc = re.sub(r'<!--.*?-->', lambda m: '\n' * m.group(0).count('\n'), src, flags=re.S)
    open_tags = []
    sibling_conditions = [None]
    for m in TAG.finditer(src_nc):
        line = src_nc[:m.start()].count('\n') + 1
        if m.group(4):
            closing_tag = m.group(4)
            if not open_tags:
                fails.append(f"{rel}:{line}: WXML_UNEXPECTED_CLOSE </{closing_tag}>")
                continue
            opening_tag, opening_line = open_tags.pop()
            sibling_conditions.pop()
            if opening_tag != closing_tag:
                fails.append(
                    f"{rel}:{line}: WXML_CLOSE_MISMATCH expected </{opening_tag}> "
                    f"for line {opening_line}, got </{closing_tag}>"
                )
            continue
        attrs, selfclose = m.group(2) or '', m.group(3)
        cond = ('if' if re.search(r'\bwx:if\b', attrs) else
                'elif' if re.search(r'\bwx:elif\b', attrs) else
                'else' if re.search(r'\bwx:else\b', attrs) else None)
        if cond in ('elif', 'else') and sibling_conditions[-1] not in ('if', 'elif'):
            fails.append(f"{rel}:{line}: wx:{cond} 缺少同级 wx:if 前驱")
        sibling_conditions[-1] = cond
        if not selfclose and m.group(1) not in VOID_TAGS:
            open_tags.append((m.group(1), line))
            sibling_conditions.append(None)
    for opening_tag, opening_line in reversed(open_tags):
        fails.append(f"{rel}:{opening_line}: WXML_UNCLOSED_TAG <{opening_tag}>")

print('\n'.join(f"  {f}" for f in fails) if fails else '  clean')
print('WX_LINT=' + ('FAIL' if fails else 'PASS'))
sys.exit(1 if fails else 0)
