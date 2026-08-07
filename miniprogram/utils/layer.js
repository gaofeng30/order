// 开屏装饰图层 — 本机持久化访问模块（模拟后端，风格参照 util.js 的 orderMode 访问器）
// 配置与图片仅存于本机：wx.setStorageSync + USER_DATA_PATH 文件，跨设备下发需后端（超出当前范围）
const KEY = 'layerOverlay.v1';
const DEFAULTS = {
  v: 1,            // 配置版本，便于后续迁移
  enabled: false,  // 是否启用
  src: '',         // 图片持久路径（USER_DATA_PATH 下）
  cx: 0.5,         // 图片中心 x，屏宽比例 0-1
  cy: 0.38,        // 图片中心 y，屏高比例 0-1
  size: 0.35,      // 图片宽度，屏宽比例 0.15-0.60
  ar: 1,           // 图片高宽比（编辑器定高用）
};

function fs() { return wx.getFileSystemManager(); }

// 读配置：合并默认值；图片文件被系统清理时静默降级，绝不返回坏图路径
function get() {
  let cfg = {};
  try { cfg = wx.getStorageSync(KEY) || {}; } catch (e) { cfg = {}; }
  cfg = Object.assign({}, DEFAULTS, cfg);
  if (cfg.src) {
    try { fs().accessSync(cfg.src); }
    catch (e) { cfg.src = ''; cfg.enabled = false; }
  }
  return cfg;
}

function save(cfg) {
  wx.setStorageSync(KEY, Object.assign({}, DEFAULTS, cfg, { v: 1 }));
}

// 临时图片落盘：时间戳文件名避免更换后 <image> 命中旧缓存；替换时清理旧文件
function persistImage(tempPath, oldSrc) {
  const dest = wx.env.USER_DATA_PATH + '/layer-overlay-' + Date.now() + '.png';
  fs().copyFileSync(tempPath, dest);
  if (oldSrc) { try { fs().unlinkSync(oldSrc); } catch (e) { /* 旧文件可能已不存在 */ } }
  return dest;
}

// 移除图层：删文件 + 清配置
function clear() {
  const cfg = get();
  if (cfg.src) { try { fs().unlinkSync(cfg.src); } catch (e) {} }
  try { wx.removeStorageSync(KEY); } catch (e) {}
}

module.exports = { get, save, persistImage, clear, DEFAULTS };
