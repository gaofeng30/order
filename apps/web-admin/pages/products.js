/* 菜品管理 —— 对应 apps/wechat-miniprogram/pages/admin-products + admin-product-edit
   PC 形态：表格 + 勾选批量操作 + 右侧编辑抽屉。
   批量改价 / 批量上下架是 PC 端相对手机端的主要效率增益。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  let cat = '全部';
  let kw = '';
  const picked = new Set();

  const LABEL = { on: '可购', soldout: '售罄', off: '已下架' };
  const TONE = { on: 'ok', soldout: 'mute', off: 'mute' };

  function render(el) {
    picked.clear();
    el.innerHTML =
      `<div class="toolbar">
         <div class="segs" id="cats"></div>
         <span class="sp"></span>
         <div class="search">
           ${I.svg('search', 15)}
           <input class="inp" id="kw" placeholder="搜索菜品名称" value="${T.esc(kw)}">
         </div>
         <button class="btn btn--primary" data-new>${I.svg('plus', 16)}上架新菜</button>
       </div>
       <div id="batch"></div>
       <div id="tbl-host"></div>`;

    el.querySelector('[data-new]').onclick = () => openEdit(el, null);
    const kwEl = el.querySelector('#kw');
    kwEl.oninput = () => { kw = kwEl.value; paint(el, true); };

    paint(el);
  }

  function paint(el, keepFocus) {
    const cats = ['全部'].concat(window.__store.cats.filter(c => c.on).map(c => c.name));
    el.querySelector('#cats').innerHTML = cats.map(c =>
      `<span class="seg${c === cat ? ' on' : ''}" data-cat="${T.esc(c)}">${T.esc(c)}</span>`).join('');
    el.querySelectorAll('[data-cat]').forEach(n => {
      n.onclick = () => { cat = n.dataset.cat; picked.clear(); paint(el); };
    });

    Api.listProducts().then(all => {
      const list = all.filter(m =>
        (cat === '全部' || m.cat === cat) &&
        (!kw || m.name.indexOf(kw) > -1));

      // 过滤后清掉已不可见的勾选，避免批量操作误伤
      Array.from(picked).forEach(id => { if (!list.some(m => m.id === id)) picked.delete(id); });

      paintBatch(el, list);

      const host = el.querySelector('#tbl-host');
      host.innerHTML = T.render({
        cols: [
          { t: `<input type="checkbox" data-act="all" ${list.length && picked.size === list.length ? 'checked' : ''}>`, w: '36px',
            render: r => `<input type="checkbox" data-act="pick" data-id="${r.id}" ${picked.has(r.id) ? 'checked' : ''}>` },
          { t: '菜品', render: r => `
            <div class="row gap12">
              <img class="thumb" src="${Api.imgUrl(r.img)}" alt="" onerror="this.style.visibility='hidden'">
              <div class="grow">
                <div class="pd-nm">${T.esc(r.name)}${(r.imgs || []).length > 1
                  ? `<span class="faint tnum" style="font-weight:400;margin-left:6px">${r.imgs.length} 图</span>` : ''}</div>
                <div class="faint ellipsis" style="font-size:12px;max-width:320px">${T.esc(r.desc || '')}</div>
              </div>
            </div>` },
          { t: '分类', w: '92px', render: r => T.esc(r.cat) },
          { t: '售价', w: '76px', cls: 'num', render: r => T.money(r.price) },
          { t: '库存', w: '72px', cls: 'num', render: r =>
            `<span class="tnum${r.stock > 0 && r.stock <= 8 ? ' low-stock' : ''}">${r.stock}</span>` },
          { t: '销量', w: '64px', cls: 'num', render: r => `<span class="faint tnum">${r.sold}</span>` },
          { t: '状态', w: '82px', render: r => T.pill(LABEL[r.status], TONE[r.status]) },
          { t: '操作', w: '234px', cls: 'act', render: r => `
            <button class="btn btn--sm btn--line" data-act="sold" data-id="${r.id}">${r.status === 'soldout' ? '恢复售卖' : '标记售罄'}</button>
            <button class="btn btn--sm btn--line" data-act="shelf" data-id="${r.id}">${r.status === 'off' ? '上架' : '下架'}</button>
            <button class="btn btn--sm btn--ghost-blue" data-act="edit" data-id="${r.id}">编辑</button>` },
        ],
        rows: list,
        empty: kw ? `没有匹配「${T.esc(kw)}」的菜品` : '该分类下暂无菜品',
      });

      T.bind(host, {
        all(_, node) {
          if (node.checked) list.forEach(m => picked.add(m.id));
          else picked.clear();
          paint(el);
        },
        pick(id, node) {
          if (node.checked) picked.add(id); else picked.delete(id);
          paintBatch(el, list);
          const head = host.querySelector('[data-act="all"]');
          if (head) head.checked = list.length && picked.size === list.length;
        },
        sold(id) {
          const cur = list.find(m => m.id === id);
          const nx = cur.status === 'soldout' ? 'on' : 'soldout';
          Api.setProductStatus(id, nx).then(() => {
            paint(el);
            window.Toast.show(nx === 'soldout' ? '已置售罄' : '已恢复售卖', { icon: 'tag' });
          });
        },
        shelf(id) {
          const cur = list.find(m => m.id === id);
          const nx = cur.status === 'off' ? 'on' : 'off';
          Api.setProductStatus(id, nx).then(() => {
            paint(el);
            window.Toast.show(nx === 'on' ? '已上架' : '已下架', { icon: nx === 'on' ? 'check' : 'box' });
          });
        },
        edit(id) { openEdit(el, list.find(m => m.id === id)); },
      });

      if (keepFocus) {
        const k = el.querySelector('#kw');
        k.focus();
        k.setSelectionRange(k.value.length, k.value.length);
      }
    });
  }

  /* ---------------- 批量操作条 ---------------- */
  function paintBatch(el, list) {
    const host = el.querySelector('#batch');
    if (!picked.size) { host.innerHTML = ''; return; }
    host.innerHTML =
      `<div class="batch-bar">
         <b class="tnum">${picked.size}</b><span>项已选</span>
         <span class="grow"></span>
         <button class="btn btn--sm btn--line" data-b="on">批量上架</button>
         <button class="btn btn--sm btn--line" data-b="off">批量下架</button>
         <button class="btn btn--sm btn--line" data-b="soldout">批量置售罄</button>
         <button class="btn btn--sm btn--ghost-blue" data-b="price">批量改价</button>
         <button class="btn btn--sm btn--line" data-b="clear">取消选择</button>
       </div>`;

    host.querySelectorAll('[data-b]').forEach(n => {
      n.onclick = () => {
        const a = n.dataset.b;
        if (a === 'clear') { picked.clear(); paint(el); return; }
        if (a === 'price') return batchPrice(el, list);
        const ids = Array.from(picked);
        Promise.all(ids.map(id => Api.setProductStatus(id, a))).then(() => {
          window.Toast.show(`已批量${{ on: '上架', off: '下架', soldout: '置售罄' }[a]} ${ids.length} 项`, { icon: 'tag' });
          paint(el);
        });
      };
    });
  }

  function batchPrice(el, list) {
    window.Modal.open({
      title: `批量改价 · ${picked.size} 项`,
      bodyHtml:
        `<div class="fld">
           <div class="fld-lb">调整方式</div>
           <select class="sel" id="bp-mode">
             <option value="set">统一设为固定价</option>
             <option value="pct">按百分比调整</option>
             <option value="delta">加减固定金额</option>
           </select>
         </div>
         <div class="fld">
           <div class="fld-lb" id="bp-lb">目标价格（元）</div>
           <input class="inp tnum" id="bp-val" type="number" step="0.5" placeholder="例如 28">
           <div class="fld-hint" id="bp-hint">将覆盖所选菜品的现有售价。</div>
         </div>`,
      footerHtml:
        `<button class="btn btn--line" data-a="c">取消</button>
         <button class="btn btn--blue" data-a="ok">确认改价</button>`,
      onMount(root, close) {
        const mode = root.querySelector('#bp-mode');
        const lb = root.querySelector('#bp-lb');
        const hint = root.querySelector('#bp-hint');
        mode.onchange = () => {
          const m = mode.value;
          lb.textContent = m === 'set' ? '目标价格（元）' : (m === 'pct' ? '调整百分比（%）' : '加减金额（元）');
          hint.textContent = m === 'set' ? '将覆盖所选菜品的现有售价。'
            : (m === 'pct' ? '正数上调、负数下调，例如 -10 表示统一降价 10%。结果四舍五入到角。'
                           : '正数上调、负数下调，例如 -2 表示每份便宜 2 元。');
        };
        root.querySelector('[data-a="c"]').onclick = close;
        root.querySelector('[data-a="ok"]').onclick = () => {
          const v = Number(root.querySelector('#bp-val').value);
          if (!Number.isFinite(v)) return window.Toast.show('请输入数值', { icon: 'warn' });
          const m = mode.value;
          const ids = Array.from(picked);
          const jobs = ids.map(id => {
            const p = list.find(x => x.id === id);
            let np = m === 'set' ? v : (m === 'pct' ? p.price * (1 + v / 100) : p.price + v);
            np = Math.round(np * 10) / 10;
            if (!(np > 0)) return Promise.reject(new Error(`「${p.name}」调整后价格为 ${np}，必须大于 0`));
            return Api.saveProduct({ id: p.id, name: p.name, price: np, stock: p.stock, cat: p.cat, desc: p.desc, imgs: p.imgs });
          });
          Promise.all(jobs).then(() => {
            close();
            window.Toast.show(`已批量改价 ${ids.length} 项`, { icon: 'check' });
            paint(el);
          }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
        };
      },
    });
  }

  /* ---------------- 编辑抽屉（原 admin-product-edit 页） ---------------- */
  function openEdit(el, p) {
    const isEdit = !!p;
    const cats = window.__store.cats.filter(c => c.on).map(c => c.name);
    let imgs = (p && p.imgs ? p.imgs.slice() : []);

    window.Drawer.open({
      title: isEdit ? '编辑菜品' : '上架新菜',
      bodyHtml:
        `<div class="fld">
           <div class="fld-lb">菜品图片 <span class="faint">最多 3 张，第一张为封面</span></div>
           <div class="img-grid" id="imgs"></div>
           <div class="fld-hint">电脑端可直接拖拽或选择本机图片；小程序端需走相册选择。</div>
         </div>
         <div class="fld">
           <div class="fld-lb">菜品名称 <span class="req">*</span></div>
           <input class="inp" id="f-name" value="${p ? T.esc(p.name) : ''}" placeholder="例如 商务双拼饭">
         </div>
         <div class="fld-row">
           <div class="fld">
             <div class="fld-lb">售价（元）<span class="req">*</span></div>
             <input class="inp tnum" id="f-price" type="number" step="0.5" value="${p ? p.price : ''}">
           </div>
           <div class="fld">
             <div class="fld-lb">库存 <span class="req">*</span></div>
             <input class="inp tnum" id="f-stock" type="number" value="${p ? p.stock : 0}">
           </div>
         </div>
         <div class="fld">
           <div class="fld-lb">分类 <span class="req">*</span></div>
           <select class="sel" id="f-cat">
             <option value="">请选择分类</option>
             ${cats.map(c => `<option value="${T.esc(c)}"${p && p.cat === c ? ' selected' : ''}>${T.esc(c)}</option>`).join('')}
           </select>
         </div>
         <div class="fld">
           <div class="fld-lb">描述</div>
           <textarea class="txa" id="f-desc" placeholder="出餐说明、口感、适用场景">${p ? T.esc(p.desc || '') : ''}</textarea>
         </div>
         <input type="file" id="f-file" accept="image/*" multiple hidden>`,
      footerHtml:
        (isEdit ? '<button class="btn btn--danger" data-a="del">删除菜品</button>' : '') +
        `<span class="grow"></span>
         <button class="btn btn--line" data-a="c">取消</button>
         <button class="btn btn--primary" data-a="ok">保存</button>`,
      onMount(root, close) {
        const file = root.querySelector('#f-file');

        function paintImgs() {
          root.querySelector('#imgs').innerHTML =
            imgs.map((src, i) => `
              <div class="img-cell">
                <img src="${Api.imgUrl(src)}" alt="">
                ${i === 0 ? '<span class="img-cover">封面</span>' : ''}
                <button class="img-del" data-rm="${i}">${I.svg('close', 13)}</button>
              </div>`).join('') +
            (imgs.length < 3 ? `<div class="img-cell add" data-add>${I.svg('plus', 20, '#8f9384')}<span>添加</span></div>` : '');

          const add = root.querySelector('[data-add]');
          if (add) add.onclick = () => file.click();
          root.querySelectorAll('[data-rm]').forEach(n => {
            n.onclick = () => { imgs.splice(Number(n.dataset.rm), 1); paintImgs(); };
          });
        }
        paintImgs();

        file.onchange = () => {
          const list = Array.from(file.files).slice(0, 3 - imgs.length);
          Promise.all(list.map(f => Api.uploadImage(f))).then(urls => {
            imgs = imgs.concat(urls).slice(0, 3);
            file.value = '';
            paintImgs();
          });
        };

        root.querySelector('[data-a="c"]').onclick = close;

        const del = root.querySelector('[data-a="del"]');
        if (del) del.onclick = () => {
          window.Modal.confirm({
            title: '删除菜品',
            body: `确认删除「${T.esc(p.name)}」？删除后不可恢复。`,
            okText: '删除', danger: true,
          }).then(yes => {
            if (!yes) return;
            Api.deleteProduct(p.id).then(() => {
              close();
              paint(el);
              window.Toast.show('已删除', { icon: 'check' });
            }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
          });
        };

        root.querySelector('[data-a="ok"]').onclick = () => {
          Api.saveProduct({
            id: p ? p.id : '',
            name: root.querySelector('#f-name').value,
            price: root.querySelector('#f-price').value,
            stock: root.querySelector('#f-stock').value,
            cat: root.querySelector('#f-cat').value,
            desc: root.querySelector('#f-desc').value,
            imgs,
          }).then(() => {
            close();
            paint(el);
            window.Toast.show(isEdit ? '已保存' : '已上架', { icon: 'check' });
          }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
        };
      },
    });
  }

  window.Pages = window.Pages || {};
  window.Pages['products'] = { sub: '上下架、售罄、价格与库存', render };
})();
