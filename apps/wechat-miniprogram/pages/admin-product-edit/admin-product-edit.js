/* 菜品新增 / 编辑 —— 保存真正落库，支持多图上传
   图片经 api.uploadImage 上传后返回可访问地址；后端就位前返回的是本次运行期内的临时路径。 */
const data = require('../../utils/data.js');
const api = require('../../utils/api.js');

const MAX_IMGS = 3;

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    cats: data.CATS,
    maxImgs: MAX_IMGS,
    isEdit: false,
    f: { id: '', name: '', price: '', cat: data.CATS[0], desc: '', imgs: [] },
  },

  onLoad(opts) {
    if (!opts.id) return;
    api.getProduct(opts.id).then(m => {
      this.setData({
        isEdit: true,
        f: {
          id: m.id, name: m.name, price: String(m.price),
          cat: m.cat, desc: m.desc || '',
          imgs: (m.imgs && m.imgs.length) ? m.imgs.slice() : (m.img ? [m.img] : []),
        },
      });
    }).catch(() => this.toast('菜品不存在', 'warn'));
  },

  toast(msg, icon) { this.selectComponent('#toast').show(msg, { icon: icon || 'check' }); },

  onInput(e) {
    const k = e.currentTarget.dataset.k;
    let v = e.detail.value;
    if (k === 'price') v = v.replace(/[^0-9.]/g, '');
    this.setData({ ['f.' + k]: v });
  },
  pickCat(e) { this.setData({ 'f.cat': e.currentTarget.dataset.c }); },

  // ---- 图片 ----
  addImg() {
    const left = MAX_IMGS - this.data.f.imgs.length;
    if (left <= 0) return this.toast(`最多 ${MAX_IMGS} 张`, 'warn');
    wx.chooseMedia({
      count: left,
      mediaType: ['image'],
      sizeType: ['compressed'],
      success: r => {
        Promise.all(r.tempFiles.map(f => api.uploadImage(f.tempFilePath))).then(urls => {
          this.setData({ 'f.imgs': this.data.f.imgs.concat(urls).slice(0, MAX_IMGS) });
        });
      },
      fail: () => { /* 用户取消 */ },
    });
  },
  previewImg(e) {
    wx.previewImage({ current: this.data.f.imgs[+e.currentTarget.dataset.i], urls: this.data.f.imgs });
  },
  delImg(e) {
    const imgs = this.data.f.imgs.slice();
    imgs.splice(+e.currentTarget.dataset.i, 1);
    this.setData({ 'f.imgs': imgs });
  },
  setCover(e) {
    const i = +e.currentTarget.dataset.i;
    const imgs = this.data.f.imgs.slice();
    imgs.unshift(imgs.splice(i, 1)[0]);
    this.setData({ 'f.imgs': imgs });
    this.toast('已设为封面', 'eye');
  },

  cancel() { wx.navigateBack({ fail: () => wx.reLaunch({ url: '/pages/admin-products/admin-products' }) }); },

  save() {
    api.saveProduct(this.data.f)
      .then(() => { this.toast('已保存'); setTimeout(() => this.cancel(), 600); })
      .catch(err => this.toast(err.message, 'warn'));
  },

  askDelete() {
    wx.showModal({
      title: '删除菜品',
      content: `删除「${this.data.f.name}」后不可恢复。`,
      confirmText: '删除',
      confirmColor: '#b4483c',
      success: res => {
        if (!res.confirm) return;
        api.deleteProduct(this.data.f.id).then(r => {
          this.toast('已删除', 'box');
          setTimeout(() => this.cancel(), 700);
        }).catch(err => this.toast(err.message, 'warn'));
      },
    });
  },
});
