#!/usr/bin/env python3
"""WXML 标签结构由静态门禁保证。
用法: check_wxml_balance.py <repo-root>

断言的是能力事实：把构造好的坏模板喂给 lint，它必须报错；
把含 `>` 属性值的好模板喂给它，它必须不报错。不断言源码里出现了什么词。
"""
import os, re, shutil, subprocess, sys, tempfile

root = sys.argv[1]
MP = os.path.join(root, 'apps/wechat-miniprogram')
LINT = os.path.join(MP, 'tests/lint_wx.py')
fails = []


def check(label, fn):
    try:
        fn()
    except AssertionError as e:
        fails.append(f'{label}: {e}')
    except Exception as e:                      # noqa: BLE001
        fails.append(f'{label}: {type(e).__name__}: {e}')


def run_lint(target):
    r = subprocess.run([sys.executable, LINT, target], capture_output=True, text=True)
    return r.returncode, r.stdout + r.stderr


def sandbox(wxml):
    """把一份模板放进只含它的最小目录，用同一支 lint 检查。"""
    d = tempfile.mkdtemp(prefix='wxmlgate-')
    os.makedirs(os.path.join(d, 'pages', 'x'), exist_ok=True)
    with open(os.path.join(d, 'pages', 'x', 'x.wxml'), 'w', encoding='utf-8') as f:
        f.write(wxml)
    try:
        return run_lint(d)
    finally:
        shutil.rmtree(d, ignore_errors=True)


def rejects(label, wxml, must_say=None, line_ref=False):
    def run():
        code, out = sandbox(wxml)
        assert code != 0, f'lint accepted it:\n{out}'
        if must_say:
            assert must_say in out, f'the report does not say "{must_say}":\n{out}'
        if line_ref:
            assert re.search(r'第\s*\d+\s*行', out), f'no line reference for the other end:\n{out}'
    check(label, run)


def accepts(label, wxml):
    def run():
        code, out = sandbox(wxml)
        assert code == 0, f'lint rejected a valid template:\n{out}'
    check(label, run)


# ---- 三类结构错误必须各自被检出 ----
rejects('the gate rejects an orphan closing tag',
        '<view>\n  <view>a</view>\n  </view>\n</view>\n', must_say='孤立')

rejects('the gate rejects an unclosed tag',
        '<view>\n  <scroll-view>\n    <view>a</view>\n</view>\n', must_say='未闭合')

rejects('the gate rejects crossed nesting and names the other end',
        '<view>\n  <scroll-view>\n  </view>\n</scroll-view>\n', line_ref=True)

# ---- 不得产生假阳性 ----
accepts('the gate accepts a balanced template',
        '<view class="a">\n  <scroll-view>\n    <view>hi</view>\n  </scroll-view>\n</view>\n')

accepts('a comparison inside an attribute is not a false positive',
        '<view>\n  <view wx:if="{{qty > 0}}">a</view>\n  <view wx:else>b</view>\n</view>\n')

accepts('a multi-line self-closing component is not a false positive',
        '<view>\n  <stepper\n    wx:if="{{qtyMap[m.id] > 0}}"\n    value="{{q}}"\n  />\n'
        '  <money v="{{t}}" size="30" />\n</view>\n')

accepts('void elements need no closing tag',
        '<view>\n  <icon name="a" size="12">\n  <image src="b">\n  <input value="c">\n</view>\n')

accepts('tags inside comments are ignored',
        '<view>\n  <!-- <scroll-view> 被删掉了 </view> -->\n  <view>a</view>\n</view>\n')


# ---- 既有检查不回归 ----
def elif_guard():
    code, out = sandbox('<view>\n  <view>a</view>\n  <view wx:else>b</view>\n</view>\n')
    assert code != 0, f'the existing wx:else check regressed:\n{out}'
    assert 'wx:else' in out, f'the wx:else report changed shape:\n{out}'


check('the existing wx:elif check still works', elif_guard)


# ---- 仓库现状 ----
VOID = {'image', 'input', 'icon', 'br'}
TAG = re.compile(r'<([a-zA-Z][\w-]*)((?:[^<>"\']|"[^"]*"|\'[^\']*\')*?)(/?)>|</([a-zA-Z][\w-]*)>', re.S)


def imbalances(src):
    """门禁自带的独立解析：不拿被测 lint 的结论当证据。"""
    src = re.sub(r'<!--.*?-->', lambda m: '\n' * m.group(0).count('\n'), src, flags=re.S)
    stack, bad = [], []
    for m in TAG.finditer(src):
        line = src[:m.start()].count('\n') + 1
        if m.group(4):
            if not stack:
                bad.append(f'{line}: orphan </{m.group(4)}>')
            elif stack[-1][0] != m.group(4):
                bad.append(f'{line}: </{m.group(4)}> vs <{stack[-1][0]}> opened at {stack[-1][1]}')
                stack.pop()
            else:
                stack.pop()
            continue
        if m.group(3) or m.group(1) in VOID:
            continue
        stack.append((m.group(1), line))
    bad += [f'{ln}: <{nm}> never closed' for nm, ln in stack]
    return bad


def repo_clean():
    found = []
    for d, dirs, fs in os.walk(MP):
        dirs[:] = [x for x in dirs if x not in ('node_modules', '.git')]
        for f in sorted(fs):
            if not f.endswith('.wxml'):
                continue
            path = os.path.join(d, f)
            for b in imbalances(open(path, encoding='utf-8').read()):
                found.append(f'{os.path.relpath(path, MP)}:{b}')
    assert not found, 'templates are structurally broken:\n  ' + '\n  '.join(found)
    code, out = run_lint(MP)
    assert code == 0, f'the repository has WXML structure errors:\n{out}'
    src = open(os.path.join(MP, 'pages/profile/profile.wxml'), encoding='utf-8').read()
    assert '切换身份' in src, 'the switch-identity entry disappeared'
    i = src.index('switch-id')
    assert '</scroll-view>' not in src[:i], 'the switch-identity entry fell outside the scroll body'


check('every wxml in the repository is balanced', repo_clean)

if fails:
    print('\n'.join(f'  {f}' for f in fails))
    print(f'WXML_BALANCE_GATE=FAIL ({len(fails)}/10)')
    sys.exit(1)
print('WXML_BALANCE_GATE=PASS')
