const api = require('../../utils/apiClient.js');
const merchantStore = require('../../utils/merchantStore.js');
const menuStore = require('../../utils/reservationMenuStore.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { listState: 'loading', list: [], search: '' },
  onShow() { return this.load(); },
  async load() {
    this.setData({ listState: 'loading', list: [] });
    try {
      const options = await menuStore.loadPickupOptions();
      const day = options.dates[0];
      const meal = day && day.mealPeriods.find(item => item.available && item.pickupTimes.length);
      if (!day || !meal) throw new Error('no merchant product selection');
      this._serviceDate = day.date;
      const menu = await menuStore.loadMenu({ date: day.date, mealPeriod: meal.mealPeriod, time: meal.pickupTimes[0] });
      this._all = menu.categories.flatMap(category => category.products.map(product => Object.assign({}, product, {
        cat: category.name, price: product.price_text, soldoutLabel: product.soldOut ? '恢复售卖' : '标记售罄',
        pillLabel: product.soldOut ? '售罄' : '可售', pillTone: product.soldOut ? 'mute' : 'ok',
      })));
      this.applySearch();
      return true;
    } catch (error) { this._all = []; this.setData({ listState: 'error', list: [] }); return false; }
  },
  onSearch(e) { this.setData({ search: (e.detail.value || '').trim() }); this.applySearch(); },
  applySearch() {
    const key = this.data.search.toLocaleLowerCase();
    const list = (this._all || []).filter(product => !key || product.name.toLocaleLowerCase().includes(key));
    this.setData({ listState: list.length ? 'ready' : 'empty', list });
  },
  async toggleSoldout(e) {
    const id = e.currentTarget.dataset.id;
    const product = this._all.find(item => item.id === id);
    if (!product || !this._serviceDate) return false;
    try {
      const result = await merchantStore.setSoldOut(id, this._serviceDate, !product.soldOut, api.newIdempotencyKey('soldout'));
      product.soldOut = result.sold_out;
      product.soldoutLabel = product.soldOut ? '恢复售卖' : '标记售罄';
      product.pillLabel = product.soldOut ? '售罄' : '可售';
      product.pillTone = product.soldOut ? 'mute' : 'ok';
      this.applySearch();
      return true;
    } catch (error) { return false; }
  },
});
