// 开屏装饰图层 — 用户端展示组件（身份选择页）
// pageLifetimes.show 在宿主页面每次显示时重读配置：商户端改完设置返回即生效，页面 js 零改动
const layer = require('../../utils/layer.js');

Component({
  data: { cfg: { enabled: false, src: '' } },
  // 首次显示与每次返回都会触发 show，无需再挂 attached（避免进页双读 storage）
  pageLifetimes: { show() { this.setData({ cfg: layer.get() }); } },
});
