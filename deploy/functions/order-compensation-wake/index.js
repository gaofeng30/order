'use strict';

// No business credentials or retry loop: the database owns durable delivery.
const endpoint = 'https://order.suianwuye.com/internal/compensation/wake';

function createHandler(request = globalThis.fetch) {
  return async function wake(event) {
    if (!event || Array.isArray(event) || typeof event !== 'object' ||
        Object.keys(event).length !== 1 || typeof event.token !== 'string' ||
        !/^[a-f0-9]{64}$/.test(event.token)) {
      return { accepted: false };
    }
    try {
      const response = await request(endpoint, {
        method: 'POST',
        headers: { Authorization: `Bearer ${event.token}` },
        redirect: 'error',
        signal: AbortSignal.timeout(25000)
      });
      return { accepted: response.status === 204 };
    } catch {
      // Transport diagnostics can contain the bearer token. Never return/log them.
      return { accepted: false };
    }
  };
}

exports.createHandler = createHandler;
exports.main = createHandler();
