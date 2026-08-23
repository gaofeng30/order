const { isRuntimeOrigin } = require('./runtimeEndpoint.js');

class CatalogError extends Error {
  constructor(code) {
    super(code === 'PRODUCT_NOT_FOUND' ? 'product not found' : 'catalog unavailable');
    this.name = 'CatalogError';
    this.code = code;
  }
}

function request(path, notFoundCode) {
  return new Promise((resolve, reject) => {
    let baseUrl;
    try {
      const globalData = getApp().globalData;
      const endpoint = globalData.runtimeEndpoint;
      baseUrl = globalData.apiBaseUrl;
      if (!endpoint
        || endpoint.state !== 'ready'
        || endpoint.origin !== baseUrl
        || !isRuntimeOrigin(endpoint.envVersion, baseUrl)) {
        throw new CatalogError('CATALOG_UNAVAILABLE');
      }
      wx.request({
        url: `${baseUrl}${path}`,
        method: 'GET',
        success(response) {
          if (response.statusCode === 200) {
            resolve(response.data);
            return;
          }
          if (response.statusCode === 404 && notFoundCode) {
            reject(new CatalogError(notFoundCode));
            return;
          }
          reject(new CatalogError('CATALOG_UNAVAILABLE'));
        },
        fail() {
          reject(new CatalogError('CATALOG_UNAVAILABLE'));
        },
      });
    } catch (error) {
      reject(new CatalogError('CATALOG_UNAVAILABLE'));
    }
  });
}

function listCatalog() {
  return request('/api/v1/catalog');
}

function getProduct(id) {
  return request(`/api/v1/catalog/products/${encodeURIComponent(String(id))}`, 'PRODUCT_NOT_FOUND');
}

module.exports = { CatalogError, listCatalog, getProduct };
