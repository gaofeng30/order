const HUES = [['#5b8f3f', '#3d6b2f'], ['#b79a55', '#8c7038'], ['#3a5f8f', '#173b6d']];

/* 菜品图：有封面显示封面，没有或加载失败时显示按品类配色的首字占位（D12、D15） */
Component({
  properties: {
    item: { type: Object, value: null },
    h: { type: Number, value: 184 },     // rpx
    w: { type: Number, value: 0 },        // rpx, 0 = 自适应/铺满
    radius: { type: Number, value: 28 },  // rpx
    fontsize: { type: Number, value: 0 }, // rpx, 0 = 按高度推算
    tag: { type: Boolean, value: true },
    char: { type: String, value: '' },
    lazy: { type: Boolean, value: false },
  },
  data: { bg: '', ch: '食', fs: 80, src: '', showImg: false },
  observers: {
    'item,h,fontsize,char': function () { this.calc(); },
  },
  lifetimes: { attached() { this.calc(); } },
  methods: {
    calc() {
      const it = this.data.item || {};
      const src = it.img || (it.cover && it.cover.url) || '';
      const index = Array.from(String(it.name || '')).reduce((sum, char) => sum + char.codePointAt(0), 0) % HUES.length;
      const hue = HUES[index];
      const bg = `linear-gradient(150deg, ${hue[0]}, ${hue[1]})`;
      const ch = this.data.char || (it.name ? Array.from(it.name)[0] : '食');
      const fs = this.data.fontsize || Math.round(this.data.h * 0.42);
      this.setData({ src, showImg: !!src, bg, ch, fs });
    },
    onError() { this.setData({ showImg: false }); },
  },
});
