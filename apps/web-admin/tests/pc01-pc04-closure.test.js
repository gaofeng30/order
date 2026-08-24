const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const root = path.resolve(__dirname, '..');

function settle() { return new Promise(resolve => setImmediate(resolve)); }

function loadApi(replies, calls) {
  const window = {
    crypto: { randomUUID: () => 'pc-pages-closure' },
    sessionStorage: { getItem: () => 'owner-session' },
    fetch: async (url, options) => {
      calls.push({ url: String(url), options: options || {} });
      const body = replies.shift();
      return {
        ok: true,
        status: 200,
        headers: { get: () => 'application/json' },
        json: async () => body,
      };
    },
  };
  vm.runInNewContext(fs.readFileSync(path.join(root, 'data/api.js'), 'utf8'), { window, URL, Blob, FormData });
  return window.Api;
}

test('PC02 order search sends the selected business date with pickup-number or phone query', async () => {
  const calls = [];
  const api = loadApi([{ orders: [], next_after_id: null }], calls);

  await api.searchOrders('0007', { date: '2026-08-23' });

  const request = new URL(calls[0].url, 'http://order.local');
  assert.equal(request.pathname, '/api/v1/admin/orders');
  assert.equal(request.searchParams.get('q'), '0007');
  assert.equal(request.searchParams.get('date'), '2026-08-23');
});

test('PC04 corrupt quote snapshot is rendered as a data-validation failure', async () => {
  const calls = [];
  const api = loadApi([{
    prepayments: [{
      id: '71', out_trade_no: 'ORDER-PAY-71', transaction_id: 'WX-71',
      amount_cents: 1250, blocking_reason: 'QUOTE_SNAPSHOT_INVALID',
      contact_name: '验收顾客', phone_masked: '138****0001', items: [],
    }],
    next_after_id: null,
  }], calls);

  const pending = await api.listPendingPayments();

  assert.equal(pending.length, 1);
  assert.equal(pending[0].cause, '数据校验不通过');
  assert.equal(pending[0].causeDetail, 'QUOTE_SNAPSHOT_INVALID');
});

test('PC03 seven-day shortcut keeps the exact service-date range in UTC+8', async () => {
  const summaries = [];
  const from = { value: '' };
  const to = { value: '' };
  const quickToday = { dataset: { quick: 'today' } };
  const quickSeven = { dataset: { quick: '7' } };
  const exportButton = {};
  const nodes = {
    '#f-from': from,
    '#f-to': to,
    '#sum-host': { innerHTML: '' },
    '#fin-tabs': { innerHTML: '' },
    '#fin-host': { innerHTML: '' },
    '[data-export]': exportButton,
  };
  const host = {
    innerHTML: '',
    querySelector: selector => nodes[selector] || null,
    querySelectorAll: selector => selector === '[data-quick]' ? [quickToday, quickSeven] : [],
  };
  const window = {
    Api: {
      today: () => '2026-08-24',
      financeSummary: async range => { summaries.push({ ...range }); return { gross: 0, count: 0, refundAmount: 0, refundCount: 0, pendingCount: 0, net: 0, unbuiltCount: 0, unbuiltAmount: 0, discountCut: 0, staffCount: 0 }; },
      listPayments: async () => [],
      listRefunds: async () => [],
      yuan: cents => (Number(cents) / 100).toFixed(2),
    },
    Table: { render: () => '', esc: String, money: String, pill: String },
    Icon: { svg: () => '' },
    Toast: { show() {} },
    Pages: {},
  };
  vm.runInNewContext(fs.readFileSync(path.join(root, 'pages/finance.js'), 'utf8'), { window, URL, Blob, setTimeout });

  window.Pages.finance.render(host);
  await settle();
  quickSeven.onclick();
  await settle();

  assert.deepEqual(summaries.at(-1), { from: '2026-08-18', to: '2026-08-24' });
  assert.equal(from.value, '2026-08-18');
  assert.equal(to.value, '2026-08-24');
});
