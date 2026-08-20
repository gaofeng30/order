const assert = require('node:assert/strict');
const { execFileSync } = require('node:child_process');
const path = require('node:path');
const test = require('node:test');
const { miniprogramRoot } = require('./page-harness.js');

// Node harness 只加载 JS，看不见 WXSS/WXML 的编译期错误。
// 两次真实故障促成本用例：
//   1. 按行正则删样式，多行规则只删掉首行选择器，留下孤儿规则体
//      → WXSS compile error: unexpected token ';'
//   2. 删掉 wx:if 分支后 wx:else 失去同级前驱
//      → WXML compile error: wx:if not found
test('wxss and wxml are structurally compilable', () => {
  const script = path.join(miniprogramRoot, 'tests', 'lint_wx.py');
  let out = '';
  try {
    out = execFileSync('python3', [script, miniprogramRoot], { encoding: 'utf8' });
  } catch (error) {
    out = `${error.stdout || ''}${error.stderr || ''}`;
    assert.fail(`wx lint failed:\n${out}`);
  }
  assert.match(out, /WX_LINT=PASS/);
});
