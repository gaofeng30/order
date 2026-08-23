const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const root = path.resolve(__dirname, '..');

function settle() {
  return new Promise(resolve => setImmediate(resolve));
}

function child() {
  return {
    innerHTML: '',
    value: '',
    dataset: {},
    classList: { contains: () => false },
    querySelector: () => null,
    querySelectorAll: () => [],
  };
}

function renderRoot(selectors) {
  return {
    innerHTML: '',
    querySelector(selector) {
      return selectors[selector] || null;
    },
    querySelectorAll() { return []; },
  };
}

test('dashboard renders all PRD stats from the server, including refund and pending production', async () => {
  const liveHost = child();
  const more = child();
  const host = renderRoot({ '#live-host': liveHost, '[data-more]': more });
  const window = {
    Api: {
      dashboardStats: async () => ({
        today_revenue_cents: 12345,
        today_orders: 7,
        month_revenue_cents: 67890,
        month_orders: 31,
        refund_cents: 405,
        pending_production: 3,
        product_sales: [{ name: '黑椒牛柳', quantity: 9 }],
      }),
      // Deliberately empty: pending-production is a server statistic, not a
      // count inferred from the first page of orders.
      listOrders: async () => [],
      itemsSummary: () => '',
      yuan: cents => (Number(cents) / 100).toFixed(2),
    },
    Table: {
      esc: String,
      render: () => '',
      bind() {},
      pill: String,
    },
    Icon: { svg: () => '' },
    Pages: {},
    App: { go() {} },
  };
  vm.runInNewContext(fs.readFileSync(path.join(root, 'pages/dashboard.js'), 'utf8'), { window });

  window.Pages.dashboard.render(host);
  await settle();

  assert.match(host.innerHTML, /当日营收/);
  assert.match(host.innerHTML, /当月订单/);
  assert.match(host.innerHTML, /退款金额/);
  assert.match(host.innerHTML, /¥4\.05/);
  assert.match(host.innerHTML, />3<\/b><span>单待制作/);
  assert.match(host.innerHTML, /黑椒牛柳/);
});

test('admin order adapter preserves the server Chinese six-state projection and pickup point', async () => {
  const window = {
    crypto: { randomUUID: () => 'transaction-read' },
    sessionStorage: { getItem: () => 'session-token' },
    fetch: async () => ({
      ok: true,
      status: 200,
      headers: { get: () => 'application/json' },
      json: async () => ({ orders: [{
        id: '41', order_no: 'ORDER-41', pickup_number: '0041', state: '已预约',
        pickup_date: '2026-08-25', pickup_time: '12:00', pickup_point: '北门', items: [],
      }] }),
    }),
  };
  vm.runInNewContext(fs.readFileSync(path.join(root, 'data/api.js'), 'utf8'), { window, URL, Blob, FormData });

  const orders = await window.Api.listOrders('全部');
  assert.equal(orders[0].status, '已预约');
  assert.equal(orders[0].pickupPoint, '北门');
  assert.equal(window.Api.canRefund(orders[0].status), true);
});

test('finance adapters consume every existing server cursor page', async () => {
  const calls = [];
  const replies = [
    { payments: [{ id: '1', state: '已完成', items: [] }], next_after_id: '50' },
    { payments: [{ id: '51', state: '已退款', items: [] }], next_after_id: null },
    { refunds: [{ id: '7', state: '退款中' }], next_after_id: '60' },
    { refunds: [{ id: '61', state: '已退款' }], next_after_id: null },
  ];
  const window = {
    sessionStorage: { getItem: () => 'session-token' },
    fetch: async (url) => {
      calls.push(url);
      const body = replies.shift();
      return { ok: true, status: 200, headers: { get: () => 'application/json' }, json: async () => body };
    },
  };
  vm.runInNewContext(fs.readFileSync(path.join(root, 'data/api.js'), 'utf8'), { window, URL, Blob, FormData });

  const range = { from: '2026-08-18', to: '2026-08-24' };
  const payments = await window.Api.listPayments(range);
  const refunds = await window.Api.listRefunds(range);
  assert.equal(payments.map(item => item.id).join(','), '1,51');
  assert.equal(refunds.map(item => String(item.id)).join(','), '7,61');
  assert.match(calls[0], /limit=100/);
  assert.match(calls[1], /after_id=50/);
  assert.match(calls[3], /after_id=60/);
});

test('finance cursor must advance strictly or the adapter fails closed', async () => {
  let calls = 0;
  const window = {
    sessionStorage: { getItem: () => 'session-token' },
    fetch: async () => {
      calls += 1;
      return {
        ok: true,
        status: 200,
        headers: { get: () => 'application/json' },
        json: async () => ({ payments: [], next_after_id: '50' }),
      };
    },
  };
  vm.runInNewContext(fs.readFileSync(path.join(root, 'data/api.js'), 'utf8'), { window, URL, Blob, FormData });

  await assert.rejects(window.Api.listPayments({ from: '2026-08-18', to: '2026-08-24' }), /响应无法解析/);
  assert.equal(calls, 2);
});

test('order detail renders the server payable line subtotal without inventing unit fields', async () => {
  const order = {
    id: '41', no: 'ORDER-41', code: '0041', status: '已预约',
    pickupDate: '2026-08-25', pickupTime: '12:00', pickupPoint: '北门',
    paidAt: '2026-08-24T06:00:00Z', txnId: 'LOCAL-41', contact: '验收用户',
    phone: '138****0000', orderNote: '少盐', subtotal: 2500, discountCut: 0,
    discountRate: 100, total: 2500,
    // This is the real admin HTTP adapter shape: product id, name, quantity,
    // payable line subtotal in cents.
    items: [['9', '黑椒牛柳', 2, 2500]],
  };
  const search = child();
  const lanes = child();
  const uncollected = child();
  const tableHost = child();
  const detailHost = child();
  const host = renderRoot({
    '#f-kw': search,
    '#lanes': lanes,
    '#unc-host': uncollected,
    '#tbl-host': tableHost,
    '#detail': detailHost,
    '#kw-hint': null,
  });
  const window = {
    Api: {
      LANES: ['已预约', '制作中', '待取餐', '已完成', '退款中', '已退款', '全部'],
      laneCounts: () => ({ '已预约': 1, '制作中': 0, '待取餐': 0, '已完成': 0, '退款中': 0, '已退款': 0, '全部': 1 }),
      uncollectedCount: () => 0,
      codeHint: () => '',
      listOrders: async () => [order],
      searchOrders: async () => [order],
      findOrder: id => id === order.id ? order : null,
      itemsSummary: () => '黑椒牛柳×2',
      yuan: cents => Number.isFinite(Number(cents)) ? (Number(cents) / 100).toFixed(2) : 'INVALID',
      canRefund: () => false,
      currentAccount: () => ({ name: '店主' }),
    },
    Table: {
      esc: String,
      render: () => '',
      bind() {},
      pill: String,
      money: value => `¥${value}`,
    },
    Icon: { svg: () => '' },
    Pages: {},
    Modal: { open() {} },
    Toast: { show() {} },
  };
  vm.runInNewContext(fs.readFileSync(path.join(root, 'pages/orders.js'), 'utf8'), { window });

  window.Pages.orders.render(host);
  await settle();

  assert.match(detailHost.innerHTML, /黑椒牛柳/);
  assert.match(detailHost.innerHTML, /¥25\.00/);
  assert.doesNotMatch(detailHost.innerHTML, /INVALID|¥—/);
});
