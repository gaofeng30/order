Component({
  properties: {
    size: { type: Number, value: 300 },        // rpx
    dark: { type: String, value: '#0b2f63' },
  },
  data: {
    src: '',   // 离屏 canvas 导出的图片，真正用于展示
  },
  observers: {
    'size,dark': function () { this.draw(); },
  },
  lifetimes: {
    attached() {
      // 等节点渲染后再绘制
      setTimeout(() => this.draw(), 50);
    },
  },
  methods: {
    draw(retry = 0) {
      const q = this.createSelectorQuery();
      q.select('#qrc').fields({ node: true, size: true }).exec((res) => {
        if (!res || !res[0] || !res[0].node) {
          // 节点尚未就绪，重试若干次，避免静默画不出来
          if (retry < 5) setTimeout(() => this.draw(retry + 1), 60);
          return;
        }
        const canvas = res[0].node;
        const ctx = canvas.getContext('2d');
        const g = getApp().globalData;
        const dpr = (wx.getWindowInfo ? wx.getWindowInfo().pixelRatio : 2) || 2;
        const px = this.data.size * (g.screenWidth || 375) / 750; // rpx -> px
        canvas.width = px * dpr;
        canvas.height = px * dpr;
        ctx.scale(dpr, dpr);
        ctx.clearRect(0, 0, px, px);
        ctx.fillStyle = '#ffffff';
        ctx.fillRect(0, 0, px, px);

        const n = 21, cell = px / n;
        let seed = 7;
        const rnd = () => { seed = (seed * 9301 + 49297) % 233280; return seed / 233280; };
        ctx.fillStyle = this.data.dark;
        for (let y = 0; y < n; y++) {
          for (let x = 0; x < n; x++) {
            const finder = (x < 7 && y < 7) || (x > 13 && y < 7) || (x < 7 && y > 13);
            if (finder) continue;
            if (rnd() > 0.52) ctx.fillRect(x * cell, y * cell, cell + 0.6, cell + 0.6);
          }
        }
        // 三个定位角
        [[0, 0], [14, 0], [0, 14]].forEach(([fx, fy]) => {
          ctx.fillStyle = this.data.dark;
          ctx.fillRect(fx * cell, fy * cell, 7 * cell, 7 * cell);
          ctx.fillStyle = '#ffffff';
          ctx.fillRect((fx + 1) * cell, (fy + 1) * cell, 5 * cell, 5 * cell);
          ctx.fillStyle = this.data.dark;
          ctx.fillRect((fx + 2) * cell, (fy + 2) * cell, 3 * cell, 3 * cell);
        });

        // 导出成图片，用普通 <image> 展示，避免原生 canvas 不随滚动 / 穿透层级
        wx.canvasToTempFilePath({
          canvas,
          x: 0, y: 0,
          width: canvas.width,
          height: canvas.height,
          destWidth: canvas.width,
          destHeight: canvas.height,
          success: (r) => this.setData({ src: r.tempFilePath }),
        }, this);
      });
    },
  },
});
