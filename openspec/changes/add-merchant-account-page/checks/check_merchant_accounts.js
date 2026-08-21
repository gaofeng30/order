#!/usr/bin/env node
/* PC 商户账号名单门禁。用法: node check_merchant_accounts.js <repo-root> */
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const root = process.argv[2];
const WA = path.join(root, 'apps/web-admin');
const fails = [], pending = [];
const check = (label, fn) => {
  try {
    const r = fn();
    if (r && typeof r.then === 'function') pending.push(r.catch(e => fails.push(`${label}: ${String(e.message).split('\n')[0]}`)));
  } catch (e) { fails.push(`${label}: ${String(e.message).split('\n')[0]}`); }
};
const read = rel => fs.readFileSync(path.join(WA, rel), 'utf8');
const ctx = () => {
  const sb = { window: {}, console, setTimeout, clearTimeout, Promise,
               DecompressionStream, TextDecoder, Response, Uint8Array, DataView, ArrayBuffer };
  sb.globalThis = sb;
  const c = vm.createContext(sb);
  for (const rel of ['data/xlsx.js', 'data/seed.js', 'data/api.js']) vm.runInContext(read(rel), c, { filename: rel });
  const w = sb.window;
  w.__store = {
    accounts: JSON.parse(JSON.stringify(w.Seed.MERCHANT_ACCOUNTS || [])),
    staff: JSON.parse(JSON.stringify(w.Seed.STAFF_WHITELIST)),
    settings: JSON.parse(JSON.stringify(w.Seed.SETTINGS)),
    store: { status: '营业中' },
  };
  return w;
};

check('the merchant account seed carries phone, name, role and system fields', () => {
  const { Seed } = ctx();
  const list = Seed.MERCHANT_ACCOUNTS;
  assert.ok(Array.isArray(list) && list.length > 0, 'no merchant account seed');
  for (const a of list) {
    for (const k of ['id', 'phone', 'name', 'role', 'enabled']) {
      assert.equal(Object.hasOwn(a, k), true, `${a.id} missing ${k}`);
    }
    assert.equal(['owner', 'staff'].includes(a.role), true, `${a.id}.role = ${a.role}`);
  }
  assert.ok(list.some(a => a.role === 'owner' && a.enabled), 'seed needs an enabled owner');
  assert.ok(list.some(a => a.role === 'staff'), 'seed needs a sub-account');
});

check('the two rosters are stored separately and never share records', () => {
  const { Seed } = ctx();
  assert.notEqual(Seed.MERCHANT_ACCOUNTS, Seed.STAFF_WHITELIST);
  const ids = new Set(Seed.STAFF_WHITELIST.map(r => r.id));
  for (const a of Seed.MERCHANT_ACCOUNTS) assert.equal(ids.has(a.id), false, 'roster ids collide');
});

check('the contract exposes the merchant roster and role labels', () => {
  const { Api } = ctx();
  for (const m of ['listMerchantAccounts', 'saveMerchantAccount', 'setMerchantAccountEnabled', 'deleteMerchantAccount']) {
    assert.equal(typeof Api[m], 'function', `contract missing ${m}`);
  }
  assert.equal(JSON.stringify(Api.ROLES), JSON.stringify(['owner', 'staff']));
  assert.equal(JSON.stringify(Api.ROLE_LABEL), JSON.stringify({ owner: '主账号', staff: '子账号' }));
});

check('saving validates phone, name, role and uniqueness', () => {
  const w = ctx();
  const existing = w.__store.accounts[0];
  return w.Api.saveMerchantAccount({ phone: '13900008888', name: '新店员', role: 'staff' })
    .then(created => {
      assert.equal(created.enabled, true, 'new account must default to enabled');
      assert.equal(created.boundOpenId, '', 'new account must start unbound');
      return w.Api.saveMerchantAccount({ name: '无号', role: 'staff' }).then(() => { throw new Error('accepted a missing phone'); }, () => {});
    })
    .then(() => w.Api.saveMerchantAccount({ phone: '123', name: '格式错', role: 'staff' }).then(() => { throw new Error('accepted a malformed phone'); }, () => {}))
    .then(() => w.Api.saveMerchantAccount({ phone: '13900009999', role: 'staff' }).then(() => { throw new Error('accepted a missing name'); }, () => {}))
    .then(() => w.Api.saveMerchantAccount({ phone: '13900009999', name: '角色错', role: 'admin' }).then(() => { throw new Error('accepted an unknown role'); }, () => {}))
    .then(() => w.Api.saveMerchantAccount({ phone: existing.phone, name: '重复', role: 'staff' }).then(() => { throw new Error('accepted a duplicate phone'); }, () => {}));
});

check('the last enabled owner can never be removed, disabled or demoted', () => {
  const w = ctx();
  // 先把名单压到只剩一个启用的主账号
  const owners = w.__store.accounts.filter(a => a.role === 'owner' && a.enabled);
  const keep = owners[0];
  w.__store.accounts = w.__store.accounts.filter(a => a.id === keep.id || a.role !== 'owner');
  return w.Api.deleteMerchantAccount(keep.id)
    .then(() => { throw new Error('deleted the last enabled owner'); }, () => {})
    .then(() => w.Api.setMerchantAccountEnabled(keep.id, false).then(() => { throw new Error('disabled the last enabled owner'); }, () => {}))
    .then(() => w.Api.saveMerchantAccount({ id: keep.id, phone: keep.phone, name: keep.name, role: 'staff' })
      .then(() => { throw new Error('demoted the last enabled owner'); }, () => {}))
    .then(() => {
      const still = w.__store.accounts.find(a => a.id === keep.id);
      assert.equal(still.role, 'owner');
      assert.equal(still.enabled, true);
    });
});

check('a second owner makes the first one removable', () => {
  const w = ctx();
  const first = w.__store.accounts.find(a => a.role === 'owner' && a.enabled);
  return w.Api.saveMerchantAccount({ phone: '13900007777', name: '备用主账号', role: 'owner' })
    .then(() => w.Api.setMerchantAccountEnabled(first.id, false))
    .then(off => assert.equal(off.enabled, false, 'could not disable an owner while another exists'));
});

check('editing keeps the wechat binding', () => {
  const w = ctx();
  const bound = w.__store.accounts.find(a => a.boundOpenId);
  assert.ok(bound, 'seed needs a bound account');
  return w.Api.saveMerchantAccount({ id: bound.id, phone: bound.phone, name: '改名了', role: bound.role })
    .then(updated => {
      assert.equal(updated.name, '改名了');
      assert.equal(updated.boundOpenId, bound.boundOpenId, 'edit dropped the wechat binding');
    });
});

check('the page is registered, reachable and offers no bulk import', () => {
  assert.equal(fs.existsSync(path.join(WA, 'pages/accounts.js')), true, 'pages/accounts.js missing');
  assert.match(read('index.html'), /pages\/accounts\.js/, 'index.html does not load the page');
  assert.match(read('app.js'), /r: 'accounts'/, 'sidebar has no accounts route');
  const src = read('pages/accounts.js');
  assert.match(src, /Api\.ROLE_LABEL/, 'page does not render roles from the contract');
  assert.match(src, /折扣/, 'page does not distinguish itself from the discount whitelist');
  /* PRD §6.13.4：商户账号名单不提供批量导入。
     断言的是「不具备导入能力」这一事实，不是「批量导入」四个字不出现 ——
     页面注释里说明本页为何不提供导入是正当的。 */
  assert.doesNotMatch(src, /ImportFlow\.render|preview\w*Import|commit\w*Import/, 'accounts page wires up an import flow');
  assert.doesNotMatch(src, /type="file"|accept="\.xlsx"/, 'accounts page offers a file input');
  assert.doesNotMatch(read('app.js'), /r: 'accounts-import'/, 'sidebar offers an accounts import route');
  assert.doesNotMatch(read('index.html'), /pages\/accounts-import\.js/, 'an accounts import page is registered');
});

check('the PRD states the account fields, the role scopes and the last-owner rule', () => {
  const prd = fs.readFileSync(path.join(root, 'docs/product/online-ordering-system-prd-0818.md'), 'utf8');
  const s44 = prd.slice(prd.indexOf('### 4.4'), prd.indexOf('## 5. 用户端功能'));
  assert.ok(s44.length > 200, 'PRD §4.4 not found');
  for (const frag of ['最后一个主账号不可失效', '唯一启用的主账号', '降级为子账号',
                      '服务端裁决', '先添加并启用另一个主账号',
                      '子账号只能用小程序商户端的订单、核销、菜品三个页面',
                      '停用保留记录', '删除不可恢复']) {
    assert.ok(s44.includes(frag), `PRD §4.4 missing: ${frag}`);
  }
  /* §16.5 的商户账号名单行必须已从「未建」转出，否则 PRD 与仓库现状不符 */
  const gaps = prd.slice(prd.indexOf('## 16.5'), prd.indexOf('## 16.6'));
  const row = gaps.split('\n').find(l => l.includes('商户账号名单页'));
  assert.ok(row, '§16.5 lost the merchant roster row');
  assert.doesNotMatch(row, /\*\*PC 侧未建\*\*/, '§16.5 still calls the merchant roster page unbuilt');
});

check('all javascript parses', () => {
  const files = [];
  (function walk(d) {
    for (const e of fs.readdirSync(d, { withFileTypes: true })) {
      const p = path.join(d, e.name);
      if (e.isDirectory()) walk(p); else if (e.name.endsWith('.js')) files.push(p);
    }
  })(WA);
  for (const f of files) new vm.Script(fs.readFileSync(f, 'utf8'), { filename: f });
  console.log(`  parsed ${files.length} javascript files`);
});

Promise.all(pending).then(() => {
  if (fails.length) { console.log(fails.map(f => `  ${f}`).join('\n')); console.log('MERCHANT_ACCOUNTS_GATE=FAIL'); process.exit(1); }
  console.log('MERCHANT_ACCOUNTS_GATE=PASS');
});
