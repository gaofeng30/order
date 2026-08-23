const ENV_VERSIONS = ['develop', 'trial', 'release'];

function errorState(envVersion, errorCode) {
  return { state: 'error', envVersion, origin: '', errorCode };
}

function isValidHostname(hostname) {
  if (hostname.length > 253 || hostname.endsWith('.')) return false;
  if (/^[0-9.]+$/.test(hostname)) {
    const parts = hostname.split('.');
    return parts.length === 4 && parts.every(part => {
      if (!/^\d{1,3}$/.test(part)) return false;
      if (part.length > 1 && part.startsWith('0')) return false;
      return Number(part) <= 255;
    });
  }
  return hostname.split('.').every(label => (
    /^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$/.test(label)
  ));
}

function normalizeOrigin(configured) {
  if (typeof configured !== 'string' || !configured || configured !== configured.trim()) return null;
  const originMatch = /^(https?):\/\/([^/?#]+)\/?$/i.exec(configured);
  if (!originMatch) return null;

  const authorityMatch = /^([^:]+)(?::(\d+))?$/.exec(originMatch[2]);
  if (!authorityMatch) return null;
  const hostname = authorityMatch[1];
  const port = authorityMatch[2] || '';
  if (!isValidHostname(hostname)) return null;
  if (port && (Number(port) < 1 || Number(port) > 65535)) return null;

  const protocol = originMatch[1].toLowerCase();
  const normalizedPort = port ? `:${port}` : '';
  return {
    protocol,
    hostname: hostname.toLowerCase(),
    origin: `${protocol}://${hostname.toLowerCase()}${normalizedPort}`,
  };
}

function isDeployableHTTPSOrigin(normalized) {
  if (!normalized || normalized.protocol !== 'https') return false;
  const labels = normalized.hostname.split('.');
  const isNumericNotation = labels.every(label => /^(?:\d+|0x[0-9a-f]+)$/i.test(label));
  const isLocalhost = normalized.hostname === 'localhost'
    || normalized.hostname.endsWith('.localhost');
  return labels.length > 1 && !isNumericNotation && !isLocalhost;
}

function isDevelopLoopbackOrigin(normalized) {
  return normalized
    && normalized.protocol === 'http'
    && normalized.hostname === '127.0.0.1';
}

function isRuntimeOrigin(envVersion, origin) {
  if (!ENV_VERSIONS.includes(envVersion)) return false;
  const normalized = normalizeOrigin(origin);
  if (!normalized || normalized.origin !== origin) return false;
  return envVersion === 'develop'
    ? isDevelopLoopbackOrigin(normalized)
    : isDeployableHTTPSOrigin(normalized);
}

function detectEnvVersion(wxApi) {
  if (!wxApi || typeof wxApi.getAccountInfoSync !== 'function') return 'develop';
  try {
    const account = wxApi.getAccountInfoSync();
    return account && account.miniProgram && account.miniProgram.envVersion;
  } catch (error) {
    return 'unknown';
  }
}

function resolveRuntimeEndpoint(wxApi, deploymentConfig) {
  const detected = detectEnvVersion(wxApi);
  const envVersion = typeof detected === 'string' && detected ? detected : 'unknown';
  if (!ENV_VERSIONS.includes(envVersion)) {
    return errorState(envVersion, 'RUNTIME_ENDPOINT_ENV_UNSUPPORTED');
  }
  const configured = deploymentConfig && deploymentConfig[envVersion];
  if (configured === undefined || configured === null || configured === '') {
    return errorState(envVersion, 'RUNTIME_ENDPOINT_UNCONFIGURED');
  }
  const normalized = normalizeOrigin(configured);
  const isAllowed = envVersion === 'develop'
    ? isDevelopLoopbackOrigin(normalized)
    : isDeployableHTTPSOrigin(normalized);
  if (!isAllowed) {
    return errorState(envVersion, 'RUNTIME_ENDPOINT_INVALID');
  }
  return { state: 'ready', envVersion, origin: normalized.origin, errorCode: '' };
}

module.exports = { isRuntimeOrigin, resolveRuntimeEndpoint };
