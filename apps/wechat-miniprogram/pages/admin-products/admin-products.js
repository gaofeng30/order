const api = require('../../utils/apiClient.js');
const merchantStore = require('../../utils/merchantStore.js');
const menuStore = require('../../utils/reservationMenuStore.js');
const dateLabel = require('../../utils/dateLabel.js');

function monthDay(date) {
  const parts = date.split('-').map(Number);
  return `${parts[1]}月${parts[2]}日`;
}

/* 固定只操作今天（A20）：取餐时间表里的「今天」= 北京时间今天；
   每个餐段按其第一个取餐时间读菜单。菜单接口在截单、休息中时仍返回当天菜品与售罄状态，
   因此今天截单后页面照常显示；某个餐段读取失败时，其余餐段照常显示。 */
async function loadToday() {
  const today = dateLabel.shanghaiToday();
  const options = await menuStore.loadPickupOptions();
  const day = options.dates.find(item => item.date === today);
  const meals = day ? day.mealPeriods.filter(meal => meal.pickupTimes.length) : [];
  const settled = await Promise.all(meals.map(meal => menuStore
    .loadMenu({ date: today, mealPeriod: meal.mealPeriod, time: meal.pickupTimes[0] })
    .then(menu => ({ menu }), error => ({ error }))));
  const menus = settled.filter(item => item.menu).map(item => item.menu);
  if (meals.length && !menus.length) throw settled[0].error;
  // 按菜单顺序合并分类；全天供应的菜在午餐、晚餐都会出现，只保留一次（A23）
  const categories = [];
  const byCategory = {};
  const products = {};
  for (const menu of menus) {
    for (const category of menu.categories) {
      if (!byCategory[category.id]) {
        byCategory[category.id] = { id: category.id, name: category.name, ids: [] };
        categories.push(byCategory[category.id]);
      }
      for (const product of category.products) {
        if (products[product.id]) continue;
        products[product.id] = {
          id: product.id, name: product.name, cover: product.cover,
          // 商户端显示标价（原价），不显示本人的员工价（D3）
          price: product.original_price_text, soldOut: product.soldOut, saving: false,
        };
        byCategory[category.id].ids.push(product.id);
      }
    }
  }
  return { date: today, categories, products, partial: menus.length < meals.length };
}

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    listState: 'loading', groups: [], search: '',
    title: `今天（${monthDay(dateLabel.shanghaiToday())}）售罄设置 · 次日自动恢复`,
  },
  onShow() {
    this.load(this._shown ? 'silent' : 'reset');
    this._shown = true;
  },
  onUnload() { this._generation = (this._generation || 0) + 1; },
  async load(mode) {
    // 有开关正在保存时不做后台刷新，避免覆盖未完成的结果
    if (mode === 'silent' && this._products && Object.values(this._products).some(item => item.saving)) return false;
    const generation = (this._generation || 0) + 1;
    this._generation = generation;
    if (mode === 'reset') this.setData({ listState: 'loading', groups: [] });
    try {
      const result = await loadToday();
      if (generation !== this._generation) return false;
      this._date = result.date;
      this._categories = result.categories;
      this._products = result.products;
      this.setData({ title: `今天（${monthDay(result.date)}）售罄设置 · 次日自动恢复` });
      this.render();
      if (result.partial && mode !== 'silent') {
        this.selectComponent('#toast').show('部分餐段的菜品没加载出来，请稍后重试', { icon: 'warn' });
      }
      return true;
    } catch (error) {
      if (generation !== this._generation) return false;
      if (mode !== 'silent' || this.data.listState === 'loading') this.setData({ listState: 'error', groups: [] });
      return false;
    }
  },
  retry() { return this.load('reset'); },
  // 搜索时无匹配的分类整组隐藏（A23）
  render() {
    const key = this.data.search.toLocaleLowerCase();
    const groups = [];
    for (const category of this._categories || []) {
      const items = category.ids
        .map(id => this._products[id])
        .filter(product => !key || product.name.toLocaleLowerCase().includes(key));
      if (items.length) groups.push({ id: category.id, name: category.name, items });
    }
    const total = Object.keys(this._products || {}).length;
    let listState = 'ready';
    if (!total) listState = 'empty';
    else if (!groups.length) listState = 'nomatch';
    this.setData({ listState, groups });
  },
  onSearch(e) {
    const search = (e.detail.value || '').trim();
    if (search === this.data.search) return;
    this.setData({ search });
    this.render();
  },
  clearSearch() {
    this.setData({ search: '' });
    this.render();
  },
  // 更新单个开关：只 setData 这一行（minimal setData）
  patch(id) {
    const product = this._products[id];
    const patch = {};
    this.data.groups.forEach((group, groupIndex) => group.items.forEach((item, itemIndex) => {
      if (item.id !== id) return;
      patch[`groups[${groupIndex}].items[${itemIndex}].soldOut`] = product.soldOut;
      patch[`groups[${groupIndex}].items[${itemIndex}].saving`] = product.saving;
    }));
    if (Object.keys(patch).length) this.setData(patch);
  },
  // 售罄开关：不确认；请求中置灰；成功 toast；失败复位（A22）
  async toggle(e) {
    const id = e.currentTarget.dataset.id;
    const product = this._products && this._products[id];
    if (!product || product.saving || !this._date) return false;
    // 作废进行中的静默刷新，避免刷新结果替换掉正在保存的这一行
    this._generation = (this._generation || 0) + 1;
    const previous = product.soldOut;
    const next = !previous;
    product.soldOut = next;
    product.saving = true;
    this.patch(id);
    const toast = this.selectComponent('#toast');
    try {
      const result = await merchantStore.setSoldOut(id, this._date, next, api.newIdempotencyKey('soldout'));
      product.soldOut = result.sold_out;
      product.saving = false;
      this.patch(id);
      toast.show(product.soldOut ? `${product.name} 已售罄` : `${product.name} 已恢复可售`);
      return true;
    } catch (error) {
      product.soldOut = previous;
      product.saving = false;
      this.patch(id);
      toast.show(error && error.code === 'FORBIDDEN' ? '没有操作权限' : '保存失败，请重试', { icon: 'warn' });
      return false;
    }
  },
});
