const data = require('../../utils/data.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { cats: [] },
  onLoad() {
    this.setData({ cats: JSON.parse(JSON.stringify(data.ADMIN_CATS)) });
  },
  toggle(e) {
    const id = e.currentTarget.dataset.id;
    const cats = this.data.cats.map(c => c.id === id ? { ...c, on: !c.on } : c);
    this.setData({ cats });
  },
  newCat() { this.selectComponent('#toast').show('新增分类 · 建设中', { icon: 'plus' }); },
});
