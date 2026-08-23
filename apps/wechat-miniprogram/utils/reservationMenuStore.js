const api = require('./apiClient.js');
const catalogStore = require('./catalogStore.js');

function invalid(code) { return new api.APIError(code || 'MENU_UNAVAILABLE'); }
function id(value) {
  if (typeof value !== 'string' || !/^[1-9]\d*$/.test(value)) throw invalid();
  return value;
}
function text(value) {
  if (typeof value !== 'string') throw invalid();
  return value;
}
function exactBoolean(value) {
  if (typeof value !== 'boolean') throw invalid();
  return value;
}

function parsePickupOptions(body) {
  if (!body || !Array.isArray(body.dates)) {
    throw invalid('PICKUP_OPTIONS_UNAVAILABLE');
  }
  const dates = body.dates.map(day => {
    if (!day || !/^\d{4}-\d{2}-\d{2}$/.test(day.date) || typeof day.available !== 'boolean'
      || !Array.isArray(day.meal_periods)) throw invalid('PICKUP_OPTIONS_UNAVAILABLE');
    const mealPeriods = day.meal_periods.map(meal => {
      if (!meal || !['lunch', 'dinner'].includes(meal.meal_period) || typeof meal.available !== 'boolean'
        || !Array.isArray(meal.pickup_times) || !/^\d{2}:\d{2}$/.test(meal.cutoff_time)) {
        throw invalid('PICKUP_OPTIONS_UNAVAILABLE');
      }
      const pickupTimes = meal.pickup_times.map(time => {
        if (typeof time !== 'string' || !/^\d{2}:\d{2}$/.test(time)) throw invalid('PICKUP_OPTIONS_UNAVAILABLE');
        return time;
      });
      return { mealPeriod: meal.meal_period, available: meal.available, cutoffTime: meal.cutoff_time, pickupTimes };
    });
    return { date: day.date, available: day.available, mealPeriods };
  });
  return { dates };
}

function firstAvailable(options) {
  for (const day of options.dates) {
    if (!day.available) continue;
    for (const meal of day.mealPeriods) {
      if (meal.available && meal.pickupTimes.length) {
        return { date: day.date, mealPeriod: meal.mealPeriod, time: meal.pickupTimes[0] };
      }
    }
  }
  return null;
}

function parseStoreStatus(value) {
  if (!value || typeof value !== 'object' || Array.isArray(value)
    || !['open', 'closed', 'cutoff'].includes(value.business_status)) throw invalid();
  return {
    businessStatus: value.business_status,
    serviceDateAvailable: exactBoolean(value.service_date_available),
    mealAvailable: exactBoolean(value.meal_available),
    cutoffPassed: exactBoolean(value.cutoff_passed),
  };
}

function parseMenu(body) {
  if (!body || !body.selection || !body.store_status || !Array.isArray(body.categories)) throw invalid();
  const selection = body.selection;
  if (!/^\d{4}-\d{2}-\d{2}$/.test(selection.date) || !/^\d{2}:\d{2}$/.test(selection.time)
    || !['lunch', 'dinner'].includes(selection.meal_period)) throw invalid();
  const storeStatus = parseStoreStatus(body.store_status);
  const currentOrderable = storeStatus.businessStatus === 'open' && storeStatus.serviceDateAvailable
    && storeStatus.mealAvailable && !storeStatus.cutoffPassed;
  const categories = body.categories.map(category => {
    const categoryID = id(category.id);
    if (!Array.isArray(category.products)) throw invalid();
    return {
      id: categoryID,
      name: text(category.name),
      products: category.products.map(product => {
        let base;
        try { base = catalogStore.withPrice(catalogStore.parseFrozenProduct(product)); } catch (error) { throw invalid(); }
        if (base.category_id !== categoryID || !base.listed
          || (base.meal_period !== 'all' && base.meal_period !== selection.meal_period)) throw invalid();
        const orderable = currentOrderable && !base.sold_out;
        let availabilityLabel = '当前不可下单';
        if (base.sold_out) availabilityLabel = '已售罄';
        else if (orderable) availabilityLabel = '可选择';
        return Object.assign(base, { soldOut: base.sold_out, orderable, availabilityLabel });
      }),
    };
  });
  return {
    selection: { date: selection.date, mealPeriod: selection.meal_period, time: selection.time },
    storeStatus, orderable: currentOrderable, categories,
  };
}

async function loadPickupOptions() { return parsePickupOptions(await api.getOptional('/api/v1/menu/pickup-options')); }
async function loadMenu(selection) {
  if (!selection || !/^\d{4}-\d{2}-\d{2}$/.test(selection.date) || !/^\d{2}:\d{2}$/.test(selection.time)) {
    throw invalid('INVALID_MENU_SELECTION');
  }
  return parseMenu(await api.getOptional(`/api/v1/menu?date=${selection.date}&time=${encodeURIComponent(selection.time)}`));
}

module.exports = { firstAvailable, loadMenu, loadPickupOptions, parseMenu, parsePickupOptions };
