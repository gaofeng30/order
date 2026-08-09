/* 券票卡 —— 商户端券列表 / 用户端卡包 / 确认页选券 三处共用
   仅负责呈现，视图模型由调用方用 utils/promo.js 组装。 */
Component({
  properties: {
    face: { type: String, value: '' },      // 券面：'¥5' / '8折'
    cond: { type: String, value: '' },      // 使用条件
    name: { type: String, value: '' },
    scope: { type: String, value: '' },     // 适用范围
    period: { type: String, value: '' },    // 有效期
    tags: { type: Array, value: [] },       // 适用等级名（商户端展示）
    reason: { type: String, value: '' },    // 不可用原因
    badge: { type: String, value: '' },     // 角标文案
    badgeTone: { type: String, value: 'ok' },
    dim: { type: Boolean, value: false },   // 置灰（停用 / 过期 / 不可用）
    picked: { type: Boolean, value: false },
    showPick: { type: Boolean, value: false },
    attached: { type: Boolean, value: false }, // 下方拼接操作条时用
    notch: { type: String, value: '#f4f3ec' }, // 齿孔需与所在容器底色一致
  },
  methods: {
    tap() { this.triggerEvent('tap'); },
  },
});
