/* Modal —— 对应小程序 wx.showModal 与页面内弹层 */
(function () {
  const host = () => document.getElementById('overlay-host');
  let closer = null;

  // open({ title, bodyHtml, footerHtml, width, onMount(root, close) })
  function open(cfg) {
    close();
    const mask = document.createElement('div');
    mask.className = 'mask';
    const box = document.createElement('div');
    box.className = 'modal';
    if (cfg.width) box.style.width = cfg.width;
    box.innerHTML =
      (cfg.title ? `<div class="md-head">${cfg.title}</div>` : '') +
      `<div class="md-body">${cfg.bodyHtml || ''}</div>` +
      (cfg.footerHtml ? `<div class="md-foot">${cfg.footerHtml}</div>` : '');

    host().appendChild(mask);
    host().appendChild(box);
    mask.addEventListener('click', close);
    closer = () => { mask.remove(); box.remove(); closer = null; };
    if (cfg.onMount) cfg.onMount(box, close);
    return box;
  }

  function close() { if (closer) closer(); }

  // confirm({ title, body, okText, cancelText, danger }) → Promise<boolean>
  function confirm(cfg) {
    return new Promise(resolve => {
      const okCls = cfg.danger ? 'btn--danger' : 'btn--blue';
      open({
        title: cfg.title,
        bodyHtml: cfg.body || '',
        footerHtml:
          `<button class="btn btn--line" data-a="cancel">${cfg.cancelText || '取消'}</button>` +
          `<button class="btn ${okCls}" data-a="ok">${cfg.okText || '确定'}</button>`,
        onMount(root, done) {
          root.querySelector('[data-a="cancel"]').onclick = () => { done(); resolve(false); };
          root.querySelector('[data-a="ok"]').onclick = () => { done(); resolve(true); };
        },
      });
    });
  }

  window.Modal = { open, close, confirm };
})();
