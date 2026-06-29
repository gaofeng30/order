const data = require('../../utils/data.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { cats: [], showAdd: false, newName: '' },
  onLoad() {
    this.setData({ cats: JSON.parse(JSON.stringify(data.ADMIN_CATS)) });
  },
  toggle(e) {
    const id = e.currentTarget.dataset.id;
    const cats = this.data.cats.map(c => c.id === id ? { ...c, on: !c.on } : c);
    this.setData({ cats });
  },
  newCat() { this.setData({ showAdd: true, newName: '' }); },
  closeAdd() { this.setData({ showAdd: false }); },
  onAddInput(e) { this.setData({ newName: e.detail.value }); },
  confirmAdd() {
    const name = (this.data.newName || '').trim();
    if (!name) { this.selectComponent('#toast').show('请输入分类名称', { icon: 'warn' }); return; }
    if (this.data.cats.some(c => c.name === name)) { this.selectComponent('#toast').show('该分类已存在', { icon: 'warn' }); return; }
    const cats = this.data.cats.slice();
    const sort = cats.length ? cats[cats.length - 1].sort + 1 : 1;
    cats.push({ id: 'c' + Date.now(), name, sort, on: true, count: 0 });
    this.setData({ cats, showAdd: false });
    this.selectComponent('#toast').show('已新增「' + name + '」');
  },
});
