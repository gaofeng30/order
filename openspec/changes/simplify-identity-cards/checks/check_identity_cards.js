#!/usr/bin/env node
/* 身份卡只标身份；微信授权控件不承载布局（PRD §4.4、项目方 2026-08-22 要求）。
   用法: node check_identity_cards.js <repo-root> */
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const root = process.argv[2];
const MP = path.join(root, 'apps/wechat-miniprogram');
const fails = [];
const check = (label, fn) => {
  try { fn(); } catch (e) { fails.push(`${label}: ${String(e.message).split('\n')[0]}`); }
};
const read = rel => fs.readFileSync(path.join(MP, rel), 'utf8');

const wxml = () => read('pages/launch/launch.wxml');
const wxss = () => read('pages/launch/launch.wxss');
// 取出 <button ...> 的开标签
const buttonTag = () => {
  const s = wxml();
  const i = s.indexOf('<button');
  assert.ok(i >= 0, 'the identity screen has no authorisation button');
  let j = i, quote = null;
  for (; j < s.length; j += 1) {
    const c = s[j];
    if (quote) { if (c === quote) quote = null; continue; }
    if (c === '"' || c === "'") { quote = c; continue; }
    if (c === '>') break;
  }
  return s.slice(i, j + 1);
};

check('the auth control carries no layout class', () => {
  const tag = buttonTag();
  assert.doesNotMatch(tag, /\bid-card\b/,
    'the layout class sits on the <button>; wechat button defaults (auto margins, padding, font-size) will fight it');
  assert.match(tag, /open-type\s*=\s*"getPhoneNumber"/, 'the button lost the wechat authorisation trigger');
  assert.match(tag, /bindgetphonenumber/, 'the authorisation result is no longer handled');
});

check('the layout class sits inside the button', () => {
  const s = wxml();
  const bi = s.indexOf('<button');
  const be = s.indexOf('</button>', bi);
  assert.ok(bi >= 0 && be > bi, 'the button is not a container');
  const inner = s.slice(bi, be);
  assert.match(inner, /class="[^"]*\bid-card\b/, 'no card container inside the authorisation button');
  assert.match(inner, /商户端/, 'the merchant label left the button');
});

check('both cards use the same layout rule', () => {
  const s = wxml();
  const cards = [...s.matchAll(/class="([^"]*\bid-card\b[^"]*)"/g)].map(m => m[1]);
  assert.equal(cards.length, 2, `expected two cards, found ${cards.length}`);
  for (const c of cards) {
    assert.ok(c.split(/\s+/).includes('id-card'), `a card does not use the shared layout class: ${c}`);
  }
});

check('the auth button is neutralised, not merely recoloured', () => {
  const src = wxss().replace(/\/\*[\s\S]*?\*\//g, '');
  const rules = [...src.matchAll(/([^{}]+)\{([^}]*)\}/g)].map(m => ({ sel: m[1].trim(), body: m[2] }));
  const tag = buttonTag();
  const cls = (tag.match(/class="([^"]*)"/) || [, ''])[1].split(/\s+/).filter(Boolean);
  assert.ok(cls.length, 'the authorisation button has no class to neutralise it');
  const own = rules.filter(r => r.sel.split(',').some(one => cls.some(c => one.trim() === '.' + c)));
  assert.ok(own.length, 'the authorisation button has no style rule');
  const body = own.map(r => r.body).join(' ');
  for (const decl of [/margin:\s*0/, /padding:\s*0/, /width:\s*100%/, /background:\s*transparent/]) {
    assert.match(body, decl, `the button style does not neutralise ${decl}`);
  }
  const after = rules.filter(r => r.sel.includes('::after') && cls.some(c => r.sel.includes('.' + c)));
  assert.ok(after.length && /border:\s*none/.test(after.map(r => r.body).join(' ')),
    'the system border of the button is not removed');
});

check('the cards carry no descriptive subtitle', () => {
  const s = wxml();
  assert.doesNotMatch(s, /id-desc/, 'a card still renders a descriptive subtitle');
  for (const gone of ['浏览菜单', '在线点单', '接单', '核销', '经营管理']) {
    assert.ok(!s.includes(gone), `the card still carries the description text ${gone}`);
  }
  assert.match(s, /用户端/, 'the user card lost its label');
  assert.match(s, /商户端/, 'the merchant card lost its label');
});

check('the removed style is gone, not orphaned', () => {
  assert.doesNotMatch(wxss(), /\.id-desc\b/,
    'the subtitle rule survived with nothing referencing it');
  const src = wxss();
  assert.equal(src.split('{').length, src.split('}').length, 'wxss braces are unbalanced');
});

check('all javascript parses', () => {
  const files = [];
  (function walk(d) {
    for (const e of fs.readdirSync(d, { withFileTypes: true })) {
      if (e.name === 'node_modules' || e.name === '__pycache__' || e.name === 'tests') continue;
      const p = path.join(d, e.name);
      if (e.isDirectory()) walk(p); else if (e.name.endsWith('.js')) files.push(p);
    }
  })(MP);
  for (const f of files) new vm.Script(fs.readFileSync(f, 'utf8'), { filename: f });
  console.log(`  parsed ${files.length} javascript files`);
});

if (fails.length) {
  console.log(fails.map(f => `  ${f}`).join('\n'));
  console.log(`IDENTITY_CARDS_GATE=FAIL (${fails.length}/7)`);
  process.exit(1);
}
console.log('IDENTITY_CARDS_GATE=PASS');
