const data = require('../../utils/data.js');
const { findByCode, codeHint } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    sims: ['0118', '0112', '0090'],
    code: '',
    match: null,     // { o, err, rows }
  },
  onCode(e) { this.setData({ code: (e.detail.value || '').trim() }); },
  simTap(e) { this.tryVerify(e.currentTarget.dataset.c); },
  manual() { if (this.data.code) this.tryVerify(this.data.code); },
  /* 手工输入只匹配当前营业日的取餐号（§6.6）。跨营业日的号不解析，
     但必须说清它属于哪个营业日 —— 否则和「号不存在」不可区分。 */
  tryVerify(c) {
    const o = findByCode(c);
    if (!o) {
      const hint = codeHint(c);
      this.selectComponent('#toast').show(hint || '无效取餐号「' + c + '」', { icon: 'warn' });
      return;
    }
    let err = '';
    if (o.status === '已完成') err = '该订单已核销';
    else if (o.status === '已退款') err = '该订单已退款，不可核销';
    else if (o.status === '退款中') err = '该订单退款处理中，不可核销';
    else if (o.status !== '待取餐') err = '订单尚未备好';
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
