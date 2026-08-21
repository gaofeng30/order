/* 整数分 → 元的唯一格式化入口（§5.6：金额一律以整数分保存与计算）。

   刻意不复用 catalogStore.formatCents：后者在输入非法时抛
   CatalogError('CATALOG_UNAVAILABLE')，那是目录接口的错误语义，
   用在渲染金额上会把一个显示问题伪装成网络故障。

   页面与模板 MUST NOT 自己做 / 100 或 toFixed(2) —— 一旦允许各处转换，
   舍入就会分散到 N 个地方，而 N 个地方迟早有一个算得不一样。 */
function yuan(cents) {
  // Number(null) 是 0，不加这道判断会把「没有金额」显示成 0.00
  if (cents === null || cents === undefined || cents === '') return '—';
  const n = Number(cents);
  if (!Number.isFinite(n)) return '—';
  const neg = n < 0;
  const v = Math.abs(Math.round(n));
  return (neg ? '-' : '') + Math.floor(v / 100) + '.' + String(v % 100).padStart(2, '0');
}

module.exports = { yuan };
