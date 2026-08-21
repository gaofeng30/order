#!/usr/bin/env node
/* PC「支付待处理」门禁（PRD §7.3）。用法: node check_pending.js <repo-root>
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
  w.__store = {
    aOrders: JSON.parse(JSON.stringify(w.Seed.ADMIN_ORDERS)),
    pending: JSON.parse(JSON.stringify(w.Seed.PENDING_PAYMENTS || [])),
    menu: JSON.parse(JSON.stringify(w.Seed.MENU)),
    accounts: JSON.parse(JSON.stringify(w.Seed.MERCHANT_ACCOUNTS)),
    settings: JSON.parse(JSON.stringify(w.Seed.SETTINGS)),
    store: { status: '营业中' },
  };
  return w;
};
const SIX = ['已预约', '制作中', '待取餐', '已完成', '退款中', '已退款'];
const CAUSES = ['商品已下架', '取餐时间已过', '数据校验不通过'];

check('a pending entry carries the payment facts but is not an order', () => {
  const w = ctx();
  const list = w.Seed.PENDING_PAYMENTS;
  assert.ok(Array.isArray(list) && list.length > 0, 'no pending payment seed');
  for (const p of list) {
    /* 钱已经收到了，所以支付事实必须齐全 —— 否则对不上微信账单也退不了款 */
    for (const k of ['id', 'outTradeNo', 'txnId', 'paidAt', 'amount', 'contact', 'phone',
                     'pickupDate', 'pickupTime', 'mealPeriod', 'items', 'cause', 'detectedAt']) {
      assert.equal(Object.hasOwn(p, k), true, `${p.outTradeNo || p.id} missing ${k}`);
    }
    assert.equal(Number.isInteger(p.amount) && p.amount > 0, true, `${p.outTradeNo}.amount is not a positive integer of cents`);
    assert.match(p.txnId, /^42000\d{13,}$/, `${p.outTradeNo}.txnId is not a wechat transaction id`);
    assert.equal(CAUSES.includes(p.cause), true, `${p.outTradeNo}.cause = ${p.cause} is not one of §7.3's causes`);
    /* §7.3：该链路不引入 异常 状态，条目也不是订单 —— 不得带六态状态或取餐号 */
    assert.equal(Object.hasOwn(p, 'status') && SIX.includes(p.status), false, `${p.outTradeNo} carries an order state`);
    assert.equal(Object.hasOwn(p, 'code'), false, `${p.outTradeNo} already holds a pickup code`);
  }
  assert.equal(new Set(list.map(p => p.txnId)).size, list.length, 'transaction ids are not unique');
  for (const c of CAUSES) assert.ok(list.some(p => p.cause === c), `seed has no entry caused by ${c}`);
});

check('pending entries never leak into the order lanes', () => {
  const w = ctx();
  return w.Api.listOrders('全部').then(all => {
    const nos = new Set(all.map(o => o.no));
    for (const p of w.Seed.PENDING_PAYMENTS) {
      assert.equal(nos.has(p.outTradeNo), false, `${p.outTradeNo} shows up as an order`);
    }
    assert.equal(w.Api.laneCounts()['全部'], w.Seed.ADMIN_ORDERS.length, 'lane counts absorbed the pending entries');
    assert.equal(w.Api.LANES.includes('支付待处理'), false, '支付待处理 leaked into the lane set');
  });
});

check('the contract exposes the pending reads and both manual actions', () => {
  const w = ctx();
  for (const m of ['listPendingPayments', 'rebuildOrder', 'refundPendingPayment', 'pendingPaymentCount']) {
    assert.equal(typeof w.Api[m], 'function', `contract missing ${m}`);
  }
});

check('rebuilding is refused while the blocking cause still holds', () => {
  const w = ctx();
  /* §7.3 补建失败的原因就是这些。原因未解除就建单，等于给一道做不出来的菜发取餐号。 */
  const blocked = w.__store.pending.filter(p => p.cause !== '数据校验不通过');
  assert.ok(blocked.length > 0, 'seed has no blocked entry');
  return Promise.all(blocked.map(p =>
    w.Api.rebuildOrder(p.id).then(
      () => { throw new Error(`${p.outTradeNo} was rebuilt while still ${p.cause}`); },
      e => {
        assert.match(e.message, /下架|已过|校验/, `refusal does not name the cause: ${e.message}`);
        assert.equal(w.__store.aOrders.some(o => o.txnId === p.txnId), false, 'a partial order was created anyway');
      })));
});

check('rebuilding succeeds once the cause is cleared', () => {
  const w = ctx();
  const p = w.__store.pending.find(x => x.cause === '商品已下架');
  assert.ok(p, 'seed has no 商品已下架 entry');
  /* 主账号把菜品重新上架后重试，这是该原因唯一的解法 */
  return Promise.all(p.items.map(it => w.Api.setProductStatus(it[0], 'on')))
    .then(() => w.Api.rebuildOrder(p.id))
    .then(order => {
      assert.equal(order.txnId, p.txnId, 'the rebuilt order lost the transaction id');
      assert.equal(order.total, p.amount, `rebuilt total ${order.total} != paid ${p.amount}`);
      assert.equal(SIX.includes(order.status), true, `rebuilt order state ${order.status} is not one of the six`);
      /* §7.4：距取餐不足 30 分钟直接进 制作中，否则 已预约 */
      assert.equal(['已预约', '制作中'].includes(order.status), true, `rebuilt into ${order.status}`);
      /* §7.8：取餐号 4 位数字，按取餐日期累计且当日唯一 */
      assert.match(order.code, /^\d{4}$/, `pickup code ${order.code} is not four digits`);
      const sameDay = w.__store.aOrders.filter(o => o.pickupDate === order.pickupDate);
      assert.equal(new Set(sameDay.map(o => o.code)).size, sameDay.length, 'pickup code collides on the same business day');
      /* 建单后条目必须离开待处理列表 */
      return w.Api.listPendingPayments().then(rest => {
        assert.equal(rest.some(x => x.id === p.id), false, 'the entry stayed in the pending list after rebuild');
      });
    });
});

check('rebuilding is idempotent and never issues a second pickup code', () => {
  const w = ctx();
  const p = w.__store.pending.find(x => x.cause === '商品已下架');
  return Promise.all(p.items.map(it => w.Api.setProductStatus(it[0], 'on')))
    .then(() => w.Api.rebuildOrder(p.id))
    .then(first => w.Api.rebuildOrder(p.id).then(
      () => { throw new Error('a repeated rebuild was accepted'); },
      () => {
        const made = w.__store.aOrders.filter(o => o.txnId === p.txnId);
        assert.equal(made.length, 1, `${made.length} orders for one prepay record`);
        assert.equal(made[0].code, first.code, 'the pickup code changed on retry');
      }));
});

check('voiding a pending entry refunds the full amount and records the operator', () => {
  const w = ctx();
  const p = w.__store.pending.find(x => x.cause === '取餐时间已过');
  assert.ok(p, 'seed has no 取餐时间已过 entry');
  const me = w.Api.currentAccount();
  return w.Api.refundPendingPayment(p.id, '   ').then(
    () => { throw new Error('a void with a blank reason was accepted'); },
    () => w.Api.refundPendingPayment(p.id, '取餐时间已过，无法补建').then(r => {
      assert.equal(r.refund.amount, p.amount, 'the void did not refund the full amount');
      assert.equal(r.refund.status, '退款中', 'a void must wait for wechat, not settle itself');
      assert.equal(r.refund.operator, me.name, `operator is ${r.refund.operator}, not ${me.name}`);
      assert.equal(r.refund.reason, '取餐时间已过，无法补建', 'the reason was not recorded');
      return w.Api.listPendingPayments().then(rest =>
        assert.equal(rest.some(x => x.id === p.id), false, 'the entry stayed pending after being voided'));
    }));
});

check('a voided entry never becomes an order but does reach the refund ledger', () => {
  const w = ctx();
  const p = w.__store.pending.find(x => x.cause === '取餐时间已过');
  return w.Api.refundPendingPayment(p.id, '取餐时间已过').then(() =>
    w.Api.listOrders('全部').then(all => {
      assert.equal(all.some(o => o.txnId === p.txnId), false, 'voiding created an order');
      return w.Api.listRefunds({}).then(list => {
        const hit = list.find(r => r.txnId === p.txnId);
        assert.ok(hit, 'the voided payment is missing from the refund ledger');
        assert.equal(hit.amount, p.amount, 'refund ledger shows the wrong amount');
        assert.ok(hit.orderNo, 'the refund cannot be traced back to anything');
      });
    }));
});

check('the finance page surfaces the received-but-unbuilt gap', () => {
  const w = ctx();
  /* 钱已在微信账户里。若实收合计不含这些条目又不作说明，本页就会比微信账单少一块，
     而这正是对账页存在的意义 —— 单边账必须被说出来，不能悄悄少算。 */
  const D = '2026-08-21';
  return w.Api.financeSummary({ from: D, to: D }).then(s => {
    /* 字段名不能叫 pendingCount —— 那个已被「未到账退款笔数」占用，同名不同义
       在对账页上就是事故。这里用 unbuilt（已收款未建单）。 */
    for (const k of ['unbuiltCount', 'unbuiltAmount']) {
      assert.equal(Object.hasOwn(s, k), true, `summary missing ${k}`);
    }
    const expect = w.Seed.PENDING_PAYMENTS.filter(p => p.paidAt.slice(0, 10) === D);
    assert.equal(s.unbuiltCount, expect.length, `unbuiltCount ${s.unbuiltCount} != ${expect.length}`);
    assert.equal(s.unbuiltAmount, expect.reduce((a, p) => a + p.amount, 0), 'unbuiltAmount is wrong');
    assert.ok(s.unbuiltCount > 0, 'seed has no pending payment on the reconciliation day');
    assert.notEqual(s.unbuiltCount, s.pendingCount, 'unbuilt and unsettled-refund counts must stay distinct fields');
    /* 净额等式不得因此改变：待处理条目未计入实收，只作旁注 */
    assert.equal(s.net, s.gross - s.refundAmount, 'net equation broke');
  }).then(() => {
    const src = read('pages/finance.js');
    assert.match(src, /unbuiltAmount|unbuiltCount/, 'the finance page ignores the received-but-unbuilt figures');
    assert.match(src, /支付待处理/, 'the finance page does not point at where to resolve them');
  });
});

check('the page is registered, reachable and reads only through the contract', () => {
  assert.equal(fs.existsSync(path.join(WA, 'pages/pending.js')), true, 'pages/pending.js missing');
  assert.match(read('index.html'), /pages\/pending\.js/, 'index.html does not load the page');
  assert.match(read('app.js'), /r: 'pending'/, 'sidebar has no pending route');
  const src = read('pages/pending.js');
  for (const m of ['listPendingPayments', 'rebuildOrder', 'refundPendingPayment']) {
    assert.match(src, new RegExp(`Api\\.${m}\\(`), `page does not call ${m}`);
  }
  assert.doesNotMatch(src, /Seed\.PENDING_PAYMENTS|__store\.pending/, 'page reaches past the contract into the store');
  assert.doesNotMatch(src, /\/\s*100\b/, 'page converts cents to yuan by hand');
  /* 两个动作都不可逆，都必须二次确认 */
  assert.match(src, /Modal\./, 'the manual actions are not behind a confirmation');
});

check('the page explains that the scanner itself is not built', () => {
  const src = read('pages/pending.js');
  /* §7.3 的定时扫描与微信查询接口属后端。页面只是人工出口，
     不说清楚的话主账号会以为这个列表是实时的。 */
  assert.match(src, /定时|扫描|后端|微信/, 'the page never says where these entries come from');
  const prd = fs.readFileSync(path.join(root, 'docs/product/online-ordering-system-prd-0818.md'), 'utf8');
  const gaps = prd.slice(prd.indexOf('## 16.5'), prd.indexOf('## 16.6'));
  assert.match(gaps, /支付待处理|对账兜底/, 'PRD does not record the pending-payment backend gap');
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
  if (fails.length) { console.log(fails.map(f => `  ${f}`).join('\n')); console.log('PENDING_GATE=FAIL'); process.exit(1); }
  console.log('PENDING_GATE=PASS');
});
