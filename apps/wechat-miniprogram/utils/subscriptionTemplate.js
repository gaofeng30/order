const ENV_VERSIONS = ['develop', 'trial', 'release'];
const KINDS = ['READY', 'REFUND_RESULT'];

function validTemplateID(value) {
  if (typeof value !== 'string' || value.length < 1 || value.length > 128) return false;
  for (let index = 0; index < value.length; index += 1) {
    const code = value.charCodeAt(index);
    if (code < 0x21 || code > 0x7e) return false;
  }
  return true;
}

function resolveSubscriptionTemplateIds(envVersion, deploymentConfig) {
  if (!ENV_VERSIONS.includes(envVersion)) return {};
  const configured = deploymentConfig && deploymentConfig[envVersion];
  if (!configured || typeof configured !== 'object' || Array.isArray(configured)) return {};
  const resolved = {};
  for (const kind of KINDS) {
    const value = configured[kind];
    if (value === undefined || value === '') continue;
    if (!validTemplateID(value)) return {};
    resolved[kind] = value;
  }
  return resolved;
}

module.exports = { resolveSubscriptionTemplateIds };
