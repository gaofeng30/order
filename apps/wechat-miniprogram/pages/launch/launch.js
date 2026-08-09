const { nav } = require('../../utils/util.js');

// 微信绿色双气泡图标（与设计稿一致，inline SVG → image）
const WX_SVG =
  '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="#07c160">' +
  '<path d="M9.5 4C5.36 4 2 6.86 2 10.4c0 1.98 1.06 3.74 2.72 4.92L4 18l2.74-1.4c.86.24 1.78.36 2.76.36.25 0 .5-.01.74-.03A6.1 6.1 0 0 1 9.9 15c0-3.2 3.06-5.7 6.85-5.7.3 0 .59.02.88.05C16.95 6.2 13.6 4 9.5 4z"/>' +
  '<path d="M22 15.1c0-2.9-2.82-5.1-6.1-5.1-3.4 0-6.1 2.3-6.1 5.1 0 2.83 2.7 5.1 6.1 5.1.7 0 1.38-.1 2-.27L20 21l-.5-1.86c1.5-.94 2.5-2.4 2.5-4.04z"/></svg>';

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    auth: false,
    wxIcon: 'data:image/svg+xml,' + encodeURIComponent(WX_SVG),
  },
  noop() {},
  toBrand() { nav.toBrand(); },
  go(e) {
    const to = e.currentTarget.dataset.to;
    nav.go(to);
  },
  openAuth() { this.setData({ auth: true }); },
  closeAuth() { this.setData({ auth: false }); },
  allowAuth() {
    this.setData({ auth: false });
    nav.go('home');
  },
});
