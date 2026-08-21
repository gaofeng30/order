#!/usr/bin/env node
/* PC 员工折扣白名单门禁。用法: node check_staff_page.js <repo-root> */
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
  const sb = { window: {}, console, setTimeout, clearTimeout, Promise };
  sb.globalThis = sb;
  const c = vm.createContext(sb);
  for (const rel of ['data/seed.js', 'data/api.js']) vm.runInContext(read(rel), c, { filename: rel });
  return sb.window;
};
const store = w => {
  w.__store = {
    staff: JSON.parse(JSON.stringify(w.Seed.STAFF_WHITELIST)),
    settings: JSON.parse(JSON.stringify(w.Seed.SETTINGS)),
    store: { status: '营业中' },
  };
  return w;
};

check('the whitelist seed carries only the two writable fields plus system fields', () => {
  const { Seed } = ctx();
  assert.ok(Array.isArray(Seed.STAFF_WHITELIST) && Seed.STAFF_WHITELIST.length > 0, 'no whitelist seed');
  for (const r of Seed.STAFF_WHITELIST) {
    for (const req of ['id', 'phone', 'name', 'enabled', 'joinAt', 'bound', 'spend', 'orders']) {
      assert.equal(Object.hasOwn(r, req), true, `${r.id} missing ${req}`);
    }
    for (const gone of ['org', 'dept', 'jobNo', 'remark', 'levelId']) {
      assert.equal(Object.hasOwn(r, gone), false, `${r.id} still carries ${gone}`);
    }
  }
});

check('the contract exposes the whitelist and the global discount rate', () => {
  const { Api } = ctx();
  for (const m of ['listStaff', 'saveStaff', 'deleteStaff', 'setStaffEnabled', 'getDiscountRate', 'saveDiscountRate']) {
    assert.equal(typeof Api[m], 'function', `contract missing ${m}`);
  }
});

check('saving a whitelist entry validates phone, name and uniqueness', () => {
  const w = store(ctx());
  const existing = w.Seed.STAFF_WHITELIST[0];
  return w.Api.saveStaff({ phone: '13900001111', name: '测试员工' })
    .then(created => {
      assert.equal(created.enabled, true, 'new entry must default to enabled');
      assert.ok(created.joinAt, 'new entry must get a join date');
      assert.equal(created.bound, false);
      // 手机号缺失 / 格式错 / 姓名缺失 / 重复，四种都必须拒绝
      return w.Api.saveStaff({ name: '无手机号' }).then(() => { throw new Error('accepted a missing phone'); }, () => {});
    })
    .then(() => w.Api.saveStaff({ phone: '12345', name: '格式错' }).then(() => { throw new Error('accepted a malformed phone'); }, () => {}))
    .then(() => w.Api.saveStaff({ phone: '13900002222' }).then(() => { throw new Error('accepted a missing name'); }, () => {}))
    .then(() => w.Api.saveStaff({ phone: existing.phone, name: '重复' }).then(() => { throw new Error('accepted a duplicate phone'); }, () => {}));
});

check('editing keeps the system fields and toggling status never resets them', () => {
  const w = store(ctx());
  const target = w.__store.staff[0];
  const snapshot = { joinAt: target.joinAt, bound: target.bound, spend: target.spend, orders: target.orders };
  return w.Api.saveStaff({ id: target.id, phone: target.phone, name: '改过的名字' })
    .then(updated => {
      assert.equal(updated.name, '改过的名字');
      for (const k of Object.keys(snapshot)) {
        assert.deepEqual(updated[k], snapshot[k], `edit reset ${k}`);
      }
      return w.Api.setStaffEnabled(target.id, false);
    })
    .then(off => {
      assert.equal(off.enabled, false, 'disable failed');
      for (const k of Object.keys(snapshot)) {
        assert.deepEqual(off[k], snapshot[k], `toggle reset ${k}`);
      }
    });
});

check('the global discount rate is an integer percent and is validated', () => {
  const w = store(ctx());
  return w.Api.getDiscountRate()
    .then(rate => {
      assert.equal(Number.isInteger(rate), true, `rate must be an integer percent, got ${rate}`);
      assert.ok(rate >= 1 && rate <= 100, `rate out of range: ${rate}`);
      return w.Api.saveDiscountRate(85);
    })
    .then(saved => {
      assert.equal(saved, 85);
      return w.Api.saveDiscountRate(0).then(() => { throw new Error('accepted 0'); }, () => {});
    })
    .then(() => w.Api.saveDiscountRate(101).then(() => { throw new Error('accepted 101'); }, () => {}))
    .then(() => w.Api.saveDiscountRate(85.5).then(() => { throw new Error('accepted a fractional percent'); }, () => {}));
});

check('the page is registered and reachable from the sidebar', () => {
  assert.equal(fs.existsSync(path.join(WA, 'pages/staff.js')), true, 'pages/staff.js missing');
  assert.match(read('index.html'), /pages\/staff\.js/, 'index.html does not load the page');
  assert.match(read('app.js'), /r: 'staff'/, 'sidebar has no staff route');
  const src = read('pages/staff.js');
  assert.match(src, /折扣率/, 'page does not surface the discount rate');
  assert.match(src, /Api\.saveStaff/, 'page does not save entries');
  assert.match(src, /Api\.setStaffEnabled/, 'page has no status toggle');
  // 只有两个可填字段
  for (const gone of ['f-org', 'f-dept', 'f-jobno', 'f-remark']) {
    assert.doesNotMatch(src, new RegExp(gone), `page still edits ${gone}`);
  }
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
  if (fails.length) { console.log(fails.map(f => `  ${f}`).join('\n')); console.log('STAFF_PAGE_GATE=FAIL'); process.exit(1); }
  console.log('STAFF_PAGE_GATE=PASS');
});
