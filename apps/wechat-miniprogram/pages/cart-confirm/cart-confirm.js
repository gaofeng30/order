const menuStore = require('../../utils/reservationMenuStore.js');
const identityStore = require('../../utils/identityStore.js');
const { cart, nav } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { groups: [], loading: true, picker: null, czVisible: false, czItem: null, czInit: null, flavors: [] },
  onShow() {
    this.setData({ flavors: (getApp().globalData.storefrontFlavors || []).slice() });
    return this.loadGroups();
  },
  onUnload() { this._requestID = (this._requestID || 0) + 1; },
  async loadGroups() {
    const requestID = (this._requestID || 0) + 1;
    this._requestID = requestID;
    const groups = cart.groups().map(group => Object.assign({}, group, {
      state: 'loading', locked: cart.isLocked(group.selection), total: cart.total(undefined, group.selection),
      busy: !!(cart.payment(group.selection) && cart.payment(group.selection).busy),
      times: [], reason: '',
    }));
    this.setData({ groups, loading: false });
    if (!groups.length) return;
    let options;
    let staff = false;
    try {
      const results = await Promise.all([menuStore.loadPickupOptions(), identityStore.load().catch(() => null)]);
      options = results[0];
      const pricing = results[1] && results[1].pricing_identity;
      staff = !!(pricing && pricing.kind === 'STAFF' && Number.isSafeInteger(pricing.rate_percent)
        && pricing.rate_percent >= 1 && pricing.rate_percent <= 100);
    } catch (error) {
      if (requestID === this._requestID) this.setData({ groups: groups.map(group => Object.assign({}, group, { state: 'error' })) });
      return;
    }
    await Promise.all(groups.map(async (group, index) => {
      if (group.locked) {
        group.state = 'pending';
      } else {
        try {
          const day = options.dates.find(item => item.date === group.selection.date);
          const meal = day && day.mealPeriods.find(item => item.mealPeriod === group.selection.mealPeriod);
          group.times = day && day.available && meal && meal.available ? meal.pickupTimes : [];
          if (!group.times.length) {
            group.state = 'unavailable';
            group.reason = '该餐段已不可下单，请移除或返回菜单重新选购';
          } else {
            const selection = Object.assign({}, group.selection);
            if (!group.times.includes(selection.time)) {
              selection.time = group.times[0];
              selection.timeConfirmed = false;
            }
            const menu = await menuStore.loadMenu(selection);
            if (requestID !== this._requestID) return;
            const products = {};
            const availability = {};
            menu.categories.forEach(category => category.products.forEach(product => {
              products[product.id] = staff ? product : Object.assign({}, product, {
                isStaffPrice: false, price_cents: product.original_unit_price_cents,
              });
              availability[product.id] = { orderable: product.orderable, soldOut: product.soldOut };
            }));
            cart.setSelection(selection);
            cart.reprice(products, selection);
            group.selection = selection;
            group.items = cart.list(availability, selection);
            group.total = cart.total(availability, selection);
            group.state = menu.orderable && !group.items.some(item => item.unavailableLabel) ? 'ready' : 'unavailable';
            group.reason = menuStore.unavailableReason(menu.storeStatus, selection.mealPeriod) || '请移除本组不可售的菜品';
          }
        } catch (error) { group.state = 'error'; }
      }
      if (requestID === this._requestID) this.setData({ [`groups[${index}]`]: group });
    }));
  },
  group(event) { return this.data.groups.find(group => group.key === event.currentTarget.dataset.group); },
  openPicker(event) {
    const group = this.group(event);
    if (!group || group.state !== 'ready' || group.locked) return;
    this.setData({ picker: group });
  },
  closePicker() { this.setData({ picker: null }); },
  selectTime(event) {
    const group = this.data.picker;
    const time = event.currentTarget.dataset.time;
    if (!group || !group.times.includes(time) || cart.isLocked(group.selection)) return;
    const selection = Object.assign({}, group.selection, { time, timeConfirmed: true });
    cart.setSelection(selection);
    const index = this.data.groups.findIndex(item => item.key === group.key);
    this.setData({ [`groups[${index}].selection`]: selection, picker: null });
  },
  checkout(event) {
    const group = this.group(event);
    if (!group) return;
    if (cart.payment(group.selection) && cart.payment(group.selection).busy) return this.loadGroups();
    if (group.locked) return nav.go('confirm', { group: group.key });
    if (group.state !== 'ready') return;
    if (!group.selection.timeConfirmed) return this.openPicker(event);
    nav.go('confirm', { group: group.key });
  },
  removeItem(event) {
    const group = this.group(event);
    if (!group || group.locked) return;
    cart.remove(event.currentTarget.dataset.id, group.selection);
    return this.loadGroups();
  },
  editItem(event) {
    const group = this.group(event);
    if (!group || group.locked) return;
    const entry = cart.entry(event.currentTarget.dataset.id, group.selection);
    if (!entry) return;
    this._editingSelection = group.selection;
    this.setData({ czVisible: true, czItem: entry.product,
      czInit: { qty: entry.qty, flavors: entry.flavors, note: entry.note } });
  },
  onCzClose() { this.setData({ czVisible: false }); },
  onCzConfirm(event) {
    cart.setPrefs(this.data.czItem, event.detail, this._editingSelection);
    this.setData({ czVisible: false });
    return this.loadGroups();
  },
  goMenu() { nav.tabTo('menu'); },
});
