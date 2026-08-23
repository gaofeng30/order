const api = require('./apiClient.js');

const STATUS = { open: '营业中', closed: '休息中', cutoff: '已截单' };

function text(value, required) {
  if (typeof value !== 'string' || value !== value.trim() || (required && !value)) {
    throw new api.APIError('STOREFRONT_UNAVAILABLE');
  }
  return value;
}

function parseFlavors(value) {
  if (!Array.isArray(value)) throw new api.APIError('STOREFRONT_UNAVAILABLE');
  const seen = new Set();
  return value.map(item => {
    const flavor = text(item, true);
    if (seen.has(flavor)) throw new api.APIError('STOREFRONT_UNAVAILABLE');
    seen.add(flavor);
    return flavor;
  });
}

function finiteRatio(value, minimum, maximum) {
  return typeof value === 'number' && Number.isFinite(value) && value >= minimum && value <= maximum;
}

function parseLaunchLayer(value) {
  if (value === undefined || value === null) return null;
  if (!value || typeof value !== 'object' || Array.isArray(value)
    || !value.image || typeof value.image !== 'object' || Array.isArray(value.image)) {
    throw new api.APIError('STOREFRONT_UNAVAILABLE');
  }
  const objectKey = text(value.image.object_key, true);
  if (!finiteRatio(value.center_x, 0, 1) || !finiteRatio(value.center_y, 0, 1)
    || !finiteRatio(value.width_ratio, Number.MIN_VALUE, 1)
    || !finiteRatio(value.aspect_ratio, Number.MIN_VALUE, Number.MAX_VALUE)) {
    throw new api.APIError('STOREFRONT_UNAVAILABLE');
  }
  let url;
  try {
    url = api.resolvePublicURL(value.image.url, objectKey);
  } catch (error) {
    throw new api.APIError('STOREFRONT_UNAVAILABLE');
  }
  const widthPercent = value.width_ratio * 100;
  return {
    image: { objectKey, url },
    centerX: value.center_x,
    centerY: value.center_y,
    widthRatio: value.width_ratio,
    aspectRatio: value.aspect_ratio,
    leftPercent: value.center_x * 100,
    topPercent: value.center_y * 100,
    widthPercent,
    heightVw: widthPercent / value.aspect_ratio,
  };
}

function parse(value) {
  const settings = value && value.storefront;
  if (!settings || typeof settings !== 'object' || Array.isArray(settings)) {
    throw new api.APIError('STOREFRONT_UNAVAILABLE');
  }
  const name = settings.name;
  const address = settings.address;
  const point = settings.pickup_point;
  const announcement = settings.announcement;
  const status = settings.business_status;
  if (!Object.hasOwn(STATUS, status)) throw new api.APIError('STOREFRONT_UNAVAILABLE');
  return {
    name: text(name, true), address: text(address, true), pickupPoint: text(point, true),
    announcement: text(announcement, false), businessStatus: status,
    businessStatusLabel: STATUS[status], launchLayer: parseLaunchLayer(settings.launch_layer),
    flavors: parseFlavors(settings.flavors),
  };
}

async function load() { return parse(await api.getOptional('/api/v1/storefront/settings')); }

module.exports = { load, parse };
