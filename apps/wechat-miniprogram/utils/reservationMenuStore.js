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

function parsePickupOptions(body) {
  if (!body || body.timezone !== 'Asia/Shanghai' || !Array.isArray(body.dates)) throw invalid('PICKUP_OPTIONS_UNAVAILABLE');
  const dates = body.dates.map(day => {
    if (!day || !/^\d{4}-\d{2}-\d{2}$/.test(day.date) || typeof day.orderable !== 'boolean' || !Array.isArray(day.meals)) throw invalid('PICKUP_OPTIONS_UNAVAILABLE');
    const meals = day.meals.map(meal => {
      if (!meal || !['lunch', 'dinner'].includes(meal.code) || typeof meal.orderable !== 'boolean'
        || !Array.isArray(meal.pickup_times) || typeof meal.cutoff_at !== 'string') throw invalid('PICKUP_OPTIONS_UNAVAILABLE');
      const times = meal.pickup_times.map(time => {
        if (typeof time !== 'string' || !/^\d{2}:\d{2}$/.test(time)) throw invalid('PICKUP_OPTIONS_UNAVAILABLE');
        return time;
      });
      return { code: meal.code, cutoffAt: meal.cutoff_at, orderable: meal.orderable, times };
    });
    return { date: day.date, orderable: day.orderable, meals };
  });
  return { timezone: body.timezone, dates };
}

function firstAvailable(options) {
  for (const day of options.dates) {
    if (!day.orderable) continue;
    for (const meal of day.meals) {
      if (meal.orderable && meal.times.length) {
        return { date: day.date, mealPeriod: meal.code, time: meal.times[0] };
      }
    }
  }
  return null;
}

function parseMenu(body) {
  if (!body || !body.selection || !body.meal || !Array.isArray(body.categories)) throw invalid();
  const selection = body.selection;
  if (!/^\d{4}-\d{2}-\d{2}$/.test(selection.date) || !/^\d{2}:\d{2}$/.test(selection.time)
    || selection.timezone !== 'Asia/Shanghai' || !['lunch', 'dinner'].includes(body.meal.code)
    || typeof body.meal.orderable !== 'boolean') throw invalid();
  const categories = body.categories.map(category => {
    const categoryID = id(category.id);
    if (!Array.isArray(category.products)) throw invalid();
    return {
      id: categoryID,
      name: text(category.name),
      products: category.products.map(product => {
        const base = catalogStore.withPrice(product);
        if (base.category_id !== categoryID || typeof product.sold_out !== 'boolean' || typeof product.orderable !== 'boolean') throw invalid();
        return Object.assign(base, {
          soldOut: product.sold_out,
          orderable: body.meal.orderable && product.orderable && !product.sold_out,
          availabilityLabel: product.sold_out ? '已售罄' : (body.meal.orderable && product.orderable ? '可选择' : '当前不可下单'),
        });
      }),
    };
  });
  return {
    selection: { date: selection.date, mealPeriod: body.meal.code, time: selection.time },
    mealOrderable: body.meal.orderable,
    categories,
  };
}

async function loadPickupOptions() {
  return parsePickupOptions(await api.getOptional('/api/v1/menu/pickup-options'));
}

async function loadMenu(selection) {
  if (!selection || !/^\d{4}-\d{2}-\d{2}$/.test(selection.date) || !/^\d{2}:\d{2}$/.test(selection.time)) throw invalid('INVALID_MENU_SELECTION');
  return parseMenu(await api.getOptional(`/api/v1/menu?date=${selection.date}&time=${encodeURIComponent(selection.time)}`));
}

module.exports = { firstAvailable, loadMenu, loadPickupOptions, parseMenu, parsePickupOptions };
