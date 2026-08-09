/* Toast —— 对应小程序 components/toast，含「撤销」二次操作位 */
(function () {
  const host = () => document.getElementById('toast-host');

  // show(msg, { icon, onUndo, duration })
  function show(msg, opts) {
    const o = opts || {};
    const el = document.createElement('div');
    el.className = 'toast';
    const ic = o.icon || 'check';
    const tone = ic === 'warn' ? 'ti--warn' : 'ti--check';
    el.innerHTML =
      `<span class="${tone}">${window.Icon.svg(ic, 17)}</span><span>${esc(msg)}</span>` +
      (o.onUndo ? '<span class="tundo">撤销</span>' : '');
    host().appendChild(el);

    let timer = null;
    const dismiss = () => {
      if (!el.parentNode) return;
      clearTimeout(timer);
      el.classList.add('out');
      setTimeout(() => el.remove(), 200);
    };
    if (o.onUndo) {
      el.querySelector('.tundo').addEventListener('click', () => { o.onUndo(); dismiss(); });
    }
    timer = setTimeout(dismiss, o.duration || (o.onUndo ? 4000 : 2200));
    return dismiss;
  }

  function esc(s) {
    return String(s).replace(/[&<>"']/g, c =>
      ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
  }

  window.Toast = { show, esc };
})();
