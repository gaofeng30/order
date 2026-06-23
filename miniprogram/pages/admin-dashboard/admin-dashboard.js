const data = require('../../utils/data.js');
const { nav, itemsSummary, advanceOrder, advanceMeta } = require('../../utils/util.js');

const BIZ = ['营业中', '休息中', '已截单'];
const KPI = [
  { k: '当日营收', v: '¥1,286', ic: 'wallet', c: '#467a32' },
  { k: '当日订单', v: '38', ic: 'receipt', c: '#2a5fa6' },
  { k: '当月营收', v: '¥32,840', ic: 'chart', c: '#a4873f' },
  { k: '当月订单', v: '906', ic: 'calendar', c: '#5e6354' },
];

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    biz: BIZ,
    kpi: KPI,
    storeStatus: '营业中',
    todos: [],
    live: [],
    rank: [],
  },
  onShow() { this.build(); },
  bizColor(b) { return b === '营业中' ? '#3d6b2f' : (b === '已截单' ? '#8a8f7c' : '#c98b3a'); },
  build() {
    const g = getApp().globalData;
    const orders = g.aOrders;
    const pend = orders.filter(o => o.status === '待制作').length;
    const overtime = orders.filter(o => o.status === '待取餐' && o.mins >= 22).length;
    const lowStock = 2;
    const todos = [];
    if (pend) todos.push({ n: pend, label: '单待制作', tone: 'warn', to: '待制作' });
    if (overtime) todos.push({ n: overtime, label: '单待取超时', tone: 'info', to: '待取餐' });
    if (lowStock) todos.push({ n: lowStock, label: '个库存告急', tone: 'mute', to: '' });

    const live = orders.filter(o => ['待制作', '待取餐'].includes(o.status)).slice(0, 3)
      .map(o => ({ ...o, summary: itemsSummary(o.items), meta: advanceMeta(o.status) }));

    const max = data.RANK[0].sold;
    const rank = data.RANK.map(r => {
      const m = data.itemById(r.id);
      return { id: r.id, name: m.name, sold: r.sold, pct: Math.round(r.sold / max * 100) };
    });

    const todosColored = todos.map(t => ({ ...t, dot: t.tone === 'warn' ? '#d2483a' : (t.tone === 'info' ? '#2a5fa6' : '#bcbfa8') }));
    this.setData({ storeStatus: g.store.status, todos: todosColored, live, rank });
  },
  setBiz(e) {
    const b = e.currentTarget.dataset.b;
    getApp().globalData.store.status = b;
    this.setData({ storeStatus: b });
    this.selectComponent('#toast').show('已切换为「' + b + '」', { icon: b === '营业中' ? 'check' : 'power' });
  },
  toTodo(e) {
    const to = e.currentTarget.dataset.to;
    if (!to) return;
    nav.go('admin-orders', { lane: to });
  },
  kpiTap(e) { this.selectComponent('#toast').show(e.currentTarget.dataset.k + '明细 · 建设中', { icon: 'chart' }); },
  toAllOrders() { nav.replace('admin-orders'); },
  advance(e) {
    advanceOrder(e.currentTarget.dataset.id, this.selectComponent('#toast'), () => this.build());
  },
  viewOrder(e) {
    const o = getApp().globalData.aOrders.find(x => x.id === e.currentTarget.dataset.id);
    getApp().globalData._aSel = o;
    nav.go('admin-order-detail', { id: o.id });
  },
  rankPress(e) {
    this.selectComponent('#toast').show('「' + e.currentTarget.dataset.name + '」已置售罄', { icon: 'tag' });
  },
});
