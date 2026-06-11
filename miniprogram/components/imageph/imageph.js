const { HUES } = require('../../utils/data.js');

Component({
  properties: {
    item: { type: Object, value: null },
    h: { type: Number, value: 184 },     // rpx
    w: { type: Number, value: 0 },        // rpx, 0 = 自适应/铺满
    radius: { type: Number, value: 28 },  // rpx
    fontsize: { type: Number, value: 0 }, // rpx, 0 = 按高度推算
    tag: { type: Boolean, value: true },
    char: { type: String, value: '' },
  },
  data: { bg: '', ch: '食', fs: 80, hasImg: false },
  observers: {
    'item,h,fontsize,char': function () { this.calc(); },
  },
  lifetimes: { attached() { this.calc(); } },
  methods: {
    calc() {
      const it = this.data.item || {};
      const hasImg = !!it.img;
      const hue = HUES[it.cat] || ['#5b8f3f', '#3d6b2f'];
      const bg = `linear-gradient(150deg, ${hue[0]}, ${hue[1]})`;
      const ch = this.data.char || (it.name ? it.name[0] : '食');
      const fs = this.data.fontsize || Math.round(this.data.h * 0.42);
      this.setData({ hasImg, bg, ch, fs });
    },
  },
});
