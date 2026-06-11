const ICONS = require('../../utils/icons.js');

Component({
  properties: {
    name: { type: String, value: '' },
    size: { type: Number, value: 22 },        // px
    color: { type: String, value: 'currentColor' },
    stroke: { type: Number, value: 2 },
    fill: { type: String, value: 'none' },
  },
  data: { uri: '' },
  observers: {
    'name,color,stroke,fill': function () { this.build(); },
  },
  lifetimes: {
    attached() { this.build(); },
  },
  methods: {
    build() {
      const d = ICONS[this.data.name] || '';
      // currentColor 在 image 中无效，回退为深色
      const color = this.data.color === 'currentColor' ? '#22271d' : this.data.color;
      const svg =
        `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" ` +
        `fill="${this.data.fill}" stroke="${color}" stroke-width="${this.data.stroke}" ` +
        `stroke-linecap="round" stroke-linejoin="round"><path d="${d}"/></svg>`;
      const uri = 'data:image/svg+xml,' + encodeURIComponent(svg);
      this.setData({ uri });
    },
  },
});
