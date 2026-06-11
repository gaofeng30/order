Component({
  properties: {
    v: { type: null, value: 0 },
    blue: { type: Boolean, value: false },
    size: { type: Number, value: 32 },  // rpx
    cur: { type: Boolean, value: true },
    color: { type: String, value: '' },  // 覆盖颜色（如白色）
  },
});
