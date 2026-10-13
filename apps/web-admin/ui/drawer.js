/* Drawer：编辑状态与关闭动作属于当前实例。 */
(function () {
  let active = null;
  function values(box) {
    return JSON.stringify(Array.from(box.querySelectorAll('input:not([type=file]),select,textarea')).map(n => [n.id, n.type === 'checkbox' ? n.checked : n.value]));
  }
  function open(cfg) {
    close(true);
    const host = document.getElementById('overlay-host');
    const mask = document.createElement('div'); mask.className = 'mask drawer-mask';
    const box = document.createElement('div'); box.className = 'drawer' + (cfg.wide ? ' wide' : '');
    box.setAttribute('role', 'dialog'); box.setAttribute('aria-label', cfg.title || '编辑');
    box.innerHTML = `<div class="dr-head"><span class="dr-t">${window.Toast.esc(cfg.title || '')}</span>${cfg.tag ? `<span class="tag-p2">${window.Toast.esc(cfg.tag)}</span>` : ''}<span class="grow"></span><button class="ibtn" data-a="x" aria-label="关闭">${window.Icon.svg('close', 18)}</button></div><div class="dr-body">${cfg.bodyHtml || ''}</div>${cfg.footerHtml ? `<div class="dr-foot">${cfg.footerHtml}</div>` : ''}`;
    host.append(mask, box);
    let baseline = '', asking = null, closed = false;
    const dirty = () => cfg.guardChanges && (values(box) !== baseline || (cfg.isDirty && cfg.isDirty()));
    const canClose = () => {
      if (closed || !dirty()) return Promise.resolve(true);
      if (!asking) asking = window.Modal.confirm({title:'退出编辑',body:'有未保存的修改，确定退出？',cancelText:'继续编辑',okText:'放弃修改'}).finally(() => { asking = null; });
      return asking;
    };
    const destroy = () => { if (closed) return; closed = true; mask.remove(); box.remove(); if (active === instance) active = null; };
    const done = async saved => { if (saved === true || await canClose()) destroy(); };
    const instance = { destroy, canClose, done };
    active = instance; mask.onclick = () => done(); box.querySelector('[data-a="x"]').onclick = () => done();
    if (cfg.onMount) cfg.onMount(box, done);
    baseline = values(box);
    return box;
  }
  function close(force) { if (active) { if (force === true) active.destroy(); else return active.done(); } }
  window.Drawer = { open, close, isOpen: () => !!active, canClose: () => active ? active.canClose() : Promise.resolve(true) };
})();
