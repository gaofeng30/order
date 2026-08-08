/* 优惠券 —— 对应 miniprogram/pages/admin-coupons + admin-coupon-edit（二期能力）

   发放模型：券按等级自动生效，商户勾选适用等级后该等级会员在卡包中直接可见可用，
   没有领取动作。券是一条规则而非一张票，不为每个用户生成券实例，因此不设发放总量；
   成本控制依靠有效期与每人可用次数。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  const TODAY = () => window.Seed.TODAY;

  function render(el) {
    el.innerHTML =
      `<div class="page-head">
         <span class="ph-s grow">券按等级自动生效，会员无需领取。一单只用一张，与等级折扣叠加。演示时钟：${TODAY()}</span>
         <button class="btn btn--primary" data-new>${I.svg('plus', 16)}新建优惠券</button>
       </div>
       <div id="tbl-host"></div>`;

    el.querySelector('[data-new]').onclick = () => openEdit(el, null);
    paint(el);
  }

  function benefit(c) {
    if (c.type === 'cut') {
      return `满 ${c.threshold} 减 ${c.amount}`.replace('满 0 ', '无门槛 ');
    }
    return `${(c.rate / 10).toFixed(1)} 折 · 封顶 ${c.cap} 元` + (c.threshold ? ` · 满 ${c.threshold}` : '');
  }

  function scopeText(c) {
    if (c.scope === 'all') return '全场通用';
    if (c.scope === 'cat') return '分类：' + c.catNames.join('、');
    const names = c.itemIds.map(id => { const m = window.Seed.itemById(id); return m ? m.name : id; });
    return '菜品：' + names.join('、');
  }

  function paint(el) {
    Promise.all([Api.listCoupons(), Api.listLevels()]).then(([list, levels]) => {
      const lvName = id => (levels.find(l => l.id === id) || { name: '已删除档位' }).name;
      const host = el.querySelector('#tbl-host');

      host.innerHTML = T.render({
        cols: [
          { t: '券名称', w: '156px', render: r => {
            const expired = r.end < TODAY();
            return `<b>${T.esc(r.name)}</b>` +
              (expired ? '<span class="pill pill--mute" style="margin-left:6px">已过期</span>' : '');
          } },
          { t: '类型', w: '64px', render: r => r.type === 'cut'
            ? '<span class="pill pill--ok">满减</span>' : '<span class="pill pill--info">折扣</span>' },
          { t: '优惠', w: '160px', render: r => `<span class="tnum">${T.esc(benefit(r))}</span>` },
          { t: '适用等级', w: '170px', render: r => r.levelIds.length
            ? r.levelIds.map(id => `<span class="pill pill--info" style="margin:1px 3px 1px 0">${T.esc(lvName(id))}</span>`).join('')
            : '<span class="pill pill--mute">无（已停用）</span>' },
          { t: '适用范围', render: r => `<span class="ellipsis" style="display:block;max-width:230px">${T.esc(scopeText(r))}</span>` },
          // 有效期与每人可用次数是这张券的两项成本闸门，放在一起看
          { t: '有效期 / 限次', w: '176px', render: r =>
            `<div class="faint tnum" style="font-size:12.5px">${r.start} ~ ${r.end}</div>
             <div class="tnum" style="font-size:12.5px">每人 ${r.perLimit} 次</div>` },
          { t: '启用', w: '56px', render: r => `<button class="sw${r.enabled ? ' on' : ''}" data-act="en" data-id="${r.id}"></button>` },
          { t: '操作', w: '104px', cls: 'act', render: r =>
            `<button class="btn btn--sm btn--ghost-blue" data-act="edit" data-id="${r.id}">配置</button>
             <button class="ibtn danger" data-act="del" data-id="${r.id}">${I.svg('trash', 16)}</button>` },
        ],
        rows: list,
        empty: '还没有优惠券',
      });

      T.bind(host, {
        en(id, node) {
          const c = list.find(x => x.id === id);
          Api.setCouponEnabled(id, !c.enabled).then(r => {
            paint(el);
            window.Toast.show(r.enabled ? `「${r.name}」已启用` : `「${r.name}」已停用`, { icon: r.enabled ? 'check' : 'box' });
          });
        },
        edit(id) { openEdit(el, list.find(x => x.id === id)); },
        del(id) {
          const c = list.find(x => x.id === id);
          window.Modal.confirm({
            title: '删除优惠券',
            body: `确认删除「${T.esc(c.name)}」？删除后该券立即从会员卡包中消失，已使用的历史订单不受影响。`,
            okText: '删除', danger: true,
          }).then(yes => {
            if (!yes) return;
            Api.deleteCoupon(id).then(() => { paint(el); window.Toast.show('已删除', { icon: 'check' }); })
              .catch(e => window.Toast.show(e.message, { icon: 'warn' }));
          });
        },
      });
    });
  }

  /* ---------------- 配置抽屉（原 admin-coupon-edit 页） ---------------- */
  function openEdit(el, c) {
    const isEdit = !!c;
    Api.listLevels().then(levels => {
      const cats = window.__store.cats.map(x => x.name);
      const menu = window.__store.menu;
      const cur = c || {
        name: '', type: 'cut', amount: 5, rate: 90, cap: 10, threshold: 0,
        levelIds: [], scope: 'all', catNames: [], itemIds: [],
        start: TODAY(), end: TODAY(), perLimit: 1, enabled: true,
      };
      const picked = {
        levels: new Set(cur.levelIds || []),
        cats: new Set(cur.catNames || []),
        items: new Set(cur.itemIds || []),
      };

      window.Drawer.open({
        title: isEdit ? '配置优惠券' : '新建优惠券',
        tag: '二期',
        wide: true,
        bodyHtml:
          `<div class="fld">
             <div class="fld-lb">券名称 <span class="req">*</span></div>
             <input class="inp" id="f-name" value="${T.esc(cur.name)}" placeholder="会员在卡包里看到的名字">
           </div>

           <div class="fld">
             <div class="fld-lb">优惠类型 <span class="req">*</span></div>
             <div class="segs">
               <span class="seg${cur.type === 'cut' ? ' on' : ''}" data-type="cut">满减</span>
               <span class="seg${cur.type === 'discount' ? ' on' : ''}" data-type="discount">折扣</span>
             </div>
           </div>

           <div id="type-cut" style="${cur.type === 'cut' ? '' : 'display:none'}">
             <div class="fld-row">
               <div class="fld"><div class="fld-lb">减免金额（元）<span class="req">*</span></div>
                 <input class="inp tnum" id="f-amount" type="number" min="1" value="${cur.amount}"></div>
               <div class="fld"><div class="fld-lb">使用门槛（元）</div>
                 <input class="inp tnum" id="f-th1" type="number" min="0" value="${cur.threshold}">
                 <div class="fld-hint">0 = 无门槛</div></div>
             </div>
           </div>

           <div id="type-dis" style="${cur.type === 'discount' ? '' : 'display:none'}">
             <div class="fld-row">
               <div class="fld"><div class="fld-lb">折扣（%）<span class="req">*</span></div>
                 <input class="inp tnum" id="f-rate" type="number" min="1" max="99" value="${cur.rate}">
                 <div class="fld-hint">80 = 打 8 折</div></div>
               <div class="fld"><div class="fld-lb">封顶金额（元）<span class="req">*</span></div>
                 <input class="inp tnum" id="f-cap" type="number" min="1" value="${cur.cap}">
                 <div class="fld-hint">折扣券必填，防止大额订单失控</div></div>
               <div class="fld"><div class="fld-lb">使用门槛（元）</div>
                 <input class="inp tnum" id="f-th2" type="number" min="0" value="${cur.threshold}"></div>
             </div>
           </div>

           <div class="fld">
             <div class="fld-lb">适用等级 <span class="req">*</span></div>
             <div class="segs" id="lv-pick">
               ${levels.map(l => `<span class="chip blue${picked.levels.has(l.id) ? ' on' : ''}" data-lv="${l.id}">${T.esc(l.name)}</span>`).join('')}
             </div>
             <div class="fld-hint">勾选后该等级会员在卡包中直接可见可用，没有领取动作。若一张券的适用等级被清空，该券会自动停用。</div>
           </div>

           <div class="fld">
             <div class="fld-lb">适用范围 <span class="req">*</span></div>
             <div class="segs">
               <span class="seg${cur.scope === 'all' ? ' on' : ''}" data-scope="all">全场通用</span>
               <span class="seg${cur.scope === 'cat' ? ' on' : ''}" data-scope="cat">指定分类</span>
               <span class="seg${cur.scope === 'item' ? ' on' : ''}" data-scope="item">指定菜品</span>
             </div>
             <div class="segs" id="scope-cat" style="margin-top:10px;${cur.scope === 'cat' ? '' : 'display:none'}">
               ${cats.map(n => `<span class="chip${picked.cats.has(n) ? ' on' : ''}" data-cat="${T.esc(n)}">${T.esc(n)}</span>`).join('')}
             </div>
             <div class="segs" id="scope-item" style="margin-top:10px;${cur.scope === 'item' ? '' : 'display:none'}">
               ${menu.map(m => `<span class="chip${picked.items.has(m.id) ? ' on' : ''}" data-item="${m.id}">${T.esc(m.name)}</span>`).join('')}
             </div>
           </div>

           <div class="fld-row">
             <div class="fld"><div class="fld-lb">生效日期 <span class="req">*</span></div>
               <input class="inp tnum" id="f-start" type="date" value="${cur.start}"></div>
             <div class="fld"><div class="fld-lb">失效日期 <span class="req">*</span></div>
               <input class="inp tnum" id="f-end" type="date" value="${cur.end}"></div>
             <div class="fld"><div class="fld-lb">每人可用次数 <span class="req">*</span></div>
               <input class="inp tnum" id="f-limit" type="number" min="1" value="${cur.perLimit}"></div>
           </div>
           <div class="fld-hint" style="margin-top:-6px">
             券不为每个用户生成实例，因此不设发放总量；成本控制依靠有效期与每人可用次数两项。
           </div>

           <div class="fld row gap12" style="margin-top:14px">
             <button class="sw${cur.enabled ? ' on' : ''}" id="f-en"></button>
             <span style="font-size:13px">启用该券</span>
           </div>`,
        footerHtml:
          `<span class="grow"></span>
           <button class="btn btn--line" data-a="c">取消</button>
           <button class="btn btn--primary" data-a="ok">保存</button>`,
        onMount(root, close) {
          let type = cur.type;
          let scope = cur.scope;

          root.querySelectorAll('[data-type]').forEach(n => n.onclick = () => {
            type = n.dataset.type;
            root.querySelectorAll('[data-type]').forEach(x => x.classList.toggle('on', x.dataset.type === type));
            root.querySelector('#type-cut').style.display = type === 'cut' ? '' : 'none';
            root.querySelector('#type-dis').style.display = type === 'discount' ? '' : 'none';
          });

          root.querySelectorAll('[data-scope]').forEach(n => n.onclick = () => {
            scope = n.dataset.scope;
            root.querySelectorAll('[data-scope]').forEach(x => x.classList.toggle('on', x.dataset.scope === scope));
            root.querySelector('#scope-cat').style.display = scope === 'cat' ? '' : 'none';
            root.querySelector('#scope-item').style.display = scope === 'item' ? '' : 'none';
          });

          const toggle = (sel, set, key) => root.querySelectorAll(sel).forEach(n => n.onclick = () => {
            const v = n.dataset[key];
            if (set.has(v)) set.delete(v); else set.add(v);
            n.classList.toggle('on');
          });
          toggle('[data-lv]', picked.levels, 'lv');
          toggle('[data-cat]', picked.cats, 'cat');
          toggle('[data-item]', picked.items, 'item');

          const en = root.querySelector('#f-en');
          en.onclick = () => en.classList.toggle('on');

          root.querySelector('[data-a="c"]').onclick = close;
          root.querySelector('[data-a="ok"]').onclick = () => {
            Api.saveCoupon({
              id: c ? c.id : '',
              name: root.querySelector('#f-name').value,
              type,
              amount: root.querySelector('#f-amount').value,
              rate: root.querySelector('#f-rate').value,
              cap: root.querySelector('#f-cap').value,
              threshold: (type === 'cut' ? root.querySelector('#f-th1') : root.querySelector('#f-th2')).value,
              levelIds: Array.from(picked.levels),
              scope,
              catNames: Array.from(picked.cats),
              itemIds: Array.from(picked.items),
              start: root.querySelector('#f-start').value,
              end: root.querySelector('#f-end').value,
              perLimit: root.querySelector('#f-limit').value,
              enabled: en.classList.contains('on'),
            }).then(() => {
              close(); paint(el);
              window.Toast.show(isEdit ? '已保存' : '已新建优惠券', { icon: 'check' });
            }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
          };
        },
      });
    });
  }

  window.Pages = window.Pages || {};
  window.Pages['coupons'] = { render };
})();
