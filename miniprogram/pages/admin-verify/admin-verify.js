const data = require('../../utils/data.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    sims: ['A118', 'A126', 'A090'],
    code: '',
    match: null,     // { o, err, rows }
  },
  onCode(e) { this.setData({ code: (e.detail.value || '').toUpperCase() }); },
  findOrder(c) {
    return getApp().globalData.aOrders.find(o => o.code.toUpperCase() === c.toUpperCase());
  },
  simTap(e) { this.tryVerify(e.currentTarget.dataset.c); },
  manual() { if (this.data.code) this.tryVerify(this.data.code); },
  tryVerify(c) {
    const o = this.findOrder(c);
    if (!o) { this.selectComponent('#toast').show('无效取餐号「' + c + '」', { icon: 'warn' }); return; }
    let err = '';
    if (o.status === '已完成') err = '该订单已核销';
    else if (o.status !== '待取餐') err = (o.status === '待接单' || o.status === '制作中') ? '订单尚未备好' : '订单状态异常';
    const rows = o.items.map(([id, q, p]) => { const m = data.itemById(id); return { name: m.name, q, p, sub: p * q }; });
    this.setData({ match: { o, err, rows } });
  },
  closeSheet() { this.setData({ match: null }); },
  confirm() {
    const o = this.data.match.o;
    o.status = '已完成';
    this.setData({ match: null, code: '' });
    this.selectComponent('#toast').show('核销成功 · 看板营收/订单已更新', { icon: 'check' });
  },
});
