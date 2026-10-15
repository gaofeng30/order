const { isRuntimeEndpointReady } = require('./runtimeEndpoint.js');

// The fixed UAT gateway and develop origin share the existing HTTP callback contract.
function requestRuntime(endpoint, path, options) {
  try {
    if (!isRuntimeEndpointReady(endpoint, endpoint && endpoint.origin)) throw new Error('runtime unavailable');
    wx.request({ ...options, url: `${endpoint.origin}${path}` });
  } catch (error) {
    options.fail();
  }
}

module.exports = { requestRuntime };
