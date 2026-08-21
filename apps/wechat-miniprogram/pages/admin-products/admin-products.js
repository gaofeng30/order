const data = require('../../utils/data.js');
const api = require('../../utils/api.js');
const { nav } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    cats: ['全部'].concat(data.CATS),
    cat: '全部',
    list: [],
  },
  onShow() { this.build(); },
  build() {
    /* 上下架与当日售罄是两个独立维度（§6.5）：status 只表达上下架，
       售罄按取餐日期查记录。展示标签从两者现算，不再存第三个综合状态。 */
    const day = data.BUSINESS_DAY;
    const list = data.menuList().filter(m => this.data.cat === '全部' || m.cat === this.data.cat).map(m => {
      const shelved = m.status === 'on';
      const soldOut = data.isSoldOut(m.id, day);
      return {
        id: m.id, name: m.name, price: m.price, cat: m.cat, img: m.img,
        imgCount: (m.imgs || []).length,
        soldOut,
        pillLabel: !shelved ? '已下架' : (soldOut ? '售罄' : '可购'),
        pillTone: shelved && !soldOut ? 'ok' : 'mute',
        soldoutLabel: soldOut ? '恢复售卖' : '标记售罄',
        on: shelved,
      };
    });
    this.setData({ list });
  },
  switchCat(e) { this.setData({ cat: e.currentTarget.dataset.c }, () => this.build()); },
  /* 售罄只写当前营业日的记录，不动 status。商户站在现场，他能判断的只有今天；
     提前关闭明天的供应属于排产，一期不做。 */
  toggleSoldout(e) {
    const id = e.currentTarget.dataset.id;
    const day = data.BUSINESS_DAY;
    const nx = !data.isSoldOut(id, day);
    api.setSoldOut(id, day, nx).then(() => {
      this.build();
      this.selectComponent('#toast').show(nx ? `已置售罄 · 仅限 ${day}` : '已恢复售卖', { icon: 'tag' });
    });
  },
});
