Component({
  data: { visible: false, msg: '', icon: 'check' },
  _timer: null,
  methods: {
    /**
     * @param {string} msg
     * @param {object} opts { icon, duration }
     * 生效 spec 禁止生产撤销，因此 Toast 不提供回退动作。
     */
    show(msg, opts = {}) {
      this.setData({ visible: true, msg, icon: opts.icon || 'check' });
      clearTimeout(this._timer);
      this._timer = setTimeout(() => this.setData({ visible: false }), opts.duration || 3000);
    },
    hide() { clearTimeout(this._timer); this.setData({ visible: false }); },
  },
});
