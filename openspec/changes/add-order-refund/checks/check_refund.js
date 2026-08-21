#!/usr/bin/env node
/* 主账号发起退款门禁（PRD §6.7、§7.1、§7.7）。用法: node check_refund.js <repo-root>
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
    accounts: JSON.parse(JSON.stringify(w.Seed.MERCHANT_ACCOUNTS)),
    settings: JSON.parse(JSON.stringify(w.Seed.SETTINGS)),
    store: { status: '营业中' },
  };
  return w;
};
/* §7.1 旁路：已预约 由主账号取消进退款中；制作中 / 待取餐 / 已完成 由主账号发起退款进退款中 */
const REFUNDABLE = ['已预约', '制作中', '待取餐', '已完成'];
const pick = (w, st) => w.__store.aOrders.find(o => o.status === st);

check('the refund contract offers no way to refund a partial amount', () => {
  const w = ctx();
  assert.equal(typeof w.Api.refundOrder, 'function', 'contract missing refundOrder');
  /* §7.7 一期只支持原路全额退款，部分退款必须拒绝且不创建退款记录。
     最硬的拒绝方式是接口里根本没有金额入参 —— 无法表达的请求不需要校验。 */
  assert.equal(w.Api.refundOrder.length <= 2, true,
    `refundOrder takes ${w.Api.refundOrder.length} params; a full refund needs only (id, reason)`);
  const src = read('data/api.js');
  const body = src.slice(src.indexOf('function refundOrder'), src.indexOf('function refundOrder') + 1600);
  assert.doesNotMatch(body, /\bamount\s*[,)]/, 'refundOrder accepts a caller-supplied amount');
});

check('every refund is for the full order amount', () => {
  const w = ctx();
  /* 既有数据与新产生的退款都必须满足 退款金额 === 订单实付 */
  for (const o of w.Seed.ADMIN_ORDERS) {
    if (!o.refund) continue;
    assert.equal(o.refund.amount, o.total, `${o.no} seeds a partial refund (${o.refund.amount} of ${o.total})`);
  }
  const target = pick(w, '待取餐');
  return w.Api.refundOrder(target.id, '测试').then(r => {
    assert.equal(r.refund.amount, target.total, 'a new refund is not the full order amount');
  });
});

check('a refund can start from any of the four live states', () => {
  const w = ctx();
  return Promise.all(REFUNDABLE.map(st => {
    const o = pick(w, st);
    assert.ok(o, `seed has no ${st} order to refund`);
    return w.Api.refundOrder(o.id, `退款：${st}`).then(r => {
      assert.equal(r.status, '退款中', `${st} order went to ${r.status}, not 退款中`);
      /* 只有微信确认退款成功才是 已退款（§7.7）。前端不得直接置为终态。 */
      assert.notEqual(r.status, '已退款', 'the PC admin must not settle a refund by itself');
      assert.equal(r.refund.status, '退款中', 'refund record is not pending');
    });
  }));
});

check('an already refunding or refunded order cannot be refunded again', () => {
  const w = ctx();
  return Promise.all(['退款中', '已退款'].map(st => {
    const o = pick(w, st);
    assert.ok(o, `seed has no ${st} order`);
    const before = JSON.stringify(o.refund);
    return w.Api.refundOrder(o.id, '重复退款')
      .then(() => { throw new Error(`refunding a ${st} order was allowed`); },
            () => { assert.equal(JSON.stringify(w.__store.aOrders.find(x => x.id === o.id).refund), before,
                                 `${st} order's refund record was overwritten`); });
  }));
});

check('repeating a refund is idempotent and creates no second record', () => {
  const w = ctx();
  const o = pick(w, '制作中');
  return w.Api.refundOrder(o.id, '第一次').then(first =>
    w.Api.refundOrder(o.id, '第二次').then(
      () => { throw new Error('a repeated refund was accepted'); },
      () => w.Api.listRefunds({}).then(list => {
        const mine = list.filter(r => r.orderId === o.id);
        assert.equal(mine.length, 1, `${mine.length} refund records for one order`);
        assert.equal(mine[0].no, first.refund.no, 'the refund number changed on retry');
      })));
});

check('a refund requires a reason and records the logged-in operator', () => {
  const w = ctx();
  assert.equal(typeof w.Api.currentAccount, 'function', 'contract missing currentAccount');
  const me = w.Api.currentAccount();
  assert.equal(me.role, 'owner', 'the PC admin session is not an owner');
  const blank = pick(w, '待取餐');
  return w.Api.refundOrder(blank.id, '   ').then(
    () => { throw new Error('a refund with a blank reason was accepted'); },
    () => {
      const o = pick(w, '已完成');
      return w.Api.refundOrder(o.id, '客户未取餐').then(r => {
        assert.equal(r.refund.reason, '客户未取餐', 'the reason was not recorded');
        assert.equal(r.refund.operator, me.name, `operator is ${r.refund.operator}, not the logged-in ${me.name}`);
        assert.match(r.refund.at, /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/, 'refund time is not a timestamp');
      });
    });
});

check('the uncollected filter is a query, not a seventh state', () => {
  const w = ctx();
  /* §6.7：营业日结束后仍处于 待取餐 的订单，作为筛选条件提供，不是订单状态 */
  assert.equal(typeof w.Api.today, 'function', 'contract missing today()');
  const today = w.Api.today();
  assert.equal(w.Api.LANES.includes('未取餐'), false, '未取餐 leaked into the lane set');
  assert.equal(Object.hasOwn(w.Api.ACT, '未取餐'), false, '未取餐 leaked into the state set');
  return w.Api.listOrders('待取餐', { uncollected: true }).then(list => {
    const expect = w.Seed.ADMIN_ORDERS.filter(o => o.status === '待取餐' && o.pickupDate < today);
    assert.ok(expect.length > 0, 'seed has no order left uncollected past its business day');
    assert.equal(JSON.stringify(list.map(o => o.no).sort()), JSON.stringify(expect.map(o => o.no).sort()),
      'uncollected filter does not match 待取餐 orders past their business day');
    return w.Api.listOrders('待取餐').then(all => {
      assert.ok(all.length > list.length, 'the filter excludes nothing; today\'s 待取餐 orders leak in');
    });
  });
});

check('the orders page offers a refund and never an undo', () => {
  const src = read('pages/orders.js');
  assert.match(src, /Api\.refundOrder\(/, 'orders page never calls refundOrder');
  assert.match(src, /uncollected/, 'orders page offers no uncollected filter');
  /* §7.1 生产禁止撤销或回退已完成的转换 */
  assert.doesNotMatch(src, /revertOrder|onUndo|data-undo/, 'orders page offers an undo');
  /* 退款是不可逆动作，必须二次确认并说明后果 */
  assert.match(src, /Modal\.|confirm/i, 'refund is not behind a confirmation');
});

check('the finance page no longer advertises partial refunds', () => {
  const w = ctx();
  const src = read('pages/finance.js');
  /* 断言的是能力事实：契约不再导出部分退款标记，页面也不再渲染它 */
  assert.doesNotMatch(read('data/api.js'), /partial:\s*o\.refund\.amount\s*<\s*o\.total/,
    'the contract still computes a partial-refund flag');
  assert.doesNotMatch(src, /r\.partial/, 'the finance page still branches on a partial-refund flag');
  /* 归集与净额等式仍须成立 */
  const D = '2026-08-21';
  return w.Api.financeSummary({ from: D, to: D }).then(s => {
    assert.equal(s.net, s.gross - s.refundAmount, 'net equation broke');
  });
});

check('both pages take the business day from one place', () => {
  const w = ctx();
  assert.match(w.Api.today(), /^\d{4}-\d{2}-\d{2}$/, 'today() is not a date');
  for (const rel of ['pages/finance.js', 'pages/orders.js']) {
    assert.doesNotMatch(read(rel), /['"]20\d\d-\d\d-\d\d['"]/, `${rel} hardcodes a business day`);
  }
});

/* 本项接管 archive/2026-08-21-add-finance-page 的
   'the summary nets refunds off receipts by amount, not by order count'。
   那条用一笔部分退款来区分「按金额扣」与「按笔数扣」，而 §7.7 明写一期只支持
   原路全额退款、部分退款必须拒绝 —— 那笔种子数据本身就违反 PRD，是上一个
   change 的错误。这里换成 PRD 允许的区分数据：跨区间的退款。
   原断言中的净额等式与整数分要求在下方原样保留。 */
check('a refund settled outside its payment range still nets', () => {
  const w = ctx();
  /* 种子里有一笔 08-21 付款、08-22 到账的退款。按 08-22 筛选时区间内没有任何收款，
     净额必须是负的退款额 —— 一个"按被退订单笔数扣"的实现在这里无从扣起，会给出 0。 */
  const D = '2026-08-22';
  return w.Api.financeSummary({ from: D, to: D }).then(s => {
    assert.equal(s.count, 0, 'the range should hold no receipts');
    assert.ok(s.refundAmount > 0, 'the cross-range refund did not land in its settlement day');
    assert.equal(s.gross, 0, `gross ${s.gross} should be zero`);
    assert.equal(s.net, -s.refundAmount, `net ${s.net} != ${-s.refundAmount}`);
    for (const k of ['gross', 'refundAmount', 'net']) {
      assert.equal(Number.isInteger(s[k]), true, `${k} is not an integer number of cents`);
    }
  }).then(() => {
    /* 接管原断言：正常区间的三项与净额等式 */
    const D = '2026-08-21';
    const S = w.Seed.ADMIN_ORDERS;
    const gross = S.filter(o => o.paidAt.slice(0, 10) === D).reduce((a, o) => a + o.total, 0);
    const refund = S.filter(o => o.refund && o.refund.at.slice(0, 10) === D).reduce((a, o) => a + o.refund.amount, 0);
    return w.Api.financeSummary({ from: D, to: D }).then(s => {
      assert.equal(s.gross, gross, `gross ${s.gross} != ${gross}`);
      assert.equal(s.refundAmount, refund, `refundAmount ${s.refundAmount} != ${refund}`);
      assert.equal(s.net, gross - refund, `net ${s.net} != ${gross - refund}`);
      assert.equal(s.count, S.filter(o => o.paidAt.slice(0, 10) === D).length, 'count is wrong');
    });
  });
});

check('the uncollected filter clears whenever the lane changes', () => {
  const src = read('pages/orders.js');
  /* 切泳道与退款后跳转都必须把它清掉，否则目标泳道会被筛成空列表 */
  const clears = [...src.matchAll(/uncollected\s*=\s*false/g)].length;
  assert.ok(clears >= 2, `uncollected is only cleared ${clears} time(s); lane switch and post-refund jump both need it`);
  assert.match(src, /lane\s*!==\s*'待取餐'[\s\S]{0,60}uncollected\s*=\s*false/,
    'switching away from 待取餐 does not clear the filter');
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
  if (fails.length) { console.log(fails.map(f => `  ${f}`).join('\n')); console.log('REFUND_GATE=FAIL'); process.exit(1); }
  console.log('REFUND_GATE=PASS');
});
