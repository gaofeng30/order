const { nav } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {},
  go(e) {
    const to = e.currentTarget.dataset.to;
    nav.go(to);
  },
});
