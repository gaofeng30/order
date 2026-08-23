/* 分类管理 —— 对应 apps/wechat-miniprogram/pages/admin-categories
   PC 形态：拖拽排序（手机端只能上下移按钮）+ 启停开关 + 删除保护。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  let dragId = '';

  function render(el) {
    el.innerHTML =
      `<div class="page-head">
         <span class="ph-s grow">关闭「用户端可见」后，该分类及其菜品在用户端点单页不再展示；分类下仍有菜品时无法删除。</span>
         <button class="btn btn--primary" data-new>${I.svg('plus', 16)}新增分类</button>
       </div>
       <div class="tbl-wrap" id="cat-host"></div>`;

    el.querySelector('[data-new]').onclick = () => openAdd(el);
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
           <span style="width:72px;text-align:right">操作</span>
         </div>` +
        list.map(c => {
          const count = c.count;
          return `<div class="cat-row" draggable="true" data-id="${c.id}">
            <span class="cat-grip">${I.svg('sort', 16, '#b6b9a6')}</span>
            <span class="grow cat-nm">${T.esc(c.name)}</span>
            <span style="width:88px" class="faint tnum">${count} 个</span>
            <span style="width:96px"><button class="sw${c.on ? ' on' : ''}" data-on="${c.id}"></button></span>
            <span style="width:72px;text-align:right">
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

      bindDrag(el, host);
    });
  }

  // 拖拽排序：dragover 时实时插入占位，drop 时按 DOM 顺序提交
  function bindDrag(el, host) {
    host.querySelectorAll('.cat-row').forEach(row => {
      row.addEventListener('dragstart', e => {
        dragId = row.dataset.id;
        row.classList.add('dragging');
        e.dataTransfer.effectAllowed = 'move';
        // Firefox 需要写入数据才会触发 drop
        e.dataTransfer.setData('text/plain', dragId);
      });
      row.addEventListener('dragend', () => {
        row.classList.remove('dragging');
        const ids = Array.from(host.querySelectorAll('.cat-row')).map(n => n.dataset.id);
        Api.reorderCategories(ids).then(() => {
          paint(el);
          window.Toast.show('顺序已保存', { icon: 'sort' });
        });
      });
      row.addEventListener('dragover', e => {
        e.preventDefault();
        const dragging = host.querySelector('.dragging');
        if (!dragging || dragging === row) return;
        const r = row.getBoundingClientRect();
        const after = (e.clientY - r.top) > r.height / 2;
        row.parentNode.insertBefore(dragging, after ? row.nextSibling : row);
      });
    });
  }

  function openAdd(el) {
    window.Modal.open({
      title: '新增分类',
      bodyHtml:
        `<div class="fld">
           <div class="fld-lb">分类名称 <span class="req">*</span></div>
           <input class="inp" id="c-name" placeholder="例如 节庆礼盒" autocomplete="off">
           <div class="fld-hint">新增分类默认对用户端开放，可随时关闭。</div>
         </div>`,
      footerHtml:
        `<button class="btn btn--line" data-a="c">取消</button>
         <button class="btn btn--blue" data-a="ok">确认新增</button>`,
      onMount(root, close) {
        const inp = root.querySelector('#c-name');
        inp.focus();
        const submit = () => {
          Api.addCategory(inp.value).then(c => {
            close(); paint(el);
            window.Toast.show(`已新增「${c.name}」`, { icon: 'check' });
          }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
        };
        inp.onkeydown = e => { if (e.key === 'Enter') submit(); };
        root.querySelector('[data-a="c"]').onclick = close;
        root.querySelector('[data-a="ok"]').onclick = submit;
      },
    });
  }

  window.Pages = window.Pages || {};
  window.Pages['categories'] = { sub: '拖动左侧手柄调整用户端的分类顺序', render };
})();
