#!/usr/bin/env node
/* PC 财务与对账页门禁（PRD §6.11）。用法: node check_finance.js <repo-root>
   跨 vm realm 一律用 JSON.stringify 比较，不用 deepEqual。 */
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
  w.__store = { aOrders: JSON.parse(JSON.stringify(w.Seed.ADMIN_ORDERS)),
                settings: JSON.parse(JSON.stringify(w.Seed.SETTINGS)), store: { status: '营业中' } };
  return w;
};
const DAY = '2026-08-21';

check('the contract exposes the reconciliation reads', () => {
  const w = ctx();
  for (const m of ['listPayments', 'listRefunds', 'financeSummary', 'buildPaymentExport']) {
    assert.equal(typeof w.Api[m], 'function', `contract missing ${m}`);
  }
});

check('payments are grouped by payment date, not by business date', () => {
  const w = ctx();
  /* 种子里有一笔 08-20 付款、08-21 取餐的预约单。微信账单以交易时间为准，
     财务页按支付日期归集；若误按营业日期，这笔会跑到 08-21 去。 */
  const cross = w.Seed.ADMIN_ORDERS.find(o => o.paidAt.slice(0, 10) !== o.pickupDate);
  assert.ok(cross, 'seed has no order whose payment date differs from its business date');
  return w.Api.listPayments({ from: DAY, to: DAY }).then(list => {
    assert.equal(list.some(o => o.no === cross.no), false,
      `${cross.no} paid on ${cross.paidAt.slice(0, 10)} leaked into ${DAY} by business date`);
    const expect = w.Seed.ADMIN_ORDERS.filter(o => o.paidAt.slice(0, 10) === DAY).map(o => o.no).sort();
    assert.equal(JSON.stringify(list.map(o => o.no).sort()), JSON.stringify(expect), 'payment day set is wrong');
  });
});

check('refunds are grouped by refund date, not by the order payment date', () => {
  const w = ctx();
  /* 种子里有一笔 08-21 付款、08-22 才退到账的单。退款必须落在到账那天，
     否则和微信账单的退款明细对不上。 */
  const cross = w.Seed.ADMIN_ORDERS.find(o => o.refund && o.refund.at.slice(0, 10) !== o.paidAt.slice(0, 10));
  assert.ok(cross, 'seed has no refund settled on a different day than the payment');
  return Promise.all([
    w.Api.listRefunds({ from: DAY, to: DAY }),
    w.Api.listRefunds({ from: cross.refund.at.slice(0, 10), to: cross.refund.at.slice(0, 10) }),
  ]).then(([sameDay, refundDay]) => {
    assert.equal(sameDay.some(r => r.no === cross.refund.no), false,
      `refund ${cross.refund.no} settled on ${cross.refund.at.slice(0, 10)} leaked into ${DAY}`);
    assert.equal(refundDay.some(r => r.no === cross.refund.no), true, 'refund missing from its own settlement day');
  });
});

check('a refund record carries the four facts PRD 6.11 requires', () => {
  const w = ctx();
  return w.Api.listRefunds({ from: '2026-01-01', to: '2026-12-31' }).then(list => {
    assert.ok(list.length > 0, 'no refunds at all');
    for (const r of list) {
      for (const k of ['no', 'amount', 'status', 'operator']) {
        assert.ok(r[k] != null && String(r[k]) !== '', `refund record missing ${k}`);
      }
      /* 退款记录必须能回溯到订单，否则对账时无法定位 */
      assert.ok(r.orderNo && r.txnId, 'refund record cannot be traced back to its order');
    }
  });
});

check('the summary nets refunds off receipts by amount, not by order count', () => {
  const w = ctx();
  const S = w.Seed.ADMIN_ORDERS;
  const gross = S.filter(o => o.paidAt.slice(0, 10) === DAY).reduce((s, o) => s + o.total, 0);
  const refund = S.filter(o => o.refund && o.refund.at.slice(0, 10) === DAY).reduce((s, o) => s + o.refund.amount, 0);
  /* 种子里有一笔部分退款：按订单数扣会扣掉整单实付，按金额扣才对 */
  const partial = S.find(o => o.refund && o.refund.amount < o.total);
  assert.ok(partial, 'seed has no partial refund to distinguish the two');
  return w.Api.financeSummary({ from: DAY, to: DAY }).then(sum => {
    assert.equal(sum.gross, gross, `gross ${sum.gross} != ${gross}`);
    assert.equal(sum.refundAmount, refund, `refundAmount ${sum.refundAmount} != ${refund}`);
    assert.equal(sum.net, gross - refund, `net ${sum.net} != ${gross - refund}`);
    for (const k of ['gross', 'refundAmount', 'net']) {
      assert.equal(Number.isInteger(sum[k]), true, `${k} is not an integer number of cents`);
    }
    assert.equal(sum.count, S.filter(o => o.paidAt.slice(0, 10) === DAY).length, 'count is wrong');
  });
});

check('the export carries a BOM, a header and one line per payment', () => {
  const w = ctx();
  return Promise.all([w.Api.buildPaymentExport({ from: DAY, to: DAY }), w.Api.listPayments({ from: DAY, to: DAY })])
    .then(([csv, list]) => {
      assert.equal(typeof csv, 'string', 'export is not text');
      /* Excel 不认无 BOM 的 UTF-8 CSV，中文会乱码 —— 商户拿到的就是一堆问号 */
      assert.equal(csv.charCodeAt(0), 0xFEFF, 'export has no UTF-8 BOM');
      const lines = csv.replace(/^﻿/, '').trim().split(/\r?\n/);
      assert.equal(lines.length, list.length + 1, `export has ${lines.length} lines for ${list.length} payments + header`);
      for (const col of ['订单号', '支付时间', '微信交易号', '实付金额']) {
        assert.ok(lines[0].includes(col), `export header missing ${col}`);
      }
      /* 交易号是长数字串。Excel 会把它当数字并转成科学计数法，必须防住 */
      const sample = list[0];
      const row = lines.find(l => l.includes(sample.no));
      assert.ok(row, 'export lost a payment row');
      assert.equal(row.includes(sample.txnId), true, 'export lost the transaction id');
      assert.doesNotMatch(row, /,\d+\.\d{2}E\+/i, 'export lets excel mangle a long number');
      /* 金额必须是元，不能把分直接倒出去 */
      assert.ok(row.includes(w.Api.yuan(sample.total)), `export does not carry ${w.Api.yuan(sample.total)}`);
    });
});

check('the page is registered, reachable and reads only through the contract', () => {
  assert.equal(fs.existsSync(path.join(WA, 'pages/finance.js')), true, 'pages/finance.js missing');
  assert.match(read('index.html'), /pages\/finance\.js/, 'index.html does not load the page');
  assert.match(read('app.js'), /r: 'finance'/, 'sidebar has no finance route');
  const src = read('pages/finance.js');
  for (const m of ['listPayments', 'listRefunds', 'financeSummary', 'buildPaymentExport']) {
    assert.match(src, new RegExp(`Api\\.${m}\\(`), `page does not call ${m}`);
  }
  /* 页面 MUST NOT 自己碰种子或做分转元的算术 */
  assert.doesNotMatch(src, /Seed\.ADMIN_ORDERS|__store\.aOrders/, 'page reaches past the contract into the store');
  assert.doesNotMatch(src, /\/\s*100\b/, 'page converts cents to yuan by hand');
});

check('the page states what it can and cannot reconcile', () => {
  const src = read('pages/finance.js');
  /* 断言的是事实：自动拉取微信账单未实现这件事必须写在页面上，
     否则商户会以为「对上了」是系统核过的。 */
  assert.match(src, /微信/, 'page never mentions the wechat bill it reconciles against');
  assert.match(src, /净额|净收/, 'page does not surface a net figure');
  const prd = fs.readFileSync(path.join(root, 'docs/product/online-ordering-system-prd-0818.md'), 'utf8');
  const gaps = prd.slice(prd.indexOf('## 16.5'), prd.indexOf('## 16.6'));
  assert.match(gaps, /微信.*账单|账单.*核对/, 'PRD does not record automatic bill reconciliation as a backend gap');
});

check('a negative or zero figure is signed correctly', () => {
  const src = read('pages/finance.js');
  /* Api.yuan 只产出数字，负号会落在 ¥ 后面（¥-12.00）。汇总卡必须经过
     一个把符号提到 ¥ 之前的辅助函数，不能直接拼 '¥' + Api.yuan(...)。 */
  const cards = src.slice(src.indexOf('function paintSummary'), src.indexOf('function paintTabs'));
  assert.doesNotMatch(cards, /'[¥−-]*¥'\s*\+\s*Api\.yuan/, 'summary card concatenates ¥ with a possibly negative number');
  assert.match(cards, /amt\(/, 'summary card does not go through the sign helper');
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
  if (fails.length) { console.log(fails.map(f => `  ${f}`).join('\n')); console.log('FINANCE_GATE=FAIL'); process.exit(1); }
  console.log('FINANCE_GATE=PASS');
});
