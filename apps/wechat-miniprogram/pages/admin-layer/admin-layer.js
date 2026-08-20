// 开屏图层 — 商户端编辑页
// 上传透明 PNG → 在等比缩放的页面预览上拖拽定位、滑杆调大小 → 保存后展示在
// 身份选择页（见 utils/layer.js）
const layer = require('../../utils/layer.js');

const D = layer.DEFAULTS; // 默认位置/大小与用户端渲染共用一份定义
const PREVIEW_SCALE = 0.6;

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    mock: 'launch',      // 预览背景：身份选择页
    src: '',             // 预览图片路径（临时或已持久化）
    enabled: false,
    sizePct: 35,         // 大小滑杆值 = size * 100（屏宽百分比）
    // 预览画布几何（onLoad 计算，px）
    screenW: 375, screenH: 667, scale: PREVIEW_SCALE, previewW: 0, previewH: 0,
    // movable-view 几何（px）
    imgW: 0, imgH: 0, mx: 0, my: 0,
  },

  onLoad() {
    let win;
    try { win = wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync(); }
    catch (e) { win = wx.getSystemInfoSync(); }
    // 用 windowWidth/Height：真实图层铺满 .page（100vh = 窗口高），预览坐标系必须与之一致
    //（Android 虚拟导航栏机型 screenHeight > windowHeight，用 screen 会产生纵向偏移）
    const screenW = win.windowWidth || win.screenWidth || 375;
    const screenH = win.windowHeight || win.screenHeight || 667;
    const previewW = Math.round(screenW * PREVIEW_SCALE);
    const previewH = Math.round(screenH * PREVIEW_SCALE);

    const cfg = layer.get();
    // 非渲染态：中心比例坐标 / 高宽比 / 已落盘路径 / 未保存的临时路径 / 拖拽实时位置 / 待移除标记
    this._cx = cfg.cx; this._cy = cfg.cy; this._ar = cfg.ar;
    this._savedSrc = cfg.src;
    this._tempPath = '';
    this._x = 0; this._y = 0;
    this._pendingRemove = false;

    this.setData({
      screenW, screenH, previewW, previewH,
      src: cfg.src, enabled: cfg.enabled, sizePct: Math.round(cfg.size * 100),
      ...this.geometry(cfg.size, previewW, previewH),
    });
  },

  // 由 size 与中心比例 (_cx/_cy/_ar) 计算 movable-view 几何，并钳制在画布内
  geometry(size, previewW, previewH) {
    const imgW = size * previewW;
    const imgH = imgW * this._ar;
    let mx = this._cx * previewW - imgW / 2;
    let my = this._cy * previewH - imgH / 2;
    mx = Math.max(0, Math.min(mx, previewW - imgW));
    my = Math.max(0, Math.min(my, previewH - imgH));
    this._x = mx; this._y = my;
    return { imgW, imgH, mx, my };
  },

  // 把拖拽后的 px 位置折算回中心比例
  foldDrag() {
    const { imgW, imgH, previewW, previewH } = this.data;
    if (!previewW || !imgW) return;
    this._cx = (this._x + imgW / 2) / previewW;
    this._cy = (this._y + imgH / 2) / previewH;
  },


  // 拖拽期间 movable-view 自驱动，只记录位置不 setData（避免回环）；
  // 只认触摸来源——程序 setData 与阻尼动画也会触发 bindchange，若混入会把滞后的动画位置折算进坐标
  onMove(e) {
    const s = e.detail.source;
    if (s !== 'touch' && s !== 'touch-out-of-bounds') return;
    this._x = e.detail.x; this._y = e.detail.y;
  },
  // 拖完把 px 位置折算回比例，并同步 data.mx/my（保持属性值与视图实际位置一致，
  // 否则后续 geometry 钳制结果若恰好等于旧属性值，movable-view 不会回位）
  onDragEnd() {
    this.foldDrag();
    this.setData({ mx: this._x, my: this._y });
  },

  choose() {
    wx.chooseMedia({
      count: 1, mediaType: ['image'], sourceType: ['album'], sizeType: ['original'],
      success: (res) => {
        const path = res.tempFiles[0].tempFilePath;
        wx.getImageInfo({
          src: path,
          success: (info) => {
            if ((info.type || '').toLowerCase() !== 'png') {
              this.toast('请选择透明背景的 PNG 图片', 'warn');
              return;
            }
            if (!this.data.src) { this._cx = D.cx; this._cy = D.cy; }
            this._tempPath = path;
            this._pendingRemove = false; // 选了新图即取消待移除状态
            this._ar = info.height / info.width;
            this.setData({
              src: path, enabled: true,
              ...this.geometry(this.data.sizePct / 100, this.data.previewW, this.data.previewH),
            });
          },
          fail: () => this.toast('图片读取失败', 'warn'),
        });
      },
    });
  },

  // 滑杆缩放：先折算当前拖拽位置，再围绕中心重算几何
  // （movable-view 只改宽度不会重新钳制位置，须连同钳制后的 x/y 一起 setData）
  onSize(e) {
    this.foldDrag();
    const sizePct = e.detail.value;
    this.setData({
      sizePct,
      ...this.geometry(sizePct / 100, this.data.previewW, this.data.previewH),
    });
  },

  onEnable(e) {
    const v = e.detail.value;
    if (v && !this.data.src) {
      this.toast('请先上传图片', 'warn');
      // data.enabled 本就是 false，同值 setData 不会把拨过去的 switch 拨回来，
      // 先写 true 再在下一帧写回 false 强制重渲染
      this.setData({ enabled: true });
      wx.nextTick(() => this.setData({ enabled: false }));
      return;
    }
    this.setData({ enabled: v });
  },

  // 移除只是暂存状态，与本页其余编辑一致：点「保存」才真正删除，「取消」可反悔
  remove() {
    wx.showModal({
      title: '移除图层',
      content: '保存后将删除图片与位置设置',
      confirmColor: '#0b2f63',
      success: (r) => {
        if (!r.confirm) return;
        this._pendingRemove = true;
        this._tempPath = '';
        this._cx = D.cx; this._cy = D.cy; this._ar = 1;
        this.setData({
          src: '', enabled: false, sizePct: Math.round(D.size * 100),
          ...this.geometry(D.size, this.data.previewW, this.data.previewH),
        });
        this.toast('已移除，保存后生效');
      },
    });
  },

  save() {
    this.foldDrag();
    // 已确认移除且未选新图：删除文件与配置后返回
    if (this._pendingRemove && !this._tempPath) {
      layer.clear();
      this._pendingRemove = false; this._savedSrc = '';
      this.toast('图层已保存');
      setTimeout(() => wx.navigateBack({ fail: () => wx.reLaunch({ url: '/pages/admin-profile/admin-profile' }) }), 600);
      return;
    }
    let src = this._savedSrc;
    if (this._tempPath) {
      try { src = layer.persistImage(this._tempPath, this._savedSrc); }
      catch (e) { this.toast('图片保存失败，请重试', 'warn'); return; }
      this._savedSrc = src; this._tempPath = '';
    }
    layer.save({
      enabled: this.data.enabled,
      src,
      cx: this._cx, cy: this._cy,
      size: this.data.sizePct / 100,
      ar: this._ar,
    });
    this.toast('图层已保存');
    setTimeout(() => wx.navigateBack({ fail: () => wx.reLaunch({ url: '/pages/admin-profile/admin-profile' }) }), 600);
  },

  cancel() { wx.navigateBack(); },
  noop() {},
  toast(msg, icon) { this.selectComponent('#toast').show(msg, { icon: icon || 'check' }); },
});
