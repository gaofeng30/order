/* ============================================================
   算价引擎 —— 等级折扣 + 优惠券
   二期能力，不在一期合同范围。

   已确认的业务规则（写死，不做可配置）：
   1. 算价链：原价小计 → 等级折扣 → 减券 → 应付。
   2. 一单只用一张券；券与等级折扣叠加，不择优。
   3. 券的「门槛判定」与「减免作用」均锚定可用商品小计，
      且作用在等级折扣之后的金额上，保证优惠永不超过实付。
   4. 折扣券必须有封顶金额。
   5. 默认自动选中优惠力度最大的券，用户可手动切换。
   6. 不可用的券不隐藏，置底并给出具体原因。

   金额一律用「分」做整数运算，仅在输出时转元，避免浮点误差。
   ============================================================ */
const data = require('./data.js');

const yuanToCent = y => Math.round(Number(y || 0) * 100);
const centToYuan = c => (c / 100).toFixed(2);
// 去掉无意义的尾随零：3000 → '30'，3050 → '30.5'
const centToText = c => centToYuan(c).replace(/\.00$/, '').replace(/(\.\d)0$/, '$1');

// 折扣百分比转中文「几折」：85 → '8.5折'，90 → '9折'，100 → '无折扣'
function discountLabel(discount) {
  const d = Number(discount);
  if (!(d > 0) || d >= 100) return '无折扣';
  return String(d / 10).replace(/\.0$/, '') + '折';
}

// 券面文案
function faceText(c) {
  return c.type === 'cut' ? '¥' + centToText(yuanToCent(c.amount)) : discountLabel(c.rate);
}

// 适用范围文案
function scopeLabel(c) {
  if (c.scope === 'all') return '全场通用';
  if (c.scope === 'cat') return '限' + (c.catNames || []).join('、');
  const names = (c.itemIds || []).map(id => {
    const m = data.itemById(id);
    return m ? m.name : '已删除菜品';
  });
  return '限' + names.join('、');
}

// 使用条件文案
function condLabel(c) {
  const th = yuanToCent(c.threshold);
  const base = th > 0 ? `满 ¥${centToText(th)} 可用` : '无门槛';
  return c.type === 'discount' ? `${base} · 最高减 ¥${centToText(yuanToCent(c.cap))}` : base;
}

// 某菜品是否落在券的适用范围内
function itemInScope(c, item) {
  if (c.scope === 'all') return true;
  if (c.scope === 'cat') return (c.catNames || []).indexOf(item.cat) > -1;
  return (c.itemIds || []).indexOf(item.id) > -1;
}

// 购物车中落在券范围内的商品原价小计（分）
function eligibleCent(items, c) {
  return items.reduce((a, ci) => a + (itemInScope(c, ci.item) ? yuanToCent(ci.item.price) * ci.q : 0), 0);
}

// 按等级折扣率折算金额（分）
function applyLevel(cent, level) {
  if (!level || !(level.discount > 0) || level.discount >= 100) return cent;
  return Math.round(cent * level.discount / 100);
}

/* 单张券对当前购物车的结果
   → { ok, cutC, reason }  cutC 为可减金额（分） */
function evalCoupon(c, items, level, usedCount, today) {
  const t = today || data.TODAY;
  if (t < c.start) return { ok: false, cutC: 0, reason: `${c.start} 起可用` };
  if (t > c.end) return { ok: false, cutC: 0, reason: '已过期' };
  if ((usedCount || 0) >= c.perLimit) return { ok: false, cutC: 0, reason: `本期限用 ${c.perLimit} 次，已用完` };

  const eligC = eligibleCent(items, c);
  if (eligC <= 0) return { ok: false, cutC: 0, reason: scopeLabel(c) + '，购物车暂无适用商品' };

  const baseC = applyLevel(eligC, level);          // 券作用于等级折扣后的可用商品小计
  const thC = yuanToCent(c.threshold);
  if (baseC < thC) return { ok: false, cutC: 0, reason: `还差 ¥${centToText(thC - baseC)} 可用` };

  let cutC;
  if (c.type === 'cut') {
    cutC = Math.min(yuanToCent(c.amount), baseC);
  } else {
    cutC = Math.min(Math.round(baseC * (100 - c.rate) / 100), yuanToCent(c.cap), baseC);
  }
  return { ok: cutC > 0, cutC, reason: cutC > 0 ? '' : '本单无可减金额' };
}

/* 整单算价
   入参 { items, level, coupons, used, couponId }
     items    cart.list() 的结果 [{ item, q }]
     level    当前用户等级对象，非会员传 null
     coupons  本人可见的券（api.listMyCoupons）
     used     { [couponId]: 已用次数 }
     couponId 手动选中的券 id；未传或不可用时自动选最优
   出参见文件末尾注释 */
function calc(opt) {
  const items = opt.items || [];
  const level = opt.level || null;
  const used = opt.used || {};
  const list = (opt.coupons || []).filter(c => c.enabled);

  const subtotalC = items.reduce((a, ci) => a + yuanToCent(ci.item.price) * ci.q, 0);
  const afterLevelC = applyLevel(subtotalC, level);
  const levelCutC = subtotalC - afterLevelC;

  const usable = [];
  const unusable = [];
  list.forEach(c => {
    const r = evalCoupon(c, items, level, used[c.id], opt.today);
    const row = {
      id: c.id, coupon: c, face: faceText(c), scope: scopeLabel(c), cond: condLabel(c),
      cutC: r.cutC, cutText: centToText(r.cutC), reason: r.reason,
    };
    if (r.ok) usable.push(row); else unusable.push(row);
  });
  usable.sort((a, b) => b.cutC - a.cutC);

  // couponId === 'none' 表示用户明确选择不使用优惠券；
  // 手动选中的券若已不可用（改了购物车等），回落到最优券
  let picked = null;
  if (opt.couponId !== 'none') {
    if (opt.couponId) picked = usable.find(r => r.id === opt.couponId) || null;
    if (!picked) picked = usable[0] || null;
  }

  const couponCutC = picked ? Math.min(picked.cutC, afterLevelC) : 0;
  const payableC = Math.max(0, afterLevelC - couponCutC);

  return {
    subtotalC, levelCutC, couponCutC, payableC,
    subtotal: centToText(subtotalC),
    levelCut: centToText(levelCutC),
    couponCut: centToText(couponCutC),
    payable: centToText(payableC),
    hasLevel: !!(level && level.discount < 100),
    levelName: level ? level.name : '',
    levelLabel: level ? discountLabel(level.discount) : '',
    coupon: picked ? picked.coupon : null,
    couponId: picked ? picked.id : '',
    couponName: picked ? picked.coupon.name : '',
    usable, unusable,
    totalCutC: levelCutC + couponCutC,
    totalCut: centToText(levelCutC + couponCutC),
  };
}

module.exports = {
  calc, evalCoupon, faceText, scopeLabel, condLabel, discountLabel,
  yuanToCent, centToYuan, centToText, eligibleCent, itemInScope,
};
