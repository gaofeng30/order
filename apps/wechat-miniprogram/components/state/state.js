Component({
  properties: {
    kind: { type: String, value: 'loading' }, // loading / error / empty
    text: { type: String, value: '' },
    action: { type: String, value: '' },      // 按钮文案；error 默认「重试」，empty 不传则无按钮
    icon: { type: String, value: '' },
    compact: { type: Boolean, value: false }, // 卡片内使用时减小上下留白
  },
  methods: {
    onAction() { this.triggerEvent('action'); },
  },
});
