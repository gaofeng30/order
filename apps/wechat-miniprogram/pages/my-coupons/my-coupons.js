/* 我的优惠券 —— 二期能力，不在一期合同范围
   券按等级自动生效，用户没有领取动作；此页只做分态展示。 */
const api = require('../../utils/api.js');
const data = require('../../utils/data.js');
const promo = require('../../utils/promo.js');

const md = d => (d || '').slice(5);
const maskPhone = p => (p && p.length === 11 ? p.slice(0, 3) + '****' + p.slice(7) : p);

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    isMember: false,
    member: null,
    level: null,
    levelLabel: '—',
    initial: '会',
    phoneMask: '',
    usableCount: 0,
    tab: 'usable',
    tabs: [],
    list: [],
  },
  onShow() { this.load(); },

  load() {
    Promise.all([api.getMyMembership(), api.listMyCoupons(), api.myCouponUsed()]).then(([me, cps, used]) => {
      if (!me.isMember) return this.setData({ isMember: false, list: [], usableCount: 0 });

      const today = data.TODAY;
      const all = cps.map(c => {
        const usedN = used[c.id] || 0;
        let state = 'usable';
        let badge = '可用';
        let tone = 'ok';
        let reason = '';
        if (today > c.end) { state = 'expired'; badge = '已过期'; tone = 'mute'; }
        else if (usedN >= c.perLimit) { state = 'used'; badge = '已用完'; tone = 'mute'; reason = `有效期内限用 ${c.perLimit} 次`; }
        else if (today < c.start) { badge = '未开始'; tone = 'info'; reason = `${md(c.start)} 起可用`; }
        else if (c.perLimit - usedN < c.perLimit) { reason = `还可用 ${c.perLimit - usedN} 次`; }
        return {
          id: c.id, state,
          face: promo.faceText(c), cond: promo.condLabel(c), name: c.name,
          scope: promo.scopeLabel(c), period: `${md(c.start)} 至 ${md(c.end)}`,
          badge, badgeTone: tone, reason, dim: state !== 'usable',
        };
      });

      const cnt = id => all.filter(x => x.state === id).length;
      this.setData({
        isMember: true,
        member: me.member,
        level: me.level,
        levelLabel: promo.discountLabel(me.level.discount),
        initial: me.member.name ? me.member.name[0] : '会',
        phoneMask: maskPhone(me.member.phone),
        usableCount: cnt('usable'),
        _all: all,
        tabs: [
          { id: 'usable', name: '可用', count: cnt('usable') },
          { id: 'used', name: '已用完', count: cnt('used') },
          { id: 'expired', name: '已过期', count: cnt('expired') },
        ],
        list: all.filter(x => x.state === this.data.tab),
      });
    });
  },

  switchTab(e) {
    const tab = e.currentTarget.dataset.id;
    this.setData({ tab, list: (this.data._all || []).filter(x => x.state === tab) });
  },
});
