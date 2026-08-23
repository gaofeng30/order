const api = require('./apiClient.js');
const catalogApi = require('./catalogApi.js');

const DECIMAL_ID = /^[1-9]\d*$/;
const MEAL_PERIODS = new Set(['all', 'lunch', 'dinner']);

function invalidCatalog() { return new api.APIError('CATALOG_UNAVAILABLE'); }
function requiredString(value) {
  if (typeof value !== 'string') throw invalidCatalog();
  return value;
}
function canonicalID(value) {
  if (typeof value !== 'string' || !DECIMAL_ID.test(value)) throw invalidCatalog();
  return value;
}
function cents(value, positive) {
  if (!Number.isSafeInteger(value) || value < (positive ? 1 : 0)) throw invalidCatalog();
  return value;
}
function parseImages(value) {
  if (!Array.isArray(value) || value.length > 3) throw invalidCatalog();
  const seen = new Set();
  return value.map(image => {
    if (!image || typeof image !== 'object' || Array.isArray(image)) throw invalidCatalog();
    const objectKey = requiredString(image.object_key);
    if (!objectKey || seen.has(objectKey)) throw invalidCatalog();
    seen.add(objectKey);
    let url;
    try { url = api.resolvePublicURL(image.url, objectKey); } catch (error) { throw invalidCatalog(); }
    return { objectKey, url };
  });
}

function parseFrozenProduct(value) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw invalidCatalog();
  const original = cents(value.original_unit_price_cents, true);
  const hasStaffPrice = Object.hasOwn(value, 'staff_unit_price_cents');
  const staff = hasStaffPrice ? cents(value.staff_unit_price_cents, false) : undefined;
  if (hasStaffPrice && staff > original) throw invalidCatalog();
  if (!MEAL_PERIODS.has(value.meal_period) || typeof value.listed !== 'boolean'
    || typeof value.sold_out !== 'boolean') throw invalidCatalog();
  const images = parseImages(value.images);
  return {
    id: canonicalID(value.id), category_id: canonicalID(value.category_id),
    name: requiredString(value.name), description: requiredString(value.description),
    specification: requiredString(value.specification), meal_period: value.meal_period,
    images, cover: images.length ? images[0] : null,
    listed: value.listed, sold_out: value.sold_out,
    original_unit_price_cents: original, staff_unit_price_cents: staff,
    isStaffPrice: hasStaffPrice, price_cents: hasStaffPrice ? staff : original,
  };
}

function snapshotProduct(value) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw invalidCatalog();
  const original = cents(value.original_unit_price_cents, true);
  const hasStaffPrice = value.isStaffPrice === true;
  const staff = hasStaffPrice ? cents(value.staff_unit_price_cents, false) : undefined;
  const effective = cents(value.price_cents, false);
  if ((hasStaffPrice && (staff > original || effective !== staff))
    || (!hasStaffPrice && effective !== original)) throw invalidCatalog();
  if (!MEAL_PERIODS.has(value.meal_period) || !Array.isArray(value.images) || value.images.length > 3
    || typeof value.listed !== 'boolean' || typeof value.sold_out !== 'boolean') throw invalidCatalog();
  const seen = new Set();
  const images = value.images.map(image => {
    if (!image || typeof image.objectKey !== 'string' || !image.objectKey
      || seen.has(image.objectKey) || typeof image.url !== 'string' || !image.url) throw invalidCatalog();
    seen.add(image.objectKey);
    return { objectKey: image.objectKey, url: image.url };
  });
  return {
    id: canonicalID(value.id), category_id: canonicalID(value.category_id),
    name: requiredString(value.name), description: requiredString(value.description),
    specification: requiredString(value.specification),
    meal_period: value.meal_period,
    images, cover: images.length ? images[0] : null,
    listed: value.listed, sold_out: value.sold_out,
    original_unit_price_cents: original, staff_unit_price_cents: staff,
    isStaffPrice: hasStaffPrice, price_cents: effective,
  };
}

function formatCents(value) {
  const valueCents = cents(value, false);
  return `${Math.floor(valueCents / 100)}.${String(valueCents % 100).padStart(2, '0')}`;
}
function withPrice(product) {
  const snapshot = snapshotProduct(product);
  return Object.assign({}, snapshot, {
    price_text: formatCents(snapshot.price_cents),
    original_price_text: formatCents(snapshot.original_unit_price_cents),
    staff_price_text: snapshot.isStaffPrice ? formatCents(snapshot.staff_unit_price_cents) : '',
  });
}
function parseCatalog(value) {
  if (!value || typeof value !== 'object' || !Array.isArray(value.categories)) throw invalidCatalog();
  return { categories: value.categories.map(category => {
    if (!category || typeof category !== 'object' || !Array.isArray(category.products)) throw invalidCatalog();
    const id = canonicalID(category.id);
    const products = category.products.map(product => {
      const parsed = parseFrozenProduct(product);
      if (parsed.category_id !== id) throw invalidCatalog();
      return withPrice(parsed);
    });
    return { id, name: requiredString(category.name), products };
  }) };
}
function parseProduct(value) {
  if (!value || typeof value !== 'object' || !Object.hasOwn(value, 'product')) throw invalidCatalog();
  return withPrice(parseFrozenProduct(value.product));
}
function normalizeFailure(error) {
  if (error && (error.code === 'PRODUCT_NOT_FOUND' || error.code === 'CATALOG_UNAVAILABLE')) return error;
  return invalidCatalog();
}
async function loadCatalog() {
  try { return parseCatalog(await catalogApi.listCatalog()); } catch (error) { throw normalizeFailure(error); }
}
async function loadProduct(id, selection) {
  try { return parseProduct(await catalogApi.getProduct(id, selection)); } catch (error) { throw normalizeFailure(error); }
}
function flattenProducts(categories) { return categories.reduce((all, category) => all.concat(category.products), []); }

module.exports = {
  flattenProducts, formatCents, loadCatalog, loadProduct,
  parseFrozenProduct, snapshotProduct, withPrice,
};
