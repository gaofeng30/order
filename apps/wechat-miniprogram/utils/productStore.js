const api = require('./apiClient.js');
const catalogStore = require('./catalogStore.js');

function parse(body) {
  const product = body && body.product;
  if (!product || typeof product !== 'object') throw new api.APIError('CATALOG_UNAVAILABLE');
  const priceCents = product.original_unit_price_cents;
  const normalized = Object.assign({}, product, { price_cents: priceCents });
  const base = catalogStore.withPrice(normalized);
  if (!Array.isArray(product.images) || typeof product.listed !== 'boolean' || typeof product.sold_out !== 'boolean'
    || !['lunch', 'dinner'].includes(product.meal_period)) throw new api.APIError('CATALOG_UNAVAILABLE');
  const images = product.images.map(image => {
    if (!image || typeof image.object_key !== 'string' || !image.object_key || typeof image.url !== 'string'
      || !image.url.startsWith('https://')) throw new api.APIError('CATALOG_UNAVAILABLE');
    return { objectKey: image.object_key, url: image.url };
  });
  if (images.length > 3) throw new api.APIError('CATALOG_UNAVAILABLE');
  return Object.assign(base, {
    images,
    mealPeriod: product.meal_period,
    listed: product.listed,
    soldOut: product.sold_out,
    orderable: product.listed && !product.sold_out,
    staffUnitPriceCents: product.staff_unit_price_cents,
  });
}

async function load(id, selection) {
  if (!selection || !/^\d{4}-\d{2}-\d{2}$/.test(selection.date) || !/^\d{2}:\d{2}$/.test(selection.time)) {
    throw new api.APIError('SELECTION_REQUIRED');
  }
  const body = await api.getOptional(`/api/v1/catalog/products/${id}?date=${selection.date}&time=${encodeURIComponent(selection.time)}`);
  return parse(body);
}

module.exports = { load, parse };
