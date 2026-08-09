Component({
  data: { visible: false, msg: '', icon: 'check', undoable: false },
  _timer: null,
  _onUndo: null,
  methods: {
    /**
     * @param {string} msg
     * @param {object} opts { icon, onUndo, duration }
     */
    show(msg, opts = {}) {
      this._onUndo = typeof opts.onUndo === 'function' ? opts.onUndo : null;
      this.setData({ visible: true, msg, icon: opts.icon || 'check', undoable: !!this._onUndo });
      clearTimeout(this._timer);
      this._timer = setTimeout(() => this.setData({ visible: false }), opts.duration || 3000);
    },
    hide() { clearTimeout(this._timer); this.setData({ visible: false }); },
    onUndo() { if (this._onUndo) this._onUndo(); this.hide(); },
  },
});
