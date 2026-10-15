/* 分类管理：主账号改名、拖拽排序、上移/下移、启停与带商品删除保护。 */
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
    return Api.listCategories().then(list => {
      const host = el.querySelector('#cat-host');
      host.innerHTML =
        `<div class="cat-head">

           <span class="cat-drag-slot" aria-hidden="true"></span>
           <span class="grow">分类名称</span>
           <span style="width:88px">菜品数</span>
           <span style="width:96px">用户端可见</span>
           <span style="width:178px;text-align:right">操作</span>
         </div>` +
        list.map((c,index) => {
          const count = c.count;
          return `<div class="cat-row" data-id="${c.id}">

            <button class="ibtn cat-drag-slot cat-drag-handle" draggable="true" aria-label="拖动调整分类顺序" title="拖动调整顺序" ${list.length < 2 ? 'disabled' : ''}>⠿</button>
            <span class="grow cat-nm">${T.esc(c.name)}</span>
            <span style="width:88px" class="faint tnum">${count} 个</span>
            <span style="width:96px"><button class="sw${c.on ? ' on' : ''}" data-on="${c.id}"></button></span>
            <span style="width:178px;text-align:right">
              <button class="ibtn" data-act="cat-up" data-id="${c.id}" title="上移" ${index === 0 ? 'disabled' : ''}>↑</button>
              <button class="ibtn" data-act="cat-down" data-id="${c.id}" title="下移" ${index === list.length-1 ? 'disabled' : ''}>↓</button>
              <button class="btn btn--sm btn--ghost-blue" data-act="cat-edit" data-id="${c.id}">编辑</button>
              <button class="ibtn danger" data-del="${c.id}" ${count > 0 ? 'disabled title="请先移走或删除分类内菜品"' : 'title="删除分类"'}>${I.svg('trash', 16)}</button>${count > 0 ? '<span class="cat-delete-note">请先移出或删除该分类下的菜品</span>' : ''}
            </span>
          </div>`;
        }).join('') +
        (list.length ? '' : '<div class="tbl-empty">还没有分类</div>');

      bindDrag(el, host, list);

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
    saveOrder(el, ids);
  }

  async function saveOrder(el, ids) {
    const host = el.querySelector('#cat-host');
    if (host.dataset.saving) return;
    host.dataset.saving = 'true';
    host.setAttribute('aria-busy', 'true');
    const controls = [...el.querySelectorAll('button')].map(n => [n, n.disabled]);
    controls.forEach(([n]) => { n.disabled = true; });
    try {
      await Api.reorderCategories(ids);
      await paint(el);
      window.Toast.show('顺序已保存', { icon: 'sort' });
    } catch (e) {
      window.Toast.show(e.message, { icon: 'warn' });
    } finally {
      delete host.dataset.saving;
      host.removeAttribute('aria-busy');
      controls.forEach(([n, disabled]) => { n.disabled = disabled; });
    }
  }

  function bindDrag(el, host, list) {
    let source = null;
    const rows = [...host.querySelectorAll('.cat-row')];
    const clearMarks = () => rows.forEach(row => row.classList.remove('drop-before', 'drop-after'));
    const reset = () => {
      source = null;
      clearMarks();
      rows.forEach(row => row.classList.remove('dragging'));
    };
    rows.forEach(row => {
      const handle = row.querySelector('.cat-drag-handle');
      handle.ondragstart = e => {
        if (host.dataset.saving || handle.disabled) { e.preventDefault(); return; }
        source = row.dataset.id;
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/plain', source);
        e.dataTransfer.setDragImage(row, 20, row.offsetHeight / 2);
        row.classList.add('dragging');
      };
      handle.ondragend = reset;
      row.ondragover = e => {
        if (!source || host.dataset.saving) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        clearMarks();
        if (source === row.dataset.id) return;
        const rect = row.getBoundingClientRect();
        row.classList.add(e.clientY < rect.top + rect.height / 2 ? 'drop-before' : 'drop-after');
      };
      row.ondrop = e => {
        if (!source || host.dataset.saving) return;
        e.preventDefault();
        const id = source;
        const target = row.dataset.id;
        const rect = row.getBoundingClientRect();
        const after = e.clientY >= rect.top + rect.height / 2;
        reset();
        if (id === target) return;
        const ids = list.map(c => c.id).filter(x => x !== id);
        ids.splice(ids.indexOf(target) + (after ? 1 : 0), 0, id);
        if (ids.every((x, i) => x === list[i].id)) return;
        saveOrder(el, ids);
      };
    });
    host.ondragleave = e => {
      if (!host.contains(e.relatedTarget)) clearMarks();
    };
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
        const submit = window.FormUI.submit(root.querySelector('[data-a="ok"]'), async () => {
          const command = editing ? Api.renameCategory(category.id, inp.value) : Api.addCategory(inp.value);
          await command.then(c => {
            close(); paint(el);
            window.Toast.show(editing ? `已保存「${c.name}」` : `已新增「${c.name}」`, { icon: 'check' });
          }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
        });
        inp.onkeydown = e => { if (e.key === 'Enter') submit(); };
        root.querySelector('[data-a="c"]').onclick = close;
        root.querySelector('[data-a="ok"]').onclick = submit;
      },
    });
  }

  window.Pages = window.Pages || {};
  window.Pages['categories'] = { sub: '拖动左侧手柄或使用上移、下移调整分类顺序', render };
})();
