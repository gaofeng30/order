const api = require('./apiClient.js');

const STATUS = { open: '营业中', closed: '休息中', cutoff: '已截单' };

function text(value, required) {
  if (typeof value !== 'string' || (required && !value.trim())) throw new api.APIError('STOREFRONT_UNAVAILABLE');
  return value;
}

function parse(value) {
  const settings = value && (value.settings || value.storefront);
  if (!settings || typeof settings !== 'object' || Array.isArray(settings)) {
    throw new api.APIError('STOREFRONT_UNAVAILABLE');
  }
  const name = settings.store_name === undefined ? settings.name : settings.store_name;
  const address = settings.store_address === undefined ? settings.address : settings.store_address;
  const point = settings.pickup_point;
  const announcement = settings.announcement;
  const status = settings.business_status;
  if (!Object.hasOwn(STATUS, status)) throw new api.APIError('STOREFRONT_UNAVAILABLE');
  return {
    name: text(name, true), address: text(address, true), pickupPoint: text(point, true),
    announcement: text(announcement, false), businessStatus: status,
    businessStatusLabel: STATUS[status], launchLayer: settings.launch_layer || null,
    flavors: Array.isArray(settings.flavors) ? settings.flavors.slice() : [],
  };
}

async function load() { return parse(await api.getOptional('/api/v1/storefront/settings')); }

module.exports = { load, parse };
