/* 会员名单 —— 二期能力，不在一期合同范围
   手机号为唯一识别键，与微信授权手机号比对；名单是会员的唯一来源。 */
const api = require('../../utils/api.js');
const { nav } = require('../../utils/util.js');

const maskPhone = p => (p && p.length === 11 ? p.slice(0, 3) + '****' + p.slice(7) : p);

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    kw: '',
    tab: 'all',
    tabs: [],
    list: [],
    total: 0,
    boundCount: 0,
    offCount: 0,
  },
  onShow() { this.load(); },

  load() {
    Promise.all([api.listLevels(), api.listMembers({})]).then(([levels, all]) => {
      const tabs = [{ id: 'all', name: '全部', count: all.total }].concat(
        levels.map(l => ({ id: l.id, name: l.name, count: all.list.filter(m => m.levelId === l.id).length }))
      );
      this.setData({
        tabs,
        total: all.total,
        boundCount: all.list.filter(m => m.bound).length,
        offCount: all.list.filter(m => !m.enabled).length,
        _levels: levels,
      });
      this.build(levels);
    });
  },

  build(levels) {
    const lvs = levels || this.data._levels || [];
    api.listMembers({ kw: this.data.kw, levelId: this.data.tab === 'all' ? '' : this.data.tab }).then(r => {
      this.setData({
        list: r.list.map(m => {
          const lv = lvs.find(l => l.id === m.levelId);
          const org = [m.org, m.dept].filter(Boolean).join(' · ');
          return {
            id: m.id, name: m.name, initial: m.name ? m.name[0] : '会',
            phoneMask: maskPhone(m.phone),
            levelName: lv ? lv.name : '等级已删除',
            orgLine: org ? (m.jobNo ? org + ' · 工号 ' + m.jobNo : org) : '未填写单位信息',
            spend: m.spend, orders: m.orders, bound: m.bound, enabled: m.enabled,
          };
        }),
      });
    });
  },

  onKw(e) { this.setData({ kw: e.detail.value }, () => this.build()); },
  clearKw() { this.setData({ kw: '' }, () => this.build()); },
  switchTab(e) { this.setData({ tab: e.currentTarget.dataset.id }, () => this.build()); },
  edit(e) { nav.go('admin-member-edit', { id: e.currentTarget.dataset.id }); },
  toNew() { nav.go('admin-member-edit'); },
  toImport() { nav.go('admin-member-import'); },
});
