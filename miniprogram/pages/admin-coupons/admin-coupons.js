/* 优惠券列表 —— 二期能力，不在一期合同范围 */
const api = require('../../utils/api.js');
const data = require('../../utils/data.js');
const { nav } = require('../../utils/util.js');
const promo = require('../../utils/promo.js');

const md = d => (d || '').slice(5);   // '2026-08-01' → '08-01'

// 券的展示状态：停用 > 未开始 > 已过期 > 进行中
function stateOf(c, today) {
  if (!c.enabled) return { id: 'off', badge: '已停用', tone: 'mute', dim: true };
  if (today < c.start) return { id: 'upcoming', badge: '未开始', tone: 'info', dim: false };
  if (today > c.end) return { id: 'expired', badge: '已过期', tone: 'mute', dim: true };
  return { id: 'live', badge: '进行中', tone: 'ok', dim: false };
}

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { tab: 'all', tabs: [], list: [] },
  onShow() { this.load(); },

  load() {
    Promise.all([api.listCoupons(), api.listLevels()]).then(([cps, levels]) => {
      const all = cps.map(c => {
        const st = stateOf(c, data.TODAY);
        return {
          id: c.id, raw: c, state: st.id,
          face: promo.faceText(c), cond: promo.condLabel(c), name: c.name,
          scope: promo.scopeLabel(c), period: `${md(c.start)} 至 ${md(c.end)}`,
          tags: c.levelIds.map(id => { const l = levels.find(x => x.id === id); return l ? l.name : '等级已删除'; }),
          badge: st.badge, badgeTone: st.tone, dim: st.dim,
          enabled: c.enabled, perLimit: c.perLimit,
        };
      });
      const cnt = id => (id === 'all' ? all.length : all.filter(x => x.state === id).length);
      this.setData({
        _all: all,
        tabs: [
          { id: 'all', name: '全部', count: cnt('all') },
          { id: 'live', name: '进行中', count: cnt('live') },
          { id: 'upcoming', name: '未开始', count: cnt('upcoming') },
          { id: 'expired', name: '已过期', count: cnt('expired') },
          { id: 'off', name: '已停用', count: cnt('off') },
        ],
        list: this.data.tab === 'all' ? all : all.filter(x => x.state === this.data.tab),
      });
    });
  },

  toast(msg, icon) { this.selectComponent('#toast').show(msg, { icon: icon || 'check' }); },
  switchTab(e) {
    const tab = e.currentTarget.dataset.id;
    const all = this.data._all || [];
    this.setData({ tab, list: tab === 'all' ? all : all.filter(x => x.state === tab) });
  },
  toNew() { nav.go('admin-coupon-edit'); },
  edit(e) { nav.go('admin-coupon-edit', { id: e.currentTarget.dataset.id }); },
  toggle(e) {
    const id = e.currentTarget.dataset.id;
    const row = this.data._all.find(x => x.id === id);
    api.setCouponEnabled(id, !row.enabled).then(c => {
      this.load();
      this.toast(c.enabled ? '已启用' : '已停用', c.enabled ? 'check' : 'box');
    });
  },
  askDelete(e) {
    const id = e.currentTarget.dataset.id;
    const row = this.data._all.find(x => x.id === id);
    wx.showModal({
      title: '删除优惠券',
      content: `删除「${row.name}」后，该券将立即从所有适用会员的卡包中消失。`,
      confirmText: '删除',
      confirmColor: '#b4483c',
      success: r => {
        if (!r.confirm) return;
        api.deleteCoupon(id).then(() => { this.load(); this.toast('已删除', 'box'); });
      },
    });
  },
});
