import http from 'node:http';

const catalog = {
  categories: [
    {
      id: '101',
      name: '主食',
      products: [
        {
          id: '1001',
          category_id: '101',
          name: '恢复后的热菜',
          description: '由 loopback fixture 返回',
          specification: '份',
          price_cents: 1800,
        },
      ],
    },
    {
      id: '102',
      name: '饮品',
      products: [
        {
          id: '1002',
          category_id: '102',
          name: '无糖热饮',
          description: '用于菜单分类交互',
          specification: '杯',
          price_cents: 600,
        },
      ],
    },
  ],
};

function writeJSON(response, statusCode, body) {
  response.writeHead(statusCode, {
    'Access-Control-Allow-Origin': '*',
    'Content-Type': 'application/json; charset=utf-8',
  });
  response.end(JSON.stringify(body));
}

export async function startCatalogFixture() {
  let catalogRequests = 0;
  const server = http.createServer((request, response) => {
    if (request.method !== 'GET' || request.url !== '/api/v1/catalog') {
      writeJSON(response, 404, { error: { code: 'NOT_FOUND' } });
      return;
    }
    catalogRequests += 1;
    if (catalogRequests === 1) {
      writeJSON(response, 503, { error: { code: 'CATALOG_UNAVAILABLE' } });
      return;
    }
    writeJSON(response, 200, catalog);
  });

  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(8080, '127.0.0.1', resolve);
  });

  return {
    close: () => new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve())),
  };
}
