const { statusTone } = require('../../utils/util.js');

Component({
  options: { multipleSlots: false },
  properties: {
    s: { type: String, value: '' },           // 状态文案，自动映射语义色
    tone: { type: String, value: '' },          // 手动指定语义: ok/info/warn/mute
    dot: { type: Boolean, value: false },
    ondark: { type: Boolean, value: false },
    label: { type: String, value: '' },         // 覆盖显示文案
  },
  data: { cls: 'mute', text: '' },
  observers: {
    's,tone,ondark,label': function () { this.calc(); },
  },
  lifetimes: { attached() { this.calc(); } },
  methods: {
    calc() {
      const cls = this.data.ondark ? 'ondark' : (this.data.tone || statusTone(this.data.s));
      this.setData({ cls, text: this.data.label || this.data.s });
    },
  },
});
