const menuStore = require('../../utils/reservationMenuStore.js');
const identityStore = require('../../utils/identityStore.js');
const { nav, cart, pickup } = require('../../utils/util.js');

const MEAL_LABELS = { lunch: '午餐', dinner: '晚餐' };

function firstBrowsable(options) {
  for (const day of options.dates) {
    for (const meal of day.mealPeriods) {
      if (meal.pickupTimes.length) {
        return { date: day.date, mealPeriod: meal.mealPeriod, time: meal.pickupTimes[0] };
      }
    }
  }
  return null;
}

async function canShowStaffPrice() {
  const session = getApp().globalData.session;
  if (!session || session.state !== 'ready' || !session.accessToken) return false;
  try {
    const identity = await identityStore.load();
    const pricing = identity && identity.pricing_identity;
    return pricing && pricing.kind === 'STAFF' && Number.isSafeInteger(pricing.rate_percent)
      && pricing.rate_percent >= 1 && pricing.rate_percent <= 100;
  } catch (error) {
    return false;
  }
}

function maskStaffPrice(product) {
  if (!product || !product.isStaffPrice) return product;
  return Object.assign({}, product, {
    staff_unit_price_cents: undefined,
    isStaffPrice: false,
    price_cents: product.original_unit_price_cents,
    price_text: product.original_price_text,
    staff_price_text: '',
  });
}

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    listState: 'loading', optionState: 'loading', cats: [], groups: [], qtyMap: {}, count: 0,
    total: 0, active: '', intoView: '', sheet: false, cartItems: [], _offsets: [],
    czVisible: false, czItem: null, czInit: null, czLabel: '加入购物车',
    pickup: null, pickerVisible: false, pickerDates: [], pickerGroups: [], pickerDate: '', search: '', flavors: [],
    canCheckout: false,
  },
  onLoad() { this.refresh(); },
  async onShow() {
    this.refresh();
    this.setData({ flavors: (getApp().globalData.storefrontFlavors || []).slice() });
    await this.loadOptionsAndMenu();
  },
  async loadOptionsAndMenu() {
    this._menuOrderable = false;
    this.setData({ optionState: 'loading', listState: 'loading', groups: [], cats: [], canCheckout: false });
    try {
      const options = await menuStore.loadPickupOptions();
      this._options = options;
      const current = pickup.get();
      const validCurrent = current && options.dates.some(day => day.date === current.date && day.available
        && day.mealPeriods.some(meal => meal.mealPeriod === current.mealPeriod
          && meal.available && meal.pickupTimes.includes(current.time)));
      const available = validCurrent ? current : menuStore.firstAvailable(options);
      const selected = available || firstBrowsable(options);
      this.setPicker(options, selected && selected.date);
      if (!selected) {
        pickup.set(null);
        this.setData({ optionState: 'empty', listState: 'empty', pickup: null });
        return false;
      }
      pickup.set(selected);
      this.setData({ optionState: available ? 'ready' : 'browse', pickup: Object.assign({}, selected, { label: pickup.label() }) });
      return this.loadMenu();
    } catch (error) {
      pickup.set(null);
      this._options = null;
      this._allGroups = [];
      this.setData({ optionState: 'error', listState: 'error', pickup: null, groups: [], cats: [] });
      return false;
    }
  },
  setPicker(options, selectedDate) {
    const dates = options.dates.map(day => ({
      date: day.date, label: day.date.slice(5), available: day.available,
    }));
    const date = selectedDate || (dates[0] && dates[0].date) || '';
    const source = options.dates.find(day => day.date === date);
    const groups = source ? source.mealPeriods.map(meal => ({
      key: meal.mealPeriod, name: MEAL_LABELS[meal.mealPeriod], cutOff: !meal.available,
      cutoffLabel: !meal.available ? `已截单 · ${meal.cutoffTime} 截止` : '',
      times: meal.available ? meal.pickupTimes.slice() : [], date,
    })) : [];
    this.setData({ pickerDates: dates, pickerGroups: groups, pickerDate: date });
  },
  openPicker() { if (this._options) this.setData({ pickerVisible: true }); },
  closePicker() { this.setData({ pickerVisible: false }); },
  pickPickerDate(e) {
    const date = e.currentTarget.dataset.date;
    const day = this._options && this._options.dates.find(item => item.date === date);
    if (!day || !day.available) return false;
    this.setPicker(this._options, date);
    return true;
  },
  async pickPickerTime(e) {
    const { date, period, t } = e.currentTarget.dataset;
    const day = this._options && this._options.dates.find(item => item.date === date);
    const meal = day && day.mealPeriods.find(item => item.mealPeriod === period);
    if (!day || !day.available || !meal || !meal.available || !meal.pickupTimes.includes(t)) return false;
    const selected = { date, mealPeriod: period, time: t };
    pickup.set(selected);
    this.setData({ pickerVisible: false, pickup: Object.assign({}, selected, { label: pickup.label() }) });
    return this.loadMenu();
  },
  retryCatalog() { return this.loadOptionsAndMenu(); },
  async loadMenu() {
    const selected = pickup.get();
    if (!selected) return false;
    this._menuOrderable = false;
    this.setData({ listState: 'loading', groups: [], cats: [], active: '', canCheckout: false });
    try {
      const showStaffPrice = await canShowStaffPrice();
      const menu = await menuStore.loadMenu(selected);
      if (menu.selection.date !== selected.date || menu.selection.time !== selected.time
        || menu.selection.mealPeriod !== selected.mealPeriod) throw new Error('selection drift');
      this._allGroups = showStaffPrice ? menu.categories : menu.categories.map(group => Object.assign({}, group, {
        products: group.products.map(maskStaffPrice),
      }));
      this._menuOrderable = menu.orderable;
      this._productsById = {};
      this._allGroups.forEach(group => group.products.forEach(product => { this._productsById[product.id] = product; }));
      this.applySearch(this.data.search);
      this.refresh();
      return true;
    } catch (error) {
      this._allGroups = [];
      this._productsById = {};
      this._menuOrderable = false;
      this.setData({ listState: 'error', groups: [], cats: [], active: '', canCheckout: false });
      return false;
    }
  },
  onSearch(e) { this.applySearch((e.detail.value || '').trim()); },
  applySearch(search) {
    const lower = search.toLocaleLowerCase();
    const groups = (this._allGroups || []).map(group => ({
      id: group.id, name: group.name,
      products: lower ? group.products.filter(product =>
        `${product.name} ${product.description} ${product.specification}`.toLocaleLowerCase().includes(lower)) : group.products.slice(),
    })).filter(group => group.products.length);
    this.setData({
      search,
      listState: groups.length ? 'ready' : 'empty', groups,
      cats: groups.map(group => ({ id: group.id, name: group.name })),
      active: groups.length ? groups[0].id : '',
    }, () => this.measure());
  },
  toast(message, icon) { this.selectComponent('#toast').show(message, { icon: icon || 'check' }); },
  refresh() {
    const raw = getApp().globalData.cart;
    const qtyMap = {};
    Object.keys(raw).forEach(id => { qtyMap[id] = raw[id].qty; });
    const count = cart.count();
    this.setData({ qtyMap, count, total: cart.total(), cartItems: cart.list(), canCheckout: count > 0 && this._menuOrderable === true });
  },
  add(e) {
    const id = e.currentTarget.dataset.id;
    const product = this._productsById && this._productsById[id];
    if (!product || !product.orderable) return false;
    cart.add(product);
    this.refresh();
    return true;
  },
  sub(e) { cart.sub(e.currentTarget.dataset.id); this.refresh(); },
  clear() { cart.clear(); this.refresh(); this.setData({ sheet: false }); },
  openCustomize(e) {
    const item = this._productsById[e.currentTarget.dataset.id];
    if (!item || !item.orderable) return this.toast(item && item.soldOut ? '商品已售罄' : '当前不可下单', 'warn');
    this.setData({ czVisible: true, czItem: item, czInit: null, czLabel: '加入购物车' });
  },
  editItem(e) {
    const entry = cart.entry(e.currentTarget.dataset.id);
    if (!entry) return;
    this.setData({ sheet: false, czVisible: true, czItem: entry.product,
      czInit: { qty: entry.qty, flavors: entry.flavors, note: entry.note }, czLabel: '保存' });
  },
  onCzClose() { this.setData({ czVisible: false }); },
  onCzConfirm(e) { cart.setPrefs(this.data.czItem, e.detail); this.setData({ czVisible: false }); this.refresh(); },
  jump(e) { this.setData({ active: e.currentTarget.dataset.id, intoView: `sec-${e.currentTarget.dataset.idx}` }); },
  onScroll() {},
  onReady() {},
  measure() {},
  goDetail(e) {
    const id = e && e.currentTarget && e.currentTarget.dataset && e.currentTarget.dataset.id;
    if (!this._productsById || !this._productsById[id]) return false;
    nav.go('detail', { id });
    return true;
  },
  openSheet() { if (this.data.count) this.setData({ sheet: true }); },
  closeSheet() { this.setData({ sheet: false }); },
  goConfirm() { if (this.data.canCheckout && pickup.get()) { this.setData({ sheet: false }); nav.go('confirm'); } },
  noop() {},
});
