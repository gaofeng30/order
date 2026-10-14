/* 菜品管理 —— 对应 apps/wechat-miniprogram/pages/admin-products + admin-product-edit
   PC 形态：表格 + 勾选批量操作 + 右侧编辑抽屉。
   批量改价 / 批量上下架是 PC 端相对手机端的主要效率增益。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  let cat = '全部';
  let kw = '';
  let categoryCache = [];
  const picked = new Set();
  let generation = 0, bulkBusy = false, productRead = null;
  const pending = new Set();
  async function change(el,id,action) {
    if(pending.has(id))return;pending.add(id);
    el.querySelectorAll('tr[data-id]').forEach(row=>{if(row.dataset.id===id)row.querySelectorAll('button').forEach(b=>b.disabled=true);});
    try {await action();} catch(e){window.Toast.show(e.message,{icon:'warn'});} finally {pending.delete(id);if(el.isConnected)paint(el);}
  }

  /* 下架优先决定最终可售展示；重新上架沿用当天的售罄记录。 */
  function render(el) {
    picked.clear();
    productRead = null;
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
       <div class="fld-hint sold-note">下架菜品不可售；上架后按今日售罄状态判断。售罄仅影响今天，不影响明日预约。</div><div id="batch"></div>
       <div id="tbl-host"></div>`;

    el.querySelector('[data-new]').onclick = () => openEdit(el, null);
    const kwEl = el.querySelector('#kw');
    kwEl.oninput = () => { kw = kwEl.value; paint(el, true, true); };

    Api.listCategories().then(list => {
      categoryCache = list;
      paint(el);
    }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
  }

  function paint(el, keepFocus, reuseProducts = false) {
    const cats = ['全部'].concat(categoryCache.filter(c => c.on).map(c => c.name));
    el.querySelector('#cats').innerHTML = cats.map(c =>
      `<span class="seg${c === cat ? ' on' : ''}" data-cat="${T.esc(c)}">${T.esc(c)}</span>`).join('');
    el.querySelectorAll('[data-cat]').forEach(n => {
      n.onclick = () => { cat = n.dataset.cat; picked.clear(); paint(el, false, true); };
    });

    const version=++generation;
    if (!reuseProducts || !productRead) productRead = Api.listProducts();
    productRead.then(all => {
      if(version!==generation || !el.isConnected)return;
      const list = all.filter(m =>
        (cat === '全部' || m.cat === cat) &&
        (!kw || m.name.indexOf(kw) > -1));

      // 过滤后清掉已不可见的勾选，避免批量操作误伤
      Array.from(picked).forEach(id => { if (!list.some(m => m.id === id)) picked.delete(id); });

      paintBatch(el, list);

      const host = el.querySelector('#tbl-host');
      host.innerHTML = T.render({
        cols: [
          { t: `<label class="select-all"><input type="checkbox" data-act="all" ${list.length && picked.size === list.length ? 'checked' : ''}><span>全选</span></label><span class="selection-count" data-selection-count>${picked.size ? '已选 '+picked.size+' 项' : ''}</span>`, w: '80px',
            render: r => `<input type="checkbox" data-act="pick" data-id="${r.id}" ${picked.has(r.id) ? 'checked' : ''}>` },
          { t: '菜品', w: '360px', cls: 'product-info', render: r => `
            <div class="row gap12">
              <span class="thumb-frame"><span class="image-placeholder">暂无图片</span>${r.img ? `<img class="thumb" src="${T.esc(Api.imgUrl(r.img))}" alt="" onerror="this.remove()">` : ''}</span>
              <div class="grow">
                <div class="pd-nm">${T.esc(r.name)}${(r.imgs || []).length > 1
                  ? `<span class="faint tnum" style="font-weight:400;margin-left:6px">${r.imgs.length} 图</span>` : ''}</div>
                <div class="faint ellipsis" style="font-size: 12px;max-width:320px">${T.esc(r.desc || '')}</div>
              </div>
            </div>` },
          { t: '分类', w: '92px', render: r => T.esc(r.cat) },
          { t: '餐段', w: '68px', render: r => T.esc(Api.MEAL_LABEL[r.meal] || '—') },
          { t: '售价', w: '76px', cls: 'num', render: r => T.money(r.price) },
          { t: '上架状态', w: '90px', render: r => T.pill(r.status === 'off' ? '已下架' : '已上架',r.status === 'off'?'mute':'ok') },
          { t: '今日售卖', w: '115px', render: r => {
            const off = r.status === 'off';
            const sold = Api.isSoldOut(r.id, Api.today());
            const saleLabel = off ? '不可售' : sold ? '售罄' : '可售';
            const saleClass = off ? 'muted' : sold ? 'text-danger' : 'text-green';
            const switchClass = off ? '' : sold ? 'sold' : 'on';
            const actionLabel = off ? '已下架，不可售' : sold ? '恢复今日售卖' : '标记今日售罄';
            return `<div class="row gap8"><button class="sw ${switchClass}" data-act="sold" data-id="${r.id}" aria-label="${actionLabel}" aria-pressed="${!off && !sold}" ${off || pending.has(r.id) ? 'disabled' : ''}></button><span class="${saleClass}">${saleLabel}</span></div>`;
          } },
          { t: '操作', cls: 'act', render: r => `
            <button class="ibtn" data-act="up" data-id="${r.id}" title="分类内上移" aria-label="分类内上移" ${all.filter(x=>x.categoryId===r.categoryId)[0].id===r.id?'disabled':''}>↑</button>
            <button class="ibtn" data-act="down" data-id="${r.id}" title="分类内下移" aria-label="分类内下移" ${all.filter(x=>x.categoryId===r.categoryId).slice(-1)[0].id===r.id?'disabled':''}>↓</button>
            <button class="btn btn--sm btn--line" data-act="shelf" data-id="${r.id}">${r.status === 'off' ? '上架' : '下架'}</button>
            <button class="btn btn--sm btn--ghost-blue" data-act="edit" data-id="${r.id}">编辑</button>` },
        ],
        rows: list,
        empty: kw ? `没有匹配「${T.esc(kw)}」的菜品` : '该分类下暂无菜品',
      });

      const allBox=host.querySelector('[data-act="all"]');
      if(allBox)allBox.indeterminate=picked.size>0&&picked.size<list.length;
      T.bind(host, {
        all(_, node) {
          if (node.checked) list.forEach(m => picked.add(m.id));
          else picked.clear();
          host.querySelectorAll('[data-act="pick"]').forEach(input => {input.checked=node.checked;});
          node.indeterminate=false;
          host.querySelector('[data-selection-count]').textContent=picked.size?'已选 '+picked.size+' 项':'';
          paintBatch(el,list);
        },
        pick(id, node) {
          if (node.checked) picked.add(id); else picked.delete(id);
          paintBatch(el, list);
          const head = host.querySelector('[data-act="all"]');
          if (head) {head.checked = list.length>0 && picked.size === list.length;head.indeterminate=picked.size>0&&picked.size<list.length;}
          host.querySelector('[data-selection-count]').textContent=picked.size?'已选 '+picked.size+' 项':'';
        },
        /* 售罄只写当前营业日的记录，不动 status。D 日售罄不影响 D+1 的预约。 */
        sold(id) {
          const current = list.find(product => product.id === id);
          if (!current || current.status === 'off') return;
          const nx = !Api.isSoldOut(id, Api.today());
          change(el,id,()=>Api.setSoldOut(id, nx).then(() => {
            window.Toast.show(nx ? `已置售罄 · 仅限 ${Api.today()}` : '已恢复今日售卖', { icon: 'tag' });
          }));
        },
        shelf(id) {
          const cur = list.find(m => m.id === id);
          const nx = cur.status === 'off' ? 'on' : 'off';
          change(el,id,()=>Api.setProductStatus(id, nx).then(() => {
            window.Toast.show(nx === 'on' ? '已上架' : '已下架', { icon: nx === 'on' ? 'check' : 'box' });
          }));
        },
        edit(id) { openEdit(el, list.find(m => m.id === id)); },
        up(id) { moveProduct(el, all, id, -1); },
        down(id) { moveProduct(el, all, id, 1); },
      });

      if (keepFocus) {
        const k = el.querySelector('#kw');
        k.focus();
        k.setSelectionRange(k.value.length, k.value.length);
      }
    }).catch(e => {
      if (version !== generation || !el.isConnected) return;
      productRead = null;
      window.Toast.show(e.message || '菜品加载失败，请重试', { icon: 'warn' });
    });
  }

  function moveProduct(el, all, id, delta) {
    const current = all.find(p => p.id === id);
    if (!current) return;
    const siblings = all.filter(p => p.categoryId === current.categoryId);
    const index = siblings.findIndex(p => p.id === id);
    const target = index + delta;
    if (target < 0 || target >= siblings.length) return;
    [siblings[index], siblings[target]] = [siblings[target], siblings[index]];
    Api.reorderProducts(current.categoryId, siblings.map(p => p.id)).then(() => {
      paint(el);
      window.Toast.show('菜品顺序已保存', { icon: 'sort' });
    }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
  }

  /* ---------------- 批量操作条 ---------------- */
  function paintBatch(el, list) {
    const host=el.querySelector('#batch');
    if(!picked.size){host.innerHTML='';return;}
    host.innerHTML=`<div class="batch-bar"><span class="grow">批量操作</span><button class="btn btn--sm btn--line" data-b="on">批量上架</button><button class="btn btn--sm btn--line" data-b="off">批量下架</button><button class="btn btn--sm btn--line" data-b="sold">批量今日售罄</button><button class="btn btn--sm btn--line" data-b="price">批量改价</button><button class="btn btn--sm btn--danger" data-b="delete">批量删除</button></div>`;
    host.querySelectorAll('[data-b]').forEach(button=>button.disabled=bulkBusy);
    host.querySelectorAll('[data-b]').forEach(button=>button.onclick=async()=>{
      if(bulkBusy)return;
      const kind=button.dataset.b;if(kind==='price'){batchPrice(el,list);return;}
      const selected=list.filter(p=>picked.has(p.id)),action={on:'上架',off:'下架',sold:'标记今日售罄',delete:'删除'}[kind];
      if(!selected.length)return;
      bulkBusy=true;
      try {
        const yes=await window.Modal.confirm({title:`批量${action}`,body:`<p>确认${action}以下 ${selected.length} 道菜品？</p><ul class="confirm-items">${selected.map(p=>`<li>${T.esc(p.name)}</li>`).join('')}</ul>${kind==='sold'?'<p>仅影响今天，不影响明日预约。</p>':''}${kind==='delete'?'<p class="text-danger">删除后不可恢复，历史订单中的菜品与价格快照不变。</p>':''}`,okText:kind==='delete'?`确认删除 ${selected.length} 道菜品`:'确定',danger:kind==='delete'});
        if(!yes || !el.isConnected)return;
        host.querySelectorAll('button').forEach(b=>b.disabled=true);
        const results=await Promise.allSettled(selected.map(p=>kind==='delete'?Api.deleteProduct(p.id):kind==='sold'?Api.setSoldOut(p.id,true):Api.setProductStatus(p.id,kind)));
        if(kind==='delete') {
          const failures=[];
          results.forEach((result,index)=>{
            const product=selected[index];
            if(result.status==='fulfilled')picked.delete(product.id);
            else failures.push({name:product.name,message:result.reason && result.reason.message || '删除失败，请稍后核对'});
          });
          if(failures.length && el.isConnected) {
            window.Modal.open({
              title:failures.length===selected.length?'菜品删除失败':'部分菜品未能删除',
              bodyHtml:`<p>${failures.length<selected.length?'成功删除的菜品已移除。':''}以下菜品删除失败，请核对原因与当前状态：</p><ul class="confirm-items">${failures.map(p=>`<li>${T.esc(p.name)}：${T.esc(p.message)}</li>`).join('')}</ul>`,
              footerHtml:'<button class="btn btn--line" data-close>关闭</button>',
              onMount(root,close){root.querySelector('[data-close]').onclick=close;},
            });
          } else if(failures.length)window.Toast.show('部分菜品删除失败，请返回菜品管理核对',{icon:'warn'});
          else window.Toast.show(`已删除 ${selected.length} 道菜品`);
        } else {
          const failed=results.find(r=>r.status==='rejected');
          if(failed)window.Toast.show(failed.reason.message||'部分操作失败，请查看当前状态',{icon:'warn'});
          else window.Toast.show(`已批量${action}`);
        }
        if(el.isConnected)paint(el);
      } finally {bulkBusy=false;if(host.isConnected)host.querySelectorAll('button').forEach(b=>b.disabled=false);}
    });
  }

  function batchPrice(el, list) {
    const selected=list.filter(p=>picked.has(p.id));
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
           <input class="inp tnum" id="bp-val" type="number" step="0.1" placeholder="例如 28">
           <div class="fld-hint" id="bp-hint">将覆盖所选菜品的现有售价，四舍五入到角。</div>
         </div><div data-price-preview></div>`,
      footerHtml:
        `<button class="btn btn--line" data-a="c">取消</button>
         <button class="btn btn--blue" data-a="ok">确认改价</button>`,
      onMount(root, close) {
        const mode = root.querySelector('#bp-mode');
        const lb = root.querySelector('#bp-lb');
        const hint = root.querySelector('#bp-hint');
        const input=root.querySelector('#bp-val');
        const calculate=()=>{
          if(!input.value.trim())throw new Error('请输入数值');
          const value=Number(input.value);if(!Number.isFinite(value))throw new Error('请输入有效数值');
          return selected.map(p=>{
            const raw=mode.value==='set'?value:mode.value==='pct'?p.price*(1+value/100):p.price+value;
            const price=Math.round((raw+Number.EPSILON)*10)/10;
            if(!Number.isFinite(price)||price<=0||Math.round(price*100)>4294967295)throw new Error(`「${p.name}」调整后售价无效`);
            return Object.assign({},p,{price});
          });
        };
        const preview=()=>{
          const m=mode.value;lb.textContent=m==='set'?'目标价格（元）':m==='pct'?'调整百分比（%）':'加减金额（元）';
          hint.textContent=(m==='set'?'将覆盖所选菜品的现有售价。':'正数上调，负数下调。')+'结果统一四舍五入到角。';
          let rows;try{rows=calculate();}catch(e){root.querySelector('[data-price-preview]').textContent=e.message;return;}
          root.querySelector('[data-price-preview]').innerHTML=T.render({cols:[{t:'菜品名称',render:p=>T.esc(p.name)},{t:'当前售价',render:p=>T.money(Number(selected.find(x=>x.id===p.id).price).toFixed(2))},{t:'调整后售价',render:p=>T.money(p.price.toFixed(2))}],rows});
        };
        mode.onchange=input.oninput=preview;preview();
        root.querySelector('[data-a="c"]').onclick=close;
        const button=root.querySelector('[data-a="ok"]');
        button.onclick=window.FormUI.submit(button,async()=>{
          const rows=calculate();
          const results=await Promise.allSettled(rows.map(p=>Api.saveProduct(p)));
          if(el.isConnected)paint(el);
          const failure=results.find(r=>r.status==='rejected');
          if(failure)throw failure.reason;
          close();window.Toast.show(`已批量改价 ${rows.length} 项`);
        });
      },
    });
  }

  /* ---------------- 编辑抽屉（原 admin-product-edit 页） ---------------- */
  function openEdit(el, p) {
    const isEdit = !!p;
    const cats = categoryCache.filter(c => c.on || (p && c.id === p.categoryId)).map(c => c.name);
    let imgs = (p && p.imgs ? p.imgs.slice() : []);

    window.Drawer.open({
      title: isEdit ? '编辑菜品' : '上架新菜',
      guardChanges:true,
      isDirty:()=>JSON.stringify(imgs)!==JSON.stringify(p&&p.imgs?p.imgs:[]),
      bodyHtml:
        `<div class="fld">
           <div class="fld-lb">菜品图片 <span class="faint">最多 3 张，第一张为封面</span></div>
           <div id="imgs"></div>
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
         </div>
         <div class="fld">
           <div class="fld-lb">餐段可售 <span class="req">*</span></div>
           <select class="sel" id="f-meal">
             ${Api.MEALS.map(m => `<option value="${m}"${p && p.meal === m ? ' selected' : ''}>${Api.MEAL_LABEL[m]}</option>`).join('')}
           </select>
           <div class="fld-hint">用户端菜单按所选取餐时间的餐段过滤商品。</div>
         </div>
         <div class="fld">
           <div class="fld-lb">分类 <span class="req">*</span></div>
           <select class="sel" id="f-cat">
             <option value="">请选择分类</option>
             ${cats.map(c => `<option value="${T.esc(c)}"${p && p.cat === c ? ' selected' : ''}>${T.esc(c)}${categoryCache.find(category=>category.name===c)?.on?'':'（已隐藏）'}</option>`).join('')}
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
        let uploading=false;
        const save=root.querySelector('[data-a="ok"]');
        const imageEditor=window.ImageEditor.mount(root.querySelector('#imgs'),imgs,value=>{imgs=value;},value=>{uploading=value;save.disabled=value;save.textContent=value?'上传中…':'保存';});
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
              close(true);
              if(el.isConnected)paint(el);
              window.Toast.show('已删除', { icon: 'check' });
            }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
          });
        };

        save.onclick = window.FormUI.submit(save,async () => {
          if(uploading)return;
          imageEditor.setLocked(true);
          try { await Api.saveProduct({
            id: p ? p.id : '',
            name: root.querySelector('#f-name').value,
            price: root.querySelector('#f-price').value,
            cat: root.querySelector('#f-cat').value,
            meal: root.querySelector('#f-meal').value,
            desc: root.querySelector('#f-desc').value,
            imgs,
          }).then(() => {
            close(true);
            if(el.isConnected)paint(el);
            window.Toast.show(isEdit ? '已保存' : '已上架', { icon: 'check' });
          });
          } finally {imageEditor.setLocked(false);}
        },'保存中…');
      },
    });
  }

  window.Pages = window.Pages || {};
  window.Pages['products'] = { sub: '上下架、售罄、餐段与价格', render };
})();
