const { yuan } = require('../../utils/money.js');

Component({
  properties: {
    // 整数分。金额一律以整数分保存与计算（§5.6），组件内部完成转元。
    cents: { type: null, value: null },
    // 已经是文本或元的场合（菜品价、目录 price_text）仍走 v
    v: { type: null, value: 0 },
    blue: { type: Boolean, value: false },
    size: { type: Number, value: 32 },  // rpx
    cur: { type: Boolean, value: true },
    color: { type: String, value: '' },  // 覆盖颜色（如白色）
  },
  observers: {
    'cents, v': function (cents, v) {
      this.setData({ text: cents === null || cents === undefined ? v : yuan(cents) });
    },
  },
  data: { text: '' },
});
