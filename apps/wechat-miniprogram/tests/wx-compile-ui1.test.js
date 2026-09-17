const assert = require('node:assert/strict');
const { execFileSync, spawnSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
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

test('wxml lint rejects mismatched nested closing tags', () => {
  const script = path.join(miniprogramRoot, 'tests', 'lint_wx.py');
  const fixtureRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'order-wx-lint-mismatch-'));

  try {
    fs.writeFileSync(
      path.join(fixtureRoot, 'mismatch.wxml'),
      '<scroll-view><view></scroll-view></view>\n',
      'utf8',
    );
    const result = spawnSync('python3', [script, fixtureRoot], { encoding: 'utf8' });
    assert.equal(result.error, undefined, 'wx lint process did not start');
    assert.equal(typeof result.status, 'number', 'wx lint process did not return an exit status');
    assert.notEqual(result.status, 0, `wx lint falsely accepted mismatched tags:\n${result.stdout}`);
    assert.match(`${result.stdout}${result.stderr}`, /WXML_CLOSE_MISMATCH/);
  } finally {
    assert.equal(path.dirname(fixtureRoot), os.tmpdir());
    assert.match(path.basename(fixtureRoot), /^order-wx-lint-mismatch-/);
    assert.equal(fs.lstatSync(fixtureRoot).isSymbolicLink(), false);
    fs.rmSync(fixtureRoot, { recursive: true });
  }
});
