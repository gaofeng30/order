/* 会员等级管理 —— 二期能力，不在一期合同范围
   档位可自由增删改名与排序；删除时必须指定会员去向，并自动清理券的适用等级。 */
const api = require('../../utils/api.js');
const { discountLabel } = require('../../utils/promo.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    levels: [],
    editVisible: false,
    f: { id: '', name: '', zhe: '', desc: '' },
    zheHint: '',
    delVisible: false,
    delLv: null,
    impact: { memberCount: 0, coupons: [] },
    migrateOpts: [],
    migrateTo: '',
  },
  onShow() { this.load(); },

  load() {
    Promise.all([api.listLevels(), api.listMembers({}), api.listCoupons()]).then(([levels, mb, cps]) => {
      this.setData({
        levels: levels.map(l => ({
          id: l.id, name: l.name, desc: l.desc, discount: l.discount,
          label: discountLabel(l.discount),
          count: mb.list.filter(m => m.levelId === l.id).length,
          couponCount: cps.filter(c => c.levelIds.indexOf(l.id) > -1).length,
        })),
      });
    });
  },

  toast(msg, icon) { this.selectComponent('#toast').show(msg, { icon: icon || 'check' }); },

  // ---- 排序 ----
  move(e) {
    const { idx, d } = e.currentTarget.dataset;
    const i = +idx;
    const j = i + (+d);
    const list = this.data.levels;
    if (j < 0 || j >= list.length) return;
    const ids = list.map(l => l.id);
    ids.splice(j, 0, ids.splice(i, 1)[0]);
    api.reorderLevels(ids).then(() => this.load());
  },

  // ---- 新增 / 编辑 ----
  openAdd() {
    this.setData({ editVisible: true, f: { id: '', name: '', zhe: '', desc: '' }, zheHint: '' });
  },
  openEdit(e) {
    const lv = this.data.levels.find(l => l.id === e.currentTarget.dataset.id);
    if (!lv) return;
    const zhe = String(lv.discount / 10);
    this.setData({
      editVisible: true,
      f: { id: lv.id, name: lv.name, zhe, desc: lv.desc || '' },
      zheHint: discountLabel(lv.discount),
    });
  },
  closeEdit() { this.setData({ editVisible: false }); },
  onInput(e) {
    const k = e.currentTarget.dataset.k;
    let v = e.detail.value;
    if (k === 'zhe') v = v.replace(/[^0-9.]/g, '');
    const patch = { ['f.' + k]: v };
    if (k === 'zhe') {
      const n = Number(v);
      patch.zheHint = (n > 0 && n <= 10) ? discountLabel(n * 10) : '需在 1–10 之间';
    }
    this.setData(patch);
  },
  saveEdit() {
    const f = this.data.f;
    const n = Number(f.zhe);
    if (!(n > 0 && n <= 10)) return this.toast('折扣需在 1–10 之间', 'warn');
    api.saveLevel({ id: f.id, name: f.name, desc: f.desc, discount: Math.round(n * 10) })
      .then(() => { this.setData({ editVisible: false }); this.load(); this.toast('已保存'); })
      .catch(err => this.toast(err.message, 'warn'));
  },

  // ---- 删除 + 迁移 ----
  askDelete(e) {
    const id = e.currentTarget.dataset.id;
    const lv = this.data.levels.find(l => l.id === id);
    if (!lv) return;
    if (this.data.levels.length <= 1) return this.toast('至少保留一个等级', 'warn');
    api.levelImpact(id).then(impact => {
      const opts = this.data.levels.filter(l => l.id !== id).map(l => ({ id: l.id, name: l.name }));
      opts.push({ id: 'none', name: '降为非会员' });
      this.setData({ delVisible: true, delLv: lv, impact, migrateOpts: opts, migrateTo: opts[0].id });
    });
  },
  closeDelete() { this.setData({ delVisible: false }); },
  pickMigrate(e) { this.setData({ migrateTo: e.currentTarget.dataset.id }); },
  confirmDelete() {
    api.deleteLevel(this.data.delLv.id, this.data.migrateTo)
      .then(r => {
        this.setData({ delVisible: false });
        this.load();
        this.toast(r.disabledCoupons ? `已删除 · ${r.disabledCoupons} 张券被停用` : '已删除');
      })
      .catch(err => this.toast(err.message, 'warn'));
  },
});
