const api = require('./apiClient.js');
const catalogStore = require('./catalogStore.js');

function unavailable() { return new api.APIError('CATALOG_UNAVAILABLE'); }

function parse(body) {
  const product = body && body.product;
  if (!product || typeof product !== 'object' || Array.isArray(product)) throw unavailable();
  try {
    const base = catalogStore.withPrice(catalogStore.parseFrozenProduct(product));
    return Object.assign(base, {
      mealPeriod: base.meal_period,
      soldOut: base.sold_out,
      orderable: base.listed && !base.sold_out,
      staffUnitPriceCents: base.staff_unit_price_cents,
    });
  } catch (error) {
    throw unavailable();
  }
}

async function load(id, selection) {
  if (!/^[1-9]\d*$/.test(String(id)) || !selection
    || !/^\d{4}-\d{2}-\d{2}$/.test(selection.date)
    || !/^\d{2}:\d{2}$/.test(selection.time)
    || !['lunch', 'dinner'].includes(selection.mealPeriod)) {
    throw new api.APIError('SELECTION_REQUIRED');
  }
  const body = await api.getOptional(`/api/v1/catalog/products/${encodeURIComponent(String(id))}?date=${selection.date}&time=${encodeURIComponent(selection.time)}`);
  const product = parse(body);
  if (product.mealPeriod !== 'all' && product.mealPeriod !== selection.mealPeriod) throw unavailable();
  return product;
}

module.exports = { load, parse };
