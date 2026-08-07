/* 会员新增 / 编辑 —— 二期能力，不在一期合同范围 */
const api = require('../../utils/api.js');
const { discountLabel } = require('../../utils/promo.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    isEdit: false,
    m: null,
    initial: '会',
    levels: [],
    levelLabel: '—',
    f: { id: '', phone: '', name: '', levelId: '', org: '', dept: '', jobNo: '', remark: '', enabled: true },
  },

  onLoad(opts) {
    api.listLevels().then(levels => {
      const lvs = levels.map(l => ({ id: l.id, name: l.name, label: discountLabel(l.discount), discount: l.discount }));
      this.setData({ levels: lvs });
      if (!opts.id) {
        this.setData({ 'f.levelId': lvs.length ? lvs[0].id : '' }, () => this.syncLevelLabel());
        return;
      }
      api.getMember(opts.id).then(m => {
        this.setData({
          isEdit: true, m,
          initial: m.name ? m.name[0] : '会',
          f: {
            id: m.id, phone: m.phone, name: m.name, levelId: m.levelId,
            org: m.org, dept: m.dept, jobNo: m.jobNo, remark: m.remark, enabled: m.enabled,
          },
        }, () => this.syncLevelLabel());
      }).catch(() => this.toast('会员不存在', 'warn'));
    });
  },

  syncLevelLabel() {
    const lv = this.data.levels.find(l => l.id === this.data.f.levelId);
    this.setData({ levelLabel: lv ? lv.label : '—' });
  },

  toast(msg, icon) { this.selectComponent('#toast').show(msg, { icon: icon || 'check' }); },

  onInput(e) {
    const k = e.currentTarget.dataset.k;
    let v = e.detail.value;
    if (k === 'phone') v = v.replace(/\D/g, '').slice(0, 11);
    const patch = { ['f.' + k]: v };
    if (k === 'name') patch.initial = v ? v[0] : '会';
    this.setData(patch);
  },
  pickLevel(e) {
    this.setData({ 'f.levelId': e.currentTarget.dataset.id }, () => this.syncLevelLabel());
  },
  toggleEnabled() { this.setData({ 'f.enabled': !this.data.f.enabled }); },

  cancel() { wx.navigateBack({ fail: () => wx.reLaunch({ url: '/pages/admin-members/admin-members' }) }); },

  save() {
    api.saveMember(this.data.f)
      .then(() => { this.toast('已保存'); setTimeout(() => this.cancel(), 600); })
      .catch(err => this.toast(err.message, 'warn'));
  },

  askDelete() {
    wx.showModal({
      title: '移出名单',
      content: `将「${this.data.f.name || this.data.f.phone}」从会员名单中移除？移除后该用户按原价结算，且无券可用。`,
      confirmText: '移除',
      confirmColor: '#b4483c',
      success: res => {
        if (!res.confirm) return;
        api.deleteMember(this.data.f.id)
          .then(() => { this.toast('已移出名单', 'box'); setTimeout(() => this.cancel(), 600); })
          .catch(err => this.toast(err.message, 'warn'));
      },
    });
  },
});
