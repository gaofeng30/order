const api = require('./apiClient.js');

class CatalogError extends Error {
  constructor(code) {
    super(code === 'PRODUCT_NOT_FOUND' ? 'product not found' : 'catalog unavailable');
    this.name = 'CatalogError';
    this.code = code;
  }
}

function unavailable(error) {
  if (error && error.code === 'PRODUCT_NOT_FOUND') return new CatalogError('PRODUCT_NOT_FOUND');
  return new CatalogError('CATALOG_UNAVAILABLE');
}

async function listCatalog() {
  try { return await api.getOptional('/api/v1/catalog'); } catch (error) { throw unavailable(error); }
}

async function getProduct(id, selection) {
  if (!/^[1-9]\d*$/.test(String(id)) || !selection
    || !/^\d{4}-\d{2}-\d{2}$/.test(selection.date)
    || !/^\d{2}:\d{2}$/.test(selection.time)) throw new CatalogError('CATALOG_UNAVAILABLE');
  const path = `/api/v1/catalog/products/${encodeURIComponent(String(id))}`
    + `?date=${selection.date}&time=${encodeURIComponent(selection.time)}`;
  try { return await api.getOptional(path); } catch (error) { throw unavailable(error); }
}

module.exports = { CatalogError, listCatalog, getProduct };
