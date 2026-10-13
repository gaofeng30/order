/* 日期文案的唯一入口（G8）：今天/明天/昨天用相对说法，今年写「10月8日」，往年写「2025年12月30日」。

   门店时区固定为 Asia/Shanghai（PRD §16）。「今天」一律按北京时间计算，不读设备时区；
   'YYYY-MM-DD' 只做字符串拆分，不交给 new Date() 解析，避免被当成 UTC 零点。 */
const SHANGHAI_OFFSET_MS = 8 * 60 * 60 * 1000;
const DAY_MS = 24 * 60 * 60 * 1000;
const DATE = /^(\d{4})-(\d{2})-(\d{2})$/;

function shanghaiToday(nowMs) {
  const now = Number.isFinite(nowMs) ? nowMs : Date.now();
  return new Date(now + SHANGHAI_OFFSET_MS).toISOString().slice(0, 10);
}

function dayNumber(date) {
  const match = DATE.exec(date);
  if (!match) return null;
  return Date.UTC(Number(match[1]), Number(match[2]) - 1, Number(match[3])) / DAY_MS;
}

function dayLabel(date, today) {
  const target = dayNumber(date);
  if (target === null) return '';
  const base = dayNumber(today || shanghaiToday());
  const diff = base === null ? null : target - base;
  if (diff === 0) return '今天';
  if (diff === 1) return '明天';
  if (diff === -1) return '昨天';
  const [year, month, day] = date.split('-').map(Number);
  const sameYear = base !== null && (today || shanghaiToday()).slice(0, 4) === date.slice(0, 4);
  return sameYear ? `${month}月${day}日` : `${year}年${month}月${day}日`;
}

// 后端 RFC3339 时间点 → 「昨天 21:05」
function instantLabel(value, nowMs) {
  const instant = Date.parse(value);
  if (!Number.isFinite(instant)) return '';
  const local = new Date(instant + SHANGHAI_OFFSET_MS).toISOString();
  return `${dayLabel(local.slice(0, 10), shanghaiToday(nowMs))} ${local.slice(11, 16)}`;
}

module.exports = { dayLabel, instantLabel, shanghaiToday };
