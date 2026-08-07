/* 优惠券新建 / 编辑 —— 二期能力，不在一期合同范围
   规则：券按等级自动生效（无领取动作）；一单一张；与等级折扣叠加；
        折扣券封顶必填；限定范围时门槛与减免均锚定可用商品小计。 */
const api = require('../../utils/api.js');
const data = require('../../utils/data.js');
const promo = require('../../utils/promo.js');

const md = d => (d || '').slice(5);
const num = v => (v || '').replace(/[^0-9.]/g, '');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    isEdit: false,
    cats: [],
    levels: [],
    groups: [],
    pickedNames: '',
    zheHint: '',
    pv: { face: '¥0', cond: '无门槛', name: '未命名优惠券', scope: '全场通用', period: '', tags: [] },
    f: {
      id: '', name: '', type: 'cut', amount: '', zhe: '', cap: '', threshold: '',
      levelIds: [], scope: 'all', catNames: [], itemIds: [],
      start: data.TODAY, end: data.TODAY, perLimit: '1', enabled: true,
    },
    pickerVisible: false,
    tmpIds: [],
  },

  onLoad(opts) {
    Promise.all([api.listLevels(), api.listMembers({}), api.listProducts()]).then(([levels, mb, products]) => {
      this.setData({
        _levels: levels,
        _products: products,
        cats: data.CATS.map(c => ({ name: c, on: false })),
        levels: levels.map(l => ({
          id: l.id, name: l.name, on: false,
          count: mb.list.filter(m => m.levelId === l.id).length,
        })),
      });
      if (!opts.id) return this.syncPreview();
      api.getCoupon(opts.id).then(c => {
        this.setData({
          isEdit: true,
          f: {
            id: c.id, name: c.name, type: c.type,
            amount: c.amount ? String(c.amount) : '',
            zhe: c.rate ? String(c.rate / 10) : '',
            cap: c.cap ? String(c.cap) : '',
            threshold: c.threshold ? String(c.threshold) : '',
            levelIds: c.levelIds.slice(), scope: c.scope,
            catNames: c.catNames.slice(), itemIds: c.itemIds.slice(),
            start: c.start, end: c.end, perLimit: String(c.perLimit), enabled: c.enabled,
          },
          levels: this.data.levels.map(l => Object.assign({}, l, { on: c.levelIds.indexOf(l.id) > -1 })),
          cats: this.data.cats.map(x => Object.assign({}, x, { on: c.catNames.indexOf(x.name) > -1 })),
          zheHint: c.rate ? promo.discountLabel(c.rate) : '',
        }, () => this.syncPreview());
      }).catch(() => this.toast('优惠券不存在', 'warn'));
    });
  },

  toast(msg, icon) { this.selectComponent('#toast').show(msg, { icon: icon || 'check' }); },

  // 表单 → 接口入参
  payload() {
    const f = this.data.f;
    return {
      id: f.id, name: f.name, type: f.type,
      amount: Number(f.amount) || 0,
      rate: f.type === 'discount' ? Math.round((Number(f.zhe) || 0) * 10) : 0,
      cap: Number(f.cap) || 0,
      threshold: Number(f.threshold) || 0,
      levelIds: f.levelIds, scope: f.scope, catNames: f.catNames, itemIds: f.itemIds,
      start: f.start, end: f.end, perLimit: Number(f.perLimit) || 0, enabled: f.enabled,
    };
  },

  syncPreview() {
    const c = this.payload();
    const lvs = this.data._levels || [];
    const names = (this.data.f.itemIds || []).map(id => {
      const m = (this.data._products || []).find(p => p.id === id);
      return m ? m.name : '';
    }).filter(Boolean);
    this.setData({
      pickedNames: names.join('、'),
      pv: {
        face: promo.faceText(c),
        cond: promo.condLabel(c),
        name: c.name || '未命名优惠券',
        scope: promo.scopeLabel(c),
        period: `${md(c.start)} 至 ${md(c.end)}`,
        tags: c.levelIds.map(id => { const l = lvs.find(x => x.id === id); return l ? l.name : ''; }).filter(Boolean),
      },
    });
  },

  onInput(e) {
    const k = e.currentTarget.dataset.k;
    let v = e.detail.value;
    if (k !== 'name') v = num(v);
    const patch = { ['f.' + k]: v };
    if (k === 'zhe') {
      const n = Number(v);
      patch.zheHint = (n > 0 && n < 10) ? promo.discountLabel(n * 10) : '需在 1–9.9 之间';
    }
    this.setData(patch, () => this.syncPreview());
  },
  pickType(e) { this.setData({ 'f.type': e.currentTarget.dataset.t }, () => this.syncPreview()); },
  onStart(e) { this.setData({ 'f.start': e.detail.value }, () => this.syncPreview()); },
  onEnd(e) { this.setData({ 'f.end': e.detail.value }, () => this.syncPreview()); },
  toggleEnabled() { this.setData({ 'f.enabled': !this.data.f.enabled }, () => this.syncPreview()); },

  toggleLevel(e) {
    const id = e.currentTarget.dataset.id;
    const ids = this.data.f.levelIds.slice();
    const i = ids.indexOf(id);
    if (i > -1) ids.splice(i, 1); else ids.push(id);
    this.setData({
      'f.levelIds': ids,
      levels: this.data.levels.map(l => Object.assign({}, l, { on: ids.indexOf(l.id) > -1 })),
    }, () => this.syncPreview());
  },

  pickScope(e) { this.setData({ 'f.scope': e.currentTarget.dataset.s }, () => this.syncPreview()); },
  toggleCat(e) {
    const name = e.currentTarget.dataset.c;
    const arr = this.data.f.catNames.slice();
    const i = arr.indexOf(name);
    if (i > -1) arr.splice(i, 1); else arr.push(name);
    this.setData({
      'f.catNames': arr,
      cats: this.data.cats.map(x => Object.assign({}, x, { on: arr.indexOf(x.name) > -1 })),
    }, () => this.syncPreview());
  },

  // ---- 菜品多选弹层 ----
  openPicker() {
    this.setData({ pickerVisible: true, tmpIds: this.data.f.itemIds.slice() }, () => this.buildGroups());
  },
  buildGroups() {
    const sel = this.data.tmpIds;
    const products = this.data._products || [];
    const groups = data.CATS.map(cat => {
      const items = products.filter(p => p.cat === cat)
        .map(p => Object.assign({}, p, { on: sel.indexOf(p.id) > -1 }));
      return { cat, items, allOn: items.length > 0 && items.every(i => i.on) };
    }).filter(g => g.items.length);
    this.setData({ groups });
  },
  toggleItem(e) {
    const id = e.currentTarget.dataset.id;
    const ids = this.data.tmpIds.slice();
    const i = ids.indexOf(id);
    if (i > -1) ids.splice(i, 1); else ids.push(id);
    this.setData({ tmpIds: ids }, () => this.buildGroups());
  },
  toggleGroup(e) {
    const cat = e.currentTarget.dataset.c;
    const g = this.data.groups.find(x => x.cat === cat);
    if (!g) return;
    let ids = this.data.tmpIds.slice();
    const catIds = g.items.map(i => i.id);
    if (g.allOn) ids = ids.filter(id => catIds.indexOf(id) < 0);
    else catIds.forEach(id => { if (ids.indexOf(id) < 0) ids.push(id); });
    this.setData({ tmpIds: ids }, () => this.buildGroups());
  },
  closePicker() { this.setData({ pickerVisible: false }); },
  confirmPicker() {
    this.setData({ pickerVisible: false, 'f.itemIds': this.data.tmpIds.slice() }, () => this.syncPreview());
  },

  cancel() { wx.navigateBack({ fail: () => wx.reLaunch({ url: '/pages/admin-coupons/admin-coupons' }) }); },
  save() {
    api.saveCoupon(this.payload())
      .then(() => { this.toast('已保存'); setTimeout(() => this.cancel(), 600); })
      .catch(err => this.toast(err.message, 'warn'));
  },
});
