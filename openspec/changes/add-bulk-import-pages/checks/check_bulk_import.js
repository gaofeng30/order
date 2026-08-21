#!/usr/bin/env node
/* PC 批量导入门禁。用法: node check_bulk_import.js <repo-root>
   契约层的 .xlsx 解析在 Node vm 沙箱中真实执行，夹具由 xlsx-fixture.js 手工构造，
   同时覆盖 stored 与 deflate-raw 两种 ZIP 压缩方式。 */
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { buildXlsx } = require('./xlsx-fixture.js');

/* 注意：vm 沙箱内构造的数组/对象与本 realm 原型不同，assert.deepEqual 会误报
   「same structure but not reference-equal」。跨沙箱的结构比较一律用 JSON.stringify。 */

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
  const sb = {
    window: {}, console, setTimeout, clearTimeout, Promise,
    DecompressionStream, TextDecoder, TextEncoder, Response, Uint8Array, DataView, ArrayBuffer,
  };
  sb.globalThis = sb;
  const c = vm.createContext(sb);
  for (const rel of ['data/xlsx.js', 'data/seed.js', 'data/api.js']) {
    vm.runInContext(read(rel), c, { filename: rel });
  }
  const w = sb.window;
  w.__store = {
    menu: JSON.parse(JSON.stringify(w.Seed.MENU)),
    cats: JSON.parse(JSON.stringify(w.Seed.ADMIN_CATS)),
    staff: JSON.parse(JSON.stringify(w.Seed.STAFF_WHITELIST)),
    settings: JSON.parse(JSON.stringify(w.Seed.SETTINGS)),
    store: { status: '营业中' },
  };
  return w;
};
const file = (buf, name = 'x.xlsx') => ({ name, arrayBuffer: () => Promise.resolve(buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength)) });

check('the contract exposes a local xlsx reader and four import methods', () => {
  const w = ctx();
  assert.equal(typeof w.Xlsx.readRows, 'function', 'window.Xlsx.readRows missing');
  for (const m of ['previewProductImport', 'commitProductImport', 'previewStaffImport', 'commitStaffImport']) {
    assert.equal(typeof w.Api[m], 'function', `contract missing ${m}`);
  }
});

check('the reader handles both stored and deflated sheets and shared strings', () => {
  const w = ctx();
  const rows = [['姓名', '手机号'], ['张三', '13800001111'], ['李四', '13800002222']];
  return w.Xlsx.readRows(file(buildXlsx(rows)))
    .then(got => {
      assert.equal(JSON.stringify(got), JSON.stringify(rows), 'deflated sheet mis-parsed');
      return w.Xlsx.readRows(file(buildXlsx(rows, { storedSheet: true })));
    })
    .then(got => assert.equal(JSON.stringify(got), JSON.stringify(rows), 'stored sheet mis-parsed'));
});

check('a non-xlsx upload is refused', () => {
  const w = ctx();
  return w.Api.previewStaffImport(file(Buffer.from('not a zip'), 'a.csv'))
    .then(() => { throw new Error('accepted a non-xlsx file'); }, () => {});
});

check('a missing required column aborts the whole file', () => {
  const w = ctx();
  return w.Api.previewStaffImport(file(buildXlsx([['姓名'], ['张三']])))
    .then(() => { throw new Error('accepted a file without the phone column'); }, () => {});
});

check('staff preview counts, reports row numbers and ignores unknown columns', () => {
  const w = ctx();
  const existing = w.Seed.STAFF_WHITELIST[0];
  const rows = [
    ['姓名', '手机号', '工号'],            // 工号为未知列，应被忽略并提示
    ['新员工甲', '13900001111', 'A1'],     // 新增
    [existing.name, existing.phone, 'A2'], // 更新
    ['缺号', '', 'A3'],                    // 异常：手机号必填
    ['', '13900003333', 'A4'],             // 异常：姓名必填
    ['新员工甲', '13900001111', 'A5'],     // 异常：同文件内重复
  ];
  return w.Api.previewStaffImport(file(buildXlsx(rows))).then(p => {
    assert.equal(p.added, 1, `added=${p.added}`);
    assert.equal(p.updated, 1, `updated=${p.updated}`);
    assert.equal(p.errors.length, 3, `errors=${p.errors.length}`);
    assert.equal(JSON.stringify(p.errors.map(e => e.row)), '[4,5,6]', 'row numbers must be 1-based sheet rows');
    for (const e of p.errors) assert.ok(e.reason && e.reason.length > 0, 'error needs a reason');
    assert.equal(JSON.stringify(p.ignoredColumns), JSON.stringify(['工号']), 'unknown column not reported');
    assert.ok(p.token, 'preview must return a commit token');
    // 预览阶段不得写入
    assert.equal(w.__store.staff.length, w.Seed.STAFF_WHITELIST.length, 'preview wrote data');
  });
});

check('staff commit is idempotent and never re-enables a disabled record', () => {
  const w = ctx();
  const disabled = w.__store.staff.find(r => !r.enabled);
  assert.ok(disabled, 'seed needs a disabled record');
  const before = w.__store.staff.length;
  const rows = [['姓名', '手机号'], ['新甲', '13900001111'], [disabled.name, disabled.phone]];
  let token;
  return w.Api.previewStaffImport(file(buildXlsx(rows)))
    .then(p => { token = p.token; return w.Api.commitStaffImport(token); })
    .then(r => {
      assert.equal(r.added, 1); assert.equal(r.updated, 1);
      assert.equal(w.__store.staff.length, before + 1);
      const after = w.__store.staff.find(x => x.phone === disabled.phone);
      assert.equal(after.enabled, false, 'commit re-enabled a disabled record');
      assert.equal(after.joinAt, disabled.joinAt, 'commit reset joinAt');
      assert.equal(after.spend, disabled.spend, 'commit reset spend');
      return w.Api.commitStaffImport(token);   // 重复提交
    })
    .then(r2 => {
      assert.equal(r2.duplicate, true, 'repeat commit must be marked duplicate');
      assert.equal(w.__store.staff.length, before + 1, 'repeat commit wrote again');
    });
});

check('product preview only adds, flags duplicates and lists new categories', () => {
  const w = ctx();
  const existing = w.Seed.MENU[0];
  const rows = [
    ['菜品名称', '售价', '分类', '餐段可售', '描述'],
    ['夏日冷面', '18', '夏日凉菜', '午餐', '爽口'],      // 新增 + 新分类
    ['冰镇酸梅汤', '8', '夏日凉菜', '全天', ''],          // 新增 + 同批同名分类只建一次
    [existing.name, '30', existing.cat, '全天', ''],     // 异常：只新增不更新
    ['价格错', '三十二', existing.cat, '全天', ''],       // 异常：售价非数值
    ['餐段错', '12', existing.cat, '中午', ''],           // 异常：餐段非三选一
  ];
  return w.Api.previewProductImport(file(buildXlsx(rows, { numericCols: [1] }))).then(p => {
    assert.equal(p.added, 2, `added=${p.added}`);
    assert.equal(p.updated, 0, 'product import must never update');
    assert.equal(p.errors.length, 3, `errors=${p.errors.length}`);
    assert.equal(JSON.stringify(p.errors.map(e => e.row)), '[4,5,6]');
    assert.equal(JSON.stringify(p.newCategories), JSON.stringify(['夏日凉菜']), 'new categories not deduped or not reported');
    assert.equal(w.__store.menu.length, w.Seed.MENU.length, 'preview wrote data');
  });
});

check('product commit creates categories once and leaves existing products untouched', () => {
  const w = ctx();
  const existing = JSON.parse(JSON.stringify(w.Seed.MENU[0]));
  const catsBefore = w.__store.cats.length;
  const rows = [
    ['菜品名称', '售价', '分类', '餐段可售', '描述'],
    ['夏日冷面', '18', '夏日凉菜', '午餐', '爽口'],
    ['冰镇酸梅汤', '8', '夏日凉菜', '全天', ''],
    [existing.name, '999', existing.cat, '全天', '试图覆盖'],
  ];
  return w.Api.previewProductImport(file(buildXlsx(rows, { numericCols: [1] })))
    .then(p => w.Api.commitProductImport(p.token))
    .then(r => {
      assert.equal(r.added, 2);
      assert.equal(w.__store.cats.length, catsBefore + 1, 'category created more than once');
      const kept = w.__store.menu.find(m => m.id === existing.id);
      assert.equal(kept.price, existing.price, 'existing product was overwritten');
      assert.equal(kept.desc, existing.desc, 'existing product description was overwritten');
      const created = w.__store.menu.find(m => m.name === '夏日冷面');
      assert.equal(created.meal, 'lunch');
      assert.equal(created.status, 'on', 'imported product must default to sellable');
      assert.ok(!created.img || created.imgs.length === 0, 'import must not fabricate images');
    });
});

check('both import pages are registered and reachable', () => {
  for (const p of ['pages/product-import.js', 'pages/staff-import.js']) {
    assert.equal(fs.existsSync(path.join(WA, p)), true, `${p} missing`);
    assert.match(read('index.html'), new RegExp(p.replace('/', '\\/')), `index.html does not load ${p}`);
  }
  const app = read('app.js');
  assert.match(app, /r: 'product-import'/, 'sidebar has no product-import route');
  assert.match(app, /r: 'staff-import'/, 'sidebar has no staff-import route');
  assert.match(read('index.html'), /data\/xlsx\.js/, 'index.html does not load the reader');
  // 页面只调契约，不自行解析
  for (const p of ['pages/product-import.js', 'pages/staff-import.js']) {
    assert.doesNotMatch(read(p), /Xlsx\.readRows|DecompressionStream/, `${p} parses the file itself`);
  }
  assert.match(read('pages/product-import.js'), /图片/, 'product import page must state that images are not imported');
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
  if (fails.length) { console.log(fails.map(f => `  ${f}`).join('\n')); console.log('BULK_IMPORT_GATE=FAIL'); process.exit(1); }
  console.log('BULK_IMPORT_GATE=PASS');
});
