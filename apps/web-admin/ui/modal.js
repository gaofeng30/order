/* Modal：实例独立关闭；遮罩退出等同取消。 */
(function () {
  const instances = [];
  let revision = 0;
  function open(cfg) {
    ++revision;
    const previous = cfg.stack && instances.length > 0;
    if (!cfg.stack) close();
    const host = document.getElementById('overlay-host');
    const mask = document.createElement('div'); mask.className = 'mask modal-mask';
    if (previous) mask.style.zIndex = '430';
    const box = document.createElement('div'); box.className = 'modal';
    if (previous) box.style.zIndex = '440';
    box.setAttribute('role', 'dialog'); box.setAttribute('aria-modal', 'true');
    box.setAttribute('aria-label', cfg.title || '提示');
    if (cfg.width) box.style.width = cfg.width;
    box.innerHTML = (cfg.title ? `<div class="md-head">${window.Toast.esc(cfg.title)}</div>` : '') +
      `<div class="md-body">${cfg.bodyHtml || ''}</div>` +
      (cfg.footerHtml ? `<div class="md-foot">${cfg.footerHtml}</div>` : '');
    let closed = false;
    const done = reason => {
      if (closed) return;
      ++revision;
      closed = true; mask.remove(); box.remove();
      const index = instances.indexOf(done);
      if (index >= 0) instances.splice(index, 1);
      if (cfg.onDismiss) cfg.onDismiss(typeof reason === 'string' ? reason : 'cancel');
    };
    instances.push(done); host.append(mask, box); mask.onclick = done;
    if (cfg.onMount) cfg.onMount(box, done);
    return box;
  }
  function close() { while (instances.length) instances[instances.length - 1]('replaced'); }
  function confirm(cfg) {
    return new Promise(resolve => {
      open({ title: cfg.title, bodyHtml: cfg.body || '', onDismiss: () => resolve(false),
        footerHtml: `<button class="btn btn--line" data-a="cancel">${window.Toast.esc(cfg.cancelText || '取消')}</button><button class="btn ${cfg.danger ? 'btn--danger' : 'btn--primary'}" data-a="ok">${window.Toast.esc(cfg.okText || '确定')}</button>`,
        onMount(root, done) {
          root.querySelector('[data-a="cancel"]').onclick = done;
          root.querySelector('[data-a="ok"]').onclick = () => { resolve(true); done(); };
        }
      });
    });
  }
  window.Modal = { open, close, confirm, revision: () => revision };
})();
