const makeQRCode = require('../../utils/vendor/qrcode-generator.js');
makeQRCode.stringToBytes = makeQRCode.stringToBytesFuncs['UTF-8'];

Component({
  properties: {
    value: { type: String, value: '' },
    size: { type: Number, value: 300 },        // rpx
    dark: { type: String, value: '#0b2f63' },
  },
  data: {
    src: '',   // 离屏 canvas 导出的图片，真正用于展示
  },
  observers: {
    'value,size,dark': function () { this.draw(); },
  },
  lifetimes: {
    attached() {
      // 等节点渲染后再绘制
      setTimeout(() => this.draw(), 50);
    },
    detached() { this._qrGeneration = (this._qrGeneration || 0) + 1; },
  },
  methods: {
    draw(retry = 0) {
      const generation = this._qrGeneration = (this._qrGeneration || 0) + 1;
      const value = this.data.value;
      this.setData({ src: '' });
      if (typeof value !== 'string' || !value) return;
      let code;
      try {
        code = makeQRCode(0, 'M');
        code.addData(value, 'Byte');
        code.make();
      } catch (error) { return; }
      const q = this.createSelectorQuery();
      q.select('#qrc').fields({ node: true, size: true }).exec((res) => {
        if (generation !== this._qrGeneration) return;
        if (!res || !res[0] || !res[0].node) {
          // 节点尚未就绪，重试若干次，避免静默画不出来
          if (retry < 5) setTimeout(() => {
            if (generation === this._qrGeneration) this.draw(retry + 1);
          }, 60);
          return;
        }
        const canvas = res[0].node;
        const ctx = canvas.getContext('2d');
        const g = getApp().globalData;
        const dpr = (wx.getWindowInfo ? wx.getWindowInfo().pixelRatio : 2) || 2;
        const px = Math.round(this.data.size * (g.screenWidth || 375) / 750 * dpr);
        const n = code.getModuleCount();
        const cell = Math.floor(px / (n + 8));
        if (cell < 1) return;
        const offset = Math.floor((px - cell * (n + 8)) / 2) + 4 * cell;
        canvas.width = px;
        canvas.height = px;
        ctx.clearRect(0, 0, px, px);
        ctx.fillStyle = '#ffffff';
        ctx.fillRect(0, 0, px, px);

        ctx.fillStyle = this.data.dark;
        for (let y = 0; y < n; y++) {
          for (let x = 0; x < n; x++) {
            if (code.isDark(y, x)) ctx.fillRect(offset + x * cell, offset + y * cell, cell, cell);
          }
        }

        // 导出成图片，用普通 <image> 展示，避免原生 canvas 不随滚动 / 穿透层级
        wx.canvasToTempFilePath({
          canvas,
          x: 0, y: 0,
          width: canvas.width,
          height: canvas.height,
          destWidth: canvas.width,
          destHeight: canvas.height,
          success: (r) => {
            if (generation === this._qrGeneration) this.setData({ src: r.tempFilePath });
          },
        }, this);
      });
    },
  },
});
