import http from 'node:http';

const storefront = {
  storefront: {
    name: '绥安食品', address: '党政办公中心后院老食堂', pickup_point: '北门',
    announcement: '今日供应以服务端为准', business_status: 'open', launch_layer: null,
    flavors: [],
  },
};
const pickupOptions = {
  timezone: 'Asia/Shanghai',
  dates: [{
    date: '2026-08-25', available: true,
    meal_periods: [{ meal_period: 'dinner', cutoff_time: '17:00', available: true, pickup_times: ['17:30'] }],
  }],
};
const menu = {
  selection: { date: '2026-08-25', time: '17:30', meal_period: 'dinner' },
  store_status: {
    business_status: 'open', service_date_available: true, meal_available: true, cutoff_passed: false,
  },
  categories: [
    { id: '101', name: '主食', products: [{ id: '1001', category_id: '101', name: '恢复后的热菜', description: '由 loopback fixture 返回', specification: '份', meal_period: 'all', images: [], listed: true, sold_out: false, original_unit_price_cents: 1800 }] },
    { id: '102', name: '饮品', products: [{ id: '1002', category_id: '102', name: '无糖热饮', description: '用于菜单分类交互', specification: '杯', meal_period: 'all', images: [], listed: true, sold_out: false, original_unit_price_cents: 600 }] },
  ],
};

function writeJSON(response, statusCode, body) {
  response.writeHead(statusCode, { 'Access-Control-Allow-Origin': '*', 'Content-Type': 'application/json; charset=utf-8' });
  response.end(JSON.stringify(body));
}

export async function startCatalogFixture() {
  let pickupRequests = 0;
  const server = http.createServer((request, response) => {
    const url = new URL(request.url, 'http://127.0.0.1:8080');
    if (request.method === 'POST' && url.pathname === '/api/v1/auth/miniprogram/session') {
      writeJSON(response, 201, { access_token: 'ui1-session-token-abcdefghijklmnopqrstuvwxyz012345', token_type: 'Bearer', expires_at: '2999-08-25T08:00:00Z' });
      return;
    }
    if (request.method === 'GET' && url.pathname === '/api/v1/me/identity') {
      writeJSON(response, 200, { identity: {
        primary_phone: { bound: false, masked_phone: '' },
        extra_phone: { set: false, masked_phone: '' },
        pricing_identity: { kind: 'VISITOR', rate_percent: 100 },
        merchant: { bound: false },
      } });
      return;
    }
    if (request.method === 'GET' && url.pathname === '/api/v1/storefront/settings') {
      writeJSON(response, 200, storefront);
      return;
    }
    if (request.method === 'GET' && url.pathname === '/api/v1/orders') {
      writeJSON(response, 200, { orders: [] });
      return;
    }
    if (request.method === 'GET' && url.pathname === '/api/v1/menu/pickup-options') {
      pickupRequests += 1;
      if (pickupRequests === 1) writeJSON(response, 503, { error: { code: 'MENU_UNAVAILABLE' } });
      else writeJSON(response, 200, pickupOptions);
      return;
    }
    if (request.method === 'GET' && url.pathname === '/api/v1/menu') {
      writeJSON(response, 200, menu);
      return;
    }
    writeJSON(response, 404, { error: { code: 'NOT_FOUND' } });
  });

  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(8080, '127.0.0.1', resolve);
  });
  return { close: () => new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve())) };
}
