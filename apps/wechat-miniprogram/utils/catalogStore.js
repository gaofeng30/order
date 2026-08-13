const catalogApi = require('./catalogApi.js');

const DECIMAL_ID = /^[1-9]\d*$/;

function invalidCatalog() {
  return new catalogApi.CatalogError('CATALOG_UNAVAILABLE');
}

function requiredString(value) {
  if (typeof value !== 'string') throw invalidCatalog();
  return value;
}

function canonicalID(value) {
  if (typeof value !== 'string' || !DECIMAL_ID.test(value)) throw invalidCatalog();
  return value;
}

function snapshotProduct(value) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw invalidCatalog();
  if (!Number.isSafeInteger(value.price_cents) || value.price_cents < 0) throw invalidCatalog();
  return {
    id: canonicalID(value.id),
    category_id: canonicalID(value.category_id),
    name: requiredString(value.name),
    description: requiredString(value.description),
    specification: requiredString(value.specification),
    price_cents: value.price_cents,
  };
}

function formatCents(cents) {
  if (!Number.isSafeInteger(cents) || cents < 0) throw invalidCatalog();
  return `${Math.floor(cents / 100)}.${String(cents % 100).padStart(2, '0')}`;
}

function withPrice(product) {
  const snapshot = snapshotProduct(product);
  return Object.assign({}, snapshot, { price_text: formatCents(snapshot.price_cents) });
}

function parseCatalog(value) {
  if (!value || typeof value !== 'object' || !Array.isArray(value.categories)) throw invalidCatalog();
  return {
    categories: value.categories.map(category => {
      if (!category || typeof category !== 'object' || !Array.isArray(category.products)) throw invalidCatalog();
      const id = canonicalID(category.id);
      const products = category.products.map(product => {
        const snapshot = snapshotProduct(product);
        if (snapshot.category_id !== id) throw invalidCatalog();
        return snapshot;
      });
      return { id, name: requiredString(category.name), products };
    }),
  };
}

function parseProduct(value) {
  if (!value || typeof value !== 'object' || !Object.hasOwn(value, 'product')) throw invalidCatalog();
  return snapshotProduct(value.product);
}

function normalizeFailure(error) {
  if (error && (error.code === 'PRODUCT_NOT_FOUND' || error.code === 'CATALOG_UNAVAILABLE')) return error;
  return invalidCatalog();
}

async function loadCatalog() {
  try {
    return parseCatalog(await catalogApi.listCatalog());
  } catch (error) {
    throw normalizeFailure(error);
  }
}

async function loadProduct(id) {
  try {
    return parseProduct(await catalogApi.getProduct(id));
  } catch (error) {
    throw normalizeFailure(error);
  }
}

function flattenProducts(categories) {
  return categories.reduce((all, category) => all.concat(category.products), []);
}

module.exports = {
  flattenProducts,
  formatCents,
  loadCatalog,
  loadProduct,
  snapshotProduct,
  withPrice,
};
