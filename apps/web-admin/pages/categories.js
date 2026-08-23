/* 分类管理：主账号改名、上移/下移、启停与带商品删除保护。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  function render(el) {
    el.innerHTML =
      `<div class="page-head">
         <span class="ph-s grow">关闭「用户端可见」后，该分类及其菜品在用户端点单页不再展示；分类下仍有菜品时无法删除。</span>
         <button class="btn btn--primary" data-new>${I.svg('plus', 16)}新增分类</button>
       </div>
       <div class="tbl-wrap" id="cat-host"></div>`;

    el.querySelector('[data-new]').onclick = () => openForm(el, null);
    paint(el);
  }

  function paint(el) {
    Api.listCategories().then(list => {
      const host = el.querySelector('#cat-host');
      host.innerHTML =
        `<div class="cat-head">
           <span style="width:34px"></span>
           <span class="grow">分类名称</span>
           <span style="width:88px">菜品数</span>
           <span style="width:96px">用户端可见</span>
           <span style="width:178px;text-align:right">操作</span>
         </div>` +
        list.map(c => {
          const count = c.count;
          return `<div class="cat-row" data-id="${c.id}">
            <span class="cat-grip">${I.svg('sort', 16, '#b6b9a6')}</span>
            <span class="grow cat-nm">${T.esc(c.name)}</span>
            <span style="width:88px" class="faint tnum">${count} 个</span>
            <span style="width:96px"><button class="sw${c.on ? ' on' : ''}" data-on="${c.id}"></button></span>
            <span style="width:178px;text-align:right">
              <button class="ibtn" data-act="cat-up" data-id="${c.id}" title="上移">↑</button>
              <button class="ibtn" data-act="cat-down" data-id="${c.id}" title="下移">↓</button>
              <button class="btn btn--sm btn--ghost-blue" data-act="cat-edit" data-id="${c.id}">编辑</button>
              <button class="ibtn danger" data-del="${c.id}">${I.svg('trash', 16)}</button>
            </span>
          </div>`;
        }).join('') +
        (list.length ? '' : '<div class="tbl-empty">还没有分类</div>');

      host.querySelectorAll('[data-on]').forEach(n => {
        n.onclick = () => {
          const id = n.dataset.on;
          const cur = list.find(c => c.id === id);
          Api.setCategoryEnabled(id, !cur.on).then(c => {
            paint(el);
            window.Toast.show(c.on ? `「${c.name}」已对用户端开放` : `「${c.name}」已隐藏`, { icon: c.on ? 'check' : 'box' });
          });
        };
      });

      host.querySelectorAll('[data-del]').forEach(n => {
        n.onclick = () => {
          const c = list.find(x => x.id === n.dataset.del);
          window.Modal.confirm({
            title: '删除分类',
            body: `确认删除「${T.esc(c.name)}」？分类下仍有菜品时无法删除。`,
            okText: '删除', danger: true,
          }).then(yes => {
            if (!yes) return;
            Api.deleteCategory(c.id)
              .then(() => { paint(el); window.Toast.show('已删除', { icon: 'check' }); })
              .catch(e => window.Toast.show(e.message, { icon: 'warn' }));
          });
        };
      });

      host.querySelectorAll('[data-act="cat-edit"]').forEach(n => {
        n.onclick = () => openForm(el, list.find(c => c.id === n.dataset.id));
      });
      host.querySelectorAll('[data-act="cat-up"], [data-act="cat-down"]').forEach(n => {
        n.onclick = () => moveCategory(el, list, n.dataset.id, n.dataset.act === 'cat-up' ? -1 : 1);
      });
    });
  }

  function moveCategory(el, list, id, delta) {
    const index = list.findIndex(c => c.id === id);
    const target = index + delta;
    if (index < 0 || target < 0 || target >= list.length) return;
    const ids = list.map(c => c.id);
    [ids[index], ids[target]] = [ids[target], ids[index]];
    Api.reorderCategories(ids).then(() => {
      paint(el);
      window.Toast.show('顺序已保存', { icon: 'sort' });
    }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
  }

  function openForm(el, category) {
    const editing = !!category;
    window.Modal.open({
      title: editing ? '编辑分类' : '新增分类',
      bodyHtml:
        `<div class="fld">
           <div class="fld-lb">分类名称 <span class="req">*</span></div>
           <input class="inp" id="c-name" value="${editing ? T.esc(category.name) : ''}" placeholder="例如 节庆礼盒" autocomplete="off">
           <div class="fld-hint">${editing ? '改名后菜品编辑页同步读取新分类名。' : '新增分类默认对用户端开放，可随时关闭。'}</div>
         </div>`,
      footerHtml:
        `<button class="btn btn--line" data-a="c">取消</button>
         <button class="btn btn--blue" data-a="ok">${editing ? '保存' : '确认新增'}</button>`,
      onMount(root, close) {
        const inp = root.querySelector('#c-name');
        inp.focus();
        const submit = () => {
          const command = editing ? Api.renameCategory(category.id, inp.value) : Api.addCategory(inp.value);
          command.then(c => {
            close(); paint(el);
            window.Toast.show(editing ? `已保存「${c.name}」` : `已新增「${c.name}」`, { icon: 'check' });
          }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
        };
        inp.onkeydown = e => { if (e.key === 'Enter') submit(); };
        root.querySelector('[data-a="c"]').onclick = close;
        root.querySelector('[data-a="ok"]').onclick = submit;
      },
    });
  }

  window.Pages = window.Pages || {};
  window.Pages['categories'] = { sub: '使用上移、下移调整用户端的分类顺序', render };
})();
