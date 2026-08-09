/* 会员等级 —— 对应 apps/wechat-miniprogram/pages/admin-levels（二期能力，不在一期合同范围）
   删除等级前必须预览影响面并选择迁移目标：该档会员整体迁移，
   券的适用等级中摘除该档，摘空则该券自动停用（见 data/api.js deleteLevel）。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  function render(el) {
    el.innerHTML =
      `<div class="page-head">
         <span class="ph-s grow">等级按顺序从低到高排列，一人一档。折扣为常驻折扣，与优惠券叠加使用。拖动手柄可调整顺序。</span>
         <button class="btn btn--primary" data-new>${I.svg('plus', 16)}新增等级</button>
       </div>
       <div class="tbl-wrap" id="lv-host"></div>`;

    el.querySelector('[data-new]').onclick = () => openEdit(el, null);
    paint(el);
  }

  function paint(el) {
    Promise.all([Api.listLevels(), Api.listMembers({})]).then(([list, mem]) => {
      const host = el.querySelector('#lv-host');
      host.innerHTML =
        `<div class="cat-head">
           <span style="width:34px"></span>
           <span style="width:56px">档位</span>
           <span class="grow">等级名称</span>
           <span style="width:96px">常驻折扣</span>
           <span style="width:88px">会员数</span>
           <span style="width:104px;text-align:right">操作</span>
         </div>` +
        list.map(lv => {
          const n = mem.list.filter(m => m.levelId === lv.id).length;
          return `<div class="cat-row" draggable="true" data-id="${lv.id}">
            <span class="cat-grip">${I.svg('sort', 16, '#b6b9a6')}</span>
            <span style="width:56px" class="faint tnum">L${lv.sort}</span>
            <span class="grow">
              <div class="cat-nm">${T.esc(lv.name)}</div>
              <div class="faint" style="font-size:12px">${T.esc(lv.desc || '—')}</div>
            </span>
            <span style="width:96px"><b class="price-blue tnum">${(lv.discount / 10).toFixed(1)} 折</b></span>
            <span style="width:88px" class="faint tnum">${n} 人</span>
            <span style="width:104px;text-align:right">
              <button class="btn btn--sm btn--ghost-blue" data-edit="${lv.id}">编辑</button>
              <button class="ibtn danger" data-del="${lv.id}">${I.svg('trash', 16)}</button>
            </span>
          </div>`;
        }).join('');

      host.querySelectorAll('[data-edit]').forEach(n =>
        n.onclick = () => openEdit(el, list.find(x => x.id === n.dataset.edit)));
      host.querySelectorAll('[data-del]').forEach(n =>
        n.onclick = () => askDelete(el, list, list.find(x => x.id === n.dataset.del)));

      bindDrag(el, host);
    });
  }

  function bindDrag(el, host) {
    host.querySelectorAll('.cat-row').forEach(row => {
      row.addEventListener('dragstart', e => {
        row.classList.add('dragging');
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/plain', row.dataset.id);
      });
      row.addEventListener('dragend', () => {
        row.classList.remove('dragging');
        const ids = Array.from(host.querySelectorAll('.cat-row')).map(n => n.dataset.id);
        Api.reorderLevels(ids).then(() => { paint(el); window.Toast.show('档位顺序已保存', { icon: 'sort' }); });
      });
      row.addEventListener('dragover', e => {
        e.preventDefault();
        const dragging = host.querySelector('.dragging');
        if (!dragging || dragging === row) return;
        const r = row.getBoundingClientRect();
        row.parentNode.insertBefore(dragging, (e.clientY - r.top) > r.height / 2 ? row.nextSibling : row);
      });
    });
  }

  /* ---------------- 新增 / 编辑 ---------------- */
  function openEdit(el, lv) {
    const isEdit = !!lv;
    window.Drawer.open({
      title: isEdit ? '编辑等级' : '新增等级',
      tag: '二期',
      bodyHtml:
        `<div class="fld">
           <div class="fld-lb">等级名称 <span class="req">*</span></div>
           <input class="inp" id="f-name" value="${lv ? T.esc(lv.name) : ''}" placeholder="例如 中级会员">
         </div>
         <div class="fld">
           <div class="fld-lb">常驻折扣 <span class="req">*</span></div>
           <div class="row gap12">
             <input class="inp tnum" id="f-dis" type="number" min="1" max="100" value="${lv ? lv.discount : 95}" style="width:120px">
             <span class="faint" id="f-dis-t">＝ ${lv ? (lv.discount / 10).toFixed(1) : '9.5'} 折</span>
           </div>
           <div class="fld-hint">填百分比：100 = 无折扣，85 = 打 8.5 折。该折扣对本档会员长期生效。</div>
         </div>
         <div class="fld">
           <div class="fld-lb">说明</div>
           <textarea class="txa" id="f-desc" placeholder="这一档面向哪些人">${lv ? T.esc(lv.desc || '') : ''}</textarea>
         </div>`,
      footerHtml:
        `<span class="grow"></span>
         <button class="btn btn--line" data-a="c">取消</button>
         <button class="btn btn--primary" data-a="ok">保存</button>`,
      onMount(root, close) {
        const dis = root.querySelector('#f-dis');
        dis.oninput = () => {
          const v = Number(dis.value);
          root.querySelector('#f-dis-t').textContent = v > 0 ? `＝ ${(v / 10).toFixed(1)} 折` : '';
        };
        root.querySelector('[data-a="c"]').onclick = close;
        root.querySelector('[data-a="ok"]').onclick = () => {
          Api.saveLevel({
            id: lv ? lv.id : '',
            name: root.querySelector('#f-name').value,
            discount: dis.value,
            desc: root.querySelector('#f-desc').value,
          }).then(() => {
            close(); paint(el);
            window.Toast.show(isEdit ? '已保存' : '已新增等级', { icon: 'check' });
          }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
        };
      },
    });
  }

  /* ---------------- 删除：先看影响面，再选迁移目标 ---------------- */
  function askDelete(el, list, lv) {
    if (list.length <= 1) return window.Toast.show('至少保留一个等级', { icon: 'warn' });

    Api.levelImpact(lv.id).then(im => {
      const others = list.filter(x => x.id !== lv.id);
      window.Modal.open({
        title: `删除「${T.esc(lv.name)}」`,
        width: '460px',
        bodyHtml:
          `<div class="imp-box">
             <div class="imp-row"><b class="tnum">${im.memberCount}</b> 名会员在这一档</div>
             <div class="imp-row"><b class="tnum">${im.coupons.length}</b> 张券把这一档列为适用范围
               ${im.coupons.length ? `<div class="faint" style="font-size:12px;margin-top:4px">${im.coupons.map(c => T.esc(c.name)).join('、')}</div>` : ''}
             </div>
           </div>
           <div class="fld" style="margin-top:14px;margin-bottom:0">
             <div class="fld-lb">这 ${im.memberCount} 名会员迁移到 <span class="req">*</span></div>
             <select class="sel" id="mg">
               ${others.map(o => `<option value="${o.id}">${T.esc(o.name)}（${(o.discount / 10).toFixed(1)} 折）</option>`).join('')}
               <option value="none">降为非会员（移出名单）</option>
             </select>
             <div class="fld-hint">券的适用等级中会自动摘除这一档；若某张券因此没有任何适用等级，该券将自动停用。</div>
           </div>`,
        footerHtml:
          `<button class="btn btn--line" data-a="c">取消</button>
           <button class="btn btn--danger" data-a="ok">确认删除</button>`,
        onMount(root, close) {
          root.querySelector('[data-a="c"]').onclick = close;
          root.querySelector('[data-a="ok"]').onclick = () => {
            Api.deleteLevel(lv.id, root.querySelector('#mg').value).then(r => {
              close(); paint(el);
              window.Toast.show(r.disabledCoupons
                ? `已删除 · ${r.disabledCoupons} 张券因适用等级摘空已停用`
                : '已删除', { icon: 'check' });
            }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
          };
        },
      });
    });
  }

  window.Pages = window.Pages || {};
  window.Pages['levels'] = { render };
})();
