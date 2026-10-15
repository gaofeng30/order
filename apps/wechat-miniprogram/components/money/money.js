const { yuan } = require('../../utils/money.js');

Component({
  properties: {
    // 整数分。金额一律以整数分保存与计算（§5.6），组件内部完成转元。
    cents: { type: null, value: null },
    // 已经是文本或元的场合（菜品价、目录 price_text）仍走 v
    v: { type: null, value: null },
    staff: { type: Boolean, value: false }, // 员工价：红色（E3）
    size: { type: Number, value: 32 },  // rpx
    cur: { type: Boolean, value: true },
    color: { type: String, value: '' },  // 覆盖颜色（如白色）
  },
  observers: {
    // 没有金额时整体不渲染，不显示「¥0」或「¥—」（D4）
    'cents, v': function (cents, v) {
      let text = '';
      if (cents !== null && cents !== undefined && cents !== '') {
        const value = yuan(cents);
        text = value === '—' ? '' : value;
      } else if (v !== null && v !== undefined && v !== '') {
        text = String(v);
      }
      this.setData({ text });
    },
  },
  data: { text: '' },
});
