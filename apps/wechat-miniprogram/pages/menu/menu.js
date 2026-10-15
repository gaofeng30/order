const menuStore = require('../../utils/reservationMenuStore.js');
const identityStore = require('../../utils/identityStore.js');
const storefrontStore = require('../../utils/storefrontStore.js');
const { dayLabel } = require('../../utils/dateLabel.js');
const { nav, cart, pickup } = require('../../utils/util.js');

const MEAL_LABELS = { lunch: '午餐', dinner: '晚餐' };

// 最近一次成功加载的菜单。经 tabbar 切回时页面实例是新建的，先用它直接显示列表，再后台静默刷新（U14）
let menuCache = null;

function selectionKey(selected) {
  return selected ? `${selected.date} ${selected.mealPeriod} ${selected.time}` : '';
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
    listState: 'loading', optionState: 'loading', cats: [], groups: [], qtyMap: {}, count: 0, unavailableCount: 0,
    total: '0.00', active: '', intoView: '', catIntoView: '', sheet: false, cartItems: [],
    czVisible: false, czItem: null, czInit: null, czLabel: '加入购物车',
    pickup: null, pickerVisible: false, pickerDates: [], pickerGroups: [], pickerDate: '',
    search: '', keyword: '', searchOpen: false, searchFocus: false, flavors: [], pickupPoint: '', menuReason: '',
    canCheckout: false, returnToConfirm: false, restoreTop: 0,
  },
  onLoad(opts) {
    const pages = getCurrentPages();
    const previous = pages[pages.length - 2];
    this._returnToConfirm = !!(opts && opts.returnToConfirm === '1' && previous && previous.route === 'pages/confirm/confirm');
    this.setData({ returnToConfirm: this._returnToConfirm });
    const selected = pickup.get();
    if (menuCache && selected && menuCache.key === selectionKey(selected)) this.hydrate(selected);
    this.refresh();
  },
  async onShow() {
    this.refresh();
    this.setData({ flavors: (getApp().globalData.storefrontFlavors || []).slice() });
    this.loadStorefront();
    // 只有首次进入显示加载；之后保留列表与滚动位置，后台静默刷新（U14）
    await this.loadOptionsAndMenu(this._menuLoaded === true);
  },
  onUnload() {
    this._optionsRequestID = (this._optionsRequestID || 0) + 1;
    this._menuRequestID = (this._menuRequestID || 0) + 1;
    clearTimeout(this._jumpTimer);
  },
  hydrate(selected) {
    this._options = menuCache.options;
    this._allGroups = menuCache.groups;
    this._menuOrderable = menuCache.orderable;
    this._menuReason = menuCache.reason;
    this._availability = menuCache.availability;
    this._productsById = menuCache.productsById;
    this._menuLoaded = true;
    this.setPicker(this._options, selected.date);
    this.setData({ optionState: menuCache.optionState, pickup: Object.assign({}, selected, { label: pickup.periodLabel() }) });
    this.applySearch('', true);
    // 经 tabbar 切回菜单时页面实例是新建的，按缓存恢复滚动位置（U14）
    if (menuCache.scrollTop) this.setData({ restoreTop: menuCache.scrollTop });
  },
  // 门店行显示营业设置里的取餐地点；未加载到之前不显示（U6）
  loadStorefront() {
    return storefrontStore.load().then(settings => {
      if (settings.pickupPoint !== this.data.pickupPoint) this.setData({ pickupPoint: settings.pickupPoint });
    }).catch(() => {});
  },
  async loadOptionsAndMenu(silent) {
    const requestID = (this._optionsRequestID || 0) + 1;
    this._optionsRequestID = requestID;
    if (!silent) {
      this._menuOrderable = false;
      this.setData({ optionState: 'loading', listState: 'loading', restoreTop: 0, groups: [], cats: [], canCheckout: false });
    }
    try {
      const options = await menuStore.loadOrderableOptions();
      if (requestID !== this._optionsRequestID) return false;
      this._options = options;
      const current = pickup.get();
      const currentDay = current && options.dates.find(day => day.date === current.date && day.available);
      const currentMeal = currentDay && currentDay.mealPeriods.find(meal => meal.mealPeriod === current.mealPeriod && meal.available);
      const available = currentMeal
        ? (currentMeal.pickupTimes.includes(current.time) ? current : Object.assign({}, current, {
          time: currentMeal.pickupTimes[0], timeConfirmed: false,
        })) : menuStore.firstAvailable(options);
      const selected = available;
      this.setPicker(options, selected && selected.date);
      if (!selected) {
        pickup.set(null);
        getApp().globalData.menuState = null;
        menuCache = null;
        this._menuOrderable = false;
        this._menuReason = '近期没有可下单的餐段';
        this._availability = {};
        this._allGroups = [];
        this._menuLoaded = false;
        this.setData({ optionState: 'empty', listState: 'empty', pickup: null, restoreTop: 0, groups: [], cats: [] });
        this.refresh();
        return false;
      }
      // 取餐时间被改到别的时段时列表整体不同，按首次加载显示
      const changed = selectionKey(selected) !== selectionKey(this.data.pickup);
      pickup.set(selected);
      this.setData({ optionState: 'ready', pickup: Object.assign({}, selected, { label: pickup.periodLabel() }) });
      return this.loadMenu(silent && !changed);
    } catch (error) {
      if (requestID !== this._optionsRequestID || silent) return false;
      pickup.set(null);
      menuCache = null;
      this._menuOrderable = false;
      this._options = null;
      this._allGroups = [];
      this.setData({ optionState: 'error', listState: 'error', pickup: null, restoreTop: 0, groups: [], cats: [] });
      return false;
    }
  },
  setPicker(options, selectedDate) {
    const dates = options.dates.map(day => ({
      date: day.date, label: dayLabel(day.date), available: day.available,
    }));
    const date = selectedDate || (dates[0] && dates[0].date) || '';
    const source = options.dates.find(day => day.date === date);
    const groups = source ? source.mealPeriods.map(meal => ({
      key: meal.mealPeriod, name: MEAL_LABELS[meal.mealPeriod], available: meal.available, date,
    })) : [];
    this.setData({ pickerDates: dates, pickerGroups: groups, pickerDate: date });
  },
  openPicker() { if (this._options) this.setData({ pickerVisible: true }); },
  closePicker() { this.setData({ pickerVisible: false }); },
  async pickPickerDate(e) {
    const date = e.currentTarget.dataset.date;
    const day = this._options && this._options.dates.find(item => item.date === date);
    if (!day || !day.available) return false;
    const current = pickup.get();
    const meal = day.mealPeriods.find(item => item.available && current && item.mealPeriod === current.mealPeriod)
      || day.mealPeriods.find(item => item.available);
    if (!meal) return false;
    this.setPicker(this._options, date);
    return this.selectMeal(date, meal.mealPeriod, false);
  },
  pickPickerMeal(e) {
    const { date, period } = e.currentTarget.dataset;
    return this.selectMeal(date, period, true);
  },
  async selectMeal(date, period, close) {
    const day = this._options && this._options.dates.find(item => item.date === date);
    const meal = day && day.mealPeriods.find(item => item.mealPeriod === period);
    if (!day || !day.available || !meal || !meal.available || !meal.pickupTimes.length) return false;
    this._optionsRequestID = (this._optionsRequestID || 0) + 1;
    const current = pickup.get();
    const selected = current && current.date === date && current.mealPeriod === period && meal.pickupTimes.includes(current.time)
      ? current : { date, mealPeriod: period, time: meal.pickupTimes[0], timeConfirmed: false };
    pickup.set(selected);
    this.setData({ pickerVisible: !close, pickup: Object.assign({}, selected, { label: pickup.periodLabel() }) });
    return this.loadMenu(false);
  },
  retryCatalog() { return this.loadOptionsAndMenu(false); },
  async loadMenu(silent) {
    const requestID = (this._menuRequestID || 0) + 1;
    this._menuRequestID = requestID;
    const selected = pickup.get();
    if (!selected) return false;
    if (!silent) {
      this._menuOrderable = false;
      this._menuReason = '';
      // 新时段的菜单回来之前不沿用旧时段的可售判断（E2）
      this._availability = null;
      this.setData({ listState: 'loading', restoreTop: 0, groups: [], cats: [], active: '', canCheckout: false, menuReason: '' });
    }
    try {
      const showStaffPrice = await canShowStaffPrice();
      if (requestID !== this._menuRequestID) return false;
      const menu = await menuStore.loadMenu(selected);
      if (requestID !== this._menuRequestID) return false;
      if (menu.selection.date !== selected.date || menu.selection.time !== selected.time
        || menu.selection.mealPeriod !== selected.mealPeriod) throw new Error('selection drift');
      this._allGroups = showStaffPrice ? menu.categories : menu.categories.map(group => Object.assign({}, group, {
        products: group.products.map(maskStaffPrice),
      }));
      this._menuOrderable = menu.orderable;
      this._menuReason = menuStore.unavailableReason(menu.storeStatus, selected.mealPeriod);
      this._productsById = {};
      this._availability = {};
      this._allGroups.forEach(group => group.products.forEach(product => {
        this._productsById[product.id] = product;
        this._availability[product.id] = { orderable: product.orderable, soldOut: product.soldOut };
      }));
      // 菜单状态共享给详情页、确认页（E2、D7）
      getApp().globalData.menuState = {
        date: selected.date, time: selected.time, mealPeriod: selected.mealPeriod,
        orderable: menu.orderable, reason: this._menuReason, availability: this._availability,
      };
      // 用最新价格刷新购物车，数量、口味、备注不变（E3）
      cart.reprice(this._productsById);
      this._menuLoaded = true;
      menuCache = {
        key: selectionKey(selected), options: this._options, optionState: this.data.optionState,
        groups: this._allGroups, orderable: this._menuOrderable, reason: this._menuReason,
        availability: this._availability, productsById: this._productsById,
        scrollTop: menuCache && menuCache.key === selectionKey(selected) ? menuCache.scrollTop : 0,
      };
      this.applySearch(this.data.search, silent);
      this.refresh();
      return true;
    } catch (error) {
      if (requestID !== this._menuRequestID || silent) return false;
      this._allGroups = [];
      this._productsById = {};
      this._availability = null;
      this._menuOrderable = false;
      this._menuLoaded = false;
      this.setData({ listState: 'error', restoreTop: 0, groups: [], cats: [], active: '', canCheckout: false });
      return false;
    }
  },
  onSearch(e) { this.applySearch(e.detail.value || ''); },
  toggleSearch() {
    const open = !this.data.searchOpen;
    this.setData({ searchOpen: open, searchFocus: open }, () => this.measure());
    if (!open) this.applySearch('');
  },
  finishSearch() { this.setData({ searchFocus: false }, () => this.measure()); },
  clearSearch() { this.applySearch(''); },
  // keepPosition：静默刷新时保留当前分类高亮
  applySearch(search, keepPosition) {
    const keyword = search.trim();
    const terms = keyword.toLocaleLowerCase().split(/\s+/).filter(Boolean);
    const all = this._allGroups || [];
    const groups = all.map(group => ({
      id: group.id, name: group.name,
      products: terms.length ? group.products.filter(product => {
        const text = `${product.name} ${product.description} ${product.specification}`.toLocaleLowerCase();
        return terms.every(term => text.includes(term));
      }) : group.products.slice(),
    })).filter(group => group.products.length);
    let listState = 'ready';
    if (!groups.length) listState = keyword && all.some(group => group.products.length) ? 'noresult' : 'empty';
    const keepActive = keepPosition && groups.some(group => group.id === this.data.active);
    this.setData({
      search, keyword, listState, groups,
      // 列表卸载（非 ready）后，重建时不再按旧位置滚动（U14）
      restoreTop: keepPosition && listState === 'ready' ? this.data.restoreTop : 0,
      intoView: keepPosition ? this.data.intoView : '',
      catIntoView: keepPosition ? this.data.catIntoView : '',
      cats: groups.map(group => ({ id: group.id, name: group.name })),
      active: keepActive ? this.data.active : (groups.length ? groups[0].id : ''),
    }, () => {
      if (!keepPosition) this.setData({ intoView: 'menu-top', catIntoView: 'cat-0' }, () => this.measure());
      else this.measure();
    });
  },
  toast(message, icon) { this.selectComponent('#toast').show(message, { icon: icon || 'check' }); },
  // 不可售的菜留在购物车但不计入件数与合计（E2、D1）
  refresh() {
    const qtyMap = {};
    cart.list().forEach(line => { qtyMap[line.id] = line.q; });
    const count = cart.count(undefined, null);
    const cartItems = cart.list(undefined, null);
    this.setData({
      qtyMap, count, unavailableCount: 0, total: cart.total(undefined, null), cartItems,
      canCheckout: count > 0,
      menuReason: this._menuReason || '',
    });
    return cartItems.length;
  },
  unorderableReason(product) {
    if (!product) return '该时段不供应';
    if (product.soldOut) return '该菜品已售罄';
    return this._menuReason || '暂时无法下单，请稍后重试';
  },
  async add(e) {
    const id = e.currentTarget.dataset.id;
    const selection = e.currentTarget.dataset.selection || pickup.get();
    if (cart.isLocked(selection)) return this.toast('该餐段有待确认支付，请先处理', 'warn');
    let product = this._productsById && this._productsById[id];
    if (e.currentTarget.dataset.selection) {
      try {
        const menu = await menuStore.loadMenu(selection);
        product = menu.categories.flatMap(group => group.products).find(item => item.id === id);
        if (!await canShowStaffPrice()) product = maskStaffPrice(product);
      } catch (error) { return this.toast('菜品状态暂时没读到，请重试', 'warn'); }
    }
    if (!product || !product.orderable) {
      this.toast(this.unorderableReason(product), 'warn');
      return false;
    }
    cart.add(product, selection);
    this.refresh();
    return true;
  },
  // 购物车减到 0 时弹层自动关闭（U16）
  sub(e) {
    const selection = e.currentTarget.dataset.selection || pickup.get();
    if (cart.isLocked(selection)) return this.toast('该餐段有待确认支付，请先处理', 'warn');
    cart.sub(e.currentTarget.dataset.id, selection);
    if (!this.refresh() && this.data.sheet) this.setData({ sheet: false });
  },
  clear() {
    wx.showModal({
      title: '清空购物车？',
      content: '已选的口味和备注也会清除，待确认支付的餐段会保留',
      confirmText: '清空',
      confirmColor: '#c4432f',
      success: (res) => {
        if (!res.confirm) return;
        cart.clear();
        this.refresh();
        this.setData({ sheet: false });
      },
    });
  },
  openCustomize(e) {
    this._czSelection = pickup.get();
    if (cart.isLocked(this._czSelection)) return this.toast('该餐段有待确认支付，请先处理', 'warn');
    const item = this._productsById && this._productsById[e.currentTarget.dataset.id];
    if (!item || !item.orderable) return this.toast(this.unorderableReason(item), 'warn');
    this.setData({ czVisible: true, czItem: item, czInit: null, czLabel: '加入购物车' });
  },
  editItem(e) {
    this._czSelection = e.currentTarget.dataset.selection || pickup.get();
    if (cart.isLocked(this._czSelection)) return this.toast('该餐段有待确认支付，请先处理', 'warn');
    const entry = cart.entry(e.currentTarget.dataset.id, this._czSelection);
    if (!entry) return;
    this.setData({ sheet: false, czVisible: true, czItem: entry.product,
      czInit: { qty: entry.qty, flavors: entry.flavors, note: entry.note }, czLabel: '保存' });
  },
  onCzClose() { this.setData({ czVisible: false }); },
  onCzConfirm(e) {
    if (Object.prototype.hasOwnProperty.call(this.data.czItem, 'orderable') && !this.data.czItem.orderable) return;
    cart.setPrefs(this.data.czItem, e.detail, this._czSelection);
    this.setData({ czVisible: false });
    this.refresh();
  },
  // 点击分类：滚动到对应分区；滚动动画期间不让滚动监听改写高亮
  jump(e) {
    const { id, idx } = e.currentTarget.dataset;
    this._jumpLock = true;
    this.releaseJumpLock(500);
    this.setData({ active: id, catIntoView: `cat-${Math.max(0, idx - 1)}`, intoView: '' },
      () => this.setData({ intoView: `sec-${idx}` }));
  },
  releaseJumpLock(delay) {
    clearTimeout(this._jumpTimer);
    this._jumpTimer = setTimeout(() => { this._jumpLock = false; }, delay);
  },
  // 分类标签随滚动自动高亮，并把高亮标签滚入可见区（U15）
  onScroll(e) {
    if (menuCache && !this.data.keyword) menuCache.scrollTop = e.detail.scrollTop;
    if (this._jumpLock) { this.releaseJumpLock(150); return; }
    const offsets = this._offsets || [];
    if (!offsets.length) return;
    const { scrollTop, scrollHeight } = e.detail;
    let idx = 0;
    if (this._viewH && scrollTop + this._viewH >= scrollHeight - 4) idx = offsets.length - 1;
    else offsets.forEach((top, i) => { if (top <= scrollTop + 8) idx = i; });
    const cat = this.data.cats[idx];
    if (!cat || cat.id === this.data.active) return;
    this.setData({ active: cat.id, catIntoView: `cat-${Math.max(0, idx - 1)}` });
  },
  // 记录各分区在滚动内容中的位置（px）
  measure() {
    this._offsets = [];
    if (this.data.listState !== 'ready') return;
    const query = wx.createSelectorQuery();
    query.select('#menuScroll').boundingClientRect();
    query.select('#menuScroll').scrollOffset();
    query.selectAll('.sec-anchor').boundingClientRect();
    query.exec((res) => {
      const box = res && res[0];
      const offset = res && res[1];
      const anchors = res && res[2];
      if (!box || !offset || !anchors) return;
      this._viewH = box.height;
      this._offsets = anchors.map(anchor => anchor.top - box.top + offset.scrollTop);
    });
  },
  goDetail(e) {
    const id = e && e.currentTarget && e.currentTarget.dataset && e.currentTarget.dataset.id;
    if (!this._productsById || !this._productsById[id]) return false;
    nav.go('detail', { id });
    return true;
  },
  openSheet() { if (this.data.cartItems.length) this.setData({ sheet: true }); },
  closeSheet() { this.setData({ sheet: false }); },
  returnToConfirmation() {
    if (!this._returnToConfirm) return false;
    wx.navigateBack({ delta: 1, fail: () => this.toast('暂时无法返回确认页，请稍后重试', 'warn') });
    return true;
  },
  checkoutBlockReason() {
    if (this.data.listState === 'loading') return '菜单加载中，请稍候';
    if (!this._menuOrderable) return this._menuReason || '暂时无法下单，请稍后重试';
    if (this.data.unavailableCount) return '请先移除不可售的菜';
    return '请先选择菜品';
  },
  goConfirm() {
    if (!this.data.canCheckout) {
      this.toast(this.checkoutBlockReason(), 'warn');
      return false;
    }
    this.setData({ sheet: false });
    const pages = getCurrentPages();
    const previous = pages[pages.length - 2];
    const groups = cart.groups();
    if (groups.length === 1 && this._returnToConfirm && previous
      && previous.route === 'pages/confirm/confirm' && previous._groupKey === groups[0].key) {
      return this.returnToConfirmation();
    }
    if (groups.length === 1) nav.go('confirm', { group: groups[0].key });
    else nav.go('cart-confirm');
    return true;
  },
  noop() {},
});
