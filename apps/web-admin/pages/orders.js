/* 订单管理 —— 对应 apps/wechat-miniprogram/pages/admin-orders + admin-order-detail
   PC 形态：左侧订单表格 + 右侧常驻详情面板，选中即显示，不跳页。 */
(function () {
  const Api = window.Api, T = window.Table, I = window.Icon;

  let lane = '待制作';
  let selId = '';

  function render(el) {
    // 由工作台带过来的泳道 / 选中项
    if (window.__ordersLane) { lane = window.__ordersLane; window.__ordersLane = ''; }
    if (window.__orderSel) { selId = window.__orderSel; window.__orderSel = ''; }

    el.innerHTML =
      `<div class="split-l">
         <div class="toolbar">
           <div class="segs" id="lanes"></div>
           <span class="sp"></span>
           <span class="faint" style="font-size:12.5px">选中左侧订单，右侧显示完整单据</span>
         </div>
         <div id="tbl-host"></div>
       </div>
       <aside class="split-r" id="detail"></aside>`;

    paint(el);
  }

  function paint(el) {
    const counts = Api.laneCounts();
    el.querySelector('#lanes').innerHTML = Api.LANES.map(l =>
      `<span class="seg${l === lane ? ' on' : ''}" data-lane="${l}">${l}<span class="cnt">${counts[l]}</span></span>`
    ).join('');
    el.querySelectorAll('[data-lane]').forEach(n => {
      n.onclick = () => { lane = n.dataset.lane; paint(el); };
    });

    Api.listOrders(lane).then(list => {
      // 选中项不在当前泳道时，回落到本泳道第一条
      if (!list.some(o => o.id === selId)) selId = list.length ? list[0].id : '';

      const host = el.querySelector('#tbl-host');
      host.innerHTML = T.render({
        // 左表只做可扫读的索引：订单号 / 联系人 / 金额 都在右侧详情面板里，
        // 分栏布局的意义就在于不重复展示同一份信息。
        cols: [
          { t: '取餐号', w: '72px', render: r => `<b class="tnum">${r.code}</b>` },
          { t: '菜品', render: r => {
            const band = [r.flavor && r.flavor !== '—' ? r.flavor : '', r.note].filter(Boolean).join(' · ');
            return `<div class="ellipsis">${T.esc(Api.itemsSummary(r.items))}</div>` +
                   (band ? `<div class="ord-band ellipsis">${T.esc(band)}</div>` : '');
          } },
          { t: '等待', w: '84px', render: r => `<span class="faint tnum">${r.time}</span><br><span class="faint tnum" style="font-size:11.5px">${r.mins} 分钟</span>` },
          { t: '状态', w: '84px', render: r => T.pill(r.status) },
          { t: '', w: '84px', cls: 'act', render: r => {
            const m = Api.advanceMeta(r.status);
            return m.isView ? '' : `<button class="btn btn--sm ${m.cls}" data-act="adv" data-id="${r.id}">${m.label}</button>`;
          } },
        ],
        rows: list,
        empty: `「${lane}」暂无订单`,
        rowCls: r => (r.id === selId ? 'sel' : ''),
        onRow: true,
      });

      T.bind(host, {
        adv(id) {
          Api.advanceOrder(id).then(r => {
            window.Toast.show(`已${r.act}「${r.code}」`, {
              icon: 'check',
              onUndo: () => { Api.revertOrder(id, r.prev); paint(el); },
            });
            paint(el);
          }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
        },
        row(id) { selId = id; paint(el); },
      });

      detail(el, selId);
    });
  }

  function detail(el, id) {
    const host = el.querySelector('#detail');
    const o = id ? Api.findOrder(id) : null;
    if (!o) {
      host.innerHTML = `<div class="dt-empty">${I.svg('receipt', 30, '#c7cab8')}<div>选择左侧订单查看详情</div></div>`;
      return;
    }

    const rows = o.items.map(([iid, q, p]) => {
      const m = window.Seed.itemById(iid);
      return { name: m ? m.name : '已删除菜品', q, p, sub: p * q };
    });
    const m = Api.advanceMeta(o.status);

    host.innerHTML =
      `<div class="dt-hero">
         <span class="ring r1"></span>
         <div class="dt-code tnum">${o.code}</div>
         <div class="dt-no tnum">${o.no}</div>
         <span class="pill pill--ondark"><i class="pd"></i>${o.status}</span>
       </div>
       <div class="dt-body">
         <div class="sec-h"><span class="t">菜品明细</span></div>
         <div class="card card-pad">
           ${rows.map(r => `
             <div class="dt-item">
               <span class="grow">${T.esc(r.name)}</span>
               <span class="faint tnum">×${r.q}</span>
               <span class="tnum" style="width:56px;text-align:right">${T.money(r.sub)}</span>
             </div>`).join('')}
           <div class="dt-total"><span class="grow">合计</span>${T.money(o.total)}</div>
         </div>

         <div class="sec-h"><span class="t">取餐与备注</span></div>
         <div class="card card-pad">
           <div class="kv"><span class="k">联系人</span><span class="v">${T.esc(o.contact)}</span></div>
           <div class="kv"><span class="k">手机号</span><span class="v tnum">${o.phone}</span></div>
           <div class="kv"><span class="k">下单时间</span><span class="v tnum">${o.time} · ${o.mins} 分钟前</span></div>
           <div class="kv"><span class="k">口味</span><span class="v">${T.esc(o.flavor || '—')}</span></div>
           <div class="kv"><span class="k">备注</span><span class="v">${T.esc(o.note || '—')}</span></div>
           <div class="kv"><span class="k">取餐点</span><span class="v">${window.Seed.STORE.pickupWindow}</span></div>
         </div>
       </div>
       <div class="dt-foot">
         <button class="btn btn--line" data-print>${I.svg('printer', 16)}打印小票</button>
         <span class="grow"></span>
         ${m.isView ? `<span class="faint" style="font-size:12.5px">该订单已${o.status}</span>`
                    : `<button class="btn ${m.cls}" data-adv>${m.label}</button>`}
       </div>`;

    const printBtn = host.querySelector('[data-print]');
    printBtn.onclick = () => window.Toast.show('打印机对接为后期演进 · 一期不含', { icon: 'warn' });

    const advBtn = host.querySelector('[data-adv]');
    if (advBtn) advBtn.onclick = () => {
      Api.advanceOrder(o.id).then(r => {
        window.Toast.show(`已${r.act}「${r.code}」`, {
          icon: 'check',
          onUndo: () => { Api.revertOrder(o.id, r.prev); paint(el); },
        });
        paint(el);
      }).catch(e => window.Toast.show(e.message, { icon: 'warn' }));
    };
  }

  window.Pages = window.Pages || {};
  window.Pages['orders'] = { sub: '履约流转：待制作 → 待取餐 → 已完成', flush: true, render };
})();
