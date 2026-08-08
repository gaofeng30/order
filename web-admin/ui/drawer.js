/* Drawer —— 右侧抽屉。
   PC 上「列表 + 编辑」是一体的：小程序里的 admin-product-edit /
   admin-member-edit / admin-coupon-edit 三个独立页在这里都是抽屉。 */
(function () {
  const host = () => document.getElementById('overlay-host');
  let closer = null;

  // open({ title, tag, bodyHtml, footerHtml, wide, onMount(root, close) })
  function open(cfg) {
    close();
    const mask = document.createElement('div');
    mask.className = 'mask';
    const box = document.createElement('div');
    box.className = 'drawer' + (cfg.wide ? ' wide' : '');
    box.innerHTML =
      `<div class="dr-head">
         <span class="dr-t">${cfg.title || ''}</span>
         ${cfg.tag ? `<span class="tag-p2">${cfg.tag}</span>` : ''}
         <span class="grow"></span>
         <button class="ibtn" data-a="x">${window.Icon.svg('close', 18)}</button>
       </div>
       <div class="dr-body">${cfg.bodyHtml || ''}</div>
       ${cfg.footerHtml ? `<div class="dr-foot">${cfg.footerHtml}</div>` : ''}`;

    host().appendChild(mask);
    host().appendChild(box);
    closer = () => { mask.remove(); box.remove(); closer = null; };
    mask.addEventListener('click', close);
    box.querySelector('[data-a="x"]').addEventListener('click', close);
    if (cfg.onMount) cfg.onMount(box, close);
    return box;
  }

  function close() { if (closer) closer(); }
  function isOpen() { return !!closer; }

  window.Drawer = { open, close, isOpen };
})();
